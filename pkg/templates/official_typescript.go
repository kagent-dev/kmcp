package templates

// getOfficialTypeScriptFiles returns the file templates for Official TypeScript SDK projects
func (g *Generator) getOfficialTypeScriptFiles(templateType string, data map[string]interface{}) map[string]string {
	files := map[string]string{
		"package.json":  g.getOfficialTypeScriptPackageJson(templateType, data),
		"tsconfig.json": g.getOfficialTypeScriptTsConfig(templateType, data),
		"README.md":     g.getOfficialTypeScriptReadme(templateType, data),
		"Dockerfile":    g.getOfficialTypeScriptDockerfile(templateType, data),
		".gitignore":    g.getOfficialTypeScriptGitignore(templateType, data),
		".env.example":  g.getOfficialTypeScriptEnvExample(templateType, data),

		// Official SDK structure - minimal and focused
		"src/index.ts":  g.getOfficialTypeScriptMain(templateType, data),
		"src/server.ts": g.getOfficialTypeScriptServer(templateType, data),
		"src/tools.ts":  g.getOfficialTypeScriptTools(templateType, data),

		// Tests
		"src/index.test.ts": g.getOfficialTypeScriptTest(templateType, data),
		"jest.config.js":    g.getOfficialTypeScriptJestConfig(templateType, data),

		// Dev tools
		"nodemon.json": g.getOfficialTypeScriptNodemonConfig(templateType, data),
		".eslintrc.js": g.getOfficialTypeScriptEslintConfig(templateType, data),
		".prettierrc":  g.getOfficialTypeScriptPrettierConfig(templateType, data),
	}

	// Add template-specific files
	switch templateType {
	case "http":
		files["src/http-client-tools.ts"] = g.getOfficialTypeScriptHTTPClientTools(templateType, data)
	case "data":
		files["src/data-processor-tools.ts"] = g.getOfficialTypeScriptDataProcessorTools(templateType, data)
	case "workflow":
		files["src/workflow-executor-tools.ts"] = g.getOfficialTypeScriptWorkflowExecutorTools(templateType, data)
	case "multi-tool":
		files["src/http-client-tools.ts"] = g.getOfficialTypeScriptHTTPClientTools(templateType, data)
		files["src/data-processor-tools.ts"] = g.getOfficialTypeScriptDataProcessorTools(templateType, data)
		files["src/workflow-executor-tools.ts"] = g.getOfficialTypeScriptWorkflowExecutorTools(templateType, data)
	}

	return files
}

// getOfficialTypeScriptPackageJson generates a minimal package.json
func (g *Generator) getOfficialTypeScriptPackageJson(templateType string, data map[string]interface{}) string {
	return `{
  "name": "{{.ProjectNameKebab}}",
  "version": "0.1.0",
  "description": "{{.ProjectName}} MCP server built with Official TypeScript SDK",
  "main": "dist/index.js",
  "type": "module",
  "scripts": {
    "build": "tsc",
    "start": "node dist/index.js",
    "dev": "nodemon",
    "test": "jest",
    "lint": "eslint src --ext .ts",
    "lint:fix": "eslint src --ext .ts --fix",
    "format": "prettier --write src/**/*.ts",
    "clean": "rm -rf dist"
  },
  "keywords": ["mcp", "typescript", "official", "sdk"],
  "author": {
    "name": "{{.Author}}",
    "email": "{{.Email}}"
  },
  "license": "MIT",
  "dependencies": {
    "@modelcontextprotocol/sdk": "^0.4.0",
    "zod": "^3.22.4"{{if eq .Template "database"}},
    "pg": "^8.11.3",
    "@types/pg": "^8.10.9"{{end}}{{if eq .Template "filesystem"}},
    "chokidar": "^3.5.3",
    "fs-extra": "^11.1.1",
    "@types/fs-extra": "^11.0.4"{{end}}{{if eq .Template "api-client"}},
    "axios": "^1.6.2",
    "node-fetch": "^3.3.2"{{end}}{{if eq .Template "multi-tool"}},
    "pg": "^8.11.3",
    "@types/pg": "^8.10.9",
    "chokidar": "^3.5.3",
    "fs-extra": "^11.1.1",
    "@types/fs-extra": "^11.0.4",
    "axios": "^1.6.2",
    "node-fetch": "^3.3.2"{{end}}
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

// getOfficialTypeScriptTsConfig generates tsconfig.json
func (g *Generator) getOfficialTypeScriptTsConfig(templateType string, data map[string]interface{}) string {
	return `{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "lib": ["ES2020"],
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "declaration": true,
    "sourceMap": true,
    "resolveJsonModule": true,
    "allowSyntheticDefaultImports": true,
    "moduleResolution": "node",
    "noEmitOnError": true,
    "isolatedModules": true
  },
  "include": [
    "src/**/*"
  ],
  "exclude": [
    "node_modules",
    "dist",
    "**/*.test.ts"
  ]
}`
}

// getOfficialTypeScriptReadme generates README
func (g *Generator) getOfficialTypeScriptReadme(templateType string, data map[string]interface{}) string {
	return `# {{.ProjectName}}

