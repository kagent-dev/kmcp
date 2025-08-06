# Development

- [cmd](cmd): Contains the code for KMCP CLI.
- [pkg/agentgateway](pkg/agentgateway/): Contains the code for the kubernetes controller.
- [pkg/frameworks](pkg/frameworks/): Contains the generator code for the supported MCP frameworks (fastmcp-python, mcp-go).


## How to run everything locally

Running locally.

1. Build the KMCP CLI.

```shell
make build-cli
```

2. Create your mcp project.

```shell
dist/kmcp init python my-mcp-python
```

3. Run project locally via the [mcp inspector](https://github.com/modelcontextprotocol/inspector)

```shell
dist/kmcp run --project-dir ./my-mcp-python/
```
----------------------------------------------------------------

Running in a kubernetes environment. 

1. Create a cluster.

```shell
kind create cluster --name kind
```

2. Package the helm chart.

```shell
make helm-package VERSION=<version_number>
```

3. Install the KMCP helm chart.

```shell
helm install kmcp dist/kmcp-<version_number>.tgz --namespace kmcp-system --create-namespace
```

4. Build and load your mcp docker image into the kind cluster.

```bash
dist/kmcp build --project-dir my-mcp-python --kind-load
```

5. Deploy your mcp server.

```bash
kmcp deploy --file my-mcp-python/kmcp.yaml
```

Your MCP server will be automatically port-forwarded on port 3000 and the MCP inspector will be spun up so you can access it on `http://localhost:6274`.