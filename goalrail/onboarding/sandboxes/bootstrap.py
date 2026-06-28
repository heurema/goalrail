"""
Provider-agnostic sandbox bootstrap for Goalrail hosts.

Composes a :class:`~goalrail.onboarding.sandboxes.base.SandboxLauncher`
into the full host-bootstrap flow: build the Goalrail wheels locally,
ship + install them in the sandbox, run ``goalrail login`` *inside the
sandbox* when requested, and register the sandbox as a host by holding
``goalrail host`` open in it. The end state is a sandbox whose sessions
are reachable from the Goalrail server's UI, TUI, and ``goalrail
resume``.

The OAuth token is minted and stored by the sandbox's own CLI rather
than shipped from the laptop. Shipping a laptop-minted token was
fundamentally broken: modern CLIs store the live token in the OS
keyring (so the shippable file was stale), the local and in-sandbox CLI
versions disagree on the token-cache layout, and U2M refresh tokens are
single-use — laptop and sandbox can't both hold the same one. Logging
in inside the sandbox makes it the sole token holder and sidesteps all
three.

Everything provider-specific (transport, image quirks, pip flags) lives
behind the launcher.
"""

from __future__ import annotations

import os
import re
import shlex
import shutil
import subprocess
import tarfile
import tempfile
import webbrowser
from pathlib import Path
from typing import TYPE_CHECKING
from urllib.parse import parse_qs, urlparse

import click

if TYPE_CHECKING:
    from collections.abc import Iterable

    from goalrail.onboarding.sandboxes.base import RemoteProcess, SandboxLauncher


# ── Constants ──────────────────────────────────────────

WHEEL_PACKAGE_PATHS: tuple[str, ...] = ("sdks/python-client", "sdks/ui", ".")
"""Repo-relative paths of the three packages we bundle for sandbox
install: the python client SDK, the UI SDK, and the goalrail package
itself (which path-depends on the first two)."""

DEFAULT_WHEELS_TGZ: str = "/tmp/oa-wheels.tgz"
"""Local staging path for the packed wheel tarball. Rebuilt fresh on
every bootstrap so the sandbox always gets exactly the current
checkout's code."""

DEFAULT_BUILD_LOG: str = "/tmp/goalrail-sandbox-build.log"
"""Default ``uv build`` log location."""

DEFAULT_SANDBOX_NAME: str = "goalrail-host"
"""Default label used when ``goalrail sandbox create`` provisions a
new sandbox."""

_REMOTE_WHEELS_TGZ: str = "/tmp/oa-wheels.tgz"
"""Where :func:`ship_wheels` places the wheel tarball inside the
sandbox before unpacking it."""

# Matches ANSI CSI escape sequences. In-sandbox login output arrives
# over a PTY, so URL lines may be wrapped in color/cursor codes that
# must be stripped before parsing.
_ANSI_ESCAPE_PATTERN: re.Pattern[str] = re.compile(r"\x1b\[[0-9;?]*[A-Za-z]")


# ── Wheel build ────────────────────────────────────────


def build_wheels(
    repo_root: Path,
    *,
    tgz_path: Path = Path(DEFAULT_WHEELS_TGZ),
    build_log: Path = Path(DEFAULT_BUILD_LOG),
    pypi_proxy: str | None = None,
) -> None:
    """
    Build the Goalrail wheels and pack them into a single tarball.

    Builds the three packages in :data:`WHEEL_PACKAGE_PATHS` via
    ``uv build --wheel`` into a staging directory, then tars the staging
    directory into *tgz_path*. The python-client and ui SDKs are
    path-deps of the root package, so all three must be built fresh in
    the same pass.

    Always builds fresh: the sandbox must end up running exactly the
    code in *repo_root*. (An earlier existence-based tarball cache made
    users reason about staleness via a --rebuild-wheels flag, with
    silently shipping old code as the failure mode.)

    :param repo_root: Path to the goalrail repo checkout, e.g.
        ``Path("/home/me/goalrail")``.
    :param tgz_path: Output path for the packed tarball.
    :param build_log: Path to write the combined ``uv build`` log to.
    :param pypi_proxy: ``UV_INDEX_URL`` override, or ``None`` to use
        ambient uv configuration. Launchers supply this via
        ``SandboxLauncher.wheel_build_index_url`` when the build machine
        sits on a network that can't reach public PyPI.
    :raises click.ClickException: If ``uv`` is not on PATH or any
        package's ``uv build`` exits non-zero.
    """
    if shutil.which("uv") is None:
        raise click.ClickException(
            "`uv` is required to build wheels. Install via "
            "`curl -LsSf https://astral.sh/uv/install.sh | sh` and retry."
        )

    click.echo(f"▸ Building Goalrail wheels → {tgz_path}")
    env = os.environ.copy()
    if pypi_proxy is not None:
        env.setdefault("UV_INDEX_URL", pypi_proxy)

    build_log.write_text("", encoding="utf-8")
    with tempfile.TemporaryDirectory(prefix="oa-wheels-") as stage_str:
        stage = Path(stage_str)
        for pkg in WHEEL_PACKAGE_PATHS:
            click.echo(f"  → {pkg}")
            _uv_build_wheel(repo_root / pkg, pkg, stage=stage, build_log=build_log, env=env)
        wheel_count = _pack_wheels(stage, tgz_path)

    click.echo(f"  → packed {wheel_count} wheels at {tgz_path}")


