# Upgrade experience: `goalrail upgrade` + release-available notices

How Goalrail keeps a user's CLI — and the local processes it spawns — current
across PyPI releases. Three pieces ship together:

1. A **version-aware server signature** so a running local server is cycled
   onto new code automatically after any upgrade.
2. A **`goalrail upgrade`** command that upgrades the CLI and gracefully cycles
   the local server / daemon / runners.
3. A **"release available" notice** that fires only when a newer release is
   actually on PyPI, and points at `goalrail upgrade`.

## Background

- The CLI ships as `goalrail` plus legacy `goalrail` / `goalrail` aliases →
  `goalrail.cli:main()`; the installed
  version comes from `importlib.metadata.version("goalrail")`.
- Releases publish three lockstep packages (`goalrail`, `goalrail-client`,
  `goalrail-ui-sdk`) to PyPI via `.github/workflows/release-goalrail.yml`.
  **No GitHub Releases are cut**, so the source of truth for "latest version"
  is the PyPI JSON API: `https://pypi.org/pypi/goalrail/json` → `info.version`.
- The local server is a detached process on `:6767`, tracked by
  `~/.goalrail/local_server.pid` plus a config-signature sidecar
  (`local_server.sig`). `ensure_local_goalrail_server()` reuses a healthy
  server **iff its recorded signature matches**; otherwise it stops it and
  respawns. All durable state lives in sqlite (`~/.goalrail/chat.db`), not in
  process memory — which is what makes cycling a server safe.
- PR #172 removed an earlier startup check that nagged on *install age* (it
  fired even when you were already on the latest version) and did a synchronous
  `git fetch` on the hot path. The module (`goalrail/update_check.py`) stayed in
  the tree, dormant. This work rewires it correctly.

## Key constraint

You cannot hot-patch a running Python process. "Upgrade running
servers/daemons/runners gracefully" really means **cycle them**:

- **Server / daemon** — safe to stop and respawn; they rehydrate from sqlite.
- **Runners** — an in-flight runner *is* a running agent loop and cannot adopt
  new code mid-run. The honest options are **drain** (let it finish; new
  sessions get new code) or **stop** (lose in-flight work).

## 1. Version-aware server signature

`server_config_signature()` (`goalrail/host/local_server.py`) now folds the
installed package version into the signature alongside the resolved auth source:

```python
payload = json.dumps({"auth": resolve_auth_source(), "version": version}, sort_keys=True)
```

Effect: after *any* upgrade — `goalrail upgrade` **or** a manual
`uv tool upgrade` — the next CLI command sees the running server's recorded
signature no longer
match and respawns it on the new code through the **existing** config-drift
respawn path. This is the keystone: "running servers get upgraded" works even
when the user never runs the dedicated command. (Running from a source tree with
no registered distribution contributes an empty version and is unaffected.)

## 2. `goalrail upgrade`

`goalrail/cli.py`, command `upgrade`. Flow:

1. Bail on a source checkout / editable install (`_find_repo_root()` /
   `is_editable`) → tell the user to `git pull`.
2. Read the install shape (`_read_installed_wheel_info`) and compare the
   installed version with the latest on the configured index
   (`fetch_latest_version`, PEP 440 compare).
   - Already current → report and exit 0.
   - `--check` → print the available delta and exit non-zero (scriptable).
3. **Drain and wait** (default): poll the local server's *connected* sessions
   until idle so an upgrade never yanks a running agent turn. `--force` stops
   immediately (SIGTERM→SIGKILL). `Ctrl-C` aborts the wait.
4. Stop the server + daemon (`_stop_local_server_and_daemon`) — *before*
   swapping code, so the live process never serves half-upgraded modules.
5. Run the installer-appropriate command (`_build_upgrade_suggestion` +
   `_run_upgrade_command`): `uv tool upgrade goalrail`, `pip install -U
   goalrail`, `pipx upgrade goalrail`, `--reinstall <vcs_url>`, etc.
6. **Lazy respawn**: do not restart the server. The next `goalrail` command
   spawns a fresh new-code server via the signature change above.

Most of steps 2/5 reuse helpers that already existed in `update_check.py`.

## 3. "Release available" notice (the PR #172 redo)

`goalrail/update_check.py`, installed-wheel path; wired into `main()` behind
`_should_skip_update_check(argv)` and a `sys.stderr.isatty()` gate.

- **Only when newer**: compares installed vs. the cached latest version;
  notifies only when `latest > current`.
- **Configured-index aware**: `fetch_latest_version` queries the resolved
  index's Simple Repository API (PEP 691 JSON, PEP 503 HTML fallback), not
  PyPI's Warehouse-only JSON API. `_resolve_index_url()` checks
  `GOALRAIL_INDEX_URL` / `UV_INDEX_URL` / `PIP_INDEX_URL`, then the uv/pip
  **config files** (`uv.toml`'s `index-url` or default `[[index]]`; `pip.conf`'s
  `[global] index-url`), default `pypi.org/simple`. So it works on corporate
  mirrors / air-gapped networks even when the mirror lives in a config file
  (the common case), and matches what `goalrail upgrade` pulls.
- **Fire once per release**: the cache tracks `last_notified_version`; the
  notice shows once per new version, never on every invocation.
- **Never on the hot path**: the foreground only reads the cache (instant). The
  network lookup runs in a **detached** background process
  (`refresh_update_cache`, spawned via `python -c` so it can't recurse into the
  CLI) that refreshes the cache for next time.
- **Quiet + opt-out**: TTY-only, skipped for `--help` / `version` / internal TUI
  commands / `upgrade` itself, and silenced by `GOALRAIL_NO_UPDATE_CHECK`.
- Dev clones keep the existing git "commits behind origin/main" notice
  (pointing at `git pull`), unchanged.

The notice points at `goalrail upgrade`; it no longer runs an interactive upgrade
itself (that responsibility moved to the command).

## Decisions

- **Source of truth**: the configured package index's Simple Repository API
  (default `pypi.org/simple`; honors `GOALRAIL_INDEX_URL` / `UV_INDEX_URL` /
  `PIP_INDEX_URL` and the uv/pip config files). Picks the latest
  non-pre-release. No GitHub Releases are cut.
- **Drain posture**: drain-and-wait by default, `--force` to stop now.
- **Restart**: lazy respawn (no auto-restart) — simplest, fewest surprises.
- **Notice cadence**: once per new release.

## Not in scope (possible follow-ups)

- True rolling drain (new server on a new port, old one finishes) — unnecessary
  for a local single-user server.
- Pre-release / channel opt-in (the probe filters pre-releases today).
- Project-local `uv.toml` / `pyproject [tool.uv]` index URLs (only the user/
  system uv config and pip.conf are read — matching how `uv tool install`
  resolves a global tool; use `GOALRAIL_INDEX_URL` for anything else).
- A `/api/version` drift warning when attaching to a server you didn't spawn.
- A `config` toggle mirroring `GOALRAIL_NO_UPDATE_CHECK`.
