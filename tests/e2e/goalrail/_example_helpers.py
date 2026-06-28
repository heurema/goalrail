"""Shared helpers for the per-example e2e tests under ``test_example_*.py``.

Each example under ``examples/`` has its own test file that
exercises that example's intended functionality through the real
``goalrail run`` subprocess. The helpers in this module keep the
per-file boilerplate minimal: resolve the YAML path, build argv,
run subprocess, assert common invariants.

Not a conftest because these are called explicitly by test functions
(not picked up as fixtures).
"""

from __future__ import annotations

import shutil
import subprocess
from pathlib import Path

import pytest

from tests._model_pools import resolve_model
from tests.e2e._run_with_group_timeout import run_with_group_timeout

# Default low-effort prompt used by examples whose purpose is to
# demonstrate a feature but not run heavy tool logic on every call.
# Per-example tests override this for features that only fire with a
# specific prompt shape.
DEFAULT_PROMPT = "Reply with just the word 'OK'."

# Default harness + model when a YAML doesn't pin one. We prefer
# the openai-agents harness against dogfood GPT-5-mini because it
# honors OPENAI_BASE_URL / OPENAI_API_KEY env vars directly (no
# ~/provider config patching), which matches how the rest of the e2e
# suite authenticates.
DEFAULT_HARNESS = "openai-agents"
DEFAULT_MODEL = resolve_model("openai/gpt-5-mini", key=__name__)

# Subprocess wall-clock budget. The openai-agents harness against
# dogfood typically finishes a one-turn reply in 5-15s; 180s covers
# cold-start + slow days.
RUN_TIMEOUT_SEC = 180


def example_yaml_path(repo_root: Path, name: str) -> Path:
    """
    Return the YAML / config file path for an example agent.

    Resolution order (first hit wins):

    1. ``examples/<name>.yaml`` — shipped examples that remain in
       the top-level ``examples/`` directory.
    2. ``tests/resources/examples/<name>.yaml`` — single-YAML demos
       moved to the test-resources tree.
    3. ``tests/resources/examples/<name>/`` — multi-file AGENTSPEC
       demos (``archer``, ``coder``, ``openai-coder``).
    4. ``tests/resources/agents/<name>/`` — test-only fixtures
       (aspirational specs, incremental-feature variants).

    :param repo_root: Unified repo root; typically the
        ``goalrail_repo_root`` fixture value.
    :param name: Agent name. For top-level YAMLs, the filename
        stem (``"hello_world"`` → ``tests/resources/examples/hello_world.yaml``);
        for dir-shaped agents, the directory name (``"archer"``).
    :returns: Absolute :class:`Path` to the YAML / config file.
    :raises FileNotFoundError: When none of the layouts match in
        any of the roots.
    """
    # Shipped examples still in examples/ (1).
    top_level = repo_root / "examples" / f"{name}.yaml"
    if top_level.is_file():
        return top_level
    # Test-resource single-YAML demos (2).
    res_top = repo_root / "tests" / "resources" / "examples" / f"{name}.yaml"
    if res_top.is_file():
        return res_top
    # Dir-shaped demos + test fixtures (3, 4).
    for root in (
        repo_root / "tests" / "resources" / "examples",
        repo_root / "tests" / "resources" / "agents",
    ):
        agent_dir = root / name
        legacy = agent_dir / f"{name}.yaml"
        if legacy.is_file():
            return legacy
        agentspec = agent_dir / "config.yaml"
        if agentspec.is_file():
            return agentspec
    raise FileNotFoundError(
        f"No YAML for agent {name!r} — checked "
        f"examples/{name}.yaml, "
        f"tests/resources/examples/{name}.yaml, "
        f"tests/resources/examples/{name}/ and tests/resources/agents/{name}/ "
        f"for both LEGACY ({name}.yaml) and AGENTSPEC (config.yaml) "
        f"layouts."
    )


def run_one_shot(
    *,
    goalrail_python: Path,
    goalrail_repo_root: Path,
    goalrail_credentials_env: dict[str, str],
    example_name: str,
    prompt: str = DEFAULT_PROMPT,
    harness: str | None = DEFAULT_HARNESS,
    model: str | None = DEFAULT_MODEL,
) -> subprocess.CompletedProcess[str]:
    """
    Invoke ``goalrail run <yaml> -p <prompt>`` one-shot.

    :param goalrail_python: Interpreter with goalrail + required
        SDKs installed. Provided by the ``goalrail_python`` fixture.
    :param goalrail_repo_root: Cwd so the YAML's ``callable:``
        dotted paths resolve via repo-root-on-sys.path. Provided by
        the ``goalrail_repo_root`` fixture.
    :param goalrail_credentials_env: PAT + BASE_URL + profile env
        populated from ``--llm-api-key``. Provided by the
        ``goalrail_credentials_env`` fixture.
    :param example_name: Agent name; see :func:`example_yaml_path`
        for resolution order (top-level YAML, AGENTSPEC dir under
        ``examples/``, or test fixture under
        ``tests/resources/agents/``).
    :param prompt: User message; default ``DEFAULT_PROMPT``.
    :param harness: ``--harness`` value, or ``None`` to let the
        YAML's ``executor.type`` win (used when the YAML pins a
        specific harness like ``claude_sdk``).
    :param model: ``--model`` override, only passed when *harness*
        is non-None (co-selected).
    :returns: The completed subprocess. Caller decides which
        fields to assert.
    """
    yaml_path = example_yaml_path(goalrail_repo_root, example_name)
    return run_one_shot_at_path(
        goalrail_python=goalrail_python,
        goalrail_repo_root=goalrail_repo_root,
        goalrail_credentials_env=goalrail_credentials_env,
        yaml_path=yaml_path,
        prompt=prompt,
        harness=harness,
        model=model,
    )


