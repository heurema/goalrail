"""Background daemon entry point for auto-launched host processes.

Spawned by ``_ensure_host_daemon`` in ``cli.py`` when ``run`` /
``claude`` / ``codex`` register this machine as a host. Runs the same
:class:`HostProcess` loop as ``goalrail host``.

Two modes:

- ``--server <url>``: connect to an existing (remote or local) Goalrail server.
- ``--local``: this daemon owns a local Goalrail server — start (or reuse) a
  persistent background ``goalrail server`` on loopback and connect to
  it. The CLI discovers the resulting URL via the local-server pidfile.
"""

from __future__ import annotations

import argparse
import logging


def main() -> None:
    """Parse args and run the host process.

    Exactly one of ``--server <url>`` or ``--local`` must be given. In
    ``--local`` mode the daemon starts/reuses the background local AP
    server itself and connects to that.

    :returns: None.
    :raises SystemExit: If neither / both of ``--server`` and ``--local``
        are provided.
    """
    parser = argparse.ArgumentParser(
        description="Background host daemon",
    )
    parser.add_argument(
        "--server",
        default=None,
        help="AP server URL to connect to (remote or local).",
    )
    parser.add_argument(
        "--local",
        action="store_true",
        help="Start (or reuse) a local Goalrail server and connect to it.",
    )
    args = parser.parse_args()

    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s [%(name)s] %(message)s",
    )

    if args.local == bool(args.server):
        # Both or neither — the CLI always passes exactly one; fail loud.
        parser.error("exactly one of --server <url> or --local is required")

    if args.local:
        # The daemon owns the local server: start/reuse it, then connect.
        from goalrail.host.local_server import ensure_local_goalrail_server

        server_url = ensure_local_goalrail_server().url
    else:
        server_url = args.server

    from goalrail.host.connect import run_host_process

    run_host_process(server_url=server_url)


if __name__ == "__main__":
    main()
