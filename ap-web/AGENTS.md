# AGENTS.md

Guidance for agents working under `ap-web/` in the main `heurema/goalrail`
repository.

## Scope

`ap-web/` owns the server-served web UI plus native app shells that load that UI:

- Vite/React web UI in `ap-web/`.
- Electron desktop shell in `ap-web/electron/`.
- iOS SwiftUI/WKWebView shell in `ap-web/ios/`.

Website deployment for `https://goalrail.dev` is not handled here. Site changes
live in the separate `goalrail-site` repository and production rollout goes
through `/Users/vi/personal/standalone/infra/scripts/deploy-goalrail-site.sh`.

## Versioning

- Do not bump versions for build-only tasks.
- Do not edit `ap-web/electron/package.json` `version`, iOS
  `MARKETING_VERSION`, Python package versions, tags, or release branches unless
  the user explicitly asks for a versioned release.
- iOS Fastlane beta/release lanes compute `CFBundleVersion` at archive time; do
  not commit build-number churn.

## Web UI Build

Use this when a task asks to build the web UI bundle shipped by the Goalrail
server or Python wheel:

```bash
npm --prefix ap-web ci --legacy-peer-deps
npm --prefix ap-web run build
```

The bundle is written to `goalrail/server/static/web-ui/`. Treat it as a build
artifact unless the task is explicitly about packaging a release artifact.

## Electron Desktop Build

From `ap-web/electron/`:

```bash
npm install
npm run test
npm run build          # current platform
npm run build:mac      # macOS DMG + ZIP, signed if a local identity exists
npm run build:linux    # AppImage + deb
npm run build:win      # NSIS installer
```

Use `npm run build:mac:release` only when the user explicitly asks for a
notarized macOS release and the required Apple signing/notarization credentials
are already available. Never print or commit signing credentials.

Build outputs land under `ap-web/electron/dist/`; do not commit them unless the
user explicitly asks for a release artifact handoff.

## iOS Build

Use Xcode for local development builds. For TestFlight/App Store packaging,
follow `ap-web/ios/RELEASE.md`:

```bash
cd ap-web/ios
bundle install
bundle exec fastlane tests
bundle exec fastlane beta
```

Use `bundle exec fastlane release` only when the user explicitly asks for an App
Store Connect binary upload. Fastlane credentials live in ignored env files; do
not print or commit them.

## Reporting

For any build, report:

- exact commands run;
- output artifact paths;
- whether versions were changed;
- tests/checks run;
- anything not verified, especially signing, notarization, TestFlight, or App
  Store upload status.
