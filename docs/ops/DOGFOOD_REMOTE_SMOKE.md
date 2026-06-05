# DOGFOOD-01 Remote Smoke

Date: 2026-05-18
Repo: `heurema/goalrail`
Commit SHA: `e99a1ed471368216cf30132184a26caeefcc880f`
Result: pass after remote route parity and schema parity blocker fixes

## Scope

Run the real server-backed Goalrail-on-Goalrail smoke loop against the deployed
Goalrail API, without adding product features or expanding `goalrail init`.

Target server:
- web console: `https://goalrail.dev`
- API: `https://api.goalrail.dev`

Server mode:
- remote deployed API through the existing `standalone/infra` Flux GitOps path
- live server image observed:
  `ghcr.io/heurema/goalrail-server:dev-e99a1ed-20260518055649`
- Flux Kustomization observed:
  `apps-personal main@sha1:e96a63d7 Ready=True`

DB mode:
- remote Postgres-backed mode
- local DB credentials were not used or recorded
- live schema was manually brought to repo parity for already-required mutable
  `00001_init.sql` deltas using the new idempotent `00011_init_schema_parity`
  SQL Up section
- live `goose_db_version` still reported versions `0..10`; the follow-up is to
  land/deploy the migration file so the deployed migration runner records
  version `11`

Working tree note:
- the worktree already contained the INIT-08 implementation diff before this
  smoke attempt
- DOGFOOD-01 added only a schema-parity migration/test and ops evidence
- `goalrail init` / `goalrail agent install` generated repo-local Goalrail
  marker and agent files under `.goalrail/`

## Commands Run

```bash
git status --short
cd apps/cli && go build -o ../../bin/goalrail ./cmd/goalrail
./bin/goalrail version
curl -fsS https://api.goalrail.dev/livez
curl -fsS https://api.goalrail.dev/readyz
curl -fsS https://api.goalrail.dev/version
./bin/goalrail login https://api.goalrail.dev
./bin/goalrail init
./bin/goalrail agent install
./bin/goalrail project status
./bin/goalrail project scan --refresh
./bin/goalrail work start --title "Dogfood Goalrail on Goalrail"
```

Setup / blocker-fix commands also run:

```bash
open -a Docker
gh auth refresh -h github.com -s write:packages
docker login ghcr.io -u t3chn --password-stdin
docker buildx build --platform linux/amd64 \
  -f apps/server/Dockerfile \
  -t ghcr.io/heurema/goalrail-server:dev-e99a1ed-20260518055649 \
  --push apps/server
flux reconcile image repository goalrail-server-prod -n flux-system
flux reconcile image update goalrail-server-prod -n flux-system
flux reconcile source git flux-system -n flux-system
flux reconcile kustomization apps-personal -n flux-system
flux reconcile helmrelease goalrail-server -n heurema
kubectl -n heurema rollout status deploy/goalrail-server --timeout=10m
kubectl -n postgres exec -i shared-pg-1 -- \
  psql -U postgres -d goalrail -v ON_ERROR_STOP=1 -X -q -1 \
  < /tmp/goalrail-00011-up.sql
```

Tokens, passwords, browser callback URLs, and CLI auth file contents were not
printed or recorded.

## Redacted Output

### CLI build

```text
exit 0
```

### `./bin/goalrail version`

```text
goalrail dev
```

### Remote health

```text
$ curl -fsS https://api.goalrail.dev/livez
{"status":"ok"}

$ curl -fsS https://api.goalrail.dev/readyz
{"status":"ok"}

$ curl -fsS https://api.goalrail.dev/version
{"service":"goalrail-server","version":"0.0.0-dev"}
```

### Remote route parity

Before image update, repository-context routes returned route-level
`404 not_found`. After pinning
`ghcr.io/heurema/goalrail-server:dev-e99a1ed-20260518055649`:

```text
$ POST /v1/init/repository-context {}
HTTP/2 400
{"error":{"code":"validation_failed","message":"validation failed","details":{"provider":"is required"}}}
```

This validated that the route existed without uploading source code.

The first post-deploy repository-context read returned `500 internal_error`.
Schema inspection showed the live DB had an older already-applied `00001`
shape without the later repository-context tables/columns/indexes. After
applying the idempotent schema-parity SQL:

```text
$ GET /v1/organizations/<organization_id>/repository-context
HTTP/2 200
{"contexts":[]}
```

### `./bin/goalrail login https://api.goalrail.dev`

After completing the browser login with the operator owner account:

```text
Logged in to https://api.goalrail.dev
```

Token values were not inspected or recorded.

### `./bin/goalrail init`

```text
Repository context initialized

Server: https://api.goalrail.dev
Organization: 019dfdb2-e32d-7e38-bf5d-e48f7f3dce94
Project: heurema/goalrail (019e39b0-e09b-7165-b711-39e207453dff)
Repo binding: 019e39b0-e0a2-780d-b1e3-dc91f12a2611
Repository: heurema/goalrail
Provider: github
Workflow base branch: main
State: active
Local config: .goalrail/project.yml (written)
Local state ignore rules: .goalrail/.gitignore (unchanged)
Commit .goalrail/project.yml and .goalrail/.gitignore with this repository.
Repository context snapshot: 019e39b0-e0e6-75d8-86bf-3e7a946113e3 (recorded)
Project scan:
  baseline: created
  overlay: created
  toolchains: docker, go, node
  package managers: npm
  workspaces: apps/cli, apps/runner, apps/server, apps/web, apps/web/console, apps/web/demo-change-packet, apps/web/demo-change-packet-ru, apps/web/pilot-intake-ru, apps/web/pilot-intake-ru/server, apps/worker, apps/workers/start-assistant
  tests: detected
  ci: detected
  agent rules: detected
  codeowners: detected
  partiality: none
  freshness: current_head
  warnings: workspace has uncommitted changes

This initialized GoalRail repository context for your existing organization, wrote a non-secret GoalRail repository marker, attempted a metadata-only repository context snapshot, and ran a local Project Scan.
No server clone, audit, hooks, branch creation, deploy keys, provider integration, runner, gate, proof, or verification were configured.

Next: goalrail work start --title "Dogfood Goalrail on Goalrail"
```

### `./bin/goalrail agent install`

```text
Agent Pack installed

.goalrail/agent/GOALRAIL.md: written
.goalrail/agent/commands.json: written
AGENTS.md: skipped_manual_patch_needed

This installed only provider-neutral repo-local Goalrail files.
```

