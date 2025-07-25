package templates

import "kagent.dev/kmcp/pkg/manifest"

// ProjectConfig contains configuration for generating a new project
type ProjectConfig struct {
	Name         string
	Framework    string
	Author       string
	Email        string
	Directory    string
	NoGit        bool
	Verbose      bool
	Version      string
	Tools        map[string]manifest.ToolConfig
	Secrets      manifest.SecretsConfig
	Build        manifest.BuildConfig
	Dependencies manifest.DependencyConfig
}
