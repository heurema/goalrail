"""E2E coverage for the ``GOALRAIL_MODEL`` env-var fallback on ``goalrail run``.

The fallback fires in ``goalrail/chat.py:_apply_overrides_to_raw`` when the
spec has no ``executor.model`` / ``executor.harness`` and no ``--model`` /
``--harness`` flag is passed. Helper-level coverage lives in
``tests/cli/test_chat.py``; this file spawns a real subprocess so a regression
between the helper and the FM API surfaces too.
"""

from __future__ import annotations

import subprocess
from pathlib import Path

from tests.e2e._run_with_group_timeout import run_with_group_timeout
from tests.e2e.goalrail.conftest import configure_mock_llm

# ``databricks-`` prefix is load-bearing on two counts:
# 1. ``databricks-`` exempts ``llm.connection`` from
#    ``goalrail.spec.validator._validate_executor_llm``; any other prefix
#    rejects the YAML before any FM API call happens.
# 2. ``databricks-gpt-`` routes through ``goalrail.llms.routing.infer_
#    harness_from_model`` to ``openai-agents``; a bare ``databricks-`` prefix
#    leaves ``executor.harness=""`` and the runtime wedges (no validator
#    catches the empty harness when ``llm.model`` is set).
#
# The valid model is set to ``mock-model`` (routed to the "default" key
# of the mock LLM queue, so any configured response is returned).
# For the bogus-model case we use a ``databricks-gpt-`` prefix so routing
# reaches the mock server, then configure an error response for that key.
_VALID_MODEL = "mock-model"
_BOGUS_MODEL = "databricks-gpt-this-model-does-not-exist-goalrail-env-test-9f3a"

_PROMPT = "say hi in 5 words"
# Wall-clock budget for the subprocess. ``goalrail run`` spawns the
# AP server + runner as grandchildren, so a plain ``subprocess.run``
# timeout could not reap them — the grandchildren kept the captured
# pipe open and ``communicate()`` wedged the shard ~15+ min past the
# nominal timeout (the bug that suppressed
# ``test_goalrail_model_env_var_bogus_value_fails_with_named_error``).
# ``run_with_group_timeout`` SIGKILLs the whole process group at the
# deadline, so the budget below is a hard ceiling regardless of how
# the grandchildren behave. A bogus model 404s on the first FM API
# call (404 is not in the SDK's retryable set), so the negative case
# resolves in seconds; the positive sibling is one short turn. 120s
# covers server+runner cold-start plus a slow gateway day on the
# positive path while staying under the CI per-test --timeout=180
# cap, so the group cleanup fires before pytest's thread-timeout
# gives up. Either way the shard can no longer wedge for minutes.
_RUN_TIMEOUT_SEC = 120.0
_MIN_ASSISTANT_CHARS = 4

_MINIMAL_YAML = (
    "name: hello_world\nprompt: You are a friendly assistant. Say hello and answer questions.\n"
)


def _run_goalrail_with_model_env(
    *,
    model_env_value: str,
    goalrail_python: Path,
    goalrail_repo_root: Path,
    mock_credentials_env: dict[str, str],
    tmp_path: Path,
    harness: str | None = None,
) -> subprocess.CompletedProcess[str]:
    """
    Run ``goalrail run <minimal>.yaml -p "..."`` with ``GOALRAIL_MODEL`` set.

    Writes a minimal no-``executor`` YAML to *tmp_path*; reusing the shared
    ``hello_world.yaml`` would defeat the test because that file declares
    ``executor.model``, which short-circuits the env-var fallback gate.

    Uses :func:`run_with_group_timeout` rather than ``subprocess.run``
    because ``goalrail run`` spawns the AP server + runner as
    grandchildren in the same process group; a stock ``subprocess.run``
    timeout only kills the immediate child, leaving the grandchildren to
    hold the captured pipe open and wedge ``communicate()`` long past the
    deadline.

    :param model_env_value: ``GOALRAIL_MODEL`` value (real or bogus).
    :param tmp_path: Per-test tmp dir for the minimal YAML.
    :param mock_credentials_env: Mock-LLM env vars pointing at the
        mock server.
    :returns: Subprocess result with stdout/stderr captured as text.
    :raises subprocess.TimeoutExpired: When the run exceeds
        ``_RUN_TIMEOUT_SEC``; the whole process group is SIGKILLed and
        any captured stdout/stderr is attached to the exception.
    """
    yaml_path = tmp_path / "hello_world_no_executor.yaml"
    yaml_path.write_text(_MINIMAL_YAML)
    env = dict(mock_credentials_env)
    env["GOALRAIL_MODEL"] = model_env_value
    return run_with_group_timeout(
        [
            str(goalrail_python),
            "-m",
            "goalrail",
            "run",
            str(yaml_path),
            "-p",
            _PROMPT,
            "--no-session",
        ]
        + (["--harness", "openai-agents"] if harness else []),
        env=env,
        cwd=str(goalrail_repo_root),
        capture_output=True,
        text=True,
        timeout=_RUN_TIMEOUT_SEC,
    )


