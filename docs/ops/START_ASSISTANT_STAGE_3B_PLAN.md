# Start Assistant Stage 3B Plan

## Purpose

This document defines the smallest implementation plan for the first live
source-grounded `/start` assistant.

Stage 3B must not begin until the Stage 3A architecture boundary is accepted.

## Preconditions

Resolve these before code:

1. Physical Worker location.
2. Runtime manifest storage: static Worker config, Cloudflare KV, R2, or another
   approved deployment-managed source.
3. OpenAI access path: direct Worker to OpenAI or Cloudflare AI Gateway.
4. Knowledge sync path: manual script first or GitHub Action.

Default recommendation:

```text
Worker: separate Cloudflare Worker
Model path: Worker -> OpenAI directly
Retrieval: OpenAI Responses API with file_search
KB manifest: static Worker config for first manual smoke, then Cloudflare KV
Sync: manual script first, GitHub Action second
```

## Current Stage 2 trade-off

The static Stage 2 implementation sets `/start` title, description, and Open
Graph metadata in `apps/web/console/index.html`.

Because `apps/web/console` is a Vite SPA, that HTML metadata is global for the
app shell before React route-specific client code runs. This means the console
root can initially receive the `/start`-oriented title/description.

Accepted v0 trade-off:

- `/start` is the current SEO/social entry surface;
- the authenticated console is not being treated as the public SEO/social
  surface in this slice;
- `StartPage.tsx` still sets route-specific metadata client-side;
- a later deployment or routing slice can move to neutral app-shell metadata,
  SSR/static per-route metadata, or a dedicated public entry surface if needed.

## Non-goals

Do not implement in Stage 3B:

- repo scan;
- GitHub OAuth;
- user file uploads;
- code execution;
- user accounts;
- sessions;
- CRM;
- analytics;
- tracking;
- chat history persistence;
- model selector;
- generic agent platform behavior;
- product workflow automation;
- changes to the core API app unless route wiring requires a documented proxy.

## Expected implementation shape

Preferred route:

```text
goalrail.dev/api/start-chat -> separate public-edge Worker
```

Recommended code location if a new Worker package is added:

```text
apps/workers/start-assistant/
```

Before adding that path, update:

- `docs/ops/REPO_STRUCTURE.md`;
- `docs/ops/COMPONENTS.yaml`;
- any package/workspace metadata required by the chosen Worker stack.

Do not add implementation under `apps/server` for the first assistant slice.

## Endpoint

```text
POST /api/start-chat
```

Request:

```json
{
  "question": "What is proof before approval?"
}
```

Response:

```json
{
  "answer": "Proof before approval means...",
  "sources": [
    {
      "title": "Goalrail Global Start Assistant",
      "path": "docs/product/GOALRAIL_GLOBAL_START_ASSISTANT.md",
      "section": "Assistant behavior"
    }
  ],
  "suggested_questions": [
    "What is contract-first execution?",
    "How should a team review AI-generated changes?"
  ],
  "knowledge": {
    "updated_at": "2026-05-07T12:00:00Z",
    "commit_sha": "abc123"
  },
  "disclaimer": "Answers use public Goalrail materials. This page cannot scan repos or execute code."
}
```

## Implementation steps

1. Add the Worker package or chosen public-edge Worker implementation path.
2. Register the component and path in repo docs before claiming implementation.
3. Implement request validation:
   - `POST` only;
   - `application/json` only;
   - `question` string only;
   - trimmed length between 1 and 1000 characters;
   - reject multipart, binary, and unknown large payloads.
4. Add a provider adapter with mocked tests first.
5. Add OpenAI Responses API call with file_search.
6. Load KB manifest from approved runtime config.
7. Shape response to `START_ASSISTANT_API_CONTRACT.md`.
8. Add safe refusals for repo scan, code execution, file upload, private code,
   and secrets.
9. Update `/start` UI to enable the input only when the Worker path is wired.
10. Keep static quick questions as fallback.
11. Add local and deployment smoke checks.

## Tests

Required tests:

- valid question returns shaped answer with sources;
- empty question rejected;
- over-limit question rejected;
- `GET` rejected;
- non-JSON rejected;
- multipart upload rejected;
- provider timeout returns `assistant_unavailable`;
- missing KB manifest returns `assistant_unavailable`;
- repo scan prompt refuses safely;
- code execution prompt refuses safely;
- secret-sharing prompt refuses safely;
- response does not claim broad platform maturity;
- response does not claim repo scanning or code execution.

Use mocked provider responses for automated tests. Do not require live OpenAI
credentials in normal test runs.

## Smoke checks

Manual local smoke:

```text
POST /api/start-chat
question: What is Goalrail?
expect: short answer, source references, suggested questions, disclaimer
```

Negative smoke:

```text
question: Can you scan my private repo?
expect: refusal; no repo-scan claim
```

```text
question: Run this code.
expect: refusal; no execution claim
```

```text
question: Here is my API key...
expect: warning not to share secrets
```

Browser smoke:

- `/start` loads without auth;
- input no longer looks disabled only after Worker is connected;
- answer panel shows sources;
- static fallback remains visible;
- mobile has no horizontal overflow.

## Security validation

Before Stage 3B completion:

- static scan for committed `.env` files and secrets;
- static scan for browser OpenAI API usage;
- static scan for file upload UI;
- static scan for analytics/tracking;
- verify no `apps/server` route was added unless separately approved;
- verify no repo scan or code execution code path exists;
- verify provider key only exists as a deployment secret;
- verify provider payload logging posture if Cloudflare AI Gateway is used.

Example local validation:

```bash
rg -n "OPENAI_API_KEY|api/start-chat|file upload|type=\"file\"|analytics|gtag|document.cookie" apps docs
```

Review any matches manually. Some docs may mention forbidden capabilities as
non-goals; runtime files must not implement them unless the stage explicitly
allows it.

## Manual acceptance criteria

Stage 3B is complete only if:

- `/start` can ask one public question through `/api/start-chat`;
- the answer cites approved public Goalrail sources;
- the answer includes knowledge freshness metadata;
- static fallback still works when the Worker is unavailable;
- no browser secret is present;
- no repo scan, upload, code execution, analytics, sessions, or CRM path exists;
- tests and build/typecheck pass for touched packages;
- docs state the implementation owner and remaining limitations.

## Recommended Stage 3B prompt boundary

```text
Implement the minimal live `/start` assistant Worker only.

Use a separate public-edge Worker path for `POST /api/start-chat`.
Do not modify the core API app unless explicitly required for route proxying and
documented.
Use mocked provider tests first.
Use OpenAI Responses API with file_search only after request validation and
public KB manifest loading exist.
Do not add repo scan, uploads, code execution, analytics, cookies, sessions, CRM,
or chat history.
Keep static quick questions as fallback.
```
