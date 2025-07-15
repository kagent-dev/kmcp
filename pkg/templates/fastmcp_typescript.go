package templates

// getFastMCPTypeScriptFiles returns the file templates for FastMCP TypeScript projects
func (g *Generator) getFastMCPTypeScriptFiles(templateType string, data map[string]interface{}) map[string]string {
	files := map[string]string{
		"package.json":  g.getFastMCPTypeScriptPackageJson(templateType, data),
		"tsconfig.json": g.getFastMCPTypeScriptTsConfig(templateType, data),
		"README.md":     g.getFastMCPTypeScriptReadme(templateType, data),
		"Dockerfile":    g.getFastMCPTypeScriptDockerfile(templateType, data),
		".gitignore":    g.getFastMCPTypeScriptGitignore(templateType, data),
		".env.example":  g.getFastMCPTypeScriptEnvExample(templateType, data),

		// New modular structure
		"src/main.ts": g.getFastMCPTypeScriptMain(templateType, data),

		// Tools directory
		"src/tools/index.ts":      g.getFastMCPTypeScriptToolsIndex(templateType, data),
		"src/tools/echo.ts":       g.getFastMCPTypeScriptEchoTool(templateType, data),
		"src/tools/calculator.ts": g.getFastMCPTypeScriptCalculatorTool(templateType, data),

		// Resources directory
		"src/resources/index.ts": g.getFastMCPTypeScriptResourcesIndex(templateType, data),

		// Core directory (generated framework code)
		"src/core/index.ts":    g.getFastMCPTypeScriptCoreIndex(templateType, data),
		"src/core/server.ts":   g.getFastMCPTypeScriptCoreServer(templateType, data),
		"src/core/registry.ts": g.getFastMCPTypeScriptCoreRegistry(templateType, data),

		// Configuration files
		"config/server.yaml": g.getFastMCPTypeScriptServerConfig(templateType, data),
		"config/tools.yaml":  g.getFastMCPTypeScriptToolsConfig(templateType, data),

		// Tests
		"tests/tools.test.ts":  g.getFastMCPTypeScriptTestTools(templateType, data),
		"tests/server.test.ts": g.getFastMCPTypeScriptTestServer(templateType, data),
		"jest.config.js":       g.getFastMCPTypeScriptJestConfig(templateType, data),

		// Build and dev tools
		"nodemon.json": g.getFastMCPTypeScriptNodemonConfig(templateType, data),
		".eslintrc.js": g.getFastMCPTypeScriptEslintConfig(templateType, data),
		".prettierrc":  g.getFastMCPTypeScriptPrettierConfig(templateType, data),
	}

	// Add template-specific files
	switch templateType {
	case "http":
		files["src/tools/http-client.ts"] = g.getFastMCPTypeScriptHTTPTool(templateType, data)
	case "data":
		files["src/tools/data-processor.ts"] = g.getFastMCPTypeScriptDataTool(templateType, data)
	case "workflow":
		files["src/tools/workflow-executor.ts"] = g.getFastMCPTypeScriptWorkflowTool(templateType, data)
	case "multi-tool":
		files["src/tools/http-client.ts"] = g.getFastMCPTypeScriptHTTPTool(templateType, data)
		files["src/tools/data-processor.ts"] = g.getFastMCPTypeScriptDataTool(templateType, data)
		files["src/tools/workflow-executor.ts"] = g.getFastMCPTypeScriptWorkflowTool(templateType, data)
	}

	return files
}

