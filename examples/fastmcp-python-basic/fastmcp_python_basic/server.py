"""Main FastMCP server implementation with example tools."""

import asyncio
import json
import os
import sys
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional

from fastmcp import FastMCP
from pydantic import BaseModel, Field


class CalculationRequest(BaseModel):
    """Request model for calculation operations."""
    operation: str = Field(..., description="The operation to perform: add, subtract, multiply, divide")
    a: float = Field(..., description="First number")
    b: float = Field(..., description="Second number")


class FileListRequest(BaseModel):
    """Request model for listing files."""
    directory: str = Field(".", description="Directory to list files from")
    pattern: str = Field("*", description="File pattern to match")


class SystemInfoResponse(BaseModel):
    """Response model for system information."""
    timestamp: str
    python_version: str
    platform: str
    working_directory: str
    environment_variables: Dict[str, str]


# Initialize FastMCP server
mcp = FastMCP("Basic MCP Server")


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


@mcp.tool()
def list_files(request: FileListRequest) -> Dict[str, Any]:
    """List files in a directory with optional pattern matching.
    
    This tool lists files in the specified directory and can filter
    results using glob patterns.
    """
    try:
        directory = Path(request.directory)
        
        if not directory.exists():
            return {
                "error": f"Directory does not exist: {request.directory}",
                "directory": request.directory
            }
        
        if not directory.is_dir():
            return {
                "error": f"Path is not a directory: {request.directory}",
                "directory": request.directory
            }
        
        # List files matching pattern
        files = []
        for file_path in directory.glob(request.pattern):
            if file_path.is_file():
                stat = file_path.stat()
                files.append({
                    "name": file_path.name,
                    "path": str(file_path),
                    "size": stat.st_size,
                    "modified": datetime.fromtimestamp(stat.st_mtime).isoformat(),
                    "is_file": True
                })
            elif file_path.is_dir():
                files.append({
                    "name": file_path.name,
                    "path": str(file_path),
                    "is_file": False
                })
        
        return {
            "directory": request.directory,
            "pattern": request.pattern,
            "files": files,
            "count": len(files)
        }
    except Exception as e:
        return {
            "error": f"File listing error: {str(e)}",
            "directory": request.directory,
            "pattern": request.pattern
        }


@mcp.tool()
def system_info() -> SystemInfoResponse:
    """Get system information and environment details.
    
    This tool provides information about the current system environment,
    Python version, and other runtime details.
    """
    import platform
    
    # Get a subset of environment variables (excluding sensitive ones)
    env_vars = {}
    safe_vars = ["PATH", "HOME", "USER", "SHELL", "LANG", "PWD"]
    for var in safe_vars:
        if var in os.environ:
            env_vars[var] = os.environ[var]
    
    return SystemInfoResponse(
        timestamp=datetime.now().isoformat(),
        python_version=sys.version,
        platform=platform.platform(),
        working_directory=os.getcwd(),
        environment_variables=env_vars
    )


@mcp.tool()
def echo(message: str) -> Dict[str, Any]:
    """Echo a message back to the client.
    
    This is a simple tool that returns the input message along with
    a timestamp, useful for testing connectivity and basic functionality.
    """
    return {
        "message": message,
        "timestamp": datetime.now().isoformat(),
        "length": len(message)
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
    main() 