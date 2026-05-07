---
id: goalrail_global_start_assistant
title: Goalrail Global Start Assistant
kind: product_canon
authority: canonical
status: draft
owner: product
truth_surfaces:
  - global_entry_page
  - public_assistant_boundary
  - knowledge_surface
lifecycle: incubating
review_after: 2026-06-07
supersedes: []
superseded_by: null
related_docs:
  - docs/INDEX.md
  - docs/product/GOALRAIL_PRODUCT_CONCEPT.md
  - docs/product/GOALRAIL_GTM_MODEL.md
  - docs/product/GOALRAIL_OFFER.md
  - docs/product/GOALRAIL_PUBLIC_NARRATIVE.md
  - docs/product/GOALRAIL_PUBLIC_LANGUAGE.md
  - docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md
---
# Goalrail Global Start Assistant

## 1. Purpose

`/start` is the English-first global entry surface for Goalrail.

It receives traffic from:

- LinkedIn posts and comments;
- GitHub README and public docs;
- Dev.to / Habr / Reddit / Hacker News / X when used later;
- direct referrals from technical and business conversations.

The page must help a new reader understand Goalrail without forcing them through a long landing page.

The page should answer:

```text
What is Goalrail?
Why does it matter now?
What can I ask about?
What is the next safe step?
```

## 2. Product boundary

`/start` is not a self-serve SaaS onboarding page.

It is not a repo scanner.
It is not a code execution surface.
It is not a chat support bot with unlimited product claims.
It is not a broad AI agent interface.

It is a guided, source-grounded entry page over approved public Goalrail knowledge.

## 3. Core positioning

Hero line:

```text
Ask Goalrail about AI-assisted delivery.
```

Supporting line:

```text
From business goal to verified code change.
```

Short explanation:

```text
Goalrail is a control layer for teams using AI coding tools and trying not to lose intent, scope, checks, proof, and approval.
```

## 4. Page concept

The page should be a hybrid of:

- short positioning;
- input box;
- guided question cards;
- source-grounded answer panel;
- artifact cards;
- soft pilot fit check CTA.

It should not start as a long text landing page.

The cold reader may not know what to ask, so the page must guide them with suggested questions.

## 5. Page structure

### 5.1 Hero

```text
Ask Goalrail about AI-assisted delivery.

From business goal to verified code change.
```

Input placeholder:

```text
Ask about repo readiness, contracts, proof, approval, or AI delivery drift...
```

Primary action:

```text
Ask
```

Secondary links:

```text
View artifacts
View GitHub
Request a pilot fit check
```

### 5.2 Suggested questions

Initial question cards:

- What is Goalrail?
- Is my repo ready for coding agents?
- What is contract-first execution?
- What does proof before approval mean?
- How is Goalrail different from an AI IDE?
- What would a pilot fit check look like?
- What is AI delivery drift?
- How should a team review AI-generated changes?

Optional business questions:

- Why should a CTO care?
- Where does AI coding create hidden risk?
- How can a team adopt AI without losing control?

### 5.3 Answer panel

The answer panel must show:

- short answer;
- source list;
- suggested follow-up questions;
- knowledge timestamp / revision when available.

Example footer:

```text
Answers use public Goalrail materials. This page cannot scan repos or execute code.
```

### 5.4 Artifact cards

Cards should link to public artifacts when available:

- Contract-first execution;
- Proof before approval;
- Repo readiness;
- AI delivery drift;
- Approval gate;
- Goal on Rails series.

### 5.5 Soft CTA

CTA copy:

```text
Have a real workflow where AI is making your team faster but harder to control?
```

Primary CTA:

```text
Request a pilot fit check
```

Microcopy:

```text
Best fit: one team, one repo or workflow, one visible task-to-proof loop.
```

## 6. Assistant behavior

The assistant must be source-grounded.

It must answer only from approved public Goalrail knowledge sources.

If the answer is not in the sources, it must say that the public knowledge base does not answer it yet.

It must not invent:

- product maturity;
- repo scanning;
- code execution;
- autonomous delivery;
- supported integrations;
- customer results;
- pricing details not in current canon;
- guarantees.

Default answer style:

- short;
- concrete;
- no hype;
- no hard sell;
- use Goalrail terms carefully;
- end with one useful next question or link.

## 7. Public knowledge sources

Only explicit public sources should be indexed.

Allowed source classes:

- canonical product docs approved for public explanation;
- public narrative docs;
- public language docs;
- README;
- public brand docs when relevant;
- published public posts or manually approved post copies;
- public demo notes or artifact descriptions.

Forbidden source classes:

- secrets;
- credentials;
- raw private chats;
- raw call transcripts;
- client data;
- private commercial notes;
- unreviewed personal notes;
- anything not intended for public readers.

## 8. Implementation posture

Recommended sequence:

1. `v0.1` static guided page with hardcoded question cards and static answers.
2. `v0.2` source-grounded assistant through Cloudflare Worker and OpenAI Responses API file search.
3. `v0.3` custom RAG with Cloudflare Vectorize / R2 / D1 only if hosted file search becomes limiting.

Do not start with a direct browser call to OpenAI.

Do not expose OpenAI keys to the browser.

## 9. Safety boundaries

The `/start` assistant must include clear copy:

```text
This assistant answers from public Goalrail materials.
It cannot scan your repo from this page.
It does not execute code.
Do not paste secrets, private code, or customer data.
```

Hard refusals / redirects:

- private repo scan request;
- code execution request;
- file upload request;
- request involving secrets;
- demand for safety guarantee;
- request to replace developers;
- request to claim full autonomous delivery.

Safe redirect example:

```text
I cannot scan a repo from this page. For a pilot fit check, describe your team size, AI tools, repo or workflow shape, and one review or proof problem you are seeing.
```

## 10. Success criteria

The `/start` surface is useful if it produces:

- profile traffic that can understand Goalrail in under two minutes;
- fewer repeated explanation conversations;
- useful questions from visitors;
- qualified pilot-fit conversations;
- safe, source-grounded answers that do not overclaim.

Initial metrics can be manual:

- number of visits if logs are allowed later;
- number of assistant questions;
- number of pilot fit check clicks;
- number of inbound DMs mentioning `/start`;
- quality of questions asked.

No analytics or tracking should be added without a separate explicit decision.
