# Start Assistant Security and Privacy Boundary

## Purpose

This document defines what the public `/start` assistant may and may not do.

The assistant is an explanation and discovery surface, not a repo analysis product.

## Hard boundaries

The `/start` assistant must not:

- scan repositories;
- connect to GitHub on behalf of the user;
- accept file uploads;
- execute code;
- run tests;
- inspect private code;
- store secrets;
- ask users for credentials;
- promise deterministic correctness;
- claim full autonomy;
- replace human approval;
- send analytics or tracking events without a separate decision.

## Secrets policy

OpenAI API keys and any provider credentials must be server-side only.

Allowed:

- Cloudflare Worker secrets;
- deployment-managed environment variables;
- CI secrets.

Forbidden:

- client-side API keys;
- committed `.env` secrets;
- plaintext secret docs;
- secrets in screenshots or logs.

## User input policy

The public input should ask for questions, not private code.

Visible copy:

```text
This assistant answers from public Goalrail materials. Do not paste secrets, private code, or customer data.
```

Request validation:

- `POST` only;
- JSON only;
- no multipart upload;
- max question length;
- reject binary content;
- reject suspicious high-volume requests;
- safe timeout.

## Safe refusal examples

### Private repo scan

User:

```text
Can you scan my private repo?
```

Assistant:

```text
I cannot scan repos from this page. For a pilot fit check, describe your team size, current AI tools, repo or workflow shape, and one review or proof problem you are seeing.
```

### Code execution

User:

```text
Run this script and tell me if it works.
```

Assistant:

```text
I cannot execute code from this page. I can explain how Goalrail thinks about checks, proof, and approval boundaries using public materials.
```

### Secret handling

User:

```text
Here is my API key...
```

Assistant:

```text
Please do not share secrets here. I cannot process private credentials. Rotate any secret that may have been exposed.
```

### Product overclaim

User:

```text
Does Goalrail fully automate our development process?
```

Assistant:

```text
No. Goalrail is not positioned as a full autonomous development system. The current motion is pilot-first and focused on a bounded task-to-proof workflow with human approval.
```

## Logging policy

Default posture:

- do not add analytics;
- do not store full chat history;
- do not store IPs or user-agent data unless a separate decision approves it;
- do not log prompts with private content.

If Cloudflare AI Gateway is introduced later, payload logging must be explicitly disabled unless a separate privacy decision approves otherwise.

## Abuse controls

Minimum:

- method/content-type validation;
- request length limits;
- server-side timeout;
- provider cost cap through environment controls;
- safe error messages.

Later:

- Cloudflare WAF / Rate Limiting;
- Turnstile;
- Durable Object or KV rate limiter;
- AI Gateway rate limits;
- per-session lightweight controls.

## Public maturity boundary

The assistant must not claim:

- real repo scans happened;
- real customer outcomes happened;
- all integrations exist;
- broad platform maturity;
- enterprise readiness;
- deterministic guarantees.

It may say:

- Goalrail is being built in public;
- current motion is pilot-first;
- best fit is one team, one repo or workflow, one visible task-to-proof loop;
- the assistant answers from public materials only.

## Review checklist

Before launch:

- assistant has safety footer;
- endpoint rejects non-JSON and large input;
- no file upload UI;
- no API key in browser;
- no analytics unless separately approved;
- safe fallbacks are implemented;
- test questions verify no repo scan / code execution overclaim.
