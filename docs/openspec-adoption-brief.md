# OpenSpec adoption brief — owner discussion, 2026-08-01

Input for a later Context Pack. This is a discussion record, not a confirmed
intent: the change starts from its own Context Pack and its own owner
confirmation.

**Superseded on the same day it was written.** The adoption question it raises is
now carried by the change `openspec-adoption-v0`, whose Context Pack holds every
measurement below with a source and an observation time, and whose confirmed
intent settles what Goalrail will and will not do about it. Read this only for
how the question arrived; read the change for the answer.

## What prompted it

`gr init` was run for the first time against a repository that already had
OpenSpec with a custom schema — Baseline, schema `intent-driven` v2. The install
succeeded and nothing was damaged, but it exposed a gap: **Goalrail switched the
schema key without telling the user what that switch meant.**

## What actually happened, measured

Refusal path, run without confirmation:

- `gr init` stopped at `EnsureConfig` with
  `the OpenSpec configuration names a custom schema (intent-driven); confirm the
  switch explicitly to adopt the Goalrail schema`, exit 1.
- The repository was untouched: `openspec/config.yaml` byte-identical by sha256,
  no overlay, no marker, no ignore rules, `git status` unchanged.
- Ordering makes this reliable — the configuration is checked before the first
  write, by design.

Adoption path, run with `-confirm-schema-switch`:

- `config.action = switched`, `previous_schema = intent-driven`. Exactly one line
  of `config.yaml` changed. The project's `context:` block and its whole `rules:`
  block survived byte-for-byte.
- Seven overlay files created under `openspec/schemas/goalrail-intent/`.
- Hooks were a pure addition to an existing `.claude/settings.local.json` —
  nothing removed.
- The pre-existing `openspec/schemas/intent-driven/` was left in place, and the
  open change kept its own pinned `schema: intent-driven`, so nothing broke.
- `openspec validate --all` afterwards: 7 passed, 0 failed.

So the mechanics are sound. The gap is in what the user was told.

## The gap

Baseline's `config.yaml` carries project rules that were written against
`intent-driven`:

```yaml
rules:
  intent:
    - Use the project-local $grilling skill explicitly and ask one decision question at a time.
    - Maintain an atomic human-intent ledger with decision context, minimal exact quote, status, interpretation, and lineage.
    - Treat status confirmed without explicit confirmation evidence as draft.
```

Those rules survived the switch — correctly, Goalrail does not own them. But the
schema underneath them changed, and the two now overlap and partly disagree:

- `intent-driven` uses the status vocabulary `draft` / `confirmed`.
  `goalrail-intent` uses `candidate` / `confirmed`. A rule that says "treat
  status confirmed without evidence as draft" now names a status its schema does
  not define.
- `intent-driven`'s instruction told the agent to run the `$grilling` skill and
  keep an atomic human-intent ledger. `goalrail-intent` replaces that with
  stable intent IDs, three semantic groups, and an owner-language confirmation
  view. The surviving rule still demands the ledger. Whether that is a
  complementary house rule or dead weight is a judgement nobody made.
- `goalrail-intent` adds a required `context` artifact. Nothing told the user
  that every future change now needs a Context Pack first.

None of this was reported. `gr init` printed what it created and what it
switched; it did not print what the switch **means** for what was already there.

## What the owner asked for

Think through the already-has-OpenSpec case properly:

- how to merge an existing configuration **reliably**;
- state what Goalrail **adds**;
- state what **duplicates** what the project already had;
- give the user a decision surface — what to keep, what may be overwritten, what
  to delete — **with Goalrail's recommendation for each**, not a bare prompt.

## Design constraints this must respect

- **Goalrail owns the `schema:` key and the overlay. Nothing else.** The
  line-level edit in `EnsureConfig` exists so a project's prose and rules survive
  a rewrite. Any merge surface must keep that boundary; a parse-and-reserialize
  round trip would reformat content Goalrail does not own.
- **"Delete the old schema" is frequently wrong.** Archived changes and open
  changes pin their schema per change in `.openspec.yaml`. Removing
  `openspec/schemas/<old>/` breaks every change that still names it. Any
  recommendation to delete must first prove nothing references it.
- **The refusal is a feature.** Stopping before the first write on a foreign
  custom schema is what made today's probe safe. A richer merge surface must not
  weaken it into a wizard that edits as it asks.
- **Non-interactive use must stay possible.** `gr init` is run by agents, in CI,
  and in scripts. A decision surface cannot become a mandatory prompt; it needs a
  report mode and a flag-driven answer.

## Open questions

1. Is the deliverable a **report** (`gr doctor` grows a section naming overlaps
   and recommendations) or an **interactive adoption command**? A report is
   cheaper, composes with non-interactive use, and can carry recommendations —
   it may be enough.
2. How does Goalrail know a project rule **duplicates** a schema instruction?
   Textual overlap is unreliable. A curated map of known-superseded rules is
   honest but does not generalize; declaring nothing and only reporting *what
   changed in the schema* may be the truthful smaller step.
3. Should adoption record what the previous schema was, so a later `gr update`
   can tell the difference between a project that adopted and one that always
   ran Goalrail?
4. Does the same surface cover the reverse — a user who wants to keep their own
   schema and take only the overlay?

## Related, noticed on the way

- The refusal message names the remedy ("confirm the switch explicitly") but not
  the flag (`-confirm-schema-switch`). The user hits a wall and has to go find
  it in `--help`. Same class as the recurring error shapes in issue #43.
- **Resolved the same day.** The released binary was v0.1.2 and did not contain
  the merged `unshareable-registration-v0` work, so anyone installing from the
  documentation got behaviour this project had already fixed and not shipped.
  v0.1.3 was cut from `main` and the fixed behaviour was verified against it on a
  fresh repository: the marker goes to `.git/info/exclude` by itself and the
  shared `.gitignore` is never created. Baseline, adopted under v0.1.2, still
  carries the hand-written exclude line the fixed version would have written.
- **Exercised the same day.** `gr update` reports `already_current` and
  `verified` against an unchanged canon, and states plainly that it updates the
  repository's harness rather than the binary. Worth recording because the
  wording elsewhere invites the other reading: upgrading is two steps — download
  the binary as the install documentation describes, then `gr update` per
  repository.
