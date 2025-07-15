package templates

// getOfficialPythonFiles returns the file templates for Official Python SDK projects
func (g *Generator) getOfficialPythonFiles(templateType string, data map[string]interface{}) map[string]string {
	files := map[string]string{
		"pyproject.toml":  g.getOfficialPythonPyprojectToml(templateType, data),
		".python-version": g.getOfficialPythonPythonVersion(templateType, data),
		"README.md":       g.getOfficialPythonReadme(templateType, data),
		"Dockerfile":      g.getOfficialPythonDockerfile(templateType, data),
		".gitignore":      g.getOfficialPythonGitignore(templateType, data),
		".env.example":    g.getOfficialPythonEnvExample(templateType, data),

		// Official SDK structure - minimal and focused
		"src/server.py":   g.getOfficialPythonServer(templateType, data),
		"src/tools.py":    g.getOfficialPythonTools(templateType, data),
		"src/__init__.py": "",

		// Main entry point
		"main.py": g.getOfficialPythonMain(templateType, data),

		// Tests
		"tests/__init__.py":    "",
		"tests/test_server.py": g.getOfficialPythonTestServer(templateType, data),
		"tests/test_tools.py":  g.getOfficialPythonTestTools(templateType, data),
	}

	// Add template-specific files
	switch templateType {
	case "http":
		files["src/http_client_tools.py"] = g.getOfficialPythonHTTPClientTools(templateType, data)
	case "data":
		files["src/data_processor_tools.py"] = g.getOfficialPythonDataProcessorTools(templateType, data)
	case "workflow":
		files["src/workflow_executor_tools.py"] = g.getOfficialPythonWorkflowExecutorTools(templateType, data)
	case "multi-tool":
		files["src/http_client_tools.py"] = g.getOfficialPythonHTTPClientTools(templateType, data)
		files["src/data_processor_tools.py"] = g.getOfficialPythonDataProcessorTools(templateType, data)
		files["src/workflow_executor_tools.py"] = g.getOfficialPythonWorkflowExecutorTools(templateType, data)
	}

	return files
}

// getOfficialPythonPyprojectToml generates a minimal pyproject.toml
func (g *Generator) getOfficialPythonPyprojectToml(templateType string, data map[string]interface{}) string {
	return `[project]
name = "{{.ProjectNameKebab}}"
version = "0.1.0"
description = "{{.ProjectName}} MCP server built with Official Python SDK"
authors = [
    {name = "{{.Author}}", email = "{{.Email}}"}
]
readme = "README.md"
requires-python = ">=3.10"
dependencies = [
    "mcp>=1.0.0",
    "pydantic>=2.0.0",
    "click>=8.0.0",{{if eq .Template "database"}}
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
{{.ProjectNameKebab}} = "main:main"

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

// getOfficialPythonReadme generates a focused README for official SDK
func (g *Generator) getOfficialPythonReadme(templateType string, data map[string]interface{}) string {
	return `# {{.ProjectName}}

A Model Context Protocol (MCP) server built with the Official Python SDK.

## Overview

This MCP server provides {{if eq .Template "basic"}}basic tools and functionality{{else if eq .Template "database"}}database integration capabilities{{else if eq .Template "filesystem"}}filesystem access and management{{else if eq .Template "api-client"}}API client integration{{else if eq .Template "multi-tool"}}comprehensive multi-tool functionality{{else}}custom MCP tools{{end}} using the official MCP Python SDK.

## Installation

### Local Development

1. **Install uv** (if not already installed):
   ` + "```bash" + `
   curl -LsSf https://astral.sh/uv/install.sh | sh
   ` + "```" + `

2. **Install dependencies**:
   ` + "```bash" + `
   uv sync
   ` + "```" + `

3. **Run the server**:
   ` + "```bash" + `
   uv run python main.py
   ` + "```" + `

### Docker

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
      "args": ["main.py"],
      "cwd": "/path/to/project"
    }
  }
}
` + "```" + `

