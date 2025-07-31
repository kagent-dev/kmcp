<div align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/kagent-dev/kmcp/main/img/kmcp-logo-dark.svg" alt="kmcp" width="400">
    <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/kagent-dev/kmcp/main/img/kmcp-logo-light.svg" alt="kmcp" width="400">
    <img alt="kmcp" src="https://raw.githubusercontent.com/kagent-dev/kmcp/main/img/kmcp-logo-light.svg">
  </picture>
  <div>
    <a href="https://github.com/kagent-dev/kmcp/releases">
      <img src="https://img.shields.io/github/v/release/kagent-dev/kmcp?style=flat&label=Latest%20version" alt="Release">
    </a>
    <a href="https://github.com/kagent-dev/kmcp/actions/workflows/tag.yaml">
      <img src="https://github.com/kagent-dev/kmcp/actions/workflows/tag.yaml/badge.svg" alt="Build Status" height="20">
    </a>
    <a href="https://opensource.org/licenses/Apache-2.0">
      <img src="https://img.shields.io/badge/License-Apache2.0-brightgreen.svg?style=flat" alt="License: Apache 2.0">
    </a>
    <a href="https://github.com/kagent-dev/kmcp">
      <img src="https://img.shields.io/github/stars/kagent-dev/kmcp.svg?style=flat&logo=github&label=Stars" alt="Stars">
    </a>
    <a href="https://discord.gg/Fu3k65f2k3">
      <img src="https://img.shields.io/discord/1346225185166065826?style=flat&label=Join%20Discord&color=6D28D9" alt="Discord">
    </a>
  </div>
</div>

# kmcp

**Model Context Protocol Development & Deployment Platform**

Kmcp is a platform for developing and deploying Model Context Protocol (MCP) servers particularly for cloud-native deployment. Kmcp provides a kubernetes controller that automates the lifecycle management of an MCP server deployment alongside a CLI tool that supports MCP project scaffolding in multiple languages and functionality to deploy your MCP server to a kubernetes environment.

## Get started

- [Quick Start](https://kagent.dev/docs/kmcp/quick-start)

## Documentation

The kmcp documentation is available at [kagent.dev/docs/kmcp](https://kagent.dev/docs/kmcp).

## 🎯 What is MCP?

The Model Context Protocol (MCP) is an open standard developed by Anthropic that standardizes how AI applications provide context to Large Language Models (LLMs). Think of MCP as a universal adapter that allows AI models to seamlessly connect to various data sources and external tools.

### Key Benefits:

- **Standardized Data Access**: Consistent way for AI models to access external data
- **Tool Integration**: Connect AI assistants to business tools and internal systems  
- **Structured Responses**: Ensure consistent, structured outputs from AI models
- **Improved Context**: Give AI models the context they need for better responses
- **Vendor Independence**: Open standard that works across different AI providers

### MCP Architecture:

- **MCP Clients**: AI applications (Claude Desktop, Cursor, etc.) that consume MCP services
- **MCP Servers**: Lightweight programs that expose capabilities through standardized interfaces
- **Transport Protocols**: Communication via stdio (standard input/output) and HTTP with SSE

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

Thanks to all contributors!

<a href="https://github.com/kagent-dev/kmcp/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=kagent-dev/kmcp" />
</a>

## 📈 Star History

<a href="https://www.star-history.com/#kagent-dev/kmcp&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=kagent-dev/kmcp&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=kagent-dev/kmcp&type=Date" />
   <img alt="Star history of kagent-dev/kmcp over time" src="https://api.star-history.com/svg?repos=kagent-dev/kmcp&type=Date" />
 </picture>
</a>

## 📄 License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## 🔗 Resources

- [Model Context Protocol Specification](https://spec.modelcontextprotocol.io/)
- [MCP Documentation](https://modelcontextprotocol.io/)
- [Anthropic's MCP Announcement](https://www.anthropic.com/news/model-context-protocol)
- [FastMCP Python Documentation](https://github.com/jlowin/fastmcp)
