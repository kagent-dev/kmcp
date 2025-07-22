package templates

// ProjectConfig contains configuration for generating a new project
type ProjectConfig struct {
	Name      string
	Framework string
	Template  string
	Author    string
	Email     string
	Directory string
	NoGit     bool
	Verbose   bool
}
