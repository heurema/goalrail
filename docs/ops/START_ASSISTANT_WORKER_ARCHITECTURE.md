# Start Assistant Worker Architecture

## Purpose

This document defines the Stage 3A architecture for turning the static `/start`
page into a live source-grounded assistant.

Stage 3A is documentation only. It does not add Worker code, backend routes,
OpenAI calls, vector store sync, secrets, runtime dependencies, analytics, or
tracking.

## Ownership

| Surface | Owner | Boundary |
| --- | --- | --- |
| `/start` page | `apps/web/console` | Public static entry page and browser UI |
| `/api/start-chat` | Separate public-edge assistant Worker | Anonymous public Q&A over approved public Goalrail knowledge |
| Core Goalrail API | `apps/server` | Authenticated product API and canonical Goalrail state |
| Public KB source | GitHub repository | Whitelisted public docs and approved public artifacts |
| Retrieval index | OpenAI vector store | Compiled retrieval artifact, not source of truth |

The first live assistant slice should use a separate assistant Worker / tunnel
Worker path. It should not run inside the core API app unless a later decision
explicitly approves that boundary.

## Route model

```text
Browser /start
  -> POST /api/start-chat
      -> public-edge assistant Worker
          -> approved public KB manifest
          -> OpenAI Responses API with file_search
          -> answer with source references and suggested questions
```

Expected browser behavior:

- same-origin `POST /api/start-chat`;
- JSON only;
- no file upload;
- no browser API keys;
- no direct browser call to OpenAI;
- no cookies, sessions, analytics, or tracking by default.

Expected routing behavior:

- `/start` remains served by the public frontend surface;
- `/api/start-chat` is intercepted by the assistant Worker at the public edge;
- the core API keeps `/v1/...` product routes and does not own anonymous
  assistant traffic in the first live slice.

## Why not browser to OpenAI

The browser must not call OpenAI directly because that would:

- expose provider credentials;
- move prompt and retrieval policy into user-controlled code;
- remove server-side validation and rate controls;
- make source-grounding harder to enforce;
- create an easy path to unbounded cost and abuse;
- make it harder to prevent private code, secrets, or large payloads from
  reaching the provider.

The browser only sends a short public question to Goalrail-owned infrastructure.

## Why not the core API app first

The first live assistant should not run in `apps/server` because:

- `/start` is anonymous public traffic, while the core API is the authenticated
  product control-plane boundary;
- assistant abuse, provider cost, and public prompt handling should be isolated
  from canonical product state;
- the assistant must not imply server-side repo scan, execution, auth, tracker
  sync, CRM, or product workflow maturity;
- an edge Worker is easier to deploy, rate-limit, and roll back independently;
- keeping this outside the core API preserves the current pilot-first scope.

The core API may later expose product-owned assistant or clarification features,
but that is a separate architecture decision.

## Worker responsibilities

The Worker should:

- accept only `POST /api/start-chat`;
- require `Content-Type: application/json`;
- parse a small JSON request with a `question` string;
- trim and limit question length;
- reject file uploads, multipart bodies, binary content, repo-scan requests, and
  unknown large payloads;
- load the approved public KB manifest from Worker configuration, Cloudflare KV,
  or an equivalent deployment-managed source;
- call OpenAI Responses API with file_search against the approved retrieval
  index;
- enforce a short system instruction that forbids product overclaiming;
- normalize response shape to the documented API contract;
- include source references and suggested next questions;
- return safe fallback errors when retrieval or provider calls fail.

The Worker must not:

- scan repositories;
- connect to GitHub on behalf of a visitor;
- accept file uploads;
- execute code or run checks;
- ingest private code;
- ask for credentials;
- store chat history by default;
- send CRM events;
- add analytics or tracking;
- expose model or vector store controls to public users.

## Request and response flow

1. Browser sends:

   ```json
   {
     "question": "What is contract-first execution?"
   }
   ```

2. Worker validates method, content type, size, and `question`.
3. Worker resolves current KB metadata:
   - retrieval index identifier;
   - source commit SHA;
   - knowledge updated timestamp;
   - allowed source list revision.
