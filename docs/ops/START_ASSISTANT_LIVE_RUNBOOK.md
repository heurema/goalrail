---
id: start_assistant_live_runbook
title: Start Assistant Live Runbook
kind: ops_status
authority: operational
status: current
owner: ops
truth_surfaces:
  - start_assistant_live_route
  - start_assistant_deploy_smoke
lifecycle: active-core
review_after: 2026-08-05
supersedes: []
superseded_by: null
related_docs:
  - docs/ops/START_ASSISTANT_STAGE_3B_PLAN.md
  - docs/ops/START_ASSISTANT_WORKER_ARCHITECTURE.md
  - docs/ops/START_ASSISTANT_PUBLIC_KB_PIPELINE.md
  - docs/ops/CONSOLE_MAIN_DEPLOYMENT_WIRING.md
  - apps/workers/start-assistant/README.md
  - apps/web/console/README.md
---
# Start Assistant Live Runbook

## Current live state

The English global `/start` surface is live as a bounded public entry slice:

- page route: `https://goalrail.dev/start`;
- page owner: `apps/web/console`;
- static deployment owner: external `11me/infra` Flux GitOps path;
- console source ref verified live: `031a03d85bd19c754b773ad365eca9acfb462345`;
- assistant route: `POST https://goalrail.dev/api/start-chat`;
- assistant owner: separate Cloudflare Worker from
  `apps/workers/start-assistant`;
- Cloudflare route: `goalrail.dev/api/start-chat*`;
- Worker version verified live: `77b2dbc5-b7aa-42d0-b91b-3b313f8c6f50`;
- public KB commit SHA:
  `263075db460d762fe7fa1f09d30709bc68e8eb5c`;
- public KB updated at: `2026-05-07T15:19:12.980Z`.

The assistant route is not owned by the core `apps/server` API. The core API
continues to serve `https://api.goalrail.dev` for authenticated product routes.

## Ownership boundaries

- `/start` UI: `apps/web/console`.
- Static serving and SPA fallback: external `11me/infra`.
- `/api/start-chat`: Cloudflare Worker from `apps/workers/start-assistant`.
- Public KB source of truth: `.goalrail/public-kb/manifest.yaml` plus explicitly
  whitelisted repository docs.
- Public KB retrieval artifact: provider-side compiled index, not source of
  truth.
- Runtime secrets and provider config: deployment-managed only, never committed.

## Deploy and update

### Public KB sync

Public KB sync is automated as an operator-triggered GitHub Actions workflow:

```bash
GH_SAFE_ACCOUNT=t3chn gh workflow run start-assistant-public-kb-sync.yml \
  -f publish_to_worker=true \
  -f confirm=PUBLISH_START_ASSISTANT_KB
```

The workflow validates the public KB on PRs and relevant `main` pushes without
secrets. The publish path runs only through `workflow_dispatch`, protected
environment `start-assistant-kb-sync`, and deployment secrets. It uploads a new
OpenAI file_search index, updates Worker runtime secrets through Wrangler, and
deploys the Worker route with `--keep-vars`, then runs a live assistant
freshness smoke. It does not print provider IDs or Worker config values to logs.

### Worker route

From the Worker package:

```bash
cd apps/workers/start-assistant
npx --yes wrangler whoami
npx --yes wrangler deploy --config wrangler.toml --route "goalrail.dev/api/start-chat*" --keep-vars
```

`--keep-vars` is intentional: production values for `OPENAI_API_KEY`,
`OPENAI_START_MODEL`, `OPENAI_START_VECTOR_STORE_ID`,
`START_ASSISTANT_KB_REVISION`, and `START_ASSISTANT_KB_UPDATED_AT` are
deployment-managed and must not be written into the repository.

### Console route

The main console deployment is owned outside this repository by `11me/infra`.
The known deployment helper is:

```bash
<infra-repo>/scripts/deploy-goalrail-console.sh \
  --goalrail-repo <normal-goalrail-clone> \
  --ref <goalrail-main-commit> \
  --commit-push \
  --reconcile
```

