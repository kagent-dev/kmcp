package templates

// getFastMCPPythonFiles returns the file templates for FastMCP Python projects
func (g *Generator) getFastMCPPythonFiles(templateType string, data map[string]interface{}) map[string]string {
	files := map[string]string{
		"pyproject.toml":  g.getFastMCPPythonPyprojectToml(templateType, data),
		".python-version": g.getFastMCPPythonPythonVersion(templateType, data),
		"README.md":       g.getFastMCPPythonReadme(templateType, data),
		"Dockerfile":      g.getFastMCPPythonDockerfile(templateType, data),
		".gitignore":      g.getFastMCPPythonGitignore(templateType, data),
		".env.example":    g.getFastMCPPythonEnvExample(templateType, data),

		// Core application structure
		"src/__init__.py": "",
		"src/main.py":     g.getFastMCPPythonMain(templateType, data),

		// Core framework (dynamic loading implementation)
		"src/core/__init__.py": g.getFastMCPPythonCoreInit(templateType, data),
		"src/core/server.py":   g.getFastMCPPythonCoreServer(templateType, data),
		"src/core/utils.py":    g.getFastMCPPythonCoreUtils(templateType, data),

		// Tools directory with example tools
		"src/tools/__init__.py": g.getFastMCPPythonToolsInit(templateType, data),
		"src/tools/echo.py":     g.getFastMCPPythonExampleEcho(templateType, data),

		// Tests
		"tests/__init__.py":       "",
		"tests/test_tools.py":     g.getFastMCPPythonTestTools(templateType, data),
		"tests/test_server.py":    g.getFastMCPPythonTestServer(templateType, data),
		"tests/test_discovery.py": g.getFastMCPPythonTestDiscovery(templateType, data),
	}

	return files
}

// getFastMCPPythonPyprojectToml generates the pyproject.toml template with FastMCP dependency
func (g *Generator) getFastMCPPythonPyprojectToml(_ string, _ map[string]interface{}) string {
	return `[project]
name = "{{.ProjectNameKebab}}"
version = "0.1.0"
description = "{{.ProjectName}} MCP server built with FastMCP"
authors = [
    {name = "{{.Author}}", email = "{{.Email}}"}
]
readme = "README.md"
requires-python = ">=3.10"
dependencies = [
    "fastmcp>=0.1.0",
    "pydantic>=2.0.0",
    "pyyaml>=6.0.0",
]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
packages = ["src"]

[project.scripts]
{{.ProjectNameKebab}}-server = "src.main:main"

[tool.uv]
dev-dependencies = [
    "pytest>=7.0.0",
    "pytest-asyncio>=0.21.0",
    "black>=22.0.0",
    "mypy>=1.0.0",
    "ruff>=0.1.0",
]

[tool.black]
line-length = 88
target-version = ['py310']

[tool.ruff]
line-length = 88
target-version = "py310"
select = ["E", "F", "I", "N", "W", "UP"]

[tool.mypy]
python_version = "3.10"
warn_return_any = true
warn_unused_configs = true
disallow_untyped_defs = true`
}

// getFastMCPPythonMain generates the main entry point with dynamic loading
func (g *Generator) getFastMCPPythonMain(_ string, _ map[string]interface{}) string {
	return `#!/usr/bin/env python3
"""{{.ProjectName}} MCP server with dynamic tool loading.

This server automatically discovers and loads tools from the src/tools/ directory.
Each tool file should contain a function decorated with @mcp.tool().
"""

import sys
from pathlib import Path

# Add src to Python path
sys.path.insert(0, str(Path(__file__).parent))

from core.server import DynamicMCPServer


def main() -> None:
    """Main entry point for the MCP server."""
    try:
        # Create server with dynamic tool loading
        server = DynamicMCPServer(
            name="{{.ProjectName}}",
            tools_dir="src/tools"
        )
        
        # Load tools and start server
        server.load_tools()
        server.run()
        
    except KeyboardInterrupt:
        print("\nShutting down server...")
    except Exception as e:
        print(f"Server error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
`
}

// getFastMCPPythonCoreInit generates the core package init
func (g *Generator) getFastMCPPythonCoreInit(_ string, _ map[string]interface{}) string {
	return `"""Core framework for {{.ProjectName}} MCP server.

This package provides the dynamic tool loading system that automatically
discovers and registers tools from the src/tools/ directory.
"""

from .server import DynamicMCPServer

__all__ = ["DynamicMCPServer"]
`
}