def test_goalrail_model_env_var_drives_successful_run(
    goalrail_python: Path,
    goalrail_repo_root: Path,
    mock_credentials_env: dict[str, str],
    mock_llm_server_url: str,
    tmp_path: Path,
) -> None:
    """
    Smoke test: a valid model in ``GOALRAIL_MODEL`` produces a successful turn.

    A pass alone doesn't prove the env var was honored (the default model also
    succeeds); the bogus-value sibling carries the decisive proof. This test
    catches the env-var path going from "silently dropped" to "actively broken".

    :param goalrail_python: Interpreter with goalrail installed.
    :param goalrail_repo_root: Repo root (subprocess cwd).
    :param mock_credentials_env: Mock-LLM env vars.
    :param mock_llm_server_url: Mock server URL for configuring queues.
    :param tmp_path: Per-test tmp dir for the minimal YAML.
    """
    configure_mock_llm(mock_llm_server_url, [{"text": "hi there"}])
    result = _run_goalrail_with_model_env(
        model_env_value=_VALID_MODEL,
        goalrail_python=goalrail_python,
        goalrail_repo_root=goalrail_repo_root,
        mock_credentials_env=mock_credentials_env,
        tmp_path=tmp_path,
        harness="openai-agents",
    )

    # Non-zero exit means either the env var never reached the executor block
    # or the resolved model failed at the FM API — both silently break users.
    assert result.returncode == 0, (
        f"goalrail run with GOALRAIL_MODEL={_VALID_MODEL!r} exited "
        f"with code {result.returncode}.\n"
        f"stdout: {result.stdout!r}\nstderr: {result.stderr!r}"
    )

    # Short / empty reply hints at a downgraded model or output-extraction regression.
    text = result.stdout.strip()
    assert len(text) >= _MIN_ASSISTANT_CHARS, (
        f"Expected assistant reply >= {_MIN_ASSISTANT_CHARS} chars; "
        f"got {len(text)} (stdout={text!r})."
    )


def test_goalrail_model_env_var_bogus_value_fails_with_named_error(
    goalrail_python: Path,
    goalrail_repo_root: Path,
    mock_credentials_env: dict[str, str],
    mock_llm_server_url: str,
    tmp_path: Path,
) -> None:
    """
    Decisive test: a bogus ``GOALRAIL_MODEL`` fails with the bogus name in stderr.

    A failure that names the sentinel can only happen if the env-var value
    traveled the full pipeline to the FM API. If the env var were silently
    dropped, the default model would succeed (or fail with its own name).

    The mock server is configured with an error response keyed to the bogus
    model name so the subprocess fails decisively without hitting a real
    gateway.

    :param goalrail_python: Interpreter with goalrail installed.
    :param goalrail_repo_root: Repo root (subprocess cwd).
    :param mock_credentials_env: Mock-LLM env vars.
    :param mock_llm_server_url: Mock server URL for configuring queues.
    :param tmp_path: Per-test tmp dir for the minimal YAML.
    """
    # Configure an error response keyed to the bogus model name.
    # The mock server resolves the queue by matching the request's
    # ``model`` field — if the bogus model env var travels through
    # the pipeline correctly, the server returns a 404 error that
    # names the model; if the env var is silently dropped, the
    # default queue fires a success response and the test fails
    # on the ``returncode != 0`` assertion below.
    configure_mock_llm(
        mock_llm_server_url,
        [
            {
                "error": f"model not found: {_BOGUS_MODEL}",
                "status_code": 404,
            }
        ],
        key=_BOGUS_MODEL,
    )
    result = _run_goalrail_with_model_env(
        model_env_value=_BOGUS_MODEL,
        goalrail_python=goalrail_python,
        goalrail_repo_root=goalrail_repo_root,
        mock_credentials_env=mock_credentials_env,
        tmp_path=tmp_path,
    )

    # Exit 0 means the env var was dropped and the default model took over.
    assert result.returncode != 0, (
        f"goalrail run with GOALRAIL_MODEL={_BOGUS_MODEL!r} unexpectedly "
        f"succeeded (exit 0); the env var was silently dropped.\n"
        f"stdout: {result.stdout!r}\nstderr: {result.stderr!r}"
    )

    # The bogus model name must appear in combined output — the mock
    # server echoes it in the error message, which travels through the
    # SDK's exception path to stderr. This proves the env-var value
    # reached the FM API call rather than being silently discarded.
    combined = result.stdout + result.stderr
    assert _BOGUS_MODEL in combined, (
        f"Bogus model {_BOGUS_MODEL!r} not in subprocess output — either the "
        f"env var was dropped and the default model took over, or the mock "
        f"server's error response was not surfaced in the exception.\n"
        f"stdout: {result.stdout!r}\nstderr: {result.stderr!r}"
    )
