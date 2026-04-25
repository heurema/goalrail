package readiness

import (
	"testing"
	"testing/fstest"

	"github.com/heurema/goalrail/apps/cli/internal/spine"
)

func TestScanScoresFakeRepos(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		fsys       fstest.MapFS
		wantScore  int
		wantStatus string
	}{
		{
			name: "minimal repo",
			fsys: fstest.MapFS{
				"README.md": {Data: []byte("demo")},
			},
			wantScore:  17,
			wantStatus: spine.ReadinessStatusNotReady,
		},
		{
			name: "rich repo",
			fsys: fstest.MapFS{
				"README.md":                {Data: []byte("demo")},
				"LICENSE":                  {Data: []byte("Apache-2.0")},
				".github/workflows/ci.yml": {Data: []byte("name: ci")},
				"package.json":             {Data: []byte(`{"scripts":{"test":"vitest"}}`)},
				"AGENTS.md":                {Data: []byte("rules")},
				".github/CODEOWNERS":       {Data: []byte("* @team")},
			},
			wantScore:  100,
			wantStatus: spine.ReadinessStatusReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			report, err := Scan(tt.fsys)
			if err != nil {
				t.Fatalf("Scan() error = %v", err)
			}
			if report.Score != tt.wantScore {
				t.Fatalf("Score = %d, want %d", report.Score, tt.wantScore)
			}
			if report.Status != tt.wantStatus {
				t.Fatalf("Status = %q, want %q", report.Status, tt.wantStatus)
			}
		})
	}
}