A Model Context Protocol (MCP) server built with the Official TypeScript SDK.

## Overview

This MCP server provides {{if eq .Template "basic"}}basic tools and functionality{{else if eq .Template "database"}}database integration capabilities{{else if eq .Template "filesystem"}}filesystem access and management{{else if eq .Template "api-client"}}API client integration{{else if eq .Template "multi-tool"}}comprehensive multi-tool functionality{{else}}custom MCP tools{{end}} using the official MCP TypeScript SDK.

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

` + "```bash" + `
npm run dev
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
      "command": "node",
      "args": ["dist/index.js"],
      "cwd": "/path/to/project"
    }
  }
}
` + "```" + `

### Configuration

Edit ` + "`.env`" + ` to configure environment variables for your server.

### Adding New Tools

1. Define your tool in ` + "`src/tools.ts`" + `
2. Register it in ` + "`src/server.ts`" + `
3. Follow the MCP specification for tool definitions

## Development

### Running Tests

` + "```bash" + `
npm test
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

## Resources

- [MCP Specification](https://spec.modelcontextprotocol.io/)
- [Official TypeScript SDK](https://github.com/modelcontextprotocol/typescript-sdk)
- [MCP Documentation](https://modelcontextprotocol.io/)

## License

This project is licensed under the MIT License.
`
}

// getOfficialTypeScriptMain generates the main entry point
func (g *Generator) getOfficialTypeScriptMain(templateType string, data map[string]interface{}) string {
	return `#!/usr/bin/env node
/**
 * {{.ProjectName}} MCP Server
 * Built with Official TypeScript SDK
 */

import { createServer } from './server.js';

async function main() {
  try {
    const server = await createServer();
    await server.run();
  } catch (error) {
    console.error('Server error:', error);
    process.exit(1);
  }
}

if (import.meta.url === ` + "`file://${process.argv[1]}`" + `) {
  main();
}
`
}

// getOfficialTypeScriptServer generates the server implementation
func (g *Generator) getOfficialTypeScriptServer(templateType string, data map[string]interface{}) string {
	return `/**
 * {{.ProjectName}} MCP Server using Official TypeScript SDK
 */

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  Tool,
  CallToolResult,
  TextContent,
  McpError,
  ErrorCode,
} from '@modelcontextprotocol/sdk/types.js';

import { getAvailableTools, callTool } from './tools.js';

export class {{.ProjectNamePascal}}Server {
  private server: Server;
  private tools: Tool[];

  constructor() {
    this.server = new Server(
      {
        name: '{{.ProjectName}}',
        version: '0.1.0',
      },
      {
        capabilities: {
          tools: {},
        },
      }
    );
    
    this.tools = getAvailableTools();
    this.setupToolHandlers();
  }

  private setupToolHandlers(): void {
    // Handle tool listing
    this.server.setRequestHandler(ListToolsRequestSchema, async () => {
      return {
        tools: this.tools,
      };
    });

    // Handle tool calls
    this.server.setRequestHandler(CallToolRequestSchema, async (request) => {
      const { name, arguments: args } = request.params;

      try {
        const result = await callTool(name, args || {});
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify(result, null, 2),
            } as TextContent,
          ],
        } as CallToolResult;
      } catch (error) {
        if (error instanceof Error && error.message.includes('Unknown tool')) {
          throw new McpError(ErrorCode.InvalidRequest, ` + "`Unknown tool: ${name}`" + `);
        }
        
        console.error(` + "`Error calling tool ${name}:`" + `, error);
        throw new McpError(ErrorCode.InternalError, ` + "`Tool execution failed: ${error}`" + `);
      }
    });
  }

  async run(): Promise<void> {
    const transport = new StdioServerTransport();
    await this.server.connect(transport);
    console.error('{{.ProjectName}} MCP Server running on stdio');
    console.error(` + "`Available tools: ${this.tools.map(t => t.name).join(', ')}`" + `);
  }
}

