# Goalrail Start Assistant Worker

Minimal public-edge Worker for the live `/start` assistant.

Current scope:

- owns `POST /api/start-chat`;
- validates JSON-only short public questions;
- refuses repo scan, code execution, file upload, private code, and secret
  prompts before provider calls;
- calls OpenAI Responses API with `file_search` when provider configuration is
  present;
- supports a local/test mock provider mode;
- returns answer, sources, suggested questions, knowledge freshness metadata,
  and safety disclaimer;
- does not touch `apps/server`;
- does not add repo scan, code execution, uploads, analytics, cookies, sessions,
  CRM, chat history, or tracking.

Runtime configuration names:

- `OPENAI_API_KEY`
- `OPENAI_START_MODEL`
- `OPENAI_START_VECTOR_STORE_ID`
- `START_ASSISTANT_KB_REVISION`
- `START_ASSISTANT_KB_UPDATED_AT`
- `START_ASSISTANT_PROVIDER_MODE` set to `mock` only for local/test smoke

Do not commit real values for these variables.

Local commands:

```bash
npm --prefix apps/workers/start-assistant test
npm --prefix apps/workers/start-assistant run dev
```

The local dev server defaults to `http://127.0.0.1:8787` and enters mock mode
when OpenAI provider configuration is absent.
