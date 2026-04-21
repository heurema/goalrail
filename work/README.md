# Work

Repo-tracked planning and execution memory for Goalrail.

Canonical work artifacts live under:

- `work/goals/` — bounded goals and slices
- `work/reports/` — completion notes, handoffs, and outcome reports
- `work/demo/` — bounded demo-planning packs, scenario libraries, and replay-readiness notes for demo assets that may later live in a separate executable repo
- `work/_templates/` — starter shapes for new artifacts

Work artifacts support delivery, but they do not outrank `docs/product/` or `docs/ops/` as product truth.

Rule:
- `work/demo/` is planning-only unless a separate bounded implementation slice explicitly says otherwise
- executable demo runtime, app code, and dependency-bearing sandboxes do not belong under `work/`