// getFastMCPTypeScriptPackageJson generates the package.json template
func (g *Generator) getFastMCPTypeScriptPackageJson(templateType string, data map[string]interface{}) string {
	return `{
  "name": "{{.ProjectNameKebab}}",
  "version": "0.1.0",
  "description": "{{.ProjectName}} MCP server built with FastMCP TypeScript",
  "main": "dist/main.js",
  "scripts": {
    "build": "tsc",
    "start": "node dist/main.js",
    "dev": "nodemon",
    "test": "jest",
    "test:watch": "jest --watch",
    "lint": "eslint src --ext .ts",
    "lint:fix": "eslint src --ext .ts --fix",
    "format": "prettier --write src/**/*.ts",
    "clean": "rm -rf dist"
  },
  "keywords": ["mcp", "fastmcp", "typescript", "ai", "llm"],
  "author": {
    "name": "{{.Author}}",
    "email": "{{.Email}}"
  },
  "license": "MIT",
  "dependencies": {
    "@fastmcp/core": "^0.1.0",
    "@fastmcp/server": "^0.1.0",
    "yaml": "^2.3.4",
    "zod": "^3.22.4"{{if eq .Template "database"}},
    "pg": "^8.11.3",
    "@types/pg": "^8.10.9"{{end}}{{if eq .Template "filesystem"}},
    "chokidar": "^3.5.3",
    "fs-extra": "^11.1.1",
    "@types/fs-extra": "^11.0.4"{{end}}{{if eq .Template "api-client"}},
    "axios": "^1.6.2",
    "node-fetch": "^3.3.2",
    "@types/node-fetch": "^2.6.9"{{end}}{{if eq .Template "multi-tool"}},
    "pg": "^8.11.3",
    "@types/pg": "^8.10.9",
    "chokidar": "^3.5.3",
    "fs-extra": "^11.1.1",
    "@types/fs-extra": "^11.0.4",
    "axios": "^1.6.2",
    "node-fetch": "^3.3.2",
    "@types/node-fetch": "^2.6.9"{{end}}
  },
  "devDependencies": {
    "@types/node": "^20.10.5",
    "@types/jest": "^29.5.8",
    "@typescript-eslint/eslint-plugin": "^6.13.2",
    "@typescript-eslint/parser": "^6.13.2",
    "eslint": "^8.55.0",
    "jest": "^29.7.0",
    "nodemon": "^3.0.2",
    "prettier": "^3.1.0",
    "ts-jest": "^29.1.1",
    "ts-node": "^10.9.1",
    "typescript": "^5.3.3"
  },
  "engines": {
    "node": ">=18.0.0"
  }
}`
}

// getFastMCPTypeScriptTsConfig generates the tsconfig.json template
func (g *Generator) getFastMCPTypeScriptTsConfig(templateType string, data map[string]interface{}) string {
	return `{
  "compilerOptions": {
    "target": "ES2020",
    "module": "commonjs",
    "lib": ["ES2020"],
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    "resolveJsonModule": true,
    "allowSyntheticDefaultImports": true,
    "experimentalDecorators": true,
    "emitDecoratorMetadata": true,
    "moduleResolution": "node",
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"],
      "@tools/*": ["src/tools/*"],
      "@resources/*": ["src/resources/*"],
      "@core/*": ["src/core/*"]
    }
  },
  "include": [
    "src/**/*"
  ],
  "exclude": [
    "node_modules",
    "dist",
    "tests"
  ]
}`
}

// getFastMCPTypeScriptReadme generates the README.md template
func (g *Generator) getFastMCPTypeScriptReadme(templateType string, data map[string]interface{}) string {
	return `# {{.ProjectName}}

{{.ProjectName}} is a Model Context Protocol (MCP) server built with FastMCP TypeScript using a modular architecture.

## Overview

This MCP server provides {{if eq .Template "basic"}}basic tools and functionality{{else if eq .Template "database"}}database integration capabilities{{else if eq .Template "filesystem"}}filesystem access and management{{else if eq .Template "api-client"}}API client integration{{else if eq .Template "multi-tool"}}comprehensive multi-tool functionality{{else}}custom MCP tools{{end}} using a clean, modular architecture.

## Project Structure

` + "```" + `
src/
├── tools/              # Business logic implementations
│   ├── echo.ts         # Echo tool
│   ├── calculator.ts   # Calculator tool
│   └── ...
├── resources/          # Resource handlers
├── core/               # Generated framework code
│   ├── server.ts       # MCP server setup
│   └── registry.ts     # Tool registration
└── main.ts             # Entry point
config/
├── server.yaml         # Server configuration
└── tools.yaml          # Tool configuration
` + "```" + `

## Installation

### Local Development

1. **Install dependencies**:
   ` + "```bash" + `
   npm install
   ` + "```" + `

2. **Build the project**:
   ` + "```bash" + `
   npm run build
   ` + "```" + `

3. **Run the server**:
   ` + "```bash" + `
   npm start
   ` + "```" + `

### Development Mode

1. **Start development server with hot reload**:
   ` + "```bash" + `
   npm run dev
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
      "command": "node",
      "args": ["dist/main.js"],
      "cwd": "/path/to/project"
    }
  }
}
` + "```" + `

### Configuration

- **Server Configuration**: Edit ` + "`config/server.yaml`" + ` to modify server behavior
- **Tool Configuration**: Edit ` + "`config/tools.yaml`" + ` to configure individual tools
- **Environment Variables**: Copy ` + "`.env.example`" + ` to ` + "`.env`" + ` for local secrets

### Adding New Tools

1. Create a new tool file in ` + "`src/tools/`" + `
2. Implement your tool class
3. Add it to the registry in ` + "`src/core/registry.ts`" + `
4. Configure it in ` + "`config/tools.yaml`" + `

## Development

### Running Tests

` + "```bash" + `
npm test
` + "```" + `

### Watch Mode

` + "```bash" + `
npm run test:watch
` + "```" + `

### Code Formatting

` + "```bash" + `
npm run format
npm run lint:fix
` + "```" + `

### Type Checking

` + "```bash" + `
npm run build
` + "```" + `

## License

This project is licensed under the MIT License.
`
}

