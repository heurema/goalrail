# Milestone 3: ap-web Native Wrapper Docs and iOS Setup Copy

Date: 2026-06-26

## Scope

Updated low-risk `ap-web` native wrapper prose and one visible iOS setup string from Goalrail.

Changed surfaces:

- `ap-web` README product prose.
- Electron wrapper README product prose.
- iOS wrapper README and release prose.
- Shared native platform assets README.
- iOS connect screen copy.

## Intentionally Left For Later

Current technical names remain unchanged:

- `goalrail` CLI/server command.
- `GOALRAIL_*` environment variables.
- `GoalrailLogo` / `GoalrailLogoReverse` asset names.
- `Goalrail.xcodeproj`, `Goalrail` target/scheme, and `GoalrailTests`.
- Electron packaging artifacts such as `Goalrail-<version>-<arch>.dmg`.
- Per-user app data paths such as `~/Library/Application Support/Goalrail/settings.json`.

These names need a native packaging and compatibility plan before renaming.

## Verification

Commands:

```sh
codebase-memory-mcp cli index_status '{"project":"Users-vi-personal-heurema-goalrail"}'
codebase-memory-mcp cli search_code '{"project":"Users-vi-personal-heurema-goalrail","pattern":"Goalrail","limit":250}'
codebase-memory-mcp cli get_code_snippet '{"project":"Users-vi-personal-heurema-goalrail","qualified_name":"Users-vi-personal-heurema-goalrail.ap-web.ios.Goalrail.ConnectView.body"}'
git diff --check
git grep -n -I -E 'Goalrail|Goalrail|goalrail\.ai|heurema' -- ap-web/README.md ap-web/electron/README.md ap-web/ios/README.md ap-web/ios/RELEASE.md ap-web/platform-assets/README.md ap-web/ios/Goalrail/ConnectView.swift
PATH="/Users/vi/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin:$PATH" npm --prefix ap-web run build
xcrun swift-format lint ap-web/ios/Goalrail/ConnectView.swift
```

Results:

- Codebase-memory index was ready.
- `git diff --check` passed.
- Scoped residual search only shows intentional legacy technical names.
- `ap-web` production build passed.
- `swift-format lint` passed for the changed Swift file.

Blocked verification:

```sh
xcodebuild -project ap-web/ios/Goalrail.xcodeproj -scheme Goalrail -destination 'generic/platform=iOS Simulator' build
```

Result: blocked by local Xcode plugin loading failure for `IDESimulatorFoundation`; Xcode suggested running `xcodebuild -runFirstLaunch`.

## Next Milestone

Continue with low-risk docs outside `ap-web` only where references are product prose. Keep GHCR image names, GitHub org/repo links, bot identities, package names, API namespaces, and deployment resources unchanged until there is a compatibility plan.
