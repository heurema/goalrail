#!/usr/bin/env python3
"""Fixture-backed tests for scripts/pr_intake_gate.py."""

from __future__ import annotations

import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))

from scripts.pr_intake_gate import (  # noqa: E402
    GateError,
    get_label_details,
    is_gate_comment,
    load_minimal_yaml,
    missing_required_sections,
    path_matches,
    run_optional_side_effect,
)

FULL_EXTERNAL_BODY = """## Goal / intent

Fix a concrete Goalrail contributor problem now.

## No-code alternative

No-code alternative is insufficient because the repository needs a deterministic check.

## Why code is needed

The repository needs an automated check rather than reviewer memory.

## ComponentImpact

- [x] `docs/ops/COMPONENTS.yaml` reviewed

Affected components:
- repository_governance

## DocImpact

Docs updated:
- CONTRIBUTING.md

## Rule Stack checklist

- [x] I checked `docs/product/GOALRAIL_RULE_STACK.md`
- [x] This PR does not expand MVP scope silently

## Validation / proof

Commands run / walkthrough:
- python3 tools/pr-intake-gate/test_pr_intake_gate.py

Evidence / proof:
- deterministic fixture output

Closes #42
"""

FULL_CONTEXT_NO_LINK_BODY = FULL_EXTERNAL_BODY.replace("\nCloses #42\n", "\n")

MISSING_NO_CODE_BODY = """## Goal / intent

Fix a concrete Goalrail contributor problem now.

## Why code is needed

The repository needs an automated check rather than reviewer memory.

## ComponentImpact

- [x] `docs/ops/COMPONENTS.yaml` reviewed

## DocImpact

Docs updated:
- CONTRIBUTING.md

## Rule Stack checklist

- [x] I checked `docs/product/GOALRAIL_RULE_STACK.md`

## Validation / proof

Commands run / walkthrough:
- python3 tools/pr-intake-gate/test_pr_intake_gate.py

Closes #42
"""

MISSING_CONTEXT_BODY = """## Goal / intent

Fix a concrete Goalrail contributor problem now.

## No-code alternative

No-code alternative is insufficient.

Closes #42
"""


def write_event(path: Path, body: str, labels: list[str], association: str, author_login: str = "contributor") -> None:
    event = {
        "repository": {"full_name": "heurema/goalrail"},
        "pull_request": {
            "number": 123,
            "title": "Test PR",
            "body": body,
            "author_association": association,
            "user": {"login": author_login},
            "labels": [{"name": label} for label in labels],
            "base": {"sha": "base-sha"},
            "head": {"sha": "head-sha"},
        },
    }
    path.write_text(json.dumps(event), encoding="utf-8")


def run_case(
    name: str,
    expected_status: int,
    expected_verdict: str,
    files: list[dict[str, object]],
    body: str = "",
    labels: list[str] | None = None,
    association: str = "CONTRIBUTOR",
    author_permission: str | None = None,
) -> tuple[dict[str, object], str]:
    labels = labels or []
    with tempfile.TemporaryDirectory(prefix=f"goalrail-pr-intake-{name}-") as tmp_raw:
        tmp = Path(tmp_raw)
        event_path = tmp / "event.json"
        summary_path = tmp / "summary.md"
        write_event(event_path, body, labels, association)

        env = os.environ.copy()
        env.update(
            {
                "GITHUB_EVENT_PATH": str(event_path),
                "GITHUB_STEP_SUMMARY": str(summary_path),
                "PR_INTAKE_GATE_CHANGED_FILES_JSON": json.dumps(files),
                "PR_INTAKE_GATE_DRY_RUN": "1",
            }
        )
        if author_permission is not None:
            env["PR_INTAKE_GATE_AUTHOR_PERMISSION"] = author_permission
        result = subprocess.run(
            [sys.executable, "scripts/pr_intake_gate.py"],
            cwd=ROOT,
            env=env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )

        if result.returncode != expected_status:
            raise AssertionError(
                f"{name}: expected exit {expected_status}, got {result.returncode}\n"
                f"stdout:\n{result.stdout}\nstderr:\n{result.stderr}"
            )
        payload = json.loads(result.stdout)
        if payload["verdict"] != expected_verdict:
            raise AssertionError(f"{name}: expected verdict {expected_verdict}, got {payload['verdict']}")
        summary = summary_path.read_text(encoding="utf-8")
        if "PR Intake Gate" not in summary:
            raise AssertionError(f"{name}: missing step summary")
        print(f"ok - {name}")
        return payload, result.stderr


