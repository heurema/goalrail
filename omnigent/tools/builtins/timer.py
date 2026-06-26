"""
LLM-callable timer builtins.

Two tools:

- :class:`SysTimerSetTool` (``sys_timer_set``) — schedules a timer
  that fires inbox notifications at a future timestamp.
- :class:`SysTimerCancelTool` (``sys_timer_cancel``) — cancels a
  previously scheduled timer by ``timer_id``.

Both tools are gated on the agent spec's top-level ``timers:`` flag
(see :attr:`AgentSpec.timers`, defaulting to ``False`` to match the
inner stack). On the sessions-native path the timer workflow has
not yet been re-implemented on the runner; ``sys_timer_set`` raises
``NotImplementedError`` and ``sys_timer_cancel`` always returns
``status="not_found"``.

The tools are **synchronous** (``is_async() == False``): the LLM
gets the ``timer_id`` directly so it can later cancel by ID. The
firing (when implemented) arrives as a ``[System: timer X fired]``
system message in the conversation, with ``kind="timer"`` so the
parent's end-of-turn auto-collect (which consults
:data:`_DRAIN_KINDS`) does NOT block on pending firings.

See ``designs/SERVER_HARNESS_CONTRACT.md`` §Timers and step 10.
"""

from __future__ import annotations

import json
import logging
import uuid
from typing import Any

from omnigent.tools.base import Tool, ToolContext

_logger = logging.getLogger(__name__)

# Maximum ``seconds`` value the LLM can pass to ``sys_timer_set``.
# A pragmatic cap (~12 days) that's long enough for any realistic
# scheduling use case and short enough that an obvious typo (e.g.
# the LLM hallucinating ``seconds=99999999``) can't park a timer
# indefinitely. If a real use case exceeds this, the cap gets
# revisited together with the design tradeoffs of long-lived
# timer workflows.
_MAX_TIMER_SECONDS = 1_000_000.0


class SysTimerSetTool(Tool):
    """
    Schedule a timer that fires inbox notifications.

    The LLM passes ``seconds`` (delay), optional ``repeat`` (default
    ``False``), and optional ``note`` (string echoed back in each
    firing). The tool generates a fresh ``timer_id`` of the form
    ``"timer_<32-char hex>"``, starts a
    the runner-side timer task pinned to that id via
    :class:`SetWorkflowID`, and returns the id immediately.

    Firings arrive later in the conversation as ``[System: timer X
    fired]`` system messages between iterations (the existing
    ``async_work_complete`` drain path). Repeating timers continue
    until ``sys_timer_cancel`` is called.
    """

    @classmethod
    def name(cls) -> str:
        """:returns: ``"sys_timer_set"``."""
        return "sys_timer_set"

    @classmethod
    def description(cls) -> str:
        """
        :returns: Description visible to the LLM in tool listings.
        """
        return (
            "Schedule a timer that fires after a delay. The firing "
            "appears as a [System: timer X fired] message in the "
            "conversation; you can include an optional note that's "
            "echoed back in the firing. Set repeat=true for a "
            "recurring timer (cancel via sys_timer_cancel). The "
            "tool returns immediately with the timer_id."
        )

    def get_schema(self) -> dict[str, Any]:
        """
        :returns: OpenAI tool schema with ``seconds`` (number,
            required), ``repeat`` (boolean, optional, default
            ``False``), and ``note`` (string, optional).
        """
        return {
            "type": "function",
            "function": {
                "name": self.name(),
                "description": self.description(),
                "parameters": {
                    "type": "object",
                    "properties": {
                        "seconds": {
                            "type": "number",
                            "description": (
                                "Delay before the timer fires, in "
                                "seconds. Must be non-negative; the "
                                "first firing happens after this "
                                "delay. For repeat=true, also the "
                                "interval between firings."
                            ),
                        },
                        "repeat": {
                            "type": "boolean",
                            "description": (
                                "When true, the timer fires every "
                                "`seconds` until cancelled. When "
                                "false (default), fires once."
                            ),
                            "default": False,
                        },
                        "note": {
                            "type": "string",
                            "description": (
                                "Optional string echoed in each "
                                "firing's [System: timer X fired] "
                                "message. Useful to disambiguate "
                                "multiple timers."
                            ),
                        },
                    },
                    "required": ["seconds"],
                    "additionalProperties": False,
                },
            },
        }

    def invoke(self, arguments: str, ctx: ToolContext) -> str:
        """
        Generate a ``timer_id``, start the
        the runner-side timer task, return the id to the LLM.

        :param arguments: JSON-encoded args, e.g.
            ``'{"seconds": 5, "repeat": false, "note": "x"}'``.
        :param ctx: Provides ``ctx.conversation_id`` — the
            conversation the workflow appends firing messages to.
            Required; the tool fails loud when it's ``None``.
        :returns: JSON string ``{"timer_id", "status": "scheduled",
            "seconds", "repeat", "note"}`` on success, or
            ``{"error": "..."}`` on validation failure.
        """
        try:
            args = json.loads(arguments) if arguments else {}
        except json.JSONDecodeError as exc:
            return json.dumps({"error": f"invalid arguments: {exc}"})

        seconds_raw = args.get("seconds")
        if not isinstance(seconds_raw, (int, float)) or isinstance(seconds_raw, bool):
            # Reject bool explicitly because Python's ``isinstance(True, int)``
            # is True; allowing it would silently coerce ``True`` to 1.0.
            return json.dumps({"error": "seconds must be a number"})
        seconds = float(seconds_raw)
        if seconds < 0:
            return json.dumps({"error": "seconds must be non-negative"})
        if seconds > _MAX_TIMER_SECONDS:
            return json.dumps({"error": f"seconds must be <= {_MAX_TIMER_SECONDS}"})

        repeat_raw = args.get("repeat", False)
        if not isinstance(repeat_raw, bool):
            return json.dumps({"error": "repeat must be a boolean"})
        repeat = bool(repeat_raw)

        note_raw = args.get("note")
        if note_raw is not None and not isinstance(note_raw, str):
            return json.dumps({"error": "note must be a string"})
        note: str | None = note_raw

        if ctx.conversation_id is None:
            # Fail loud — the timer workflow needs a stable
            # destination to append firing messages to.
            return json.dumps({"error": "sys_timer_set requires a conversation context"})

        timer_id = f"timer_{uuid.uuid4().hex}"
        _spawn_timer_workflow(
            timer_id=timer_id,
            conversation_id=ctx.conversation_id,
            seconds=seconds,
            repeat=repeat,
            note=note,
        )

        return json.dumps(
            {
                "timer_id": timer_id,
                "status": "scheduled",
                "seconds": seconds,
                "repeat": repeat,
                "note": note,
            }
        )