Use a normal clone with a real `.git/` directory for that script. Some Codex
worktrees expose `.git` as a file and are not suitable as the deploy source.

If `https://goalrail.dev/start` returns 404 after a console deploy, check the
external nginx SPA fallback first. The required behavior is equivalent to:

```text
try_files $uri $uri/ /index.html
```

## Smoke checks

Public DNS can be checked through DNS-over-HTTPS if local resolver tooling is
timing out:

```bash
curl -sS 'https://1.1.1.1/dns-query?name=goalrail.dev&type=A' \
  -H 'accept: application/dns-json'
```

Use one returned A record with `curl --resolve` when local DNS is unreliable:

```bash
curl --resolve goalrail.dev:443:<ip> -D - https://goalrail.dev/start
```

Expected:

- HTTP 200;
- no `Set-Cookie`;
- app bundle contains `/api/start-chat`.

Positive assistant smoke:

```bash
curl --resolve goalrail.dev:443:<ip> -sS -D - \
  -X POST https://goalrail.dev/api/start-chat \
  -H 'Content-Type: application/json' \
  --data '{"question":"What is Goalrail?"}'
```

Expected:

- HTTP 200;
- answer grounded in public Goalrail materials;
- at least one source reference;
- `knowledge.commit_sha`;
- `knowledge.updated_at`;
- disclaimer saying the page cannot scan repos or execute code.

Boundary smokes:

```bash
curl --resolve goalrail.dev:443:<ip> -sS -D - \
  -X POST https://goalrail.dev/api/start-chat \
  -H 'Content-Type: application/json' \
  --data '{"question":"Can you scan my repository?"}'
```

Expected: safe refusal, no repo-scan claim.

```bash
curl --resolve goalrail.dev:443:<ip> -sS -o /tmp/start-get.json \
  -w 'status=%{http_code}\n' https://goalrail.dev/api/start-chat
```

Expected: `status=405`.

```bash
curl --resolve goalrail.dev:443:<ip> -sS -o /tmp/start-extra.json \
  -w 'status=%{http_code}\n' \
  -X POST https://goalrail.dev/api/start-chat/extra \
  -H 'Content-Type: application/json' \
  --data '{"question":"What is Goalrail?"}'
```

Expected: `status=404`.

Browser smoke:

- `/start` loads without auth;
- static quick questions remain available;
- live answers show sources and freshness;
- mobile layout has no horizontal overflow.

## Rollback

Worker rollback is pointer-based:

1. redeploy the previous Worker version, or repoint Worker runtime config to the
   previous known-good vector store;
2. rerun positive and refusal smokes;
3. expire the failed retrieval index only after diagnosis.

Do not roll back by editing public source docs to match a broken retrieval
artifact.

Console rollback:

1. redeploy the previous known-good Goalrail source ref through `11me/infra`;
2. verify `/start` and the root console route;
3. verify the nginx SPA fallback still serves client routes.

## Non-goals and boundaries

The live `/start` assistant still must not add or imply:

- repo scan;
- user file upload;
- private code ingestion;
- code execution;
- analytics;
- tracking;
- cookies;
- sessions;
- CRM;
- browser OpenAI keys;
- core API ownership of `/api/start-chat`;
- mature self-serve SaaS availability;
- broad autonomous agent platform behavior.

## Known limitations

- Public KB sync is operator-triggered through GitHub Actions. It is not
  automatic on every `main` push.
- Runtime manifest storage is still Worker secrets, not KV/R2.
- Provider index identifiers and secrets are not recorded in the repository.
- Route-specific metadata is client-side in the Vite SPA. The app-shell HTML is
  still shared before React runs.
- The live route depends on external infra SPA fallback. That fallback is
  recorded here as required behavior, not as repo-owned infrastructure source.
- Local resolver tooling may time out for `goalrail.dev`; DNS-over-HTTPS plus
  `curl --resolve` is the current reliable smoke path from this workstation.

## Next slice

The next bounded slice is manifest lifecycle hardening: move from Worker-secret
runtime pointers to KV/R2 or another approved manifest pointer with explicit
rollback retention.
