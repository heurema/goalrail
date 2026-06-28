"""Sandbox CLI commands: run a Goalrail host in a remote sandbox.

``goalrail sandbox …`` bootstraps a Goalrail host inside a sandbox
from one of the registered providers (``--provider``) so that sessions
on it are reachable from the server-hosted UI, TUI, and ``goalrail
resume``. Provider availability is build-dependent, so cli.py
registers this group only when at least one provider is available.

The provider-agnostic step implementations live in
:mod:`goalrail.onboarding.sandboxes`.
"""

from __future__ import annotations

from pathlib import Path

import click

from goalrail.inner import ui
from goalrail.onboarding.sandboxes import (
    SandboxLauncher,
    available_providers,
    get_launcher,
)


def _goalrail_repo_root() -> Path:
    """
    Locate the goalrail checkout that hosts the three packages
    we build wheels for (``sdks/python-client``, ``sdks/ui``, ``.``).

    Strategy:

    1. Walk up from the current working directory looking for a parent
       that contains both ``sdks/python-client`` and ``goalrail``.
       This succeeds when the user runs ``goalrail sandbox …`` from
       inside a checkout.
    2. Fall back to ``Path(__file__).resolve().parents[1]`` so an
       editable install (``pip install -e .``) still works when the
       user invokes from outside the checkout.

    :returns: Absolute path to the repo root.
    :raises click.ClickException: If neither strategy finds the expected
        ``sdks/python-client`` directory.
    """
    cwd = Path.cwd().resolve()
    candidates: list[Path] = [cwd, *cwd.parents]
    # ``cli_sandbox.py`` lives at ``<repo>/goalrail/cli_sandbox.py``;
    # parents[1] is the repo root that hosts ``sdks/`` and
    # ``goalrail/``. Editable installs (``pip install -e .``) point
    # ``__file__`` into the checkout, so this branch covers the "invoke
    # from outside cwd" case. Wheel installs land in site-packages where
    # the parent check below will (correctly) miss and we raise.
    candidates.append(Path(__file__).resolve().parents[1])
    for candidate in candidates:
        if (candidate / "sdks" / "python-client").is_dir() and (candidate / "goalrail").is_dir():
            return candidate
    raise click.ClickException(
        "Could not locate the goalrail repo root from "
        f"{cwd}. Pass --repo-root explicitly or run from inside a checkout."
    )


def _resolve_repo_root(repo_root: Path | None) -> Path:
    """
    Resolve a ``--repo-root`` option value to an absolute path.

    :param repo_root: User-supplied override, or ``None`` to autodetect.
    :returns: Absolute repo root path.
    :raises click.ClickException: If autodetection fails or the supplied
        path doesn't look like a checkout.
    """
    if repo_root is None:
        return _goalrail_repo_root()
    resolved = repo_root.resolve()
    if not (resolved / "sdks" / "python-client").is_dir():
        raise click.ClickException(
            f"--repo-root {resolved} doesn't contain sdks/python-client; "
            "point it at an goalrail checkout."
        )
    return resolved


def _require_cli_bootstrap(launcher: SandboxLauncher) -> None:
    """
    Reject managed-only providers up front with an actionable message.

    Some providers implement only the server-managed launch subset
    (``supports_cli_bootstrap`` is ``False``); reaching their missing
    file-shipping / streaming primitives mid-flow would surface as an
    opaque capability error after real work already ran.

    :param launcher: The resolved provider launcher.
    :raises click.ClickException: When the provider has no CLI
        bootstrap flow.
    """
    if not launcher.supports_cli_bootstrap:
        raise click.ClickException(
            f"The '{launcher.provider}' provider supports server-managed "
            "sessions only — create one with "
            '`POST /v1/sessions {"host_type": "managed"}` (or the Web '
            "UI's New Sandbox option) against a server configured with "
            f"`sandbox.provider: {launcher.provider}`."
        )


def _normalize_server_url(server_url: str) -> str:
    """
    Validate and normalize a ``--server`` value.

    Validation runs at the CLI boundary so a malformed URL fails
    BEFORE any sandbox work — without it, a scheme-less value sails
    through provisioning, wheel build, and ship, and only explodes at
    the final in-sandbox ``goalrail login`` step.

    :param server_url: Raw ``--server`` value, e.g.
        ``"https://app.example.com/"``.
    :returns: The URL without its trailing slash (a trailing slash
        breaks server-side URL joins).
    :raises click.ClickException: If the value does not start with
        ``http://`` or ``https://``.
    """
    normalized = server_url.rstrip("/")
    if not normalized.startswith(("http://", "https://")):
        raise click.ClickException(
            f"--server must be a full URL including the scheme, e.g. "
            f"https://{normalized.lstrip('/')} — got {server_url!r}."
        )
    return normalized


def _print_ready_banner(provider: str, sandbox_id: str, server_url: str) -> None:
    """
    Print the final "sandbox ready" instructions after a create.

    :param provider: The provider the sandbox was created with.
    :param sandbox_id: The sandbox the host is running in.
    :param server_url: Server URL for the connect hint (``--server``
        is required on create, so it is always known here).
    """
    ui.console.print()
    ui.success("Sandbox ready.")
    ui.console.print()
    ui.kv("Sandbox", f"{sandbox_id}  (provider: {provider})")
    ui.kv("Server", server_url)
    ui.console.print()
    click.echo("To register the sandbox as a host with your server:")
    click.echo(
        f"  goalrail sandbox connect --provider {provider} --sandbox-id {sandbox_id} "
        f"--server {server_url}\n"
    )