export function createServer(): {{.ProjectNamePascal}}Server {
  return new {{.ProjectNamePascal}}Server();
}
`
}

// getOfficialTypeScriptTools generates the tools implementation
func (g *Generator) getOfficialTypeScriptTools(templateType string, data map[string]interface{}) string {
	return `/**
 * Tool implementations for {{.ProjectName}} MCP Server
 */

import { Tool } from '@modelcontextprotocol/sdk/types.js';
import { z } from 'zod';

// Tool schemas
const EchoArgsSchema = z.object({
  message: z.string().describe('Message to echo back'),
});

const CalculateArgsSchema = z.object({
  operation: z.enum(['add', 'subtract', 'multiply', 'divide']).describe('The operation to perform'),
  a: z.number().describe('First number'),
  b: z.number().describe('Second number'),
});

const SystemInfoArgsSchema = z.object({});

export function getAvailableTools(): Tool[] {
  return [
    {
      name: 'echo',
      description: 'Echo a message back to the client',
      inputSchema: {
        type: 'object',
        properties: {
          message: {
            type: 'string',
            description: 'Message to echo back',
          },
        },
        required: ['message'],
      },
    },
    {
      name: 'calculate',
      description: 'Perform basic arithmetic calculations',
      inputSchema: {
        type: 'object',
        properties: {
          operation: {
            type: 'string',
            enum: ['add', 'subtract', 'multiply', 'divide'],
            description: 'The operation to perform',
          },
          a: {
            type: 'number',
            description: 'First number',
          },
          b: {
            type: 'number',
            description: 'Second number',
          },
        },
        required: ['operation', 'a', 'b'],
      },
    },
    {
      name: 'system_info',
      description: 'Get basic system information',
      inputSchema: {
        type: 'object',
        properties: {},
        required: [],
      },
    },
  ];
}

export async function callTool(name: string, args: any): Promise<any> {
  switch (name) {
    case 'echo':
      return await echoTool(EchoArgsSchema.parse(args));
    case 'calculate':
      return await calculateTool(CalculateArgsSchema.parse(args));
    case 'system_info':
      return await systemInfoTool(SystemInfoArgsSchema.parse(args));
    default:
      throw new Error(` + "`Unknown tool: ${name}`" + `);
  }
}

async function echoTool(args: z.infer<typeof EchoArgsSchema>): Promise<any> {
  return {
    message: args.message,
    timestamp: new Date().toISOString(),
    length: args.message.length,
    server: '{{.ProjectName}}',
  };
}

async function calculateTool(args: z.infer<typeof CalculateArgsSchema>): Promise<any> {
  const { operation, a, b } = args;
  
  let result: number;
  
  switch (operation) {
    case 'add':
      result = a + b;
      break;
    case 'subtract':
      result = a - b;
      break;
    case 'multiply':
      result = a * b;
      break;
    case 'divide':
      if (b === 0) {
        throw new Error('Division by zero is not allowed');
      }
      result = a / b;
      break;
  }
  
  return {
    result: Math.round(result * 100) / 100,
    operation,
    inputs: { a, b },
  };
}

async function systemInfoTool(args: z.infer<typeof SystemInfoArgsSchema>): Promise<any> {
  return {
    platform: process.platform,
    nodeVersion: process.version,
    architecture: process.arch,
    uptime: process.uptime(),
    memory: process.memoryUsage(),
    timestamp: new Date().toISOString(),
  };
}
`
}

// getOfficialTypeScriptDockerfile generates Dockerfile
func (g *Generator) getOfficialTypeScriptDockerfile(templateType string, data map[string]interface{}) string {
	return `# Official TypeScript MCP Server Dockerfile
FROM node:18-alpine

