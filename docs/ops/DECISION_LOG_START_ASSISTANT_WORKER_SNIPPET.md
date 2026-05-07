# Decision Log Snippet - Start Assistant Worker

This snippet was applied to `docs/ops/DECISIONS.md` as D-0090 after the Stage 3A
architecture was accepted.

## D-0090 - Start assistant live path uses a separate public-edge Worker

Date: 2026-05-07
Status: accepted

Decision:

- The live `/start` assistant will use a separate public-edge assistant Worker
  for `POST /api/start-chat`.
- The first live assistant slice will not run inside the core Goalrail API app.
- Browser code will not call OpenAI directly.
- The Worker will answer only from an approved public KB retrieval index built
  from an explicit GitHub whitelist.
- GitHub remains the source of truth; the retrieval index is a compiled
  artifact.
- The first KB sync path should be manual, with GitHub Action sync added only
  after chunk quality, source boundaries, and rollback are proven.
- The first retrieval path is OpenAI Responses API with file_search.
- Cloudflare AI Gateway may be introduced later only with payload logging
  posture explicitly decided.
- The assistant must not scan repositories, execute code, accept file uploads,
  ingest private code, store chat history, add analytics/tracking, or imply broad
  product maturity.

Rationale:

- Anonymous public assistant traffic has different abuse, privacy, and cost
  boundaries than the authenticated product API.
- A separate Worker keeps provider secrets server-side while isolating public
  Q&A from canonical Goalrail state.
- Manual KB sync first gives the team a reviewable path for public-source
  quality before automating provider uploads.
