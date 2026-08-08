## 1. Record the rule

- [x] 1.1 Add the `restoration` class, its digest binding and its ordering condition to the requirement that already makes exceptions first-class lineage evidence
- [x] 1.2 Add the two decided questions to the verifier requirement, with distinct stable reasons for a late claim and a mis-bound claim
- [x] 1.3 State that an affirmative result does not settle the boundary question

## 2. Pin the failure first

- [x] 2.1 A fixture reproducing the `#85` commit order: material commits, then intent, then confirmation, with no prior restoration claim
- [x] 2.2 Record it admitted against the current verifier, which is the defect

## 3. Decide ordering and binding

- [x] 3.1 Represent the `restoration` exception with its bound artifact digest and effect scope
- [x] 3.2 Decide ancestry against the first commit touching a material path in scope, from the frozen range rather than the event's timestamp
- [x] 3.3 Decide the digest binding against the retained snapshot or the committed specification file at its recorded revision
- [x] 3.4 Two stable reason identifiers, one per failure, and no path classified by anything but the existing policy

## 4. Prove it

- [x] 4.1 The `#85` fixture is denied with the ordering reason
- [x] 4.2 The same fixture with the claim anchored in an ancestor is accepted, and classified by the exception rather than `VALID`
- [x] 4.3 A stale or absent bound digest is denied with the binding reason, distinguishable from the ordering one
- [x] 4.4 A work unit that both claims restoration and amends a requirement in scope has no valid claim
- [x] 4.5 Break-glass and bootstrap fixtures are unaffected by the ordering condition
- [x] 4.6 `go vet ./...` and `go test ./...`

## 5. Point the contract at it

- [x] 5.1 One reference in `AGENTS.md` naming the canonical requirement, restating no policy
- [x] 5.2 Confirm no second materiality rule entered the delta

## 6. State the limit

- [x] 6.1 Record in the change that the live check awaits repository adoption (#89) and owner-gated activation, and that fixtures are what this change verifies

## 7. Recorded during implementation

- [x] 7.1 The frozen range did not carry commit parents, so ancestry was not decidable from what the verifier received; the collector now emits them and the design records why position is not ancestry
- [x] 7.2 An anchor outside the frozen range is refused rather than trusted, which is stricter than the prose alone implies

## 8. Recorded after independent review

- [x] 8.1 The anchor was self-reported: an `anchor_commit` field the author writes, checked against a graph read at head, so a late claim could name an early commit. Precedence is now derived from where the claim's own artifact entered the history, and the field is removed
- [x] 8.2 Precedence is checked against every material touch in scope, because reverse topological order is a partial order and parallel branches admit an order that is not ancestry
- [x] 8.3 Binding compares a reference and digest recorded together on one lineage target, rather than the presence of any replica under the claimed digest
- [x] 8.4 Any policy-declared normative path amended inside the claim's scope invalidates it, not only the artifact the claim names
- [x] 8.5 Each new check is shown to fail when disabled, so a passing test is not mistaken for a load-bearing one

## 9. Recorded after the second review

- [x] 9.1 Binding checks the relation as well as the pair: every target carries a reference and digest, so a claim could otherwise bind a commit or receipt and call it a requirement
- [x] 9.2 A policy authorizing restoration without declaring normative paths refuses the claim rather than treating every path as non-normative
- [x] 9.3 The `AGENTS.md` pointer is made true by archiving the delta into the canonical specification, not by softening what it claims
- [x] 9.4 A mutation showed the first binding test proved nothing — the work unit is a lineage source, not a target, so it was refused either way; the fixture now binds a real commit target
