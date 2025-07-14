package templates

// getEasyMCPTypeScriptFiles returns the file templates for EasyMCP TypeScript projects
func (g *Generator) getEasyMCPTypeScriptFiles(templateType string, data map[string]interface{}) map[string]string {
	files := map[string]string{
		"package.json":                      g.getEasyMCPTypeScriptPackageJson(templateType, data),
		"tsconfig.json":                     g.getEasyMCPTypeScriptTsConfig(templateType, data),
		"README.md":                         g.getEasyMCPTypeScriptReadme(templateType, data),
		"Dockerfile":                        g.getEasyMCPTypeScriptDockerfile(templateType, data),
		".gitignore":                        g.getEasyMCPTypeScriptGitignore(templateType, data),
		".env.example":                      g.getEasyMCPTypeScriptEnvExample(templateType, data),
		
		// Simple structure - fewer files, more straightforward
		"src/index.ts":                      g.getEasyMCPTypeScriptMain(templateType, data),
		"src/tools.ts":                      g.getEasyMCPTypeScriptTools(templateType, data),
		"src/config.ts":                     g.getEasyMCPTypeScriptConfig(templateType, data),
		
		// Minimal testing setup
		"src/index.test.ts":                 g.getEasyMCPTypeScriptTest(templateType, data),
		"jest.config.js":                    g.getEasyMCPTypeScriptJestConfig(templateType, data),
		
		// Dev tools
		"nodemon.json":                      g.getEasyMCPTypeScriptNodemonConfig(templateType, data),
		".eslintrc.js":                      g.getEasyMCPTypeScriptEslintConfig(templateType, data),
		".prettierrc":                       g.getEasyMCPTypeScriptPrettierConfig(templateType, data),
	}

	// Add template-specific additional tools
	switch templateType {
	case "database":
		files["src/database.ts"] = g.getEasyMCPTypeScriptDatabase(templateType, data)
	case "filesystem":
		files["src/filesystem.ts"] = g.getEasyMCPTypeScriptFilesystem(templateType, data)
	case "api-client":
		files["src/api-client.ts"] = g.getEasyMCPTypeScriptAPIClient(templateType, data)
	case "multi-tool":
		files["src/database.ts"] = g.getEasyMCPTypeScriptDatabase(templateType, data)
		files["src/filesystem.ts"] = g.getEasyMCPTypeScriptFilesystem(templateType, data)
		files["src/api-client.ts"] = g.getEasyMCPTypeScriptAPIClient(templateType, data)
	}

	return files
}

// getEasyMCPTypeScriptPackageJson generates a simplified package.json
func (g *Generator) getEasyMCPTypeScriptPackageJson(templateType string, data map[string]interface{}) string {
	return `{
  "name": "{{.ProjectNameKebab}}",
  "version": "0.1.0",
  "description": "{{.ProjectName}} MCP server built with EasyMCP TypeScript",
  "main": "dist/index.js",
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
  "keywords": ["mcp", "easymcp", "typescript", "ai", "simple"],
  "author": {
    "name": "{{.Author}}",
    "email": "{{.Email}}"
  },
  "license": "MIT",
  "dependencies": {
    "easymcp": "^0.1.0",
    "dotenv": "^16.3.1"{{if eq .Template "database"}},
    "pg": "^8.11.3",
    "@types/pg": "^8.10.9"{{end}}{{if eq .Template "filesystem"}},
    "fs-extra": "^11.1.1",
    "@types/fs-extra": "^11.0.4"{{end}}{{if eq .Template "api-client"}},
    "axios": "^1.6.2"{{end}}{{if eq .Template "multi-tool"}},
    "pg": "^8.11.3",
    "@types/pg": "^8.10.9",
    "fs-extra": "^11.1.1",
    "@types/fs-extra": "^11.0.4",
    "axios": "^1.6.2"{{end}}
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

// getEasyMCPTypeScriptTsConfig generates a simple tsconfig.json
func (g *Generator) getEasyMCPTypeScriptTsConfig(templateType string, data map[string]interface{}) string {
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
    "sourceMap": true,
    "resolveJsonModule": true,
    "allowSyntheticDefaultImports": true,
    "moduleResolution": "node"
  },
  "include": [
    "src/**/*"
  ],
  "exclude": [
    "node_modules",
    "dist"
  ]
}`
}

