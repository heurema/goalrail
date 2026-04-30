# PR Intake Gate

Deterministic fixture tests for `scripts/pr_intake_gate.py`.

Run:

```bash
python3 tools/pr-intake-gate/test_pr_intake_gate.py
```

The workflow itself runs from trusted base-branch code through `pull_request_target` and reads PR metadata through the GitHub API. It must never checkout, import, install, or execute PR head code.