def _uv_build_wheel(
    pkg_dir: Path,
    pkg: str,
    *,
    stage: Path,
    build_log: Path,
    env: dict[str, str],
) -> None:
    """
    Run ``uv build --wheel`` for one package into the staging directory.

    :param pkg_dir: Absolute package directory to build from.
    :param pkg: Repo-relative package label for log/error messages,
        e.g. ``"sdks/python-client"``.
    :param stage: Staging directory uv writes the wheel into.
    :param build_log: Combined build log to append uv's output to.
    :param env: Environment for the uv subprocess (carries the
        ``UV_INDEX_URL`` override when set).
    :raises click.ClickException: If ``uv build`` exits non-zero.
    """
    with build_log.open("a", encoding="utf-8") as log:
        log.write(f"\n=== uv build {pkg} ===\n")
        log.flush()
        result = subprocess.run(
            ["uv", "build", "--wheel", "--out-dir", str(stage)],
            cwd=pkg_dir,
            stdout=log,
            stderr=subprocess.STDOUT,
            env=env,
            check=False,
        )
    if result.returncode != 0:
        raise click.ClickException(f"`uv build` for {pkg} failed — see {build_log}")


def _pack_wheels(stage: Path, tgz_path: Path) -> int:
    """
    Tar every wheel in the staging directory into *tgz_path*.

    :param stage: Directory holding the freshly-built ``*.whl`` files.
    :param tgz_path: Output path for the packed tarball.
    :returns: Number of wheels packed.
    """
    wheels = sorted(stage.glob("*.whl"))
    with tarfile.open(tgz_path, "w:gz") as tar:
        for wheel in wheels:
            tar.add(wheel, arcname=wheel.name)
    return len(wheels)


# ── Wheel install ──────────────────────────────────────


def ship_wheels(
    launcher: SandboxLauncher,
    sandbox_id: str,
    *,
    wheels_tgz: Path,
) -> None:
    """
    Install Goalrail wheels into a sandbox.

    Performs three remote operations, in order: ship wheels.tgz →
    pip install (the launcher supplies the image-appropriate flags) →
    PATH-export persistence. Credentials are *not* shipped — the
    sandbox performs its own server login when requested.

    :param launcher: The provider's launcher.
    :param sandbox_id: Target sandbox, e.g. ``"lovable-wattlebird-1530"``.
    :param wheels_tgz: Local path to the packed wheel tarball.
    :raises click.ClickException: If any of the three steps fail.
    """
    click.echo("▸ Shipping wheels into the sandbox")

    click.echo("  → wheels")
    launcher.put(sandbox_id, wheels_tgz, _REMOTE_WHEELS_TGZ)

    click.echo("  → pip install")
    launcher.run(sandbox_id, launcher.wheel_install_command(_REMOTE_WHEELS_TGZ))

    click.echo("  → PATH persistence in sandbox")
    launcher.run(
        sandbox_id,
        "for f in ~/.bashrc ~/.bash_profile; do "
        'grep -q ".local/bin" "$f" 2>/dev/null || '
        'echo "export PATH=\\$HOME/.local/bin:\\$PATH" >> "$f"; '
        "done",
    )


# ── App OAuth (minted inside the sandbox) ──────────────


def _extract_oauth_url(line: str) -> str | None:
    """
    Pull an OAuth authorize URL out of one line of CLI output.

    Login commands usually print the verification URL on its own line
    (``https://<host>/oidc/v1/authorize?...``). Sandbox output is
    wrapped in a PTY, so the line may carry ANSI codes and a trailing
    carriage return; both are stripped before matching.

    :param line: One line of combined stdout/stderr from the in-sandbox
        login process.
    :returns: The authorize URL if this line contains one, else
        ``None``.
    """
    clean = _ANSI_ESCAPE_PATTERN.sub("", line).strip()
    start = clean.find("https://")
    if start == -1 or "/oidc/v1/authorize" not in clean:
        return None
    return clean[start:]


