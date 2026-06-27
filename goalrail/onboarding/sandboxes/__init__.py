"""
Sandbox launchers: run Goalrail hosts in remote sandboxes.

Public API for the ``goalrail sandbox`` CLI and anything else that
bootstraps a sandbox-backed host. Providers are registered by name in
:data:`_LAUNCHERS`; launcher modules may be absent from a given
distribution (e.g. the Databricks Lakebox launcher), in which case the
provider simply isn't offered.
"""

from __future__ import annotations

import importlib
import importlib.util

import click

from goalrail.onboarding.sandboxes.base import (
    RemoteCommandResult,
    RemoteProcess,
    SandboxCapabilityError,
    SandboxLauncher,
)
from goalrail.onboarding.sandboxes.bootstrap import (
    DEFAULT_SANDBOX_NAME,
    DerivedWorkspace,
    bootstrap_sandbox_host,
    build_wheels,
    connect_sandbox_host,
    derive_workspace,
    login_app_oauth_in_sandbox,
    set_sandbox_host_name,
    ship_wheels,
)

__all__ = [
    "DEFAULT_SANDBOX_NAME",
    "DerivedWorkspace",
    "RemoteCommandResult",
    "RemoteProcess",
    "SandboxCapabilityError",
    "SandboxLauncher",
    "available_providers",
    "bootstrap_sandbox_host",
    "build_wheels",
    "connect_sandbox_host",
    "derive_workspace",
    "get_launcher",
    "login_app_oauth_in_sandbox",
    "set_sandbox_host_name",
    "ship_wheels",
]

# Provider name → "module:ClassName" of its SandboxLauncher. Modules are
# imported lazily (some pull in optional SDKs) and may be absent from a
# distribution entirely (e.g. lakebox).
_LAUNCHERS: dict[str, str] = {
    "lakebox": "goalrail.onboarding.sandboxes.lakebox:LakeboxLauncher",
    "modal": "goalrail.onboarding.sandboxes.modal:ModalSandboxLauncher",
    "daytona": "goalrail.onboarding.sandboxes.daytona:DaytonaSandboxLauncher",
    "boxlite": "goalrail.onboarding.sandboxes.boxlite:BoxliteSandboxLauncher",
    # CoreWeave Sandbox via the official cwsandbox SDK (the
    # `goalrail[cwsandbox]` extra), imported lazily like modal/daytona.
    "cwsandbox": "goalrail.onboarding.sandboxes.cwsandbox:CWSandboxLauncher",
    "islo": "goalrail.onboarding.sandboxes.islo:IsloSandboxLauncher",
    # E2B (https://e2b.dev) via the official `e2b` SDK (the
    # `goalrail[e2b]` extra), imported lazily like modal/daytona.
    "e2b": "goalrail.onboarding.sandboxes.e2b:E2BSandboxLauncher",
    "openshell": "goalrail.onboarding.sandboxes.openshell:OpenShellSandboxLauncher",
    # On-demand Kubernetes runner Pod via the official kubernetes client (the
    # `goalrail[kubernetes]` extra), imported lazily like modal/daytona.
    "kubernetes": "goalrail.onboarding.sandboxes.kubernetes:KubernetesSandboxLauncher",
}


def available_providers() -> tuple[str, ...]:
    """
    List the sandbox providers whose launcher modules exist in this
    build.

    Uses ``find_spec`` (no import side effects), so it is cheap enough
    to call at CLI startup to decide whether to register the
    ``goalrail sandbox`` command group.

    :returns: Provider names in registration order, e.g.
        ``("lakebox", "modal")`` internally or ``("modal",)`` in the
        OSS build (where the lakebox module is excluded).
    """
    available: list[str] = []
    for name, target in _LAUNCHERS.items():
        module_name = target.partition(":")[0]
        if importlib.util.find_spec(module_name) is not None:
            available.append(name)
    return tuple(available)


def get_launcher(provider: str, *, workspace_host: str | None = None) -> SandboxLauncher:
    """
    Resolve a provider name to a launcher instance.

    :param provider: Provider name, e.g. ``"lakebox"``.
    :param workspace_host: Databricks workspace fronting the target
        server (derived from ``--server`` via
        :func:`~goalrail.onboarding.sandboxes.bootstrap.derive_workspace`),
        e.g. ``"https://example.databricks.com"``. Consumed only by
        the lakebox launcher, which pins its local ``databricks
        lakebox`` calls to it so sandboxes are created in the server's
        workspace. Other providers' sandboxes don't live in a
        Databricks workspace, so the value is meaningless for them and
        deliberately not forwarded — the CLI derives it from --server
        for every provider, so rejecting it here would break them.
    :returns: A fresh launcher for the provider.
    :raises click.ClickException: If the provider is unknown or its
        launcher module is not present in this build.
    """
    target = _LAUNCHERS.get(provider)
    if target is None or provider not in available_providers():
        offered = ", ".join(available_providers()) or "(none in this build)"
        raise click.ClickException(
            f"Unknown or unavailable sandbox provider '{provider}'. Available: {offered}."
        )
    if provider == "lakebox" and workspace_host is not None:
        # Imported here (not at module top) because the lakebox module
        # may be absent from a distribution; the availability check above
        # guarantees it exists in this one.
        from goalrail.onboarding.sandboxes.lakebox import LakeboxLauncher

        return LakeboxLauncher(workspace_host=workspace_host)
    module_name, _, class_name = target.partition(":")
    module = importlib.import_module(module_name)
    launcher_cls: type[SandboxLauncher] = getattr(module, class_name)
    return launcher_cls()
