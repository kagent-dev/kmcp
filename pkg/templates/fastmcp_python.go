package templates

// getFastMCPPythonFiles returns the file templates for FastMCP Python projects
func (g *Generator) getFastMCPPythonFiles(templateType string, data map[string]interface{}) map[string]string {
	packageName := data["ProjectNameSnake"].(string)

	files := map[string]string{
		"pyproject.toml":                    g.getFastMCPPythonPyprojectToml(templateType, data),
		"README.md":                         g.getFastMCPPythonReadme(templateType, data),
		"Dockerfile":                        g.getFastMCPPythonDockerfile(templateType, data),
		".gitignore":                        g.getFastMCPPythonGitignore(templateType, data),
		packageName + "/__init__.py":        g.getFastMCPPythonInit(templateType, data),
		packageName + "/server.py":          g.getFastMCPPythonServer(templateType, data),
		"tests/__init__.py":                 "",
		"tests/test_" + packageName + ".py": g.getFastMCPPythonTests(templateType, data),
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
requires-python = ">=3.9"
dependencies = [
    "mcp>=1.0.0",
    "fastmcp>=0.1.0",{{if eq .Template "database"}}
    "asyncpg>=0.29.0",
    "sqlalchemy>=2.0.0",{{end}}{{if eq .Template "filesystem"}}
    "watchdog>=3.0.0",{{end}}{{if eq .Template "api-client"}}
    "httpx>=0.25.0",
    "aiohttp>=3.8.0",{{end}}
]

[project.optional-dependencies]
dev = [
    "pytest>=7.0.0",
    "pytest-asyncio>=0.21.0",
    "black>=22.0.0",
    "mypy>=1.0.0",
    "ruff>=0.1.0",
]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[project.scripts]
{{.ProjectNameKebab}}-server = "{{.ProjectNameSnake}}.server:main"

[tool.black]
line-length = 88
target-version = ['py39']

[tool.ruff]
line-length = 88
target-version = "py39"
select = ["E", "F", "I", "N", "W", "UP"]

[tool.mypy]
python_version = "3.9"
warn_return_any = true
warn_unused_configs = true
disallow_untyped_defs = true`
}

// getFastMCPPythonReadme generates the README.md template
func (g *Generator) getFastMCPPythonReadme(templateType string, data map[string]interface{}) string {
	return `# {{.ProjectName}}

{{.ProjectName}} is a Model Context Protocol (MCP) server built with FastMCP.

## Overview

This MCP server provides {{if eq .Template "basic"}}basic tools and functionality{{else if eq .Template "database"}}database integration capabilities{{else if eq .Template "filesystem"}}filesystem access and management{{else if eq .Template "api-client"}}API client integration{{else if eq .Template "multi-tool"}}comprehensive multi-tool functionality{{else}}custom MCP tools{{end}}.

## Installation

### Local Development

1. **Install dependencies**:
   ` + "```bash" + `
   pip install -e .
   ` + "```" + `

2. **Run the server**:
   ` + "```bash" + `
   python -m {{.ProjectNameSnake}}.server
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
      "args": ["-m", "{{.ProjectNameSnake}}.server"],
      "cwd": "/path/to/project"
    }
  }
}
` + "```" + `

## Development

### Running Tests

` + "```bash" + `
pip install -e ".[dev]"
pytest
` + "```" + `

### Code Formatting

` + "```bash" + `
black .
ruff check .
` + "```" + `

### Type Checking

` + "```bash" + `
mypy .
` + "```" + `

## License

This project is licensed under the MIT License.
`
}

// getFastMCPPythonDockerfile generates the Dockerfile template
func (g *Generator) getFastMCPPythonDockerfile(templateType string, data map[string]interface{}) string {
	return `# Multi-stage build for {{.ProjectName}} MCP server
FROM python:3.11-slim as builder

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements first for layer caching
COPY pyproject.toml README.md ./

# Install build tools
RUN pip install --no-cache-dir --upgrade pip setuptools wheel

# Copy source code
COPY {{.ProjectNameSnake}}/ ./{{.ProjectNameSnake}}/

# Install the application
RUN pip install --no-cache-dir -e .

# Production stage
FROM python:3.11-slim