// getEasyMCPTypeScriptReadme generates a simple README
func (g *Generator) getEasyMCPTypeScriptReadme(templateType string, data map[string]interface{}) string {
	return `# {{.ProjectName}}

A simple MCP server built with EasyMCP TypeScript - get started in minutes!

## 🚀 Quick Start

1. **Install dependencies**:
   ` + "```bash" + `
   npm install
   ` + "```" + `

2. **Start development**:
   ` + "```bash" + `
   npm run dev
   ` + "```" + `

3. **Build for production**:
   ` + "```bash" + `
   npm run build
   npm start
   ` + "```" + `

## 📁 Project Structure

This project uses a simple, flat structure for easy navigation:

` + "```" + `
src/
├── index.ts          # Main server entry point
├── tools.ts          # Tool implementations
├── config.ts         # Configuration management
└── index.test.ts     # Simple tests
` + "```" + `

## 🔧 Configuration

Edit ` + "`.env`" + ` or ` + "`src/config.ts`" + ` to customize your server:

` + "```typescript" + `
export const config = {
  serverName: '{{.ProjectName}}',
  logLevel: 'info',
  // Add your configuration here
};
` + "```" + `

## 🛠️ Adding Tools

Adding new tools is simple - just add them to ` + "`src/tools.ts`" + `:

` + "```typescript" + `
server.addTool('myTool', {
  description: 'My custom tool',
  parameters: {
    // Define parameters
  },
  handler: async (params) => {
    // Your tool logic here
    return { result: 'success' };
  }
});
` + "```" + `

## 📦 Build & Deploy

### Docker
` + "```bash" + `
kmcp build --docker
docker run -i {{.ProjectNameKebab}}:latest
` + "```" + `

### Node.js
` + "```bash" + `
npm run build
npm start
` + "```" + `

## 🧪 Testing

` + "```bash" + `
npm test
` + "```" + `

## 📖 Learn More

- [EasyMCP Documentation](https://easymcp.dev)
- [MCP Protocol Specification](https://spec.modelcontextprotocol.io/)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)

## 📄 License

MIT License
`
}

// getEasyMCPTypeScriptMain generates the main server file
func (g *Generator) getEasyMCPTypeScriptMain(templateType string, data map[string]interface{}) string {
	return `/**
 * {{.ProjectName}} MCP Server
 * Built with EasyMCP TypeScript for simplicity and speed
 */

import { EasyMCP } from 'easymcp';
import { config } from './config';
import { registerTools } from './tools';

async function main() {
  // Create EasyMCP server
  const server = new EasyMCP({
    name: config.serverName,
    version: '0.1.0',
    description: '{{.ProjectName}} MCP server',
  });

  // Register tools
  await registerTools(server);

  // Start server
  await server.start();

  console.log('🚀 {{.ProjectName}} MCP server is running!');
  console.log('📋 Available tools:', server.getToolNames());

  // Handle graceful shutdown
  process.on('SIGINT', async () => {
    console.log('\n🛑 Shutting down server...');
    await server.stop();
    process.exit(0);
  });
}

// Start the server
if (require.main === module) {
  main().catch((error) => {
    console.error('❌ Server error:', error);
    process.exit(1);
  });
}

export { main };
`
}

// getEasyMCPTypeScriptTools generates the tools file
func (g *Generator) getEasyMCPTypeScriptTools(templateType string, data map[string]interface{}) string {
	return `/**
 * Tool implementations for {{.ProjectName}}
 */

import { EasyMCP } from 'easymcp';

/**
 * Register all tools with the server
 */
export async function registerTools(server: EasyMCP) {
  // Echo tool - simple message echo
  server.addTool('echo', {
    description: 'Echo a message back to the client',
    parameters: {
      type: 'object',
      properties: {
        message: {
          type: 'string',
          description: 'Message to echo back'
        }
      },
      required: ['message']
    },
    handler: async (params: { message: string }) => {
      return {
        message: params.message,
        timestamp: new Date().toISOString(),
        server: '{{.ProjectName}}'
      };
    }
  });

  // Calculator tool - basic math operations
  server.addTool('calculate', {
    description: 'Perform basic arithmetic calculations',
    parameters: {
      type: 'object',
      properties: {
        operation: {
          type: 'string',
          enum: ['add', 'subtract', 'multiply', 'divide'],
          description: 'The operation to perform'
        },
        a: {
          type: 'number',
          description: 'First number'
        },
        b: {
          type: 'number',
          description: 'Second number'
        }
      },
      required: ['operation', 'a', 'b']
    },
    handler: async (params: { operation: string; a: number; b: number }) => {
      const { operation, a, b } = params;
      
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
        default:
          throw new Error(` + "`Unknown operation: ${operation}`" + `);
      }
      
      return {
        result: Math.round(result * 100) / 100, // Round to 2 decimal places
        operation,
        inputs: { a, b }
      };
    }
  });

  // System info tool - get basic system information
  server.addTool('systemInfo', {
    description: 'Get basic system information',
    parameters: {
      type: 'object',
      properties: {},
      required: []
    },
    handler: async () => {
      return {
        platform: process.platform,
        nodeVersion: process.version,
        uptime: process.uptime(),
        memory: process.memoryUsage(),
        timestamp: new Date().toISOString()
      };
    }
  });

  console.log('✅ Registered tools: echo, calculate, systemInfo');
}
`
}

