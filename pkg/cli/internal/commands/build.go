package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kagent-dev/kmcp/pkg/cli/internal/manifest"
	"github.com/stoewer/go-strcase"

	"github.com/kagent-dev/kmcp/pkg/cli/internal/build"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build MCP server as a Docker image",
	Long: `Build an MCP server from the current project.
	
This command will detect the project type and build the appropriate
MCP server Docker image.

Examples:
  kmcp build                    # Build Docker image from current directory
  kmcp build --project-dir ./my-project  # Build Docker image from specific directory`,
	RunE: runBuild,
}

var (
	buildTag             string
	buildPush            bool
	buildKindLoad        bool
	buildDir             string
	buildPlatform        string
	buildKindLoadCluster string
)

func init() {
	addRootSubCmd(buildCmd)

	buildCmd.Flags().StringVarP(&buildTag, "tag", "t", "", "Docker image tag (alias for --output)")
	buildCmd.Flags().BoolVar(&buildPush, "push", false, "Push Docker image to registry")
	buildCmd.Flags().BoolVar(&buildKindLoad, "kind-load", false, "Load image into kind cluster (requires kind)")
	buildCmd.Flags().StringVar(&buildKindLoadCluster, "kind-load-cluster", "",
		"Name of the kind cluster to load image into (default: current cluster)")
	buildCmd.Flags().StringVarP(&buildDir, "project-dir", "d", "", "Build directory (default: current directory)")
	buildCmd.Flags().StringVar(&buildPlatform, "platform", "", "Target platform (e.g., linux/amd64,linux/arm64)")
}

func runBuild(_ *cobra.Command, _ []string) error {
	// Determine build directory
	buildDirectory := buildDir
	if buildDirectory == "" {
		var err error
		buildDirectory, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	imageName := buildTag
	if imageName == "" {
		// Load project manifest
		manifestManager := manifest.NewManager(buildDirectory)
		if !manifestManager.Exists() {
			return fmt.Errorf(
				"kmcp.yaml not found in %s. Run 'kmcp init' first or specify a valid path with --project-dir",
				buildDirectory,
			)
		}

		projectManifest, err := manifestManager.Load()
		if err != nil {
			return fmt.Errorf("failed to load project manifest: %w", err)
		}

		version := projectManifest.Version
		if version == "" {
			version = "latest"
		}
		imageName = fmt.Sprintf("%s:%s", strcase.KebabCase(projectManifest.Name), version)
	}

	// Execute build
	builder := build.New()
	opts := build.Options{
		ProjectDir: buildDirectory,
		Tag:        imageName,
		Platform:   buildPlatform,
		Verbose:    Verbose,
	}

	if err := builder.Build(opts); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	if buildPush {
		fmt.Printf("Pushing Docker image %s...\n", imageName)
		if err := runDocker("push", imageName); err != nil {
			return fmt.Errorf("docker push failed: %w", err)
		}
		fmt.Printf("✅ Docker image pushed successfully\n")
	}
	if buildKindLoad || buildKindLoadCluster != "" {
		fmt.Printf("Loading Docker image %s into kind cluster...\n", imageName)
		kindArgs := []string{"load", "docker-image", imageName}
		clusterName := buildKindLoadCluster
		if clusterName == "" {
			var err error
			clusterName, err = getCurrentKindClusterName()
			if err != nil {
				if Verbose {
					fmt.Printf("could not detect kind cluster name: %v, using default\n", err)
				}
				clusterName = "kind" // default to kind cluster
			}
		}

		kindArgs = append(kindArgs, "--name", clusterName)

		if err := runKind(kindArgs...); err != nil {
			return fmt.Errorf("kind load failed: %w", err)
		}
		fmt.Printf("✅ Docker image loaded into kind cluster %s\n", clusterName)
	}

	return nil
}

func runDocker(args ...string) error {
	if Verbose {
		fmt.Printf("Running: docker %s\n", strings.Join(args, " "))
	}
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runKind(args ...string) error {
	if Verbose {
		fmt.Printf("Running: kind %s\n", strings.Join(args, " "))
	}
	cmd := exec.Command("kind", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
