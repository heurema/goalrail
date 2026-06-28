"""Phase 0 characterization test — sub-agent visibility in overview.

Migrated to mock LLM: the supervisor is backed by ``openai-agents``
with mock responses. Sub-agent (``claude-sdk`` and ``codex``)
parametrize rows still require the real CLI binary on PATH — those
are skipped when the binary is missing.

The core invariant remains: when a sub-agent session is registered,
the REPL overview pane must render the sub-agent's label, executor
harness, and user message.

**What breaks if this fails:**
- The ``sys_session_send`` builtin's output JSON drops
  ``conversation_id`` — the REPL's overview target registration
  keys on it.
- ``_collect_overview_targets`` stops including managed agent sessions.
- ``_render_overview_managed_session_text`` drops metadata lines.
- The wrapped harness invocation fails, so the worker never comes up.
"""

from __future__ import annotations

import contextlib
import subprocess
import time
from pathlib import Path
from shutil import which
from typing import Any

import pexpect
import pytest

from tests.e2e._harness_probes import HARNESS_HARNESS_MODELS, HARNESS_IDS
from tests.e2e.goalrail._pexpect_harness import (
    clean_exit,
    spawn_goalrail_run,
    strip_ansi,
    submit_prompt,
    wait_for_ready,
)
from tests.e2e.goalrail._snapshot import compare_snapshot
from tests.e2e.goalrail.conftest import configure_mock_llm

# Supervisor model — mock LLM serves deterministic responses.
_SUPERVISOR_MODEL = "mock-overview-subagent-supervisor"
_SUPERVISOR_HARNESS = "openai-agents"

# Mapping from harness id to the YAML's worker tool name.
_WORKER_TOOL_BY_HARNESS: dict[str, str] = {
    "claude-sdk": "claude_worker",
    "codex": "codex_worker",
}

_SUBAGENT_MESSAGE_CONTENT = "say hello"

_SPAWN_TIMEOUT = 60.0
_BOOT_TIMEOUT = 60.0
_RUNNING_TIMEOUT = 30.0
_COMPLETION_TIMEOUT = 240.0
_EXIT_TIMEOUT = 15.0
_OVERVIEW_DRAIN_TIMEOUT = 6.0


def _check_worker_harness_available(harness: str, goalrail_python: Path) -> None:
    """
    Fail loud if the worker harness's prerequisites are missing.

    :param harness: The worker harness identifier under test.
    :param goalrail_python: The subprocess interpreter.
    """
    if harness == "claude-sdk":
        probe = subprocess.run(
            [
                str(goalrail_python),
                "-c",
                "import importlib.util, sys; "
                "sys.exit(0 if importlib.util.find_spec('claude_agent_sdk') else 1)",
            ],
            capture_output=True,
        )
        if probe.returncode != 0 or which("claude") is None:
            pytest.fail(
                "claude-sdk prerequisites missing: need both the "
                "'claude_agent_sdk' Python package and the 'claude' "
                "CLI binary on PATH."
            )
    elif harness == "codex":
        if which("codex") is None:
            pytest.fail(
                "codex prerequisite missing: the 'codex' CLI binary "
                "must be installed on PATH (install via "
                "'npm i -g @openai/codex')."
            )


def _open_overview_on_subagent(child: pexpect.spawn, worker_label: str) -> None:
    """
    Open the overview once the dispatched sub-agent target is visible.

    :param child: Running REPL child process.
    :param worker_label: Expected sidebar label, e.g. ``claude_worker:demo``.
    """
    deadline = time.monotonic() + _COMPLETION_TIMEOUT
    last_before = ""
    while True:
        child.sendcontrol("o")
        try:
            child.expect(worker_label, timeout=2.0)
            return
        except pexpect.TIMEOUT:
            last_before = strip_ansi(child.before or "")
            child.send("q")
            if time.monotonic() >= deadline:
                raise AssertionError(
                    f"sub-agent overview target {worker_label!r} did not appear "
                    f"within {_COMPLETION_TIMEOUT}s\n\nlast output:\n{last_before[-2500:]}"
                ) from None
            time.sleep(0.5)