@click.group("sandbox")
def sandbox() -> None:
    """
    Run a Goalrail host inside a remote sandbox.

    \b
    Subcommands:
      create   Provision a sandbox + bootstrap Goalrail into it.
      connect  Register the sandbox as a host with your server (runs
               `goalrail host` in the sandbox).

    \b
    Provider notes:
      modal    Sandboxes live at most 24 hours (platform cap). Needs
               `pip install 'goalrail[modal]'` + `modal token new`.
      daytona  No lifetime cap (idle auto-stop is disabled). Needs
               `pip install 'goalrail[daytona]'` + DAYTONA_API_KEY.
               Free-tier (Tier 1/2) orgs only reach allowlisted
               domains, so `connect` needs an allowlisted --server
               (see deploy/daytona/README.md).
      islo     Uses the built-in HTTP client. Needs ISLO_API_KEY
               (and optionally ISLO_BASE_URL for non-default API
               endpoints).

    For provider-side sandbox lifecycle (list / status / delete /
    start / stop), use the provider's own CLI or dashboard directly
    (e.g. `modal sandbox list`).
    """


@sandbox.command("create")
@click.option(
    "--provider",
    type=click.Choice(available_providers()),
    required=True,
    help="Sandbox provider to use.",
)
@click.option(
    "--sandbox-id",
    "sandbox_id",
    default=None,
    # Re-ship code into an existing long-lived sandbox. Disposable
    # providers just create a new one.
    hidden=True,
    help="Attach to an existing sandbox by id (skip provisioning).",
)
@click.option(
    "--name",
    "sandbox_name",
    default=None,
    help="Label for the new sandbox.",
)
@click.option(
    "--server",
    "server_url",
    required=True,
    help=(
        "Server URL the sandbox will register with. The bootstrap "
        "finishes by logging the sandbox in to it (`goalrail login` "
        "inside the sandbox — one browser step when required)."
    ),
)
@click.option(
    "--repo-root",
    "repo_root",
    type=click.Path(file_okay=False, path_type=Path),
    default=None,
    help="Path to the goalrail checkout.",
)
@click.option(
    "--no-auth",
    "skip_auth",
    is_flag=True,
    default=False,
    hidden=True,
    help=(
        "Skip the in-sandbox server login. Providers that can't "
        "forward the callback port (e.g. modal) skip it automatically."
    ),
)
def sandbox_create(
    provider: str,
    sandbox_id: str | None,
    sandbox_name: str | None,
    server_url: str,
    repo_root: Path | None,
    skip_auth: bool,
) -> None:
    """
    Provision a sandbox and ship Goalrail into it.

    Builds the local wheels from your local checkout, installs them
    into the fresh sandbox, and finishes by logging the sandbox in to
    the server (``goalrail login`` runs inside the sandbox; the
    browser step is driven from this machine). Sandboxes are disposable
    — when your code changes, just create a new one.

    After this finishes, run ``goalrail sandbox connect`` to register
    the sandbox as a host with your server.
    """
    from goalrail.onboarding.sandboxes import (
        DEFAULT_SANDBOX_NAME,
        bootstrap_sandbox_host,
    )

    app_url = _normalize_server_url(server_url)
    launcher = get_launcher(provider)
    _require_cli_bootstrap(launcher)
    # The in-sandbox login only exists for providers that can forward
    # the browser's callback port — others skip it automatically, no
    # --no-auth acknowledgement required.
    if not launcher.supports_local_port_forward:
        skip_auth = True
    sandbox_id = bootstrap_sandbox_host(
        launcher,
        sandbox_id=sandbox_id,
        sandbox_name=sandbox_name or DEFAULT_SANDBOX_NAME,
        server_url=app_url,
        repo_root=_resolve_repo_root(repo_root),
        skip_auth=skip_auth,
    )
    _print_ready_banner(provider, sandbox_id, app_url)


@sandbox.command("connect")
@click.option(
    "--provider",
    type=click.Choice(available_providers()),
    required=True,
    help="Sandbox provider to use.",
)
@click.option("--sandbox-id", "sandbox_id", required=True, help="Sandbox to register as a host.")
@click.option(
    "--server",
    "server_url",
    required=True,
    help="Server URL the sandbox will register with.",
)
@click.option(
    "--host-name",
    "host_name",
    default=None,
    help=(
        "Name to register the sandbox as. Defaults to the sandbox's hostname. "
        "The server's hosts table is keyed on (owner, name), so sandboxes "
        "sharing a hostname collide; pass a unique value per sandbox."
    ),
)
def sandbox_connect(
    provider: str,
    sandbox_id: str,
    server_url: str,
    host_name: str | None,
) -> None:
    """
    Register the sandbox as a host with your server.

    Runs ``goalrail host --server <url>`` inside the sandbox — the
    host resolves its own credentials. The remote command holds a
    WebSocket open until interrupted — Ctrl-C tears down the foreground
    transport and the remote process.

    Pass ``--host-name <label>`` when registering multiple sandboxes —
    sandboxes that share a hostname collide on the server's
    (owner, name) primary key.
    """
    from goalrail.onboarding.sandboxes import connect_sandbox_host

    app_url = _normalize_server_url(server_url)
    launcher = get_launcher(provider)
    _require_cli_bootstrap(launcher)
    connect_sandbox_host(
        launcher,
        sandbox_id,
        server_url=app_url,
        host_name=host_name,
    )