# Create app directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Create non-root user
RUN addgroup -g 1001 -S mcpuser && \
    adduser -S mcpuser -u 1001

# Change ownership to non-root user
RUN chown -R mcpuser:mcpuser /app

# Switch to non-root user
USER mcpuser

# Default command
CMD ["npm", "start"]
`
}

// getOfficialTypeScriptGitignore generates .gitignore
func (g *Generator) getOfficialTypeScriptGitignore(templateType string, data map[string]interface{}) string {
	return `# Dependencies
node_modules/

# Build output
dist/
build/

# Environment files
.env
.env.local
.env.*.local

# Logs
npm-debug.log*
yarn-debug.log*
yarn-error.log*
*.log

# Runtime data
pids
*.pid
*.seed
*.pid.lock

# Coverage
coverage/
*.lcov
.nyc_output

# TypeScript
*.tsbuildinfo

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Testing
.jest/
`
}

// getOfficialTypeScriptEnvExample generates .env.example
func (g *Generator) getOfficialTypeScriptEnvExample(templateType string, data map[string]interface{}) string {
	return `# {{.ProjectName}} Environment Variables
# Copy this file to .env and update with your values

# Server Configuration
NODE_ENV=development
LOG_LEVEL=info

# API Keys (add your own)
# API_KEY=your-api-key-here
# DATABASE_URL=your-database-url-here

# {{.ProjectName}} specific settings
# Add your custom environment variables here
`
}

// getOfficialTypeScriptTest generates tests
func (g *Generator) getOfficialTypeScriptTest(templateType string, data map[string]interface{}) string {
	return `/**
 * Tests for {{.ProjectName}} MCP Server
 */

import { createServer } from './server.js';
import { callTool } from './tools.js';

describe('{{.ProjectName}} MCP Server', () => {
  let server: any;

  beforeAll(async () => {
    server = createServer();
  });

  it('should create server successfully', () => {
    expect(server).toBeDefined();
  });

  it('should have expected tools', () => {
    expect(server.tools).toBeDefined();
    const toolNames = server.tools.map((t: any) => t.name);
    expect(toolNames).toContain('echo');
    expect(toolNames).toContain('calculate');
    expect(toolNames).toContain('system_info');
  });
});

describe('Tool Functions', () => {
  it('should handle echo tool', async () => {
    const result = await callTool('echo', { message: 'Hello, World!' });
    
    expect(result).toHaveProperty('message', 'Hello, World!');
    expect(result).toHaveProperty('timestamp');
    expect(result).toHaveProperty('length', 13);
    expect(result).toHaveProperty('server', '{{.ProjectName}}');
  });

  it('should handle calculate tool', async () => {
    const result = await callTool('calculate', {
      operation: 'add',
      a: 5,
      b: 3,
    });
    
    expect(result).toHaveProperty('result', 8);
    expect(result).toHaveProperty('operation', 'add');
    expect(result).toHaveProperty('inputs', { a: 5, b: 3 });
  });

  it('should handle calculator division by zero', async () => {
    await expect(callTool('calculate', {
      operation: 'divide',
      a: 5,
      b: 0,
    })).rejects.toThrow('Division by zero is not allowed');
  });

  it('should handle system_info tool', async () => {
    const result = await callTool('system_info', {});
    
    expect(result).toHaveProperty('platform');
    expect(result).toHaveProperty('nodeVersion');
    expect(result).toHaveProperty('architecture');
    expect(result).toHaveProperty('uptime');
    expect(result).toHaveProperty('memory');
    expect(result).toHaveProperty('timestamp');
  });

  it('should handle unknown tool', async () => {
    await expect(callTool('unknown_tool', {})).rejects.toThrow('Unknown tool: unknown_tool');
  });
});
`
}

// Configuration files
func (g *Generator) getOfficialTypeScriptJestConfig(templateType string, data map[string]interface{}) string {
	return `module.exports = {
  preset: 'ts-jest/presets/default-esm',
  testEnvironment: 'node',
  roots: ['<rootDir>/src'],
  testMatch: ['**/*.test.ts'],
  transform: {
    '^.+\\.ts$': ['ts-jest', {
      useESM: true,
    }],
  },
  moduleNameMapping: {
    '^(\\.{1,2}/.*)\\.js$': '$1',
  },
  collectCoverageFrom: [
    'src/**/*.ts',
    '!src/**/*.test.ts',
  ],
  coverageDirectory: 'coverage',
  coverageReporters: ['text', 'lcov'],
};
`
}

func (g *Generator) getOfficialTypeScriptNodemonConfig(templateType string, data map[string]interface{}) string {
	return `{
  "watch": ["src"],
  "ext": "ts,json",
  "ignore": ["src/**/*.test.ts"],
  "exec": "ts-node --esm src/index.ts",
  "env": {
    "NODE_ENV": "development"
  }
}
`
}

func (g *Generator) getOfficialTypeScriptEslintConfig(templateType string, data map[string]interface{}) string {
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
    '@typescript-eslint/no-explicit-any': 'warn',
    '@typescript-eslint/no-unused-vars': 'warn',
    'no-console': 'off',
  },
};
`
}