def raise_gate_error() -> None:
    raise GateError("synthetic write failure")


def main() -> int:
    marker = "<!-- goalrail-pr-intake-gate -->"
    assert path_matches("README.md", "README.md")
    assert path_matches("docs/brand/INDEX.md", "docs/**/*.md")
    assert path_matches("docs/product/GOALRAIL_RULE_STACK.md", "docs/product/**")
    assert path_matches(".github/workflows/pr-intake-gate.yml", ".github/**")
    assert not path_matches("src/runtime.md", "*.md")
    assert not is_gate_comment({"body": marker, "user": {"login": "contributor", "type": "User"}}, marker)
    assert is_gate_comment({"body": marker, "user": {"login": "github-actions[bot]", "type": "Bot"}}, marker)
    config = load_minimal_yaml(str(ROOT / ".github" / "pr-intake-gate.yml"))
    assert get_label_details(config, "intake/pass")["color"] == "2ea44f"
    assert get_label_details(config, "intake/high-risk")["description"]
    assert not missing_required_sections(FULL_EXTERNAL_BODY, config["external_context"]["required_sections"])
    assert "No-code alternative" in missing_required_sections(MISSING_NO_CODE_BODY, config["external_context"]["required_sections"])
    assert run_optional_side_effect("test no-op", lambda: None) is True
    assert run_optional_side_effect("test failure", raise_gate_error) is False
    print("ok - helper semantics")

    trusted_permission, _ = run_case(
        "trusted_permission_passes_high_risk",
        0,
        "pass",
        [{"filename": ".github/workflows/docs-check.yml", "additions": 1, "deletions": 0}],
        association="CONTRIBUTOR",
        author_permission="admin",
    )
    assert trusted_permission["trusted_author"] is True
    assert trusted_permission["trust_source"] == "permission:admin"

    trusted_fallback, _ = run_case(
        "trusted_association_fallback_passes_high_risk",
        0,
        "pass",
        [{"filename": ".github/workflows/docs-check.yml", "additions": 1, "deletions": 0}],
        association="OWNER",
        author_permission="none",
    )
    assert trusted_fallback["trusted_author"] is True
    assert trusted_fallback["trust_source"] == "author_association:OWNER"

    run_case(
        "external_docs_only_passes",
        0,
        "pass",
        [{"filename": "docs/brand/README.md", "additions": 2, "deletions": 1}],
        author_permission="none",
    )
    run_case(
        "external_high_risk_fails",
        1,
        "high-risk",
        [{"filename": ".github/workflows/docs-check.yml", "additions": 1, "deletions": 0}],
        author_permission="none",
    )
    first_time, first_time_stderr = run_case(
        "first_time_external_high_risk_fails_with_signal",
        1,
        "high-risk",
        [{"filename": ".github/workflows/docs-check.yml", "additions": 1, "deletions": 0}],
        association="FIRST_TIMER",
        author_permission="none",
    )
    assert first_time["first_time_external"] is True
    assert "intake/first-time-contributor" in first_time_stderr
    run_case(
        "external_non_trivial_missing_no_code_fails",
        1,
        "no-code-alternative",
        [{"filename": "docs/brand/README.md", "additions": 31, "deletions": 0}],
        body=MISSING_NO_CODE_BODY,
        author_permission="none",
    )
    run_case(
        "external_non_trivial_missing_context_fails",
        1,
        "needs-more-context",
        [{"filename": "docs/brand/README.md", "additions": 31, "deletions": 0}],
        body=MISSING_CONTEXT_BODY,
        author_permission="none",
    )
    run_case(
        "external_full_context_without_link_fails",
        1,
        "needs-linked-intent",
        [{"filename": "docs/brand/README.md", "additions": 31, "deletions": 0}],
        body=FULL_CONTEXT_NO_LINK_BODY,
        author_permission="none",
    )
    run_case(
        "external_full_context_with_link_passes",
        0,
        "pass",
        [{"filename": "docs/brand/README.md", "additions": 31, "deletions": 0}],
        body=FULL_EXTERNAL_BODY,
        author_permission="none",
    )
    run_case(
        "accepted_external_non_high_risk_passes",
        0,
        "pass",
        [{"filename": "docs/brand/README.md", "additions": 31, "deletions": 0}],
        labels=["intake/accepted-for-pr"],
        author_permission="none",
    )
    run_case(
        "override_passes_external_high_risk",
        0,
        "pass",
        [{"filename": ".github/workflows/docs-check.yml", "additions": 1, "deletions": 0}],
        labels=["maintainer/override-intake"],
        author_permission="none",
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