// getFastMCPPythonCoreServer generates the dynamic server implementation
func (g *Generator) getFastMCPPythonCoreServer(_ string, _ map[string]interface{}) string {
	return `"""Dynamic MCP server implementation with automatic tool discovery.

This server automatically discovers and loads tools from the tools directory.
Each tool file should contain a function decorated with @mcp.tool().
"""

import os
import sys
import importlib.util
from pathlib import Path
from typing import Dict, Any, List, Callable

import yaml
from fastmcp import FastMCP

from .utils import load_config, get_shared_config


# Global FastMCP instance for tools to import
mcp = FastMCP(name="Dynamic Server")


class DynamicMCPServer:
    """MCP server with dynamic tool loading capabilities."""
    
    def __init__(self, name: str, tools_dir: str = "src/tools"):
        """Initialize the dynamic MCP server.
        
        Args:
            name: Server name
            tools_dir: Directory containing tool files
        """
        global mcp
        self.name = name
        self.tools_dir = Path(tools_dir)
        self.config = self._load_config()
        
        # Update global FastMCP instance
        mcp = FastMCP(name=self.name)
        self.mcp = mcp
        
        # Track loaded tools
        self.loaded_tools: List[str] = []
        
    def _load_config(self) -> Dict[str, Any]:
        """Load configuration from kmcp.yaml."""
        return load_config("kmcp.yaml")
    
    def load_tools(self) -> None:
        """Discover and load all tools from the tools directory."""
        if not self.tools_dir.exists():
            print(f"Tools directory {self.tools_dir} does not exist")
            return
            
        # Find all Python files in tools directory
        tool_files = list(self.tools_dir.glob("*.py"))
        tool_files = [f for f in tool_files if f.name != "__init__.py"]
        
        if not tool_files:
            print(f"No tool files found in {self.tools_dir}")
            return
            
        loaded_count = 0
        
        for tool_file in tool_files:
            try:
                # Simply import the module - tools auto-register via @mcp.tool() decorator
                tool_name = tool_file.stem
                if self._import_tool_module(tool_file, tool_name):
                    self.loaded_tools.append(tool_name)
                    loaded_count += 1
                    print(f"✅ Loaded tool module: {tool_name}")
                else:
                    print(f"❌ Failed to load tool module: {tool_name}")
                    
            except Exception as e:
                print(f"❌ Error loading tool {tool_file.name}: {e}")
                # Fail fast - if any tool fails to load, stop the server
                sys.exit(1)
                
        print(f"📦 Successfully loaded {loaded_count} tools")
        
        if loaded_count == 0:
            print("⚠️  No tools loaded. Server starting without tools.")
    
    def _import_tool_module(self, tool_file: Path, tool_name: str) -> bool:
        """Import a tool module, which auto-registers tools via decorators.
        
        Args:
            tool_file: Path to the tool file
            tool_name: Name of the tool (same as filename)
            
        Returns:
            True if module was imported successfully
        """
        try:
            # Load the module
            spec = importlib.util.spec_from_file_location(tool_name, tool_file)
            if spec is None or spec.loader is None:
                return False
                
            module = importlib.util.module_from_spec(spec)
            
            # Add to sys.modules so it can be imported by other modules
            sys.modules[f"tools.{tool_name}"] = module
            
            # Execute the module - this will trigger @mcp.tool() decorators
            spec.loader.exec_module(module)
            
            return True
            
        except Exception as e:
            print(f"Error importing {tool_file}: {e}")
            return False
    

    
    def run(self) -> None:
        """Run the FastMCP server."""
        if not self.loaded_tools:
            print("⚠️  No tools loaded. Server starting without tools.")
        
        self.mcp.run()
`
}

