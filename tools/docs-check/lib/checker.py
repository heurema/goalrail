from __future__ import annotations

import json
import re
from datetime import date, datetime, timezone
from pathlib import Path, PurePosixPath
from typing import Any

from .components import ComponentsParseResult, parse_components_document
from .frontmatter import FrontmatterParseResult, parse_frontmatter


SCHEMA_VERSION = "goalrail.docs-check-report.v0"
REQUIRED_FRONTMATTER_FIELDS = [
    "id",
    "title",
    "kind",
    "authority",
    "status",
    "owner",
    "truth_surfaces",
    "lifecycle",
    "review_after",
    "supersedes",
    "superseded_by",
    "related_docs",
]
ENUMS = {
    "kind": {
        "product_canon",
        "architecture_canon",
        "ops_status",
        "ops_plan",
        "adr",
        "brand_canon",
        "public_entry",
        "research_note",
        "derived_view",
        "reference",
    },
    "authority": {
        "canonical",
        "operational",
        "advisory",
        "derived",
        "public_entry",
        "reference",
    },
    "status": {"current", "draft", "superseded", "retired", "reference"},
    "lifecycle": {"active-core", "incubating", "parked", "retired"},
}
RULES_RUN = [
    "frontmatter",
    "links",
    "claims",
    "lifecycle",
    "authority",
    "absolute-paths",
    "components",
]
EXCLUDED_DIR_NAMES = {".git", ".jj", "__pycache__"}
LIVE_SCAN_SKIP_PREFIXES = {
    PurePosixPath("evals/cases"),
}
COMPONENTS_PATH = PurePosixPath("docs/ops/COMPONENTS.yaml")
EXTERNAL_LINK_PREFIXES = ("http://", "https://", "mailto:", "app://")
ABSOLUTE_PATH_PATTERNS = [
    (re.compile(r"/Users/[^\s)`]+"), "unix-user-home"),
    (re.compile(r"/home/[^\s)`]+"), "unix-home"),
    (re.compile(r"[A-Za-z]:\\\\Users\\\\[^\s)`]+"), "windows-user-home"),
]
LINK_PATTERN = re.compile(r"!?\[[^\]]*\]\(([^)]+)\)")
DATE_PATTERN = re.compile(r"^\d{4}-\d{2}-\d{2}$")
HARD_CLAIM_PATTERNS = [
    (re.compile(r"\bruntime is integrated\b", re.IGNORECASE), "runtime_integration"),
    (re.compile(r"\busers can run\b", re.IGNORECASE), "cli_command"),
    (re.compile(r"\bcli command exists\b", re.IGNORECASE), "cli_command"),
    (re.compile(r"\bworks end-to-end\b", re.IGNORECASE), "generic_runtime"),
    (re.compile(r"\bservice provides\b", re.IGNORECASE), "generic_runtime"),
    (re.compile(r"\bsyncs with\b", re.IGNORECASE), "tracker_sync"),
    (re.compile(r"\bproduces proof\b", re.IGNORECASE), "proof_generation"),
    (re.compile(r"\bimplemented\b", re.IGNORECASE), "generic_runtime"),
    (re.compile(r"\bavailable\b", re.IGNORECASE), "generic_runtime"),
]
ALLOWED_CLAIM_MARKERS = [
    "must support",
    "should support",
    "is designed to",
    "target",
    "planned",
    "future",
    "not implemented yet",
    "docs-only",
]
AMBIGUOUS_CLAIM_PATTERNS = [
    re.compile(r"\bcan run tasks\b", re.IGNORECASE),
    re.compile(r"\bready for use\b", re.IGNORECASE),
    re.compile(r"\bsupports delivery\b", re.IGNORECASE),
]
COMPONENT_STATUS_ENUM = {
    "not_started",
    "planned",
    "docs_only",
    "prototype",
    "implemented",
    "parked",
    "retired",
}
REQUIRED_COMPONENT_IDS = [
    "project_spine",
    "product_concept",
    "contract_lifecycle",
    "task_shaping",
    "runtime_registry",
    "primary_runtime_adapter",
    "advisory_panel",
    "gate_verify",
    "proof_artifact",
    "web_surface",
    "cli_surface",
    "tracker_sync",
    "docs_governance",
]
REQUIRED_COMPONENT_FIELDS = [
    "name",
    "status",
    "truth_owner",
    "implementation_paths",
    "public_claim_allowed",
]


class CheckerConfigError(RuntimeError):
    pass


