"""Per-turn LLM judge for advisor v3 brain-model selection.

The runner-side cost advisor (:mod:`omnigent.runner.cost_advisor`)
drives a :class:`Judge` over every user turn. This module ships the
production judge :class:`LLMJudge`: on EACH turn it makes ONE cheap,
low-token, strict-JSON LLM call and maps the result to a SINGLE
:class:`~omnigent.cost_plan.AdvisorVerdict` (or ``None``) sizing the
turn's difficulty to a model for the orchestrator's OWN brain.

The judge design and rubric craft are ported from an earlier unmerged
``cost_control_judge.py``, which RESPAWNED the brain harness
to re-tier a session once; advisor v3 reuses the one-shot strict-JSON
call and few-shot rubric shape but re-decides the contract: a per-turn
single-verdict for the brain (not a once-per-session harness choice, and
not the v2 tier partition over sub-agent dispatches).

Firm contract decisions (not re-litigated):

- **Per-turn, single verdict.** One brain runs the whole turn, so the
  judge picks ONE model. A mixed-difficulty query takes the MAX tier its
  parts need — the brain handles the whole turn at that level.
- **Conversational turns → ``None``.** "ok", "thanks", "continue",
  "what's the status" produce no verdict; the prior turn's selection (if
  any) keeps standing. The null verdict is a first-class JSON outcome,
  not an error.
- **Difficulty → tier.** Difficult coding / architecture / tricky
  debugging → expensive; medium knowledge work → medium; trivial →
  cheap. The rubric carries few-shot examples for each.
- **Tier model, clamped.** The verdict's model is drawn from the chosen
  tier's configured list; a strayed pin is clamped to ``tiers[tier][0]``
  with a warning.
- **Fail-open.** A broken judge (LLM error, timeout, malformed output
  after one retry, unknown tier) returns ``None`` and logs a warning; it
  NEVER fails or blocks the user's turn in any mode.

The judge model defaults to the FIRST model of the cheapest configured
tier and is overridable via ``cost_optimize.advisor_model``. The call
goes through the generic multi-provider
:class:`~omnigent.llms.client.Client`, which the runner already depends
on.
"""

from __future__ import annotations

import json
import logging
from collections.abc import AsyncIterator, Mapping
from dataclasses import dataclass
from typing import Any, Protocol

from omnigent.cost_plan import (
    TIER_ORDER,
    AdvisorVerdict,
    tier_rank,
)
from omnigent.llms.types import MessageOutput, Response, ResponseStreamEvent

_logger = logging.getLogger(__name__)

# Marker key (inside ``executor.config.cost_optimize``) overriding the
# judge model; absent => the cheapest configured tier's first model.
ADVISOR_MODEL_KEY = "advisor_model"

# Judge call resilience: one cheap, low-token, fast call, retried once
# on any error before the judge gives up and returns None (fail-open).
_JUDGE_MAX_TOKENS = 512
_JUDGE_TIMEOUT_S = 30
_JUDGE_ATTEMPTS = 2


class LLMClientLike(Protocol):
    """
    Structural view of the one LLM-client method the judge calls.

    Matches :class:`omnigent.llms.client.Client`'s ``responses.create``
    surface; declared structurally so tests can pass a scripted stub that
    returns canned :class:`~omnigent.llms.types.Response` objects without
    standing up the real multi-provider client.
    """

    @property
    def responses(self) -> _ResponsesLike:
        """:returns: The namespace exposing ``create``."""
        ...


