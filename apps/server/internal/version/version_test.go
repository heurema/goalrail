package version_test

import (
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/version"
)

func TestCurrent(t *testing.T) {
	info := version.Current()

	if info.Service != "goalrail-server" {
		t.Fatalf("Service = %q, want %q", info.Service, "goalrail-server")
	}
	if info.Version != "0.0.0-dev" {
		t.Fatalf("Version = %q, want %q", info.Version, "0.0.0-dev")
	}
}