def run_root_scan(root: Path, mode: str = "report-only") -> dict[str, Any]:
    root = root.resolve()
    findings: list[dict[str, Any]] = []
    checked_files: list[str] = []

    components_file = root / COMPONENTS_PATH
    if components_file.exists():
        rel_path = components_file.relative_to(root).as_posix()
        checked_files.append(rel_path)
        findings.extend(validate_components_file(root, components_file, rel_path))

    for markdown_file in iter_markdown_files(root, include_fixture_cases=False):
        rel_path = markdown_file.relative_to(root).as_posix()
        checked_files.append(rel_path)
        findings.extend(scan_markdown_file(root, markdown_file, rel_path))

    findings = sort_findings(findings)
    return build_report(
        mode=mode,
        root=root.as_posix(),
        checked_files=checked_files,
        findings=findings,
        fixture_results=[],
    )


def run_changed_files_scan(root: Path, changed_files_path: Path) -> tuple[dict[str, Any], int]:
    root = root.resolve()
    changed_files_path = changed_files_path.resolve()
    findings: list[dict[str, Any]] = []
    checked_files: list[str] = []

    for rel_path in load_changed_files(changed_files_path):
        pure_path = PurePosixPath(rel_path)
        if not is_repo_relative_path(rel_path):
            raise CheckerConfigError("changed-files entries must use repo-relative paths")

        candidate = root / pure_path
        if not candidate.exists() or candidate.is_dir():
            continue

        if pure_path == COMPONENTS_PATH:
            checked_files.append(rel_path)
            findings.extend(validate_components_file(root, candidate, rel_path))
            continue

        if supports_changed_markdown_scan(pure_path):
            checked_files.append(rel_path)
            findings.extend(scan_markdown_file(root, candidate, rel_path))

    findings = sort_findings(findings)
    report = build_report(
        mode="changed-files",
        root=root.as_posix(),
        checked_files=checked_files,
        findings=findings,
        fixture_results=[],
    )
    exit_code = 1 if report["summary"]["hard_count"] > 0 else 0
    return report, exit_code


def run_fixture_self_test(fixtures_root: Path) -> tuple[dict[str, Any], int]:
    fixtures_root = fixtures_root.resolve()
    fixture_results: list[dict[str, Any]] = []
    all_findings: list[dict[str, Any]] = []
    checked_files: list[str] = []

    for expected_path in sorted(fixtures_root.rglob("expected.json")):
        case_dir = expected_path.parent
        category = case_dir.parent.name
        actual = scan_fixture_case(case_dir, category)
        expected = json.loads(expected_path.read_text(encoding="utf-8"))
        passed = actual == expected
        fixture_results.append(
            {
                "category": category,
                "case": case_dir.name,
                "passed": passed,
                "expected_path": expected_path.relative_to(fixtures_root).as_posix(),
                "actual": actual,
                "expected": expected,
            }
        )
        checked_files.extend(
            [f"{category}/{case_dir.name}/{path}" for path in actual["checked_files"]]
        )
        all_findings.extend(
            [
                {**finding, "path": f"{category}/{case_dir.name}/{finding['path']}"}
                for finding in actual["findings"]
            ]
        )

    report = build_report(
        mode="self-test",
        root=fixtures_root.as_posix(),
        checked_files=checked_files,
        findings=sort_findings(all_findings),
        fixture_results=fixture_results,
    )
    exit_code = 1 if any(not result["passed"] for result in fixture_results) else 0
    return report, exit_code


def scan_fixture_case(case_dir: Path, category: str) -> dict[str, Any]:
    if category == "components":
        return scan_components_fixture_case(case_dir)
    if category == "changed-files":
        return scan_changed_files_fixture_case(case_dir)

    markdown_files = sorted(path for path in case_dir.rglob("*.md") if path.name != "expected.json")
    status_data = load_status(case_dir / "status.json")
    findings: list[dict[str, Any]] = []
    docs: dict[str, dict[str, Any]] = {}

    for markdown_file in markdown_files:
        rel_path = markdown_file.relative_to(case_dir).as_posix()
        text = markdown_file.read_text(encoding="utf-8")
        parse_result = parse_frontmatter(text)
        docs[rel_path] = {
            "text": text,
            "frontmatter": parse_result.data,
            "parse_result": parse_result,
        }

        findings.extend(check_absolute_paths(text, rel_path, category))
        findings.extend(check_markdown_links(case_dir, markdown_file.parent, text, rel_path))

        requires_frontmatter = category in {"frontmatter", "lifecycle", "authority"}
        if parse_result.data is None:
            if requires_frontmatter:
                findings.append(
                    make_finding(
                        severity="hard",
                        check="frontmatter",
                        path=rel_path,
                        line=1,
                        message="Missing frontmatter block.",
                        rule="frontmatter.missing",
                        expected="document with frontmatter",
                        actual="document without frontmatter",
                    )
                )
            continue

        findings.extend(validate_frontmatter(parse_result.data, rel_path))
        if category in {"lifecycle", "frontmatter"}:
            findings.extend(check_lifecycle(parse_result.data, rel_path))
        if category in {"authority", "frontmatter"}:
            findings.extend(check_authority(parse_result.data, rel_path))
        if category == "claims":
            findings.extend(check_claims(text, rel_path, status_data))

    if category == "lifecycle":
        findings.extend(check_superseded_related_docs(docs))

    findings = sort_findings(findings)
    checked_files = [path.relative_to(case_dir).as_posix() for path in markdown_files]
    return build_case_scan_result(checked_files, findings)


