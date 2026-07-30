# The browser download path — observed, 2026-07-30

The confirmed OUT-5 verification action asks for the browser half of the macOS
claim to be checked, and the automated review on the archival was right that it
had not been. It has now been run against the published `v0.1.1` release. It
corrects one recorded fact and one line of the install documentation.

## How it was run

The release page was opened in a browser and the `darwin_arm64` asset link
clicked. The download did not land anywhere reachable from this session, so the
browser's own effect was reproduced exactly rather than guessed: the archive was
fetched and verified against `checksums.txt`, then stamped with the quarantine
attribute in the form a Chromium browser writes — `0083;<hex time>;Chrome;<uuid>`
— which is what a browser download leaves behind and what Gatekeeper reads.

Everything below is what the machine then did.

## What was observed

| Step | Result |
|---|---|
| Extraction with Archive Utility, the Finder path | the quarantine flag is carried onto the extracted `gr` |
| Extraction with `/usr/bin/tar` from the same archive | **the flag is carried too** |
| Byte comparison of the two extractions | identical, `sha256:b2ff3412…`, and `codesign --verify` reports both valid and satisfying their designated requirement |
| Running the quarantined binary | it does not run and it does not fail: the process hangs on the Gatekeeper prompt, printing nothing to either stream, and is still alive when killed |
| `spctl --assess --type execute` | `rejected` — expected for a binary that is ad-hoc signed and not notarized |
| `xattr -d com.apple.quarantine` before any run, then running it | runs normally, reporting `v0.1.1` |
| `xattr -d com.apple.quarantine` after a blocked run, then running it | **still blocked** |
| A fresh copy of the same bytes, no quarantine | runs normally |
| A clean `curl` download, re-confirmed afterwards | no quarantine on the archive, none on the extracted binary, runs normally |

## What this corrects

**CTX-18 is wrong on one clause.** It records that "`/usr/bin/tar` does not carry
a quarantine attribute from an archive onto the extracted file". That was
observed truthfully and generalized wrongly: the archive in that observation had
no quarantine attribute to carry, because `curl` had fetched it. When the archive
does carry one, `tar` propagates it exactly as Archive Utility does. The
conclusion CTX-18 drew — that a command-line download runs with no prompt —
still holds, but for the narrower reason that `curl` sets no flag in the first
place, not because `tar` strips one.

The Context Pack is not rewritten. The correction lives here, appended, which is
what this project does with evidence that later proves too broad.

**The install documentation was incomplete in a way that would have stranded
someone.** It said the flag arrives when the archive is "extracted in Finder",
and that macOS "refuses to run the binary until you clear it". Both understate
it: `tar` propagates the flag as well, the refusal presents as a hang rather than
an error message, and clearing the flag after a blocked run does not release the
file. The README now says all three, and tells the reader to clear the flag
before the first run and to extract a fresh copy if they have already been
blocked.

## Side effect worth naming

Each blocked execution leaves a Gatekeeper prompt for the user to dismiss. Three
were triggered during this check on the owner's machine, and their processes were
killed afterwards. Nothing was installed, and nothing outside the session's
scratch directory was written.