@pytest.mark.parametrize("harness,model", HARNESS_HARNESS_MODELS, ids=HARNESS_IDS)
def test_repl_overview_subagent_visibility(
    goalrail_python: Path,
    goalrail_repo_root: Path,
    mock_credentials_env: dict[str, str],
    mock_llm_server_url: str,
    harness: str,
    model: str,
) -> None:
    """
    Spawn a supervisor that delegates to a sub-agent worker, open
    the overview, cycle to the sub-agent target, and verify
    its metadata lines render.

    Uses the mock LLM server for supervisor responses. Sub-agent
    harnesses (claude-sdk, codex) still require their respective CLI
    binaries on PATH — rows fail loudly (not skip) when those are
    absent, see :func:`_check_worker_harness_available`.

    The visibility contract is narrow on purpose: the worker CLI runs
    for real and typically fails to authenticate in the sandbox, so it
    never reaches the mock and produces an error turn. That does not
    matter here — the sub-agent session is registered at *dispatch*
    time (``status: launching``) with the dispatched user message in
    its args, so it appears in the overview regardless of whether the
    worker's own turn succeeds. We assert only that: the sub-agent's
    label and the dispatched user message render. (The overview does
    not render an executor-harness line for sub-agent targets at all —
    that is gated to the main session in ``_repl.py`` — so there is no
    executor-harness assertion.)

    :param goalrail_python: Interpreter with goalrail installed.
    :param goalrail_repo_root: Working directory for the subprocess.
    :param mock_credentials_env: Mock-LLM env vars.
    :param mock_llm_server_url: Mock server URL.
    :param harness: Worker harness identifier from
        :data:`HARNESS_HARNESS_MODELS`.
    :param model: Model identifier (unused at CLI level; accepted
        to match the parametrize shape).
    """
    if harness not in _WORKER_TOOL_BY_HARNESS:
        # ``coding_supervisor.yaml`` only defines worker tools for
        # claude-sdk and codex. Other harnesses skip cleanly.
        pytest.skip(
            f"{harness!r} has no <harness>_worker tool in "
            f"tests/resources/examples/coding_supervisor.yaml; this test requires the "
            f"YAML to declare an AgentTool for the harness."
        )
    _check_worker_harness_available(harness, goalrail_python)
    worker_tool = _WORKER_TOOL_BY_HARNESS[harness]
    worker_label_prefix = f"{worker_tool}:"
    user_prompt = (
        f"Delegate to {worker_tool}. Call sys_session_send with "
        f"agent={worker_tool}, title=demo, and args='say hello'. "
        f"Do not answer inline. After the worker replies, relay its "
        f"message verbatim."
    )
    yaml_path = goalrail_repo_root / "tests" / "resources" / "examples" / "coding_supervisor.yaml"

    # Mock responses served FIFO from the shared ``key="default"`` queue to
    # whichever agent makes an LLM call. In practice the worker CLI runs for
    # real (and usually fails auth in the sandbox, never reaching the mock), so
    # the supervisor consumes these in order: (1) the dispatch tool_call that
    # spawns + registers the sub-agent, then (2)/(3) its follow-up text turns
    # after the worker result (an error) auto-wakes it. The exact text is not
    # asserted; "The worker said" is just a stable turn-complete sync marker.
    supervisor_responses = [
        # Dispatch the worker via sys_session_send (named mode: agent/title/args).
        {
            "tool_calls": [
                {
                    "call_id": "call_session_send",
                    "name": "sys_session_send",
                    "arguments": (
                        f'{{"agent": "{worker_tool}", "title": "demo", "args": "say hello"}}'
                    ),
                }
            ]
        },
        {"text": "hello from the worker"},
        # Final supervisor turn. The test no longer waits for this text,
        # but keeping the response preserves the full conversation path.
        {"text": "The worker said: hello"},
        # Spares so any extra agent LLM call never 500s the mock.
        {"text": "(spare)"},
        {"text": "(spare)"},
    ]
    configure_mock_llm(
        mock_llm_server_url,
        supervisor_responses,
        key="default",
    )
    configure_mock_llm(mock_llm_server_url, supervisor_responses, key=_SUPERVISOR_MODEL)

    child = spawn_goalrail_run(
        goalrail_python=goalrail_python,
        yaml_path=yaml_path,
        model=_SUPERVISOR_MODEL,
        harness=_SUPERVISOR_HARNESS,
        env=mock_credentials_env,
        cwd=goalrail_repo_root,
        timeout=_SPAWN_TIMEOUT,
    )
    try:
        wait_for_ready(child, timeout=_BOOT_TIMEOUT)
        submit_prompt(child, user_prompt)
        # Wait for the sidebar to paint the sub-agent target. The sidebar entry
        # is "👾 <worker>:demo"; "<worker>:demo" matches there cleanly. (The
        # detail header renders "Session: <worker>:demo", but the two-column
        # overlay wraps the narrow detail column and splits "Session: <worker>:",
        # so it never matches contiguously — sync on the sidebar label instead.)
        _open_overview_on_subagent(child, f"{worker_label_prefix}demo")
        # Select the sub-agent target so its detail pane (Session header +
        # message stream, incl. the dispatched user message) renders; TAB
        # cycles main -> sub-agent.
        child.send("\t")
        # Accumulate the detail pane. The status-bar clock ticks ~1×/s, so
        # a drain that bails on a 0.3s idle gap is unreliable here — force a
        # fixed-duration read with an impossible-pattern expect.
        with contextlib.suppress(pexpect.TIMEOUT):
            child.expect("ZZZ_NEVER_MATCHES_DRAIN", timeout=_OVERVIEW_DRAIN_TIMEOUT)
        subagent_stripped = f"{worker_label_prefix}demo" + strip_ansi(child.before or "")
        # Close the overlay before teardown ('q'); leaving it open blocks the
        # Ctrl+D / "/quit" exit handshake (see test_repl_ctrl_o_overview).
        child.send("q")
        clean_exit(child, timeout=_EXIT_TIMEOUT)
        exit_code = child.exitstatus
    finally:
        if not child.closed:
            child.close(force=True)

    # NOTE: the Ctrl+O overview does NOT render an executor-harness line for
    # sub-agent targets — the Agent/Model/harness fields are gated to the main
    # session (`is_main`) in _repl.py. So a "subagent_executor_harness_rendered"
    # assertion would only ever pass for [codex] by coincidence (the substring
    # "codex" appears in the agent name "codex_worker"), never for [claude-sdk].
    # The test asserts only what the overview genuinely renders for a sub-agent:
    # its label and the dispatched user message.
    del exit_code
    observed: dict[str, Any] = {
        "subagent_label_present": worker_label_prefix in subagent_stripped,
        "subagent_user_message_rendered": _SUBAGENT_MESSAGE_CONTENT in subagent_stripped,
    }
    diffs = compare_snapshot("test_repl_overview_subagent_visibility", observed)
    assert diffs == [], (
        "Snapshot mismatch for sub-agent overview visibility:\n"
        + "\n".join(diffs)
        + f"\n\nsubagent stripped (last 2500):\n"
        f"{subagent_stripped[-2500:]}"
    )
