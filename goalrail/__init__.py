"""Goalrail: A declarative agent authoring and runtime framework."""

# Some libraries we transitively depend on call ``hashlib.md5()``
# without ``usedforsecurity=False`` for non-security content hashes.
# On FIPS-enabled OpenSSL builds the bare md5 constructor raises
# ``ValueError: digital envelope routines: EVP_DigestInit_ex disabled
# for FIPS``, which crashes the entire framework boot. Patch md5 here,
# at the package import boundary, so every consumer — including
# subprocesses spawned via ``-m goalrail`` in e2e tests — picks up
# the fix before any dependency import touches it. The flag is the
# standard Python 3.9+ opt-out for non-security md5 calls and is a
# harmless no-op on non-FIPS hosts.
import hashlib as _fips_safe_hashlib

_fips_safe_orig_md5 = _fips_safe_hashlib.md5


def _fips_safe_md5(*args, **kwargs):  # type: ignore[no-untyped-def]
    kwargs.setdefault("usedforsecurity", False)
    return _fips_safe_orig_md5(*args, **kwargs)


_fips_safe_hashlib.md5 = _fips_safe_md5  # type: ignore[assignment]

# Initialize environment compatibility hooks before submodules read process
# environment. Goalrail currently uses only canonical ``GOALRAIL_*`` names, so
# this hook is a no-op retained for a stable import path.
from goalrail._env_compat import mirror_legacy_env as _mirror_legacy_env  # noqa: E402

_mirror_legacy_env()

from goalrail.inner.datamodel import (  # noqa: E402 — must follow md5 patch
    AgentDef,
    Connection,
    Credentials,
    History,
    Memory,
    MemoryConfig,
    Message,
    ParamDef,
    SessionState,
)
from goalrail.inner.executor import (  # noqa: E402 — must follow md5 patch
    Executor,
    ExecutorConfig,
    ExecutorError,
    ExecutorEvent,
    TextChunk,
    ToolCallComplete,
    ToolCallRequest,
    TurnCancelled,
    TurnComplete,
)
from goalrail.inner.policies import (  # noqa: E402 — must follow md5 patch
    FunctionPolicy,
    Policy,
    PolicyAction,
    PolicyResult,
    PromptPolicy,
)
from goalrail.inner.tools import (  # noqa: E402 — must follow md5 patch
    AgentTool,
    CancellableFunctionTool,
    FunctionTool,
    HandoffTool,
    InheritedTool,
    MCPTool,
    SkillTool,
    Tool,
)

try:
    from goalrail.inner.claude_sdk_executor import ClaudeSDKExecutor
except ImportError:
    ClaudeSDKExecutor = None  # type: ignore[misc,assignment]
try:
    from goalrail.inner.open_responses_sdk import OpenResponsesExecutor
except ImportError:
    OpenResponsesExecutor = None  # type: ignore[misc,assignment]
try:
    from goalrail.inner.openai_agents_sdk_executor import OpenAIAgentsSDKExecutor
except ImportError:
    OpenAIAgentsSDKExecutor = None  # type: ignore[misc,assignment]
try:
    from goalrail.inner.codex_executor import CodexExecutor
except ImportError:
    CodexExecutor = None  # type: ignore[misc,assignment]
from goalrail.inner.loader import load_agent_def  # noqa: E402 — must follow md5 patch
from goalrail.inner.tracing import (  # noqa: E402 — must follow md5 patch
    disable_tracing,
    enable_tracing,
    is_tracing_enabled,
)

__all__ = [
    "AgentDef",
    "AgentTool",
    "CancellableFunctionTool",
    "ClaudeSDKExecutor",
    "CodexExecutor",
    "Connection",
    "Credentials",
    "Executor",
    "ExecutorConfig",
    "ExecutorError",
    "ExecutorEvent",
    "FunctionPolicy",
    "FunctionTool",
    "HandoffTool",
    "History",
    "InheritedTool",
    "MCPTool",
    "Memory",
    "MemoryConfig",
    "Message",
    "OpenAIAgentsSDKExecutor",
    "OpenResponsesExecutor",
    "ParamDef",
    "Policy",
    "PolicyAction",
    "PolicyResult",
    "PromptPolicy",
    "SessionState",
    "SkillTool",
    "TextChunk",
    "Tool",
    "ToolCallComplete",
    "ToolCallRequest",
    "TurnCancelled",
    "TurnComplete",
    "disable_tracing",
    "enable_tracing",
    "is_tracing_enabled",
    "load_agent_def",
]
