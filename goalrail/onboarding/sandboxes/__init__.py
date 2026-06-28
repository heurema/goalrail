"""
Sandbox launchers: run Goalrail hosts in remote sandboxes.

Public API for the ``goalrail sandbox`` CLI and anything else that
bootstraps a sandbox-backed host. Providers are registered by name in
:data:`_LAUNCHERS`; launcher modules may be absent from a given
distribution, in which case the provider simply isn't offered.
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
    bootstrap_sandbox_host,
    build_wheels,
    connect_sandbox_host,
    set_sandbox_host_name,
    ship_wheels,
)

__all__ = [
    "DEFAULT_SANDBOX_NAME",
    "RemoteCommandResult",
    "RemoteProcess",
    "SandboxCapabilityError",
    "SandboxLauncher",
    "available_providers",
    "bootstrap_sandbox_host",
    "build_wheels",
    "connect_sandbox_host",
    "get_launcher",
    "set_sandbox_host_name",
    "ship_wheels",
]

# Provider name → "module:ClassName" of its SandboxLauncher. Modules are
# imported lazily (some pull in optional SDKs) and may be absent from a
# distribution entirely.
_LAUNCHERS: dict[str, str] = {
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

    :returns: Provider names in registration order.
    """
    available: list[str] = []
    for name, target in _LAUNCHERS.items():
        module_name = target.partition(":")[0]
        if importlib.util.find_spec(module_name) is not None:
            available.append(name)
    return tuple(available)


def get_launcher(provider: str) -> SandboxLauncher:
    """
    Resolve a provider name to a launcher instance.

    :param provider: Provider name, e.g. ``"modal"``.
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
    module_name, _, class_name = target.partition(":")
    module = importlib.import_module(module_name)
    launcher_cls: type[SandboxLauncher] = getattr(module, class_name)
    return launcher_cls()
