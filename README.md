# KMCP - Model Context Protocol Development & Deployment Platform

KMCP is a comprehensive platform for developing and deploying Model Context Protocol (MCP) servers. It provides both a powerful CLI tool for local development and a Kubernetes controller for cloud-native deployment.

## Table of Contents

- [What is MCP?](#what-is-mcp)
- [Quick Start](#quick-start)
- [CLI Tool](#cli-tool)
  - [Installation](#installation)
  - [Commands](#commands)
  - [Examples](#examples)
- [Kubernetes Controller](#kubernetes-controller)
- [Supported Frameworks](#supported-frameworks)
- [Contributing](#contributing)
- [License](#license)
- [Resources](#resources)

## What is MCP?

The Model Context Protocol (MCP) is an open standard developed by Anthropic that standardizes how AI applications provide context to Large Language Models (LLMs). Think of MCP as a universal adapter that allows AI models to seamlessly connect to various data sources and external tools.

### Key Benefits:

- **Standardized Data Access**: Consistent way for AI models to access external data
- **Tool Integration**: Connect AI assistants to business tools and internal systems  
- **Structured Responses**: Ensure consistent, structured outputs from AI models
- **Improved Context**: Give AI models the context they need for better responses
- **Vendor Independence**: Open standard that works across different AI providers

### MCP Architecture:

- **MCP Clients**: AI applications (Claude Desktop, Cursor, etc.) that consume MCP services
- **MCP Servers**: Lightweight programs that expose capabilities through standardized interfaces
- **Transport Protocols**: Communication via stdio (standard input/output) and HTTP with SSE

## Quick Start

Get started with KMCP in under 5 minutes:

```bash
# Install the CLI
go install github.com/kagent-dev/kmcp/cmd/kmcp@latest

# Create a new MCP server project
kmcp init python my-mcp-server

# Build and test your server
cd my-mcp-server
kmcp build --docker --tag my-mcp-server:latest

# Run locally for testing
kmcp run

# Your MCP server is ready to use!
```

## CLI Tool

The KMCP CLI provides a complete development experience for MCP servers with project scaffolding, build automation, and deployment tools.

### Installation

#### Option 1: Go Install (Recommended)
```bash
go install github.com/kagent-dev/kmcp/cmd/kmcp@latest
```

#### Option 2: Build from Source
```bash
git clone https://github.com/kagent-dev/kmcp.git
cd kmcp
make build-cli
```

#### Option 3: Download Binary
Download the latest binary from the [releases page](https://github.com/kagent-dev/kmcp/releases).

## CLI Commands

### `kmcp init` - Initialize New MCP Server Project

Initialize a new MCP server project with subcommands for different frameworks:

```bash
kmcp init [subcommand] [project-name] [flags]
```

**Subcommands:**
- `python [project-name]` - Initialize a Python MCP server project using fastmcp-python
- `go [project-name]` - Initialize a Go MCP server project using mcp-go

**Flags:**
- `--force` - Overwrite existing directory
- `--no-git` - Skip git initialization
- `--author` - Set project author
- `--email` - Set author email
- `--description` - Set project description
- `--non-interactive` - Use defaults without prompts
- `--namespace` - Default namespace for project resources (default: "default")
- `--verbose, -v` - Show detailed output

**Go-specific Flags:**
- `--go-module-name` - The Go module name for the project (e.g., github.com/my-org/my-project)

### `kmcp build` - Build MCP Servers

Build your MCP server with Docker support:

```bash
kmcp build [flags]
```

**Flags:**
- `--docker` - Build Docker image
- `--output, -o` - Output directory or image name
- `--tag, -t` - Docker image tag
- `--platform` - Target platform (e.g., linux/amd64, linux/arm64)
- `--project-dir, -d` - Build directory (default: current directory)
- `--verbose, -v` - Show detailed build output

### `kmcp run` - Run MCP Server Locally

Run an MCP server locally using the Model Context Protocol inspector:

```bash
kmcp run [flags]
```

**Flags:**
- `--project-dir, -d` - Project directory to use (default: current directory)
- `--verbose, -v` - Show detailed output

### `kmcp deploy` - Deploy to Kubernetes

Deploy an MCP server to Kubernetes by generating MCPServer CRDs:

```bash
kmcp deploy [name] [flags]
```

**Flags:**
- `--namespace, -n` - Kubernetes namespace (default: "default")
- `--dry-run` - Generate manifest without applying to cluster
- `--output, -o` - Output file for the generated YAML
- `--image` - Docker image to deploy (overrides build image)
- `--transport` - Transport type (stdio, http)
- `--port` - Container port (default: from project config)
- `--target-port` - Target port for HTTP transport
- `--command` - Command to run (overrides project config)
- `--args` - Command arguments
- `--env` - Environment variables (KEY=VALUE)
- `--force` - Force deployment even if validation fails
- `--file, -f` - Path to kmcp.yaml file (default: current directory)
- `--environment` - Target environment for deployment (e.g., staging, production) (default: "staging")
- `--verbose, -v` - Show detailed output

### `kmcp install` - Install KMCP Controller

Install the KMCP controller and its required Custom Resource Definitions (CRDs) on a Kubernetes cluster:

```bash
kmcp install [flags]
```

**Flags:**
- `--version` - Version of the controller to deploy (defaults to kmcp version)
- `--namespace` - Namespace for the KMCP controller (defaults to kmcp-system)
- `--registry-config` - Path to Docker registry config file
- `--verbose, -v` - Show detailed output

### `kmcp add-tool` - Add MCP Tool

Generate a boilerplate for a new MCP tool that is automatically loaded by the MCP server:

```bash
kmcp add-tool [tool-name] [flags]
```

**Flags:**
- `--description, -d` - Tool description
- `--force, -f` - Overwrite existing tool file
- `--interactive, -i` - Interactive tool creation
- `--project-dir` - Project directory (default: current directory)
- `--verbose, -v` - Show detailed output

### `kmcp secrets` - Manage Project Secrets

Manage secrets for MCP server projects and apply them to the Kubernetes environment:

```bash
kmcp secrets [subcommand] [flags]
```

**Subcommands:**
- `sync [environment]` - Sync secrets to a Kubernetes environment from a local .env file. The environment is defined in the kmcp.yaml file. 

**Sync Flags:**
- `--from-file` - Source .env file to sync from (default: ".env")
- `--dry-run` - Output the generated secret YAML instead of applying it
- `--project-dir, -d` - Project directory (default: current directory)
- `--verbose, -v` - Show detailed output

## Project Configuration (kmcp.yaml)

The `kmcp.yaml` file is the central configuration for your MCP server project. It defines project metadata, build settings, tool configurations, and secret management for different environments.

### Basic Configuration

```yaml
name: my-mcp-server
framework: fastmcp-python
version: 0.1.0
description: My MCP server for API integration
author: John Doe
email: john@example.com

# Build configuration
build:
  output: my-mcp-server
  docker:
    image: my-mcp-server:latest
    dockerfile: Dockerfile
    port: 3000
    environment:
      LOG_LEVEL: info
```

### Secret Management

KMCP supports environment-specific secret management with multiple providers. Configure secrets in your `kmcp.yaml` file to enable secure deployment. Secrets are disabled by default.

```yaml
secrets:
  # Local development environment
  local:
    enabled: false
    provider: env
    file: .env.local
  
  # Staging environment
  staging:
    enabled: true
    provider: kubernetes
    secretName: my-mcp-server-secrets-staging
    namespace: default
  
  # Production environment
  production:
    enabled: true
    provider: kubernetes
    secretName: my-mcp-server-secrets-production
    namespace: production
```

### Secret Providers

- **`env`**: Load secrets from local `.env` files for development
- **`kubernetes`**: Mounts secrets as environment variables in the MCP server container

### Using Secrets in Deployment

1. **Configure your environment** in `kmcp.yaml`:
   ```yaml
   secrets:
     staging:
       enabled: true
       provider: kubernetes
       secretName: my-app-secrets-staging
       namespace: default
   ```

2. **Create your `.env` file** with your secrets:
   ```bash
   # .env.staging
   API_KEY=your-api-key-here
   DATABASE_URL=postgresql://user:pass@host:5432/db
   ```

3. **Sync secrets to Kubernetes**:
   ```bash
   kmcp secrets sync staging --from-file .env.staging
   ```

4. **Deploy with secrets**:
   ```bash
   kmcp deploy --environment staging
   ```

The secrets will be automatically mounted as environment variables in your MCP server container during deployment.

### Examples

#### Create a FastMCP Python Project

```bash
# Interactive creation
kmcp init python my-python-server

# Non-interactive with specific options
kmcp init python my-python-server \
  --author "John Doe" \
  --email "john@example.com" \
  --description "My Python MCP server" \
  --non-interactive
```

#### Create a Go MCP Project

```bash
# Interactive creation
kmcp init go my-go-server

# Non-interactive with specific options
kmcp init go my-go-server \
  --go-module-name "github.com/my-org/my-go-server" \
  --author "John Doe" \
  --email "john@example.com" \
  --non-interactive
```

#### Build and Deploy

```bash
# Build a Docker container image for your MCP server project
kmcp build --docker --tag my-server:latest

# Deploy to Kubernetes
kmcp deploy my-server --namespace production --environment production

# Run locally for testing
kmcp run --project-dir ./my-python-server
```

#### Manage Secrets

```bash
# Sync secrets from .env file to Kubernetes
kmcp secrets sync staging --from-file .env.staging

# Add a new tool to your project
kmcp add-tool weather --description "Get weather information"
```

## Kubernetes Controller

KMCP also includes a Kubernetes controller for cloud-native MCP server deployment.

### Controller Features

- **Declarative Management**: Deploy MCP servers using Kubernetes Custom Resources
- **Container-based Servers**: Run MCP servers as containerized workloads
- **Service Discovery**: Automatic Service creation for HTTP-based MCP servers
- **Configuration Management**: ConfigMap-based configuration with environment variables

### Controller Installation

```bash
# Install CRDs
kubectl apply -f https://raw.githubusercontent.com/kagent-dev/kmcp/main/config/crd/bases/kagent.dev_mcpservers.yaml

# Deploy controller
kubectl apply -f https://raw.githubusercontent.com/kagent-dev/kmcp/main/config/default/

# Or use Helm
helm repo add kmcp https://charts.kagent.dev
helm install kmcp kmcp/kmcp --namespace kmcp-system --create-namespace
```

### MCPServer Custom Resource

```yaml
apiVersion: kagent.dev/v1alpha1
kind: MCPServer
metadata:
  name: my-mcp-server
spec:
  deployment:
    image: "my-mcp-server:latest"
    port: 3000
    cmd: "python"
    args: ["src/main.py"]
  transportType: "stdio"
```

## Supported Frameworks

KMCP supports the most popular MCP frameworks:

### FastMCP Python ⭐ (Recommended)
- **Best for**: Production Python applications
- **Features**: Comprehensive toolkit, official SDK integration
- **Use cases**: Database integration, API clients, complex workflows

## Project Structure

Generated projects follow MCP best practices:

```
my-mcp-server/
├── src/
│   ├── tools/          # Tool implementations
│   ├── resources/      # Resource handlers
│   ├── prompts/        # Prompt templates
│   └── main.py         # Server entry point
├── tests/              # Test suite
├── Dockerfile          # Container definition
├── pyproject.toml      # Python dependencies
├── .env.example        # Environment variables
└── README.md           # Project documentation
```

### MCP Go
- **Best for**: High-performance Go applications and microservices
- **Features**: Native Go SDK, type-safe tool definitions, excellent performance
- **Use cases**: High-throughput services, system-level integrations, performance-critical applications
- **Requirements**: Go 1.23 or later

Generated Go projects follow MCP best practices:

```
my-go-server/
├── main.go              # Server entry point
├── go.mod               # Go module definition
├── go.sum               # Dependency checksums
├── tools/               # Tool implementations
│   ├── all_tools.go     # Tool registration
│   ├── echo.go          # Example tool
│   └── tool.go          # Tool template
├── Dockerfile           # Container definition
├── .gitignore           # Git ignore rules
├── kmcp.yaml            # Project configuration
└── README.md            # Project documentation
```



## Development

### Building from Source

```bash
git clone https://github.com/kagent-dev/kmcp.git
cd kmcp
make build-cli
```

### Running Tests

```bash
make test
make test-e2e
```

### Development Mode

```bash
# Run CLI in development mode
go run cmd/kmcp/main.go init test-project --verbose

# Build and test
make build-cli
./bin/kmcp build --help
```

## TODO - Initial Phase Completion

The following tasks remain to complete the initial phase of KMCP:

### CLI Tool Enhancements
- [ ] **Add `dev` command** - Local development server with hot reload
- [ ] **Add `validate` command** - Protocol compliance checking
- [ ] **Add `test` command** - Run MCP server tests with inspector
- [ ] **Add `list` command** - List available frameworks and templates
- [ ] **Add `migrate` command** - Upgrade projects between framework versions

### Template System Improvements
- [ ] **🚀 MAJOR: Template System Refactoring** - See [TEMPLATE_REFACTOR.md](TEMPLATE_REFACTOR.md) for comprehensive plan
  - [ ] Opinionated modular architecture with plugin-based tools
  - [ ] Project manifest system (`kmcp.yaml`)
  - [ ] Auto-generated boilerplate separation
  - [ ] CLI tool management commands (`add-tool`, `remove-tool`, etc.)
  - [ ] **Kubernetes-native secret management** - Built-in secret handling and sanitization
  - [ ] **Multi-environment support** - Local development to production workflows
- [ ] **Multi-tool template** - Advanced template with multiple tools/resources
- [ ] **API client template** - Template for REST/GraphQL API integration
- [ ] **Template validation** - Ensure all templates build and run correctly

### Build System Enhancements
- [ ] **Multi-platform builds** - Support ARM64 and x86_64 architectures
- [ ] **Build optimization** - Improve Docker build caching and layer optimization
- [ ] **Security scanning** - Integrate vulnerability scanning in build process
- [ ] **Build profiles** - Development vs production build configurations

### Testing & Validation
- [ ] **Integration tests** - End-to-end testing of generated projects
- [ ] **Framework compliance** - Validate generated servers work with MCP clients
- [ ] **Template testing** - Automated testing of all templates
- [ ] **CLI testing** - Comprehensive test suite for CLI commands

### Documentation & Examples
- [ ] **Framework comparison guide** - Help users choose the right framework
- [ ] **Advanced usage examples** - Complex MCP server implementations
- [ ] **Troubleshooting guide** - Common issues and solutions
- [ ] **Video tutorials** - Getting started and advanced workflows

### Distribution & Packaging
- [ ] **GitHub releases** - Automated binary releases for all platforms
- [ ] **Homebrew formula** - Easy installation on macOS
- [ ] **Docker image** - Containerized CLI tool
- [ ] **Package managers** - Consider APT, YUM, Chocolatey support

### IDE & Editor Integration
- [ ] **VS Code extension** - Project templates and debugging support
- [ ] **Cursor integration** - Enhanced MCP development experience
- [ ] **Language server** - MCP protocol awareness in editors

### Performance & Reliability
- [ ] **Error handling** - Comprehensive error messages and recovery
- [ ] **Progress indicators** - Better UX for long-running operations
- [ ] **Configuration caching** - Speed up repeated operations
- [ ] **Interrupt handling** - Graceful handling of Ctrl+C

### Community & Ecosystem
- [ ] **Contributing guide** - Detailed guide for contributors
- [ ] **Issue templates** - Structured bug reports and feature requests
- [ ] **Community templates** - Accept and maintain community-contributed templates
- [ ] **Plugin system** - Allow third-party framework extensions

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Resources

- [Model Context Protocol Specification](https://spec.modelcontextprotocol.io/)
- [MCP Documentation](https://modelcontextprotocol.io/)
- [Anthropic's MCP Announcement](https://www.anthropic.com/news/model-context-protocol)
- [FastMCP Python Documentation](https://github.com/jlowin/fastmcp)
