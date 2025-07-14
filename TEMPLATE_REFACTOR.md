# Template System Refactoring Plan

## Executive Summary

The current KMCP template system provides basic "hello world" examples that require significant teardown and rebuilding. This document outlines a comprehensive refactoring plan to create an **opinionated UX** that separates concerns and enables **modular customization** of MCP servers.

## Current State Problems

### 1. **Monolithic Templates**
- Single file contains everything (tools, server setup, configuration)
- Hard to understand what's boilerplate vs. business logic
- Difficult to add/remove functionality without breaking things

### 2. **Poor Developer Experience**
- Users must manually tear apart example code
- No clear patterns for adding new tools
- Dependency management is manual and error-prone
- Testing setup is ad-hoc

### 3. **Limited Customization**
- Templates are static snapshots
- No easy way to update or extend functionality
- Framework-specific knowledge required for modifications

## Proposed Solution: Opinionated Modular Architecture

### Core Principles

1. **Separation of Concerns**: Clear boundaries between framework boilerplate and business logic
2. **Convention over Configuration**: Opinionated defaults with escape hatches
3. **Modular Design**: Plugin-based architecture for tools and resources
4. **Developer Experience**: CLI-driven workflow for common tasks
5. **Framework Agnostic**: Consistent UX across all supported MCP frameworks

### Architecture Overview

```
my-mcp-server/
├── src/
│   ├── tools/              # Tool implementations (business logic)
│   │   ├── calculator.py
│   │   ├── file_manager.py
│   │   └── __init__.py
│   ├── resources/          # Resource handlers
│   │   ├── documents.py
│   │   └── __init__.py
│   ├── core/               # Framework boilerplate (generated)
│   │   ├── server.py       # MCP server setup
│   │   ├── registry.py     # Tool/resource registration
│   │   └── config.py       # Configuration management
│   └── main.py             # Entry point (minimal)
├── tests/
│   ├── tools/              # Tool-specific tests
│   └── integration/        # End-to-end tests
├── config/
│   ├── server.yaml         # Server configuration
│   └── tools.yaml          # Tool-specific config
├── kmcp.yaml               # Project manifest
└── requirements.txt        # Dependencies (auto-managed)
```

## Secret Management & Security

### Critical Security Concern