def scan_components_fixture_case(case_dir: Path) -> dict[str, Any]:
    components_file = case_dir / "COMPONENTS.yaml"
    checked_files: list[str] = []
    findings: list[dict[str, Any]] = []

    if not components_file.exists():
        findings.append(
            make_finding(
                severity="hard",
                check="components",
                path="COMPONENTS.yaml",
                line=1,
                message="Missing COMPONENTS.yaml fixture input.",
                rule="components.fixture.missing-file",
                expected="COMPONENTS.yaml present",
                actual="missing",
            )
        )
    else:
        checked_files.append("COMPONENTS.yaml")
        findings.extend(validate_components_file(case_dir, components_file, "COMPONENTS.yaml"))

    return build_case_scan_result(checked_files, sort_findings(findings))


def scan_changed_files_fixture_case(case_dir: Path) -> dict[str, Any]:
    changed_files_path = case_dir / "changed-files.txt"
    if not changed_files_path.exists():
        findings = [
            make_finding(
                severity="hard",
                check="changed-files",
                path="changed-files.txt",
                line=1,
                message="Missing changed-files.txt fixture input.",
                rule="changed-files.fixture.missing-file",
                expected="changed-files.txt present",
                actual="missing",
            )
        ]
        return build_case_scan_result([], findings)

    report, _ = run_changed_files_scan(case_dir, changed_files_path)
    return build_case_scan_result(
        report["checked_files"],
        report["findings"],
    )


def build_case_scan_result(checked_files: list[str], findings: list[dict[str, Any]]) -> dict[str, Any]:
    return {
        "summary": {
            "files_checked": len(checked_files),
            "hard_count": sum(1 for finding in findings if finding["severity"] == "hard"),
            "warning_count": sum(1 for finding in findings if finding["severity"] == "warning"),
            "info_count": sum(1 for finding in findings if finding["severity"] == "info"),
        },
        "checked_files": checked_files,
        "findings": findings,
    }


def validate_frontmatter(frontmatter: dict[str, Any], rel_path: str) -> list[dict[str, Any]]:
    findings: list[dict[str, Any]] = []
    for field in REQUIRED_FRONTMATTER_FIELDS:
        if field not in frontmatter:
            findings.append(
                make_finding(
                    severity="hard",
                    check="frontmatter",
                    path=rel_path,
                    line=1,
                    message=f"Missing required frontmatter field: {field}.",
                    rule="frontmatter.required-field",
                    expected=field,
                    actual=None,
                )
            )

    for field, allowed_values in ENUMS.items():
        if field in frontmatter and frontmatter[field] not in allowed_values:
            findings.append(
                make_finding(
                    severity="hard",
                    check="frontmatter",
                    path=rel_path,
                    line=1,
                    message=f"Invalid value for frontmatter field `{field}`.",
                    rule="frontmatter.enum",
                    expected=sorted(allowed_values),
                    actual=frontmatter[field],
                )
            )

    if "review_after" in frontmatter and not valid_review_after(frontmatter["review_after"]):
        findings.append(
            make_finding(
                severity="hard",
                check="frontmatter",
                path=rel_path,
                line=1,
                message="Frontmatter field `review_after` must use YYYY-MM-DD.",
                rule="frontmatter.review-after.format",
                expected="YYYY-MM-DD",
                actual=frontmatter["review_after"],
            )
        )

    findings.extend(validate_list_field(frontmatter, rel_path, "truth_surfaces", path_values=False))
    findings.extend(validate_list_field(frontmatter, rel_path, "supersedes", path_values=True))
    findings.extend(validate_list_field(frontmatter, rel_path, "related_docs", path_values=True))

    if "superseded_by" in frontmatter and frontmatter["superseded_by"] is not None:
        if not is_repo_relative_path(frontmatter["superseded_by"]):
            findings.append(
                make_finding(
                    severity="hard",
                    check="frontmatter",
                    path=rel_path,
                    line=1,
                    message="Frontmatter field `superseded_by` must use a repo-relative path or null.",
                    rule="frontmatter.superseded-by.path",
                    expected="repo-relative path or null",
                    actual=frontmatter["superseded_by"],
                )
            )

    return findings


