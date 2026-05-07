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

Deployment config:

- `wrangler.toml` defines the separate `goalrail-start-assistant` Worker
  package.
- Public route ownership remains deployment-managed outside this repository.
- Live deploy requires Cloudflare auth and Worker-side OpenAI/vector-store
  configuration.

Manual public KB commands:

```bash
node scripts/start-assistant/build-public-kb.mjs
node scripts/start-assistant/upload-public-kb-openai.mjs
node scripts/start-assistant/upload-public-kb-openai.mjs --execute
```

The upload command is dry-run by default. `--execute` requires `OPENAI_API_KEY`,
uploads only the generated public KB document, creates a new OpenAI vector
store, attaches the file through a vector-store file batch, and writes an
ignored runtime manifest under `.goalrail/public-kb/dist/`.

Local commands:

```bash
npm --prefix apps/workers/start-assistant test
npm --prefix apps/workers/start-assistant run dev
npm --prefix apps/workers/start-assistant run deploy:dry-run
```

The local dev server defaults to `http://127.0.0.1:8787` and enters mock mode
when OpenAI provider configuration is absent.
