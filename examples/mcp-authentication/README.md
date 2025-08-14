## MCP Authentication Example

This example shows how to protect MCP servers using the MCPServer CRD with ageentgateway using the MCP Authorization spec.


### Prerequisites

Ensure you have the KMCP controller and CRD already installed in your kubernetes environment.

### Running the example locally

Deploy keycloak to your kubernetes environment which we will use as our authorization server.

```bash
kubectl apply -f keycloak/keycloak.yaml
```

Deploy the example github mcp server

```bash
kubectl apply -f github-mcp-server.yaml
```

The example github mcp server assumes keycloak is reachable at `keycloak.default.svc.cluster.local:8080`. When testing locally add the following to your `/etc/hosts`
```
127.0.0.1 keycloak.default.svc.cluster.local
```

Then port forward keycloak

```bash
kubectl port-forward service/keycloak 8080:8080
```

Port forward the mcp server

```bash
kubectl port-forward service/github-mcp-server-with-auth 3000:3000
```

Run the mcp inspector

```bash
 npx @modelcontextprotocol/inspector
```

Upon hitting connect you will be redirected to login via keycloak for which you can use the credentials `testuser` and `testpass` to authenticate.