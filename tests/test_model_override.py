"""Unit tests for ``goalrail/model_override.py``.

The validator guards a spawn boundary: a model override is persisted on
the session row and later becomes a ``--model`` argv element (native
CLIs) or a ``HARNESS_<H>_MODEL`` env var (SDK harnesses). These tests
pin the accepted charset, the loud rejections, and the harness-support
map that ``sys_session_send`` consults before forwarding an override.
"""

from __future__ import annotations

import pytest

from goalrail.model_override import (
    MODEL_OVERRIDE_MAX_LEN,
    canonical_model_spelling,
    harness_supports_model_override,
    model_family_mismatch,
    normalize_model_for_provider,
    validate_model_override,
)


@pytest.mark.parametrize(
    "value",
    [
        "anthropic/claude-sonnet-4-6",
        "claude-opus-4-8[1m]",
        "gpt-5.4-mini",
        "openai/gpt-4o",
        "us.anthropic.claude-sonnet-4-6",
        "gateway/openai/gpt-5-4",
        "vendor:tag",
        "o3",
    ],
)
def test_validate_model_override_accepts_real_id_shapes(value: str) -> None:
    """
    Every real-world model-id shape passes unchanged.

    A failure means the charset regressed and a legitimate dispatch
    (e.g. a bracketed context-window suffix) would be rejected.
    """
    assert validate_model_override(value) == value


def test_validate_model_override_strips_whitespace() -> None:
    """Surrounding whitespace is stripped, mirroring the PATCH path."""
    assert validate_model_override("  claude-opus-4-8  ") == "claude-opus-4-8"


@pytest.mark.parametrize(
    "value",
    [
        "",
        "   ",
        # Shell metacharacters must never reach a command line.
        "claude; rm -rf /",
        "claude && evil",
        "model`id`",
        'model"quoted"',
        "model with spaces",
        "model\nnewline",
        # Flag-shaped values could be parsed as a CLI option.
        "--dangerously-skip-permissions",
        "-flag",
        # Length cap.
        "a" * (MODEL_OVERRIDE_MAX_LEN + 1),
    ],
)
def test_validate_model_override_rejects_unsafe_values(value: str) -> None:
    """
    Empty, shell-shaped, flag-shaped, and oversized values fail loud.

    A pass-through here would let a model string smuggle argv/shell
    content across the harness spawn boundary.
    """
    with pytest.raises(ValueError):
        validate_model_override(value)


@pytest.mark.parametrize(
    "harness",
    [
        "claude-native",
        "codex-native",
        "claude-sdk",
        # The user-facing alias for claude-sdk must resolve too.
        "claude",
        "codex",
        "pi",
        "openai-agents",
        "cursor",
        "antigravity",
        "kiro-native",
        "native-kiro",
    ],
)
def test_harness_supports_model_override_for_plumbed_harnesses(harness: str) -> None:
    """
    Harnesses with --model / spawn-env plumbing report support.

    A False here would make ``sys_session_send`` reject a model for a
    harness that actually honors it.
    """
    assert harness_supports_model_override(harness) is True


@pytest.mark.parametrize(
    "harness",
    [
        # No model-override plumbing on the runner path: the persisted
        # value would be silently ignored.
        "unknown-harness",
        "totally-unknown",
        None,
    ],
)
def test_harness_supports_model_override_false_for_unplumbed(harness: str | None) -> None:
    """
    Unplumbed / unknown harnesses report no support.

    A True here would silently drop the orchestrator's model choice —
    exactly the failure mode the dispatch-time gate exists to prevent.
    """
    assert harness_supports_model_override(harness) is False