As highlighted in the security analysis at [00f.net](https://00f.net/2025/06/16/leaky-mcp-servers/), MCP servers often leak secrets directly to LLM contexts. This happens when:
- API responses contain tokens/secrets that get forwarded to LLMs
- Configuration files include plaintext secrets
- Tool implementations don't sanitize sensitive data before returning responses

### Proposed Secret Management Architecture

#### 1. **MCP-Safe Secret Handling**

**Problem**: Tools need access to secrets, but secrets must never reach LLM contexts.

**Solution**: Implement a secret proxy pattern:

```python
# tools/github_tool.py
from kmcp.core import Tool, SecretManager

class GitHubTool(Tool):
    def __init__(self, config):
        self.secrets = SecretManager(config.secrets)
    
    @Tool.method("list_repos")
    def list_repos(self) -> dict:
        # Secret is accessed locally but never returned
        token = self.secrets.get("GITHUB_TOKEN")
        
        # Make API call with secret
        response = requests.get(
            "https://api.github.com/user/repos",
            headers={"Authorization": f"Bearer {token}"}
        )
        
        # Sanitize response before returning to LLM
        repos = response.json()
        return {
            "repos": [
                {
                    "name": repo["name"],
                    "description": repo["description"],
                    "url": repo["html_url"]
                    # Note: No tokens or sensitive data
                }
                for repo in repos
            ]
        }
```

#### 2. **Kubernetes-First Secret Management**

**Configuration Hierarchy**:
```yaml
# kmcp.yaml
secrets:
  # Local development
  local:
    provider: env
    source: .env.local
  
  # Staging environment
  staging:
    provider: kubernetes
    secret_name: mcp-server-secrets-staging
    namespace: staging
  
  # Production environment
  production:
    provider: kubernetes
    secret_name: mcp-server-secrets
    namespace: default
```

**Primary Secret Provider**:
- **Kubernetes Secrets** - Native Kubernetes secret management
- **Environment variables** - Local development with `.env` files
- **Future expansion** - Architecture supports additional providers

**Kubernetes Secret Types Supported**:
- `Opaque` - General key-value pairs for API tokens, passwords
- `kubernetes.io/tls` - TLS certificates for HTTPS communication
- `kubernetes.io/basic-auth` - Basic authentication credentials
- `kubernetes.io/dockerconfigjson` - Docker registry credentials

#### 3. **Kubernetes-Native Secret Workflows**

**For Development Teams**:
```bash
# Generate Kubernetes secret manifests
kmcp generate-k8s-secrets --environment staging

# Create secrets in Kubernetes cluster
kubectl apply -f secrets/staging-secrets.yaml

# Validate secret configuration
kmcp validate-secrets --environment staging
```

**Example Development Workflow**:
```bash
# Local development
kmcp init my-server --template production
cd my-server

# Generate .env template for local development
kmcp generate-env-template
# Creates .env.example with required secret keys

# For staging/production: Generate Kubernetes secrets
kmcp generate-k8s-secrets --environment staging
# Creates secrets/staging-secrets.yaml

# Apply to cluster
kubectl apply -f secrets/staging-secrets.yaml

# Validate deployment can access secrets
kmcp validate-secrets --environment staging
```

#### 4. **Built-in Secret Sanitization**

**Automatic Response Filtering**:
```python
# Automatically applied to all tool responses
class SecretSanitizer:
    PATTERNS = [
        r'Bearer\s+[A-Za-z0-9\-_]+',  # Bearer tokens
        r'[A-Za-z0-9]{40}',           # GitHub tokens
        r'sk-[A-Za-z0-9]{48}',        # OpenAI keys
        r'xoxb-[A-Za-z0-9\-]+',       # Slack tokens
    ]
    
    def sanitize_response(self, response: dict) -> dict:
        # Recursively sanitize all string values
        return self._sanitize_recursive(response)
    
    def _sanitize_recursive(self, obj):
        if isinstance(obj, str):
            for pattern in self.PATTERNS:
                obj = re.sub(pattern, '[REDACTED]', obj)
        elif isinstance(obj, dict):
            return {k: self._sanitize_recursive(v) for k, v in obj.items()}
        elif isinstance(obj, list):
            return [self._sanitize_recursive(item) for item in obj]
        return obj
```

#### 5. **Secret Validation & Monitoring**

**Configuration Validation**:
```bash
# Validate secret configuration
kmcp validate-secrets

# Check for secret leakage in responses
kmcp audit-responses --scan-secrets

# Monitor secret usage
kmcp monitor-secrets --environment production
```

**Example Validation Output**:
```
✅ Secret configuration valid
✅ All required secrets present
⚠️  Secret 'API_KEY' unused in any tools
❌ Potential secret leak detected in github_tool.list_repos response
```

### Secret Management CLI Commands

#### New CLI Commands for Secret Management:
```bash
# Secret management
kmcp add-secret GITHUB_TOKEN --value "ghp_xyz123" --environment staging
kmcp remove-secret GITHUB_TOKEN --environment staging
kmcp list-secrets --environment staging
kmcp sync-secrets --environment staging  # Sync to Kubernetes

# Kubernetes-specific commands
kmcp generate-k8s-secrets --environment staging
kmcp apply-k8s-secrets --environment staging
kmcp diff-k8s-secrets --environment staging

# Validation and monitoring
kmcp validate-secrets --environment production
kmcp audit-responses --scan-secrets
kmcp monitor-secrets --real-time
```

### Integration with Existing Tools

#### 1. **Kubernetes Secrets Integration**
```bash
# Generate Kubernetes secret manifests
kmcp generate-k8s-secrets --environment staging
# Creates secrets/staging-secrets.yaml

# Apply secrets to cluster
kubectl apply -f secrets/staging-secrets.yaml

# Verify secrets are created
kubectl get secrets -n staging

# Update MCP deployment to use secrets
kubectl apply -f deployment/mcp-server-staging.yaml
```

#### 2. **Generated Kubernetes Manifests**
```yaml
# secrets/staging-secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: mcp-server-secrets-staging
  namespace: staging
type: Opaque
data:
  GITHUB_TOKEN: <base64-encoded>
  SLACK_WEBHOOK: <base64-encoded>
  DATABASE_URL: <base64-encoded>
---
# deployment/mcp-server-staging.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcp-server
  namespace: staging
spec:
  template:
    spec:
      containers:
      - name: mcp-server
        image: my-mcp-server:latest
        envFrom:
        - secretRef:
            name: mcp-server-secrets-staging
        env:
        - name: ENVIRONMENT
          value: "staging"
        - name: MCP_SECRET_PROVIDER
          value: "kubernetes"
```

#### 3. **MCPServer Custom Resource Integration**
```yaml
# Existing KMCP controller integration
apiVersion: kagent.dev/v1alpha1
kind: MCPServer
metadata:
  name: my-mcp-server
  namespace: staging
spec:
  deployment:
    image: "my-mcp-server:latest"
    port: 3000
    env:
      ENVIRONMENT: "staging"
      MCP_SECRET_PROVIDER: "kubernetes"
    secretRef:
      name: mcp-server-secrets-staging
  transportType: "stdio"
```

#### 4. **Local Development with Kubernetes Context**
```bash
# Local development workflow
kmcp init my-server --template production
cd my-server

# Generate .env template
kmcp generate-env-template

# Pull secrets from Kubernetes for local development
kmcp pull-secrets --from-k8s --environment staging
# Populates .env.local with decrypted values

# Start local development server
kmcp dev --use-local-secrets
```

### Security Best Practices

#### 1. **Secret Isolation**
- Secrets never appear in tool responses
- Automatic sanitization of all outputs
- Separate secret configuration from application config

#### 2. **Least Privilege Access**
- Tools only access secrets they need
- Environment-specific secret scoping
- Automatic secret rotation support

#### 3. **Audit & Monitoring**
- Secret usage tracking
- Leak detection in responses
- Compliance reporting

#### 4. **Development Workflow**
- Secure secret sharing for teams
- Local development with `.env.local`
- Production secrets via secure providers

## Implementation Plan

### Phase 1: Core Infrastructure

#### 1.1 Project Manifest System
**File**: `kmcp.yaml`
```yaml
name: my-mcp-server
framework: fastmcp-python
version: 1.0.0
description: My MCP server

tools:
  calculator:
    enabled: true
    config:
      precision: 2
  file_manager:
    enabled: true
    config:
      allowed_paths: ["/tmp", "/data"]

resources:
  documents:
    enabled: true
    config:
      formats: ["pdf", "txt", "md"]

dependencies:
  auto_manage: true
  extra:
    - numpy
    - pandas
```

#### 1.2 Tool Plugin System
**Convention**: Each tool is a self-contained module with:
- Implementation file (`tools/calculator.py`)
- Configuration schema
- Dependencies declaration
- Test suite

**Example Tool Structure**:
```python
# tools/calculator.py
from kmcp.core import Tool, config

class CalculatorTool(Tool):
    """Calculator tool with basic arithmetic operations."""
    
    @config.setting("precision", default=2)
    def precision(self) -> int:
        return self.config.get("precision", 2)
    
    @Tool.method("add")
    def add(self, a: float, b: float) -> float:
        """Add two numbers."""
        return round(a + b, self.precision)
    
    @Tool.method("subtract")
    def subtract(self, a: float, b: float) -> float:
        """Subtract two numbers."""
        return round(a - b, self.precision)
    
    # Tool metadata
    dependencies = ["math"]
    config_schema = {
        "precision": {"type": "integer", "default": 2}
    }
```

#### 1.3 Auto-Generated Boilerplate
**File**: `src/core/server.py` (generated, not edited by user)
```python
# This file is auto-generated by KMCP
# Do not edit directly - use 'kmcp generate' to update

from kmcp.framework.fastmcp import FastMCPServer
from .registry import tool_registry, resource_registry
from .config import load_config

def create_server():
    config = load_config()
    server = FastMCPServer(config.server)
    
    # Auto-register tools
    for tool_name, tool_class in tool_registry.items():
        if config.tools.get(tool_name, {}).get("enabled", True):
            server.register_tool(tool_class(config.tools[tool_name]))
    
    return server
```

### Phase 2: CLI Integration

#### 2.1 Tool Management Commands
```bash
# Add new tool (interactive)
kmcp add-tool calculator

# Add tool with template
kmcp add-tool database --template postgresql

# Remove tool
kmcp remove-tool calculator

# List available tools
kmcp list-tools

# Update tool configuration
kmcp configure-tool calculator --precision 4
```

#### 2.2 Code Generation
```bash
# Generate boilerplate (after config changes)
kmcp generate

# Generate new tool scaffold
kmcp generate-tool weather_api --type api-client

# Generate resource handler
kmcp generate-resource documents --type filesystem
```

#### 2.3 Development Workflow
```bash
# Start development server with hot reload
kmcp dev

# Run tool-specific tests
kmcp test calculator

# Validate tool implementations
kmcp validate
```

### Phase 3: Framework Abstraction

#### 3.1 Unified Tool Interface
Create framework-agnostic tool interface that works across FastMCP Python, FastMCP TypeScript, etc.

#### 3.2 Template Generation
```bash
# Generate framework-specific implementation
kmcp generate --framework fastmcp-python
kmcp generate --framework fastmcp-typescript

# Migrate between frameworks
kmcp migrate --from fastmcp-python --to fastmcp-typescript
```

### Phase 4: Advanced Features

#### 4.1 Tool Marketplace
```bash
# Install community tools
kmcp install-tool slack-integration

# Publish tool to marketplace
kmcp publish-tool calculator
```

#### 4.2 Secret Management & Configuration
- **Secret Management Integration**: Built-in support for popular secret stores
- **Environment-specific configurations**: Development, staging, production configs
- **MCP-safe secret handling**: Prevent secret leakage to LLM contexts
- **Validation and type checking**: Configuration schema validation

#### 4.3 Testing Infrastructure
- Automated test generation for tools
- MCP protocol compliance testing
- Performance benchmarking

## User Experience Improvements

### Before (Current)
```bash
# User workflow
kmcp init my-server
cd my-server
# Edit generated files manually
# Figure out how to add new tools
# Manually manage dependencies
# Write tests from scratch
```

### After (Proposed)
```bash
# Streamlined workflow
kmcp init my-server --template production
cd my-server
kmcp add-tool calculator
kmcp add-tool file-manager
kmcp configure-tool calculator --precision 4
kmcp dev  # Start development server
kmcp test  # Run all tests
kmcp build --docker
```

## Benefits

### 1. **Faster Development**
- No need to understand framework internals
- Add tools with single command
- Automatic dependency management
- Built-in testing infrastructure

### 2. **Better Maintainability**
- Clear separation of concerns
- Modular architecture
- Auto-generated boilerplate
- Consistent project structure

### 3. **Easier Customization**
- Plugin-based tool system
- Configuration-driven behavior
- Framework migration support
- Community tool marketplace

### 4. **Professional Quality**
- Production-ready defaults
- Security best practices
- Performance optimizations
- Comprehensive testing

## Implementation Timeline

### Milestone 1: Core Infrastructure (4 weeks)
- [ ] Project manifest system
- [ ] Tool plugin architecture
- [ ] Auto-generated boilerplate
- [ ] Basic CLI commands
- [ ] **Secret management foundation** - SecretManager, sanitization patterns
- [ ] **Local secret handling** - .env file support, secret validation

### Milestone 2: CLI Integration (3 weeks)
- [ ] Tool management commands
- [ ] Code generation
- [ ] Development workflow
- [ ] Testing infrastructure
- [ ] **Secret management CLI** - add-secret, validate-secrets, audit-responses
- [ ] **Secret sharing** - EnvShare integration, team workflows

### Milestone 3: Framework Abstraction (3 weeks)
- [ ] Unified tool interface
- [ ] Multi-framework support
- [ ] Migration tools
- [ ] Template system

### Milestone 4: Advanced Features (4 weeks)
- [ ] Tool marketplace
- [ ] Configuration management
- [ ] Performance optimizations
- [ ] Documentation and examples
- [ ] **Advanced Kubernetes integration** - RBAC, namespaces, secret rotation
- [ ] **Secret monitoring** - Usage tracking, leak detection, compliance
- [ ] **Multi-environment support** - Advanced staging/production workflows

## Success Metrics

### Developer Experience
- **Time to first working server**: < 5 minutes
- **Time to add new tool**: < 2 minutes
- **Framework migration time**: < 30 minutes

### Code Quality
- **Boilerplate reduction**: 80% less manual code
- **Test coverage**: Automatic 90%+ coverage
- **Configuration errors**: Eliminated through validation

### Security
- **Secret leak prevention**: 100% automated sanitization
- **Secret audit compliance**: Automated compliance reporting
- **Secret rotation**: Support for automatic secret rotation

### Community Adoption
- **Template usage**: 90% of new projects use templates
- **Tool sharing**: Active community tool marketplace
- **Framework coverage**: All major MCP frameworks supported

## Risk Mitigation

### Technical Risks
- **Framework compatibility**: Maintain adapter pattern for framework changes
- **Performance overhead**: Benchmark and optimize plugin system
- **Complexity creep**: Maintain simple APIs with advanced options

### Security Risks
- **Secret leakage**: Implement comprehensive sanitization and monitoring
- **Secret storage**: Use industry-standard secret providers, avoid plaintext storage
- **Compliance requirements**: Support enterprise audit and compliance features

### Adoption Risks
- **Migration path**: Provide automated migration from current templates
- **Learning curve**: Comprehensive documentation and examples
- **Ecosystem fragmentation**: Maintain backward compatibility

## Conclusion

This refactoring transforms KMCP from a basic scaffolding tool into a comprehensive MCP development platform. By providing an opinionated, modular architecture with **Kubernetes-native secret management**, developers can focus on building MCP tools rather than managing framework boilerplate and security concerns.

The **security-first, Kubernetes-native design** ensures that secrets are handled safely from local development through production deployment. The modular architecture allows users to start simple and progressively add complexity as needed, while the CLI integration provides a professional development experience that scales from prototypes to production Kubernetes deployments.

**Key advantages of the Kubernetes-first approach:**
- **Native integration** with existing Kubernetes infrastructure
- **Seamless secret management** from local development to production
- **Enterprise-ready security** with built-in secret sanitization
- **Scalable deployment** leveraging Kubernetes orchestration
- **Future-proof architecture** that can expand to other secret providers as needed 