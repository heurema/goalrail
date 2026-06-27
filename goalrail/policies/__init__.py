"""
Policy building blocks — pure evaluators, no runtime state.

This package holds the pieces an agent author (or the parser)
reaches for when declaring / implementing a policy:

- :class:`EvaluationContext`, :class:`PolicyResult`,
  :class:`ElicitationRequest` — the data shapes that cross the
  evaluate boundary (see :mod:`goalrail.policies.types`).
- :class:`Policy` ABC (:mod:`goalrail.policies.base`) and
  the concrete subclass :class:`FunctionPolicy`.

The subclasses are pure in the important sense: they own no
mutable state across calls, do no DB I/O, and don't know about
conversations. State (label cache, conversation id,
write-through store) and orchestration (composition loop, ASK
parking, fail-closed) live in :mod:`goalrail.runtime.policies`.

Agent-author callables should import :class:`EvaluationContext`
and :class:`PolicyResult` from here, not from
``goalrail.spec.types`` — those are runtime evaluation
artifacts, not declarations that appear in a spec.
"""

from __future__ import annotations

from goalrail.policies.base import Policy
from goalrail.policies.function import (
    FunctionPolicy,
    resolve_function_policy,
)
from goalrail.policies.schema import (
    PolicyCallable,
    PolicyCallableWithConfig,
    PolicyEvent,
    PolicyResponse,
    StateUpdateEntry,
)
from goalrail.policies.types import (
    ElicitationRequest,
    EvaluationContext,
    PolicyLLMClient,
    PolicyResult,
)

__all__ = [
    "ElicitationRequest",
    "EvaluationContext",
    "FunctionPolicy",
    "Policy",
    "PolicyCallable",
    "PolicyCallableWithConfig",
    "PolicyEvent",
    "PolicyLLMClient",
    "PolicyResponse",
    "PolicyResult",
    "StateUpdateEntry",
    "resolve_function_policy",
]
