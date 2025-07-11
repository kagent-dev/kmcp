# KMCP Helm Chart

A Helm chart for deploying the KMCP (Kubernetes MCP Server Controller) to Kubernetes clusters.

## Overview

KMCP is a Kubernetes controller that manages MCP (Model Context Protocol) servers. It provides a declarative way to deploy and manage MCP servers in your Kubernetes environment.

## Prerequisites

- Kubernetes 1.11.3+
- Helm 3.0+

## Installation

### Add the Helm Repository

```bash
# Add the repository (once available)
helm repo add kmcp https://charts.kagent.dev
helm repo update
```

### Install the Chart

```bash
# Install with default values
helm install kmcp kmcp/kmcp

# Install in a specific namespace
helm install kmcp kmcp/kmcp --namespace kmcp-system --create-namespace

# Install with custom values
helm install kmcp kmcp/kmcp --values values.yaml
```

## Configuration

The following table lists the configurable parameters of the KMCP chart and their default values.

### Image Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Container image repository | `controller` |
| `image.tag` | Container image tag | `""` (uses appVersion) |
| `image.pullPolicy` | Container image pull policy | `IfNotPresent` |
| `imagePullSecrets` | Image pull secrets | `[]` |

### Controller Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `controller.replicaCount` | Number of controller replicas | `1` |
| `controller.leaderElection.enabled` | Enable leader election | `true` |
| `controller.healthProbe.bindAddress` | Health probe bind address | `:8081` |
| `controller.metrics.enabled` | Enable metrics endpoint | `true` |
| `controller.metrics.bindAddress` | Metrics bind address | `:8443` |

### RBAC Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `rbac.create` | Create RBAC resources | `true` |
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.name` | Service account name | `""` (generated) |
| `serviceAccount.annotations` | Service account annotations | `{}` |

### Resource Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `128Mi` |
| `resources.requests.cpu` | CPU request | `10m` |
| `resources.requests.memory` | Memory request | `64Mi` |

### Security Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `podSecurityContext.runAsNonRoot` | Run as non-root user | `true` |
| `podSecurityContext.seccompProfile.type` | Seccomp profile type | `RuntimeDefault` |
| `securityContext.allowPrivilegeEscalation` | Allow privilege escalation | `false` |
| `securityContext.capabilities.drop` | Capabilities to drop | `["ALL"]` |

### Service Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `8443` |
| `service.targetPort` | Service target port | `8443` |

### Custom Resource Definition

| Parameter | Description | Default |
|-----------|-------------|---------|
| `crd.create` | Create CRDs | `true` |

## Usage

After installation, you can create MCP servers using the `MCPServer` custom resource:

```yaml
apiVersion: kagent.dev/v1alpha1
kind: MCPServer
metadata:
  name: my-mcp-server
spec:
  transportType: http
  deployment:
    image: my-mcp-server:latest
    port: 8080
  httpTransport:
    targetPort: 8080
```

## Monitoring

The controller exposes metrics on port 8443 by default. These metrics can be scraped by Prometheus or other monitoring systems.

## Upgrading

To upgrade the chart:

```bash
helm upgrade kmcp kmcp/kmcp
```

## Uninstalling

To uninstall the chart:

```bash
helm uninstall kmcp
```

**Note**: This will remove all the Kubernetes resources associated with the chart and delete the release.

## Contributing

For information on contributing to this project, please see the [main repository](https://github.com/kagent-dev/kmcp).

## License

This project is licensed under the terms specified in the main repository. 