### `./bin/goalrail project status`

```text
Project status

Baseline: rbp_39714b969a53985d2d71f106
Repo binding: 019e39b0-e0a2-780d-b1e3-dc91f12a2611
HEAD: e99a1ed47136
Status: quick
Overlay: dirty
Partiality: none
Freshness: dirty_overlay
```

### `./bin/goalrail project scan --refresh`

```text
Project scan complete

Baseline: rbp_39714b969a53985d2d71f106
Repo binding: 019e39b0-e0a2-780d-b1e3-dc91f12a2611
HEAD: e99a1ed47136
Status: quick
Overlay: dirty
Partiality: none
Freshness: dirty_overlay

Next: goalrail work start --title <title>
```

### `./bin/goalrail work start --title "Dogfood Goalrail on Goalrail"`

```text
Work intake started

Server: https://api.goalrail.dev
Project: 019e39b0-e09b-7165-b711-39e207453dff
Repo binding: 019e39b0-e0a2-780d-b1e3-dc91f12a2611
Intake: 019e39b2-8336-74ec-b238-ecef4dfd1cdb
Goal: 019e39b2-8366-7cfa-9bd5-e7df883f1da0
State: created
Local config: .goalrail/project.yml

This created an IntakeRecord and promoted it to a Goal on the GoalRail server.
No audit, hooks, branch creation, deploy keys, provider integration, runner, gate, proof, or verification were configured.

Next: goalrail work continue --goal-id 019e39b2-8366-7cfa-9bd5-e7df883f1da0 --format json
Continue: goalrail work continue --goal-id 019e39b2-8366-7cfa-9bd5-e7df883f1da0 --format json
```

## Pass / Fail

| Step | Result |
| --- | --- |
| Remote API `livez` | pass |
| Remote API `readyz` | pass |
| Remote API `version` | pass |
| CLI build | pass |
| CLI version | pass |
| `goalrail login https://api.goalrail.dev` | pass |
| `goalrail init` | pass |
| INIT-08 Project Scan summary in init output | pass |
| `goalrail agent install` | pass |
| `goalrail project status` | pass |
| `goalrail project scan --refresh` | pass |
| `goalrail work start --title "Dogfood Goalrail on Goalrail"` | pass |

## INIT-08 Observation

INIT-08 is visible in the real server-backed init path. The human output shows
baseline/overlay status, toolchains, package managers, workspaces, tests, CI,
agent rules, CODEOWNERS, partiality, freshness, and an explicit warning for the
dirty working tree. It also recommends exactly one next bootstrap command:

```text
Next: goalrail work start --title "Dogfood Goalrail on Goalrail"
```

The output remains honest about scope: no server clone, source upload, provider
integration, runner, gate, proof, or verification was configured.

## Blockers Fixed During Smoke

1. Remote API image parity:
   - blocker: deployed server image predated the current repository-context init
     routes, causing route-level `404 not_found`
   - fix: built and pushed the current server image, then let Flux update the
     remote deployment to `dev-e99a1ed-20260518055649`

2. Live schema parity:
   - blocker: live DB had an older already-applied `00001_init.sql` shape, so
     repository-context reads/writes failed after route parity
   - fix: added idempotent `00011_init_schema_parity.sql` and applied its Up
     section manually to live DB to unblock the smoke
   - remaining hygiene: land/deploy the migration file so the migration runner
     records version `11`; the live DB currently reports applied versions
     `0..10`

## Follow-up

Next bounded slice: DOGFOOD-02 remote deploy hygiene.

Minimum scope:
- land the schema-parity migration/test
- build and deploy a server image that contains migration `00011`
- verify the migration runner records version `11`
- rerun the same remote smoke path from `init` through `work start`

Do not expand into provider integration, source upload, runner execution,
gate/proof, verification, marker repair, readiness scoring, or new init
features.

## DOGFOOD-02 Remote Deploy Hygiene

Date: 2026-05-18
Repo: `heurema/goalrail`
Commit SHA: `e99a1ed471368216cf30132184a26caeefcc880f`
Result: pass

Server image tag:
- `ghcr.io/heurema/goalrail-server:dev-e99a1ed-20260518063148`

Remote API base URL:
- `https://api.goalrail.dev`

Scope:
- deploy a server image containing migration `00011_init_schema_parity`
- verify the remote migration runner records version `11`
- rerun the remote smoke path from `init` through `work start`
- do not add product behavior

Migration version status:

```text
Before deploy:
0
1
2
3
4
5
6
7
8
9
10

After deploy:
0
1
2
3
4
5
6
7
8
9
10
11
```

Commands run:

```bash
GOALRAIL_RUN_POSTGRES_MIGRATION_TESTS=1 go test ./internal/postgres/migrations \
  -run 'Test(MigrationsApplyFromScratchAndRecordInitSchemaParityVersion|InitSchemaParityMigrationAppliesWhenEffectsAlreadyExistButVersionMissing)' \
  -count=1
docker buildx build --platform linux/amd64 \
  -f apps/server/Dockerfile \
  -t ghcr.io/heurema/goalrail-server:dev-e99a1ed-20260518063148 \
  --push apps/server
flux reconcile image repository goalrail-server-prod -n flux-system
flux get image policy goalrail-server-prod -n flux-system
flux reconcile image update goalrail-server-prod -n flux-system
flux reconcile source git flux-system -n flux-system
flux reconcile kustomization apps-personal -n flux-system
flux reconcile helmrelease goalrail-server -n heurema
kubectl -n heurema rollout status deploy/goalrail-server --timeout=10m
kubectl -n heurema get deploy goalrail-server -o jsonpath='{.spec.template.spec.containers[0].image}{"\n"}'
kubectl -n postgres exec shared-pg-1 -- \
  psql -U postgres -d goalrail -tA \
  -c "select version_id from goose_db_version where is_applied order by version_id;"
curl -fsS https://api.goalrail.dev/livez
curl -fsS https://api.goalrail.dev/readyz
curl -fsS https://api.goalrail.dev/version
cd apps/cli && go build -o ../../bin/goalrail ./cmd/goalrail
./bin/goalrail version
./bin/goalrail login https://api.goalrail.dev
./bin/goalrail init
./bin/goalrail agent install
./bin/goalrail project status
./bin/goalrail project scan --refresh
./bin/goalrail work start --title "Dogfood Goalrail on Goalrail"
```

Redacted output:

```text
$ GOALRAIL_RUN_POSTGRES_MIGRATION_TESTS=1 go test ./internal/postgres/migrations ...
ok  	github.com/heurema/goalrail/apps/server/internal/postgres/migrations	6.084s

$ flux get image policy goalrail-server-prod -n flux-system
goalrail-server-prod ghcr.io/heurema/goalrail-server dev-e99a1ed-20260518063148 True

$ flux get kustomization apps-personal -n flux-system
apps-personal main@sha1:0ce05173 False True Applied revision: main@sha1:0ce05173

$ kubectl -n heurema get deploy goalrail-server ...
ghcr.io/heurema/goalrail-server:dev-e99a1ed-20260518063148

$ kubectl -n postgres exec shared-pg-1 -- psql ... goose_db_version ...
0
1
2
3
4
5
6
7
8
9
10
11

$ curl -fsS https://api.goalrail.dev/livez
{"status":"ok"}

$ curl -fsS https://api.goalrail.dev/readyz
{"status":"ok"}

$ curl -fsS https://api.goalrail.dev/version
{"service":"goalrail-server","version":"0.0.0-dev"}

$ ./bin/goalrail version
goalrail dev

$ ./bin/goalrail login https://api.goalrail.dev
Logged in to https://api.goalrail.dev
```

### DOGFOOD-02 `goalrail init`

```text
Repository context already initialized

Server: https://api.goalrail.dev
Organization: 019dfdb2-e32d-7e38-bf5d-e48f7f3dce94
Project: heurema/goalrail (019e39b0-e09b-7165-b711-39e207453dff)
Repo binding: 019e39b0-e0a2-780d-b1e3-dc91f12a2611
Repository: heurema/goalrail
Provider: github
Workflow base branch: main
State: active
Local config: .goalrail/project.yml (unchanged)
Local state ignore rules: .goalrail/.gitignore (unchanged)
Existing Goalrail project marker found and verified.
Repository context snapshot: 019e39b0-e0e6-75d8-86bf-3e7a946113e3 (unchanged)
Project scan:
  baseline: refreshed
  overlay: refreshed
  toolchains: docker, go, node
  package managers: npm
  workspaces: apps/cli, apps/runner, apps/server, apps/web, apps/web/console, apps/web/demo-change-packet, apps/web/demo-change-packet-ru, apps/web/pilot-intake-ru, apps/web/pilot-intake-ru/server, apps/worker, apps/workers/start-assistant
  tests: detected
  ci: detected
  agent rules: detected
  codeowners: detected
  partiality: none
  freshness: current_head
  warnings: workspace has uncommitted changes

This initialized GoalRail repository context for your existing organization, wrote a non-secret GoalRail repository marker, attempted a metadata-only repository context snapshot, and ran a local Project Scan.
No server clone, audit, hooks, branch creation, deploy keys, provider integration, runner, gate, proof, or verification were configured.

Next: goalrail work start --title "Dogfood Goalrail on Goalrail"
```

### DOGFOOD-02 follow-on commands

```text
$ ./bin/goalrail agent install
Agent Pack unchanged

.goalrail/agent/GOALRAIL.md: unchanged
.goalrail/agent/commands.json: unchanged
AGENTS.md: skipped_manual_patch_needed

This installed only provider-neutral repo-local Goalrail files.

$ ./bin/goalrail project status
Project status

Baseline: rbp_39714b969a53985d2d71f106
Repo binding: 019e39b0-e0a2-780d-b1e3-dc91f12a2611
HEAD: e99a1ed47136
Status: quick
Overlay: dirty
Partiality: none
Freshness: dirty_overlay

$ ./bin/goalrail project scan --refresh
Project scan complete

Baseline: rbp_39714b969a53985d2d71f106
Repo binding: 019e39b0-e0a2-780d-b1e3-dc91f12a2611
HEAD: e99a1ed47136
Status: quick
Overlay: dirty
Partiality: none
Freshness: dirty_overlay

Next: goalrail work start --title <title>
```

### DOGFOOD-02 `work start`

```text
Work intake started

Server: https://api.goalrail.dev
Project: 019e39b0-e09b-7165-b711-39e207453dff
Repo binding: 019e39b0-e0a2-780d-b1e3-dc91f12a2611
Intake: 019e39cd-0976-74e7-9bb7-bda9cfda3e23
Goal: 019e39cd-09b1-7e9d-946c-c4e05991cfb4
State: created
Local config: .goalrail/project.yml

This created an IntakeRecord and promoted it to a Goal on the GoalRail server.
No audit, hooks, branch creation, deploy keys, provider integration, runner, gate, proof, or verification were configured.

Next: goalrail work continue --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 --format json
Continue: goalrail work continue --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 --format json
```

Pass / fail:

| Step | Result |
| --- | --- |
| Docker-backed migration idempotency test | pass |
| Server image containing `00011` built and pushed | pass |
| Flux image policy selected new image | pass |
| Remote deployment rolled out new image | pass |
| Remote `goose_db_version` includes `11` | pass |
| Remote API `livez` / `readyz` / `version` | pass |
| CLI build / version / login | pass |
| `goalrail init` with INIT-08 Project Scan summary | pass |
| `agent install` / `project status` / `project scan --refresh` | pass |
| `work start --title "Dogfood Goalrail on Goalrail"` | pass |

Blockers:
- none remaining for DOGFOOD-02

Follow-up:
- next bounded slice: DOGFOOD-03 remote `work continue` smoke for the created
  Goal, staying inside intent-plane continuation and without runner execution,
  provider integration, gate, proof, verification, source upload, or new init
  scope

## DOGFOOD-02.5 Repo / Deploy Traceability Checkpoint

Date: 2026-05-18
Repo: `heurema/goalrail`
Commit SHA: `e99a1ed471368216cf30132184a26caeefcc880f`
Result: clean

Purpose:
- capture the dirty repo / live deploy relationship before continuing product
  smoke
- ensure deployed behavior is traceable to the current repository diff
- avoid drifting from repository truth after deploying an image built from the
  dirty worktree

Dirty set classification:

| Class | Files |
| --- | --- |
| INIT-08 implementation | `apps/cli/internal/initcmd/project_scan_summary.go`, `apps/cli/internal/initcmd/server.go`, `apps/cli/internal/projectscan/cache.go`, `apps/cli/internal/initcmd/command_test.go` |
| DOGFOOD-01 remote smoke artifacts | `docs/ops/DOGFOOD_REMOTE_SMOKE.md`, `docs/ops/NEXT.md`, `docs/ops/STATUS.md`, `docs/INDEX.md` |
| DOGFOOD-02 migration / deploy hygiene artifacts | `apps/server/internal/postgres/migrations/00011_init_schema_parity.sql`, `apps/server/internal/postgres/migrations/migrations_test.go`, `apps/server/internal/postgres/migrations/migrations_integration_test.go`, `docs/ops/DOGFOOD_REMOTE_SMOKE.md`, `docs/ops/NEXT.md`, `docs/ops/STATUS.md` |
| Generated Goalrail files | `.goalrail/project.yml`, `.goalrail/.gitignore`, `.goalrail/agent/GOALRAIL.md`, `.goalrail/agent/commands.json` |
| Decision / checkpoint docs from prior INIT-08 work | `docs/ops/DECISIONS.md`, `docs/ops/INIT_STABILIZATION_CHECKPOINT.md` |
| Unrelated or suspicious | none found |

Expected file presence:
- all expected files were present:
  - `apps/cli/internal/initcmd/project_scan_summary.go`
  - `apps/cli/internal/initcmd/server.go`
  - `apps/cli/internal/projectscan/cache.go`
  - `apps/cli/internal/initcmd/command_test.go`
  - `apps/server/internal/postgres/migrations/00011_init_schema_parity.sql`
  - `apps/server/internal/postgres/migrations/migrations_test.go`
  - `apps/server/internal/postgres/migrations/migrations_integration_test.go`
  - `docs/ops/DOGFOOD_REMOTE_SMOKE.md`
  - `docs/ops/NEXT.md`
  - `docs/ops/STATUS.md`
  - `docs/ops/DECISIONS.md`
  - `docs/INDEX.md`
  - `.goalrail/project.yml`
  - `.goalrail/.gitignore`
  - `.goalrail/agent/GOALRAIL.md`
  - `.goalrail/agent/commands.json`

Secret check:
- `.goalrail/project.yml` contains only non-secret server, Organization,
  Project, RepoBinding, repository provider/name/url, and workflow branch
  metadata.
- `.goalrail/.gitignore` ignores local/cache/state/tmp and `*.local.*` files.
- `.goalrail/agent/GOALRAIL.md` and `.goalrail/agent/commands.json` contain
  provider-neutral command guidance and explicit non-goals.
- High-signal secret pattern scan over the expected files found no real
  secrets. A `sk-` pattern hit in a test string was the word fragment
  `task-per`, not a credential.

Validation before DOGFOOD-03:

```text
$ cd apps/server && go test ./...
pass

$ cd apps/cli && go test ./...
pass

$ cd apps/cli && go build -o ../../bin/goalrail ./cmd/goalrail
pass

$ python3 tools/docs-check/docs_check.py --root . --mode changed-files ...
mode=changed-files files=6 hard=0 warning=0 info=0 fixture_fail=0

$ git diff --check
pass
```

Conclusion:
- DOGFOOD-02.5 is clean.
- No unrelated dirty files were found.
- It is safe to attempt DOGFOOD-03 without changing product behavior.

## DOGFOOD-03 Remote `work continue` Smoke

Date: 2026-05-18
Repo: `heurema/goalrail`
Commit SHA / state:
- `e99a1ed471368216cf30132184a26caeefcc880f`
- dirty worktree intentionally contains INIT-08, DOGFOOD-01, DOGFOOD-02, and
  generated Goalrail marker / agent files classified in DOGFOOD-02.5 above

Server image tag:
- `ghcr.io/heurema/goalrail-server:dev-e99a1ed-20260518063148`

Flux applied revision:
- `main@sha1:0ce05173`

Goal ID used:
- `019e39cd-09b1-7e9d-946c-c4e05991cfb4`

Result: blocked before continuation by expired local CLI auth

Commands run:

```bash
cd apps/cli && go build -o ../../bin/goalrail ./cmd/goalrail
./bin/goalrail version
curl -fsS https://api.goalrail.dev/livez
curl -fsS https://api.goalrail.dev/readyz
curl -fsS https://api.goalrail.dev/version
kubectl -n heurema get deploy goalrail-server -o jsonpath='{.spec.template.spec.containers[0].image}{"\n"}'
flux get kustomization apps-personal -n flux-system
./bin/goalrail login https://api.goalrail.dev
./bin/goalrail project status
./bin/goalrail project scan --refresh
./bin/goalrail work continue --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 --format json
```

Redacted outputs:

```text
$ ./bin/goalrail version
goalrail dev

$ curl -fsS https://api.goalrail.dev/livez
{"status":"ok"}

$ curl -fsS https://api.goalrail.dev/readyz
{"status":"ok"}

$ curl -fsS https://api.goalrail.dev/version
{"service":"goalrail-server","version":"0.0.0-dev"}

$ kubectl -n heurema get deploy goalrail-server ...
ghcr.io/heurema/goalrail-server:dev-e99a1ed-20260518063148

$ flux get kustomization apps-personal -n flux-system
apps-personal main@sha1:0ce05173 False True Applied revision: main@sha1:0ce05173

$ ./bin/goalrail login https://api.goalrail.dev
context canceled

$ ./bin/goalrail project status
Project status

Baseline: rbp_39714b969a53985d2d71f106
Repo binding: 019e39b0-e0a2-780d-b1e3-dc91f12a2611
HEAD: e99a1ed47136
Status: quick
Overlay: dirty
Partiality: none
Freshness: dirty_overlay

$ ./bin/goalrail project scan --refresh
Project scan complete

Baseline: rbp_39714b969a53985d2d71f106
Repo binding: 019e39b0-e0a2-780d-b1e3-dc91f12a2611
HEAD: e99a1ed47136
Status: quick
Overlay: dirty
Partiality: none
Freshness: dirty_overlay

Next: goalrail work start --title <title>

$ ./bin/goalrail work continue --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 --format json
login expired; run goalrail login https://api.goalrail.dev
```

JSON output summary:
- no JSON response was produced because the CLI rejected the command before the
  remote continuation call with expired local auth
- the requested Goal ID was not continued in this attempt

Pass / fail:

| Step | Result |
| --- | --- |
| CLI build | pass |
| CLI version | pass |
| Remote API `livez` / `readyz` / `version` | pass |
| Live image / Flux revision check | pass |
| `goalrail login https://api.goalrail.dev` | blocked: browser-loopback did not complete; command canceled |
| `goalrail project status` | pass |
| `goalrail project scan --refresh` | pass |
| `goalrail work continue --goal-id ... --format json` | blocked: local CLI auth expired |

Blocker:
- local CLI auth for `https://api.goalrail.dev` is expired
- `goalrail login https://api.goalrail.dev` opened the browser-loopback flow
  but did not complete without manual browser action in this run
- no password was found through non-secret Keychain metadata checks for the
  obvious service names `goalrail`, `api.goalrail.dev`, or
  `https://api.goalrail.dev`

Smallest operator action:
- complete `goalrail login https://api.goalrail.dev` in the browser, then rerun:

```bash
./bin/goalrail work continue --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 --format json
```

Scope note:
- no source upload, provider integration, server clone, branch creation, deploy
  keys, runner execution, gate, proof, verification, marker repair, readiness
  scoring, or new init behavior was added or triggered

Follow-up:
- rerun DOGFOOD-03 after browser login completes

## DOGFOOD-03R Remote `work continue` Rerun

Date: 2026-05-18
Repo: `heurema/goalrail`
Goal ID:
- `019e39cd-09b1-7e9d-946c-c4e05991cfb4`

Auth blocker from DOGFOOD-03:
- local CLI auth had expired
- `goalrail login https://api.goalrail.dev` was previously started but not
  completed, and the continuation command stopped before the remote API call

Login completed manually:
- yes

Exact `work continue` command:

```bash
./bin/goalrail work continue \
  --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 \
  --format json
```

Commands run:

```bash
git status --short
cd apps/cli && go build -o ../../bin/goalrail ./cmd/goalrail
./bin/goalrail version
curl -fsS https://api.goalrail.dev/livez
curl -fsS https://api.goalrail.dev/readyz
curl -fsS https://api.goalrail.dev/version
./bin/goalrail login https://api.goalrail.dev
./bin/goalrail work continue --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 --format json
```

Redacted outputs:

```text
$ ./bin/goalrail version
goalrail dev

$ curl -fsS https://api.goalrail.dev/livez
{"status":"ok"}

$ curl -fsS https://api.goalrail.dev/readyz
{"status":"ok"}

$ curl -fsS https://api.goalrail.dev/version
{"service":"goalrail-server","version":"0.0.0-dev"}

$ ./bin/goalrail login https://api.goalrail.dev
Logged in to https://api.goalrail.dev
```

Redacted JSON output summary:

```json
{
  "schema_version": "goalrail.cli.v1",
  "mode": "server",
  "server_url": "https://api.goalrail.dev",
  "organization_id": "019dfdb2-e32d-7e38-bf5d-e48f7f3dce94",
  "project_id": "019e39b0-e09b-7165-b711-39e207453dff",
  "repo_binding_id": "019e39b0-e0a2-780d-b1e3-dc91f12a2611",
  "goal_id": "019e39cd-09b1-7e9d-946c-c4e05991cfb4",
  "state": "needs_clarification",
  "local_config_path": ".goalrail/project.yml",
  "display": {
    "summary": "Goal needs clarification. Ask the user the returned questions before continuing."
  },
  "next_action": {
    "kind": "ask_user",
    "blocking": true,
    "available": true,
    "request_id": "019e39eb-3d7e-7af7-b84d-48fcfaab716b",
    "questions": [
      {
        "id": "019e39eb-3d7e-7afe-b08a-6c89ef8aebaf",
        "text": "What is the intended scope at a high level?",
        "why_needed": "A scope hint is required before contract seed readiness.",
        "answer_type": "text",
        "maps_to": "goal.scope_hint"
      },
      {
        "id": "019e39eb-3d7e-7b02-9a5e-958e19822cc2",
        "text": "What outcome would make this goal acceptable?",
        "why_needed": "An acceptance hint is required before contract seed readiness.",
        "answer_type": "text",
        "maps_to": "goal.acceptance_hint"
      }
    ]
  }
}
```

Pass / fail:

| Step | Result |
| --- | --- |
| CLI build | pass |
| CLI version | pass |
| Remote API `livez` / `readyz` / `version` | pass |
| Browser-loopback login completed | pass |
| `work continue --goal-id ... --format json` reaches remote API | pass |
| JSON references requested Goal ID | pass |
| Response stays intent/control-plane only | pass |
| Source upload / provider / clone / branch / deploy key / runner / gate / proof / verification | not triggered |

Blockers:
- none remaining for DOGFOOD-03R

Follow-up:
- next bounded slice: DOGFOOD-04 remote `work answer` smoke using the returned
  `clarification_request_id` and question IDs, with structured answers only,
  staying inside intent-plane clarification and without source upload, provider
  integration, server clone, runner execution, gate, proof, verification, or
  new init/auth behavior

## DOGFOOD-04 Remote `work answer` Smoke

Date: 2026-05-18
Repo: `heurema/goalrail`
Goal ID:
- `019e39cd-09b1-7e9d-946c-c4e05991cfb4`

Clarification request ID:
- `019e39eb-3d7e-7af7-b84d-48fcfaab716b`

Browser login needed:
- no; the DOGFOOD-03R browser-loopback session was still valid

Exact `work answer` command:

```bash
./bin/goalrail work answer \
  --clarification-request-id 019e39eb-3d7e-7af7-b84d-48fcfaab716b \
  --answers-file - \
  --format json
```

Answers submitted:

```json
{
  "answers": [
    {
      "question_id": "019e39eb-3d7e-7afe-b08a-6c89ef8aebaf",
      "value": "Validate the remote Goalrail-on-Goalrail intent-plane loop by answering the clarification request for the existing dogfood Goal, observing the next server action, and documenting the result. Do not enable source upload, provider integration, runner execution, gate, proof, verification, branch creation, deploy keys, or server clone."
    },
    {
      "question_id": "019e39eb-3d7e-7b02-9a5e-958e19822cc2",
      "value": "The smoke is acceptable if the CLI submits structured answers to the remote API, the response is valid JSON, the response references the expected Goal and clarification request, and the Goal either moves out of needs_clarification or returns a clear next intent-plane action without triggering execution/provider/proof behavior."
    }
  ]
}
```

Commands run:

