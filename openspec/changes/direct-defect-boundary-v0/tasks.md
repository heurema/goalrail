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
