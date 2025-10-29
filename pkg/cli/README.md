# kMCP CLI

The `kmcp` CLI provides a set of commands to manage the entire lifecycle of your MCP server

## Download and Explore

Install the kmcp CLI on your local machine.

```bash
curl -fsSL https://raw.githubusercontent.com/kagent-dev/kmcp/refs/heads/main/scripts/get-kmcp.sh | bash
```

Verify that the kmcp CLI is installed.

```bash
kmcp --help
```

<img src="/img/cli-help-nov-25.png" alt="kmcp cli help text" width="800">

## Local Development

1. Build the kMCP CLI.

```shell
make build-cli
```

This will generate the cli binary at `dist/kmcp`. And you can use it:

```shell
dist/kmcp --help
```
