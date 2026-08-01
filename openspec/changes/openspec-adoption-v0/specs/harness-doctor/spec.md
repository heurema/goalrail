## ADDED Requirements

### Requirement: The diagnosis reports rules that predate an adoption, and stops on its own
**Intent IDs:** OUT-5, SIG-4, SIG-5

Where the marker records an adoption of a configuration that carried at least one
rule, and that rules block still matches the digest recorded at the adoption, the
diagnosis SHALL report it in one line naming the replaced schema and when the
adoption happened. Initialization discloses the same fact once, to whoever ran
it; that is routinely an agent, and the person whose rules they are may never see
it. The diagnosis is where a standing condition belongs.

Where the configuration carried no rules at the adoption, the diagnosis SHALL
report nothing. There is nothing that predates the schema, and the advisory's
stop condition is an edit to rules that do not exist — a standing condition the
user could never clear by reviewing anything.

The line SHALL state a condition, not a fault: it MUST NOT affect the overall
verdict or the exit status, and MUST NOT be phrased as an error. Rules written
against a replaced schema are frequently still correct, and Goalrail cannot tell
which ones are not.

The report SHALL stop carrying the line once the rules block no longer matches
the recorded digest, with no flag, no acknowledgement command and no stored
dismissal. Editing the rules is the act of reviewing them, so the only honest
stop condition is the one the user's own edit produces. A condition that must be
dismissed by hand becomes a condition nobody reads.

Where the marker carries no adoption record, the diagnosis SHALL report nothing
about rules at all.

The line MUST NOT reproduce the rules themselves. Initialization discloses them
in full at the moment they can still be compared against what was replaced; a
diagnosis that reprinted them at every run would be noise.

#### Scenario: Rules are unchanged since an adoption
- **WHEN** diagnosis runs where the marker records an adoption and the rules block still matches the recorded digest
- **THEN** it reports one line naming the replaced schema and the adoption time, the overall verdict is unaffected, and the exit status is what it would have been without the line

#### Scenario: The rules have been edited since the adoption
- **WHEN** the rules block is edited after an adoption and diagnosis runs again
- **THEN** the line is absent, and no flag, command or stored acknowledgement was needed to remove it

#### Scenario: An unrelated part of the configuration is edited
- **WHEN** the configuration is edited outside the rules block after an adoption and diagnosis runs
- **THEN** the line is still reported, because the rules themselves are unchanged

#### Scenario: The repository never replaced a schema
- **WHEN** diagnosis runs where the marker carries no adoption record
- **THEN** it reports nothing about rules

#### Scenario: The adopted configuration carried no rules
- **WHEN** diagnosis runs where the marker records an adoption of a configuration that carried no rules
- **THEN** it reports nothing about rules, however many times it is run

#### Scenario: The line does not reprint the rules
- **WHEN** the line is reported for a configuration carrying several rules
- **THEN** no rule text appears in the diagnosis output
