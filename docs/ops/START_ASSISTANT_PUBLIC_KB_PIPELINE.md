# Start Assistant Public KB Pipeline

## Purpose

This document defines the public knowledge pipeline for the live `/start`
assistant.

The pipeline must keep GitHub as the source of truth. OpenAI file_search,
Cloudflare Vectorize, or any other retrieval index is only a compiled artifact.

## Source of truth

The canonical source is the Goalrail GitHub repository at a specific commit.

The assistant must never treat a vector store, uploaded file set, generated
chunk file, or local machine directory as canonical truth.

Every built KB artifact should be traceable to:

- repository name;
- commit SHA;
- build timestamp;
- explicit whitelist manifest revision;
- source file path;
- heading path or chunk identifier.

## Whitelist model

Use an explicit whitelist only.

Do not index the whole repository.
Do not crawl local files.
Do not infer public safety from file extension or directory name alone.

The committed source manifest is:

```text
.goalrail/public-kb/manifest.yaml
```

The manifest is the input allowlist. Runtime deployment metadata may copy or
compile it, but must not silently widen it.

## Allowed source classes

Allowed when explicitly whitelisted:

- selected public/canonical `docs/product/*` docs;
- selected `docs/reference/start-assistant/*` public assistant reference files;
- selected `docs/brand/*` style and narrative docs that are safe for public
  readers;
- selected `docs/research/*` public research notes;
- `README.md`;
- selected public post artifacts when explicitly whitelisted;
- selected artifact descriptions that do not expose private customer data.

## Disallowed source classes

Never include:

- private chats;
- raw transcripts;
- client data;
- secrets;
- `.env` files;
- credentials;
- unpublished sales notes;
- local machine paths;
- private customer information;
- browser sessions;
- platform account cache;
- generated logs with prompts or personal data;
- unreviewed local working drafts.

## Manifest shape

The source manifest should remain simple and reviewable.

Recommended source row:

```yaml
sources:
  - path: docs/product/GOALRAIL_GLOBAL_START_ASSISTANT.md
    title: Goalrail Global Start Assistant
    priority: canon
    public: true
    include_headings: all
```

Compiled runtime manifest shape:

```json
{
  "project": "goalrail",
  "repository": "heurema/goalrail",
  "commit_sha": "abc123",
  "updated_at": "2026-05-07T12:00:00Z",
  "source_manifest_sha": "def456",
  "sources_count": 12,
  "chunks_count": 96,
  "retrieval": {
    "provider": "openai",
    "kind": "file_search",
    "index_id": "stored-server-side"
  }
}
```

Only public freshness metadata should be returned to the browser. Provider index
identifiers may stay server-side.

## Build process

Recommended build steps:

1. read `.goalrail/public-kb/manifest.yaml`;
2. verify every listed file exists in the Git checkout;
3. reject any non-whitelisted path;
4. read Markdown with a structured parser or a simple heading-aware parser;
5. split content by heading path;
6. preserve title, path, heading, priority, and commit SHA in metadata;
7. write generated chunks;
8. scan generated chunks for disallowed path classes and obvious secret markers;
9. upload only the generated public KB artifact to the retrieval provider;
10. write runtime manifest with commit SHA, updated timestamp, source manifest
    revision, chunk count, and retrieval index pointer.

Recommended generated files:

```text
.goalrail/public-kb/dist/public-manifest.json
.goalrail/public-kb/dist/chunks.ndjson
.goalrail/public-kb/dist/public-kb.md
```

Generated `dist/` artifacts are deployment-only and ignored by git. They must
not become the source of truth.

Current manual build command:

```bash
node scripts/start-assistant/build-public-kb.mjs
```

The build command reads only the committed whitelist manifest, skips missing
optional sources, splits Markdown by headings, writes the compiled manifest,
NDJSON chunks, and upload-ready public Markdown document, and scans generated
content for obvious secret markers. The current `/start` public KB is
English-only: each source row must declare `language: en`, and the build fails
if Cyrillic text is present in source or generated retrieval artifacts.

Current v0 source classes are intentionally narrow:

- English `/start` product boundary;
- English provider/product boundary material;
- English static answer seeds;
- English quick-question reference data.