def run_one_shot_at_path(
    *,
    goalrail_python: Path,
    goalrail_repo_root: Path,
    goalrail_credentials_env: dict[str, str],
    yaml_path: Path,
    prompt: str = DEFAULT_PROMPT,
    harness: str | None = DEFAULT_HARNESS,
    model: str | None = DEFAULT_MODEL,
) -> subprocess.CompletedProcess[str]:
    """
    Like :func:`run_one_shot` but takes an arbitrary ``yaml_path``
    (a file or AGENTSPEC directory) rather than an
    ``examples/<name>`` lookup.

    Needed by tests that materialize a rewritten YAML in
    ``tmp_path`` (e.g. MCP-profile overrides, goalrail endpoint
    rewrites).

    :param goalrail_python: Interpreter with goalrail installed.
    :param goalrail_repo_root: Cwd (repo root so module lookups
        land).
    :param goalrail_credentials_env: Env.
    :param yaml_path: Absolute path to the YAML file or directory.
    :param prompt: User message.
    :param harness: ``--harness`` value or ``None``.
    :param model: ``--model`` value or ``None``.
    :returns: The completed subprocess.
    """
    argv: list[str] = [
        str(goalrail_python),
        "-m",
        "goalrail",
        "run",
        str(yaml_path),
        "-p",
        prompt,
        "--no-log",
        "--no-session",
    ]
    if harness is not None:
        argv.extend(["--harness", harness])
        if model is not None:
            argv.extend(["--model", model])
    # run_with_group_timeout, not subprocess.run: grandchildren
    # (server / runner / harness) hold the pipes past timeout.
    return run_with_group_timeout(
        argv,
        env=goalrail_credentials_env,
        cwd=str(goalrail_repo_root),
        capture_output=True,
        text=True,
        timeout=RUN_TIMEOUT_SEC,
    )


