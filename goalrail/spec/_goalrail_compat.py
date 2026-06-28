"""Goalrail compatibility surface — bundled for surgical removal.

🚨 **TECH DEBT — REMOVE WHEN GOALRAIL COMPAT WORKSTREAM ENDS.**
This entire module exists *only* to support the Goalrail
integration (see ``designs/GOALRAIL_INTEGRATION.md``). It
consolidates every goalrail-specific addition that would otherwise
be scattered across ``validator.py``, ``spec/__init__.py``, and
``runtime/workflow.py``.

When Goalrail is consolidated (phase 6 of the integration design),
deleting Goalrail support means:

1. Delete this file.
2. Remove the few lines in ``validator.py``,
   ``spec/__init__.py``, and ``runtime/workflow.py`` that import
   from it (each has a single import + a single call site —
   grep for ``_goalrail_compat`` to find them).
3. Delete ``goalrail/spec/goalrail.py`` (the bidirectional
   translator).
4. The Goalrail executor module is already gone (it held an
   experimental executor ABC that has since been removed), so
   there is nothing left to delete here.
5. Remove ``ExecutorSpec.config`` from
   ``goalrail/spec/types.py`` (the only field that couldn't
   move here because Python dataclasses don't support
   externally-added fields).

That's it. No grep-the-codebase exercise; the surface is
intentionally tiny.

**Do NOT** treat this module as a general-purpose extension point
for new executor types. Add concrete typed fields and dedicated
validator branches instead.
"""

from __future__ import annotations

from pathlib import Path
from typing import TYPE_CHECKING

import yaml

from goalrail.errors import ErrorCode, GoalrailError
from goalrail.harness_aliases import canonicalize_harness

if TYPE_CHECKING:
    from goalrail.spec.types import AgentSpec
    from goalrail.spec.validator import ValidationResult


# ── Constants ──────────────────────────────────────────────────


# Value placed in :attr:`AgentSpec.executor.type` so the runtime
# selects ``GoalrailExecutor``. Single source of truth — every
# goalrail-aware site imports from here, no string duplication.
GOALRAIL_EXECUTOR_TYPE = "goalrail"


# Harness identifiers accepted by ``executor.config.harness`` when
# ``executor.type == "goalrail"``. Matches the set of internal-loop
# harnesses ``GoalrailExecutor`` wraps. Unsupported provider names are
# intentionally excluded so a provider/harness mix-up fails at spec
# validation time. See designs/GOALRAIL_INTEGRATION.md §1.
#
# ``open-responses`` is the OpenAI Responses-API harness that
# ``goalrail.inner.open_responses_sdk.OpenResponsesExecutor``
# implements; the executor_factory resolves it when the YAML
# declares ``harness: open-responses``, so the adapter must
# accept it too. It was missing from the initial allowlist, which
# made ``examples/terminal_workers.yaml``
# fail at spec-load with a "must be one of [...], got
# 'open-responses'" error.
#
# ``opencode-native`` is the native OpenCode server bridge (runner-owned
# ``opencode serve`` + SSE forwarder); its ``opencode`` / ``native-opencode``
# spellings are accepted aliases below.
GOALRAIL_HARNESSES = frozenset(
    {
        "antigravity",
        "antigravity-native",
        "claude-native",
        "claude-sdk",
        "codex",
        "codex-native",
        "copilot",
        "cursor",
        "kimi",
        "kimi-native",
        "cursor-native",
        "kiro-native",
        "goose",
        "goose-native",
        "hermes",
        "hermes-native",
        "openai-agents",
        "open-responses",
        "opencode-native",
        "pi",
        "pi-native",
        "qwen",
        "qwen-native",
    },
)
# User-facing aliases accepted in specs and normalized before runtime dispatch.
GOALRAIL_HARNESS_ALIASES = frozenset(
    {
        "claude",
        "native-kiro",
        "native-pi",
        "native-antigravity",
        "native-goose",
        "openai-agents-sdk",
        "agy",
        "google-antigravity",
        "kimi-code",
        "qwen-code",
        "opencode",
        "native-opencode",
        "native-hermes",
        "github-copilot",
        "native-kimi",
    }
)
_GOALRAIL_ACCEPTED_HARNESSES = GOALRAIL_HARNESSES | GOALRAIL_HARNESS_ALIASES


