#!/usr/bin/env python3
from __future__ import annotations

import argparse
import sys
from pathlib import Path

from lib.checker import (
    CheckerConfigError,
    render_markdown_report,
    run_fixture_self_test,
    run_root_scan,
    write_json_report,
)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Goalrail report-only docs governance checker")
    parser.add_argument("--root", help="Repository root to scan in report-only mode")
    parser.add_argument("--fixtures", help="Fixture root to self-test")
    parser.add_argument("--self-test", action="store_true", help="Run fixture self-test mode")
    parser.add_argument("--mode", default="report-only", choices=["report-only"], help="Checker mode")
    parser.add_argument("--report-json", help="Path to write the JSON report")
    parser.add_argument("--report-md", help="Path to write the Markdown report")
    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()

    try:
        if args.self_test:
            if not args.fixtures:
                raise CheckerConfigError("--self-test requires --fixtures")
            report, exit_code = run_fixture_self_test(Path(args.fixtures))
        else:
            if not args.root:
                raise CheckerConfigError("live scan requires --root")
            report = run_root_scan(Path(args.root), mode=args.mode)
            exit_code = 0

        if args.report_json:
            write_json_report(Path(args.report_json), report)
        if args.report_md:
            Path(args.report_md).write_text(render_markdown_report(report), encoding="utf-8")

        print(
            f"mode={report['mode']} files={report['summary']['files_checked']} "
            f"hard={report['summary']['hard_count']} warning={report['summary']['warning_count']} "
            f"info={report['summary']['info_count']} fixture_fail={report['summary']['fixture_fail_count']}"
        )
        return exit_code
    except CheckerConfigError as exc:
        print(f"config error: {exc}", file=sys.stderr)
        return 2
    except Exception as exc:  # pragma: no cover - defensive top-level guard
        print(f"internal error: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