// getFastMCPPythonCoreUtils generates shared utilities
func (g *Generator) getFastMCPPythonCoreUtils(_ string, _ map[string]interface{}) string {
	return `"""Shared utilities for {{.ProjectName}} MCP server."""

import os
from pathlib import Path
from typing import Dict, Any

import yaml


def load_config(config_path: str) -> Dict[str, Any]:
    """Load configuration from YAML file.
    
    Args:
        config_path: Path to the configuration file
        
    Returns:
        Configuration dictionary
    """
    try:
        with open(config_path, 'r') as f:
            return yaml.safe_load(f) or {}
    except FileNotFoundError:
        return {}
    except Exception as e:
        print(f"Error loading config from {config_path}: {e}")
        return {}


def get_shared_config() -> Dict[str, Any]:
    """Get shared configuration that tools can access.
    
    Returns:
        Shared configuration dictionary
    """
    config = load_config("kmcp.yaml")
    return config.get("tools", {})


def get_tool_config(tool_name: str) -> Dict[str, Any]:
    """Get configuration for a specific tool.
    
    Args:
        tool_name: Name of the tool
        
    Returns:
        Tool-specific configuration
    """
    shared_config = get_shared_config()
    return shared_config.get(tool_name, {})


def get_env_var(key: str, default: str = "") -> str:
    """Get environment variable with fallback.
    
    Args:
        key: Environment variable key
        default: Default value if not found
        
    Returns:
        Environment variable value or default
    """
    return os.environ.get(key, default)
`
}

// getFastMCPPythonToolsInit generates the tools package init
func (g *Generator) getFastMCPPythonToolsInit(_ string, _ map[string]interface{}) string {
	return `"""Tools package for {{.ProjectName}} MCP server.

This file is automatically generated by the dynamic loading system.
Do not edit manually - it will be overwritten when tools are loaded.
"""

from .echo import echo

__all__ = ["echo"]
`
}

// getFastMCPPythonExampleEcho generates an example echo tool
func (g *Generator) getFastMCPPythonExampleEcho(_ string, _ map[string]interface{}) string {
	return `"""Example echo tool for {{.ProjectName}} MCP server.

This is an example tool showing the basic structure for FastMCP tools.
Each tool file should contain a function decorated with @mcp.tool().
"""

from core.server import mcp
from core.utils import get_tool_config


@mcp.tool()
def echo(message: str) -> str:
    """Echo a message back to the client.
    
    Args:
        message: The message to echo
        
    Returns:
        The echoed message with any configured prefix
    """
    # Get tool-specific configuration
    config = get_tool_config("echo")
    prefix = config.get("prefix", "")
    
    # Return the message with optional prefix
    return f"{prefix}{message}" if prefix else message
`
}

// getFastMCPPythonKmcpConfig generates the kmcp.yaml configuration
//
//nolint:unused
func (g *Generator) getFastMCPPythonKmcpConfig(_ string, _ map[string]interface{}) string {
	return `# {{.ProjectName}} MCP Server Configuration

server:
  name: "{{.ProjectName}}"
  version: "0.1.0"
  description: "{{.ProjectName}} MCP server with dynamic tool loading"

# Tool-specific configuration
tools:
  # Example tool configuration
  echo:
    prefix: "[{{.ProjectName}}] "
  
  # Add configuration for your tools here
  # weather:
  #   api_key_env: "WEATHER_API_KEY"
  #   base_url: "https://api.openweathermap.org/data/2.5"
  #   timeout: 30
  
  # database:
  #   connection_string_env: "DATABASE_URL"
  #   max_connections: 10
  
  # file_processor:
  #   max_file_size: "10MB"
  #   allowed_extensions: [".txt", ".csv", ".json"]

# Global settings
global:
  # Maximum concurrent tool executions
  max_concurrent: 10
  
  # Default timeout for tools (seconds)
  default_timeout: 30
  
  # Enable debug logging
  debug: false
`
}