// getFastMCPTypeScriptDockerfile generates the Dockerfile template
func (g *Generator) getFastMCPTypeScriptDockerfile(templateType string, data map[string]interface{}) string {
	return `# Multi-stage build for {{.ProjectName}} MCP server
FROM node:18-alpine as builder

# Set working directory
WORKDIR /app

# Copy package files first for layer caching
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Production stage
FROM node:18-alpine

# Create non-root user
RUN addgroup -g 1001 -S mcpuser && \
    adduser -S mcpuser -u 1001

# Set working directory
WORKDIR /app

# Copy built application from builder stage
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package*.json ./
COPY --from=builder /app/config ./config

# Install only production dependencies
RUN npm ci --only=production && npm cache clean --force

# Change ownership to non-root user
RUN chown -R mcpuser:mcpuser /app

# Switch to non-root user
USER mcpuser

# Expose port (if needed for HTTP transport)
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD node -e "console.log('healthy')" || exit 1

# Set environment variables
ENV NODE_ENV=production

# Default command
CMD ["node", "dist/main.js"]`
}

// getFastMCPTypeScriptGitignore generates the .gitignore template
func (g *Generator) getFastMCPTypeScriptGitignore(templateType string, data map[string]interface{}) string {
	return `# Dependencies
node_modules/
npm-debug.log*
yarn-debug.log*
yarn-error.log*

# Runtime data
pids
*.pid
*.seed
*.pid.lock

# Coverage directory used by tools like istanbul
coverage/
*.lcov

# nyc test coverage
.nyc_output

# Grunt intermediate storage
.grunt

# Bower dependency directory
bower_components

# node-waf configuration
.lock-wscript

# Compiled binary addons
build/Release

# Dependency directories
jspm_packages/

# TypeScript v1 declaration files
typings/

# Optional npm cache directory
.npm

# Optional eslint cache
.eslintcache

# Optional REPL history
.node_repl_history

# Output of 'npm pack'
*.tgz

# Yarn Integrity file
.yarn-integrity

# dotenv environment variables file
.env
.env.local
.env.development.local
.env.test.local
.env.production.local

# parcel-bundler cache
.cache
.parcel-cache

# next.js build output
.next

# nuxt.js build output
.nuxt

# vuepress build output
.vuepress/dist

# Serverless directories
.serverless

# FuseBox cache
.fusebox/

# DynamoDB Local files
.dynamodb/

# TernJS port file
.tern-port

# Stores VSCode versions used for testing VSCode extensions
.vscode-test

# TypeScript build output
dist/
build/

# IDE files
.vscode/
.idea/
*.swp
*.swo
*~

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# KMCP specific
config/local.yaml
.mcpbuilder.yaml`
}

// getFastMCPTypeScriptEnvExample generates .env.example file
func (g *Generator) getFastMCPTypeScriptEnvExample(templateType string, data map[string]interface{}) string {
	return `# {{.ProjectName}} Environment Variables
# Copy this file to .env and fill in actual values

# API Keys and secrets
# API_KEY=your-api-key-here
# DATABASE_URL=postgresql://user:password@localhost:5432/database

# Server configuration
# MCP_SERVER_HOST=127.0.0.1
# MCP_SERVER_PORT=3000
# MCP_LOG_LEVEL=INFO

# Tool-specific configuration
# CALCULATOR_PRECISION=2
# ECHO_PREFIX=""
`
}