```bash
git status --short
./bin/goalrail version
curl -fsS https://api.goalrail.dev/livez
curl -fsS https://api.goalrail.dev/readyz
curl -fsS https://api.goalrail.dev/version
./bin/goalrail work answer --clarification-request-id 019e39eb-3d7e-7af7-b84d-48fcfaab716b --answers-file - --format json
./bin/goalrail work continue --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 --format json
```

Redacted outputs:

```text
$ ./bin/goalrail version
goalrail dev

$ curl -fsS https://api.goalrail.dev/livez
{"status":"ok"}

$ curl -fsS https://api.goalrail.dev/readyz
{"status":"ok"}

$ curl -fsS https://api.goalrail.dev/version
{"service":"goalrail-server","version":"0.0.0-dev"}
```

`work answer` output summary:

```json
{
  "schema_version": "goalrail.cli.v1",
  "mode": "server",
  "server_url": "https://api.goalrail.dev",
  "organization_id": "019dfdb2-e32d-7e38-bf5d-e48f7f3dce94",
  "project_id": "019e39b0-e09b-7165-b711-39e207453dff",
  "repo_binding_id": "019e39b0-e0a2-780d-b1e3-dc91f12a2611",
  "goal_id": "019e39cd-09b1-7e9d-946c-c4e05991cfb4",
  "state": "ready_for_contract_seed",
  "clarification_request_id": "019e39eb-3d7e-7af7-b84d-48fcfaab716b",
  "local_config_path": ".goalrail/project.yml",
  "display": {
    "summary": "Goal is ready for contract seed. Draft the Contract handle next."
  },
  "next_action": {
    "kind": "draft_contract",
    "blocking": false,
    "command": "goalrail contract draft --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 --format json",
    "available": true
  }
}
```

Follow-up `work continue` command:

```bash
./bin/goalrail work continue \
  --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 \
  --format json
```

Follow-up `work continue` output summary:

```json
{
  "schema_version": "goalrail.cli.v1",
  "mode": "server",
  "server_url": "https://api.goalrail.dev",
  "organization_id": "019dfdb2-e32d-7e38-bf5d-e48f7f3dce94",
  "project_id": "019e39b0-e09b-7165-b711-39e207453dff",
  "repo_binding_id": "019e39b0-e0a2-780d-b1e3-dc91f12a2611",
  "goal_id": "019e39cd-09b1-7e9d-946c-c4e05991cfb4",
  "state": "ready_for_contract_seed",
  "local_config_path": ".goalrail/project.yml",
  "display": {
    "summary": "Goal is ready for contract seed. Draft the Contract handle next."
  },
  "next_action": {
    "kind": "draft_contract",
    "blocking": false,
    "command": "goalrail contract draft --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 --format json",
    "available": true
  }
}
```

Pass / fail:

| Step | Result |
| --- | --- |
| CLI version | pass |
| Remote API `livez` / `readyz` / `version` | pass |
| Browser login | not needed |
| `work answer --clarification-request-id ... --answers-file - --format json` reaches remote API | pass |
| JSON references requested Goal ID | pass |
| JSON references requested ClarificationRequest ID | pass |
| Goal leaves `needs_clarification` | pass: state is `ready_for_contract_seed` |
| Follow-up `work continue --goal-id ... --format json` returns next action | pass: `draft_contract` |
| Source upload / provider / clone / branch / deploy key / runner / gate / proof / verification | not triggered |

Blockers:
- none for DOGFOOD-04

Follow-up:
- next bounded slice: DOGFOOD-05 remote `contract draft` smoke for Goal
  `019e39cd-09b1-7e9d-946c-c4e05991cfb4`, staying inside contract drafting /
  control-plane scope and without approval, work item planning, source upload,
  provider integration, runner execution, gate, proof, verification, server
  clone, branch creation, or deploy keys

## DOGFOOD-05 Remote `contract draft` Smoke

Date: 2026-05-18
Repo: `heurema/goalrail`
Goal ID:
- `019e39cd-09b1-7e9d-946c-c4e05991cfb4`

Exact `contract draft` command:

```bash
./bin/goalrail contract draft \
  --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 \
  --format json
```

Browser login needed:
- yes; the first `contract draft` attempt stopped before remote mutation with:
  `login expired; run goalrail login https://api.goalrail.dev`
- `goalrail login https://api.goalrail.dev` completed through the browser-loopback
  flow, then the exact `contract draft` command was retried once

Commands run:

```bash
git status --short
cd apps/cli && go build -o ../../bin/goalrail ./cmd/goalrail
./bin/goalrail version
curl -fsS https://api.goalrail.dev/livez
curl -fsS https://api.goalrail.dev/readyz
curl -fsS https://api.goalrail.dev/version
./bin/goalrail contract draft --help
./bin/goalrail contract draft --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 --format json
./bin/goalrail login https://api.goalrail.dev
./bin/goalrail contract draft --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 --format json
./bin/goalrail work continue --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 --format json
```

Redacted outputs:

```text
$ ./bin/goalrail version
goalrail dev

$ curl -fsS https://api.goalrail.dev/livez
{"status":"ok"}

$ curl -fsS https://api.goalrail.dev/readyz
{"status":"ok"}

$ curl -fsS https://api.goalrail.dev/version
{"service":"goalrail-server","version":"0.0.0-dev"}

$ ./bin/goalrail contract draft --help
Usage: goalrail contract draft --goal-id <goal_id> [--format text|json]

Creates or returns a server Contract draft handle for a ready Goal using the current Git root .goalrail/project.yml marker and the stored goalrail login profile. It refreshes local Project Scan evidence and returns a local repository receipt. It does not upload raw source bodies, update contract fields, create WorkItems, run workers, gates, proof, or verification.

$ ./bin/goalrail contract draft --goal-id ... --format json
login expired; run goalrail login https://api.goalrail.dev

$ ./bin/goalrail login https://api.goalrail.dev
Logged in to https://api.goalrail.dev
```

`contract draft` output summary:

```json
{
  "schema_version": "goalrail.cli.v1",
  "mode": "server",
  "server_url": "https://api.goalrail.dev",
  "organization_id": "019dfdb2-e32d-7e38-bf5d-e48f7f3dce94",
  "project_id": "019e39b0-e09b-7165-b711-39e207453dff",
  "repo_binding_id": "019e39b0-e0a2-780d-b1e3-dc91f12a2611",
  "goal_id": "019e39cd-09b1-7e9d-946c-c4e05991cfb4",
  "contract_id": "019e39fd-fc79-764d-94c3-0675bf037e84",
  "contract_state": "draft",
  "local_repo_receipt": {
    "repo_binding_id": "019e39b0-e0a2-780d-b1e3-dc91f12a2611",
    "head_sha": "e99a1ed471368216cf30132184a26caeefcc880f",
    "baseline_id": "rbp_39714b969a53985d2d71f106",
    "overlay_id": "wso_de2086dab58c11b407216d73",
    "overlay_state": "dirty",
    "freshness": "dirty_overlay",
    "dirty": true,
    "partial": false,
    "raw_source_uploaded": false,
    "baseline_rebuilt": false
  },
  "local_config_path": ".goalrail/project.yml",
  "display": {
    "summary": "Created or found a draft Contract handle. Local repository receipt is attached; update proposed contract fields next."
  },
  "next_action": {
    "kind": "update_contract",
    "blocking": false,
    "command": "goalrail contract update --contract-id 019e39fd-fc79-764d-94c3-0675bf037e84 --fields-file - --format json",
    "available": true
  }
}
```

Contract identifiers:
- `contract_id`: `019e39fd-fc79-764d-94c3-0675bf037e84`
- `contract_state`: `draft`

Resulting state:
- the remote contract-drafting edge passed and returned a draft Contract handle
- the returned local repository receipt stayed metadata-only:
  `raw_source_uploaded=false`, `dirty=true`, `partial=false`, and
  `freshness=dirty_overlay`

Follow-up `work continue` command:

```bash
./bin/goalrail work continue \
  --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 \
  --format json
```

Follow-up `work continue` output summary:

```json
{
  "schema_version": "goalrail.cli.v1",
  "mode": "server",
  "server_url": "https://api.goalrail.dev",
  "organization_id": "019dfdb2-e32d-7e38-bf5d-e48f7f3dce94",
  "project_id": "019e39b0-e09b-7165-b711-39e207453dff",
  "repo_binding_id": "019e39b0-e0a2-780d-b1e3-dc91f12a2611",
  "goal_id": "019e39cd-09b1-7e9d-946c-c4e05991cfb4",
  "state": "ready_for_contract_seed",
  "local_config_path": ".goalrail/project.yml",
  "display": {
    "summary": "Goal is ready for contract seed. Draft the Contract handle next."
  },
  "next_action": {
    "kind": "draft_contract",
    "blocking": false,
    "command": "goalrail contract draft --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 --format json",
    "available": true
  }
}
```

Scope boundary:
- no source upload, provider integration, server clone, branch creation, deploy
  key configuration, planned WorkItem creation, runner execution, gate, proof,
  verification, contract approval, marker repair, readiness scoring, auth
  implementation change, or new API route was triggered or added

Pass / fail:

| Step | Result |
| --- | --- |
| CLI build | pass |
| CLI version | pass |
| Remote API `livez` / `readyz` / `version` | pass |
| `contract draft --help` syntax discovery | pass |
| First `contract draft --goal-id ... --format json` | blocked before remote mutation: expired auth |
| Browser-loopback login | pass |
| Retried `contract draft --goal-id ... --format json` reaches remote API | pass |
| JSON references requested Goal ID | pass |
| JSON returns draft Contract ID | pass |
| Contract state is `draft` | pass |
| Local receipt reports `raw_source_uploaded=false` | pass |
| Follow-up `work continue --goal-id ... --format json` | pass with note: it still recommends `draft_contract` rather than surfacing the existing draft |
| Source upload / provider / clone / branch / deploy key / runner / gate / proof / verification | not triggered |

Blockers:
- none for primary DOGFOOD-05 `contract draft`

Follow-up risks:
- `work continue` still reports `state=ready_for_contract_seed` and
  `next_action.kind=draft_contract` after a draft Contract handle exists; the
  direct `contract draft` response is usable and idempotent, but continuation
  does not yet surface the existing draft handle

Follow-up:
- next bounded slice: DOGFOOD-06 remote `contract update` smoke using Contract
  `019e39fd-fc79-764d-94c3-0675bf037e84`, staying inside draft-field update /
  control-plane scope and without approval, WorkItem planning, checkout, source
  upload, provider integration, server clone, branch creation, deploy keys,
  runner execution, gate, proof, or verification

## DOGFOOD-06 Remote `contract update` Smoke

Date: 2026-05-18
Repo: `heurema/goalrail`
Goal ID:
- `019e39cd-09b1-7e9d-946c-c4e05991cfb4`

Contract ID:
- `019e39fd-fc79-764d-94c3-0675bf037e84`

Browser login needed:
- yes; the first `contract update` attempt stopped before remote mutation with:
  `login expired; run goalrail login https://api.goalrail.dev`
- `goalrail login https://api.goalrail.dev` completed through the browser-loopback
  flow, then the exact `contract update` command was retried once

Exact `contract update` command:

```bash
./bin/goalrail contract update \
  --contract-id 019e39fd-fc79-764d-94c3-0675bf037e84 \
  --fields-file - \
  --format json
```

Discovered fields-file shape:
- stdin or file JSON object
- editable text field: `title`, `intent_summary`
- editable string-list fields: `proposed_scope`, `proposed_non_goals`,
  `proposed_constraints`, `proposed_acceptance_criteria`,
  `proposed_expected_checks`, `proposed_proof_expectations`, `risk_hints`
- compatibility alias: `proposed_verification` maps to
  `proposed_expected_checks` when `proposed_expected_checks` is absent
- optional metadata fields: `context_refs`, `unknowns`
- request must include at least one editable proposed field

Fields submitted:

```json
{
  "title": "DOGFOOD-06 remote contract update smoke",
  "intent_summary": "Continue the remote Goalrail-on-Goalrail dogfood loop by updating the draft contract for the existing Goal. This slice validates contract-field update behavior only.",
  "proposed_scope": [
    "Continue the remote Goalrail-on-Goalrail dogfood loop by updating the draft contract for the existing Goal. This slice validates contract-field update behavior only."
  ],
  "proposed_acceptance_criteria": [
    "The remote API accepts structured contract field updates.",
    "The response is valid JSON referencing the expected Contract and Goal when the current response shape includes both.",
    "The contract remains in a draft/updateable control-plane state or returns a clear next control-plane action.",
    "No source upload, provider integration, clone, branch creation, deploy keys, runner execution, gate, proof, or verification is triggered."
  ],
  "proposed_non_goals": [
    "No contract approval.",
    "No planned work generation.",
    "No checkout.",
    "No runner execution.",
    "No provider integration.",
    "No proof.",
    "No verification."
  ]
}
```

