# Start Assistant Implementation Plan

## Purpose

This document defines the staged implementation plan for the Goalrail `/start` page and its source-grounded assistant.

The implementation must stay inside current Goalrail GTM boundaries:

- pilot-first;
- founder-led;
- not self-serve SaaS;
- not broad automation;
- not repo scanning from public page;
- not code execution;
- not a generic chat product.

## Target architecture

```text
Browser /start
  -> static guided UI
  -> optional POST /api/start-chat
      -> Cloudflare Worker
          -> approved public KB manifest
          -> OpenAI Responses API with file search
          -> answer + sources + suggested questions
```

Knowledge update path:

```text
GitHub main
  -> GitHub Action
      -> collect approved public docs
      -> build public knowledge files
      -> upload to OpenAI vector store
      -> write latest manifest with vector_store_id, commit SHA, timestamp
```

## Phase 0 - documentation only

Create docs:

- `docs/product/GOALRAIL_GLOBAL_START_ASSISTANT.md`
- `docs/ops/START_ASSISTANT_IMPLEMENTATION_PLAN.md`
- `docs/ops/START_ASSISTANT_KNOWLEDGE_SYNC.md`
- `docs/ops/START_ASSISTANT_SECURITY_AND_PRIVACY.md`
- `docs/ops/START_ASSISTANT_API_CONTRACT.md`
- `.goalrail/public-kb/manifest.yaml`

No runtime changes.
No secrets.
No external API calls.

## Phase 1 - static `/start` page

Goal: ship a useful global entry page without LLM risk.

Implement:

- `/start` route;
- hero;
- input-like field or disabled question input;
- quick question cards;
- static answer panel backed by local JSON or TypeScript data;
- artifact cards;
- soft CTA to pilot fit check or email / LinkedIn;
- no analytics;
- no backend requirement.

Static answers should come from:

- `docs/reference/start-assistant/quick-questions.json`
- `docs/reference/start-assistant/static-answers.md`

Quality checks:

- responsive layout;
- no false maturity claims;
- no repo-scan language;
- no code-execution language;
- no API keys;
- no analytics.

## Phase 2 - API shell

Goal: create endpoint boundary before connecting model.

Implement:

```text
POST /api/start-chat
```

Behavior:

- validates JSON body;
- accepts only `question` string;
- max question length e.g. 1000 chars;
- rejects file uploads and unknown content types;
- returns static mock answer;
- returns source placeholder;
- returns safety footer.

No OpenAI calls yet.

## Phase 3 - public KB build

Goal: compile approved public repo docs into knowledge artifacts.

Add script:

```text
scripts/build-public-kb.ts
```

Inputs:

```text
.goalrail/public-kb/manifest.yaml
```

Outputs:

```text
.goalrail/public-kb/dist/public-manifest.json
.goalrail/public-kb/dist/chunks.ndjson
```

Each chunk metadata:

```json
{
  "id": "...",
  "path": "docs/product/...",
  "title": "...",
  "heading": "...",
  "priority": "canon",
  "commit_sha": "...",
  "updated_at": "...",
  "public": true
}
```

Do not include private or unlisted files.

## Phase 4 - OpenAI file search integration

Goal: answer from public Goalrail knowledge.

Use:

- OpenAI Responses API;
- OpenAI hosted vector store / file search;
- model configured by environment variable.

Recommended environment variables:

```text
OPENAI_API_KEY
OPENAI_START_MODEL
OPENAI_START_VECTOR_STORE_ID
START_ASSISTANT_KB_REVISION
START_ASSISTANT_KB_UPDATED_AT
```

Do not hardcode model names.

`OPENAI_START_MODEL` may use a nano-class model such as `gpt-5.4-nano` if available in the account. If not available, use the current operator-approved low-latency OpenAI model.

## Phase 5 - GitHub Action knowledge sync

Goal: keep assistant knowledge aligned with repo source of truth.

On push to `main`, if public KB sources changed:

- build KB;
- upload files to OpenAI vector store;
- update manifest;
- store latest vector store ID / commit SHA / timestamp in deployment environment or Cloudflare KV/R2;
- optionally expire old vector stores.

Do not upload secrets or private data.

## Phase 6 - Cloudflare deployment

Recommended runtime:

- Cloudflare Pages or existing web deployment for `/start`;
- Cloudflare Worker for `/api/start-chat`;
- OpenAI key stored as Worker secret;
- optional Cloudflare KV/R2 for latest KB manifest;
- no direct browser call to OpenAI.

Optional later:

- Cloudflare AI Gateway with payload logging disabled;
- Cloudflare Vectorize custom RAG;
- Turnstile / WAF / Rate Limiting;
- Durable Object rate limiter.

## Non-goals

Do not implement in the first slice:

- repo upload;
- repo scan;
- GitHub OAuth;
- code execution;
- user accounts;
- CRM sync;
- analytics;
- autonomous qualification agent;
- paid checkout;
- model selector;
- chat history persistence.

## Review checklist

Before merging any implementation:

- no API key in browser bundle;
- no secret in repo;
- `/start` does not overclaim maturity;
- assistant says it cannot scan repos;
- no analytics added without decision;
- all public copy aligns with Goalrail product canon;
- model and vector store are configurable;
- failed assistant call has safe fallback;
- UI still works if assistant endpoint is unavailable.