// getFastMCPPythonReadme generates the README with dynamic loading info
func (g *Generator) getFastMCPPythonReadme(_ string, _ map[string]interface{}) string {
	return `# {{.ProjectName}}

{{.ProjectName}} is a Model Context Protocol (MCP) server built with FastMCP featuring dynamic tool loading.

## Features

- **Dynamic Tool Loading**: Tools are automatically discovered and loaded from ` + "`src/tools/`" + `
- **One Tool Per File**: Each tool is a single file with a function matching the filename
- **FastMCP Integration**: Leverages FastMCP for robust MCP protocol handling
- **Configuration Management**: Tool-specific configuration via ` + "`kmcp.yaml`" + `
- **Fail-Fast**: Server won't start if any tool fails to load
- **Auto-Generated Tests**: Automatic test generation for tool validation

## Project Structure

` + "```" + `
src/
├── tools/              # Tool implementations (one file per tool)
│   ├── echo.py         # Example echo tool
│   └── __init__.py     # Auto-generated tool registry
├── core/               # Dynamic loading framework
│   ├── server.py       # Dynamic MCP server
│   └── utils.py        # Shared utilities
└── main.py             # Entry point
kmcp.yaml               # Configuration file
tests/                  # Generated tests
` + "```" + `

## Quick Start

### Option 1: Local Development (with Python/uv)

1. **Install Dependencies**:
   ` + "```bash" + `
   uv sync
   ` + "```" + `

2. **Run the Server**:
   ` + "```bash" + `
   uv run python src/main.py
   ` + "```" + `

3. **Add New Tools**:
   ` + "```bash" + `
   # Create a new tool (no tool types needed!)
   kmcp add-tool weather
   
   # The tool file will be created at src/tools/weather.py
   # Edit it to implement your tool logic
   ` + "```" + `

### Option 2: Docker-Only Development (no local Python/uv required)

1. **Build Docker Image**:
   ` + "```bash" + `
   kmcp build --docker --verbose
   ` + "```" + `

2. **Run in Container**:
   ` + "```bash" + `
   docker run -i {{.ProjectNameKebab}}:latest
   ` + "```" + `

3. **Deploy to Kubernetes**:
   ` + "```bash" + `
   kmcp deploy --apply
   ` + "```" + `

4. **Add New Tools**:
   ` + "```bash" + `
   # Create a new tool
   kmcp add-tool weather
   
   # Edit the tool file, then rebuild
   kmcp build --docker
   ` + "```" + `

## Creating Tools

### Basic Tool Structure

Each tool is a Python file in ` + "`src/tools/`" + ` containing a function decorated with ` + "`@mcp.tool()`" + `:

` + "```python" + `
# src/tools/weather.py
from core.server import mcp
from core.utils import get_tool_config, get_env_var

@mcp.tool()
def weather(location: str) -> str:
    \"\"\"Get weather information for a location.\"\"\"
    
    # Get tool configuration
    config = get_tool_config("weather")
    api_key = get_env_var(config.get("api_key_env", "WEATHER_API_KEY"))
    base_url = config.get("base_url", "https://api.openweathermap.org/data/2.5")
    
    # TODO: Implement weather API call
    return f"Weather for {location}: Sunny, 72°F"
` + "```" + `

### Tool Examples

The generated tool template includes commented examples for common patterns:

` + "```python" + `
# HTTP API calls
# async with httpx.AsyncClient() as client:
#     response = await client.get(f"{base_url}/weather?q={location}&appid={api_key}")
#     return response.json()

# Database operations  
# async with asyncpg.connect(connection_string) as conn:
#     result = await conn.fetchrow("SELECT * FROM weather WHERE location = $1", location)
#     return dict(result)

# File processing
# with open(file_path, 'r') as f:
#     content = f.read()
#     return {"content": content, "size": len(content)}
` + "```" + `

## Configuration

Configure tools in ` + "`kmcp.yaml`" + `:

` + "```yaml" + `
tools:
  weather:
    api_key_env: "WEATHER_API_KEY"
    base_url: "https://api.openweathermap.org/data/2.5"
    timeout: 30
  
  database:
    connection_string_env: "DATABASE_URL"
    max_connections: 10
` + "```" + `

## Testing

Run the generated tests to verify your tools load correctly:

` + "```bash" + `
uv run pytest tests/
` + "```" + `

## Development

### Adding Dependencies

Update ` + "`pyproject.toml`" + ` and run:

` + "```bash" + `
uv sync
` + "```" + `

### Code Quality

` + "```bash" + `
uv run black .
uv run ruff check .
uv run mypy .
` + "```" + `

## Development Workflows

### Docker-First Development

This project supports development without requiring Python/uv installed locally:

- **Automatic Lockfile Generation**: If ` + "`uv.lock`" + ` doesn't exist, the Docker build will generate it 
  automatically
- **Self-Contained Builds**: Everything needed is included in the Docker image
- **Kubernetes-Ready**: Built images can be deployed directly to Kubernetes
- **Consistent Environment**: Same environment across development, testing, and production

### Traditional Python Development

If you prefer local development with Python/uv:

- **Fast Iteration**: Direct Python execution without container overhead
- **Rich Tooling**: Full access to Python development tools
- **Debugging**: Easier to debug and profile code

## Deployment

### Docker

` + "```bash" + `
# Build image (handles lockfile automatically)
kmcp build --docker

# Run container
docker run -i {{.ProjectNameKebab}}:latest
` + "```" + `

### Kubernetes

` + "```bash" + `
# Deploy to Kubernetes
kmcp deploy --apply

# Check deployment status
kubectl get mcpserver {{.ProjectNameKebab}}
` + "```" + `

### MCP Client Configuration

` + "```json" + `
{
  "mcpServers": {
    "{{.ProjectNameKebab}}": {
      "command": "python",
      "args": ["src/main.py"],
      "cwd": "/path/to/{{.ProjectNameKebab}}"
    }
  }
}
` + "```" + `

## License

This project is licensed under the MIT License.
`
}