// getEasyMCPTypeScriptConfig generates the config file
func (g *Generator) getEasyMCPTypeScriptConfig(templateType string, data map[string]interface{}) string {
	return `/**
 * Configuration for {{.ProjectName}}
 */

import { config as dotenvConfig } from 'dotenv';

// Load environment variables
dotenvConfig();

export const config = {
  serverName: '{{.ProjectName}}',
  version: '0.1.0',
  logLevel: process.env.LOG_LEVEL || 'info',
  
  // Server settings
  host: process.env.HOST || '127.0.0.1',
  port: parseInt(process.env.PORT || '3000'),
  
  // Add your custom configuration here
  apiKey: process.env.API_KEY,
  databaseUrl: process.env.DATABASE_URL,
  
  // Feature flags
  features: {
    enableLogging: process.env.ENABLE_LOGGING !== 'false',
    enableMetrics: process.env.ENABLE_METRICS === 'true',
  }
};

export default config;
`
}

// getEasyMCPTypeScriptTest generates simple tests
func (g *Generator) getEasyMCPTypeScriptTest(templateType string, data map[string]interface{}) string {
	return `/**
 * Simple tests for {{.ProjectName}}
 */

import { EasyMCP } from 'easymcp';
import { registerTools } from './tools';
import { config } from './config';

describe('{{.ProjectName}} MCP Server', () => {
  let server: EasyMCP;

  beforeEach(() => {
    server = new EasyMCP({
      name: config.serverName,
      version: '0.1.0',
      description: '{{.ProjectName}} MCP server',
    });
  });

  afterEach(async () => {
    if (server) {
      await server.stop();
    }
  });

  it('should create server successfully', () => {
    expect(server).toBeDefined();
    expect(server.name).toBe(config.serverName);
  });

  it('should register tools successfully', async () => {
    await registerTools(server);
    const toolNames = server.getToolNames();
    
    expect(toolNames).toContain('echo');
    expect(toolNames).toContain('calculate');
    expect(toolNames).toContain('systemInfo');
  });

  it('should handle echo tool', async () => {
    await registerTools(server);
    
    const result = await server.executeTool('echo', { message: 'Hello, World!' });
    
    expect(result).toHaveProperty('message', 'Hello, World!');
    expect(result).toHaveProperty('timestamp');
    expect(result).toHaveProperty('server', '{{.ProjectName}}');
  });

  it('should handle calculator tool', async () => {
    await registerTools(server);
    
    const result = await server.executeTool('calculate', {
      operation: 'add',
      a: 5,
      b: 3
    });
    
    expect(result).toHaveProperty('result', 8);
    expect(result).toHaveProperty('operation', 'add');
    expect(result).toHaveProperty('inputs', { a: 5, b: 3 });
  });

  it('should handle calculator division by zero', async () => {
    await registerTools(server);
    
    await expect(server.executeTool('calculate', {
      operation: 'divide',
      a: 5,
      b: 0
    })).rejects.toThrow('Division by zero is not allowed');
  });

  it('should handle systemInfo tool', async () => {
    await registerTools(server);
    
    const result = await server.executeTool('systemInfo', {});
    
    expect(result).toHaveProperty('platform');
    expect(result).toHaveProperty('nodeVersion');
    expect(result).toHaveProperty('uptime');
    expect(result).toHaveProperty('memory');
    expect(result).toHaveProperty('timestamp');
  });
});
`
}

// getEasyMCPTypeScriptDockerfile generates a simple Dockerfile
func (g *Generator) getEasyMCPTypeScriptDockerfile(templateType string, data map[string]interface{}) string {
	return `# Simple Dockerfile for {{.ProjectName}}
FROM node:18-alpine

# Create app directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy source code
COPY . .

# Build the app
RUN npm run build

# Create non-root user
RUN addgroup -g 1001 -S mcpuser && \
    adduser -S mcpuser -u 1001

# Change ownership to non-root user
RUN chown -R mcpuser:mcpuser /app

# Switch to non-root user
USER mcpuser

# Expose port
EXPOSE 3000

# Start the server
CMD ["npm", "start"]
`
}

