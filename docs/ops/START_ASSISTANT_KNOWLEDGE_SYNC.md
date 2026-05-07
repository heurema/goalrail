# Start Assistant Public Knowledge Sync

## Purpose

The `/start` assistant must answer from current public Goalrail knowledge.

GitHub remains the source of truth.
The vector store or retrieval index is only a compiled search artifact.

## Source policy

Use explicit whitelist only.
Do not index the whole repository.

Approved sources are listed in:

```text
.goalrail/public-kb/manifest.yaml
```

The manifest should only include public-safe docs.

## Recommended source classes

Allowed:

- product concept and operating model docs;
- public narrative docs;
- public language docs;
- offer and GTM docs where safe for public explanation;
- README;
- selected brand docs;
- selected published public posts;
- selected artifact descriptions.

Not allowed:

- credentials;
- API keys;
- private chat transcripts;
- client notes;
- private sales notes;
- unpublished customer data;
- raw meeting transcripts;
- personal working memory;
- anything not intended for public readers.

## Build output

The build script should produce:

```text
.goalrail/public-kb/dist/public-manifest.json
.goalrail/public-kb/dist/chunks.ndjson
```

`public-manifest.json` example:

```json
{
  "project": "goalrail",
  "commit_sha": "abc123",
  "updated_at": "2026-05-07T12:00:00Z",
  "sources_count": 12,
  "chunks_count": 96,
  "vector_store_id": "vs_optional_after_upload"
}
```

`chunks.ndjson` example row:

```json
{"id":"docs-product-offer-what-we-sell-now","path":"docs/product/GOALRAIL_OFFER.md","title":"Goalrail Offer","heading":"What we sell now","priority":"offer","text":"..."}
```

## Chunking policy

Recommended default:

- split by Markdown headings;
- target 500-900 tokens per chunk;
- keep heading path in metadata;
- include document title and source priority;
- preserve exact wording of core definitions;
- do not merge unrelated docs into one chunk.

## Source priority

Recommended priorities:

```text
canon
operating-model
offer
public-narrative
public-language
brand
research
published-post
artifact
reference
```

The assistant should prefer canonical product docs over advisory docs when sources conflict.

## Update model

### MVP update

On push to `main`:

1. build public KB;
2. upload to OpenAI vector store;
3. store latest vector store ID and source revision in runtime config;
4. optionally expire previous vector store.

### Later update

If hosted file search is limiting, move to custom RAG:

```text
GitHub Action
  -> build chunks
  -> create embeddings
  -> upsert to Cloudflare Vectorize
  -> store chunk text in R2 or D1

Worker
  -> embed query
  -> query Vectorize
  -> fetch chunk text
  -> call OpenAI with retrieved context
```

## Public page metadata

The `/start` page can show:

```text
Knowledge updated from GitHub: <date>
Source revision: <short SHA>
```

This makes freshness visible and prevents the assistant from pretending to know more than the indexed knowledge.

## Failure behavior

If the latest vector store is unavailable:

- return a safe fallback answer;
- show that the assistant is temporarily unavailable;
- keep static question cards available;
- do not call a model with no retrieval unless explicitly configured as fallback.

Example fallback:

```text
The public knowledge assistant is temporarily unavailable. You can still view the static Goalrail overview and artifacts.
```

## Security review

Before enabling sync:

- verify whitelist manually;
- verify generated chunks do not contain secrets;
- verify OpenAI upload step only uses generated public KB files;
- verify old indexes are expired or tracked;
- verify no private data enters the public assistant.
