/*
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
*/

package substrate

import (
	"fmt"
	"strings"

	"regexp"

	"github.com/kagent-dev/kmcp/api/v1alpha1"
)

const (
	// actorIDPrefix prefixes every actor id generated for an MCPServer.
	actorIDPrefix = "mcp"

	// MCPServerLabelKey labels generated ActorTemplates (and ingress proxy
	// objects) with the owning MCPServer name, so watches can map them back.
	MCPServerLabelKey = "kmcp.kagent.dev/mcp-server"

	// RoleLabelKey marks the role of generated objects.
	RoleLabelKey = "kmcp.kagent.dev/role"

	// RoleIngressProxy is the RoleLabelKey value for ingress proxy objects.
	RoleIngressProxy = "substrate-ingress-proxy"

	// FinalizerName guards MCPServer deletion until the substrate actor (and
	// its golden actor) have been removed from the control plane.
	FinalizerName = "kmcp.kagent.dev/substrate-cleanup"

	// MCPPath is the HTTP path MCP is uniformly served on through the router
	// (the in-actor agentgateway normalizes both transports onto it).
	MCPPath = "/mcp"
)

var dns1123Label = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// ActorID returns a stable DNS-1123 actor id for the MCPServer. The id doubles
// as the routing hostname label, so it must be a valid DNS label.
func ActorID(server *v1alpha1.MCPServer) string {
	raw := fmt.Sprintf("%s-%s-%s", actorIDPrefix, server.Namespace, server.Name)
	raw = strings.ToLower(raw)
	raw = strings.ReplaceAll(raw, "_", "-")
	if len(raw) > 63 {
		raw = raw[:63]
		raw = strings.TrimRight(raw, "-")
	}
	if !dns1123Label.MatchString(raw) {
		raw = fmt.Sprintf("%s-%s", actorIDPrefix, server.UID)
		if len(raw) > 63 {
			raw = raw[:63]
		}
	}
	return raw
}

// ActorHost returns the atenet router Host header value for the actor.
func ActorHost(actorID, suffix string) string {
	if suffix == "" {
		suffix = DefaultActorHostSuffix
	}
	return actorID + "." + suffix
}

// ActorTemplateName returns the name of the generated ActorTemplate, which
// lives in the MCPServer's namespace.
func ActorTemplateName(server *v1alpha1.MCPServer) string {
	return server.Name
}

// SnapshotsLocation returns the snapshot storage location for the MCPServer:
// the spec override when set, otherwise <prefix>/<namespace>/<name>.
func SnapshotsLocation(server *v1alpha1.MCPServer, prefix string) string {
	if server.Spec.Substrate != nil &&
		server.Spec.Substrate.SnapshotsConfig != nil &&
		server.Spec.Substrate.SnapshotsConfig.Location != "" {
		return server.Spec.Substrate.SnapshotsConfig.Location
	}
	if prefix == "" {
		prefix = DefaultSnapshotsLocationPrefix
	}
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(prefix, "/"), server.Namespace, server.Name)
}