class _ResponsesLike(Protocol):
    """Structural view of ``client.responses`` (the ``create`` method).

    The signature matches the subset of
    :meth:`omnigent.llms.client._ResponsesNamespace.create` the judge
    uses, so the real :class:`~omnigent.llms.client.Client` satisfies
    this protocol structurally (and the scripted test stub does too).
    """

    async def create(  # type: ignore[explicit-any]  # mirrors Client.responses.create's Any-typed input/kwargs
        self,
        *,
        input: list[dict[str, Any]],
        model: str,
        connection_params: dict[str, str] | None = None,
        timeout: int | None = None,
        **kwargs: Any,
    ) -> Response | AsyncIterator[ResponseStreamEvent]:
        """
        Make one LLM call.

        Return type widened to match the real
        :class:`~omnigent.llms.client.Client` (which also streams); the
        judge always calls non-streaming, so :meth:`LLMJudge.judge`
        asserts the result is a :class:`Response`.

        :param input: Responses-API input items (one user message).
        :param model: The judge model id, e.g. ``"databricks-claude-haiku-4-5"``.
        :param connection_params: Per-provider connection overrides, or
            ``None`` for adapter defaults.
        :param timeout: Request timeout in seconds, or ``None``.
        :param kwargs: Remaining provider kwargs, e.g. ``max_tokens``.
        :returns: A :class:`Response` (non-streaming), structurally also
            an async event iterator on the real client.
        """
        ...


@dataclass(frozen=True)
class _JudgeConfig:
    """
    Resolved inputs for one :class:`LLMJudge`.

    :param tiers: Models-only tier catalog, e.g.
        ``{"cheap": ("m1",), "expensive": ("m2",)}`` — both the menu the
        judge picks a model from and the clamp source for a strayed pin.
    :param judge_model: The model the judge call itself runs on, e.g.
        ``"databricks-claude-haiku-4-5"``.
    :param connection: Per-provider connection overrides for the judge
        call, e.g. ``{"base_url": ..., "api_key": ...}``; ``None`` uses
        adapter defaults.
    :param request_timeout: Judge-call timeout in seconds.
    """

    tiers: dict[str, tuple[str, ...]]
    judge_model: str
    connection: dict[str, str] | None
    request_timeout: int


