# Publishing

Repo-tracked public narrative and go-to-market working artifacts owned by the Punk publishing layer for Goalrail.

This is the publication plane, not a frontend static-assets `public/` directory.

This layer exists before any automation or publishing workflow.
Humans update it manually until a real operating surface exists.

## Layout

- `.punk/publishing/stories/` — durable narrative arcs
- `.punk/publishing/posts/` — concrete draft/final post assets
- `.punk/publishing/channels/` — channel notes and constraints
- `.punk/publishing/publications/` — publication receipts
- `.punk/publishing/metrics/` — manual performance snapshots
- `.punk/publishing/_templates/` — starter templates
- `.punk/publishing/_schema/` — future metadata/schema helpers

Public material must not outrun the canonical product docs.

## Manual flow

Until PubPunk or another publishing runtime is active, use this manual order:

1. Create or update a durable narrative arc in `stories/`.
2. Create the platform-specific draft in `posts/`.
3. Add one row to `posts/LEDGER.md`.
4. After manual publication, fill a receipt in `publications/`.
5. Add D+2 / D+7 metric snapshots in `metrics/`.

Drafts and receipts are repo-tracked public narrative artifacts, not proof that
automated publishing exists.