def _loopback_port_from_authorize_url(url: str) -> int:
    """
    Extract the loopback callback port from an OAuth authorize URL.

    The authorize URL carries a ``redirect_uri`` query parameter such as
    ``http://localhost:8022``. The in-sandbox CLI binds that port
    dynamically (the first free loopback port at/above 8020), so the
    actual value must be read back from the URL to know which port to
    forward.

    :param url: The authorize URL, e.g. ``"https://auth.example.com/"
        "oidc/v1/authorize?...&redirect_uri=http%3A%2F%2Flocalhost"
        "%3A8022&..."``.
    :returns: The callback port, e.g. ``8022``.
    :raises click.ClickException: If no ``redirect_uri`` with a loopback
        port can be parsed (the login URL format changed).
    """
    redirect = parse_qs(urlparse(url).query).get("redirect_uri", [None])[0]
    port = urlparse(redirect).port if redirect else None
    if port is None:
        raise click.ClickException(
            f"Could not parse an OAuth callback port from the login URL: {url}"
        )
    return port


def _read_login_url(stream: Iterable[str]) -> str | None:
    """
    Read login-process output until the OAuth verification URL appears,
    echoing every non-URL line (ANSI-stripped) as it streams.

    The echo is load-bearing for debuggability: when the in-sandbox
    login dies before printing a URL, its own error message is the
    only evidence of why — swallowing it here would leave the user
    with nothing but an exit code.

    :param stream: Line iterator over the in-sandbox login process's
        combined output (``RemoteProcess.lines``, or a list of lines in
        tests).
    :returns: The authorize URL, or ``None`` when the stream ends
        without printing one — which is NOT necessarily an error:
        ``goalrail login`` reuses a cached OAuth grant when
        one verifies against the server, completing without a browser
        step. The caller distinguishes success from failure by the
        process's exit code.
    """
    for line in stream:
        url = _extract_oauth_url(line)
        if url is not None:
            return url
        text = _ANSI_ESCAPE_PATTERN.sub("", line).rstrip()
        if text:
            click.echo(f"    {text}")
    return None


def _drain_login_output(stream: Iterable[str]) -> None:
    """
    Echo remaining login output (ANSI-stripped) until the stream ends.

    Called after the URL is revealed; the in-sandbox CLI prints a
    confirmation line when the user completes the browser flow, then
    closes the stream.

    :param stream: Line iterator over the login process's output.
    """
    for line in stream:
        text = _ANSI_ESCAPE_PATTERN.sub("", line).rstrip()
        if text:
            click.echo(f"    {text}")


def login_app_oauth_in_sandbox(
    launcher: SandboxLauncher,
    sandbox_id: str,
    *,
    server_url: str | None,
    skip: bool = False,
) -> None:
    """
    Log the sandbox in to *server_url* by running ``goalrail login``
    **inside the sandbox**, driving the browser step from the local
    machine.

    ``goalrail login`` owns the credential inference used everywhere
    else. The sandbox mints and stores its own credential rather than
    receiving one from the laptop.

    The sandbox is headless, so when the login needs a browser this:

    1. runs ``goalrail login <server_url>`` inside the sandbox over a
       PTY;
    2. reads the dynamically-chosen loopback callback port back from the
       printed authorize URL;
    3. forwards ``localhost:<port>`` on the local machine into the
       sandbox (and waits for it to bind) so the browser's OAuth
       redirect reaches the in-sandbox listener; and
    4. opens the authorize URL in the local browser.

    When the in-sandbox login completes without printing an authorize
    URL (for example, a cached grant verified against the server), the
    browser steps are skipped and success is read from the exit code.

    The sandbox ends up the sole holder of its OAuth grant — nothing is
    shipped from the laptop — which sidesteps CLI-version cache-format
    skew, OS-keyring storage, and single-use refresh-token rotation.

    :param launcher: The provider's launcher. Must support
        ``forward_local_port`` (providers without it raise
        ``SandboxCapabilityError`` naming the ``--no-auth`` escape
        hatch).
    :param sandbox_id: Target sandbox, e.g. ``"fast-tarantula-6030"``.
    :param server_url: Goalrail server URL to log in to, e.g.
        ``"https://app.example.com"``. Required unless *skip*.
    :param skip: When ``True``, skip authentication entirely (the
        ``--no-auth`` escape hatch).
    :raises click.ClickException: If *server_url* is missing, the
        forward fails to bind, or the in-sandbox login exits non-zero.
    """
    if skip:
        click.echo("▸ Skipping the in-sandbox server login")
        return
    # Fail fast for providers that can't bridge the OAuth callback port
    # (e.g. Modal) — BEFORE validating flags or touching the sandbox, so
    # the user gets the --no-auth hint instead of a misleading error
    # from a doomed in-sandbox login.
    if not launcher.supports_local_port_forward:
        raise launcher.forward_capability_error()
    if server_url is None:
        raise click.ClickException(
            "The in-sandbox login needs the server URL — pass --server, or --no-auth to skip."
        )

    click.echo(f"▸ Logging sandbox '{sandbox_id}' in to {server_url}")
    login = launcher.stream_exec(
        sandbox_id,
        f"goalrail login {shlex.quote(server_url)}",
        pty=True,
    )
    try:
        _complete_browser_login(launcher, sandbox_id, login)
    finally:
        login.close()