class LLMJudge:
    """
    Per-turn LLM judge: one call → single brain-model verdict | None.

    Implements the :class:`omnigent.runner.cost_advisor.Judge` protocol.
    Each :meth:`judge` call asks the judge to size the current query's
    difficulty to one tier and pick one model from that tier's
    configured catalog. Conversational turns yield ``None`` (prior
    selection stands); a broken judge also yields ``None`` (fail-open) —
    it never raises into the turn.
    """

    def __init__(self, config: _JudgeConfig, client: LLMClientLike) -> None:
        """
        Bind the judge to its resolved config and an LLM client.

        :param config: Resolved judge inputs (catalog, model, connection,
            timeout). Built by :func:`build_llm_judge`.
        :param client: The LLM client the judge calls; the real
            :class:`omnigent.llms.client.Client` in production, a
            scripted stub in tests.
        """
        self._config = config
        self._client = client

    async def judge(self, *, query: str, turn_anchor: str) -> AdvisorVerdict | None:
        """
        Judge one user turn into a single brain-model verdict, or ``None``.

        :param query: The turn's user message text, e.g. ``"refactor the
            auth flow"``. Empty / whitespace-only queries are treated as
            conversational (``None``) without an LLM call.
        :param turn_anchor: Caller-sampled anchor stamped onto the
            verdict (item id or ISO timestamp); the judge never reads the
            clock.
        :returns: The brain-model verdict (``applied=False`` — the
            advisor sets it when it applies the verdict), or ``None`` for
            a conversational turn (judge said so) OR a judge failure
            (fail-open) — in both cases the prior selection stays.
        """
        if not query.strip():
            return None
        parsed = await self._invoke_judge(query)
        if parsed is None:
            return None
        return self._verdict_from_parsed(parsed, turn_anchor)

    async def _invoke_judge(self, query: str) -> dict[str, Any] | None:  # type: ignore[explicit-any]  # parsed JSON verdict
        """
        Make the judge LLM call, retrying once, and parse the response.

        :param query: The user query to classify.
        :returns: The parsed JSON dict (``{"tier": ..., "model": ...}``
            or ``{"tier": null}``), or ``None`` when the call errored or
            the output was unparseable after one retry — both are judge
            failures the caller treats as "no verdict this turn".
        """
        prompt = self._build_prompt(query)
        last_error: Exception | None = None
        for _attempt in range(_JUDGE_ATTEMPTS):
            try:
                resp = await self._client.responses.create(
                    input=[
                        {
                            "role": "user",
                            "content": [{"type": "input_text", "text": prompt}],
                        }
                    ],
                    model=self._config.judge_model,
                    connection_params=self._config.connection,
                    timeout=self._config.request_timeout,
                    max_tokens=_JUDGE_MAX_TOKENS,
                )
                # The judge never streams; assert rather than branch so a
                # client regression surfaces as a (caught) failure, not a
                # silent wrong path.
                assert isinstance(resp, Response), (
                    f"judge expected Response, got {type(resp).__name__}"
                )
                return _parse_json_object(_extract_assistant_text(resp))
            except Exception as exc:  # noqa: BLE001 — fail-open: any judge error → no verdict
                last_error = exc
        _logger.warning(
            "cost_judge: judge failed after %d attempt(s) (%s); running turn unadvised",
            _JUDGE_ATTEMPTS,
            last_error,
        )
        return None

    def _verdict_from_parsed(  # type: ignore[explicit-any]  # parsed JSON dict
        self,
        parsed: dict[str, Any],
        turn_anchor: str,
    ) -> AdvisorVerdict | None:
        """
        Turn a parsed judge response into an :class:`AdvisorVerdict`.

        :param parsed: The parsed JSON dict from :func:`_parse_json_object`.
        :param turn_anchor: Anchor to stamp onto the verdict.
        :returns: The verdict (``applied=False``), or ``None`` when the
            response is the conversational null marker OR names an
            unknown tier (fail-open, logged).
        """
        tier = parsed.get("tier")
        if tier is None:
            return None
        if not isinstance(tier, str) or tier not in self._config.tiers:
            _logger.warning(
                "cost_judge: verdict named unknown/unconfigured tier %r; running turn unadvised",
                tier,
            )
            return None
        model = self._clamp_model(parsed.get("model"), tier)
        rationale = parsed.get("rationale")
        if not isinstance(rationale, str) or not rationale:
            rationale = f"LLM judge sized this turn to the {tier} tier"
        # applied is the advisor's decision (optimize vs advise vs user
        # pin), not the judge's; the judge always reports it unapplied.
        return AdvisorVerdict(
            tier=tier,
            model=model,
            applied=False,
            rationale=rationale,
            turn_anchor=turn_anchor,
        )

    def _clamp_model(self, raw_model: Any, tier: str) -> str:  # type: ignore[explicit-any]  # parsed JSON value
        """
        Resolve the verdict's model pin to a model in the named tier.

        :param raw_model: The judge's ``model`` value.
        :param tier: The verdict's (already-validated) tier name.
        :returns: ``raw_model`` when it is a string in ``tiers[tier]``;
            otherwise ``tiers[tier][0]`` (logged), so a hallucinated or
            out-of-tier pin degrades to the tier's canonical model rather
            than failing the turn.
        """
        tier_models = self._config.tiers[tier]
        if isinstance(raw_model, str) and raw_model in tier_models:
            return raw_model
        _logger.warning(
            "cost_judge: judge pinned model %r outside tier %r %s; clamping to %r",
            raw_model,
            tier,
            tier_models,
            tier_models[0],
        )
        return tier_models[0]

    def _build_prompt(self, query: str) -> str:
        """
        Assemble the strict-JSON judge prompt for one query.

        :param query: The user query to classify.
        :returns: The full single-message prompt string.
        """
        return _build_judge_prompt(self._config.tiers, query)


# ── Prompt (module-level so it is reviewable) ─────────────────────────────────