def validate_agent_def_structure(
    *,
    goalrail_python: Path,
    goalrail_repo_root: Path,
    example_name: str,
    expected_name: str,
    expected_tools: set[str] | None = None,
    expected_policies: set[str] | None = None,
    expected_executor_type: str | None = None,
    expected_executor_harness: str | None = None,
    expected_terminals: set[str] | None = None,
    expected_os_env_type: str | None = None,
) -> None:
    """
    Parse + translate an example YAML and assert the resulting
    :class:`AgentDef` has the structural shape we expect.

    Used for examples that can't run end-to-end on a laptop
    (hosted MCP servers needing OAuth, live Glean / Google
    profiles) but whose *spec translation* we still want to
    guard against regressions. Exercises the unified spec parser,
    the :func:`agent_spec_to_agent_def` translator, and the
    per-tool / per-policy / per-terminal registration paths —
    all of which the unification touched.

    :param goalrail_python: Interpreter with goalrail installed.
    :param goalrail_repo_root: Cwd so module-path resolution
        matches the CLI.
    :param example_name: Example directory under ``examples/``.
    :param expected_name: ``AgentDef.name`` must equal this
        exactly, e.g. ``"gateway_mcps_agent"``.
    :param expected_tools: Tool names that must be present on
        ``AgentDef.tools`` (subset-match; extras don't fail).
        ``None`` means skip the tool check.
    :param expected_policies: Policy names (as declared in the
        YAML's ``policies:`` dict). ``None`` skips.
    :param expected_executor_type: ``AgentDef.executor.type``
        when set by the YAML. ``None`` skips.
    :param expected_executor_harness: ``AgentDef.executor.harness``.
        ``None`` skips.
    :param expected_terminals: Names that must be present in
        ``AgentDef.terminals``.
    :param expected_os_env_type: ``AgentDef.os_env.type`` when
        set. ``None`` skips.
    """
    yaml_path = example_yaml_path(goalrail_repo_root, example_name)
    # Build a snippet that loads the agent def, then prints a
    # small JSON summary we can assert against. Using JSON
    # instead of a bunch of ``assert``s inside the snippet keeps
    # the failure message at the pytest-side under our control.
    snippet = f"""
import json
import sys
sys.path.insert(0, {str(goalrail_repo_root)!r})
from goalrail.inner.loader import load_agent_def_from_path

agent_def = load_agent_def_from_path({str(yaml_path)!r})
assert agent_def is not None, "load returned None"

tool_names = sorted(agent_def.tools.keys()) if agent_def.tools else []
policy_names = (
    sorted(agent_def.policies.keys())
    if getattr(agent_def, "policies", None)
    else []
)
terminals = (
    sorted(agent_def.terminals.keys())
    if getattr(agent_def, "terminals", None)
    else []
)
executor = agent_def.executor
os_env = agent_def.os_env
summary = {{
    "name": agent_def.name,
    "tools": tool_names,
    "policies": policy_names,
    "terminals": terminals,
    "executor_type": getattr(executor, "type", None) if executor else None,
    "executor_harness": (
        getattr(executor, "harness", None) if executor else None
    ),
    "os_env_type": getattr(os_env, "type", None) if os_env else None,
}}
print("SUMMARY:" + json.dumps(summary))
"""
    result = subprocess.run(
        [str(goalrail_python), "-c", snippet],
        cwd=str(goalrail_repo_root),
        capture_output=True,
        text=True,
        timeout=30,
    )
    assert result.returncode == 0, (
        f"Structural check failed for {example_name!r}:\n"
        f"stdout={result.stdout!r}\nstderr={result.stderr!r}"
    )
    # Extract the JSON summary — other print()s (e.g. DBOS init
    # noise) may precede it.
    import json

    summary_line = next(
        (line for line in result.stdout.splitlines() if line.startswith("SUMMARY:")),
        None,
    )
    assert summary_line is not None, (
        f"Didn't reach the SUMMARY print in the load snippet for "
        f"{example_name!r}. stdout={result.stdout!r}"
    )
    summary = json.loads(summary_line[len("SUMMARY:") :])

    assert summary["name"] == expected_name, (
        f"{example_name!r}: AgentDef.name expected {expected_name!r}, got {summary['name']!r}"
    )

    if expected_tools is not None:
        actual_tools = set(summary["tools"])
        missing = expected_tools - actual_tools
        assert not missing, (
            f"{example_name!r}: expected tools missing from AgentDef: "
            f"{sorted(missing)}. Got: {sorted(actual_tools)}"
        )

    if expected_policies is not None:
        actual_policies = set(summary["policies"])
        missing = expected_policies - actual_policies
        assert not missing, (
            f"{example_name!r}: expected policies missing from "
            f"AgentDef: {sorted(missing)}. Got: {sorted(actual_policies)}"
        )

    if expected_terminals is not None:
        actual_terminals = set(summary["terminals"])
        missing = expected_terminals - actual_terminals
        assert not missing, (
            f"{example_name!r}: expected terminals missing from "
            f"AgentDef: {sorted(missing)}. Got: {sorted(actual_terminals)}"
        )

    if expected_executor_type is not None:
        assert summary["executor_type"] == expected_executor_type, (
            f"{example_name!r}: executor.type expected "
            f"{expected_executor_type!r}, got {summary['executor_type']!r}"
        )

    if expected_executor_harness is not None:
        assert summary["executor_harness"] == expected_executor_harness, (
            f"{example_name!r}: executor.harness expected "
            f"{expected_executor_harness!r}, got "
            f"{summary['executor_harness']!r}"
        )

    if expected_os_env_type is not None:
        assert summary["os_env_type"] == expected_os_env_type, (
            f"{example_name!r}: os_env.type expected "
            f"{expected_os_env_type!r}, got {summary['os_env_type']!r}"
        )


def assert_completed_one_shot(
    result: subprocess.CompletedProcess[str],
    example_name: str,
) -> None:
    """
    Assert a one-shot ``goalrail run`` finished cleanly.

    :param result: The completed subprocess, as returned by
        :func:`run_one_shot`.
    :param example_name: Example name; used only in error messages
        so a failure inside a parametrized-like test file still
        names the specific example.
    """
    assert result.returncode == 0, (
        f"{example_name!r}: goalrail run exited with "
        f"{result.returncode} (expected 0).\n"
        f"stdout:\n{result.stdout}\nstderr:\n{result.stderr}"
    )
    # Any assistant reply must reach stdout — zero-length stdout
    # means the run exited 0 without actually streaming anything.
    # --no-log strips the banner, so stdout == "" is a regression.
    assert result.stdout.strip(), (
        f"{example_name!r}: run exited 0 but produced no stdout. stderr:\n{result.stderr}"
    )


def require_claude_sdk() -> None:
    """
    Fail loud when the ``claude_agent_sdk`` package is missing.

    Examples that pin ``executor.type: claude_sdk`` import the
    upstream SDK at turn time; a missing package surfaces mid-run
    with an ImportError that obscures the root cause. We detect
    it upfront so the failure message names the missing dependency.
    """
    try:
        import claude_agent_sdk  # noqa: F401
    except ImportError:
        pytest.fail(
            "This example requires the 'claude-agent-sdk' Python "
            "package. Install it via the [claude-code] extra or "
            "explicitly: pip install claude-agent-sdk"
        )


def require_codex_cli() -> None:
    """
    Fail loud when the ``codex`` CLI binary is missing on PATH.

    Required by examples that pin ``harness: codex`` for any
    worker agent.
    """
    if shutil.which("codex") is None:
        pytest.fail(
            "codex CLI required but not on PATH. Install per the "
            "Codex project README before running this test."
        )