// getFastMCPTypeScriptMain generates the main entry point
func (g *Generator) getFastMCPTypeScriptMain(templateType string, data map[string]interface{}) string {
	return `/**
 * Main entry point for {{.ProjectName}} MCP server.
 * 
 * This is the minimal entry point that configures and starts the MCP server.
 * All business logic is separated into tools/ and resources/ directories.
 */

import { createServer } from './core/server';

async function main(): Promise<void> {
  try {
    const server = await createServer();
    await server.start();
    
    console.log('{{.ProjectName}} MCP server is running...');
    
    // Handle graceful shutdown
    process.on('SIGINT', async () => {
      console.log('\nShutting down server...');
      await server.stop();
      process.exit(0);
    });
  } catch (error) {
    console.error('Server error:', error);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}
`
}

// getFastMCPTypeScriptToolsIndex generates the tools index
func (g *Generator) getFastMCPTypeScriptToolsIndex(templateType string, data map[string]interface{}) string {
	return `/**
 * Tools package for {{.ProjectName}} MCP server.
 * 
 * This package contains the business logic implementations for MCP tools.
 * Each tool is implemented as a separate module for maintainability.
 */

export { EchoTool } from './echo';
export { CalculatorTool } from './calculator';

// Export tool types
export type { EchoRequest } from './echo';
export type { CalculationRequest } from './calculator';
`
}

// getFastMCPTypeScriptEchoTool generates the echo tool implementation
func (g *Generator) getFastMCPTypeScriptEchoTool(templateType string, data map[string]interface{}) string {
	return `/**
 * Echo tool implementation for {{.ProjectName}} MCP server.
 */

import { z } from 'zod';

export const EchoRequestSchema = z.object({
  message: z.string().describe('Message to echo back'),
});

export type EchoRequest = z.infer<typeof EchoRequestSchema>;

export interface EchoResponse {
  message: string;
  timestamp: string;
  length: number;
  server: string;
}

export interface EchoToolConfig {
  enabled?: boolean;
  prefix?: string;
}

export class EchoTool {
  private config: EchoToolConfig;

  constructor(config: EchoToolConfig = {}) {
    this.config = {
      enabled: true,
      prefix: '',
      ...config,
    };
  }

  /**
   * Echo a message back to the client.
   * 
   * This is a simple tool that returns the input message along with
   * a timestamp, useful for testing connectivity and basic functionality.
   */
  async echo(request: EchoRequest): Promise<EchoResponse | { error: string }> {
    if (!this.config.enabled) {
      return { error: 'Echo tool is disabled' };
    }

    let message = request.message;
    if (this.config.prefix) {
      message = ` + "`${this.config.prefix}${message}`" + `;
    }

    return {
      message,
      timestamp: new Date().toISOString(),
      length: message.length,
      server: '{{.ProjectName}}',
    };
  }
}
`
}

// getFastMCPTypeScriptCalculatorTool generates the calculator tool implementation
func (g *Generator) getFastMCPTypeScriptCalculatorTool(templateType string, data map[string]interface{}) string {
	return `/**
 * Calculator tool implementation for {{.ProjectName}} MCP server.
 */

import { z } from 'zod';

export const CalculationRequestSchema = z.object({
  operation: z.enum(['add', 'subtract', 'multiply', 'divide']).describe('The operation to perform'),
  a: z.number().describe('First number'),
  b: z.number().describe('Second number'),
});

export type CalculationRequest = z.infer<typeof CalculationRequestSchema>;

export interface CalculationResponse {
  result: number;
  operation: string;
  inputs: { a: number; b: number };
}

export interface CalculatorToolConfig {
  enabled?: boolean;
  operations?: string[];
  precision?: number;
}

export class CalculatorTool {
  private config: CalculatorToolConfig;

  constructor(config: CalculatorToolConfig = {}) {
    this.config = {
      enabled: true,
      operations: ['add', 'subtract', 'multiply', 'divide'],
      precision: 2,
      ...config,
    };
  }

  /**
   * Perform basic arithmetic calculations.
   * 
   * This tool can perform addition, subtraction, multiplication, and division
   * operations on two numbers.
   */
  async calculate(request: CalculationRequest): Promise<CalculationResponse | { error: string; [key: string]: any }> {
    if (!this.config.enabled) {
      return { error: 'Calculator tool is disabled' };
    }

    if (!this.config.operations?.includes(request.operation)) {
      return {
        error: ` + "`Operation '${request.operation}' not supported`" + `,
        supported_operations: this.config.operations,
      };
    }

    try {
      let result: number;

      switch (request.operation) {
        case 'add':
          result = request.a + request.b;
          break;
        case 'subtract':
          result = request.a - request.b;
          break;
        case 'multiply':
          result = request.a * request.b;
          break;
        case 'divide':
          if (request.b === 0) {
            return {
              error: 'Division by zero is not allowed',
              operation: request.operation,
              inputs: { a: request.a, b: request.b },
            };
          }
          result = request.a / request.b;
          break;
        default:
          return {
            error: ` + "`Unknown operation: ${request.operation}`" + `,
            operation: request.operation,
            inputs: { a: request.a, b: request.b },
          };
      }

      // Apply precision if configured
      if (this.config.precision !== undefined) {
        result = Math.round(result * Math.pow(10, this.config.precision)) / Math.pow(10, this.config.precision);
      }

      return {
        result,
        operation: request.operation,
        inputs: { a: request.a, b: request.b },
      };
    } catch (error) {
      return {
        error: ` + "`Calculation error: ${error instanceof Error ? error.message : String(error)}`" + `,
        operation: request.operation,
        inputs: { a: request.a, b: request.b },
      };
    }
  }
}
`
}