# Rubric ported and rewritten from the earlier judge's _build_judge_prompt for
# advisor v3: a per-turn SINGLE tier+model verdict for the orchestrator's
# own brain (not a per-session harness choice, not a sub-agent partition),
# an explicit null verdict for conversational turns, the tier menu
# inlined, a concrete-pin requirement, and few-shot difficulty examples.
_JUDGE_RUBRIC = """\
You are a cost-control router for an AI agent orchestrator. The \
orchestrator runs the WHOLE of this user turn on a single "brain" model. \
Your job: size the turn's difficulty and pick the cheapest model tier \
that can do it WELL, so trivial work doesn't run on an expensive model \
and hard work doesn't run on a weak one.

Tiers, cheapest first, and the models available in each:
{tier_menu}

How to size difficulty:
- expensive: genuinely hard, multi-step, or high-stakes engineering — \
deep refactors, architecture/design, tricky debugging, security review, \
anything requiring careful multi-file reasoning.
- medium: ordinary knowledge work — focused code changes, writing or \
explaining a moderate amount, summarizing a document, routine analysis.
- cheap: trivial, mechanical, or very short tasks — a one-line lookup, a \
rename, a yes/no question, a tiny edit.

Rules:
1. The turn runs on ONE model. If the request mixes difficulties, pick \
the tier for the HARDEST part it contains.
2. Pin a CONCRETE model: choose one model id from the chosen tier's list \
above. Never invent a model id.
3. If the message is purely CONVERSATIONAL — an acknowledgement, thanks, \
a status check, "continue", small talk, or anything with no real work to \
do — return the null verdict so the prior selection stays in force.

Examples:
- "Refactor the auth flow to use the new token store and update all \
callers" -> expensive (multi-file refactor).
- "Summarize what this 200-line module does" -> medium (moderate \
knowledge work).
- "What's the capital of France?" -> cheap (trivial lookup).
- "ok, sounds good" -> null (conversational).
- "what's the status?" -> null (conversational).

Respond with ONLY a JSON object, no prose, no code fences, in one of \
these two shapes:

Real work:
{{"tier": "<tier name>", "model": "<model id from that tier>", \
"rationale": "<one sentence>"}}

Conversational / nothing to do:
{{"tier": null}}

User request:
{user_request}\
"""


def _build_judge_prompt(tiers: Mapping[str, tuple[str, ...]], query: str) -> str:
    """
    Render the judge prompt with the tier menu and query inlined.

    :param tiers: Models-only tier catalog, rendered cheapest-first as
        the model menu the judge must pin from.
    :param query: The user query to classify.
    :returns: The full prompt string.
    """
    ordered = sorted(
        (t for t in tiers if t in TIER_ORDER),
        key=tier_rank,
    )
    tier_menu = "\n".join(f"- {tier}: {', '.join(tiers[tier])}" for tier in ordered)
    return _JUDGE_RUBRIC.format(tier_menu=tier_menu, user_request=query)


def _extract_assistant_text(resp: Response) -> str:
    """
    Concatenate the assistant text out of a non-streaming Response.

    Uses the real :class:`MessageOutput` type the client returns so the
    isinstance gate matches production; a response with no assistant text
    raises (caught upstream as a judge failure).

    :param resp: The :class:`Response` from ``responses.create``.
    :returns: The concatenated assistant text, e.g. ``'{"tier": null}'``.
    :raises ValueError: When the response carries no assistant text.
    """
    parts: list[str] = []
    for item in resp.output:
        if not isinstance(item, MessageOutput):
            continue
        for content_part in item.content:
            parts.append(content_part.text)
    if not parts:
        raise ValueError("judge response contained no assistant text")
    return "".join(parts)


def _parse_json_object(text: str) -> dict[str, Any]:  # type: ignore[explicit-any]  # parsed JSON verdict
    """
    Parse the judge's assistant text into a JSON object.

    Strips a Markdown code fence if the model wrapped its JSON in one
    (common despite the JSON-only instruction), then parses. A non-JSON
    body or a non-object root raises (caught upstream as a judge failure
    → ``None``, fail-open).

    :param text: Raw assistant text, e.g. ``'{"tier": null}'`` or a
        fenced ``'```json\\n{...}\\n```'``.
    :returns: The parsed JSON dict.
    :raises ValueError: When *text* is not a JSON object.
    """
    stripped = text.strip()
    if stripped.startswith("```"):
        lines = stripped.splitlines()
        if lines and lines[0].startswith("```"):
            lines = lines[1:]
        if lines and lines[-1].startswith("```"):
            lines = lines[:-1]
        stripped = "\n".join(lines).strip()
    try:
        parsed = json.loads(stripped)
    except json.JSONDecodeError as exc:
        raise ValueError(f"judge output is not valid JSON: {exc.msg}") from exc
    if not isinstance(parsed, dict):
        raise ValueError(f"judge JSON root must be an object, got {type(parsed).__name__}")
    return parsed


