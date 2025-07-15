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

		// New modular structure
		"src/__init__.py": "",
		"src/main.py":     g.getFastMCPPythonMain(templateType, data),

		// Tools directory
		"src/tools/__init__.py":   g.getFastMCPPythonToolsInit(templateType, data),
		"src/tools/echo.py":       g.getFastMCPPythonEchoTool(templateType, data),
		"src/tools/calculator.py": g.getFastMCPPythonCalculatorTool(templateType, data),

		// Resources directory
		"src/resources/__init__.py": g.getFastMCPPythonResourcesInit(templateType, data),

		// Core directory (generated framework code)
		"src/core/__init__.py": g.getFastMCPPythonCoreInit(templateType, data),
		"src/core/server.py":   g.getFastMCPPythonCoreServer(templateType, data),
		"src/core/registry.py": g.getFastMCPPythonCoreRegistry(templateType, data),

		// Configuration files
		"config/server.yaml": g.getFastMCPPythonServerConfig(templateType, data),
		"config/tools.yaml":  g.getFastMCPPythonToolsConfig(templateType, data),

		// Tests
		"tests/__init__.py":    "",
		"tests/test_tools.py":  g.getFastMCPPythonTestTools(templateType, data),
		"tests/test_server.py": g.getFastMCPPythonTestServer(templateType, data),
	}

	// Add template-specific files
	switch templateType {
	case "http":
		files["src/tools/http_client.py"] = g.getFastMCPPythonHTTPTool(templateType, data)
	case "data":
		files["src/tools/data_processor.py"] = g.getFastMCPPythonDataTool(templateType, data)
	case "workflow":
		files["src/tools/workflow_executor.py"] = g.getFastMCPPythonWorkflowTool(templateType, data)
	case "multi-tool":
		files["src/tools/http_client.py"] = g.getFastMCPPythonHTTPTool(templateType, data)
		files["src/tools/data_processor.py"] = g.getFastMCPPythonDataTool(templateType, data)
		files["src/tools/workflow_executor.py"] = g.getFastMCPPythonWorkflowTool(templateType, data)
	}

	return files
}