def validate_list_field(frontmatter: dict[str, Any], rel_path: str, field_name: str, path_values: bool) -> list[dict[str, Any]]:
    findings: list[dict[str, Any]] = []
    if field_name not in frontmatter:
        return findings
    value = frontmatter[field_name]
    if not isinstance(value, list):
        findings.append(
            make_finding(
                severity="hard",
                check="frontmatter",
                path=rel_path,
                line=1,
                message=f"Frontmatter field `{field_name}` must be a list.",
                rule=f"frontmatter.{field_name}.type",
                expected="list",
                actual=type(value).__name__,
            )
        )
        return findings

    for item in value:
        if not isinstance(item, str):
            findings.append(
                make_finding(
                    severity="hard",
                    check="frontmatter",
                    path=rel_path,
                    line=1,
                    message=f"Frontmatter field `{field_name}` must contain strings.",
                    rule=f"frontmatter.{field_name}.item-type",
                    expected="string items",
                    actual=type(item).__name__,
                )
            )
            continue
        if path_values and not is_repo_relative_path(item):
            findings.append(
                make_finding(
                    severity="hard",
                    check="frontmatter",
                    path=rel_path,
                    line=1,
                    message=f"Frontmatter field `{field_name}` must use repo-relative paths.",
                    rule=f"frontmatter.{field_name}.path",
                    expected="repo-relative path",
                    actual=item,
                )
            )
    return findings


def check_lifecycle(frontmatter: dict[str, Any], rel_path: str) -> list[dict[str, Any]]:
    findings: list[dict[str, Any]] = []
    review_after = frontmatter.get("review_after")
    if isinstance(review_after, str) and valid_review_after(review_after):
        if review_after < date.today().isoformat():
            findings.append(
                make_finding(
                    severity="warning",
                    check="lifecycle",
                    path=rel_path,
                    line=1,
                    message="review_after date is in the past.",
                    rule="lifecycle.review-after-expired",
                    expected=f">= {date.today().isoformat()}",
                    actual=review_after,
                )
            )
    return findings


def check_superseded_related_docs(docs: dict[str, dict[str, Any]]) -> list[dict[str, Any]]:
    findings: list[dict[str, Any]] = []
    for rel_path, payload in docs.items():
        frontmatter = payload["frontmatter"]
        if not frontmatter or frontmatter.get("status") != "current":
            continue
        for related_doc in frontmatter.get("related_docs", []):
            if not isinstance(related_doc, str):
                continue
            target_path = (PurePosixPath(rel_path).parent / PurePosixPath(related_doc)).as_posix()
            target = docs.get(target_path)
            if not target or not target.get("frontmatter"):
                continue
            if target["frontmatter"].get("status") == "superseded":
                findings.append(
                    make_finding(
                        severity="hard",
                        check="lifecycle",
                        path=rel_path,
                        line=1,
                        message="Current document references a superseded related document.",
                        rule="lifecycle.related-doc-superseded",
                        expected="related current or draft document",
                        actual=target_path,
                    )
                )
    return findings


def check_authority(frontmatter: dict[str, Any], rel_path: str) -> list[dict[str, Any]]:
    findings: list[dict[str, Any]] = []
    authority = frontmatter.get("authority")
    truth_surfaces = frontmatter.get("truth_surfaces", [])
    if not isinstance(truth_surfaces, list):
        return findings

    if Path(rel_path).name == "INDEX.md" and authority == "canonical":
        findings.append(
            make_finding(
                severity="hard",
                check="authority",
                path=rel_path,
                line=1,
                message="INDEX.md must remain a human read-order view, not canonical authority.",
                rule="authority.index-human-view",
                expected="derived or operational authority",
                actual=authority,
            )
        )

    if authority == "public_entry" and any(looks_like_canonical_surface(item) for item in truth_surfaces):
        findings.append(
            make_finding(
                severity="hard",
                check="authority",
                path=rel_path,
                line=1,
                message="public_entry documents must not own canonical truth surfaces.",
                rule="authority.public-entry-truth-surface",
                expected="non-canonical truth surfaces",
                actual=truth_surfaces,
            )
        )

    if authority == "advisory" and any(looks_like_canonical_surface(item) for item in truth_surfaces):
        findings.append(
            make_finding(
                severity="hard",
                check="authority",
                path=rel_path,
                line=1,
                message="advisory documents must not override canonical truth surfaces.",
                rule="authority.advisory-cannot-own-canon",
                expected="advisory-only truth surfaces",
                actual=truth_surfaces,
            )
        )

    return findings