// Continue with remaining methods...

// getFastMCPTypeScriptResourcesIndex generates the resources index
func (g *Generator) getFastMCPTypeScriptResourcesIndex(templateType string, data map[string]interface{}) string {
	return `/**
 * Resources package for {{.ProjectName}} MCP server.
 * 
 * This package contains the resource handler implementations for MCP resources.
 * Resources represent data or content that can be accessed by the AI model.
 */

// Future: Add resource implementations here
// export { FileResource } from './file-resource';
// export { WebResource } from './web-resource';

// Export available resources
export {};
`
}

// getFastMCPTypeScriptCoreIndex generates the core index
func (g *Generator) getFastMCPTypeScriptCoreIndex(templateType string, data map[string]interface{}) string {
	return `/**
 * Core framework package for {{.ProjectName}} MCP server.
 * 
 * This package contains generated framework code that handles MCP protocol
 * communication and tool registration. Do not edit files in this package
 * manually - they are generated by the KMCP CLI.
 */

export { createServer } from './server';
export { ToolRegistry } from './registry';
`
}

// getFastMCPTypeScriptCoreServer generates the core server implementation
func (g *Generator) getFastMCPTypeScriptCoreServer(templateType string, data map[string]interface{}) string {
	return `/**
 * Core MCP server implementation for {{.ProjectName}}.
 * 
 * This file is generated by the KMCP CLI. Do not edit manually.
 */

import { readFile } from 'fs/promises';
import { join } from 'path';
import { FastMCPServer } from '@fastmcp/server';
import * as yaml from 'yaml';
import { ToolRegistry } from './registry';

interface ServerConfig {
  name?: string;
  transport?: {
    type?: string;
    host?: string;
    port?: number;
  };
  logging?: {
    level?: string;
  };
}

interface ToolsConfig {
  tools?: Record<string, any>;
  resources?: Record<string, any>;
}

async function loadConfig<T>(configPath: string): Promise<T> {
  try {
    const content = await readFile(configPath, 'utf-8');
    return yaml.parse(content) || {};
  } catch (error) {
    console.warn(` + "`Warning: Could not load config from ${configPath}:`" + `, error);
    return {} as T;
  }
}

export async function createServer(): Promise<FastMCPServer> {
  // Load configuration
  const configDir = join(process.cwd(), 'config');
  const serverConfig = await loadConfig<ServerConfig>(join(configDir, 'server.yaml'));
  const toolsConfig = await loadConfig<ToolsConfig>(join(configDir, 'tools.yaml'));

  // Create FastMCP server
  const serverName = serverConfig.name || '{{.ProjectName}} Server';
  const server = new FastMCPServer({
    name: serverName,
    version: '0.1.0',
  });

  // Initialize tool registry
  const registry = new ToolRegistry(toolsConfig);

  // Register tools with the server
  await registry.registerTools(server);

  return server;
}
`
}

