## ADDED Requirements

### Requirement: A release refuses a dependency closure nobody recorded
**Intent IDs:** OUT-1, OUT-2, OUT-4, SIG-1, SIG-2, SIG-4

The pinned runtime and compiler SHALL be accompanied by a record of the
dependency closure they resolve to: how many packages it holds, how many of
those carry an install script, and its digest. Loading the release inputs SHALL
compare the computed closure against that record and SHALL refuse when they
disagree, naming which of the three facts did.

The comparison SHALL sit where the inputs are loaded rather than in one command.
Building a release, verifying one, and inspecting the pins all read the same
inputs, so a check held by one of them leaves the other two able to produce a
release against a set nobody looked at.

Recording the counts beside the digest is not redundancy. A digest tells a
reviewer that something changed; the counts tell them what, and a package
acquiring an install script is the single fact most worth seeing move — a
compromise arrives through the maintainer's own publishing pipeline with valid
provenance, so what distinguishes it is the shape of the set, not its
attestations.

The expanded document SHALL carry its own schema identifier and SHALL be
registered as such, because its fields are mandatory and its decoding is strict:
one identifier cannot describe both shapes when each rejects the other's
documents.

#### Scenario: The closure matches its record
- **WHEN** the computed closure agrees with the record beside the pins
- **THEN** the release inputs load and the pins are reported

#### Scenario: A package appears or disappears
- **WHEN** the computed closure holds a different number of packages than the record
- **THEN** loading refuses and names the package count as what disagreed

#### Scenario: A package acquires an install script
- **WHEN** the computed closure holds a different number of packages carrying an install script than the record
- **THEN** loading refuses and names the install-script count as what disagreed

#### Scenario: The closure digest moves
- **WHEN** the computed closure digest differs from the record
- **THEN** loading refuses and names the digest as what disagreed

#### Scenario: A release is built without inspecting the pins first
- **WHEN** a release is built or verified after the dependency lock changed and the record was not updated
- **THEN** it refuses on the same comparison, because the check is not held by the inspecting command alone

#### Scenario: The expanded document claims the previous identifier
- **WHEN** a source lock carrying the closure record and adoption dates claims the identifier of the shape that carried neither
- **THEN** it is refused

### Requirement: A pin discloses when it was published and when it was adopted
**Intent IDs:** OUT-3, SIG-3, SIG-5

Each pinned component SHALL record when its version was published upstream and
when this repository adopted it, so the age at adoption is visible to whoever
reviews a pin move. A version adopted days after publication is the exposure a
quarantine practice exists to avoid, and this is the evidence that makes it
visible.

These dates SHALL be disclosure rather than a gate, and SHALL NOT be described
as one. Both are supplied by whoever moves the pin, so no waiting period can be
enforced on itself; the single inconsistency the record establishes alone is a
pin claiming to have been adopted before it was published. How long a version
should have been public before adoption is not settled by this requirement.

#### Scenario: Both dates are present
- **WHEN** a pinned component records its upstream publication and its adoption here
- **THEN** the record is accepted and the age at adoption is readable from it

#### Scenario: A date is absent
- **WHEN** either date is missing for a pinned component
- **THEN** the record is refused

#### Scenario: Adoption precedes publication
- **WHEN** a pin records an adoption earlier than its own publication
- **THEN** the record is refused, because that is the one inconsistency it can establish without an outside source

#### Scenario: The dates would be read as a waiting period
- **WHEN** the record is described as enforcing a minimum age before adoption
- **THEN** that violates this requirement, because the dates come from the same act they would gate
