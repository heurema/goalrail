package gitctx

import "testing"

func TestParseRemoteURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		provider string
		host     string
		fullName string
	}{
		{
			name:     "github scp",
			raw:      "git@github.com:owner/repo.git",
			provider: "github",
			host:     "github.com",
			fullName: "owner/repo",
		},
		{
			name:     "github https",
			raw:      "https://github.com/owner/repo.git",
			provider: "github",
			host:     "github.com",
			fullName: "owner/repo",
		},
		{
			name:     "github ssh",
			raw:      "ssh://git@github.com/owner/repo.git",
			provider: "github",
			host:     "github.com",
			fullName: "owner/repo",
		},
		{
			name:     "gitlab nested group",
			raw:      "git@gitlab.com:group/subgroup/repo.git",
			provider: "gitlab",
			host:     "gitlab.com",
			fullName: "group/subgroup/repo",
		},
		{
			name:     "bitbucket",
			raw:      "https://bitbucket.org/workspace/repo.git",
			provider: "bitbucket",
			host:     "bitbucket.org",
			fullName: "workspace/repo",
		},
		{
			name:     "custom",
			raw:      "ssh://git@git.example.com/team/repo.git",
			provider: "custom_git",
			host:     "git.example.com",
			fullName: "team/repo",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := ParseRemoteURL(test.raw)
			if got.Provider != test.provider {
				t.Fatalf("Provider = %q, want %q", got.Provider, test.provider)
			}
			if got.ProviderHost != test.host {
				t.Fatalf("ProviderHost = %q, want %q", got.ProviderHost, test.host)
			}
			if got.RepositoryFullName != test.fullName {
				t.Fatalf("RepositoryFullName = %q, want %q", got.RepositoryFullName, test.fullName)
			}
		})
	}
}