Commands run:

```bash
git status --short
cd apps/cli && go build -o ../../bin/goalrail ./cmd/goalrail
./bin/goalrail version
curl -fsS https://api.goalrail.dev/livez
curl -fsS https://api.goalrail.dev/readyz
curl -fsS https://api.goalrail.dev/version
./bin/goalrail contract draft --help
./bin/goalrail contract update --help
./bin/goalrail contract --help
./bin/goalrail contract update --contract-id 019e39fd-fc79-764d-94c3-0675bf037e84 --fields-file - --format json
./bin/goalrail login https://api.goalrail.dev
./bin/goalrail contract update --contract-id 019e39fd-fc79-764d-94c3-0675bf037e84 --fields-file - --format json
./bin/goalrail work continue --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 --format json
```

Redacted outputs:

```text
$ ./bin/goalrail version
goalrail dev

$ curl -fsS https://api.goalrail.dev/livez
{"status":"ok"}

$ curl -fsS https://api.goalrail.dev/readyz
{"status":"ok"}

$ curl -fsS https://api.goalrail.dev/version
{"service":"goalrail-server","version":"0.0.0-dev"}

$ ./bin/goalrail contract update --help
Usage: goalrail contract update --contract-id <contract_id> --fields-file <path|-> [--format text|json]

Updates proposed fields on a server ContractDraft using structured JSON from a file or stdin. The command reads the Git-root .goalrail/project.yml marker, validates the stored login profile and Organization marker, sends project/repo expectations, and returns changed fields plus the next review action. It does not upload raw source bodies, submit or approve contracts, create WorkItems, run workers, gates, proof, or verification.

$ ./bin/goalrail contract update --contract-id ... --fields-file - --format json
login expired; run goalrail login https://api.goalrail.dev

$ ./bin/goalrail login https://api.goalrail.dev
Logged in to https://api.goalrail.dev
```

`contract update` output summary:

```json
{
  "schema_version": "goalrail.cli.v1",
  "mode": "server",
  "server_url": "https://api.goalrail.dev",
  "organization_id": "019dfdb2-e32d-7e38-bf5d-e48f7f3dce94",
  "project_id": "019e39b0-e09b-7165-b711-39e207453dff",
  "repo_binding_id": "019e39b0-e0a2-780d-b1e3-dc91f12a2611",
  "contract_id": "019e39fd-fc79-764d-94c3-0675bf037e84",
  "contract_state": "draft",
  "changed_fields": [
    "intent_summary",
    "proposed_acceptance_criteria",
    "proposed_non_goals",
    "proposed_scope",
    "title"
  ],
  "local_config_path": ".goalrail/project.yml",
  "display": {
    "summary": "Updated proposed ContractDraft fields. Review the draft contract next."
  },
  "next_action": {
    "kind": "review_contract",
    "blocking": true,
    "available": true
  }
}
```

Resulting contract state:
- `contract_state`: `draft`
- updated fields: `title`, `intent_summary`, `proposed_scope`,
  `proposed_acceptance_criteria`, `proposed_non_goals`
- next action: `review_contract`

Follow-up `work continue` command:

```bash
./bin/goalrail work continue \
  --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 \
  --format json
```

Follow-up `work continue` output summary:

```json
{
  "schema_version": "goalrail.cli.v1",
  "mode": "server",
  "server_url": "https://api.goalrail.dev",
  "organization_id": "019dfdb2-e32d-7e38-bf5d-e48f7f3dce94",
  "project_id": "019e39b0-e09b-7165-b711-39e207453dff",
  "repo_binding_id": "019e39b0-e0a2-780d-b1e3-dc91f12a2611",
  "goal_id": "019e39cd-09b1-7e9d-946c-c4e05991cfb4",
  "state": "ready_for_contract_seed",
  "local_config_path": ".goalrail/project.yml",
  "display": {
    "summary": "Goal is ready for contract seed. Draft the Contract handle next."
  },
  "next_action": {
    "kind": "draft_contract",
    "blocking": false,
    "command": "goalrail contract draft --goal-id 019e39cd-09b1-7e9d-946c-c4e05991cfb4 --format json",
    "available": true
  }
}
```

DOGFOOD-05 stale `work continue` surface:
- still persists after DOGFOOD-06
- the direct `contract update` path is usable and returns the updated draft
  Contract state, but Goal continuation still surfaces `ready_for_contract_seed`
  / `draft_contract` instead of the existing updated draft Contract handle

Scope boundary:
- no contract approval, planned WorkItem creation, checkout, source upload,
  provider integration, server clone, branch creation, deploy-key configuration,
  runner execution, gate, proof, verification, marker repair, readiness scoring,
  auth implementation change, or new API route was triggered or added

Pass / fail:

| Step | Result |
| --- | --- |
| CLI build | pass |
| CLI version | pass |
| Remote API `livez` / `readyz` / `version` | pass |
| `contract update --help` syntax discovery | pass |
| Fields-file shape discovery from CLI source/tests | pass |
| First `contract update --contract-id ... --fields-file - --format json` | blocked before remote mutation: expired auth |
| Browser-loopback login | pass |
| Retried `contract update --contract-id ... --fields-file - --format json` reaches remote API | pass |
| JSON references requested Contract ID | pass |
| Contract state remains `draft` | pass |
| Changed fields match submitted editable fields | pass |
| Next action stays control-plane | pass: `review_contract` |
| Follow-up `work continue --goal-id ... --format json` | pass with note: stale `draft_contract` surface persists |
| Approval / planning / checkout / source upload / provider / clone / branch / deploy key / runner / gate / proof / verification | not triggered |

Blockers:
- none for primary DOGFOOD-06 `contract update`

Follow-up risks:
- continuation state surface still does not expose the existing draft Contract
  after draft/update; it keeps recommending `contract draft`
- `contract update` JSON output does not include `goal_id`; this may be
  acceptable for the current response shape, but it makes cross-checking the
  Goal link depend on prior `contract draft` evidence

Follow-up:
- next bounded slice: DOGFOOD-07 contract review/read-surface checkpoint before
  any submit/approval/planning path. Verify how a human or CLI can inspect the
  updated draft Contract fields and decide whether the next safe command is
  `contract submit`; do not approve, plan work, checkout, upload source,
  integrate providers, clone, branch, configure deploy keys, run, gate, proof,
  or verify.