# Top-level YAML keys that identify an goalrail single-file
# agent spec. ``name`` is always required. The system-prompt key
# may be either ``prompt:`` (legacy spec) or
# ``instructions:`` (cross-format alias added to match native AP
# YAML). At least one must be present so the agent has a system
# prompt; YAMLs with neither still fail loud at translation time
# (the resulting agent would have no instructions, which is
# nearly always a typo).
_GOALRAIL_NAME_KEY = "name"
_GOALRAIL_SYSTEM_PROMPT_KEYS = frozenset({"prompt", "instructions"})
_GOALRAIL_DISCRIMINATOR_KEY = "spec_version"


# ── Validator: goalrail executor branch ──────────────────────


def validate_goalrail_executor(
    spec: AgentSpec,
    result: ValidationResult,
) -> None:
    """
    Validate fields for ``executor.type: goalrail``.

    The goalrail executor wraps an goalrail harness subprocess.
    ``executor.config.harness`` is optional — when absent, the
    goalrail factory selects a default. When set, it must be one
    of :data:`GOALRAIL_HARNESSES`.

    The goalrail harness manages its own context window, so
    ``compaction`` is invalid.

    :param spec: The agent spec to check.
    :param result: Accumulator for any validation errors found.
    """
    if spec.compaction is not None:
        result.add(
            "compaction",
            f"not supported when executor.type is {GOALRAIL_EXECUTOR_TYPE!r}"
            " — harness manages context internally",
        )
    harness = spec.executor.config.get("harness")
    if not harness:
        result.add(
            "executor.config.harness",
            f"required when executor.type is {GOALRAIL_EXECUTOR_TYPE!r} — "
            f"must be one of {sorted(_GOALRAIL_ACCEPTED_HARNESSES)}",
        )
    elif canonicalize_harness(harness) not in GOALRAIL_HARNESSES:
        result.add(
            "executor.config.harness",
            f"must be one of {sorted(_GOALRAIL_ACCEPTED_HARNESSES)}, got {harness!r}",
        )


# ── YAML detection + loading ───────────────────────────────────


def is_goalrail_yaml(path: Path) -> bool:
    """
    Return ``True`` if *path* is an goalrail single-file YAML spec.

    Detection rule (from GOALRAIL_INTEGRATION design):

    - The file extension is ``.yaml`` or ``.yml``.
    - The top-level YAML document is a mapping.
    - The mapping has both ``name`` AND ``prompt`` keys.
    - The mapping does NOT have a ``spec_version`` key (which would
      identify an goalrail spec).

    Malformed YAML or non-mapping root documents return ``False`` —
    the caller (``load``) then takes its existing path and raises an
    informative error downstream.

    :param path: Path to a file on disk, already known to exist.
    :returns: ``True`` when *path* is an goalrail YAML per the rule
        above, ``False`` otherwise.
    """
    if path.suffix.lower() not in {".yaml", ".yml"}:
        return False
    try:
        raw = yaml.safe_load(path.read_text())
    except yaml.YAMLError:
        return False
    if not isinstance(raw, dict):
        return False
    if _GOALRAIL_DISCRIMINATOR_KEY in raw:
        return False
    if _GOALRAIL_NAME_KEY not in raw:
        return False
    # At least one system-prompt key must be present.
    return bool(_GOALRAIL_SYSTEM_PROMPT_KEYS.intersection(raw.keys()))