def _spawn_timer_workflow(
    *,
    timer_id: str,
    conversation_id: str,
    seconds: float,
    repeat: bool,
    note: str | None,
) -> None:
    """
    Stub entry point — raises ``NotImplementedError`` until the
    runner provides a timer implementation.

    :param timer_id: The workflow id the timer would have been
        pinned to (the value the LLM uses with ``sys_timer_cancel``).
    :param conversation_id: Conversation the timer would append
        firings to.
    :param seconds: Sleep duration before each firing.
    :param repeat: Whether the timer loops indefinitely.
    :param note: Optional caller-supplied note echoed in firings.
    """
    del timer_id, conversation_id, seconds, repeat, note
    raise NotImplementedError(
        "sys_timer_set is unavailable on the sessions-native path; "
        "the runner does not yet provide a timer implementation."
    )


class SysTimerCancelTool(Tool):
    """
    Cancel a scheduled timer by ``timer_id``.

    On the sessions-native path no active timer can exist (the
    timer workflow has not been re-implemented on the runner), so
    this always returns ``status="not_found"``. Matches the
    inner-stack semantics where a timer that already fired and
    cleaned up is indistinguishable from one that never existed.
    """

    @classmethod
    def name(cls) -> str:
        """:returns: ``"sys_timer_cancel"``."""
        return "sys_timer_cancel"

    @classmethod
    def description(cls) -> str:
        """
        :returns: Description visible to the LLM in tool listings.
        """
        return (
            "Cancel a previously scheduled timer by timer_id. "
            "Returns status='cancelled' if the timer was active, "
            "or status='not_found' if no such timer exists or the "
            "timer has already fired and finished."
        )

    def get_schema(self) -> dict[str, Any]:
        """
        :returns: OpenAI tool schema with ``timer_id`` (string,
            required) — the value the LLM received from
            ``sys_timer_set``.
        """
        return {
            "type": "function",
            "function": {
                "name": self.name(),
                "description": self.description(),
                "parameters": {
                    "type": "object",
                    "properties": {
                        "timer_id": {
                            "type": "string",
                            "description": (
                                "The timer_id returned by sys_timer_set, e.g. 'timer_a1b2c3d4...'."
                            ),
                        },
                    },
                    "required": ["timer_id"],
                    "additionalProperties": False,
                },
            },
        }

    def invoke(self, arguments: str, ctx: ToolContext) -> str:
        """
        Cancel the timer (always ``not_found`` on sessions-native).

        :param arguments: JSON-encoded args, e.g.
            ``'{"timer_id": "timer_..."}'``.
        :param ctx: Tool context (unused; cancellation is keyed on
            ``timer_id`` alone).
        :returns: JSON string
            ``{"timer_id", "status": "not_found"}``.
        """
        del ctx  # The tool doesn't need any per-invocation context.
        try:
            args = json.loads(arguments) if arguments else {}
        except json.JSONDecodeError as exc:
            return json.dumps({"error": f"invalid arguments: {exc}"})

        timer_id = args.get("timer_id")
        if not isinstance(timer_id, str) or not timer_id:
            return json.dumps({"error": "timer_id is required"})

        # No active timer can exist on the sessions-native path.
        return json.dumps({"timer_id": timer_id, "status": "not_found"})
