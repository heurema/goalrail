# Intent Snapshot

- **Intent ID:** intent-harness-init-v0
- **Version:** 1
- **Status:** candidate
- **Owner:** repository owner
- **Context Pack:** context-harness-init-v0 version 1
- **Run references:** none yet; one interactive observation of the shipping arrangement is planned and remains behind its own owner gate

## Source Evidence

- **SE-1 (owner):** `gr init` grows from marking a repository into installing the whole harness — the OpenSpec overlay with the Goalrail schema, the session-hook registration, and optional observability wiring.
- **SE-2 (owner):** `gr doctor` grows out of `gr health` into full diagnosis, including drift between a repository's copy of the schema and the canonical one, and including whether an update is available.
- **SE-3 (owner):** Updating becomes one command. The only version a user sees is Goalrail's own; OpenSpec compatibility is verified internally and its pin ships inside a Goalrail release; the user is never asked to reason about OpenSpec versions. Updating is never silent — seamless means one command, not zero.
- **SE-4 (owner):** The customization is an overlay, not a fork. The CLI stays stock; the canonical schema lives in the `gr` binary and is materialized into each repository; drift is detected by comparing digests rather than by trusting the copy.
- **SE-5 (owner):** Goalrail works fully with no observability backend — the local state root is the source of truth, and exporters are removable attachments. v0 wires only an instance the user already has; absence produces one non-nagging line. Hosted multi-tenant observability is deferred because traces carry prompts and user code.
- **SE-6 (owner):** Registration scope belongs to this work. Project scope is structurally better: registered per repository, the hook is never invoked in unrelated sessions at all, so the boundary is a property of where it is installed rather than of Goalrail's first act. `gr init` is its natural home, and for the second scaffold a separate connection step then has no reason to exist. The open approval question is answered here, in the arrangement that will ship, not by a throwaway registration in the owner's working configuration.
- **SE-7 (owner, this session):** This round produces planning artifacts only. The owner's working `~/.claude/settings.json` and `~/.codex/config.toml` are not to be touched; branch `codex/dogfood-run-v0` holds deferred work; commit, push, pull request, merge, archival, and any provider run each require a separate explicit instruction, recorded in `tasks.md` before the action.
- **SE-8 (repository):** `gr init` writes one marker file and nothing else, and the marker is git-ignored, so initialization is already a per-clone, per-user act (CTX-1, CTX-3).
- **SE-9 (repository):** `gr` has no version, no tag, and no release pipeline, so "an update is available" has neither a version to read nor a channel to query today (CTX-4).
- **SE-10 (repository):** The pinned invocation carries a recorded defect workaround — 1.6.0 resolves `new change` to the default schema unless `--schema goalrail-intent` is passed explicitly — so a repository holding only the overlay files still fails on its first change (CTX-5).
- **SE-11 (repository):** The dependency already owns the names `init`, `update`, and `doctor` with different meanings, and offers `schema validate` and `doctor` as reusable checks (CTX-6).
- **SE-12 (repository):** The dependency's own initialization writes provider prompt packs; the six it wrote into this repository call an unpinned executable without the schema argument, so they disagree with this project's invocation contract (CTX-7).
- **SE-13 (external):** The second scaffold documents no approval step for a hook configured in a settings file, documents `/hooks` as read-only, and gates its workspace trust dialog on permission rules and additional directories rather than on hooks (CTX-11, CTX-12).
- **SE-14 (external):** That scaffold's project scope is two different arrangements: a committed settings file shared with every teammate who opens the repository, and a per-user project file that is not shared (CTX-13).
- **SE-15 (repository):** The existing Langfuse adapter is a bounded read-only reader with no exporter, span emitter, or trace writer anywhere in the tree, so "the existing adapter is the exporter interface's first implementation" does not hold as stated (CTX-14).
- **SE-16 (repository):** `gr health` already reports per-scaffold attachment state with next actions, an explicit unverifiable list, and a distinct unreadable-configuration state, under a promoted requirement that forbids forging trust and requires naming what was not verified (CTX-16).
- **SE-17 (repository):** Printed remedies and promoted help text name `gr health` by name, so renaming the surface breaks advice the tool gives unless both move together and the old name keeps working (CTX-17).

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | One command installs the harness in one repository: `gr init` materializes the canonical OpenSpec overlay — the config naming the Goalrail schema, the schema, and its templates — from copies embedded in the `gr` binary, alongside the existing marker, and reports what it created, what it changed, what it left alone, and the exact pinned invocation the repository is now driven by. Repeating it changes nothing. | Run `gr init` in an empty directory, then validate the materialized overlay and create a change with the pinned CLI; run `gr init` again and diff the repository. | SE-1, SE-4, SE-8, SE-10, CTX-2, CTX-5 |
| OUT-2 | Initialization on a repository that already has OpenSpec adds the Goalrail schema and switches the config to it, touches no existing spec or change file, then validates and reports what changed. | Seed a repository with a stock OpenSpec root and one existing spec and change, run `gr init`, and confirm both survive byte-identical while the config names the Goalrail schema. | SE-1, SE-4, CTX-2 |
| OUT-3 | The session-hook registration happens inside `gr init`, per repository, for the scaffold whose settings layer allows it. It is written to that scaffold's per-user project settings file rather than a shared committed one, so initialization installs a hook into the session of the person who ran it and of nobody else. The user-scope connection command remains for the scaffold where the repository route is externally blocked; for the repository-scope scaffold, `gr connect` writes nothing and points at `gr init`. | Run `gr init` in a scratch repository and confirm the registration is in the per-user project settings file, that no user-level configuration was written, and that `gr connect` for that scaffold refuses and names `gr init`. | SE-6, SE-14, CTX-9, CTX-13 |
| OUT-4 | Removal stays complete and narrow: disconnection removes a repository-scope registration with no residue, leaves any entry it did not add untouched, and is safe to repeat. | Initialize, disconnect, and confirm no managed handler and no empty residue remain while a seeded foreign handler in the same file survives unchanged. | SE-6, CTX-9 |
| OUT-5 | The approval question is answered inside this work, for the arrangement that ships: the disclosure states what is actually established for that scaffold — documented as having no per-hook approval step, with its folder-level trust dialog covering permission rules rather than hooks — and one interactive session with the registered arrangement upgrades that from documented to observed, or records what is observed instead. Goalrail still never writes, computes, reproduces, or simulates a trust record. | Read the disclosure text against the documented evidence, then run one interactive session in a scratch repository initialized by `gr init` and record what the registration did and did not require. | SE-6, SE-13, CTX-10, CTX-11, CTX-12 |
| OUT-6 | `gr doctor` reports one diagnosis of the harness in this repository: per-scaffold attachment state as `gr health` reports it today, overlay presence and digest drift against the binary's canonical copy, whether the pinned OpenSpec invocation is usable, whether the installed `gr` carries a newer harness than this repository, and one line for observability. Every state that is not working names its next action; an optional facility that is simply absent produces one line and no next action. | Run `gr doctor` against a healthy repository, a repository with an edited template, one with no overlay, one whose recorded harness version is behind the binary, and one with no observability, and read each report. | SE-2, SE-4, SE-5, SE-16, CTX-2, CTX-16 |
| OUT-7 | A remedy the tool prescribes is a command the tool has: `gr health` keeps working as an alias that names its successor, and every printed remedy, help entry, and next action moves to `gr doctor` in the same change. | Run `gr health`, confirm it works and names `gr doctor`, and confirm no printed remedy names a command that does not exist. | SE-2, SE-17, CTX-17 |
| OUT-8 | `gr update` applies a harness update in one command: it rewrites the pinned invocation, re-materializes the overlay, re-validates with the pinned CLI, reports the version before and after together with every file it rewrote, and leaves the previous state recoverable. It never runs as a side effect of another command. | Initialize with an older recorded harness version, run `gr update`, read the report, and restore the previous state from what the command left behind. | SE-3, SE-4, CTX-2 |
| OUT-9 | `gr` carries a version of its own that the tool reports and the repository marker records, and OpenSpec's version stays internal: no command asks the user to reason about it, and the pin moves only inside a Goalrail change. | Read the version from the tool and from a freshly initialized repository's marker, and confirm no user-facing surface asks about an OpenSpec version. | SE-3, SE-9, CTX-4 |
| OUT-10 | Observability stays optional in the strong sense: no backend is a fully working configuration; an endpoint and keys already present in the environment or the state root are reported as configured; no credential is ever written into the repository. | Run `gr doctor` with and without observability configured, and confirm the repository holds no credential in either case. | SE-5, CTX-15 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | No network release check and no self-update of the `gr` binary. There is no tag, no release pipeline, and no channel to query, so `gr update` updates a repository's harness to what the installed `gr` already carries. Revisit when a release channel exists. | SE-9, CTX-4 |
| NG-2 | No exporter interface, span emitter, or trace writer, and no change to the Langfuse adapter. An exporter would be new code with no current caller; v0 observability is detection and reporting only. | SE-15, CTX-14 |
| NG-3 | No hosted or multi-tenant observability. | SE-5 |
| NG-4 | Do not fork, vendor, or re-implement the OpenSpec CLI, and do not materialize the dependency's provider prompt packs. The overlay is the schema, its templates, and the config — nothing that instructs a provider. | SE-4, SE-12, CTX-6, CTX-7 |
| NG-5 | Do not change the announcement text, the reserved escalation path, question retention, intent binding, the fail-quiet posture, or the wrapper lifecycle. | SE-1, CTX-10 |
| NG-6 | Do not register the repository-scope scaffold at user scope, do not write a shared committed settings file that would install hooks for teammates, and do not touch the owner's working `~/.claude/settings.json` or `~/.codex/config.toml` or branch `codex/dogfood-run-v0`. | SE-6, SE-7, SE-14 |
| NG-7 | Do not write, compute, reproduce, or simulate a scaffold trust record, and do not change how attachment trust is reported for the first scaffold. | SE-6, SE-16 |
| NG-8 | Do not delete a repository's committed harness files as part of disconnection. The overlay is version-controlled repository content; doctor reports its state and update repairs it, but removal stays the user's act. | SE-4, CTX-3 |
| NG-9 | Do not add a workflow engine, dashboard, database, service, credential store, or new dependency. | SE-1 |
| NG-10 | This snapshot authorizes no implementation, commit, push, pull request, merge, archival, provider run, or external effect. Each remains a separate owner gate, recorded before the action. | SE-7 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | A fresh directory becomes a working harness in one command. | A test runs `gr init` in an empty directory, then the pinned CLI validates the materialized overlay and creates a change with the Goalrail schema successfully. | SE-1, CTX-2, CTX-5 |
| SIG-2 | Initialization is idempotent. | A test runs `gr init` twice and compares every file byte for byte. | SE-1, CTX-1 |
| SIG-3 | An existing OpenSpec root survives initialization. | A test seeds a spec and a change, initializes, and asserts both files are byte-identical while the config names the Goalrail schema. | SE-4, CTX-2 |
| SIG-4 | Drift is detected and repaired rather than trusted. | A test edits one materialized template, asserts `gr doctor` names that file as drifted, runs `gr update`, and asserts the digest matches the binary's canonical copy again. | SE-4, CTX-2 |
| SIG-5 | The registration lands in the per-user project file and nowhere else. | A test runs initialization against a fake home and repository and asserts the per-user project settings file holds the managed handlers while the user-level configuration is untouched and unchanged. | SE-6, SE-14, CTX-13 |
| SIG-6 | Removal leaves no residue and no collateral. | A test initializes, seeds a foreign handler, disconnects, and asserts the foreign handler survives with its command and occurrence intact while no managed handler or empty container remains. | SE-6, CTX-9 |
| SIG-7 | The diagnosis distinguishes every state it claims to. | A test drives `gr doctor` through: not initialized, overlay missing, overlay drifted, pinned invocation unusable, attachment absent, attachment registered but unverified, harness behind the binary, observability absent, and fully working — and asserts a distinct state and next action for each, with no next action for the optional absent one. | SE-2, SE-16, CTX-16 |
| SIG-8 | No remedy names a command that does not exist. | A test asserts `gr health` still runs and names `gr doctor`, and that every remedy string the code prints resolves to a command the CLI accepts. | SE-17, CTX-17 |
| SIG-9 | An update is legible and reversible. | A test asserts the update report names the version before and after and every rewritten file, and that the previous overlay is recoverable from what the command left behind. | SE-3, CTX-2 |
| SIG-10 | Absent observability reads as fine, not as broken. | A test asserts that with no observability configured `gr doctor` reports the harness as working, prints one optional line, and prints no next action for it. | SE-5, CTX-15 |
| SIG-11 | The approval question leaves an evidence record. | One interactive session with the shipping registration is recorded under the change's evidence, stating what it establishes and what it does not, and the disclosure text is reconciled with it. | SE-6, CTX-10, CTX-11 |

## Ambiguities and Unknowns

| ID | Question | Evidence |
|---|---|---|
| AMB-1 | Whether the second scaffold, in an interactive session, runs a repository-scoped registered hook with no approval step. Documentation records no such gate, which is enough to choose the arrangement; the observation is what upgrades the disclosure from documented to observed. Both answers produce the same registration, so this does not block confirmation. | CTX-11, CTX-12, CTXQ-1 |
| AMB-2 | Whether a harnessed repository also needs a committed instruction file naming the pinned invocation and the explicit-schema workaround, or whether `gr init`'s report and `gr doctor` printing it is enough. This version answers "printing it is enough" and writes no instruction file. Revisit condition: an agent in a freshly initialized repository still guesses the invocation. | SE-10, SE-12, CTX-5, CTX-7 |

## Confirmation

- **Confirmed by:** pending
- **Confirmed at:** pending
- **Verification action:** pending — a plain-language view in the owner's language is presented for active confirmation of version 1, and silence or continuation does not confirm it
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