def diagnose_yaml_rejection(path: Path) -> str:
    """
    Explain why *path* failed :func:`is_goalrail_yaml`.

    Used by ``goalrail.spec.load`` to produce an actionable error
    message when a ``.yaml`` / ``.yml`` file is passed in but
    doesn't satisfy the goalrail-YAML detection rule. Without
    this, ``load`` falls through to the tarball-extraction branch
    and emits ``"dest is required when loading from a tarball"`` —
    technically correct (the path isn't a known YAML shape and
    isn't a directory) but useless to the user, who edited a YAML
    file and wants to know what's wrong with it.

    The return value is a single-line human-readable diagnosis
    suitable for embedding in an :class:`GoalrailError` message.

    :param path: A file path that already failed
        :func:`is_goalrail_yaml`. Caller is responsible for
        ensuring ``path.exists()`` and ``path.is_file()``.
    :returns: A short diagnostic string explaining the rejection,
        e.g. ``"missing required key 'prompt'"``,
        ``"top-level YAML must be a mapping (got list)"``, or
        ``"YAML parse error at line 3"``.
    """
    if path.suffix.lower() not in {".yaml", ".yml"}:
        return f"file extension is {path.suffix!r}, expected '.yaml' or '.yml'"
    try:
        raw = yaml.safe_load(path.read_text())
    except yaml.YAMLError as exc:
        # Strip trailing whitespace so the message stays one line —
        # PyYAML embeds the source location in its error string,
        # which is exactly what the user needs to fix the typo.
        return f"YAML parse error: {exc!s}".replace("\n", " ").rstrip()
    if raw is None:
        return "file is empty (or contains only YAML comments / null)"
    if not isinstance(raw, dict):
        return (
            f"top-level YAML must be a mapping (got "
            f"{type(raw).__name__}); expected keys 'name' and 'prompt'"
        )
    if _GOALRAIL_DISCRIMINATOR_KEY in raw:
        return (
            "file declares 'spec_version' which marks it as an goalrail "
            "spec — goalrail specs must live in a directory with a "
            "'config.yaml' (and any bundled assets), not as a single "
            "YAML file. Either remove 'spec_version' (to use the "
            "goalrail single-file format) or move the YAML into a "
            "bundle directory named 'config.yaml'."
        )
    if _GOALRAIL_NAME_KEY not in raw:
        return "missing required key 'name'. An goalrail YAML must declare a top-level 'name'."
    if not _GOALRAIL_SYSTEM_PROMPT_KEYS.intersection(raw.keys()):
        return (
            "missing system-prompt key. An goalrail YAML must declare "
            "either 'prompt:' (inline text) or 'instructions:' (path to "
            "a sibling file or inline text) at the top level."
        )
    # Should be unreachable: if all checks pass, ``is_goalrail_yaml``
    # would have returned True. Guard against a future divergence
    # between the two functions.
    return "unknown reason — file passes all known checks (likely an internal bug)"


