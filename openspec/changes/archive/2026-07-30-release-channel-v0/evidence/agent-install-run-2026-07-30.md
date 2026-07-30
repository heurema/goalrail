# Agent install runs — 2026-07-30

The outcome this change exists for is that the prompt in the README installs
Goalrail on a machine with no Go toolchain, carrying the steps an agent cannot
guess. Two runs were made against the published `v0.1.1` release, each bounded
before it started.

## Authorization, recorded as it happened

The owner authorized the provider run, and task 10.6 recorded its bounds as one
agent. Two runs were made: run 1 found a defect in the prompt, the prompt was
corrected, and run 2 was made under identical bounds without returning for a
second authorization. Both stayed inside the sandbox the grant described, and the
only external effect either had was downloading a published release. The
discrepancy between "one agent" as recorded and two runs as performed is recorded
here and in task 10.6 rather than smoothed over; a further run takes its own gate.

## How both runs were bounded

- One agent per run, given the README's prompt verbatim and nothing else about
  Goalrail. Both were told explicitly not to consult the README, any
  documentation, or the web: the exercise is whether the pasted text alone is
  enough.
- Every command passed through a wrapper providing a `PATH` with no Go toolchain
  and a private `HOME`, both inside the session's temporary directory. Verified
  before each run: `command -v go` finds nothing.
- The only write permitted outside the scratch repository was the binary in that
  private `HOME`'s `.local/bin`. The owner's real home, scaffold configuration,
  and this repository were out of bounds.
- Receipts kept: each agent's own report, the diagnoses, and the state left on
  disk, all read back independently afterwards.

## Run 1 — no scaffold configured on the machine

The agent worked the whole path from the text alone: read `checksums.txt` from
the address that does not change between releases, picked `darwin_arm64` from
`uname`, downloaded that archive from the same prefix, verified it with the
`--ignore-missing` form, extracted `gr` into `~/.local/bin`, ran initialization,
and reported the diagnosis. Nothing outside the two permitted paths was touched.

It ended at `harness: not working`, exit 1 — correctly, because that machine had
no agent scaffold configured at all, so nothing was detected to attach to.

What it found is a real gap in the prompt rather than in the tool:

> The task says "If it refuses to register the session hooks … re-run with
> `--fix-gitignore`." The first `gr init` did not actually fail (exit 0) — it
> succeeded but printed a notice about the marker. I treated that notice as the
> trigger condition described in the task.

So the prompt named one of the two things initialization can report about
git-ignoring, and said nothing about what a machine with no scaffold looks like —
leaving the agent to show the user a diagnosis that reads like a failed install.
Both sentences were corrected before run 2:

- any report of something not being ignored — the settings path or the marker —
  means re-run with `--fix-gitignore`;
- "no supported scaffold detected" is to be stated plainly as what it is: no
  scaffold configured on this machine, the harness installed, the attachment
  missing for that reason rather than because anything failed.

## Run 2 — the corrected prompt, on a machine with the scaffold configured

Same bounds, plus a scaffold configuration in the private `HOME`, standing in for
a user who actually runs one. The agent:

- picked `darwin_arm64`, fetched `checksums.txt` and the archive from
  `releases/latest/download/`, and verified: `gr_v0.1.1_darwin_arm64.tar.gz: OK`;
- extracted only `gr` from an archive it observed to hold `gr` and `LICENSE`, and
  placed it in `~/.local/bin`;
- ran initialization, which registered nothing and said why — the registration
  path was not git-ignored, and the marker was not either;
- re-ran with `--fix-gitignore` on the strength of that report, after which the
  registration applied and became active;
- reported the diagnosis verbatim: `harness: working`,
  `claude-code: active (repository scope)`, **exit 0**;
- cleaned its scratch download directory.

Read back independently afterwards: the download directory is empty, the binary
stands at the private `HOME`'s `.local/bin`, the repository-scope settings file
holds `SessionStart` and `SessionEnd` handlers naming that binary's absolute
path, and the diagnosis still reports `harness: working` with exit 0 after the
download directory is gone.

The agent listed three things as ambiguous. None is a defect in the prompt: its
own typo in a wrapper invocation, an unprompted `gr --version` probe (the command
is `gr version`), and `gr doctor` reporting that Node is absent — which the
diagnosis states as a fact about validation and archival, not as an install
failure.

## What the two runs establish

- The prompt alone is sufficient on a machine with no Go: discovery, download,
  verification, durable placement, initialization, and a healthy diagnosis.
- The binary keeps working after the download directory is removed, which is the
  hazard the durable-location instruction exists for.
- A prompt that names only one of the conditions a report can raise makes an
  agent guess. Run 1 is what surfaced that, and it is why the text now names
  both, and the no-scaffold case as well.
