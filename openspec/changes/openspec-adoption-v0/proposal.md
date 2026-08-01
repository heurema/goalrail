# Adoption tells the user what changed under their rules

## Why

Initialization already refuses to switch a foreign schema without confirmation
and already leaves existing specs, changes and configuration prose untouched —
but once the user confirms, it says nothing about what the switch means for the
rules they wrote against the schema it replaced. Baseline's adoption on
2026-08-01 left rules naming a status the adopted schema does not define, and
nothing in the toolchain would ever report it.

## What Changes

- The initialization report gains an adoption section, present only when a
  switch actually replaced another schema. It carries three mechanically derived
  facts and no judgement:
  - the artifact-level difference between the replaced schema and the adopted
    one — artifacts added or removed, dependencies changed, instructions
    changed — where both are files in the repository, and a stated reason where
    the replaced one is not, as a stock schema resolving from the installed
    package never is;
  - how many rules the configuration carries, reproduced verbatim, with a plain
    statement that they were present when the schema was replaced and that
    Goalrail neither interprets nor edits them;
  - how many open and archived changes still pin the replaced schema, so
    whether its directory may be removed is counted rather than guessed.
- The marker Goalrail already owns records the adoption: the replaced schema
  name, when it happened, and a digest of the rules block as it stood. A marker
  written before this change carries no adoption record, which reads as "never
  adopted" rather than as an error.
- The diagnosis gains one advisory line, shown while the rules block still
  matches the digest recorded at adoption, and absent once it does not. It
  follows the existing precedent for reporting a condition without nagging: one
  line, no fault, no acknowledgement command.
- No output shape is removed and no field is repurposed, so nothing here is
  breaking.

## Intent Coverage

| Change | Intent IDs | How it preserves the boundaries |
|---|---|---|
| Artifact-level schema difference in the report | OUT-1, SIG-1 | Derived from the two schema files on disk; it reports that an instruction differs, never what the difference means (NG-1) |
| Rules disclosed verbatim with a plain statement | OUT-2, SIG-2 | Reproduces and counts; renders no per-rule verdict (NG-1) and writes nothing back into the configuration (NG-2) |
| Counted pins on the replaced schema | OUT-3, SIG-3 | Answers the removal question with a count; Goalrail neither moves nor deletes the directory (NG-4) |
| Adoption recorded in the marker | OUT-4 | Written to Goalrail's own state, not to the user's configuration (NG-2); per clone by design (NG-5) |
| Advisory line in the diagnosis, self-terminating | OUT-5, SIG-4 | Appears and disappears from a digest comparison alone, with no prompt and no dismissal (NG-3) |
| Every surface stays machine-readable | SIG-5 | No terminal interaction is introduced anywhere (NG-3) |

## Capabilities

**New Capabilities**

None. Every requirement belongs to a capability that already exists.

**Modified Capabilities**

- `harness-init` — the requirement that an existing OpenSpec root survives
  initialization gains the obligation to disclose what the switch changed, and a
  new requirement covers recording the adoption in the marker.
- `harness-doctor` — a new requirement covers the adoption advisory and the
  condition under which it stops.

## Impact

- `internal/harness/config.go` — the configuration outcome already reports the
  previous schema; the adoption section needs the rules block alongside it.
- `internal/harness/canon.go` — the adopted schema is embedded; the replaced one
  is read from the repository. Comparing them is new work with no existing home.
- `internal/ambient` — the marker gains an optional adoption record. Absence
  must remain valid, because every marker written so far lacks it.
- `internal/harness/doctor.go` — one more reported line, following the
  observability precedent for a fact that is not a fault.
- `cmd/gr/ambient.go` and the initialization report structure — new fields only.
- Reading `openspec/changes/**/.openspec.yaml` to count pins is new; the OpenSpec
  adapter already parses that file and is the natural place for it.
- No new dependency. No Node runtime is introduced: the comparison reads YAML
  files Goalrail already materializes.

## Non-Goals

- No verdict on whether any individual rule is stale or contradictory.
- No editing, rewriting, reordering or deletion of project rules.
- No interactive prompt in initialization or diagnosis.
- No archival, move or deletion of the replaced schema directory, and no command
  to perform one.
- No handling of a teammate who receives an adoption through a committed
  configuration.
- No support for keeping a project's own schema while taking only the overlay.