4. Worker calls OpenAI Responses API with file_search enabled for that index.
5. Worker validates and shapes the response:
   - `answer`;
   - `sources`;
   - `suggested_questions`;
   - `knowledge.updated_at`;
   - `knowledge.commit_sha`;
   - safety disclaimer.
6. Browser renders the answer panel without storing chat history.

## Rate limiting posture

Minimum Stage 3B posture:

- method and content-type validation;
- request body size limit;
- question length limit;
- server-side timeout;
- provider request timeout;
- safe error messages;
- Cloudflare-level rate limiting if available.

Do not create app-level sessions or user identity for rate limiting in the first
slice.

Later options:

- Cloudflare WAF / Rate Limiting rules;
- Turnstile for abuse spikes;
- Durable Object or KV-backed lightweight limiter;
- Cloudflare AI Gateway rate limits.

Any persistent abuse telemetry must be approved separately.

## Logging posture

Default:

- do not log full prompt bodies;
- do not store chat history;
- do not store user-provided private content;
- do not add analytics;
- do not persist IP, user-agent, or fingerprint data from the app layer unless a
  separate decision approves it;
- log only coarse operational errors needed to operate the Worker.

If Cloudflare AI Gateway is introduced, payload logging must be disabled unless
a separate privacy decision approves otherwise.

## Secrets posture

Allowed:

- OpenAI API key stored as a Worker secret;
- optional Cloudflare AI Gateway token stored as a Worker secret;
- deployment-managed KB revision / index identifier visible only to the Worker.

Forbidden:

- API keys in browser JavaScript;
- committed `.env` files;
- real secret values in docs, logs, tests, or screenshots;
- provider credentials in GitHub-tracked files;
- public exposure of raw vector store controls.

The API response may expose public freshness metadata such as short commit SHA
and updated timestamp. It should not expose provider credentials or secret
configuration.

## Deployment assumptions

Preferred Stage 3B deployment shape:

```text
goalrail.dev/start
goalrail.dev/api/start-chat -> separate Cloudflare Worker
```

The first implementation should assume:

- separate Cloudflare Worker project or equivalent public-edge worker;
- route binding for `/api/start-chat`;
- Worker secret for provider credentials;
- deployment-managed KB manifest pointer;
- no changes to `apps/server`;
- no new public auth boundary;
- no repo clone or code execution capability.

If the implementation adds a new code path such as
`apps/workers/start-assistant`, Stage 3B must update
`docs/ops/REPO_STRUCTURE.md` and `docs/ops/COMPONENTS.yaml` before claiming that
path is an approved component.

## Local development assumptions

Local development should support:

- mock provider response for validation tests;
- mock KB metadata;
- local Worker dev server;
- no required real OpenAI call for ordinary unit tests;
- optional manual smoke with operator-provided secrets.

Local docs and test fixtures must not include real credentials.

## Failure modes

| Failure | Required behavior |
| --- | --- |
| Invalid method | Return safe `405` / JSON error |
| Non-JSON content | Return safe `415` / JSON error |
| Empty or oversized question | Return `invalid_request` |
| File upload attempt | Reject; do not parse as assistant input |
| KB manifest missing | Return `assistant_unavailable` |
| Vector store stale | Answer only if accepted by stale policy; include freshness metadata |
| OpenAI timeout | Return safe temporary unavailable response |
| Source grounding missing | Say the public KB does not answer it yet |
| User asks for repo scan | Refuse and suggest pilot fit check framing |
| User sends secret | Warn not to share secrets and recommend rotation |

## Stage 3B implementation checklist

Before implementing the live Worker:

- confirm physical Worker location;
- confirm route binding for `/api/start-chat`;
- confirm manifest storage: static Worker config or Cloudflare KV for v0;
- confirm direct OpenAI call vs Cloudflare AI Gateway;
- confirm Gateway payload logging posture if Gateway is used;
- confirm manual KB sync first and GitHub Action later;
- add component mapping before adding new runtime code;
- add route validation tests;
- add source-grounding tests with mocked provider responses;
- add refusal tests for repo scan, code execution, file upload, and secret input;
- add static browser fallback for Worker unavailable;
- run build/typecheck/tests for the touched app or Worker package;
- run static scans for secrets, env files, analytics, uploads, and backend scope
  drift.