// getFastMCPPythonPyprojectToml generates the pyproject.toml template
func (g *Generator) getFastMCPPythonPyprojectToml(templateType string, data map[string]interface{}) string {
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
    "mcp>=1.0.0",
    "fastmcp>=0.1.0",
    "pydantic>=2.0.0",
    "pyyaml>=6.0.0",{{if eq .Template "database"}}
    "asyncpg>=0.29.0",
    "sqlalchemy>=2.0.0",{{end}}{{if eq .Template "filesystem"}}
    "watchdog>=3.0.0",
    "aiofiles>=23.0.0",{{end}}{{if eq .Template "api-client"}}
    "httpx>=0.25.0",
    "aiohttp>=3.8.0",{{end}}{{if eq .Template "multi-tool"}}
    "asyncpg>=0.29.0",
    "sqlalchemy>=2.0.0",
    "watchdog>=3.0.0",
    "aiofiles>=23.0.0",
    "httpx>=0.25.0",
    "aiohttp>=3.8.0",{{end}}
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

// getFastMCPPythonReadme generates the README.md template
func (g *Generator) getFastMCPPythonReadme(templateType string, data map[string]interface{}) string {
	return `# {{.ProjectName}}

{{.ProjectName}} is a Model Context Protocol (MCP) server built with FastMCP using a modular architecture.

## Overview

This MCP server provides {{if eq .Template "basic"}}basic tools and functionality{{else if eq .Template "database"}}database integration capabilities{{else if eq .Template "filesystem"}}filesystem access and management{{else if eq .Template "api-client"}}API client integration{{else if eq .Template "multi-tool"}}comprehensive multi-tool functionality{{else}}custom MCP tools{{end}} using a clean, modular architecture.

## Project Structure

` + "```" + `
src/
├── tools/              # Business logic implementations
│   ├── echo.py         # Echo tool
│   ├── calculator.py   # Calculator tool
│   └── ...
├── resources/          # Resource handlers
├── core/               # Generated framework code
│   ├── server.py       # MCP server setup
│   └── registry.py     # Tool registration
└── main.py             # Entry point
config/
├── server.yaml         # Server configuration
└── tools.yaml          # Tool configuration
` + "```" + `

## Installation

### Local Development

1. **Install uv** (if not already installed):
   ` + "```bash" + `
   curl -LsSf https://astral.sh/uv/install.sh | sh
   # or: brew install uv
   ` + "```" + `

2. **Install dependencies and sync environment**:
   ` + "```bash" + `
   uv sync
   ` + "```" + `

3. **Run the server**:
   ` + "```bash" + `
   uv run python -m src.main
   ` + "```" + `

### Docker Deployment

1. **Build the Docker image**:
   ` + "```bash" + `
   kmcp build --docker
   ` + "```" + `

2. **Run the container**:
   ` + "```bash" + `
   docker run -i {{.ProjectNameKebab}}:latest
   ` + "```" + `

## Usage

### Integration with MCP Clients

Add this server to your MCP client configuration:

` + "```json" + `
{
  "mcpServers": {
    "{{.ProjectNameKebab}}": {
      "command": "python",
      "args": ["-m", "src.main"],
      "cwd": "/path/to/project"
    }
  }
}
` + "```" + `

### Configuration

- **Server Configuration**: Edit ` + "`config/server.yaml`" + ` to modify server behavior
- **Tool Configuration**: Edit ` + "`config/tools.yaml`" + ` to configure individual tools
- **Environment Variables**: Copy ` + "`.env.example`" + ` to ` + "`.env.local`" + ` for local secrets

### Adding New Tools

1. Create a new tool file in ` + "`src/tools/`" + `
2. Implement your tool class
3. Add it to the registry in ` + "`src/core/registry.py`" + `
4. Configure it in ` + "`config/tools.yaml`" + `

## Development

### Running Tests

` + "```bash" + `
uv run pytest
` + "```" + `

### Code Formatting

` + "```bash" + `
uv run black .
uv run ruff check .
` + "```" + `

### Type Checking

` + "```bash" + `
uv run mypy .
` + "```" + `

## License

This project is licensed under the MIT License.
`
}

// getFastMCPPythonDockerfile generates the Dockerfile template
func (g *Generator) getFastMCPPythonDockerfile(templateType string, data map[string]interface{}) string {
	return `# Multi-stage build for {{.ProjectName}} MCP server using uv
FROM python:3.11-slim as builder

# Install uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# Set working directory
WORKDIR /app

# Copy dependency files first for layer caching
COPY pyproject.toml .python-version ./
COPY uv.lock ./
COPY README.md ./

# Install dependencies with uv (much faster than pip!)
RUN uv sync --frozen --no-dev --no-cache

# Copy source code
COPY src/ ./src/
COPY config/ ./config/

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
COPY --from=builder /app/config /app/config
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
CMD ["python", "-m", "src.main"]`
}

// getFastMCPPythonGitignore generates the .gitignore template
func (g *Generator) getFastMCPPythonGitignore(templateType string, data map[string]interface{}) string {
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
config/local.yaml
.mcpbuilder.yaml`
}

// getFastMCPPythonRequirements generates requirements.txt for pip compatibility
// getFastMCPPythonPythonVersion generates the .python-version file for uv
func (g *Generator) getFastMCPPythonPythonVersion(templateType string, data map[string]interface{}) string {
	return `3.11`
}

// getFastMCPPythonEnvExample generates .env.example file
func (g *Generator) getFastMCPPythonEnvExample(templateType string, data map[string]interface{}) string {
	return `# {{.ProjectName}} Environment Variables
# Copy this file to .env.local and fill in actual values

# API Keys and secrets
# API_KEY=your-api-key-here
# DATABASE_URL=postgresql://user:password@localhost:5432/database

# Server configuration
# MCP_SERVER_HOST=127.0.0.1
# MCP_SERVER_PORT=8080
# MCP_LOG_LEVEL=INFO

# Tool-specific configuration
# CALCULATOR_PRECISION=2
# ECHO_PREFIX=""
`
}

// getFastMCPPythonMain generates the main entry point
func (g *Generator) getFastMCPPythonMain(templateType string, data map[string]interface{}) string {
	return `"""Main entry point for {{.ProjectName}} MCP server.

This is the minimal entry point that configures and starts the MCP server.
All business logic is separated into tools/ and resources/ directories.
"""

import sys
from pathlib import Path

# Add src to Python path
sys.path.insert(0, str(Path(__file__).parent))

from core.server import create_server


def main() -> None:
    """Main entry point for the MCP server."""
    try:
        server = create_server()
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

// getFastMCPPythonToolsInit generates the tools __init__.py
func (g *Generator) getFastMCPPythonToolsInit(templateType string, data map[string]interface{}) string {
	return `"""Tools package for {{.ProjectName}} MCP server.

This package contains the business logic implementations for MCP tools.
Each tool is implemented as a separate module for maintainability.
"""

# Import all tools for easy access
from .echo import EchoTool
from .calculator import CalculatorTool

# Export available tools
__all__ = ["EchoTool", "CalculatorTool"]
`
}

// getFastMCPPythonEchoTool generates the echo tool implementation
func (g *Generator) getFastMCPPythonEchoTool(templateType string, data map[string]interface{}) string {
	return `"""Echo tool implementation for {{.ProjectName}} MCP server."""

from datetime import datetime
from typing import Any, Dict
from pydantic import BaseModel, Field


class EchoRequest(BaseModel):
    """Request model for echo operations."""
    message: str = Field(..., description="Message to echo back")


class EchoTool:
    """Tool for echoing messages back to the client."""
    
    def __init__(self, config: Dict[str, Any] = None):
        """Initialize the echo tool with configuration."""
        self.config = config or {}
        self.enabled = self.config.get("enabled", True)
        self.prefix = self.config.get("prefix", "")
    
    def echo(self, request: EchoRequest) -> Dict[str, Any]:
        """Echo a message back to the client.
        
        This is a simple tool that returns the input message along with
        a timestamp, useful for testing connectivity and basic functionality.
        """
        if not self.enabled:
            return {"error": "Echo tool is disabled"}
        
        message = request.message
        if self.prefix:
            message = f"{self.prefix}{message}"
        
        return {
            "message": message,
            "timestamp": datetime.now().isoformat(),
            "length": len(message),
            "server": "{{.ProjectName}}"
        }
`
}

// getFastMCPPythonCalculatorTool generates the calculator tool implementation
func (g *Generator) getFastMCPPythonCalculatorTool(templateType string, data map[string]interface{}) string {
	return `"""Calculator tool implementation for {{.ProjectName}} MCP server."""

from typing import Any, Dict
from pydantic import BaseModel, Field


class CalculationRequest(BaseModel):
    """Request model for calculation operations."""
    operation: str = Field(..., description="The operation to perform: add, subtract, multiply, divide")
    a: float = Field(..., description="First number")
    b: float = Field(..., description="Second number")


class CalculatorTool:
    """Tool for performing basic arithmetic calculations."""
    
    def __init__(self, config: Dict[str, Any] = None):
        """Initialize the calculator tool with configuration."""
        self.config = config or {}
        self.enabled = self.config.get("enabled", True)
        self.operations = self.config.get("operations", ["add", "subtract", "multiply", "divide"])
        self.precision = self.config.get("precision", 2)
    
    def calculate(self, request: CalculationRequest) -> Dict[str, Any]:
        """Perform basic arithmetic calculations.
        
        This tool can perform addition, subtraction, multiplication, and division
        operations on two numbers.
        """
        if not self.enabled:
            return {"error": "Calculator tool is disabled"}
        
        if request.operation not in self.operations:
            return {
                "error": f"Operation '{request.operation}' not supported",
                "supported_operations": self.operations
            }
        
        try:
            result = 0.0
            
            if request.operation == "add":
                result = request.a + request.b
            elif request.operation == "subtract":
                result = request.a - request.b
            elif request.operation == "multiply":
                result = request.a * request.b
            elif request.operation == "divide":
                if request.b == 0:
                    return {
                        "error": "Division by zero is not allowed",
                        "operation": request.operation,
                        "inputs": {"a": request.a, "b": request.b}
                    }
                result = request.a / request.b
            
            # Apply precision if configured
            if self.precision is not None:
                result = round(result, self.precision)
            
            return {
                "result": result,
                "operation": request.operation,
                "inputs": {"a": request.a, "b": request.b}
            }
        except Exception as e:
            return {
                "error": f"Calculation error: {str(e)}",
                "operation": request.operation,
                "inputs": {"a": request.a, "b": request.b}
            }
`
}

// getFastMCPPythonResourcesInit generates the resources __init__.py
func (g *Generator) getFastMCPPythonResourcesInit(templateType string, data map[string]interface{}) string {
	return `"""Resources package for {{.ProjectName}} MCP server.

This package contains the resource handler implementations for MCP resources.
Resources represent data or content that can be accessed by the AI model.
"""

# Future: Add resource implementations here
# from .file_resource import FileResource
# from .web_resource import WebResource

# Export available resources
__all__ = []
`
}

// getFastMCPPythonCoreInit generates the core __init__.py
func (g *Generator) getFastMCPPythonCoreInit(templateType string, data map[string]interface{}) string {
	return `"""Core framework package for {{.ProjectName}} MCP server.

This package contains generated framework code that handles MCP protocol
communication and tool registration. Do not edit files in this package
manually - they are generated by the KMCP CLI.
"""

from .server import create_server
from .registry import ToolRegistry

__all__ = ["create_server", "ToolRegistry"]
`
}

// getFastMCPPythonCoreServer generates the core server implementation
func (g *Generator) getFastMCPPythonCoreServer(templateType string, data map[string]interface{}) string {
	return `"""Core MCP server implementation for {{.ProjectName}}.

This file is generated by the KMCP CLI. Do not edit manually.
"""

import os
import sys
from pathlib import Path
from typing import Dict, Any

import yaml
from fastmcp import FastMCP

from .registry import ToolRegistry


def load_config(config_path: str) -> Dict[str, Any]:
    """Load configuration from YAML file."""
    try:
        with open(config_path, 'r') as f:
            return yaml.safe_load(f) or {}
    except FileNotFoundError:
        return {}
    except Exception as e:
        print(f"Error loading config from {config_path}: {e}", file=sys.stderr)
        return {}


def create_server() -> FastMCP:
    """Create and configure the FastMCP server."""
    # Load configuration
    config_dir = Path(__file__).parent.parent.parent / "config"
    server_config = load_config(config_dir / "server.yaml")
    tools_config = load_config(config_dir / "tools.yaml")
    
    # Create FastMCP server
    server_name = server_config.get("name", "{{.ProjectName}} Server")
    mcp = FastMCP(server_name)
    
    # Initialize tool registry
    registry = ToolRegistry(tools_config)
    
    # Register tools with the server
    registry.register_tools(mcp)
    
    return mcp
`
}

// getFastMCPPythonCoreRegistry generates the tool registry
func (g *Generator) getFastMCPPythonCoreRegistry(templateType string, data map[string]interface{}) string {
	return `"""Tool registry for {{.ProjectName}} MCP server.

This file is generated by the KMCP CLI. Do not edit manually.
"""

from typing import Dict, Any

from tools.echo import EchoTool
from tools.calculator import CalculatorTool


class ToolRegistry:
    """Registry for managing MCP tools."""
    
    def __init__(self, config: Dict[str, Any]):
        """Initialize the tool registry with configuration."""
        self.config = config
        self.tools = {}
        self._initialize_tools()
    
    def _initialize_tools(self):
        """Initialize all available tools."""
        tools_config = self.config.get("tools", {})
        
        # Initialize echo tool
        echo_config = tools_config.get("echo", {})
        if echo_config.get("enabled", True):
            self.tools["echo"] = EchoTool(echo_config)
        
        # Initialize calculator tool
        calc_config = tools_config.get("calculator", {})
        if calc_config.get("enabled", True):
            self.tools["calculator"] = CalculatorTool(calc_config)
    
    def register_tools(self, mcp):
        """Register all tools with the FastMCP server."""
        # Register echo tool
        if "echo" in self.tools:
            mcp.tool()(self.tools["echo"].echo)
        
        # Register calculator tool
        if "calculator" in self.tools:
            mcp.tool()(self.tools["calculator"].calculate)
`
}

// getFastMCPPythonServerConfig generates server configuration
func (g *Generator) getFastMCPPythonServerConfig(templateType string, data map[string]interface{}) string {
	return `# {{.ProjectName}} MCP Server Configuration
# This file configures the overall server behavior

name: "{{.ProjectName}} Server"
description: "{{.ProjectName}} MCP server built with FastMCP"
version: "0.1.0"

# Transport configuration
transport:
  type: "stdio"  # stdio, http, or websocket
  host: "127.0.0.1"
  port: 8080

# Logging configuration
logging:
  level: "INFO"  # DEBUG, INFO, WARNING, ERROR, CRITICAL
  format: "%(asctime)s - %(name)s - %(levelname)s - %(message)s"

# Security configuration
security:
  enable_sanitization: true
  max_response_size: "10MB"
  timeout: "30s"

# Performance configuration
performance:
  max_concurrent_requests: 10
  request_timeout: "30s"
`
}

// getFastMCPPythonToolsConfig generates tools configuration
func (g *Generator) getFastMCPPythonToolsConfig(templateType string, data map[string]interface{}) string {
	return `# {{.ProjectName}} Tools Configuration
# This file configures individual tool behavior

tools:
  echo:
    enabled: true
    prefix: ""
    description: "Echo messages back to the client"
    
  calculator:
    enabled: true
    precision: 2
    operations:
      - add
      - subtract
      - multiply
      - divide
    description: "Perform basic arithmetic calculations"

# Resource configuration
resources:
  # Future: Add resource configurations here
  
# Environment-specific overrides
environments:
  development:
    logging:
      level: "DEBUG"
    tools:
      echo:
        prefix: "[DEV] "
  
  production:
    logging:
      level: "WARNING"
    performance:
      max_concurrent_requests: 50
`
}

// getFastMCPPythonTestTools generates tool tests
func (g *Generator) getFastMCPPythonTestTools(templateType string, data map[string]interface{}) string {
	return `"""Tests for {{.ProjectName}} MCP server tools."""

import sys
from pathlib import Path
import pytest

# Add src to Python path
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from tools.echo import EchoTool, EchoRequest
from tools.calculator import CalculatorTool, CalculationRequest


class TestEchoTool:
    """Test cases for the echo tool."""
    
    def test_echo_basic(self):
        """Test basic echo functionality."""
        tool = EchoTool()
        request = EchoRequest(message="Hello, World!")
        result = tool.echo(request)
        
        assert result["message"] == "Hello, World!"
        assert result["length"] == 13
        assert result["server"] == "{{.ProjectName}}"
        assert "timestamp" in result
    
    def test_echo_with_prefix(self):
        """Test echo with prefix configuration."""
        tool = EchoTool({"prefix": "[TEST] "})
        request = EchoRequest(message="Hello")
        result = tool.echo(request)
        
        assert result["message"] == "[TEST] Hello"
        assert result["length"] == 12
    
    def test_echo_disabled(self):
        """Test echo tool when disabled."""
        tool = EchoTool({"enabled": False})
        request = EchoRequest(message="Hello")
        result = tool.echo(request)
        
        assert "error" in result
        assert "disabled" in result["error"]


class TestCalculatorTool:
    """Test cases for the calculator tool."""
    
    def test_calculator_add(self):
        """Test calculator addition."""
        tool = CalculatorTool()
        request = CalculationRequest(operation="add", a=5.0, b=3.0)
        result = tool.calculate(request)
        
        assert result["result"] == 8.0
        assert result["operation"] == "add"
        assert result["inputs"]["a"] == 5.0
        assert result["inputs"]["b"] == 3.0
    
    def test_calculator_divide_by_zero(self):
        """Test calculator division by zero."""
        tool = CalculatorTool()
        request = CalculationRequest(operation="divide", a=5.0, b=0.0)
        result = tool.calculate(request)
        
        assert "error" in result
        assert "Division by zero" in result["error"]
    
    def test_calculator_invalid_operation(self):
        """Test calculator with invalid operation."""
        tool = CalculatorTool()
        request = CalculationRequest(operation="invalid", a=5.0, b=3.0)
        result = tool.calculate(request)
        
        assert "error" in result
        assert "not supported" in result["error"]
        assert "supported_operations" in result
    
    def test_calculator_precision(self):
        """Test calculator precision configuration."""
        tool = CalculatorTool({"precision": 1})
        request = CalculationRequest(operation="divide", a=10.0, b=3.0)
        result = tool.calculate(request)
        
        assert result["result"] == 3.3  # rounded to 1 decimal place
    
    def test_calculator_disabled(self):
        """Test calculator tool when disabled."""
        tool = CalculatorTool({"enabled": False})
        request = CalculationRequest(operation="add", a=5.0, b=3.0)
        result = tool.calculate(request)
        
        assert "error" in result
        assert "disabled" in result["error"]
`
}

// getFastMCPPythonTestServer generates server tests
func (g *Generator) getFastMCPPythonTestServer(templateType string, data map[string]interface{}) string {
	return `"""Tests for {{.ProjectName}} MCP server core functionality."""

import sys
from pathlib import Path
import pytest
from unittest.mock import patch, mock_open

# Add src to Python path
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from core.server import create_server, load_config
from core.registry import ToolRegistry


class TestServerConfiguration:
    """Test cases for server configuration."""
    
    def test_load_config_success(self):
        """Test successful configuration loading."""
        config_data = """
        name: "Test Server"
        logging:
          level: "DEBUG"
        """
        with patch("builtins.open", mock_open(read_data=config_data)):
            config = load_config("test.yaml")
            assert config["name"] == "Test Server"
            assert config["logging"]["level"] == "DEBUG"
    
    def test_load_config_file_not_found(self):
        """Test configuration loading with missing file."""
        with patch("builtins.open", side_effect=FileNotFoundError):
            config = load_config("missing.yaml")
            assert config == {}
    
    def test_create_server(self):
        """Test server creation."""
        with patch("core.server.load_config") as mock_load:
            mock_load.return_value = {"name": "Test Server"}
            server = create_server()
            assert server is not None


class TestToolRegistry:
    """Test cases for tool registry."""
    
    def test_registry_initialization(self):
        """Test tool registry initialization."""
        config = {
            "tools": {
                "echo": {"enabled": True},
                "calculator": {"enabled": True}
            }
        }
        registry = ToolRegistry(config)
        
        assert "echo" in registry.tools
        assert "calculator" in registry.tools
    
    def test_registry_disabled_tool(self):
        """Test tool registry with disabled tool."""
        config = {
            "tools": {
                "echo": {"enabled": False},
                "calculator": {"enabled": True}
            }
        }
        registry = ToolRegistry(config)
        
        assert "echo" not in registry.tools
        assert "calculator" in registry.tools
    
    def test_registry_register_tools(self):
        """Test tool registration with FastMCP."""
        config = {
            "tools": {
                "echo": {"enabled": True},
                "calculator": {"enabled": True}
            }
        }
        registry = ToolRegistry(config)
        
        # Mock FastMCP server
        class MockMCP:
            def __init__(self):
                self.registered_tools = []
            
            def tool(self):
                def decorator(func):
                    self.registered_tools.append(func)
                    return func
                return decorator
        
        mock_mcp = MockMCP()
        registry.register_tools(mock_mcp)
        
        assert len(mock_mcp.registered_tools) == 2
`
}

// Placeholder implementations for other template types
func (g *Generator) getFastMCPPythonDatabaseTool(templateType string, data map[string]interface{}) string {
	return `"""Database tool implementation for {{.ProjectName}} MCP server."""

from typing import Any, Dict
from pydantic import BaseModel, Field


class DatabaseQueryRequest(BaseModel):
    """Request model for database operations."""
    query: str = Field(..., description="SQL query to execute")
    params: Dict[str, Any] = Field(default_factory=dict, description="Query parameters")


class DatabaseTool:
    """Tool for database operations."""
    
    def __init__(self, config: Dict[str, Any] = None):
        """Initialize the database tool with configuration."""
        self.config = config or {}
        self.enabled = self.config.get("enabled", True)
        self.max_results = self.config.get("max_results", 100)
    
    def query(self, request: DatabaseQueryRequest) -> Dict[str, Any]:
        """Execute database query."""
        if not self.enabled:
            return {"error": "Database tool is disabled"}
        
        # TODO: Implement database connectivity
        return {
            "message": "Database integration template - implementation coming soon",
            "query": request.query,
            "params": request.params
        }
`
}

func (g *Generator) getFastMCPPythonFilesystemTool(templateType string, data map[string]interface{}) string {
	return `"""Filesystem tool implementation for {{.ProjectName}} MCP server."""

from typing import Any, Dict
from pydantic import BaseModel, Field


class ReadFileRequest(BaseModel):
    """Request model for file reading operations."""
    path: str = Field(..., description="Path to the file to read")
    encoding: str = Field(default="utf-8", description="File encoding")


class FilesystemTool:
    """Tool for filesystem operations."""
    
    def __init__(self, config: Dict[str, Any] = None):
        """Initialize the filesystem tool with configuration."""
        self.config = config or {}
        self.enabled = self.config.get("enabled", True)
        self.max_size = self.config.get("max_size", "1MB")
    
    def read_file(self, request: ReadFileRequest) -> Dict[str, Any]:
        """Read file contents."""
        if not self.enabled:
            return {"error": "Filesystem tool is disabled"}
        
        # TODO: Implement safe file reading
        return {
            "message": "Filesystem integration template - implementation coming soon",
            "path": request.path,
            "encoding": request.encoding
        }
`
}

func (g *Generator) getFastMCPPythonAPIClientTool(templateType string, data map[string]interface{}) string {
	return `"""API client tool implementation for {{.ProjectName}} MCP server."""

from typing import Any, Dict
from pydantic import BaseModel, Field


class HTTPRequest(BaseModel):
    """Request model for HTTP operations."""
    url: str = Field(..., description="URL to make request to")
    method: str = Field(default="GET", description="HTTP method")
    headers: Dict[str, str] = Field(default_factory=dict, description="HTTP headers")
    body: str = Field(default="", description="Request body")


class APIClientTool:
    """Tool for API client operations."""
    
    def __init__(self, config: Dict[str, Any] = None):
        """Initialize the API client tool with configuration."""
        self.config = config or {}
        self.enabled = self.config.get("enabled", True)
        self.timeout = self.config.get("timeout", "30s")
    
    def http_request(self, request: HTTPRequest) -> Dict[str, Any]:
        """Make HTTP request."""
        if not self.enabled:
            return {"error": "API client tool is disabled"}
        
        # TODO: Implement HTTP client
        return {
            "message": "API client integration template - implementation coming soon",
            "url": request.url,
            "method": request.method,
            "headers": request.headers
        }
`
}

// New tool functions for the simplified tool types
func (g *Generator) getFastMCPPythonHTTPTool(templateType string, data map[string]interface{}) string {
	return `"""HTTP client tool implementation for {{.ProjectName}} MCP server."""

from typing import Any, Dict
from pydantic import BaseModel, Field
import httpx


class HTTPRequestRequest(BaseModel):
    """Request model for HTTP operations."""
    method: str = Field(..., description="HTTP method (GET, POST, etc.)")
    endpoint: str = Field(..., description="API endpoint")
    data: Dict[str, Any] = Field(default_factory=dict, description="Request data")


class HTTPTool:
    """Tool for HTTP client operations."""
    
    def __init__(self, config: Dict[str, Any] = None):
        """Initialize the HTTP tool with configuration."""
        self.config = config or {}
        self.base_url = self.config.get("base_url", "")
        self.timeout = self.config.get("timeout", 30)
    
    async def make_request(self, request: HTTPRequestRequest) -> Dict[str, Any]:
        """Make an HTTP request."""
        url = f"{self.base_url.rstrip('/')}/{request.endpoint.lstrip('/')}" if self.base_url else request.endpoint
        
        try:
            async with httpx.AsyncClient(timeout=self.timeout) as client:
                response = await client.request(
                    request.method,
                    url,
                    json=request.data if request.data else None
                )
                
                try:
                    data = response.json()
                except:
                    data = response.text
                
                return {
                    "status_code": response.status_code,
                    "data": data,
                    "success": response.is_success
                }
        except Exception as e:
            return {
                "error": str(e),
                "success": False
            }
`
}

func (g *Generator) getFastMCPPythonDataTool(templateType string, data map[string]interface{}) string {
	return `"""Data processing tool implementation for {{.ProjectName}} MCP server."""

from typing import Any, Dict, List, Union
from pydantic import BaseModel, Field


class DataProcessRequest(BaseModel):
    """Request model for data processing operations."""
    data: Union[Dict, List, str] = Field(..., description="Input data to process")
    operation: str = Field(default="process", description="Operation to perform")


class DataTool:
    """Tool for data processing operations."""
    
    def __init__(self, config: Dict[str, Any] = None):
        """Initialize the data tool with configuration."""
        self.config = config or {}
    
    async def process_data(self, request: DataProcessRequest) -> Dict[str, Any]:
        """Process input data."""
        # TODO: Implement your data processing logic here
        return {
            "tool": "data_processor",
            "input_type": type(request.data).__name__,
            "operation": request.operation,
            "result": "Data processed successfully"
        }
    
    async def validate_data(self, request: DataProcessRequest) -> Dict[str, Any]:
        """Validate input data."""
        # TODO: Implement your validation logic here
        return {
            "valid": True,
            "errors": [],
            "data_type": type(request.data).__name__
        }
`
}

func (g *Generator) getFastMCPPythonWorkflowTool(templateType string, data map[string]interface{}) string {
	return `"""Workflow execution tool implementation for {{.ProjectName}} MCP server."""

from typing import Any, Dict, List
from pydantic import BaseModel, Field


class WorkflowRequest(BaseModel):
    """Request model for workflow operations."""
    steps: List[Dict[str, Any]] = Field(..., description="Workflow steps to execute")
    context: Dict[str, Any] = Field(default_factory=dict, description="Initial context")


class WorkflowTool:
    """Tool for workflow execution operations."""
    
    def __init__(self, config: Dict[str, Any] = None):
        """Initialize the workflow tool with configuration."""
        self.config = config or {}
        self.max_steps = self.config.get("max_steps", 10)
    
    async def execute_workflow(self, request: WorkflowRequest) -> Dict[str, Any]:
        """Execute a workflow with multiple steps."""
        if len(request.steps) > self.max_steps:
            return {
                "error": f"Too many steps (max {self.max_steps})"
            }
        
        results = []
        context = request.context.copy()
        
        for i, step in enumerate(request.steps):
            # TODO: Implement your step execution logic here
            step_result = await self.execute_step(step, context)
            results.append(step_result)
            
            # Update context with step results
            if step_result.get("context"):
                context.update(step_result["context"])
        
        return {
            "tool": "workflow_executor",
            "steps_executed": len(results),
            "results": results,
            "context": context
        }
    
    async def execute_step(self, step: Dict[str, Any], context: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a single workflow step."""
        # TODO: Implement your step execution logic here
        return {
            "step_type": step.get("type", "unknown"),
            "status": "success",
            "context": {}
        }
`
}
