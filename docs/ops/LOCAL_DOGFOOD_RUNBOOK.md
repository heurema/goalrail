---
id: goalrail_local_dogfood_runbook
title: Local Goalrail Dogfooding Runbook
kind: ops_status
authority: operational
status: current
owner: ops
truth_surfaces:
  - local_dogfood_flow
  - developer_experience_observations
  - dogfood_traceability
lifecycle: active-core
review_after: 2026-08-18
supersedes: []
superseded_by: null
related_docs:
  - docs/INDEX.md
  - docs/ops/STATUS.md
  - docs/ops/NEXT.md
  - docs/ops/COMPONENTS.yaml
  - docs/ops/INIT_STABILIZATION_CHECKPOINT.md
  - docs/ops/SNAPSHOT_SCAN_SHARED_SHAPE.md
  - docs/adr/ADR-0023-user-bootstrap-auth-and-cli-login-boundary.md
  - docs/adr/ADR-0024-minimal-planning-worker-loop-boundary.md
  - docs/adr/ADR-0025-repository-baseline-profile-lifecycle.md
---
# Local Goalrail Dogfooding Runbook

## Purpose

Use current Goalrail locally to manage Goalrail work through the
contract-first flow. This runbook is for dogfooding the implemented local
server, CLI, planning worker, and repository marker path while keeping product
claims bounded to current repo truth.

This is an operational dogfood path, not product canon. Product and
architecture truth still start from `docs/product/*`, `docs/ops/STATUS.md`,
and `docs/ops/COMPONENTS.yaml`.

## Boundaries

This local flow can prove that a developer can run a self-hosted Goalrail
control plane, bootstrap an owner, authenticate the CLI, initialize a local Git
worktree, create a Goal, answer clarification, draft/update/submit/approve a
Contract, create a WorkItemPlan, run the minimal planning worker once, accept a
proposal, and hand off to a draft PR.

It does not prove:
- gate decisions;
- proof generation;
- real project test execution;
- provider OAuth;
- provider clients or stored repository credentials;
- runtime adapters;
- autonomous delivery;
- tracker sync, analytics, CRM, or broad backend platform behavior;
- safe arbitrary shell or project command execution;
- WorkItem assignment, claiming, or completion.

Treat dogfood findings as observations and candidate follow-up slices. Do not
rewrite them into product claims unless the implemented surface and component
map already support the claim.

## Current Implemented Local Path

The current local path is:

1. Start local Postgres or point the server at an existing local database.
2. Apply server migrations.
3. Bootstrap the first self-hosted owner.
4. Start the local Goalrail server.
5. Authenticate the CLI.
6. Run `goalrail init`.
7. Run local Project Scan and status.
8. Create a Goal with `goalrail work start`.
9. Reconcile readiness with `goalrail work continue`.
10. Answer clarification with `goalrail work answer`.
11. Create the Contract draft with `goalrail contract draft`.
12. Update proposed draft fields with `goalrail contract update`.
13. Submit the Contract with `goalrail contract submit`.
14. Review the submitted Contract through read-only server state.
15. Approve the Contract with `goalrail contract approve`.
16. Create a WorkItemPlan with `goalrail work plan`.
17. Run `goalrail-worker --once`.
18. Inspect plan/proposal status with `goalrail work plan status`.
19. Accept the proposal with `goalrail work proposal accept`.
20. Create a draft PR manually from the accepted WorkItem.

Do not treat checkout preparation, execution preparation, runner receipts,
gate, proof, or WorkItem completion as part of this docs-only dogfood handoff
unless a later approved contract explicitly includes them.

## What Not To Claim

Do not claim that this flow provides gate, proof, real test execution,
provider OAuth, runtime adapters, autonomous delivery, full execution safety,
or verified code change. Current runner-related slices are bounded receipt and
command-plan prototypes; they are not broad runtime execution.

