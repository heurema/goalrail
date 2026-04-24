# Work

Repo-tracked planning and execution memory for Goalrail.

Canonical work artifacts live under:

- `.goalrail/work/goals/` — bounded goals and slices
- `.goalrail/work/reports/` — completion notes, handoffs, and outcome reports
- `.goalrail/work/demo/` — bounded demo-planning packs, scenario libraries, and replay-readiness notes for demo assets that may later live in a separate executable repo
- `.goalrail/work/_templates/` — starter shapes for new artifacts

Work artifacts support delivery, but they do not outrank `docs/product/` or `docs/ops/` as product truth.

Rule:
- `.goalrail/work/demo/` is planning-only unless a separate bounded implementation slice explicitly says otherwise
- executable demo runtime, app code, and dependency-bearing sandboxes do not belong under `.goalrail/work/`
