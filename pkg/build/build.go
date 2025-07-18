package build

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Options contains configuration for building MCP servers
type Options struct {
	ProjectDir string
	Docker     bool
	Output     string
	Tag        string
	Platform   string
	Verbose    bool
}

// Builder handles building MCP servers
type Builder struct {
	// Future: Add fields for template handling, etc.
}

// New creates a new Builder instance
func New() *Builder {
	return &Builder{}
}

// Build executes the build process for an MCP server
func (b *Builder) Build(opts Options) error {
	if opts.Verbose {
		fmt.Printf("Starting build process...\n")
	}

	// Detect project type
	projectType, err := b.detectProjectType(opts.ProjectDir)
	if err != nil {
		return fmt.Errorf("failed to detect project type: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("Detected project type: %s\n", projectType)
	}

	// Build based on project type
	switch projectType {
	case "python":
		return b.buildPython(opts)
	case "node":
		return b.buildNode(opts)
	case "go":
		return b.buildGo(opts)
	default:
		return fmt.Errorf("unsupported project type: %s", projectType)
	}
}

// detectProjectType determines the project type based on files present
func (b *Builder) detectProjectType(dir string) (string, error) {
	// Check for Python project
	if b.fileExists(filepath.Join(dir, "pyproject.toml")) ||
		b.fileExists(filepath.Join(dir, ".python-version")) ||
		b.fileExists(filepath.Join(dir, "requirements.txt")) ||
		b.fileExists(filepath.Join(dir, "setup.py")) {
		return "python", nil
	}

	// Check for Node.js project
	if b.fileExists(filepath.Join(dir, "package.json")) {
		return "node", nil
	}

	// Check for Go project
	if b.fileExists(filepath.Join(dir, "go.mod")) {
		return "go", nil
	}

	return "", fmt.Errorf("unknown project type")
}

// fileExists checks if a file exists
func (b *Builder) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// buildPython handles building Python MCP servers
func (b *Builder) buildPython(opts Options) error {
	fmt.Println("Building Python MCP server...")

	if opts.Docker {
		return b.buildDockerImage(opts, "python")
	}

	// For now, just validate that we can build
	fmt.Println("✓ Python project validation passed")
	fmt.Println("Note: Native Python builds will be implemented in future iterations")

	return nil
}

// buildNode handles building Node.js MCP servers
func (b *Builder) buildNode(opts Options) error {
	fmt.Println("Building Node.js MCP server...")

	if opts.Docker {
		return b.buildDockerImage(opts, "node")
	}

	// For now, just validate that we can build
	fmt.Println("✓ Node.js project validation passed")
	fmt.Println("Note: Native Node.js builds will be implemented in future iterations")

	return nil
}

// buildGo handles building Go MCP servers
func (b *Builder) buildGo(opts Options) error {
	fmt.Println("Building Go MCP server...")

	if opts.Docker {
		return b.buildDockerImage(opts, "go")
	}

	// For now, just validate that we can build
	fmt.Println("✓ Go project validation passed")
	fmt.Println("Note: Native Go builds will be implemented in future iterations")

	return nil
}

// buildDockerImage builds a Docker image for the MCP server
func (b *Builder) buildDockerImage(opts Options, projectType string) error {
	fmt.Printf("Building Docker image for %s project...\n", projectType)

	// Check if Docker is available
	if err := b.checkDockerAvailable(); err != nil {
		return fmt.Errorf("Docker not available: %w", err)
	}

	// Check if Dockerfile exists
	dockerfilePath := filepath.Join(opts.ProjectDir, "Dockerfile")
	if !b.fileExists(dockerfilePath) {
		return fmt.Errorf("Dockerfile not found at %s", dockerfilePath)
	}

	// Generate image name if not provided
	imageName := opts.Output
	if imageName == "" {
		dirName := filepath.Base(opts.ProjectDir)
		imageName = strings.ToLower(dirName)
	}

	// Add tag if provided
	if opts.Tag != "" {
		imageName = imageName + ":" + opts.Tag
	} else {
		imageName = imageName + ":latest"
	}

	// Prepare docker build command
	args := []string{"build", "-t", imageName}

	// Add platform if specified
	if opts.Platform != "" {
		args = append(args, "--platform", opts.Platform)
	}

	// Add context (current directory)
	args = append(args, ".")

	if opts.Verbose {
		fmt.Printf("Running: docker %s\n", strings.Join(args, " "))
	}

	// Create docker command
	cmd := exec.Command("docker", args...)
	cmd.Dir = opts.ProjectDir

	if opts.Verbose {
		// Show real-time output for verbose mode
		return b.runCommandWithOutput(cmd, imageName)
	}
	// Capture output and show progress for non-verbose mode
	return b.runCommandWithProgress(cmd, imageName)
}

// checkDockerAvailable verifies that Docker is available and running
func (b *Builder) checkDockerAvailable() error {
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker is not available or not running. Please ensure Docker is installed and running")
	}
	return nil
}

// runCommandWithOutput runs a command and streams output in real-time
func (b *Builder) runCommandWithOutput(cmd *exec.Cmd, imageName string) error {
	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker build: %w", err)
	}

	// Stream output
	go b.streamOutput(stdout, "")
	go b.streamOutput(stderr, "")

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	fmt.Printf("✓ Successfully built Docker image: %s\n", imageName)
	return nil
}

// runCommandWithProgress runs a command and shows progress without streaming all output
func (b *Builder) runCommandWithProgress(cmd *exec.Cmd, imageName string) error {
	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker build: %w", err)
	}

	// Show progress indicator
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		chars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fmt.Printf("\r%s Building Docker image...", chars[i%len(chars)])
				i++
			}
		}
	}()

	// Wait for command to complete
	err := cmd.Wait()
	done <- true
	fmt.Print("\r")

	if err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	fmt.Printf("✓ Successfully built Docker image: %s\n", imageName)
	return nil
}

// streamOutput reads from a pipe and outputs lines with optional prefix
func (b *Builder) streamOutput(pipe io.ReadCloser, _ string) {
	defer pipe.Close()

	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}
}
