# Intent Snapshot

- **Intent ID:** INT-openspec-adoption-v0
- **Version:** 1
- **Status:** confirmed
- **Owner:** Vitaly D.
- **Context Pack:** CTXP-openspec-adoption-v0 v1
- **Run references:** local session, 2026-08-01

## Source Evidence

- **SE-1 (owner, 2026-08-01):** "как мержить надежно существующий конфиг что gr
  добавляет что дублирует как то дать пользователю возможность принять решение
  что нужно ему оставить что можно переписать что удалить но с нашими
  рекомендациями"
- **SE-2 (owner, 2026-08-01):** "мы схему переписали о правила. Правила же тоже
  важны получаются. То есть как-то архивировать старую схему не переделать на
  новую. Давай решать."
- **SE-3 (owner, 2026-08-01):** approved the option that splits the problem by
  what is mechanically decidable, after being shown that a per-rule staleness
  verdict is not, and after being shown its three stated weaknesses.
- **SE-4 (repository, CTX-5):** Baseline's rules now name a status the adopted
  schema does not define and require a skill and a ledger it never mentions.
- **SE-5 (repository, CTX-6, CTX-7):** nothing validates project rules against
  the active schema, and rules are materialized nowhere — they mislead only by
  being read.
- **SE-6 (repository, CTX-1):** the configuration edit is line-level precisely so
  that content Goalrail does not own survives a switch.
- **SE-7 (repository, CTX-9):** OpenSpec 1.6.0 has no archive, deprecate or
  delete operation for a schema.
- **SE-8 (repository, CTX-8):** a superseded schema directory is still pinned by
  changes that name it, so deleting it is usually wrong.
- **SE-9 (repository, CTX-11):** today's real adoption reported what it created
  and switched, and disclosed none of the above.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | When Goalrail switches a repository from another schema to its own, the initialization report states what the switch mechanically changed: which artifacts the adopted schema adds or removes, which artifact dependencies changed, and which artifact instructions differ. | Read the report from a real switch and compare it against a hand-made diff of the two schema files. | SE-1, CTX-13 |
| OUT-2 | The same report states how many project rules the configuration carries, echoes them verbatim, and says plainly that they were written against the previous schema and that Goalrail neither interprets nor edits them. | Read the report on a repository whose rules are known to be stale, and confirm it neither judges nor omits them. | SE-1, SE-4, SE-5, SE-6 |
| OUT-3 | The same report states how many open and archived changes still pin the previous schema, so "can I delete it" has a computed answer rather than an opinion. | Read the count on Baseline and compare it against the `.openspec.yaml` files that name `intent-driven`. | SE-2, SE-7, SE-8 |
| OUT-4 | The adoption is recorded where Goalrail already keeps its own state — which schema was replaced, when, and a digest of the rules block as it stood — so the fact outlives the session that performed the switch. | Inspect the recorded state after a switch and confirm it names the previous schema and a digest. | SE-9 |
| OUT-5 | The diagnosis carries one advisory line for as long as the rules block is unchanged since adoption, and stops carrying it once the rules are edited, without the user having to dismiss anything. | Run the diagnosis after a switch, edit the rules, run it again, and confirm the line appears and then disappears. | SE-1, SE-9 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Goalrail does not decide whether any individual rule is stale, superseded, or contradictory. Such a verdict requires reading free-form English against a schema's prose, which no mechanical check can do honestly across other people's projects. | SE-3, SE-5 |
| NG-2 | Goalrail does not edit, rewrite, reorder, or delete project rules. The rules block stays the user's to change. | SE-3, SE-6 |
| NG-3 | No interactive prompt is introduced. Initialization and diagnosis stay usable by an agent and by CI, which is how they are actually run. | SE-3 |
| NG-4 | Goalrail does not archive, move, or delete the superseded schema directory, and does not gain a command to do so. | SE-3, SE-7, SE-8 |
| NG-5 | A teammate who receives an adoption through a committed configuration is out of scope for this version; the recorded adoption is per clone. | SE-3 |
| NG-6 | The reverse direction — keeping a project's own schema and taking only the overlay — is out of scope for this version. | SE-3 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The report names the artifact difference correctly on a real adoption. | On Baseline's switch, the report names `context` as added and names `design` as having gained dependencies; both are checkable against the two schema files. | CTX-13 |
| SIG-2 | The report discloses rules without judging them. | The rules section reproduces every rule and contains no per-rule verdict word such as "stale", "remove", or "superseded". | SE-5, NG-1 |
| SIG-3 | The superseded-schema answer is computed, not asserted. | On Baseline the count is greater than zero and the report says the directory must stay; on a repository where nothing pins it the count is zero and the report says it may be removed. | CTX-8 |
| SIG-4 | The advisory ends by itself. | The diagnosis shows the line after adoption, and stops showing it after any edit to the rules block, with no flag and no acknowledgement command. | OUT-5 |
| SIG-5 | Nothing becomes interactive. | Initialization and diagnosis produce the same machine-readable shape and exit status as before when no terminal is attached. | NG-3 |

## Ambiguities and Unknowns

None. The three weaknesses of the selected approach were stated to the owner
before approval and accepted: a digest over the rules block is silenced by any
edit that touches that block rather than only by a considered one; the recorded
adoption is per clone; and on a repository like Baseline the superseded schema
directory will never become removable, because its archive keeps pinning it.

## Confirmation

- **Confirmed by:** Vitaly D.
- **Confirmed at:** 2026-08-01
- **Verification action:** The owner was shown a plain-language view in Russian covering every outcome, boundary and success signal in this version, together with the three accepted weaknesses of the approach, and confirmed it without amendment.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
