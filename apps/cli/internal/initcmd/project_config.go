package initcmd

import "github.com/heurema/goalrail/apps/cli/internal/projectconfig"

const (
	projectConfigVersion            = projectconfig.Version
	projectConfigRelativePath       = projectconfig.RelativePath
	projectConfigConflictMessage    = projectconfig.ConflictMessage
	projectConfigUnparseableMessage = projectconfig.UnparseableMessage
	localConfigStatusWritten        = projectconfig.StatusWritten
	localConfigStatusUnchanged      = projectconfig.StatusUnchanged
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

func renderProjectConfigYAML(config projectConfig) string {
	return projectconfig.RenderYAML(config)
}

func parseProjectConfigYAML(raw string) (projectConfig, error) {
	return projectconfig.ParseYAML(raw)
}

func quoteYAMLString(value string) string {
	return projectconfig.QuoteYAMLString(value)
}
