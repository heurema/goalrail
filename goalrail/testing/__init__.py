"""Test-environment safety helpers for the Goalrail suite.

Houses additive guardrails that assert a test run is pointed at
throwaway resources (a tmp/in-memory SQLite DB, no dev/prod ports)
rather than a developer's real local instance. See
:mod:`goalrail.testing.guardrails`.
"""

from __future__ import annotations