## Chunk policy

Default:

- split by Markdown headings;
- target medium chunks that preserve local context;
- keep source title and heading path in metadata;
- keep exact wording for canonical definitions;
- do not merge unrelated documents into one chunk;
- prefer canonical product docs over research when sources conflict.

Example chunk:

```json
{
  "id": "docs-product-global-start-assistant-purpose",
  "path": "docs/product/GOALRAIL_GLOBAL_START_ASSISTANT.md",
  "title": "Goalrail Global Start Assistant",
  "heading": "Purpose",
  "priority": "canon",
  "commit_sha": "abc123",
  "updated_at": "2026-05-07T12:00:00Z",
  "public": true,
  "text": "..."
}
```

## Vector store lifecycle

Recommended v0 lifecycle:

- manual build first;
- create a new OpenAI vector store for each successful public KB build;
- publish the new runtime manifest only after upload succeeds;
- keep the previous working vector store identifier for rollback;
- expire old vector stores after a short retention window once the new one is
  verified.

Avoid updating a shared vector store in place for the first slice. A new index
per build gives a cleaner rollback path.

Later, if provider cost or lifecycle limits require it, the pipeline can switch
to an update-in-place strategy with stronger validation.

## Runtime manifest storage

Stage 3B acceptable options:

1. Static Worker config for the first manual smoke.
2. Cloudflare KV for the latest manifest pointer.
3. R2 for larger generated artifacts if the manifest grows.

Default v0 recommendation:

```text
manual sync -> static Worker config or Worker secret/config variable
```

Move to KV when the sync path becomes automated.

Do not expose the raw runtime manifest as a public browser URL unless it has
been reviewed as public-safe.

## Stale KB behavior

The Worker should know:

- `commit_sha`;
- `updated_at`;
- source manifest revision.

If the KB is stale but still available:

- answer only from that KB;
- return `knowledge.updated_at` and `knowledge.commit_sha`;
- do not imply latest repo knowledge.

If the KB is missing or retrieval fails:

- return `assistant_unavailable`;
- keep static `/start` content available;
- do not call a model without retrieval unless a later decision explicitly
  approves non-retrieval fallback.

## Rollback approach

Rollback should be pointer-based:

1. keep previous known-good runtime manifest;
2. point the Worker back to the previous retrieval index;
3. verify `/api/start-chat` smoke questions;
4. expire the failed new index after diagnosis.

Do not roll back by editing public source docs to match a broken retrieval
artifact.

## Manual sync first

The first live slice should use a manual sync script or operator-run command.

Why:

- easier to inspect the generated chunks;
- easier to catch public-boundary mistakes before automated upload;
- avoids hiding provider and manifest lifecycle decisions inside CI too early;
- lets Stage 3B test answer quality before hardening automation.

GitHub Action sync can follow after the manual flow is proven.

Current manual upload command:

```bash
node scripts/start-assistant/upload-public-kb-openai.mjs
```

The upload command is a dry run by default. With `--execute` and
`OPENAI_API_KEY` set in the operator environment, it uploads only the generated
public KB Markdown file, creates a new OpenAI vector store, attaches the file
through a vector-store file batch, waits for ingestion, and writes an ignored
runtime manifest under `.goalrail/public-kb/dist/`.

The script does not deploy the Worker and does not write secrets or provider
configuration into the repository.

## Optional future Cloudflare Vectorize path

If OpenAI hosted file_search is too limiting, a later architecture may use:

```text
GitHub Action
  -> build chunks
  -> create embeddings
  -> upsert vectors to Cloudflare Vectorize
  -> store chunk text in R2 or D1

Worker
  -> embed query
  -> query Vectorize
  -> fetch chunk text
  -> call OpenAI with retrieved context
```

This is not Stage 3B. It introduces more moving parts and should follow a
separate decision.

## Review checklist

Before enabling live KB sync:

- manifest whitelist reviewed manually;
- generated chunks reviewed for public safety;
- no secrets or private paths in generated chunks;
- provider upload uses only generated public KB files;
- old indexes are tracked and expireable;
- Worker response includes freshness metadata;
- failure path does not fall back to unguided model answers;
- no private customer data enters the public assistant.
