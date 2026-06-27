"""Tests for :mod:`goalrail.host._daemon_entry`."""

from __future__ import annotations

import subprocess
import sys


def test_daemon_entry_help_uses_goalrail_product_name() -> None:
    """Daemon argparse help should not expose the old public product name."""
    result = subprocess.run(
        [sys.executable, "-m", "goalrail.host._daemon_entry", "--help"],
        check=True,
        capture_output=True,
        text=True,
        timeout=20,
    )

    assert "local Goalrail server" in result.stdout