class TestModelFamilyMismatch:
    """model_family_mismatch enforces vendor families per harness."""

    @pytest.mark.parametrize(
        ("harness", "model"),
        [
            ("claude-native", "anthropic/claude-sonnet-4-6"),
            ("claude-sdk", "claude-opus-4-8"),
            ("codex-native", "openai/gpt-5-4"),
            ("codex", "gpt-5.1-codex"),
            ("openai-agents", "gpt-5.4-mini"),
            # openai-agents is multi-model like pi (a live SDK probe completed a
            # Claude tool-calling turn over the chat wire), so it accepts the
            # Claude / Kimi / Llama families a gateway may serve it.
            ("openai-agents", "anthropic/claude-sonnet-4-6"),
            ("openai-agents", "kimi-k2-6"),
            ("openai-agents", "meta-llama-3.3-70b-instruct"),
            # The "-sdk" / executor-type spellings canonicalize_harness
            # passes through must be multi-model too — an earlier change had
            # added them to the GPT-only set; a later change removes every
            # openai-agents spelling so none of them family-reject a non-GPT id.
            ("openai-agents-sdk", "anthropic/claude-sonnet-4-6"),
            ("openai-agents-sdk", "gpt-5.4-mini"),
            ("agents_sdk", "meta-llama-3.3-70b-instruct"),
            ("agents_sdk", "anthropic/claude-opus-4-8"),
            ("pi", "anthropic/claude-opus-4-8"),
            ("pi", "openai/gpt-5-4-mini"),
            ("pi", "meta-llama-3.3-70b-instruct"),
            # antigravity is Gemini-native: expected Gemini shapes pass, and
            # so do bare/ambiguous ids the SDK legitimately accepts (only the
            # Claude/GPT families are rejected below).
            ("antigravity", "gemini-3.5-flash"),
            ("antigravity", "gemini-2.5-flash"),
            ("agy", "gemini-3.5-flash"),
            ("google-antigravity", "gemini-2.5-flash"),
            ("antigravity", "gemini-2.5-pro"),
            ("kiro-native", "claude-sonnet-4.5"),
            ("native-kiro", "gpt-5.4-mini"),
        ],
    )
    def test_compatible_pairs_pass(self, harness: str, model: str) -> None:
        """A matching family (or a multi-model harness: pi, openai-agents) returns None."""
        assert model_family_mismatch(harness, model) is None

    @pytest.mark.parametrize(
        ("harness", "model", "expected_rule"),
        [
            ("claude-native", "openai/gpt-5-4", "only runs Claude models"),
            ("native-claude", "gpt-5.4", "only runs Claude models"),
            ("claude-sdk", "meta-llama-3.3-70b-instruct", "only runs Claude models"),
            ("codex-native", "anthropic/claude-sonnet-4-6", "only runs GPT models"),
            ("native-codex", "claude-opus-4-8", "only runs GPT models"),
            ("codex", "meta-llama-3.3-70b-instruct", "only runs GPT models"),
            # antigravity is Gemini-native: syntactically valid non-Gemini ids
            # must fail loud at the dispatch gate rather than be persisted as
            # model_override and land in HARNESS_ANTIGRAVITY_MODEL only to fail
            # later in the Gemini-native SDK path.
            ("antigravity", "gpt-5.4-mini", "Gemini-native"),
            ("antigravity", "anthropic/claude-sonnet-4-6", "Gemini-native"),
            ("antigravity", "claude-opus-4-8", "Gemini-native"),
            ("agy", "gpt-5.4-mini", "Gemini-native"),
            ("google-antigravity", "openai/gpt-5-4", "Gemini-native"),
        ],
    )
    def test_wrong_or_unknown_family_is_rejected(
        self, harness: str, model: str, expected_rule: str
    ) -> None:
        """Cross-family and undeterminable ids are rejected with the rule named.

        The alias case (``native-claude``) proves canonicalization is
        applied before the family lookup; the llama cases prove an
        undeterminable family fails loud on single-vendor harnesses
        rather than passing through to a gateway error after spawn. The
        ``agy`` / ``google-antigravity`` cases prove the antigravity rule
        applies after harness canonicalization too.
        """
        msg = model_family_mismatch(harness, model)
        assert msg is not None
        assert expected_rule in msg
        assert model in msg

    def test_rejection_names_both_multi_model_fallbacks(self) -> None:
        """Both single-vendor rejections name pi and openai-agents as multi-model fallbacks."""
        claude_msg = model_family_mismatch("claude-native", "openai/gpt-5-4")
        codex_msg = model_family_mismatch("codex-native", "anthropic/claude-sonnet-4-6")
        for msg in (claude_msg, codex_msg):
            assert msg is not None
            assert "pi" in msg
            assert "openai-agents" in msg


@pytest.mark.parametrize(
    "model",
    [
        "claude-opus-4-8",
        "gpt-5-4",
        "anthropic/claude-opus-4-8",
        "openai/gpt-4o",
        "us.anthropic.claude-sonnet-4-6",
        "claude-opus-4-8[1m]",
        "kimi-k2.6",
        "meta-llama-3.3-70b-instruct",
    ],
)
@pytest.mark.parametrize("provider_kind", ["key", "subscription", "gateway", "local", None])
def test_normalize_model_for_provider_preserves_model_ids(
    provider_kind: str | None, model: str
) -> None:
    """
    Provider-specific aliases are no longer supported.

    :param provider_kind: The provider kind under test.
    :param model: The requested model id.
    """
    assert normalize_model_for_provider(model, provider_kind) == model


@pytest.mark.parametrize(
    "model",
    [
        "anthropic/claude-haiku-4-5",
        "openai/gpt-5-4-mini",
        "claude-haiku-4-5",
        "meta-llama-3.3-70b-instruct",
        "anthropic/claude-opus-4-8[1m]",
        "openai/gpt-4o",
        "us.anthropic.claude-sonnet-4-6",
    ],
)
def test_canonical_model_spelling_preserves_model_ids(model: str) -> None:
    """
    The canonicalizer is now an identity function.

    :param model: The id to canonicalize.
    """
    assert canonical_model_spelling(model) == model


@pytest.mark.parametrize(
    ("harness", "model"),
    [
        # The family guard sees the PRE-normalization id; its verdict is
        # identical post-transform because the family token survives the
        # prefix in both directions.
        ("claude-native", "claude-opus-4-8"),
        ("codex-native", "gpt-5-4"),
    ],
)
def test_family_tokens_survive_identity_normalization(harness: str, model: str) -> None:
    """
    Guard-then-normalize ordering cannot change the family verdict.

    The dispatch gate documents family-guard-first (so errors quote the
    caller's exact id); this pins the identity-normalization property that
    keeps the order safe.
    """
    normalized = normalize_model_for_provider(model, "gateway")
    assert normalized == model
    assert model_family_mismatch(harness, model) is None
    assert model_family_mismatch(harness, normalized) is None