def _resolve_judge_model(  # type: ignore[explicit-any]  # executor_config is a YAML-shaped dict
    tiers: Mapping[str, tuple[str, ...]],
    executor_config: Mapping[str, Any] | None,
) -> str:
    """
    Resolve the model the judge call itself runs on.

    Precedence: the ``cost_optimize.advisor_model`` marker override, then
    the FIRST model of the cheapest configured tier (a cheap judge for a
    cheap decision).

    :param tiers: Models-only tier catalog (already validated non-empty
        for configured tiers).
    :param executor_config: The spec's ``executor.config`` dict (carries
        the ``cost_optimize`` marker), or ``None``.
    :returns: The judge model id, e.g. ``"databricks-claude-haiku-4-5"``.
    :raises ValueError: When no configured tier has any model (the
        catalog is unusable — config validation should have caught this).
    """
    marker = (executor_config or {}).get("cost_optimize")
    if isinstance(marker, Mapping):
        override = marker.get(ADVISOR_MODEL_KEY)
        if isinstance(override, str) and override:
            return override
    for tier in sorted((t for t in tiers if t in TIER_ORDER), key=tier_rank):
        if tiers[tier]:
            return tiers[tier][0]
    raise ValueError("cost_optimize tiers have no models; cannot resolve a judge model")


@dataclass(frozen=True)
class _RoutedJudgeCall:
    """
    The judge call's provider routing after Databricks normalization.

    :param model: The model to hand to the generic client, e.g.
        ``"databricks/databricks-claude-haiku-4-5"``.
    :param connection: Connection overrides for the call, e.g.
        ``{"base_url": "https://…/serving-endpoints", "api_key": "…"}``,
        or ``None`` for adapter defaults.
    """

    model: str
    connection: dict[str, str] | None


def _resolve_workspace_creds(profile: str | None) -> Any:  # type: ignore[explicit-any]  # WorkspaceCreds (lazy import keeps module import side-effect free)
    """
    Indirection over Databricks credential resolution for testability.

    :param profile: Profile name from ``~/.databrickscfg``, or ``None``
        for env / DEFAULT-section resolution.
    :returns: The resolved
        :class:`~omnigent.runtime.credentials.databricks.WorkspaceCreds`.
    :raises OSError: When no usable credentials exist.
    """
    from omnigent.runtime.credentials.databricks import resolve_databricks_workspace

    return resolve_databricks_workspace(profile)


def _route_databricks_judge_model(
    judge_model: str,
    connection: dict[str, str] | None,
    databricks_profile: str | None,
) -> _RoutedJudgeCall:
    """
    Route a Databricks judge model through the ``databricks`` adapter.

    A bare ``databricks-*`` id carries no provider prefix, so the generic
    client would route it to the default ``openai`` adapter
    (``api.openai.com``) and the judge would fail open on EVERY turn —
    exactly the model shape polly's shipped tier catalog uses. Prefix the
    model ``databricks/`` so it reaches the Databricks adapter, and (when
    the caller passed no explicit connection) resolve the gateway
    host/token from *databricks_profile* — the same profile the brain's
    claude-sdk gateway routing resolves. A failed credential resolution
    leaves the connection ``None`` (the adapter then auto-resolves from
    ambient ``DATABRICKS_CONFIG_PROFILE`` / DEFAULT config); the judge
    stays fail-open either way.

    :param judge_model: The resolved judge model id, e.g.
        ``"databricks-claude-haiku-4-5"`` (returned unchanged when it is
        not a Databricks id).
    :param connection: Explicit connection overrides from the spec, or
        ``None`` (resolve from *databricks_profile*).
    :param databricks_profile: The Databricks profile the brain's gateway
        routing uses, e.g. ``"my-workspace"``, or ``None`` (ambient
        resolution).
    :returns: The routed model + connection for the judge call.
    """
    if not judge_model.startswith(("databricks-", "databricks/")):
        return _RoutedJudgeCall(model=judge_model, connection=connection)
    if not judge_model.startswith("databricks/"):
        judge_model = f"databricks/{judge_model}"
    if connection is None:
        try:
            creds = _resolve_workspace_creds(databricks_profile)
        except OSError as exc:
            _logger.warning(
                "cost_judge: could not resolve Databricks credentials "
                "(profile=%r): %s; judge call will use ambient adapter defaults",
                databricks_profile,
                exc,
            )
            return _RoutedJudgeCall(model=judge_model, connection=None)
        connection = {
            "base_url": creds.host.rstrip("/") + "/serving-endpoints",
            "api_key": creds.token,
        }
    return _RoutedJudgeCall(model=judge_model, connection=connection)


