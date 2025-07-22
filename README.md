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
kmcp init my-mcp-server

# Build and test your server
cd my-mcp-server
kmcp build --docker --tag my-mcp-server:latest

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

Create a new MCP server project with interactive prompts:

```bash
kmcp init [project-name] [flags]
```

**Flags:**
- `--framework, -f` - Choose framework (fastmcp-python)
- `--force` - Overwrite existing directory
- `--no-git` - Skip git initialization
- `--author` - Set project author
- `--email` - Set author email
- `--version` - MCP server version (defaults to kmcp version)
- `--non-interactive` - Use defaults without prompts
- `--verbose, -v` - Show detailed output

#### `kmcp build` - Build MCP Servers

Build your MCP server with Docker support:

```bash
kmcp build [flags]
```

**Flags:**
- `--docker` - Build Docker image
- `--output, -o` - Output directory or image name
- `--tag, -t` - Docker image tag
- `--platform` - Target platform (e.g., linux/amd64, linux/arm64)
- `--dir, -d` - Build directory (default: current directory)
- `--verbose, -v` - Show detailed build output

### `kmcp deploy` - Deploy to Kubernetes

Deploy components to Kubernetes clusters:

```bash
kmcp deploy [command] [flags]
```

**Subcommands:**
- `mcp [name]` - Deploy MCP server to Kubernetes
- `controller` - Deploy KMCP controller to cluster

#### `kmcp deploy mcp` - Deploy MCP Server

Deploy an MCP server to Kubernetes by generating MCPServer CRDs:

```bash
kmcp deploy mcp [name] [flags]
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
- `--verbose, -v` - Show detailed output

#### `kmcp deploy controller` - Deploy KMCP Controller

Deploy the KMCP controller to a Kubernetes cluster using Helm:

```bash
kmcp deploy controller [flags]
```

**Flags:**
- `--version` - Version of the controller to deploy (defaults to kmcp version)
- `--namespace` - Namespace for the KMCP controller (defaults to kmcp-system)
- `--registry-config` - Path to Docker registry config file
- `--verbose, -v` - Show detailed output


### Examples

#### Create a FastMCP Python Project

```bash
# Interactive creation
kmcp init my-python-server

# Non-interactive with specific options
kmcp init my-python-server \
  --framework fastmcp-python \
  --template database \
  --author "John Doe" \
  --email "john@example.com" \
  --non-interactive
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
    args: ["-m", "my_mcp_server"]
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
