# Start Assistant API Contract

## Endpoint

```text
POST /api/start-chat
```

## Purpose

Answer short public questions about Goalrail using approved public knowledge sources.

## Request

```json
{
  "question": "What is contract-first execution?"
}
```

Optional later:

```json
{
  "question": "What is proof before approval?",
  "client_context": {
    "source": "start-page",
    "selected_question_id": "proof-before-approval"
  }
}
```

Do not accept:

- file uploads;
- repo URLs for scanning;
- private code;
- credentials;
- arbitrary tool commands.

## Validation

- method must be `POST`;
- content type must be `application/json`;
- `question` must be string;
- trimmed question length must be 1-1000 characters;
- reject unknown large payloads;
- return safe errors.

## Response

```json
{
  "answer": "Contract-first execution means...",
  "sources": [
    {
      "title": "Goalrail Operating Model",
      "path": "docs/product/GOALRAIL_OPERATING_MODEL.md",
      "section": "Contract-first execution"
    }
  ],
  "suggested_questions": [
    "What is proof before approval?",
    "How is Goalrail different from an AI IDE?"
  ],
  "knowledge": {
    "updated_at": "2026-05-07T12:00:00Z",
    "commit_sha": "abc123"
  },
  "disclaimer": "Answers use public Goalrail materials. This page cannot scan repos or execute code."
}
```

## Error response

```json
{
  "error": "invalid_request",
  "message": "Question must be a non-empty string under 1000 characters."
}
```

Temporary unavailable:

```json
{
  "error": "assistant_unavailable",
  "message": "The public Goalrail assistant is temporarily unavailable. Static overview and artifacts are still available."
}
```

## System instruction draft

```text
You are the public Goalrail /start assistant.

Answer only from approved public Goalrail knowledge sources.
If the sources do not answer the question, say that the public knowledge base does not answer it yet.
Do not invent product maturity.
Do not claim repo scanning, code execution, autonomous delivery, customer results, or integrations unless sources explicitly confirm them.
Keep answers short and concrete.
Prefer terms from Goalrail canon: goal intake, clarification, contract, bounded execution, checks, proof, approval, repo readiness, AI delivery drift.
Do not hard sell. Do not ask the user to book a demo.
When useful, suggest one next question or one public artifact.
```

## Source formatting

Sources should be displayed as public knowledge references, not as legal citations.

Example:

```text
Sources:
- Goalrail Offer
- Goalrail Public Narrative
```

## Suggested questions policy

The endpoint should return 2-3 follow-up questions.

Examples:

- What is repo readiness?
- What does proof before approval mean?
- What does a pilot fit check include?
- How is Goalrail different from an AI IDE?

## Runtime configuration

Required:

```text
OPENAI_API_KEY
OPENAI_START_MODEL
OPENAI_START_VECTOR_STORE_ID
```

Recommended:

```text
START_ASSISTANT_KB_REVISION
START_ASSISTANT_KB_UPDATED_AT
START_ASSISTANT_ALLOWED_ORIGINS
```

Do not hardcode secrets or model names.

## Initial model posture

Use a configurable low-cost, low-latency OpenAI model for public Q&A.

If `gpt-5.4-nano` is available in the operator account, it may be used.
If not available, use the current operator-approved nano-class or low-latency model.

Do not rely on model memory. Always ground answers in retrieval.

## Non-goals

The endpoint must not:

- maintain user accounts;
- store chat history;
- send CRM events;
- run analytics;
- process code;
- connect to repo providers;
- support model selection by public users.