# Create non-root user
RUN groupadd -r mcpuser && useradd -r -g mcpuser mcpuser

# Set working directory
WORKDIR /app

# Install runtime dependencies only
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy built application from builder stage
COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

# Copy application code
COPY . .

# Install the application
RUN pip install --no-cache-dir --no-deps -e .

# Change ownership to non-root user
RUN chown -R mcpuser:mcpuser /app

# Switch to non-root user
USER mcpuser

# Expose port (if needed for HTTP transport)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "import sys; sys.exit(0)"

# Set environment variables
ENV PYTHONPATH=/app
ENV PYTHONUNBUFFERED=1

# Default command
CMD ["python", "-m", "{{.ProjectNameSnake}}.server"]`
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
.pyre/`
}

// getFastMCPPythonInit generates the __init__.py template
func (g *Generator) getFastMCPPythonInit(templateType string, data map[string]interface{}) string {
	return `"""{{.ProjectName}} MCP Server.

{{if eq .Template "basic"}}A basic MCP server implementation using FastMCP framework.{{else if eq .Template "database"}}A database-integrated MCP server using FastMCP framework.{{else if eq .Template "filesystem"}}A filesystem-focused MCP server using FastMCP framework.{{else if eq .Template "api-client"}}An API client MCP server using FastMCP framework.{{else if eq .Template "multi-tool"}}A comprehensive multi-tool MCP server using FastMCP framework.{{else}}A custom MCP server implementation using FastMCP framework.{{end}}
"""

__version__ = "0.1.0"`
}

// getFastMCPPythonServer generates the server.py template
func (g *Generator) getFastMCPPythonServer(templateType string, data map[string]interface{}) string {
	switch templateType {
	case "basic":
		return g.getFastMCPPythonBasicServer(data)
	case "database":
		return g.getFastMCPPythonDatabaseServer(data)
	case "filesystem":
		return g.getFastMCPPythonFilesystemServer(data)
	case "api-client":
		return g.getFastMCPPythonAPIClientServer(data)
	case "multi-tool":
		return g.getFastMCPPythonMultiToolServer(data)
	default:
		return g.getFastMCPPythonBasicServer(data)
	}
}

// getFastMCPPythonBasicServer generates a basic server template
func (g *Generator) getFastMCPPythonBasicServer(data map[string]interface{}) string {
	return `"""Main FastMCP server implementation with basic tools."""

import sys
from datetime import datetime
from typing import Any, Dict

from fastmcp import FastMCP
from pydantic import BaseModel, Field


class EchoRequest(BaseModel):
    """Request model for echo operations."""
    message: str = Field(..., description="Message to echo back")


class CalculationRequest(BaseModel):
    """Request model for calculation operations."""
    operation: str = Field(..., description="The operation to perform: add, subtract, multiply, divide")
    a: float = Field(..., description="First number")
    b: float = Field(..., description="Second number")


# Initialize FastMCP server
mcp = FastMCP("{{.ProjectName}} Server")


@mcp.tool()
def echo(request: EchoRequest) -> Dict[str, Any]:
    """Echo a message back to the client.
    
    This is a simple tool that returns the input message along with
    a timestamp, useful for testing connectivity and basic functionality.
    """
    return {
        "message": request.message,
        "timestamp": datetime.now().isoformat(),
        "length": len(request.message),
        "server": "{{.ProjectName}}"
    }


@mcp.tool()
def calculator(request: CalculationRequest) -> Dict[str, Any]:
    """Perform basic arithmetic calculations.
    
    This tool can perform addition, subtraction, multiplication, and division
    operations on two numbers.
    """
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
        else:
            return {
                "error": f"Unknown operation: {request.operation}",
                "supported_operations": ["add", "subtract", "multiply", "divide"]
            }
        
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


def main() -> None:
    """Main entry point for the MCP server."""
    try:
        # Run the FastMCP server
        mcp.run()
    except KeyboardInterrupt:
        print("\nShutting down server...")
    except Exception as e:
        print(f"Server error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()`
}