Do not claim that the planning proposal is high quality merely because it can
be accepted. The first DOGFOOD-001 proposal was accepted to continue testing
the implemented flow through `WorkItem(planned)`, while its generic shape
remains UX/product friction.

## Setup Notes

Use placeholders in durable docs and PR bodies. Do not commit local machine
paths, local passwords, auth files, token material, temporary passwords, or
`.goalrail/project.yml` unless marker policy changes in a later approved slice.

Example local-only environment:

```bash
export GOALRAIL_DOGFOOD_XDG=/tmp/goalrail-dogfood-xdg
export XDG_CONFIG_HOME="$GOALRAIL_DOGFOOD_XDG"
export HOME="$GOALRAIL_DOGFOOD_XDG"

export GOALRAIL_DATABASE_HOST=127.0.0.1
export GOALRAIL_DATABASE_PORT=55432
export GOALRAIL_DATABASE_NAME=goalrail
export GOALRAIL_DATABASE_USER=goalrail
export GOALRAIL_DATABASE_PASSWORD='<local-db-password>'
export GOALRAIL_DATABASE_SSLMODE=disable
export GOALRAIL_AUTH_JWT_SECRET='<local-jwt-secret-at-least-32-characters>'
```

Example disposable Postgres container:

```bash
docker run --name goalrail-dogfood-postgres \
  -e POSTGRES_DB=goalrail \
  -e POSTGRES_USER=goalrail \
  -e POSTGRES_PASSWORD='<local-db-password>' \
  -p 55432:5432 \
  -d postgres:16-alpine
```

Apply migrations from the server module:

```bash
cd <repo-root>/apps/server
go run ./cmd/goalrail-server migrate up
```

Bootstrap the first owner:

```bash
cd <repo-root>/apps/server
go run ./cmd/goalrail-server bootstrap owner \
  --email owner@example.com \
  --display-name "Owner User" \
  --organization-slug local-goalrail \
  --organization-name "Local Goalrail" \
  --public-base-url http://localhost:8080
```

The bootstrap command may print a one-time temporary password or
`temporary_password_already_exists=true`. Use the temporary password locally
only. Do not commit it or paste it into reports.

Start the server in a local shell:

```bash
cd <repo-root>/apps/server
go run ./cmd/goalrail-server
```

Then smoke check:

```bash
curl -fsS http://localhost:8080/livez
curl -fsS http://localhost:8080/readyz
curl -fsS http://localhost:8080/version
```

Authenticate the CLI through the implemented browser-loopback flow when an
interactive browser is available:

```bash
cd <repo-root>/apps/cli
HOME=/tmp/goalrail-dogfood-xdg \
XDG_CONFIG_HOME=/tmp/goalrail-dogfood-xdg \
go run ./cmd/goalrail login http://localhost:8080 --no-browser
```

If the browser-loopback path cannot complete in an agent or noninteractive
environment, record that as UX friction. Use the documented auth API fallback
only to continue local dogfood setup, keep responses in local temp files, and
redact all token material from any report.

Initialize and scan the repository:

```bash
cd <repo-root>/apps/cli
HOME=/tmp/goalrail-dogfood-xdg \
XDG_CONFIG_HOME=/tmp/goalrail-dogfood-xdg \
go run ./cmd/goalrail init --format json

HOME=/tmp/goalrail-dogfood-xdg \
XDG_CONFIG_HOME=/tmp/goalrail-dogfood-xdg \
go run ./cmd/goalrail project scan --format json

HOME=/tmp/goalrail-dogfood-xdg \
XDG_CONFIG_HOME=/tmp/goalrail-dogfood-xdg \
go run ./cmd/goalrail project status --format json
```

## Contract-First Dogfood Commands

Create a Goal from a local body file:

```bash
cd <repo-root>/apps/cli
HOME=/tmp/goalrail-dogfood-xdg \
XDG_CONFIG_HOME=/tmp/goalrail-dogfood-xdg \
go run ./cmd/goalrail work start \
  --title "<dogfood title>" \
  --body-file /tmp/goalrail-dogfood-body.txt \
  --format json
```