// Continue with the rest of the template functions...
// getFastMCPPythonPythonVersion, getFastMCPPythonDockerfile, getFastMCPPythonGitignore,
// getFastMCPPythonEnvExample remain the same

func (g *Generator) getFastMCPPythonPythonVersion(_ string, _ map[string]interface{}) string {
	return `3.11`
}

func (g *Generator) getFastMCPPythonDockerfile(_ string, _ map[string]interface{}) string {
	return `# Multi-stage build for {{.ProjectName}} MCP server using uv
FROM python:3.11-slim as builder

# Install uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# Set working directory
WORKDIR /app

# Copy dependency files first for layer caching
COPY pyproject.toml .python-version ./
COPY README.md ./

# Copy lockfile if it exists, otherwise generate it
COPY uv.loc[k] ./
RUN if [ -f uv.lock ]; then \
        echo "Using existing lockfile"; \
        uv sync --frozen --no-dev --no-cache; \
    else \
        echo "Generating lockfile and installing dependencies"; \
        uv sync --no-dev --no-cache; \
    fi

# Copy source code
COPY src/ ./src/
COPY kmcp.yaml ./

# Production stage
FROM python:3.11-slim

# Install uv in production
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# Create non-root user
RUN groupadd -r mcpuser && useradd -r -g mcpuser mcpuser

# Set working directory
WORKDIR /app

# Copy virtual environment and application from builder
COPY --from=builder /app/.venv /app/.venv
COPY --from=builder /app/src /app/src
COPY --from=builder /app/kmcp.yaml /app/kmcp.yaml
COPY --from=builder /app/pyproject.toml /app/pyproject.toml

# Make sure scripts in .venv are usable
ENV PATH="/app/.venv/bin:$PATH"

# Install runtime dependencies only
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Change ownership to non-root user
RUN chown -R mcpuser:mcpuser /app

# Switch to non-root user
USER mcpuser

# Expose port (if needed for HTTP transport)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "import src.main; print('healthy')"

# Set environment variables
ENV PYTHONPATH=/app
ENV PYTHONUNBUFFERED=1

# Default command
CMD ["python", "src/main.py"]`
}

func (g *Generator) getFastMCPPythonGitignore(_ string, _ map[string]interface{}) string {
	return `# Byte-compiled / optimized / DLL files
__pycache__/
*.py[cod]
*$py.class

# C extensions
*.so

# Distribution / packaging
.Python
build/
develop-eggs/
dist/
downloads/
eggs/
.eggs/
lib/
lib64/
parts/
sdist/
var/
wheels/
pip-wheel-metadata/
share/python-wheels/
*.egg-info/
.installed.cfg
*.egg
MANIFEST

# PyInstaller
*.manifest
*.spec

# Installer logs
pip-log.txt
pip-delete-this-directory.txt

# Unit test / coverage reports
htmlcov/
.tox/
.nox/
.coverage
.coverage.*
.cache
nosetests.xml
coverage.xml
*.cover
*.py,cover
.hypothesis/
.pytest_cache/

# Translations
*.mo
*.pot

# Django stuff:
*.log
local_settings.py
db.sqlite3
db.sqlite3-journal

# Flask stuff:
instance/
.webassets-cache

# Scrapy stuff:
.scrapy

# Sphinx documentation
docs/_build/

# PyBuilder
target/

# Jupyter Notebook
.ipynb_checkpoints

# IPython
profile_default/
ipython_config.py

# pyenv
.python-version

# pipenv
#Pipfile.lock

# PEP 582; used by e.g. github.com/David-OConnor/pyflow
__pypackages__/

# Celery stuff
celerybeat-schedule
celerybeat.pid

# SageMath parsed files
*.sage.py

# Environments
.env
.env.local
.venv
env/
venv/
ENV/
env.bak/
venv.bak/

# Spyder project settings
.spyderproject
.spyproject

# Rope project settings
.ropeproject

# mkdocs documentation
/site

# mypy
.mypy_cache/
.dmypy.json
dmypy.json

# Pyre type checker
.pyre/

# KMCP specific
.mcpbuilder.yaml`
}