def build_llm_judge(  # type: ignore[explicit-any]  # executor_config is a YAML-shaped dict
    *,
    tiers: dict[str, tuple[str, ...]],
    executor_config: Mapping[str, Any] | None,
    connection: dict[str, str] | None,
    databricks_profile: str | None = None,
    client: LLMClientLike | None = None,
) -> LLMJudge:
    """
    Construct an :class:`LLMJudge` from advisor config.

    The one entry point the runner-side advisor calls to wire the
    production judge; it resolves the judge model and builds the generic
    LLM client lazily so importing this module is side-effect free.

    :param tiers: Models-only tier catalog from the parsed advisor config.
    :param executor_config: The spec's ``executor.config`` dict, read for
        the ``advisor_model`` override.
    :param connection: Per-provider connection overrides for the judge
        call (the orchestrator's own connection), or ``None``.
    :param databricks_profile: The Databricks profile the brain's gateway
        routing resolves for this spec (see
        :func:`omnigent.runner.cost_advisor._databricks_profile_for_spec`),
        or ``None``. Used only when the judge model is a Databricks id and
        *connection* is ``None``.
    :param client: LLM client override; ``None`` builds the real
        :class:`omnigent.llms.client.Client`. Tests pass a scripted stub —
        the Databricks judge-model routing (provider prefix + credential
        resolution) applies only on the real-client path, since an
        injected stub makes no provider call and credential I/O would be a
        pure side effect in unit tests.
    :returns: The wired judge.
    :raises ValueError: When no judge model can be resolved from *tiers*.
    """
    judge_model = _resolve_judge_model(tiers, executor_config)
    effective_client: LLMClientLike
    if client is not None:
        effective_client = client
    else:
        routed = _route_databricks_judge_model(judge_model, connection, databricks_profile)
        judge_model, connection = routed.model, routed.connection
        from omnigent.llms.client import Client

        effective_client = Client()
    config = _JudgeConfig(
        tiers=tiers,
        judge_model=judge_model,
        connection=connection,
        request_timeout=_JUDGE_TIMEOUT_S,
    )
    return LLMJudge(config, effective_client)


def resolve_advisor_mode(spec_mode: str, override: str | None) -> str | None:
    """
    Resolve the effective advisor mode for a turn.

    Precedence: per-session override > spec marker. The toggle is named
    "Cost Optimized", so turning it ON ESCALATES to optimize (apply the
    verdict) even on an advise-default spec — that is the shadow→apply
    rollout lever. Turning it OFF disables the advisor for the session.

    :param spec_mode: The mode the spec marker configured, one of
        :data:`~omnigent.cost_plan.ADVISOR_MODES` (``"advise"`` /
        ``"optimize"``). The marker parser rejects anything else, so a
        present marker always has a real mode here.
    :param override: The session's ``cost_control_mode_override`` —
        ``"on"`` (escalate to optimize), ``"off"`` (disable for this
        session), or ``None`` / absent (defer to *spec_mode*).
    :returns: The effective mode (``"advise"`` / ``"optimize"``), or
        ``None`` when the advisor is off this turn (no judge call).
    """
    if override == "off":
        return None
    if override == "on":
        # "on" = apply: escalate advise→optimize so the user toggle has the
        # effect its "Cost Optimized" name implies.
        return "optimize"
    # null / absent / any unexpected value: defer to the spec marker.
    return spec_mode


__all__ = [
    "LLMJudge",
    "build_llm_judge",
    "resolve_advisor_mode",
]
