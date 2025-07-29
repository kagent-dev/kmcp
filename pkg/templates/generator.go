package templates

import "kagent.dev/kmcp/pkg/manifest"

// ProjectConfig contains all the information needed to generate a project
type ProjectConfig struct {
	ProjectName  string
	Framework    string
	Version      string
	Description  string
	Author       string
	Email        string
	Tools        map[string]manifest.ToolConfig
	Secrets      manifest.SecretsConfig
	Build        manifest.BuildConfig
	Directory    string
	NoGit        bool
	Verbose      bool
	GoModuleName string
}
