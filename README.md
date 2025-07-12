# KMCP - Kubernetes Model Context Protocol Controller

KMCP is a Kubernetes controller that enables declarative management of Model Context Protocol (MCP) servers in Kubernetes clusters. It provides a cloud-native way to deploy, configure, and manage MCP servers that can be used by AI applications and agents to access external data sources and tools.

## What is the Model Context Protocol (MCP)?

The Model Context Protocol (MCP) is an open standard developed by Anthropic that standardizes how AI applications provide context to Large Language Models (LLMs). Think of MCP as a universal adapter that allows AI models to seamlessly connect to various data sources and external tools without requiring custom integrations for each connection.

### Key Benefits of MCP:

- **Standardized Data Access**: Provides AI models with a consistent way to access external data
- **Tool Integration**: Connects AI assistants to business tools and internal systems  
- **Structured Responses**: Ensures consistent, structured outputs from AI models
- **Improved Context**: Gives AI models the context they need for better responses
- **Vendor Independence**: Open standard that works across different AI providers

### MCP Architecture:

MCP follows a client-server architecture where:
- **MCP Clients**: AI applications (like Claude Desktop, Cursor, etc.) that consume MCP services
- **MCP Servers**: Lightweight programs that expose specific capabilities through standardized interfaces
- **Transport Protocols**: Communication methods including stdio (standard input/output) and HTTP with Server-Sent Events (SSE)

## What Does KMCP Do?

KMCP brings MCP to Kubernetes by providing:

### 1. **Declarative MCP Server Management**
- Deploy MCP servers using Kubernetes Custom Resources
- Manage server lifecycle through standard Kubernetes operations
- Support for both stdio and HTTP transport protocols

### 2. **Container-based MCP Servers**
- Run MCP servers as containerized workloads
- Leverage Kubernetes' scheduling, scaling, and reliability features
- Support for custom container images and configurations

### 3. **Service Discovery and Networking**
- Automatic Service creation for HTTP-based MCP servers
- Kubernetes-native networking and load balancing
- Secure communication within the cluster

### 4. **Configuration Management**
- ConfigMap-based configuration for MCP servers
- Environment variable injection
- Support for complex MCP server configurations

## Architecture

KMCP consists of several key components:

### MCPServer Custom Resource
The `MCPServer` CRD defines the desired state of an MCP server deployment:

```yaml
apiVersion: kagent.dev/v1alpha1
kind: MCPServer
metadata:
  name: my-mcp-server
spec:
  # Container configuration
  deployment:
    image: "my-mcp-server:latest"
    port: 3000
    cmd: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/"]
    env:
      MY_CONFIG: "value"
  
  # Transport type: stdio or http
  transportType: "stdio"
  
  # HTTP-specific configuration (when using HTTP transport)
  httpTransport:
    targetPort: 3000
    path: "/mcp"
```

### Controller Components

1. **MCPServer Controller**: Reconciles MCPServer resources and manages their lifecycle
2. **AgentGateway Translator**: Converts MCPServer specs into Kubernetes resources
3. **Status Management**: Tracks deployment status with comprehensive condition reporting

### Generated Kubernetes Resources

For each MCPServer, KMCP creates:
- **Deployment**: Runs the MCP server container(s)
- **Service**: Exposes HTTP-based MCP servers (when applicable)
- **ConfigMap**: Stores MCP server configuration
- **AgentGateway**: Provides protocol translation and routing

## Supported Transport Types

### 1. Stdio Transport
- Uses standard input/output for communication
- Ideal for local development and simple integrations
- MCP server runs as a subprocess

### 2. HTTP Transport  
- Uses HTTP with Server-Sent Events (SSE)
- Suitable for remote access and web-based integrations
- Enables load balancing and high availability

## Use Cases

### Enterprise AI Integration
- Deploy MCP servers that connect to internal databases, APIs, and services
- Provide AI agents with secure access to company data
- Enable context-aware AI applications in enterprise environments

### Development Tools
- Run MCP servers for code repositories, documentation, and development tools
- Integrate with IDEs and AI coding assistants
- Provide contextual information for software development

### Data Processing Pipelines
- Deploy MCP servers that access data lakes, warehouses, and streaming platforms
- Enable AI models to process and analyze real-time data
- Support complex data transformation workflows

## Status Conditions

KMCP provides comprehensive status reporting through standard Kubernetes conditions:

- **Accepted**: MCPServer configuration is valid
- **ResolvedRefs**: All references (images, etc.) are resolved
- **Programmed**: Kubernetes resources are created successfully  
- **Ready**: MCP server is running and ready to accept connections

## Getting Started

### Prerequisites
- Kubernetes cluster v1.11.3+
- kubectl configured to access your cluster
- Container runtime (Docker, containerd, etc.)

### Installation

1. **Install the CRDs**:
```bash
kubectl apply -f https://raw.githubusercontent.com/kagent-dev/kmcp/main/config/crd/bases/kagent.dev_mcpservers.yaml
```

2. **Deploy the Controller**:
```bash
kubectl apply -f https://raw.githubusercontent.com/kagent-dev/kmcp/main/config/default/
```

3. **Create an MCPServer**:
```bash
kubectl apply -f config/samples/v1alpha1_mcpserver.yaml
```

### Using Helm

KMCP also provides a Helm chart for easy installation:

```bash
helm repo add kmcp https://charts.kagent.dev
helm install kmcp kmcp/kmcp --namespace kmcp-system --create-namespace
```

## Examples

### Filesystem MCP Server
```yaml
apiVersion: kagent.dev/v1alpha1
kind: MCPServer
metadata:
  name: filesystem-server
spec:
  deployment:
    image: "docker.io/mcp/everything"
    port: 3000
    cmd: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/data"]
  transportType: "stdio"
```

### Database MCP Server
```yaml
apiVersion: kagent.dev/v1alpha1
kind: MCPServer
metadata:
  name: postgres-server
spec:
  deployment:
    image: "my-postgres-mcp:latest"
    port: 8080
    env:
      DATABASE_URL: "postgresql://user:pass@db:5432/mydb"
  transportType: "http"
  httpTransport:
    targetPort: 8080
    path: "/mcp"
```

## Development

### Building from Source

1. **Clone the repository**:
```bash
git clone https://github.com/kagent-dev/kmcp.git
cd kmcp
```

2. **Build and deploy**:
```bash
make docker-build docker-push IMG=<your-registry>/kmcp:tag
make deploy IMG=<your-registry>/kmcp:tag
```

### Running Tests

```bash
make test
make test-e2e
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to get started.

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
- [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)