def check_claims(text: str, rel_path: str, status_data: dict[str, Any]) -> list[dict[str, Any]]:
    if not status_data:
        return []

    findings: list[dict[str, Any]] = []
    for line_number, line in enumerate(text.splitlines(), start=1):
        lowered = line.lower()
        if any(marker in lowered for marker in ALLOWED_CLAIM_MARKERS):
            continue

        for pattern, status_key in HARD_CLAIM_PATTERNS:
            match = pattern.search(line)
            if not match:
                continue
            if not bool(status_data.get(status_key, False)):
                findings.append(
                    make_finding(
                        severity="hard",
                        check="claims",
                        path=rel_path,
                        line=line_number,
                        message="Implementation claim conflicts with provided status.",
                        rule="claims.false-implementation",
                        expected={status_key: True},
                        actual={status_key: bool(status_data.get(status_key, False)), "text": match.group(0)},
                    )
                )
        for pattern in AMBIGUOUS_CLAIM_PATTERNS:
            match = pattern.search(line)
            if match:
                findings.append(
                    make_finding(
                        severity="warning",
                        check="claims",
                        path=rel_path,
                        line=line_number,
                        message="Ambiguous implementation language should stay warning-only.",
                        rule="claims.ambiguous-language",
                        expected="explicit planned or implemented wording",
                        actual=match.group(0),
                    )
                )
    return findings


def check_markdown_links(root: Path, base_dir: Path, text: str, rel_path: str) -> list[dict[str, Any]]:
    findings: list[dict[str, Any]] = []
    for line_number, line in enumerate(text.splitlines(), start=1):
        for match in LINK_PATTERN.finditer(line):
            target = normalize_markdown_target(match.group(1))
            if not target or target.startswith("#"):
                continue
            if target.startswith(EXTERNAL_LINK_PREFIXES):
                continue
            candidate = target.split("#", 1)[0]
            resolved = (base_dir / candidate).resolve()
            try:
                resolved.relative_to(root.resolve())
            except ValueError:
                findings.append(
                    make_finding(
                        severity="hard",
                        check="links",
                        path=rel_path,
                        line=line_number,
                        message="Repo-relative link resolves outside the scan root.",
                        rule="links.outside-root",
                        expected="link inside repository root",
                        actual=target,
                    )
                )
                continue
            if not resolved.exists():
                findings.append(
                    make_finding(
                        severity="hard",
                        check="links",
                        path=rel_path,
                        line=line_number,
                        message="Broken repo-relative link.",
                        rule="links.broken-relative",
                        expected="existing repo-relative target",
                        actual=target,
                    )
                )
    return findings


def check_absolute_paths(text: str, rel_path: str, category: str) -> list[dict[str, Any]]:
    findings: list[dict[str, Any]] = []
    for line_number, line in enumerate(text.splitlines(), start=1):
        for pattern, label in ABSOLUTE_PATH_PATTERNS:
            for match in pattern.finditer(line):
                findings.append(
                    make_finding(
                        severity="hard",
                        check="absolute-paths",
                        path=rel_path,
                        line=line_number,
                        message="Local absolute path is not allowed.",
                        rule="absolute-paths.local",
                        expected="repo-relative path",
                        actual=match.group(0),
                        context={"pattern": label, "category": category},
                    )
                )
    return findings


def scan_markdown_file(root: Path, markdown_file: Path, rel_path: str) -> list[dict[str, Any]]:
    text = markdown_file.read_text(encoding="utf-8")
    parse_result = parse_frontmatter(text)
    findings: list[dict[str, Any]] = []

    findings.extend(check_absolute_paths(text, rel_path, "absolute-paths"))
    findings.extend(check_markdown_links(root, markdown_file.parent, text, rel_path))

    if parse_result.data is not None:
        findings.extend(validate_frontmatter(parse_result.data, rel_path))
        findings.extend(check_lifecycle(parse_result.data, rel_path))
        findings.extend(check_authority(parse_result.data, rel_path))

    return findings


def validate_components_file(root: Path, components_file: Path, rel_path: str) -> list[dict[str, Any]]:
    text = components_file.read_text(encoding="utf-8")
    parse_result = parse_components_document(text)
    findings: list[dict[str, Any]] = []

    for error in parse_result.errors:
        findings.append(
            make_finding(
                severity="hard",
                check="components",
                path=rel_path,
                line=error.line,
                message=error.message,
                rule="components.parse",
                expected="supported Goalrail components YAML subset",
                actual=text.splitlines()[error.line - 1] if error.line - 1 < len(text.splitlines()) else None,
            )
        )

    data = parse_result.data if isinstance(parse_result.data, dict) else {}
    findings.extend(validate_components_top_level(data, parse_result, rel_path))
    findings.extend(validate_components_entries(root, data, parse_result, rel_path))
    return findings


