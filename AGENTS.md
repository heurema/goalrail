# Goalrail Agent Instructions

## Intent First

- Treat owner statements and verified repository facts as evidence, not as an automatically correct interpretation.
- For every significant change, create or update a versioned Intent Snapshot before compiling a proposal.
- Treat OpenSpec exploration as evidence gathering and ambiguity reduction. It may propose candidate intent items, but it cannot confirm them on the owner's behalf.
- When exploration changes the understood outcome or boundary, update `intent.md` before proposal, specs, design, or tasks.
- Keep semantic intent to three groups: desired outcomes, non-goals, and observable success signals.
- A model-produced snapshot is `candidate` until the owner actively verifies it. Silence, continuation, or lack of objection is not confirmation.
- If a material ambiguity remains, stop before proposal or implementation and ask the smallest question that resolves it.
- A material intent amendment creates a new version. Do not repair history by rewriting prior evidence.
- Confirmed intent describes the requested result; it never grants permission for tools, credentials, writes, deployments, publications, or other effects.

## Artifact Language and Owner Confirmation

- Write durable repository documentation and development artifacts in English unless a specific external audience requires another language. Prefer concise, explicit, model-legible Markdown with stable identifiers and provenance over conversational prose or provider-specific prompt tricks.
- Before asking the owner to confirm a candidate Intent Snapshot, present a separate plain-language view in the resolved owner language. Resolve it in this order: an explicit owner preference, an explicit project or workflow preference, the language of the current conversation, then English as fallback. Never hard-code Russian, English, or another language as the universal confirmation language.
- In that resolved language, describe what the owner will do and see, what Goalrail will do, the important boundaries, and how success will be recognizable.
- The owner-facing view must faithfully cover every material outcome, non-goal, and success signal in the exact candidate version. Omit schema IDs, internal structure, and implementation terminology unless they change the owner's experience or decision, or the owner asks for technical detail.
- Ask one exact confirmation question that names the candidate version. The owner-facing view is a derived confirmation surface, not a fourth semantic intent group or a replacement for the versioned English artifact.

## OpenSpec

- Use the project-local `goalrail-intent` schema for Goalrail changes.
- OpenSpec 1.6.0 currently resolves the repository default as `spec-driven` in `new change`, even though `openspec/config.yaml` names the custom schema. Always pass `--schema goalrail-intent` explicitly.
- Take the invocation from `gr doctor`, which reports the compiler this machine would actually run. Where authorized setup has installed the pinned bundle, that is the bundle's own runtime and compiler entrypoint — fetched once, pinned by digest, verified per file, and resolving nothing. Where no bundle is installed it is the stock command:

  ```sh
  OPENSPEC_TELEMETRY=0 npx --yes @fission-ai/openspec@1.6.0 new change <name> --schema goalrail-intent
  ```

- Use whichever of those two the diagnosis reported, with the same telemetry-disabled prefix, for status, instructions, validation, and schema commands. Do not resolve the package again when a verified bundle is installed: that returns every planning command to the registry the bundle exists to stop consulting.
- Follow the artifact dependency order reported by `openspec status`; do not bypass `intent.md` or generate a proposal from candidate intent.
- OpenSpec is a replaceable development-time compiler/provider. Do not leak OpenSpec types into Goalrail's canonical domain.
- Planning completion is not permission to apply, commit, push, publish, deploy, or start a real canary. Each action keeps its normal owner gate.

## Scope Discipline

- Prefer the smallest reversible vertical slice and stop at its owner gate.
- Do not silently remove retained target architecture layers merely because the current slice does not implement them.
- Do not introduce workflow engines, authorization systems, sandboxes, dashboards, databases, or provider-specific abstractions before a current requirement and measured need justify them.