def _complete_browser_login(
    launcher: SandboxLauncher,
    sandbox_id: str,
    login: RemoteProcess,
) -> None:
    """
    Drive the browser half of an in-sandbox ``goalrail login``.

    Reads the authorize URL off the login process's output, bridges the
    URL's dynamically-chosen loopback callback port into the sandbox,
    opens the URL in the local browser, and waits for the login to
    finish. A login that completes without printing an authorize URL
    (cached grant verified) skips the browser steps entirely
    — success vs. failure is then read from the exit code alone.

    :param launcher: The provider's launcher (supplies the port
        forward).
    :param sandbox_id: Sandbox the login process is running in.
    :param login: The streaming in-sandbox login process; the caller
        owns its cleanup.
    :raises click.ClickException: If the forward fails or the login
        exits non-zero.
    """
    url = _read_login_url(login.lines)
    if url is None:
        # No browser needed: `goalrail login` verified a cached
        # grant against the server (or failed before the browser step
        # — the exit code tells which).
        returncode = login.wait()
        if returncode != 0:
            raise click.ClickException(
                f"`goalrail login` inside sandbox '{sandbox_id}' exited "
                f"with code {returncode} before printing a verification "
                "URL. Run it inside the sandbox manually to debug."
            )
        click.echo("  → cached credentials accepted; no browser login needed")
        return
    port = _loopback_port_from_authorize_url(url)
    # Stand up (and confirm) the forward BEFORE revealing the URL so
    # the browser redirect can't race ahead of the tunnel.
    with launcher.forward_local_port(sandbox_id, port):
        click.echo("  → Opening the OAuth URL in your browser. If it doesn't open, visit:")
        click.echo(f"    {url}")
        click.echo(
            "  → On a headless host (Arca)? Forward the callback port to your "
            f"laptop too: `ssh -L {port}:localhost:{port} -N <this-host>`."
        )
        webbrowser.open(url)
        _drain_login_output(login.lines)
        returncode = login.wait()
        if returncode != 0:
            raise click.ClickException(
                f"`goalrail login` inside sandbox '{sandbox_id}' exited with code {returncode}."
            )


# ── Host registration ──────────────────────────────────


def set_sandbox_host_name(launcher: SandboxLauncher, sandbox_id: str, host_name: str) -> None:
    """
    Update the sandbox's ``~/.goalrail/config.yaml`` to use a
    specific host name.

    The host's ``host_id`` is preserved across the edit — only the
    ``name`` field is rewritten. If config.yaml doesn't exist yet,
    a minimal one is created with a fresh host_id; the next
    ``goalrail host`` will load that file as-is.

    Implementation note: the edit runs as a Python one-liner inside
    the sandbox (instead of ``sed``) so it survives YAML quirks
    (quoting, multi-doc, etc.) and produces well-formed output.

    :param launcher: The provider's launcher.
    :param sandbox_id: Target sandbox.
    :param host_name: New host name to write into config.yaml.
    :raises click.ClickException: If the remote command fails.
    """
    click.echo(f"  → setting host name to '{host_name}' in ~/.goalrail/config.yaml")
    # Quote the host name in single quotes for the python literal,
    # then escape any single quotes the user passed in.
    safe_name = host_name.replace("'", "\\'")
    py = (
        "import os, uuid, yaml; "
        "p=os.path.expanduser('~/.goalrail/config.yaml'); "
        "os.makedirs(os.path.dirname(p), exist_ok=True); "
        "cfg=yaml.safe_load(open(p)) if os.path.exists(p) else {}; "
        "cfg=cfg or {}; "
        f"h=cfg.get('host') or {{}}; h['name']='{safe_name}'; "
        "h.setdefault('host_id', 'host_'+uuid.uuid4().hex); "
        "cfg['host']=h; "
        "yaml.safe_dump(cfg, open(p,'w'), default_flow_style=False, sort_keys=True)"
    )
    launcher.run(sandbox_id, f'python3 -c "{py}"')