def validate_components_top_level(
    data: dict[str, Any],
    parse_result: ComponentsParseResult,
    rel_path: str,
) -> list[dict[str, Any]]:
    findings: list[dict[str, Any]] = []

    required_fields = ["schema_version", "updated_at", "authority", "status_anchor", "components"]
    for field in required_fields:
        if field not in data:
            findings.append(
                make_finding(
                    severity="hard",
                    check="components",
                    path=rel_path,
                    line=1,
                    message=f"Missing required COMPONENTS field: {field}.",
                    rule="components.required-field",
                    expected=field,
                    actual=None,
                )
            )

    if "schema_version" in data and data["schema_version"] != "goalrail.components.v0":
        findings.append(
            make_finding(
                severity="hard",
                check="components",
                path=rel_path,
                line=line_for_path(parse_result, ("schema_version",), 1),
                message="COMPONENTS schema_version must be `goalrail.components.v0`.",
                rule="components.schema-version",
                expected="goalrail.components.v0",
                actual=data["schema_version"],
            )
        )

    if "updated_at" in data and not valid_review_after(data["updated_at"]):
        findings.append(
            make_finding(
                severity="hard",
                check="components",
                path=rel_path,
                line=line_for_path(parse_result, ("updated_at",), 1),
                message="COMPONENTS updated_at must use YYYY-MM-DD.",
                rule="components.updated-at.format",
                expected="YYYY-MM-DD",
                actual=data["updated_at"],
            )
        )

    if "authority" in data and data["authority"] != "operational":
        findings.append(
            make_finding(
                severity="hard",
                check="components",
                path=rel_path,
                line=line_for_path(parse_result, ("authority",), 1),
                message="COMPONENTS authority must stay `operational`.",
                rule="components.authority",
                expected="operational",
                actual=data["authority"],
            )
        )

    if "status_anchor" in data and data["status_anchor"] is not True:
        findings.append(
            make_finding(
                severity="hard",
                check="components",
                path=rel_path,
                line=line_for_path(parse_result, ("status_anchor",), 1),
                message="COMPONENTS status_anchor must be true.",
                rule="components.status-anchor",
                expected=True,
                actual=data["status_anchor"],
            )
        )

    if "components" in data and not isinstance(data["components"], dict):
        findings.append(
            make_finding(
                severity="hard",
                check="components",
                path=rel_path,
                line=line_for_path(parse_result, ("components",), 1),
                message="COMPONENTS components field must be a mapping.",
                rule="components.components.type",
                expected="mapping",
                actual=type(data["components"]).__name__,
            )
        )

    return findings