// getFastMCPTypeScriptCoreRegistry generates the tool registry
func (g *Generator) getFastMCPTypeScriptCoreRegistry(templateType string, data map[string]interface{}) string {
	return `/**
 * Tool registry for {{.ProjectName}} MCP server.
 * 
 * This file is generated by the KMCP CLI. Do not edit manually.
 */

import { FastMCPServer } from '@fastmcp/server';
import { EchoTool, EchoRequestSchema } from '../tools/echo';
import { CalculatorTool, CalculationRequestSchema } from '../tools/calculator';

interface ToolsConfig {
  tools?: Record<string, any>;
  resources?: Record<string, any>;
}

export class ToolRegistry {
  private config: ToolsConfig;
  private tools: Record<string, any> = {};

  constructor(config: ToolsConfig) {
    this.config = config;
    this.initializeTools();
  }

  private initializeTools(): void {
    const toolsConfig = this.config.tools || {};

    // Initialize echo tool
    const echoConfig = toolsConfig.echo || {};
    if (echoConfig.enabled !== false) {
      this.tools.echo = new EchoTool(echoConfig);
    }

    // Initialize calculator tool
    const calcConfig = toolsConfig.calculator || {};
    if (calcConfig.enabled !== false) {
      this.tools.calculator = new CalculatorTool(calcConfig);
    }
  }

  async registerTools(server: FastMCPServer): Promise<void> {
    // Register echo tool
    if (this.tools.echo) {
      server.tool('echo', {
        description: 'Echo messages back to the client',
        inputSchema: EchoRequestSchema,
      }, async (request) => {
        return await this.tools.echo.echo(request);
      });
    }

    // Register calculator tool
    if (this.tools.calculator) {
      server.tool('calculate', {
        description: 'Perform basic arithmetic calculations',
        inputSchema: CalculationRequestSchema,
      }, async (request) => {
        return await this.tools.calculator.calculate(request);
      });
    }
  }
}
`
}