def load_goalrail_yaml(
    path: Path,
    *,
    enforce_handler_allowlist: bool = False,
    prune_invalid_sub_agents: bool = False,
) -> AgentSpec:
    """
    Load an goalrail YAML and translate it to an
    :class:`AgentSpec`.

    Pipeline: ``goalrail.loader.load_agent_def(path)`` →
    :func:`goalrail.spec.goalrail.agent_def_to_agent_spec` →
    :func:`goalrail.spec.validator.validate`. Validation failure
    raises :class:`GoalrailError` so the caller sees the specific
    field that doesn't translate (per the fail-loud discipline).

    :param path: Path to an goalrail YAML file. Caller has
        already verified via :func:`is_goalrail_yaml`.
    :param enforce_handler_allowlist: Forwarded to
        :func:`goalrail.inner.loader.load_agent_def` — when ``True``,
        unregistered ``type: function`` policy handlers are rejected
        before the loader resolves/calls them (bundle-upload
        guard). See :func:`goalrail.spec.load`.
    :param prune_invalid_sub_agents: When ``True``, sub-agents that
        fail validation are dropped (and their ``tools.agents``
        references removed) instead of failing the whole load, with a
        WARNING logged per drop. The root agent must still validate.
        See :func:`goalrail.spec.load` for the full rationale — this
        is the execution-path backwards-compatibility guard.
    :returns: A validated :class:`AgentSpec` with
        ``executor.type == GOALRAIL_EXECUTOR_TYPE``.
    :raises GoalrailError: If the synthesized spec fails
        validation (e.g. policy translation gap), or if the
        ``goalrail`` package is not installed in the current
        Python environment.
    """
    try:
        from goalrail.inner.loader import load_agent_def
    except ImportError as exc:
        # Agent-plane can be pip-installed without the goalrail
        # source alongside (the repo layout has them as siblings,
        # but editable installs of goalrail into a fresh env
        # don't pull goalrail in). Surface a clear install hint
        # instead of a bare ``ModuleNotFoundError``.
        raise GoalrailError(
            "loading goalrail-format YAMLs requires the "
            "``goalrail`` package to be importable. Install it "
            "(``pip install -e <goalrail-root>`` from the "
            "repo, or add the root to PYTHONPATH) and retry. The "
            "failing import was: "
            f"{exc}",
            code=ErrorCode.INVALID_INPUT,
        ) from exc

    import yaml as _yaml

    from goalrail.inner.loader import _GoalrailYamlLoader
    from goalrail.spec.goalrail import agent_def_to_agent_spec
    from goalrail.spec.validator import validate

    agent_def = load_agent_def(path, enforce_handler_allowlist=enforce_handler_allowlist)
    # Read the raw YAML alongside so the translator can preserve
    # policy-level YAML fields that the goalrail loader drops
    # (label policies in particular compile to synthetic
    # FunctionPolicy callables, losing ``condition``,
    # ``match_tools``, ``action``, ``reason``, ``set_labels``).
    # Non-mapping roots are tolerated as an empty dict — the
    # goalrail loader would already have rejected them above.
    # Use _GoalrailYamlLoader (not yaml.safe_load) so that
    # booleans parse consistently — importing load_agent_def
    # mutates yaml.SafeLoader's implicit resolvers as a side
    # effect, causing yaml.safe_load to return string "false"
    # for unquoted ``false`` values (e.g. use_responses: false).
    raw = _yaml.load(path.read_text(), Loader=_GoalrailYamlLoader) or {}
    if not isinstance(raw, dict):
        raw = {}
    spec = agent_def_to_agent_spec(agent_def, raw_yaml=raw)
    if prune_invalid_sub_agents:
        # Local import avoids a module-load cycle: spec/__init__ imports
        # this module at import time, so it cannot be imported at the top
        # here. Drops sub-agents this client can't validate (version skew)
        # before the root validation gate below; see goalrail.spec.load.
        from goalrail.spec import _prune_invalid_sub_agents

        _prune_invalid_sub_agents(spec)
    result = validate(spec)
    if not result.valid:
        errors = "; ".join(f"{e.path}: {e.message}" for e in result.errors)
        message = f"invalid agent spec synthesized from goalrail YAML: {errors}"
        # An unrecognized harness *value* usually means this client
        # (the goalrail runner validating the spec) is older than the
        # server that produced it: the server knows a harness this
        # runner's allowlist doesn't. Surface that so the operator
        # checks for a version skew before assuming the spec is wrong.
        #
        # The ``"must be one of"`` prefix is the wording emitted by
        # ``validate_goalrail_executor`` (same module) for an
        # out-of-allowlist harness. It deliberately does NOT match the
        # sibling "required when executor.type is 'goalrail' — must be
        # one of ..." message for a *missing* harness, which is a plain
        # authoring mistake, not a version skew. Producer and matcher
        # live in this file, so the coupling stays local; if that
        # message is reworded, update both together.
        if any(
            e.path == "executor.config.harness" and e.message.startswith("must be one of")
            for e in result.errors
        ):
            message += (
                "\n\nNote: if this harness is valid on a newer Goalrail server, "
                "this client (runner) may be older than the server that produced "
                "the spec — upgrade the runner to pick up newer harnesses."
            )
        raise GoalrailError(message, code=ErrorCode.INVALID_INPUT)
    return spec