func (g *Generator) getFastMCPPythonEnvExample(_ string, _ map[string]interface{}) string {
	return `# {{.ProjectName}} Environment Variables
# Copy this file to .env.local and fill in actual values

# Example API keys (configure these in kmcp.yaml under tools)
# WEATHER_API_KEY=your-weather-api-key-here
# DATABASE_URL=postgresql://user:password@localhost:5432/database
# OPENAI_API_KEY=your-openai-api-key-here

# Server configuration
# MCP_SERVER_HOST=127.0.0.1
# MCP_SERVER_PORT=8080
# MCP_LOG_LEVEL=INFO
# MCP_DEBUG=false

# Tool-specific configuration
# WEATHER_TIMEOUT=30
# DB_MAX_CONNECTIONS=10
# FILE_MAX_SIZE=10485760  # 10MB in bytes
`
}

// Test functions
func (g *Generator) getFastMCPPythonTestTools(_ string, _ map[string]interface{}) string {
	return `"""Generated tests for {{.ProjectName}} MCP server tools.

This file is automatically generated to test that all tools can be loaded
and executed successfully.
"""

import sys
from pathlib import Path
import pytest

# Add src to Python path
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from core.server import DynamicMCPServer


class TestToolLoading:
    """Test that all tools can be loaded successfully."""
    
    def test_server_initialization(self):
        """Test that the server can be initialized."""
        server = DynamicMCPServer(name="Test Server", tools_dir="src/tools")
        assert server is not None
        assert server.name == "Test Server"
    
    def test_tool_discovery(self):
        """Test that tools can be discovered."""
        server = DynamicMCPServer(name="Test Server", tools_dir="src/tools")
        
        # Load tools without failing
        try:
            server.load_tools()
            assert True  # If we get here, loading succeeded
        except SystemExit:
            pytest.fail("Tool loading failed - server exited")
    
    def test_loaded_tools_count(self):
        """Test that expected tools are loaded."""
        server = DynamicMCPServer(name="Test Server", tools_dir="src/tools")
        server.load_tools()
        
        # At minimum, we should have the echo tool
        assert len(server.loaded_tools) >= 1
        assert "echo" in server.loaded_tools
    
    def test_tool_functions_callable(self):
        """Test that loaded tool functions are callable."""
        server = DynamicMCPServer(name="Test Server", tools_dir="src/tools")
        server.load_tools()
        
        for tool_name, tool_func in server.loaded_tools.items():
            assert callable(tool_func), f"Tool {tool_name} is not callable"


class TestEchoTool:
    """Test the example echo tool."""
    
    def test_echo_tool_exists(self):
        """Test that the echo tool exists and can be loaded."""
        server = DynamicMCPServer(name="Test Server", tools_dir="src/tools")
        server.load_tools()
        
        assert "echo" in server.loaded_tools
    
    def test_echo_tool_function(self):
        """Test that the echo tool function works."""
        # Import the echo function directly
        from tools.echo import echo
        
        result = echo("Hello, World!")
        assert isinstance(result, str)
        assert "Hello, World!" in result
`
}