def connect_sandbox_host(
    launcher: SandboxLauncher,
    sandbox_id: str,
    *,
    server_url: str,
    host_name: str | None = None,
) -> None:
    """
    Register the sandbox as a host by running ``goalrail host`` in it.

    The remote command holds a WebSocket open until interrupted —
    Ctrl-C tears down the foreground transport and the remote process.

    The remote command is always the bare ``goalrail host --server
    <url>``: ``goalrail host`` no longer takes a ``--profile`` flag
    — it resolves credentials itself, via a stored ``goalrail login``
    token or provider-native ambient credentials.

    When *host_name* is set, the sandbox's
    ``~/.goalrail/config.yaml`` is updated so the host registers
    with that name instead of the default ``socket.gethostname()``.
    This is useful when a provider reuses hostnames and the server's
    ``hosts`` table would otherwise collide on ``(owner, name)``.

    :param launcher: The provider's launcher.
    :param sandbox_id: Target sandbox.
    :param server_url: Goalrail App URL the runner registers with.
    :param host_name: Optional override for the host's registered
        name. ``None`` keeps whatever's already in the sandbox's
        config.yaml (usually ``socket.gethostname()``).
    :raises click.ClickException: If the remote command exits non-zero.
    """
    click.echo(f"▸ Registering sandbox '{sandbox_id}' as a host with {server_url}")
    if host_name is not None:
        set_sandbox_host_name(launcher, sandbox_id, host_name)
    click.echo("  → running `goalrail host` in the sandbox (Ctrl-C to detach)")
    returncode = launcher.exec_foreground(sandbox_id, f"goalrail host --server {server_url}")
    if returncode != 0:
        raise click.ClickException(
            f"`goalrail host` on sandbox '{sandbox_id}' exited with code {returncode}."
        )


# ── High-level orchestrator ────────────────────────────


def bootstrap_sandbox_host(
    launcher: SandboxLauncher,
    *,
    sandbox_id: str | None,
    sandbox_name: str,
    server_url: str | None,
    repo_root: Path,
    skip_auth: bool,
) -> str:
    """
    Run the full sandbox-host bootstrap end-to-end.

    Six steps: provider preflight → provision or attach sandbox →
    keep-alive → build wheels → ship wheels → ``goalrail login``
    inside the sandbox.

    :param launcher: The provider's launcher.
    :param sandbox_id: Existing sandbox id to attach to, or ``None`` to
        provision a new one.
    :param sandbox_name: Label for a new sandbox (ignored when
        *sandbox_id* is set).
    :param server_url: Goalrail server URL the sandbox logs in to.
        Required unless *skip_auth*.
    :param repo_root: Path to the Goalrail repo checkout.
    :param skip_auth: When ``True``, skip the in-sandbox login.
    :returns: The sandbox id (the one we created or attached to).
    :raises click.ClickException: Propagated from any failing step.
    :raises SandboxCapabilityError: Immediately (before any remote
        work) when auth is requested but the provider cannot forward
        the OAuth callback port — pass ``skip_auth`` for such
        providers (the CLI does this automatically).
    """
    # The login step is last; check its one hard capability requirement
    # up front so a misconfigured call fails before the wheel build and
    # ship already ran. (The CLI skips auth automatically for providers
    # without the capability; this backstops programmatic callers.)
    if not skip_auth and not launcher.supports_local_port_forward:
        raise launcher.forward_capability_error()
    launcher.prepare()
    if sandbox_id is None:
        sandbox_id = launcher.provision(sandbox_name)
    else:
        launcher.attach(sandbox_id)
    click.echo(f"  → sandbox_id={sandbox_id}")
    launcher.keep_alive(sandbox_id)
    wheels_tgz = Path(DEFAULT_WHEELS_TGZ)
    build_wheels(
        repo_root,
        tgz_path=wheels_tgz,
        pypi_proxy=launcher.wheel_build_index_url,
    )
    ship_wheels(launcher, sandbox_id, wheels_tgz=wheels_tgz)
    login_app_oauth_in_sandbox(
        launcher,
        sandbox_id,
        server_url=server_url,
        skip=skip_auth,
    )
    return sandbox_id
