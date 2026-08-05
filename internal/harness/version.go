package harness

import "runtime/debug"

// Version is this binary's own version, as the build carries it.
//
// It is the only version a user is asked to think about: the stock OpenSpec CLI's
// version is an internal dependency, pinned inside a release and verified against
// the canon before it ships. Nothing about a repository is decided by reading this
// string — currency is decided by digests — so a version that is wrong misleads
// nobody about their files.
//
// A release build sets releaseVersion through the one linker assignment checked
// by the release verifier. Ordinary local and go-install builds fall back to the
// module identity carried by Go build information. This keeps the release tag
// explicit without introducing a source constant somebody can forget to bump.
var releaseVersion string

var Version = resolveVersion(releaseVersion, debug.ReadBuildInfo)

// unknownVersion is what a build that carries no version string at all reports.
// Saying so is honest; an empty field reads as a bug and an invented number reads
// as a claim nothing established.
const unknownVersion = "unknown"

// resolveVersion reports the version verbatim, with no normalization on the way
// out. The leading `v`, the pseudo-version's timestamp and commit, and the
// `+dirty` marker all survive, because the tag, this string, and the version
// inside a release's archive names are meant to be one string that needs no
// matching rules.
//
// The reader is a parameter so the shapes can be exercised directly: a test
// binary reports the development marker in every version-control state, so no
// test can observe a real stamp from inside itself.
func resolveVersion(stamped string, read func() (*debug.BuildInfo, bool)) string {
	if stamped != "" {
		return stamped
	}
	info, ok := read()
	if !ok || info == nil || info.Main.Version == "" {
		return unknownVersion
	}
	return info.Main.Version
}
