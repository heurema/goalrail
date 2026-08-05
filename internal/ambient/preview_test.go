package ambient

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewConnectionIsReadOnlyAndMatchesConnect(t *testing.T) {
	for _, test := range []struct {
		name     string
		scaffold Scaffold
		owner    []byte
	}{
		{name: "codex empty", scaffold: ScaffoldCodex},
		{name: "codex owner config", scaffold: ScaffoldCodex, owner: []byte("model = \"gpt-5\"\n")},
		{name: "claude empty", scaffold: ScaffoldClaudeCode},
		{name: "claude owner config", scaffold: ScaffoldClaudeCode, owner: []byte("{\n  \"permissions\": {\n    \"allow\": [\"Read\"]\n  }\n}\n")},
	} {
		t.Run(test.name, func(t *testing.T) {
			home := filepath.Join(t.TempDir(), "home")
			repository := filepath.Join(t.TempDir(), "repository")
			if err := os.MkdirAll(repository, 0o755); err != nil {
				t.Fatal(err)
			}
			target, err := RegistrationTarget(test.scaffold, home, repository)
			if err != nil {
				t.Fatal(err)
			}
			if test.owner != nil {
				if err := os.MkdirAll(filepath.Dir(target.Path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target.Path, test.owner, 0o600); err != nil {
					t.Fatal(err)
				}
			}

			executable := filepath.Join(home, ".local", "bin", "gr")
			plan, err := PlanRegistration(target, executable)
			if err != nil {
				t.Fatal(err)
			}
			preview, err := PreviewConnection(plan)
			if err != nil {
				t.Fatal(err)
			}
			if preview.BeforeExists != (test.owner != nil) || !bytes.Equal(preview.Before, test.owner) {
				t.Fatalf("preview before = (%t, %q), want (%t, %q)", preview.BeforeExists, preview.Before, test.owner != nil, test.owner)
			}
			if test.owner == nil {
				if _, err := os.Lstat(target.Path); !os.IsNotExist(err) {
					t.Fatalf("preview created config: %v", err)
				}
			} else {
				raw, err := os.ReadFile(target.Path)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(raw, test.owner) {
					t.Fatalf("preview changed owner config: %q", raw)
				}
			}

			changed, err := Connect(plan)
			if err != nil {
				t.Fatal(err)
			}
			if !changed {
				t.Fatal("connection unexpectedly reported no change")
			}
			actual, err := os.ReadFile(target.Path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(actual, preview.After) {
				t.Fatalf("connected bytes differ from preview\nactual: %q\npreview: %q", actual, preview.After)
			}
		})
	}
}

func TestPreviewConnectionRefusesSymlinkInput(t *testing.T) {
	root := t.TempDir()
	owner := filepath.Join(root, "owner.toml")
	if err := os.WriteFile(owner, []byte("model = \"owner\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(root, "config.toml")
	if err := os.Symlink(owner, config); err != nil {
		t.Fatal(err)
	}
	_, err := PreviewConnection(ConnectionPlan{
		Scaffold:   ScaffoldCodex,
		ConfigPath: config,
		Executable: filepath.Join(root, "gr"),
	})
	if err == nil {
		t.Fatal("preview accepted a symlinked config")
	}
	raw, readErr := os.ReadFile(owner)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(raw) != "model = \"owner\"\n" {
		t.Fatalf("owner target changed: %q", raw)
	}
}

func TestPreviewConnectionRefusesOversizedConfig(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "config.toml")
	if err := os.WriteFile(config, bytes.Repeat([]byte{'x'}, maxPreviewConfigBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := PreviewConnection(ConnectionPlan{
		Scaffold:   ScaffoldCodex,
		ConfigPath: config,
		Executable: filepath.Join(root, "gr"),
	})
	if err == nil {
		t.Fatal("preview accepted an oversized config")
	}
}

func TestPreviewConnectionRefusesAnEditThatWouldCrossTheBound(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "config.toml")
	if err := os.WriteFile(config, bytes.Repeat([]byte{'x'}, maxPreviewConfigBytes), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := PreviewConnection(ConnectionPlan{
		Scaffold:   ScaffoldCodex,
		ConfigPath: config,
		Executable: filepath.Join(root, "gr"),
	})
	if err == nil {
		t.Fatal("preview accepted an edit whose result exceeds the config bound")
	}
}
