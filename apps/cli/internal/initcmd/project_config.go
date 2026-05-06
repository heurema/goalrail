package initcmd

import "github.com/heurema/goalrail/apps/cli/internal/projectconfig"

const (
	projectConfigVersion            = projectconfig.Version
	projectConfigRelativePath       = projectconfig.RelativePath
	projectConfigIgnoreRelativePath = projectconfig.IgnoreRelativePath
	projectConfigConflictMessage    = projectconfig.ConflictMessage
	projectConfigUnparseableMessage = projectconfig.UnparseableMessage
	localConfigStatusWritten        = projectconfig.StatusWritten
	localConfigStatusUnchanged      = projectconfig.StatusUnchanged
	localConfigStatusUpdated        = projectconfig.StatusUpdated
)

type projectConfig = projectconfig.Config
type projectConfigRepository = projectconfig.Repository

func preflightProjectConfig(gitRoot string, expected projectConfig) error {
	return projectconfig.Preflight(gitRoot, expected)
}

func readProjectConfig(gitRoot string) (projectConfig, bool, error) {
	return projectconfig.Read(gitRoot)
}

func writeProjectConfig(gitRoot string, config projectConfig) (string, error) {
	return projectconfig.Write(gitRoot, config)
}

func ensureProjectConfigGitignore(gitRoot string) (string, error) {
	return projectconfig.EnsureLocalStateGitignore(gitRoot)
}

func renderProjectConfigYAML(config projectConfig) string {
	return projectconfig.RenderYAML(config)
}

func renderProjectConfigGitignore() string {
	return projectconfig.RenderLocalStateGitignore()
}

func parseProjectConfigYAML(raw string) (projectConfig, error) {
	return projectconfig.ParseYAML(raw)
}

func quoteYAMLString(value string) string {
	return projectconfig.QuoteYAMLString(value)
}
