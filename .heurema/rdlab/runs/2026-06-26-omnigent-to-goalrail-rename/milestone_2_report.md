# Milestone 2: ap-web Visible Goalrail Labels

Date: 2026-06-26

## Scope

Updated user-visible `ap-web` labels from Omnigent/Omnigents to Goalrail where the change is branding-only and does not alter runtime identifiers, API contracts, storage keys, packages, or build paths.

Changed surfaces:

- Browser document titles.
- Electron setup page title, alt text, and server setup copy.
- Login, registration, settings, sidebar, title bar, and icon accessibility labels.
- Test fixtures that assert the visible labels or public docs URL.

## Intentionally Left For Later

Internal runtime identifiers and compatibility-bearing names remain unchanged, including:

- `OMNIGENT_*` environment variables.
- `omnigent:*` localStorage keys.
- `omnigent/server/static/web-ui` build output path.
- `OmnigentApp`, `OmnigentError`, SDK/REPL comments, package names, and module identifiers.

These require a compatibility plan and should be handled in a later milestone.

## Verification

Used the bundled Codex Node runtime because system Node `v25.9.0` exposes a broken global `localStorage` object that makes existing Vitest suites fail before assertions run.

Commands:

```sh
PATH="/Users/vi/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin:$PATH" npm --prefix ap-web test -- src/components/OttoEyes.test.tsx src/components/icons/OttoIcon.test.tsx src/shell/Sidebar.test.tsx src/shell/settingsNav.test.tsx src/shell/MarkdownRichTextViewer.test.tsx src/shell/TipTapHtmlPassthrough.test.ts
PATH="/Users/vi/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin:$PATH" npm --prefix ap-web run build
git diff --check
git grep -InE 'omnigent\.ai|omnigent-ai' -- ap-web/src ap-web/index.html ap-web/electron/setup ap-web/electron/find
```

Results:

- 6 targeted test files passed, 51 tests passed.
- `ap-web` production build passed.
- `git diff --check` passed.
- No `omnigent.ai` / `omnigent-ai` URL remains in the scoped `ap-web` search.

## Next Milestone

Handle low-risk docs and non-runtime comments next, then separately design a compatibility-preserving rename for code identifiers, environment variables, storage keys, package names, and build/output paths.