// getFastMCPPythonTests generates the test template
func (g *Generator) getFastMCPPythonTests(templateType string, data map[string]interface{}) string {
	return `"""Tests for {{.ProjectName}} MCP server."""

import pytest
from {{.ProjectNameSnake}}.server import echo, calculator, EchoRequest, CalculationRequest


def test_echo():
    """Test the echo tool."""
    request = EchoRequest(message="Hello, World!")
    result = echo(request)
    
    assert result["message"] == "Hello, World!"
    assert result["length"] == 13
    assert result["server"] == "{{.ProjectName}}"
    assert "timestamp" in result


def test_calculator_add():
    """Test calculator addition."""
    request = CalculationRequest(operation="add", a=5.0, b=3.0)
    result = calculator(request)
    
    assert result["result"] == 8.0
    assert result["operation"] == "add"
    assert result["inputs"]["a"] == 5.0
    assert result["inputs"]["b"] == 3.0


def test_calculator_divide_by_zero():
    """Test calculator division by zero."""
    request = CalculationRequest(operation="divide", a=5.0, b=0.0)
    result = calculator(request)
    
    assert "error" in result
    assert "Division by zero" in result["error"]


def test_calculator_invalid_operation():
    """Test calculator with invalid operation."""
    request = CalculationRequest(operation="invalid", a=5.0, b=3.0)
    result = calculator(request)
    
    assert "error" in result
    assert "Unknown operation" in result["error"]
    assert "supported_operations" in result`
}

// Placeholder implementations for other templates - will expand these incrementally
func (g *Generator) getFastMCPPythonDatabaseServer(data map[string]interface{}) string {
	return `"""Database-integrated FastMCP server implementation."""

import sys
from fastmcp import FastMCP

# Initialize FastMCP server
mcp = FastMCP("{{.ProjectName}} Database Server")

# TODO: Implement database tools
@mcp.tool()
def placeholder() -> str:
    """Placeholder tool - database integration coming soon."""
    return "Database integration template will be implemented in the next iteration"

def main() -> None:
    """Main entry point for the MCP server."""
    try:
        mcp.run()
    except KeyboardInterrupt:
        print("\nShutting down server...")
    except Exception as e:
        print(f"Server error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()`
}

func (g *Generator) getFastMCPPythonFilesystemServer(data map[string]interface{}) string {
	return `"""Filesystem-focused FastMCP server implementation."""

import sys
from fastmcp import FastMCP

# Initialize FastMCP server
mcp = FastMCP("{{.ProjectName}} Filesystem Server")

# TODO: Implement filesystem tools
@mcp.tool()
def placeholder() -> str:
    """Placeholder tool - filesystem integration coming soon."""
    return "Filesystem integration template will be implemented in the next iteration"

def main() -> None:
    """Main entry point for the MCP server."""
    try:
        mcp.run()
    except KeyboardInterrupt:
        print("\nShutting down server...")
    except Exception as e:
        print(f"Server error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()`
}

func (g *Generator) getFastMCPPythonAPIClientServer(data map[string]interface{}) string {
	return `"""API client FastMCP server implementation."""

import sys
from fastmcp import FastMCP

# Initialize FastMCP server
mcp = FastMCP("{{.ProjectName}} API Client Server")

# TODO: Implement API client tools
@mcp.tool()
def placeholder() -> str:
    """Placeholder tool - API client integration coming soon."""
    return "API client integration template will be implemented in the next iteration"

def main() -> None:
    """Main entry point for the MCP server."""
    try:
        mcp.run()
    except KeyboardInterrupt:
        print("\nShutting down server...")
    except Exception as e:
        print(f"Server error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()`
}

func (g *Generator) getFastMCPPythonMultiToolServer(data map[string]interface{}) string {
	return `"""Multi-tool FastMCP server implementation."""

import sys
from fastmcp import FastMCP

# Initialize FastMCP server
mcp = FastMCP("{{.ProjectName}} Multi-Tool Server")

# TODO: Implement comprehensive multi-tool functionality
@mcp.tool()
def placeholder() -> str:
    """Placeholder tool - multi-tool functionality coming soon."""
    return "Multi-tool template will be implemented in the next iteration"

def main() -> None:
    """Main entry point for the MCP server."""
    try:
        mcp.run()
    except KeyboardInterrupt:
        print("\nShutting down server...")
    except Exception as e:
        print(f"Server error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()`
}