Continue readiness:

```bash
HOME=/tmp/goalrail-dogfood-xdg \
XDG_CONFIG_HOME=/tmp/goalrail-dogfood-xdg \
go run ./cmd/goalrail work continue \
  --goal-id <goal_id> \
  --format json
```

If the next action is clarification, stop for human input. Submit structured
answers only after the human provides them:

```bash
HOME=/tmp/goalrail-dogfood-xdg \
XDG_CONFIG_HOME=/tmp/goalrail-dogfood-xdg \
go run ./cmd/goalrail work answer \
  --clarification-request-id <clarification_request_id> \
  --answers-file /tmp/goalrail-dogfood-answers.json \
  --format json
```

Draft and update a Contract:

```bash
HOME=/tmp/goalrail-dogfood-xdg \
XDG_CONFIG_HOME=/tmp/goalrail-dogfood-xdg \
go run ./cmd/goalrail contract draft \
  --goal-id <goal_id> \
  --format json

HOME=/tmp/goalrail-dogfood-xdg \
XDG_CONFIG_HOME=/tmp/goalrail-dogfood-xdg \
go run ./cmd/goalrail contract update \
  --contract-id <contract_id> \
  --fields-file /tmp/goalrail-dogfood-contract-fields.json \
  --format json
```

Submit only after human review of the draft fields:

```bash
HOME=/tmp/goalrail-dogfood-xdg \
XDG_CONFIG_HOME=/tmp/goalrail-dogfood-xdg \
go run ./cmd/goalrail contract submit \
  --contract-id <contract_id> \
  --format json
```

Approve only after explicit human approval:

```bash
HOME=/tmp/goalrail-dogfood-xdg \
XDG_CONFIG_HOME=/tmp/goalrail-dogfood-xdg \
go run ./cmd/goalrail contract approve \
  --contract-id <contract_id> \
  --confirm-user-approval \
  --format json
```

Create and process a plan:

```bash
HOME=/tmp/goalrail-dogfood-xdg \
XDG_CONFIG_HOME=/tmp/goalrail-dogfood-xdg \
go run ./cmd/goalrail work plan \
  --contract-id <contract_id> \
  --format json

cd <repo-root>/apps/worker
go run ./cmd/goalrail-worker \
  --server-url http://localhost:8080 \
  --worker-id goalrail-dogfood-planner-001 \
  --once

cd <repo-root>/apps/cli
HOME=/tmp/goalrail-dogfood-xdg \
XDG_CONFIG_HOME=/tmp/goalrail-dogfood-xdg \
go run ./cmd/goalrail work plan status \
  --plan-id <plan_id> \
  --format json
```

Accept the proposal only after explicit human acceptance:

```bash
HOME=/tmp/goalrail-dogfood-xdg \
XDG_CONFIG_HOME=/tmp/goalrail-dogfood-xdg \
go run ./cmd/goalrail work proposal accept \
  --proposal-id <proposal_id> \
  --confirm-user-acceptance \
  --format json
```

After proposal acceptance, hand off to a normal draft PR. Do not run checkout,
execution, runner, gate, or proof commands unless a later approved contract
requires them.

## Mac And Codex Notes

- On macOS, `os.UserConfigDir()` uses `HOME/Library/Application Support`.
  Setting `XDG_CONFIG_HOME` alone may not move the CLI auth file. For
  reproducible agent runs, set both `HOME` and `XDG_CONFIG_HOME` to the same
  local temp directory.
- The browser-loopback login path is useful for humans but not
  noninteractive-friendly. The `--no-browser` path still needs a localhost
  callback.
- Backgrounding the server with `nohup ... &` may not survive the Codex shell
  lifecycle. Treat that as observed environment friction. Prefer an explicit
  local terminal, a local process manager, or a documented temporary workaround
  for long dogfood runs.
- Keep all command transcripts redacted. Never paste token material,
  temporary passwords, JWT secrets, local DB passwords, or full auth files.

