# Milestone 4: Non-ap-web Safe Docs Prose

Date: 2026-06-26

## Scope

Updated low-risk docs prose outside `ap-web` from Omnigent to Goalrail where the change does not alter runtime identifiers, command names, package names, external links, image names, environment variables, deployment resources, or compatibility contracts.

Changed surfaces:

- Agent onboarding assistant instructions and skill descriptions.
- Server API docs and Codex-specific API docs.
- Codex parity README prose.
- Issue triage design prose.
- E2E coverage TODO prose where only the product name changed.

## Intentionally Left For Later

The remaining docs-scope matches are intentional for this milestone:

- Legacy install / package surfaces: Homebrew tap and `omnigent` CLI command names.
- Deployment compatibility: GHCR image refs such as `ghcr.io/omnigent-ai/omnigent-*`.
- Bot identities and attribution: `omnigent <noreply@omnigent.ai>`.
- Historical GitHub links and issue/PR references under `omnigent-ai/omnigent`.
- SDK/package examples: `omnigent_client`, `OmnigentClient`, and repository links that still reflect package naming.
- CLI/runtime contract docs that describe current emitted text and modules, such as `designs/CLI_CONTRACT.md`.
- Error/class/type names such as `OmnigentError` and lower-case executor type `omnigent`.

These require a compatibility and migration plan before renaming.

## Verification

Commands:

```sh
codebase-memory-mcp cli index_status '{"project":"Users-vi-personal-heurema-goalrail"}'
codebase-memory-mcp cli search_code '{"project":"Users-vi-personal-heurema-goalrail","pattern":"Omnigent","limit":500}'
codebase-memory-mcp cli search_code '{"project":"Users-vi-personal-heurema-goalrail","pattern":"omnigent.ai","limit":500}'
git diff --check
git grep -n -I -E 'Omnigent|Omnigents|omnigent\.ai|omnigent-ai' -- '*.md' '*.mdx' '*.txt' '*.rst' '*.adoc' ':(exclude)ap-web/**' ':(exclude).heurema/**' ':(exclude).claude/**'
```

Results:

- Codebase-memory index was ready.
- `git diff --check` passed.
- Broad docs-scope residual search now shows only the intentional categories listed above.

## Next Milestone

Do not mechanically replace the remaining matches. Next work should be a compatibility plan for runtime/package/deploy names, including:

- CLI aliases and deprecation messaging.
- `OMNIGENT_*` environment variable migration.
- Python package/module rename strategy or explicit legacy policy.
- SDK package naming and import compatibility.
- Deployment image/repo naming strategy.
- Storage key and user-data path migration.
