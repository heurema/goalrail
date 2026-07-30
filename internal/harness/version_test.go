package harness

import (
	"runtime/debug"
	"testing"
)

// TestVersionIsReportedExactlyAsTheBuildCarriesIt pins the whole resolution rule
// against the shapes the toolchain actually produces, each of them observed by
// building this repository in a scratch clone rather than guessed:
//
//   - a clean checkout exactly at tag v0.1.0 stamps `v0.1.0`;
//   - one commit later, the next patch version as a pseudo-version;
//   - with no tags at all, a v0.0.0 pseudo-version;
//   - a modified tree adds `+dirty` to whichever shape applies;
//   - a build with version control stamping off stamps `(devel)`.
//
// The reader is injected because a test binary reports `(devel)` in every Git
// state, so none of these shapes can be observed from inside a test.
func TestVersionIsReportedExactlyAsTheBuildCarriesIt(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		carried string
	}{
		{"at a release tag", "v0.1.0"},
		{"at a release tag with local modifications", "v0.1.0+dirty"},
		{"between releases", "v0.1.1-0.20260730151807-c83a28b3f471"},
		{"before any release", "v0.0.0-20260730151106-3beb72c6fa34"},
		{"before any release with local modifications", "v0.0.0-20260730151106-3beb72c6fa34+dirty"},
		{"with no version control behind it", "(devel)"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			resolved := resolveVersion(buildInfo(testCase.carried))
			if resolved != testCase.carried {
				t.Errorf("resolved %q from a build carrying %q", resolved, testCase.carried)
			}
		})
	}
}

// TestVersionSaysUnknownWhenTheBuildCarriesNone pins that the absence of a
// version is reported as such. An empty string reads as a bug in the tool and a
// number nobody stamped reads as a claim; `unknown` is the only honest answer.
func TestVersionSaysUnknownWhenTheBuildCarriesNone(t *testing.T) {
	t.Run("no build information at all", func(t *testing.T) {
		if resolved := resolveVersion(func() (*debug.BuildInfo, bool) { return nil, false }); resolved != unknownVersion {
			t.Errorf("resolved %q with no build information", resolved)
		}
	})
	t.Run("build information carrying an empty version", func(t *testing.T) {
		if resolved := resolveVersion(buildInfo("")); resolved != unknownVersion {
			t.Errorf("resolved %q from a build carrying no version string", resolved)
		}
	})
	t.Run("a reader that reports success with nothing behind it", func(t *testing.T) {
		if resolved := resolveVersion(func() (*debug.BuildInfo, bool) { return nil, true }); resolved != unknownVersion {
			t.Errorf("resolved %q from a nil build information", resolved)
		}
	})
}

// TestVersionIsNeverEmpty pins the package-level value itself: whatever this
// binary was built from, `gr version` and the diagnosis have something to report.
func TestVersionIsNeverEmpty(t *testing.T) {
	if Version == "" {
		t.Error("the reported version is empty")
	}
}

func buildInfo(version string) func() (*debug.BuildInfo, bool) {
	return func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Path: "github.com/heurema/goalrail", Version: version}}, true
	}
}
