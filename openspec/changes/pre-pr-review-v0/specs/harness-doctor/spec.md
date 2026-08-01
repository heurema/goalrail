## ADDED Requirements

### Requirement: The diagnosis reports whether the branch's review still describes it
**Intent IDs:** OUT-6, SIG-4

Where a receipt exists for the current branch, the diagnosis SHALL report in one
line whether that review still describes the branch: current where the branch's
present diff digest matches the receipt's, stale where it does not. Where no
receipt exists for the branch, it SHALL report that too. Where a receipt exists
but cannot be read or its digest cannot be recomputed — truncated, malformed,
oversized, an unknown schema, a pruned base commit — the diagnosis SHALL report
it as unreadable and name the cause, because presenting "never reviewed" for a
branch whose evidence is corrupt is the one wrong answer available here. A review that was run
and then outrun by three more commits is worse than no review, because it reads
as done.

The line SHALL state a condition, not a fault: it MUST NOT affect the overall
verdict or the exit status, and MUST NOT be phrased as an error. A branch with
no review is an ordinary state, and the review is evidence rather than a gate.

The state SHALL follow from recomputing the digest alone. Committing further
work makes a review stale by itself, and running a fresh review makes it current
by itself; there SHALL be no flag, no acknowledgement command, and no stored
dismissal. A condition that must be cleared by hand is one nobody reads.

The line MUST NOT reproduce the reviewer's report. The report is stored
verbatim in the receipt for whoever wants it; reprinting it on every diagnosis
would be noise.

#### Scenario: The review still describes the branch
- **WHEN** diagnosis runs on a branch whose receipt digest matches its present diff
- **THEN** it reports the review as current, naming the reviewer and the time, with no effect on the verdict or the exit status

#### Scenario: New work has outrun the review
- **WHEN** a commit is added after a review and diagnosis runs
- **THEN** it reports the review as stale, and no flag was needed to change that

#### Scenario: A fresh review restores currency
- **WHEN** the review is run again after that commit
- **THEN** the diagnosis reports it as current again, with no dismissal or acknowledgement anywhere

#### Scenario: The branch has never been reviewed
- **WHEN** diagnosis runs on a branch with no receipt
- **THEN** it reports the absence as a condition, and the overall verdict and exit status are what they would be without the line

#### Scenario: The report is not reprinted
- **WHEN** the line is reported for a branch whose receipt holds a long report
- **THEN** no part of the report text appears in the diagnosis output

#### Scenario: The receipt is unusable
- **WHEN** diagnosis runs where a receipt exists but cannot be read or recomputed
- **THEN** it reports the review as unreadable with the cause, and the overall verdict and exit status are unchanged