def validate_components_entries(
    root: Path,
    data: dict[str, Any],
    parse_result: ComponentsParseResult,
    rel_path: str,
) -> list[dict[str, Any]]:
    findings: list[dict[str, Any]] = []
    components = data.get("components")
    if not isinstance(components, dict):
        return findings

    for component_id in REQUIRED_COMPONENT_IDS:
        if component_id not in components:
            findings.append(
                make_finding(
                    severity="hard",
                    check="components",
                    path=rel_path,
                    line=line_for_path(parse_result, ("components",), 1),
                    message=f"Missing required component entry: {component_id}.",
                    rule="components.required-component",
                    expected=component_id,
                    actual=None,
                )
            )

    for component_id, component in components.items():
        line_number = line_for_path(parse_result, ("components", component_id), line_for_path(parse_result, ("components",), 1))
        if not isinstance(component, dict):
            findings.append(
                make_finding(
                    severity="hard",
                    check="components",
                    path=rel_path,
                    line=line_number,
                    message=f"Component `{component_id}` must be a mapping.",
                    rule="components.component.type",
                    expected="mapping",
                    actual=type(component).__name__,
                )
            )
            continue

        for field in REQUIRED_COMPONENT_FIELDS:
            if field not in component:
                findings.append(
                    make_finding(
                        severity="hard",
                        check="components",
                        path=rel_path,
                        line=line_number,
                        message=f"Component `{component_id}` is missing required field `{field}`.",
                        rule="components.component.required-field",
                        expected=field,
                        actual=None,
                    )
                )

        name = component.get("name")
        if name is not None and not isinstance(name, str):
            findings.append(
                make_finding(
                    severity="hard",
                    check="components",
                    path=rel_path,
                    line=line_for_path(parse_result, ("components", component_id, "name"), line_number),
                    message=f"Component `{component_id}` field `name` must be a string.",
                    rule="components.component.name.type",
                    expected="string",
                    actual=type(name).__name__,
                )
            )

        status = component.get("status")
        if status is not None and status not in COMPONENT_STATUS_ENUM:
            findings.append(
                make_finding(
                    severity="hard",
                    check="components",
                    path=rel_path,
                    line=line_for_path(parse_result, ("components", component_id, "status"), line_number),
                    message=f"Component `{component_id}` has invalid status.",
                    rule="components.component.status",
                    expected=sorted(COMPONENT_STATUS_ENUM),
                    actual=status,
                )
            )

        truth_owner = component.get("truth_owner")
        truth_owner_line = line_for_path(parse_result, ("components", component_id, "truth_owner"), line_number)
        if truth_owner is not None:
            if not is_repo_relative_path(truth_owner):
                findings.append(
                    make_finding(
                        severity="hard",
                        check="components",
                        path=rel_path,
                        line=truth_owner_line,
                        message=f"Component `{component_id}` truth_owner must use a repo-relative path.",
                        rule="components.component.truth-owner.path",
                        expected="repo-relative path",
                        actual=truth_owner,
                    )
                )
            elif not (root / truth_owner).exists():
                findings.append(
                    make_finding(
                        severity="hard",
                        check="components",
                        path=rel_path,
                        line=truth_owner_line,
                        message=f"Component `{component_id}` truth_owner path does not exist.",
                        rule="components.component.truth-owner.exists",
                        expected="existing repo-relative path",
                        actual=truth_owner,
                    )
                )

        implementation_paths = component.get("implementation_paths")
        implementation_paths_line = line_for_path(
            parse_result,
            ("components", component_id, "implementation_paths"),
            line_number,
        )
        if implementation_paths is not None:
            if not isinstance(implementation_paths, list):
                findings.append(
                    make_finding(
                        severity="hard",
                        check="components",
                        path=rel_path,
                        line=implementation_paths_line,
                        message=f"Component `{component_id}` field `implementation_paths` must be a list.",
                        rule="components.component.implementation-paths.type",
                        expected="list",
                        actual=type(implementation_paths).__name__,
                    )
                )
            else:
                for path_value in implementation_paths:
                    if not isinstance(path_value, str):
                        findings.append(
                            make_finding(
                                severity="hard",
                                check="components",
                                path=rel_path,
                                line=implementation_paths_line,
                                message=f"Component `{component_id}` implementation_paths must contain strings.",
                                rule="components.component.implementation-paths.item-type",
                                expected="string items",
                                actual=type(path_value).__name__,
                            )
                        )
                        continue
                    if not is_repo_relative_path(path_value):
                        findings.append(
                            make_finding(
                                severity="hard",
                                check="components",
                                path=rel_path,
                                line=implementation_paths_line,
                                message=f"Component `{component_id}` implementation path must be repo-relative.",
                                rule="components.component.implementation-paths.path",
                                expected="repo-relative path",
                                actual=path_value,
                            )
                        )
                        continue
                    if not (root / path_value).exists():
                        findings.append(
                            make_finding(
                                severity="hard",
                                check="components",
                                path=rel_path,
                                line=implementation_paths_line,
                                message=f"Component `{component_id}` implementation path does not exist.",
                                rule="components.component.implementation-paths.exists",
                                expected="existing repo-relative path",
                                actual=path_value,
                            )
                        )

        public_claim_allowed = component.get("public_claim_allowed")
        if public_claim_allowed is not None and not isinstance(public_claim_allowed, bool):
            findings.append(
                make_finding(
                    severity="hard",
                    check="components",
                    path=rel_path,
                    line=line_for_path(parse_result, ("components", component_id, "public_claim_allowed"), line_number),
                    message=f"Component `{component_id}` field `public_claim_allowed` must be a boolean.",
                    rule="components.component.public-claim-allowed.type",
                    expected="boolean",
                    actual=type(public_claim_allowed).__name__,
                )
            )

        notes = component.get("notes")
        if notes is not None and not isinstance(notes, str):
            findings.append(
                make_finding(
                    severity="hard",
                    check="components",
                    path=rel_path,
                    line=line_for_path(parse_result, ("components", component_id, "notes"), line_number),
                    message=f"Component `{component_id}` field `notes` must be a string when present.",
                    rule="components.component.notes.type",
                    expected="string",
                    actual=type(notes).__name__,
                )
            )

    return findings


def line_for_path(parse_result: ComponentsParseResult, key_path: tuple[str, ...], fallback: int) -> int:
    return parse_result.line_map.get(key_path, fallback)


def iter_markdown_files(root: Path, include_fixture_cases: bool) -> list[Path]:
    paths: list[Path] = []
    for path in root.rglob("*.md"):
        rel_path = PurePosixPath(path.relative_to(root).as_posix())
        if any(part in EXCLUDED_DIR_NAMES for part in rel_path.parts):
            continue
        if not include_fixture_cases and any(rel_path == prefix or rel_path.is_relative_to(prefix) for prefix in LIVE_SCAN_SKIP_PREFIXES):
            continue
        paths.append(path)
    return sorted(paths)


def load_changed_files(path: Path) -> list[str]:
    if not path.exists():
        raise CheckerConfigError(f"changed-files list does not exist: {path}")
    entries = [line.strip() for line in path.read_text(encoding="utf-8").splitlines()]
    return [entry for entry in entries if entry and not entry.startswith("#")]