## DOGFOOD-000 Findings

- Local setup worked through Postgres, migrations, bootstrap owner, auth,
  server smoke, CLI init, and Project Scan.
- CLI/browser login could not complete cleanly in the noninteractive Codex
  environment, so the documented auth API fallback was needed.
- macOS auth path behavior required `HOME=/tmp/goalrail-dogfood-xdg` in
  addition to `XDG_CONFIG_HOME=/tmp/goalrail-dogfood-xdg`.
- The local init and scan path produced a fresh, clean Project Scan with local
  repository identity, baseline, overlay, and no raw source upload.
- `.goalrail/project.yml` was created as the local marker, but its commit
  policy remained unclear for this dogfood slice.

## DOGFOOD-001 Findings

- Goal creation, clarification, Contract draft/update/submit/approve,
  WorkItemPlan creation, planning worker execution, proposal acceptance, and
  `WorkItem(planned)` materialization all worked through current implemented
  surfaces.
- Repeated short-lived access expiry required manual refresh through the auth
  API; CLI commands reported login expiry instead of refreshing automatically.
- Contract draft output did not expose internal seed/draft IDs needed for full
  human traceability.
- There was no CLI contract review/show surface before approval; read-only API
  GETs were needed to inspect the submitted Contract and current draft body.
- `work plan` did not provide the exact planning worker command to run next.
- The minimal planning worker proposal was too generic for confident human
  review: it proposed implementing the approved contract without projecting the
  docs-specific scope into useful WorkItems.
- Proposal acceptance/status output did not make the created WorkItem easy to
  inspect after acceptance.
- Manual ID handoff was required across Goal, clarification, Contract, plan,
  proposal, and WorkItem commands.

## Known UX And Product Backlog

1. Add a CLI contract show/review command before approval.
2. Refresh expired access tokens before authenticated CLI commands fail.
3. Clarify or support local auth path override for agent and noninteractive
   workflows.
4. Provide a clearer noninteractive/dev auth path.
5. Improve local server lifecycle guidance.
6. Make `work plan` output provide the exact worker command or next-step
   guidance.
7. Make planning proposals project contract-specific scope into WorkItems
   instead of only saying "Implement approved contract."
8. Make proposal acceptance/status expose created WorkItem IDs and task details
   clearly.
9. Decide `.goalrail/project.yml` marker policy.

## First Dogfood Trace

- organization_id: `019e3780-bab7-7a9a-99d2-fd74d5b63547`
- project_id: `019e3784-96d4-7626-a526-d074bccd3f92`
- repo_binding_id: `019e3784-96dc-76d2-88c5-c6a7c95c8766`
- goal_id: `019e3794-6020-7a9a-ad64-a8bc481c1478`
- contract_id: `019e37a1-279b-720f-a559-ee5ae03c14c4`
- plan_id: `019e37da-19c5-7e77-9ea6-533395f2be4a`
- proposal_id: `019e37e0-9ff9-75a9-a3f9-2c49cf54f187`
- work_item_id: `019e3875-f2c7-716f-9591-911e149cc62a`

## Draft PR Handoff

Use a `goalrail/` branch prefix for Goalrail-managed implementation branches.
Open a draft PR first. The PR body should include:

- Goalrail IDs;
- Summary;
- Scope;
- Non-goals;
- ComponentImpact;
- DocImpact;
- Checks run;
- UX friction observed;
- Deferred work.

For docs-only dogfood PRs, state that no Go or web tests were run because no
runtime code changed. Always run `git diff --check` and verify that local
markers, auth files, token material, temporary passwords, local DB passwords,
and private machine paths are not staged or committed.

## Cleanup Examples

Do not run cleanup until the dogfood thread is done.

```bash
kill "$(cat /tmp/goalrail-dogfood-server.pid)" 2>/dev/null || true
docker stop goalrail-dogfood-postgres
rm -rf /tmp/goalrail-dogfood-*
```