// getFastMCPTypeScriptServerConfig generates server configuration
func (g *Generator) getFastMCPTypeScriptServerConfig(templateType string, data map[string]interface{}) string {
	return `# {{.ProjectName}} MCP Server Configuration
# This file configures the overall server behavior

name: "{{.ProjectName}} Server"
description: "{{.ProjectName}} MCP server built with FastMCP TypeScript"
version: "0.1.0"

# Transport configuration
transport:
  type: "stdio"  # stdio, http, or websocket
  host: "127.0.0.1"
  port: 3000

# Logging configuration
logging:
  level: "INFO"  # DEBUG, INFO, WARNING, ERROR, CRITICAL

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

// getFastMCPTypeScriptToolsConfig generates tools configuration
func (g *Generator) getFastMCPTypeScriptToolsConfig(templateType string, data map[string]interface{}) string {
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

// Continue with remaining template methods...

// getFastMCPTypeScriptTestTools generates tool tests
func (g *Generator) getFastMCPTypeScriptTestTools(templateType string, data map[string]interface{}) string {
	return `/**
 * Tests for {{.ProjectName}} MCP server tools.
 */

import { EchoTool } from '../src/tools/echo';
import { CalculatorTool } from '../src/tools/calculator';

describe('EchoTool', () => {
  it('should echo basic message', async () => {
    const tool = new EchoTool();
    const result = await tool.echo({ message: 'Hello, World!' });
    
    expect(result).toMatchObject({
      message: 'Hello, World!',
      length: 13,
      server: '{{.ProjectName}}',
    });
    expect(result).toHaveProperty('timestamp');
  });

  it('should echo with prefix', async () => {
    const tool = new EchoTool({ prefix: '[TEST] ' });
    const result = await tool.echo({ message: 'Hello' });
    
    expect(result).toMatchObject({
      message: '[TEST] Hello',
      length: 12,
    });
  });

  it('should handle disabled tool', async () => {
    const tool = new EchoTool({ enabled: false });
    const result = await tool.echo({ message: 'Hello' });
    
    expect(result).toHaveProperty('error');
    expect((result as any).error).toContain('disabled');
  });
});

describe('CalculatorTool', () => {
  it('should perform addition', async () => {
    const tool = new CalculatorTool();
    const result = await tool.calculate({ operation: 'add', a: 5, b: 3 });
    
    expect(result).toMatchObject({
      result: 8,
      operation: 'add',
      inputs: { a: 5, b: 3 },
    });
  });

  it('should handle division by zero', async () => {
    const tool = new CalculatorTool();
    const result = await tool.calculate({ operation: 'divide', a: 5, b: 0 });
    
    expect(result).toHaveProperty('error');
    expect((result as any).error).toContain('Division by zero');
  });

  it('should handle invalid operation', async () => {
    const tool = new CalculatorTool();
    const result = await tool.calculate({ operation: 'invalid' as any, a: 5, b: 3 });
    
    expect(result).toHaveProperty('error');
    expect((result as any).error).toContain('not supported');
  });

  it('should apply precision', async () => {
    const tool = new CalculatorTool({ precision: 1 });
    const result = await tool.calculate({ operation: 'divide', a: 10, b: 3 });
    
    expect(result).toMatchObject({
      result: 3.3,
      operation: 'divide',
    });
  });

  it('should handle disabled tool', async () => {
    const tool = new CalculatorTool({ enabled: false });
    const result = await tool.calculate({ operation: 'add', a: 5, b: 3 });
    
    expect(result).toHaveProperty('error');
    expect((result as any).error).toContain('disabled');
  });
});
`
}

// getFastMCPTypeScriptTestServer generates server tests
func (g *Generator) getFastMCPTypeScriptTestServer(templateType string, data map[string]interface{}) string {
	return `/**
 * Tests for {{.ProjectName}} MCP server core functionality.
 */

import { createServer } from '../src/core/server';
import { ToolRegistry } from '../src/core/registry';

describe('Server Configuration', () => {
  it('should create server successfully', async () => {
    const server = await createServer();
    expect(server).toBeDefined();
  });
});

describe('ToolRegistry', () => {
  it('should initialize with default config', () => {
    const registry = new ToolRegistry({});
    expect(registry).toBeDefined();
  });

  it('should initialize tools based on config', () => {
    const config = {
      tools: {
        echo: { enabled: true },
        calculator: { enabled: true },
      },
    };
    const registry = new ToolRegistry(config);
    expect(registry).toBeDefined();
  });

  it('should handle disabled tools', () => {
    const config = {
      tools: {
        echo: { enabled: false },
        calculator: { enabled: true },
      },
    };
    const registry = new ToolRegistry(config);
    expect(registry).toBeDefined();
  });
});
`
}

// getFastMCPTypeScriptJestConfig generates Jest configuration
func (g *Generator) getFastMCPTypeScriptJestConfig(templateType string, data map[string]interface{}) string {
	return `module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>/src', '<rootDir>/tests'],
  testMatch: ['**/__tests__/**/*.ts', '**/?(*.)+(spec|test).ts'],
  transform: {
    '^.+\\.ts$': 'ts-jest',
  },
  collectCoverageFrom: [
    'src/**/*.ts',
    '!src/**/*.d.ts',
    '!src/**/*.test.ts',
    '!src/**/*.spec.ts',
  ],
  coverageDirectory: 'coverage',
  coverageReporters: ['text', 'lcov', 'html'],
  moduleNameMapping: {
    '^@/(.*)$': '<rootDir>/src/$1',
    '^@tools/(.*)$': '<rootDir>/src/tools/$1',
    '^@resources/(.*)$': '<rootDir>/src/resources/$1',
    '^@core/(.*)$': '<rootDir>/src/core/$1',
  },
  setupFilesAfterEnv: ['<rootDir>/jest.setup.js'],
};
`
}

// getFastMCPTypeScriptNodemonConfig generates Nodemon configuration
func (g *Generator) getFastMCPTypeScriptNodemonConfig(templateType string, data map[string]interface{}) string {
	return `{
  "watch": ["src", "config"],
  "ext": "ts,yaml,yml,json",
  "ignore": ["src/**/*.test.ts", "src/**/*.spec.ts"],
  "exec": "ts-node src/main.ts",
  "env": {
    "NODE_ENV": "development"
  }
}
`
}

// getFastMCPTypeScriptEslintConfig generates ESLint configuration
func (g *Generator) getFastMCPTypeScriptEslintConfig(templateType string, data map[string]interface{}) string {
	return `module.exports = {
  parser: '@typescript-eslint/parser',
  parserOptions: {
    project: 'tsconfig.json',
    sourceType: 'module',
  },
  plugins: ['@typescript-eslint/eslint-plugin'],
  extends: [
    'eslint:recommended',
    '@typescript-eslint/recommended',
  ],
  root: true,
  env: {
    node: true,
    jest: true,
  },
  ignorePatterns: ['.eslintrc.js', 'dist/**/*'],
  rules: {
    '@typescript-eslint/interface-name-prefix': 'off',
    '@typescript-eslint/explicit-function-return-type': 'off',
    '@typescript-eslint/explicit-module-boundary-types': 'off',
    '@typescript-eslint/no-explicit-any': 'warn',
    '@typescript-eslint/no-unused-vars': 'warn',
    '@typescript-eslint/prefer-const': 'error',
    'no-console': 'warn',
  },
};
`
}

// getFastMCPTypeScriptPrettierConfig generates Prettier configuration
func (g *Generator) getFastMCPTypeScriptPrettierConfig(templateType string, data map[string]interface{}) string {
	return `{
  "semi": true,
  "trailingComma": "es5",
  "singleQuote": true,
  "printWidth": 100,
  "tabWidth": 2,
  "useTabs": false
}
`
}

// Placeholder implementations for template-specific tools
func (g *Generator) getFastMCPTypeScriptHTTPTool(templateType string, data map[string]interface{}) string {
	return `/**
 * HTTP client tool implementation for {{.ProjectName}} MCP server.
 */

