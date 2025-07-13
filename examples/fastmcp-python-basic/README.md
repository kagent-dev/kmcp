# FastMCP Python Basic Example

This is a basic Model Context Protocol (MCP) server implementation using the FastMCP framework. It demonstrates common MCP patterns and provides several example tools that can be used by AI assistants.

## Overview

This MCP server provides four basic tools:
- **calculator**: Perform arithmetic operations (add, subtract, multiply, divide)
- **list_files**: List files in a directory with optional pattern matching
- **system_info**: Get system information and environment details
- **echo**: Echo messages back to the client (useful for testing)

## Features

- **FastMCP Framework**: Uses the high-level FastMCP library for simplified development
- **Type Safety**: Pydantic models for request/response validation
- **Error Handling**: Comprehensive error handling with informative messages
- **Docker Support**: Multi-stage Dockerfile for containerized deployment
- **Security**: Non-root user execution and minimal attack surface

## Installation

### Local Development

1. **Clone or navigate to this directory**:
   ```bash
   cd examples/fastmcp-python-basic
   ```

2. **Install dependencies**:
   ```bash
   pip install -e .
   ```

3. **Run the server**:
   ```bash
   python -m fastmcp_python_basic.server
   ```

### Docker Deployment

1. **Build the Docker image**:
   ```bash
   docker build -t fastmcp-python-basic .
   ```

2. **Run the container**:
   ```bash
   docker run -i fastmcp-python-basic
   ```

## Usage Examples

### Calculator Tool

Perform basic arithmetic operations:

```json
{
  "operation": "add",
  "a": 10,
  "b": 5
}
```

Response:
```json
{
  "result": 15,
  "operation": "add",
  "inputs": {"a": 10, "b": 5}
}
```

### List Files Tool

List files in a directory:

```json
{
  "directory": ".",
  "pattern": "*.py"
}
```

Response:
```json
{
  "directory": ".",
  "pattern": "*.py",
  "files": [
    {
      "name": "server.py",
      "path": "./server.py",
      "size": 1024,
      "modified": "2024-01-01T12:00:00",
      "is_file": true
    }
  ],
  "count": 1
}
```

### System Info Tool

Get system information:

```json
{}
```

Response:
```json
{
  "timestamp": "2024-01-01T12:00:00",
  "python_version": "3.11.0",
  "platform": "Darwin-21.6.0-x86_64-i386-64bit",
  "working_directory": "/app",
  "environment_variables": {
    "PATH": "/usr/local/bin:/usr/bin:/bin",
    "HOME": "/home/mcpuser"
  }
}
```

### Echo Tool

Echo a message:

```json
{
  "message": "Hello, MCP!"
}
```

Response:
```json
{
  "message": "Hello, MCP!",
  "timestamp": "2024-01-01T12:00:00",
  "length": 11
}
```

## Development

### Running Tests

```bash
pip install -e ".[dev]"
pytest
```

### Code Formatting

```bash
black .
ruff check .
```

### Type Checking

```bash
mypy .
```

## Project Structure

```
fastmcp-python-basic/
├── fastmcp_python_basic/
│   ├── __init__.py          # Package initialization
│   └── server.py            # Main server implementation
├── Dockerfile               # Multi-stage Docker build
├── pyproject.toml          # Python project configuration
└── README.md               # This file
```

## MCP Protocol Details

This server implements the Model Context Protocol (MCP) specification:

- **Transport**: STDIO (standard input/output)
- **Protocol**: JSON-RPC 2.0
- **Capabilities**: Tools (functions that can be called by AI assistants)

## Integration

This server can be integrated with various MCP clients including:

- **Claude Desktop**: Add to your MCP configuration
- **Cursor**: Use with MCP extension
- **VS Code**: MCP protocol support
- **Custom Applications**: Any application supporting MCP

### Example MCP Client Configuration

```json
{
  "mcpServers": {
    "basic-server": {
      "command": "python",
      "args": ["-m", "fastmcp_python_basic.server"],
      "cwd": "/path/to/project"
    }
  }
}
```

## Security Considerations

- Server runs as non-root user in Docker
- Limited environment variable exposure
- Input validation through Pydantic models
- No network access from tools (local operation only)

## License

This example is part of the KMCP project and follows the same license terms. 