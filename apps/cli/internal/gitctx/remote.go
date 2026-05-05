package gitctx

import (
	"net"
	"net/url"
	"strings"
)

type RemoteInfo struct {
	Provider           string
	ProviderHost       string
	RepositoryFullName string
}

func ParseRemoteURL(raw string) RemoteInfo {
	host, repoPath := splitRemoteURL(strings.TrimSpace(raw))
	host = normalizeHost(host)
	repoFullName := normalizeRepoPath(repoPath)

	return RemoteInfo{
		Provider:           providerForHost(host),
		ProviderHost:       host,
		RepositoryFullName: repoFullName,
	}
}

func splitRemoteURL(raw string) (string, string) {
	if raw == "" {
		return "", ""
	}

	if !strings.Contains(raw, "://") {
		if before, after, ok := strings.Cut(raw, ":"); ok && strings.Contains(before, "@") {
			_, host, _ := strings.Cut(before, "@")
			return host, after
		}
	}

	parsed, err := url.Parse(raw)
	if err == nil && parsed.Host != "" {
		host := parsed.Hostname()
		if host == "" {
			host = parsed.Host
		}
		return host, parsed.Path
	}

	withoutScheme := strings.TrimPrefix(raw, "//")
	parts := strings.SplitN(withoutScheme, "/", 2)
	if len(parts) == 2 && strings.Contains(parts[0], ".") {
		return parts[0], parts[1]
	}

	return "", raw
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	host = strings.TrimPrefix(host, "www.")
	if withoutPort, _, err := net.SplitHostPort(host); err == nil {
		host = withoutPort
	}
	return host
}

func normalizeRepoPath(repoPath string) string {
	repoPath = strings.TrimSpace(repoPath)
	repoPath = strings.TrimPrefix(repoPath, "/")
	repoPath = strings.TrimSuffix(repoPath, ".git")
	repoPath = strings.Trim(repoPath, "/")
	return repoPath
}

func providerForHost(host string) string {
	switch host {
	case "github.com":
		return "github"
	case "gitlab.com":
		return "gitlab"
	case "bitbucket.org":
		return "bitbucket"
	default:
		return "custom_git"
	}
}