func (g *Generator) getFastMCPPythonTestServer(_ string, _ map[string]interface{}) string {
	return `"""Tests for {{.ProjectName}} MCP server core functionality."""

import sys
from pathlib import Path
import pytest

# Add src to Python path
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from core.server import DynamicMCPServer
from core.utils import load_config, get_tool_config


class TestDynamicMCPServer:
    """Test the dynamic MCP server functionality."""
    
    def test_server_initialization(self):
        """Test server initialization."""
        server = DynamicMCPServer(name="Test Server", tools_dir="src/tools")
        assert server.name == "Test Server"
        assert server.tools_dir == Path("src/tools")
    
    def test_server_with_nonexistent_tools_dir(self):
        """Test server behavior with non-existent tools directory."""
        server = DynamicMCPServer(name="Test Server", tools_dir="nonexistent")
        
        # Should not raise exception, just print message
        server.load_tools()
        assert len(server.loaded_tools) == 0
    
    def test_load_config(self):
        """Test configuration loading."""
        config_data = """
        server:
          name: "Test Server"
        tools:
          echo:
            prefix: "[TEST] "
        """
        
        with patch("builtins.open", mock_open(read_data=config_data)):
            config = load_config("test.yaml")
            assert config["server"]["name"] == "Test Server"
            assert config["tools"]["echo"]["prefix"] == "[TEST] "
    
    def test_get_tool_config(self):
        """Test tool-specific configuration retrieval."""
        with patch("core.utils.load_config") as mock_load:
            mock_load.return_value = {
                "tools": {
                    "echo": {"prefix": "[TEST] "},
                    "weather": {"api_key_env": "WEATHER_API_KEY"}
                }
            }
            
            echo_config = get_tool_config("echo")
            assert echo_config["prefix"] == "[TEST] "
            
            weather_config = get_tool_config("weather")
            assert weather_config["api_key_env"] == "WEATHER_API_KEY"
            
            # Test non-existent tool
            empty_config = get_tool_config("nonexistent")
            assert empty_config == {}


class TestToolLoading:
    """Test the tool loading mechanism."""
    
    def test_tool_function_detection(self):
        """Test that tool functions are properly detected."""
        server = DynamicMCPServer(name="Test Server", tools_dir="src/tools")
        
        # This should load actual tools from the tools directory
        server.load_tools()
        
        # Verify that tools were loaded
        assert len(server.loaded_tools) > 0
        
        # Verify that echo tool specifically was loaded
        assert "echo" in server.loaded_tools
    

`
}

func (g *Generator) getFastMCPPythonTestDiscovery(_ string, _ map[string]interface{}) string {
	return `"""Tests for tool discovery and loading mechanism."""

import sys
from pathlib import Path
import pytest
import tempfile
import os

# Add src to Python path
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from core.server import DynamicMCPServer


class TestToolDiscovery:
    """Test the tool discovery mechanism."""
    
    def test_discover_tools_in_directory(self):
        """Test discovering tools in a directory."""
        with tempfile.TemporaryDirectory() as temp_dir:
            tools_dir = Path(temp_dir) / "tools"
            tools_dir.mkdir()
            
            # Create a test tool file
            tool_file = tools_dir / "test_tool.py"
            tool_content = '''
from core.server import mcp

@mcp.tool()
def test_tool(message: str) -> str:
    return f"Test: {message}"
'''
            tool_file.write_text(tool_content)
            
            # Test discovery
            server = DynamicMCPServer(name="Test", tools_dir=str(tools_dir))
            
            # Load tools - this should work without raising SystemExit
            try:
                server.load_tools()
                # If we get here, it means loading succeeded
                assert True
            except SystemExit:
                pytest.fail("Tool loading failed")
    
    def test_invalid_tool_fails_fast(self):
        """Test that invalid tools cause the server to exit."""
        with tempfile.TemporaryDirectory() as temp_dir:
            tools_dir = Path(temp_dir) / "tools"
            tools_dir.mkdir()
            
            # Create an invalid tool file (syntax error)
            tool_file = tools_dir / "invalid_tool.py"
            tool_content = '''
def invalid_tool(message: str) -> str:
    return f"Invalid: {message}"
    # This has a syntax error
    return
'''
            tool_file.write_text(tool_content)
            
            server = DynamicMCPServer(name="Test", tools_dir=str(tools_dir))
            
            # This should cause SystemExit due to fail-fast behavior
            with pytest.raises(SystemExit):
                server.load_tools()
    
    def test_tool_without_matching_function(self):
        """Test tool file without matching function name."""
        with tempfile.TemporaryDirectory() as temp_dir:
            tools_dir = Path(temp_dir) / "tools"
            tools_dir.mkdir()
            
            # Create a tool file without matching function name
            tool_file = tools_dir / "mismatch.py"
            tool_content = '''
def wrong_name(message: str) -> str:
    return f"Wrong: {message}"
'''
            tool_file.write_text(tool_content)
            
            server = DynamicMCPServer(name="Test", tools_dir=str(tools_dir))
            
            # This should cause SystemExit due to fail-fast behavior
            with pytest.raises(SystemExit):
                server.load_tools()
    
    def test_empty_tools_directory(self):
        """Test behavior with empty tools directory."""
        with tempfile.TemporaryDirectory() as temp_dir:
            tools_dir = Path(temp_dir) / "tools"
            tools_dir.mkdir()
            
            server = DynamicMCPServer(name="Test", tools_dir=str(tools_dir))
            
            # Should not raise exception
            server.load_tools()
            assert len(server.loaded_tools) == 0
    

`
}