func (g *Generator) getOfficialTypeScriptPrettierConfig(templateType string, data map[string]interface{}) string {
	return `{
  "semi": true,
  "trailingComma": "es5",
  "singleQuote": true,
  "printWidth": 80,
  "tabWidth": 2,
  "useTabs": false
}
`
}

// Template-specific tools (placeholders)
func (g *Generator) getOfficialTypeScriptHTTPClientTools(templateType string, data map[string]interface{}) string {
	return `/**
 * HTTP client tools for {{.ProjectName}} MCP Server
 */

import { Tool } from '@modelcontextprotocol/sdk/types.js';

export function getHTTPClientTools(): Tool[] {
  return [
    {
      name: 'http_request',
      description: 'Make an HTTP request',
      inputSchema: {
        type: 'object',
        properties: {
          url: {
            type: 'string',
            description: 'URL to make request to',
          },
          method: {
            type: 'string',
            enum: ['GET', 'POST', 'PUT', 'DELETE'],
            default: 'GET',
            description: 'HTTP method',
          },
        },
        required: ['url'],
      },
    },
  ];
}

export async function httpRequestTool(args: { url: string; method?: string }): Promise<any> {
  // TODO: Implement HTTP client
  return {
    message: 'HTTP client integration coming soon',
    url: args.url,
    method: args.method || 'GET',
    timestamp: new Date().toISOString(),
  };
}
`
}

func (g *Generator) getOfficialTypeScriptDataProcessorTools(templateType string, data map[string]interface{}) string {
	return `/**
 * Data processor tools for {{.ProjectName}} MCP Server
 */

import { Tool } from '@modelcontextprotocol/sdk/types.js';

export function getDataProcessorTools(): Tool[] {
  return [
    {
      name: 'process_data',
      description: 'Process data using a predefined algorithm',
      inputSchema: {
        type: 'object',
        properties: {
          data: {
            type: 'string',
            description: 'Data to process',
          },
          algorithm: {
            type: 'string',
            enum: ['reverse', 'uppercase', 'lowercase', 'trim'],
            description: 'Algorithm to apply',
          },
        },
        required: ['data', 'algorithm'],
      },
    },
  ];
}

export async function processDataTool(args: { data: string; algorithm: string }): Promise<any> {
  // TODO: Implement data processing logic
  return {
    message: 'Data processor integration coming soon',
    data: args.data,
    algorithm: args.algorithm,
    timestamp: new Date().toISOString(),
  };
}
`
}

func (g *Generator) getOfficialTypeScriptWorkflowExecutorTools(templateType string, data map[string]interface{}) string {
	return `/**
 * Workflow executor tools for {{.ProjectName}} MCP Server
 */

import { Tool } from '@modelcontextprotocol/sdk/types.js';

export function getWorkflowExecutorTools(): Tool[] {
  return [
    {
      name: 'execute_workflow',
      description: 'Execute a predefined workflow',
      inputSchema: {
        type: 'object',
        properties: {
          workflow: {
            type: 'string',
            description: 'Workflow definition in JSON format',
          },
        },
        required: ['workflow'],
      },
    },
  ];
}

export async function executeWorkflowTool(args: { workflow: string }): Promise<any> {
  // TODO: Implement workflow execution logic
  return {
    message: 'Workflow executor integration coming soon',
    workflow: args.workflow,
    timestamp: new Date().toISOString(),
  };
}
`
}