### Configuration

Edit ` + "`.env`" + ` to configure environment variables for your server.

### Adding New Tools

1. Define your tool in ` + "`src/tools.py`" + `
2. Register it in ` + "`src/server.py`" + `
3. Follow the MCP specification for tool definitions

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

## Resources

- [MCP Specification](https://spec.modelcontextprotocol.io/)
- [Official Python SDK](https://github.com/modelcontextprotocol/python-sdk)
- [MCP Documentation](https://modelcontextprotocol.io/)

## License

This project is licensed under the MIT License.
`
}

// getOfficialPythonServer generates the main server implementation
func (g *Generator) getOfficialPythonServer(templateType string, data map[string]interface{}) string {
	return `"""{{.ProjectName}} MCP Server using Official Python SDK."""

import asyncio
import logging
from typing import Any, Dict, List, Optional

from mcp.server import Server
from mcp.server.models import InitializeRequest, InitializeResponse, ListToolsRequest, ListToolsResponse, CallToolRequest, CallToolResponse
from mcp.server.stdio import stdio_server
from mcp.types import Tool, TextContent, McpError, ErrorCode

from .tools import get_available_tools, call_tool

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class {{.ProjectNamePascal}}Server:
    """{{.ProjectName}} MCP Server."""
    
    def __init__(self):
        self.server = Server("{{.ProjectName}}")
        self.tools = get_available_tools()
        self._setup_handlers()
    
    def _setup_handlers(self):
        """Set up MCP message handlers."""
        
        @self.server.list_tools()
        async def list_tools() -> ListToolsResponse:
            """List available tools."""
            return ListToolsResponse(tools=self.tools)
        
        @self.server.call_tool()
        async def call_tool_handler(request: CallToolRequest) -> CallToolResponse:
            """Handle tool calls."""
            tool_name = request.name
            arguments = request.arguments or {}
            
            try:
                result = await call_tool(tool_name, arguments)
                return CallToolResponse(
                    content=[TextContent(type="text", text=str(result))]
                )
            except KeyError:
                raise McpError(
                    ErrorCode.INVALID_REQUEST,
                    f"Unknown tool: {tool_name}"
                )
            except Exception as e:
                logger.error(f"Error calling tool {tool_name}: {e}")
                raise McpError(
                    ErrorCode.INTERNAL_ERROR,
                    f"Tool execution failed: {str(e)}"
                )
    
    async def run(self):
        """Run the server."""
        logger.info("Starting {{.ProjectName}} MCP Server")
        logger.info(f"Available tools: {[tool.name for tool in self.tools]}")
        
        async with stdio_server() as (read_stream, write_stream):
            await self.server.run(
                read_stream,
                write_stream,
                InitializeRequest(
                    protocolVersion="2024-11-05",
                    capabilities={},
                    clientInfo={
                        "name": "{{.ProjectName}}",
                        "version": "0.1.0",
                    },
                ),
            )


def create_server() -> {{.ProjectNamePascal}}Server:
    """Create and return a new server instance."""
    return {{.ProjectNamePascal}}Server()
`
}

// getOfficialPythonTools generates the tools implementation
func (g *Generator) getOfficialPythonTools(templateType string, data map[string]interface{}) string {
	return `"""Tool implementations for {{.ProjectName}} MCP Server."""

import asyncio
import platform
import time
from typing import Any, Dict, List
from datetime import datetime

from mcp.types import Tool


def get_available_tools() -> List[Tool]:
    """Get list of available tools."""
    tools = [
        Tool(
            name="echo",
            description="Echo a message back to the client",
            inputSchema={
                "type": "object",
                "properties": {
                    "message": {
                        "type": "string",
                        "description": "Message to echo back"
                    }
                },
                "required": ["message"]
            }
        ),
        Tool(
            name="calculate",
            description="Perform basic arithmetic calculations",
            inputSchema={
                "type": "object",
                "properties": {
                    "operation": {
                        "type": "string",
                        "enum": ["add", "subtract", "multiply", "divide"],
                        "description": "The operation to perform"
                    },
                    "a": {
                        "type": "number",
                        "description": "First number"
                    },
                    "b": {
                        "type": "number",
                        "description": "Second number"
                    }
                },
                "required": ["operation", "a", "b"]
            }
        ),
        Tool(
            name="system_info",
            description="Get basic system information",
            inputSchema={
                "type": "object",
                "properties": {},
                "required": []
            }
        ),
    ]
    return tools


async def call_tool(name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
    """Call a tool by name with the given arguments."""
    tool_functions = {
        "echo": echo_tool,
        "calculate": calculate_tool,
        "system_info": system_info_tool,
    }
    
    if name not in tool_functions:
        raise KeyError(f"Unknown tool: {name}")
    
    return await tool_functions[name](arguments)


async def echo_tool(arguments: Dict[str, Any]) -> Dict[str, Any]:
    """Echo a message back to the client."""
    message = arguments.get("message", "")
    return {
        "message": message,
        "timestamp": datetime.now().isoformat(),
        "length": len(message),
        "server": "{{.ProjectName}}"
    }


async def calculate_tool(arguments: Dict[str, Any]) -> Dict[str, Any]:
    """Perform basic arithmetic calculations."""
    operation = arguments.get("operation")
    a = arguments.get("a")
    b = arguments.get("b")
    
    if operation == "add":
        result = a + b
    elif operation == "subtract":
        result = a - b
    elif operation == "multiply":
        result = a * b
    elif operation == "divide":
        if b == 0:
            raise ValueError("Division by zero is not allowed")
        result = a / b
    else:
        raise ValueError(f"Unknown operation: {operation}")
    
    return {
        "result": round(result, 2),
        "operation": operation,
        "inputs": {"a": a, "b": b}
    }


async def system_info_tool(arguments: Dict[str, Any]) -> Dict[str, Any]:
    """Get basic system information."""
    return {
        "platform": platform.system(),
        "platform_version": platform.version(),
        "python_version": platform.python_version(),
        "architecture": platform.machine(),
        "processor": platform.processor(),
        "timestamp": datetime.now().isoformat()
    }
`
}

// getOfficialPythonMain generates the main entry point
func (g *Generator) getOfficialPythonMain(templateType string, data map[string]interface{}) string {
	return `#!/usr/bin/env python3
"""Main entry point for {{.ProjectName}} MCP Server."""

import asyncio
import sys
from pathlib import Path

# Add src to Python path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from server import create_server


async def main():
    """Main entry point."""
    try:
        server = create_server()
        await server.run()
    except KeyboardInterrupt:
        print("\nShutting down server...")
    except Exception as e:
        print(f"Server error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
`
}

// getOfficialPythonDockerfile generates a simple Dockerfile
func (g *Generator) getOfficialPythonDockerfile(templateType string, data map[string]interface{}) string {
	return `# Official Python MCP Server Dockerfile
FROM python:3.11-slim

# Install uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# Set working directory
WORKDIR /app

# Copy dependency files
COPY pyproject.toml .python-version ./
COPY uv.lock ./

# Install dependencies
RUN uv sync --frozen --no-dev --no-cache

# Copy source code
COPY src/ ./src/
COPY main.py ./

# Create non-root user
RUN groupadd -r mcpuser && useradd -r -g mcpuser mcpuser

# Change ownership to non-root user
RUN chown -R mcpuser:mcpuser /app

# Switch to non-root user
USER mcpuser

# Set environment variables
ENV PATH="/app/.venv/bin:$PATH"
ENV PYTHONPATH=/app
ENV PYTHONUNBUFFERED=1

# Default command
CMD ["python", "main.py"]
`
}

// getOfficialPythonGitignore generates .gitignore
func (g *Generator) getOfficialPythonGitignore(templateType string, data map[string]interface{}) string {
	return `# Python
__pycache__/
*.py[cod]
*$py.class
*.so
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
*.egg-info/
.installed.cfg
*.egg
MANIFEST

# Virtual environments
.env
.venv
env/
venv/
ENV/
env.bak/
venv.bak/

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Testing
.pytest_cache/
.coverage
htmlcov/
.tox/
.nox/

# MyPy
.mypy_cache/
.dmypy.json
dmypy.json

# Ruff
.ruff_cache/

# uv
.python-version
`
}

// getOfficialPythonEnvExample generates .env.example
func (g *Generator) getOfficialPythonEnvExample(templateType string, data map[string]interface{}) string {
	return `# {{.ProjectName}} Environment Variables
# Copy this file to .env and update with your values

# Logging
LOG_LEVEL=INFO

# API Keys (add your own)
# API_KEY=your-api-key-here
# DATABASE_URL=your-database-url-here

# {{.ProjectName}} specific settings
# Add your custom environment variables here
`
}

// getOfficialPythonPythonVersion generates .python-version
func (g *Generator) getOfficialPythonPythonVersion(templateType string, data map[string]interface{}) string {
	return `3.11`
}

// getOfficialPythonTestServer generates server tests
func (g *Generator) getOfficialPythonTestServer(templateType string, data map[string]interface{}) string {
	return `"""Tests for {{.ProjectName}} MCP Server."""

import pytest
import sys
from pathlib import Path

# Add src to Python path
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from server import create_server


class TestServer:
    """Test cases for the MCP server."""
    
    def test_server_creation(self):
        """Test server creation."""
        server = create_server()
        assert server is not None
        assert server.server.name == "{{.ProjectName}}"
    
    def test_available_tools(self):
        """Test that server has expected tools."""
        server = create_server()
        tool_names = [tool.name for tool in server.tools]
        
        assert "echo" in tool_names
        assert "calculate" in tool_names
        assert "system_info" in tool_names
`
}

// getOfficialPythonTestTools generates tool tests
func (g *Generator) getOfficialPythonTestTools(templateType string, data map[string]interface{}) string {
	return `"""Tests for {{.ProjectName}} MCP Server tools."""

import pytest
import sys
from pathlib import Path

# Add src to Python path
sys.path.insert(0, str(Path(__file__).parent.parent / "src"))

from tools import call_tool


class TestTools:
    """Test cases for MCP tools."""
    
    @pytest.mark.asyncio
    async def test_echo_tool(self):
        """Test echo tool."""
        result = await call_tool("echo", {"message": "Hello, World!"})
        
        assert result["message"] == "Hello, World!"
        assert result["length"] == 13
        assert result["server"] == "{{.ProjectName}}"
        assert "timestamp" in result
    
    @pytest.mark.asyncio
    async def test_calculate_tool_add(self):
        """Test calculator addition."""
        result = await call_tool("calculate", {
            "operation": "add",
            "a": 5,
            "b": 3
        })
        
        assert result["result"] == 8
        assert result["operation"] == "add"
        assert result["inputs"] == {"a": 5, "b": 3}
    
    @pytest.mark.asyncio
    async def test_calculate_tool_divide_by_zero(self):
        """Test calculator division by zero."""
        with pytest.raises(ValueError, match="Division by zero"):
            await call_tool("calculate", {
                "operation": "divide",
                "a": 5,
                "b": 0
            })
    
    @pytest.mark.asyncio
    async def test_system_info_tool(self):
        """Test system info tool."""
        result = await call_tool("system_info", {})
        
        assert "platform" in result
        assert "python_version" in result
        assert "timestamp" in result
    
    @pytest.mark.asyncio
    async def test_unknown_tool(self):
        """Test calling unknown tool."""
        with pytest.raises(KeyError, match="Unknown tool"):
            await call_tool("unknown_tool", {})
`
}

// Template-specific tools (placeholders)
func (g *Generator) getOfficialPythonHTTPClientTools(templateType string, data map[string]interface{}) string {
	return `"""HTTP client tools for {{.ProjectName}} MCP Server."""

from typing import Any, Dict, List
from mcp.types import Tool


def get_http_client_tools() -> List[Tool]:
    """Get HTTP client-specific tools."""
    return [
        Tool(
            name="http_request",
            description="Make an HTTP request",
            inputSchema={
                "type": "object",
                "properties": {
                    "url": {
                        "type": "string",
                        "description": "URL to make request to"
                    },
                    "method": {
                        "type": "string",
                        "enum": ["GET", "POST", "PUT", "DELETE"],
                        "default": "GET",
                        "description": "HTTP method"
                    }
                },
                "required": ["url"]
            }
        )
    ]


async def http_request_tool(arguments: Dict[str, Any]) -> Dict[str, Any]:
    """Make an HTTP request."""
    url = arguments.get("url", "")
    method = arguments.get("method", "GET")
    
    # TODO: Implement HTTP client
    return {
        "message": "HTTP client integration coming soon",
        "url": url,
        "method": method,
        "timestamp": "2025-01-16T14:49:30Z"
    }
`
}

func (g *Generator) getOfficialPythonDataProcessorTools(templateType string, data map[string]interface{}) string {
	return `"""Data processor tools for {{.ProjectName}} MCP Server."""

from typing import Any, Dict, List
from mcp.types import Tool


def get_data_processor_tools() -> List[Tool]:
    """Get data processor-specific tools."""
    return [
        Tool(
            name="process_data",
            description="Process data using a predefined algorithm",
            inputSchema={
                "type": "object",
                "properties": {
                    "algorithm": {
                        "type": "string",
                        "enum": ["sum", "average", "count"],
                        "description": "The algorithm to apply"
                    },
                    "data": {
                        "type": "array",
                        "items": {
                            "type": "number"
                        },
                        "description": "List of numbers to process"
                    }
                },
                "required": ["algorithm", "data"]
            }
        )
    ]


async def process_data_tool(arguments: Dict[str, Any]) -> Dict[str, Any]:
    """Process data using a predefined algorithm."""
    algorithm = arguments.get("algorithm")
    data = arguments.get("data")
    
    if not isinstance(data, list) or not all(isinstance(item, (int, float)) for item in data):
        raise ValueError("Data must be a list of numbers")
    
    if algorithm == "sum":
        result = sum(data)
    elif algorithm == "average":
        result = sum(data) / len(data)
    elif algorithm == "count":
        result = len(data)
    else:
        raise ValueError(f"Unknown algorithm: {algorithm}")
    
    return {
        "result": result,
        "algorithm": algorithm,
        "inputs": {"algorithm": algorithm, "data": data}
    }
`
}

func (g *Generator) getOfficialPythonWorkflowExecutorTools(templateType string, data map[string]interface{}) string {
	return `"""Workflow executor tools for {{.ProjectName}} MCP Server."""

from typing import Any, Dict, List
from mcp.types import Tool


def get_workflow_executor_tools() -> List[Tool]:
    """Get workflow executor-specific tools."""
    return [
        Tool(
            name="execute_workflow",
            description="Execute a predefined workflow",
            inputSchema={
                "type": "object",
                "properties": {
                    "workflow_name": {
                        "type": "string",
                        "description": "Name of the workflow to execute"
                    }
                },
                "required": ["workflow_name"]
            }
        )
    ]


async def execute_workflow_tool(arguments: Dict[str, Any]) -> Dict[str, Any]:
    """Execute a predefined workflow."""
    workflow_name = arguments.get("workflow_name")
    
    # TODO: Implement workflow execution logic
    return {
        "message": f"Workflow '{workflow_name}' execution coming soon",
        "workflow_name": workflow_name,
        "timestamp": "2025-01-16T14:49:30Z"
    }
`
}