// getEasyMCPTypeScriptGitignore generates .gitignore
func (g *Generator) getEasyMCPTypeScriptGitignore(templateType string, data map[string]interface{}) string {
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

# Cache
.npm
.eslintcache

# Optional npm cache directory
.npm

# Optional eslint cache
.eslintcache

# Microbundle cache
.rpt2_cache/
.rts2_cache_cjs/
.rts2_cache_es/
.rts2_cache_umd/
`
}

// getEasyMCPTypeScriptEnvExample generates .env.example
func (g *Generator) getEasyMCPTypeScriptEnvExample(templateType string, data map[string]interface{}) string {
	return `# {{.ProjectName}} Environment Variables
# Copy this file to .env and update with your values

# Server Configuration
HOST=127.0.0.1
PORT=3000
LOG_LEVEL=info

# Features
ENABLE_LOGGING=true
ENABLE_METRICS=false

# API Keys (add your own)
# API_KEY=your-api-key-here
# DATABASE_URL=your-database-url-here

# {{.ProjectName}} specific settings
# Add your custom environment variables here
`
}

// Jest, Nodemon, ESLint, and Prettier configs (simplified versions)
func (g *Generator) getEasyMCPTypeScriptJestConfig(templateType string, data map[string]interface{}) string {
	return `module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>/src'],
  testMatch: ['**/*.test.ts'],
  transform: {
    '^.+\\.ts$': 'ts-jest',
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

func (g *Generator) getEasyMCPTypeScriptNodemonConfig(templateType string, data map[string]interface{}) string {
	return `{
  "watch": ["src"],
  "ext": "ts,json",
  "ignore": ["src/**/*.test.ts"],
  "exec": "ts-node src/index.ts",
  "env": {
    "NODE_ENV": "development"
  }
}
`
}

func (g *Generator) getEasyMCPTypeScriptEslintConfig(templateType string, data map[string]interface{}) string {
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
    'no-console': 'off', // Allow console for this simple setup
  },
};
`
}

func (g *Generator) getEasyMCPTypeScriptPrettierConfig(templateType string, data map[string]interface{}) string {
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

// Simplified template-specific tools
func (g *Generator) getEasyMCPTypeScriptDatabase(templateType string, data map[string]interface{}) string {
	return `/**
 * Database tools for {{.ProjectName}}
 */

import { EasyMCP } from 'easymcp';

export function registerDatabaseTools(server: EasyMCP) {
  server.addTool('queryDatabase', {
    description: 'Execute a database query',
    parameters: {
      type: 'object',
      properties: {
        query: {
          type: 'string',
          description: 'SQL query to execute'
        }
      },
      required: ['query']
    },
    handler: async (params: { query: string }) => {
      // TODO: Implement database integration
      return {
        message: 'Database integration coming soon',
        query: params.query,
        timestamp: new Date().toISOString()
      };
    }
  });
}
`
}

func (g *Generator) getEasyMCPTypeScriptFilesystem(templateType string, data map[string]interface{}) string {
	return `/**
 * Filesystem tools for {{.ProjectName}}
 */

import { EasyMCP } from 'easymcp';

export function registerFilesystemTools(server: EasyMCP) {
  server.addTool('readFile', {
    description: 'Read a file from the filesystem',
    parameters: {
      type: 'object',
      properties: {
        path: {
          type: 'string',
          description: 'Path to the file to read'
        }
      },
      required: ['path']
    },
    handler: async (params: { path: string }) => {
      // TODO: Implement safe file reading
      return {
        message: 'Filesystem integration coming soon',
        path: params.path,
        timestamp: new Date().toISOString()
      };
    }
  });
}
`
}

func (g *Generator) getEasyMCPTypeScriptAPIClient(templateType string, data map[string]interface{}) string {
	return `/**
 * API client tools for {{.ProjectName}}
 */

import { EasyMCP } from 'easymcp';

export function registerAPIClientTools(server: EasyMCP) {
  server.addTool('httpRequest', {
    description: 'Make an HTTP request',
    parameters: {
      type: 'object',
      properties: {
        url: {
          type: 'string',
          description: 'URL to make request to'
        },
        method: {
          type: 'string',
          enum: ['GET', 'POST', 'PUT', 'DELETE'],
          default: 'GET',
          description: 'HTTP method'
        }
      },
      required: ['url']
    },
    handler: async (params: { url: string; method?: string }) => {
      // TODO: Implement HTTP client
      return {
        message: 'HTTP client integration coming soon',
        url: params.url,
        method: params.method || 'GET',
        timestamp: new Date().toISOString()
      };
    }
  });
}
`
} 