import { z } from 'zod';
import axios from 'axios';

export const HTTPRequestSchema = z.object({
  url: z.string().describe('URL to make request to'),
  method: z.enum(['GET', 'POST', 'PUT', 'DELETE', 'PATCH']).default('GET').describe('HTTP method'),
  headers: z.record(z.string()).optional().describe('HTTP headers'),
  body: z.string().optional().describe('Request body'),
});

export type HTTPRequest = z.infer<typeof HTTPRequestSchema>;

export interface HTTPToolConfig {
  enabled?: boolean;
  timeout?: number;
  allowedDomains?: string[];
}

export class HTTPTool {
  private config: HTTPToolConfig;

  constructor(config: HTTPToolConfig = {}) {
    this.config = {
      enabled: true,
      timeout: 30000,
      ...config,
    };
  }

  async httpRequest(request: HTTPRequest): Promise<any> {
    if (!this.config.enabled) {
      return { error: 'HTTP client tool is disabled' };
    }

    // TODO: Implement HTTP client
    return {
      message: 'HTTP client integration template - implementation coming soon',
      url: request.url,
      method: request.method,
      headers: request.headers,
    };
  }
}
`
}

func (g *Generator) getFastMCPTypeScriptDataTool(templateType string, data map[string]interface{}) string {
	return `/**
 * Data processor tool implementation for {{.ProjectName}} MCP server.
 */

import { z } from 'zod';
import { Pool } from 'pg';

export const DataQueryRequestSchema = z.object({
  query: z.string().describe('SQL query to execute'),
  params: z.array(z.any()).optional().describe('Query parameters'),
});

export type DataQueryRequest = z.infer<typeof DataQueryRequestSchema>;

export interface DataToolConfig {
  enabled?: boolean;
  connectionString?: string;
  maxResults?: number;
}

export class DataTool {
  private config: DataToolConfig;
  private pool?: Pool;

  constructor(config: DataToolConfig = {}) {
    this.config = {
      enabled: true,
      maxResults: 100,
      ...config,
    };
  }

  async query(request: DataQueryRequest): Promise<any> {
    if (!this.config.enabled) {
      return { error: 'Data processor tool is disabled' };
    }

    // TODO: Implement database connectivity
    return {
      message: 'Data processor integration template - implementation coming soon',
      query: request.query,
      params: request.params,
    };
  }
}
`
}

func (g *Generator) getFastMCPTypeScriptWorkflowTool(templateType string, data map[string]interface{}) string {
	return `/**
 * Workflow executor tool implementation for {{.ProjectName}} MCP server.
 */

import { z } from 'zod';
import { Pool } from 'pg';

export const WorkflowRequestSchema = z.object({
  workflow: z.string().describe('JSON workflow definition'),
  inputs: z.record(z.any()).optional().describe('Input data for the workflow'),
});

export type WorkflowRequest = z.infer<typeof WorkflowRequestSchema>;

export interface WorkflowToolConfig {
  enabled?: boolean;
  connectionString?: string;
  maxSteps?: number;
}

export class WorkflowTool {
  private config: WorkflowToolConfig;
  private pool?: Pool;

  constructor(config: WorkflowToolConfig = {}) {
    this.config = {
      enabled: true,
      maxSteps: 10,
      ...config,
    };
  }

  async executeWorkflow(request: WorkflowRequest): Promise<any> {
    if (!this.config.enabled) {
      return { error: 'Workflow executor tool is disabled' };
    }

    // TODO: Implement workflow execution logic
    return {
      message: 'Workflow executor integration template - implementation coming soon',
      workflow: request.workflow,
      inputs: request.inputs,
    };
  }
}
`
}