def supports_changed_markdown_scan(path: PurePosixPath) -> bool:
    return bool(path.parts) and path.parts[0] == "docs" and path.suffix.lower() == ".md"


def build_report(
    *,
    mode: str,
    root: str,
    checked_files: list[str],
    findings: list[dict[str, Any]],
    fixture_results: list[dict[str, Any]],
) -> dict[str, Any]:
    summary = {
        "files_checked": len(sorted(set(checked_files))),
        "hard_count": sum(1 for finding in findings if finding["severity"] == "hard"),
        "warning_count": sum(1 for finding in findings if finding["severity"] == "warning"),
        "info_count": sum(1 for finding in findings if finding["severity"] == "info"),
        "fixture_count": len(fixture_results),
        "fixture_fail_count": sum(1 for result in fixture_results if not result["passed"]),
    }
    return {
        "schema_version": SCHEMA_VERSION,
        "generated_at": datetime.now(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
        "mode": mode,
        "root": root,
        "summary": summary,
        "findings": findings,
        "checked_files": sorted(set(checked_files)),
        "fixture_results": fixture_results,
    }


def render_markdown_report(report: dict[str, Any]) -> str:
    lines = [
        "# Goalrail Docs Check Report",
        "",
        f"- Mode: `{report['mode']}`",
        f"- Root: `{report['root']}`",
        f"- Generated at: `{report['generated_at']}`",
        "",
        "## Summary",
        "",
        f"- Files checked: {report['summary']['files_checked']}",
        f"- Hard findings: {report['summary']['hard_count']}",
        f"- Warning findings: {report['summary']['warning_count']}",
        f"- Info findings: {report['summary']['info_count']}",
        f"- Fixture cases: {report['summary']['fixture_count']}",
        f"- Fixture failures: {report['summary']['fixture_fail_count']}",
        "",
        "## Rules run",
        "",
    ]
    lines.extend([f"- `{rule}`" for rule in RULES_RUN])

    lines.extend(["", "## Findings", ""])
    if not report["findings"]:
        lines.append("No findings.")
    else:
        lines.append("| Severity | Rule | Path | Line | Message |")
        lines.append("| --- | --- | --- | ---: | --- |")
        for finding in report["findings"]:
            lines.append(
                f"| {finding['severity']} | {finding['rule']} | `{finding['path']}` | {finding['line']} | {finding['message']} |"
            )

    lines.extend(["", "## Fixture results", ""])
    if not report["fixture_results"]:
        lines.append("No fixture results in this run.")
    else:
        lines.append("| Category | Case | Passed | Expected |")
        lines.append("| --- | --- | --- | --- |")
        for result in report["fixture_results"]:
            status = "yes" if result["passed"] else "no"
            lines.append(
                f"| {result['category']} | {result['case']} | {status} | `{result['expected_path']}` |"
            )

    return "\n".join(lines) + "\n"


def write_json_report(path: Path, report: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def load_status(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    return json.loads(path.read_text(encoding="utf-8"))


def valid_review_after(value: Any) -> bool:
    return isinstance(value, str) and bool(DATE_PATTERN.match(value))


def looks_like_canonical_surface(value: str) -> bool:
    lowered = value.lower()
    return any(token in lowered for token in ("product", "architecture", "mvp", "implementation-status", "canonical"))


def is_repo_relative_path(value: Any) -> bool:
    if not isinstance(value, str) or not value:
        return False
    if value.startswith(EXTERNAL_LINK_PREFIXES):
        return False
    if re.match(r"^[A-Za-z]:\\", value):
        return False
    if value.startswith("/") or value.startswith("~"):
        return False
    pure_path = PurePosixPath(value)
    if pure_path.is_absolute():
        return False
    if any(part == ".." for part in pure_path.parts):
        return False
    return True


def normalize_markdown_target(raw_target: str) -> str:
    target = raw_target.strip()
    if target.startswith("<") and target.endswith(">"):
        return target[1:-1]
    if " " in target:
        return target.split(" ", 1)[0]
    return target


def sort_findings(findings: list[dict[str, Any]]) -> list[dict[str, Any]]:
    return sorted(findings, key=lambda item: (item["path"], item["line"], item["rule"], item["message"]))


def make_finding(
    *,
    severity: str,
    check: str,
    path: str,
    line: int,
    message: str,
    rule: str,
    expected: Any,
    actual: Any,
    context: dict[str, Any] | None = None,
) -> dict[str, Any]:
    context = context or {}
    return {
        "id": f"{path}:{line}:{rule}",
        "severity": severity,
        "check": check,
        "path": path,
        "line": line,
        "message": message,
        "rule": rule,
        "expected": expected,
        "actual": actual,
        "context": context,
    }
