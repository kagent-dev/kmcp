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
	"strings"
	"testing"

	"sigs.k8s.io/yaml"

	"github.com/kagent-dev/kmcp/pkg/controller/transportadapter"
)

func TestBuildSharedProxyConfig(t *testing.T) {
	routes := []ProxyRoute{
		{Name: "zeta", Namespace: "team-b", ActorHost: "mcp-team-b-zeta.actors.resources.substrate.ate.dev"},
		{Name: "srv", Namespace: "team-a", ActorHost: "mcp-team-a-srv.actors.resources.substrate.ate.dev"},
	}

	configYAML, err := BuildSharedProxyConfig(routes, Defaults{}, DefaultProxyPort)
	if err != nil {
		t.Fatalf("BuildSharedProxyConfig: %v", err)
	}

	var cfg transportadapter.LocalConfig
	if err := yaml.Unmarshal([]byte(configYAML), &cfg); err != nil {
		t.Fatalf("unmarshal proxy config: %v", err)
	}
	if len(cfg.Binds) != 1 || cfg.Binds[0].Port != DefaultProxyPort {
		t.Fatalf("binds = %+v, want one bind on %d", cfg.Binds, DefaultProxyPort)
	}
	localRoutes := cfg.Binds[0].Listeners[0].Routes
	if len(localRoutes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(localRoutes))
	}

	// Routes are sorted by namespace/name for deterministic output.
	first := localRoutes[0]
	if first.RouteName != "team-a/srv" {
		t.Fatalf("routes not sorted: first route is %q", first.RouteName)
	}
	wantHostnames := []string{"srv.team-a", "srv.team-a.svc", "srv.team-a.svc.cluster.local"}
	if len(first.Hostnames) != len(wantHostnames) {
		t.Fatalf("hostnames = %v", first.Hostnames)
	}
	for i, hostname := range wantHostnames {
		if first.Hostnames[i] != hostname {
			t.Fatalf("hostnames = %v, want %v", first.Hostnames, wantHostnames)
		}
	}
	if first.Matches[0].Path.PathPrefix != "/" {
		t.Fatalf("route match = %+v", first.Matches)
	}
	actorHost := "mcp-team-a-srv.actors.resources.substrate.ate.dev"
	if first.Policies == nil || first.Policies.URLRewrite == nil ||
		first.Policies.URLRewrite.Authority == nil || first.Policies.URLRewrite.Authority.Full != actorHost {
		t.Fatalf("route must rewrite authority to the actor host: %+v", first.Policies)
	}
	if first.Policies.RequestHeaderModifier == nil || first.Policies.RequestHeaderModifier.Set["Host"] != actorHost {
		t.Fatalf("route must set the Host header: %+v", first.Policies.RequestHeaderModifier)
	}
	if first.Backends[0].Host != "atenet-router.ate-system.svc:80" {
		t.Fatalf("route backend = %+v", first.Backends)
	}
}

func TestBuildSharedProxyConfigEmpty(t *testing.T) {
	configYAML, err := BuildSharedProxyConfig(nil, Defaults{}, DefaultProxyPort)
	if err != nil {
		t.Fatalf("BuildSharedProxyConfig: %v", err)
	}
	var cfg transportadapter.LocalConfig
	if err := yaml.Unmarshal([]byte(configYAML), &cfg); err != nil {
		t.Fatalf("unmarshal proxy config: %v", err)
	}
	if len(cfg.Binds) != 0 {
		t.Fatalf("expected no binds without routes, got %+v", cfg.Binds)
	}
}

func TestBuildServerIngressObjects(t *testing.T) {
	scheme := newTestScheme(t)
	server := validHTTPServer()
	server.Spec.Deployment.Port = 3000

	service, slice, err := BuildServerIngressObjects(server, scheme, DefaultProxyPort,
		[]string{"10.244.1.17", "10.244.2.31"})
	if err != nil {
		t.Fatalf("BuildServerIngressObjects: %v", err)
	}

	if service.Name != server.Name || service.Namespace != server.Namespace {
		t.Fatalf("service identity = %s/%s", service.Namespace, service.Name)
	}
	if service.Spec.Selector != nil {
		t.Fatalf("service must have no selector, got %v", service.Spec.Selector)
	}
	if service.Spec.Ports[0].Port != 3000 {
		t.Fatalf("service port = %+v", service.Spec.Ports[0])
	}
	if service.Spec.Ports[0].AppProtocol == nil || *service.Spec.Ports[0].AppProtocol != "kgateway.dev/mcp" {
		t.Fatalf("service appProtocol = %v", service.Spec.Ports[0].AppProtocol)
	}

	if slice.Name != server.Name+"-proxy" {
		t.Fatalf("slice name = %q", slice.Name)
	}
	if slice.Labels["kubernetes.io/service-name"] != server.Name {
		t.Fatalf("slice must bind to the service: labels = %v", slice.Labels)
	}
	if slice.Labels["endpointslice.kubernetes.io/managed-by"] != EndpointSliceManagedBy {
		t.Fatalf("slice managed-by label = %v", slice.Labels)
	}
	// The slice port name must match the Service port name, and its number is
	// the proxy listen port (the port remap for a selector-less Service).
	if *slice.Ports[0].Name != service.Spec.Ports[0].Name {
		t.Fatalf("slice port name %q != service port name %q", *slice.Ports[0].Name, service.Spec.Ports[0].Name)
	}
	if *slice.Ports[0].Port != int32(DefaultProxyPort) {
		t.Fatalf("slice port = %d, want %d", *slice.Ports[0].Port, DefaultProxyPort)
	}
	if len(slice.Endpoints) != 2 ||
		slice.Endpoints[0].Addresses[0] != "10.244.1.17" ||
		slice.Endpoints[1].Addresses[0] != "10.244.2.31" {
		t.Fatalf("slice endpoints = %+v", slice.Endpoints)
	}

	for _, labeled := range []map[string]string{service.Labels, slice.Labels} {
		if labeled[RoleLabelKey] != RoleIngressProxy {
			t.Fatalf("missing ingress proxy role label: %v", labeled)
		}
	}
	if len(service.OwnerReferences) != 1 || len(slice.OwnerReferences) != 1 {
		t.Fatal("ingress objects must be owner-referenced to the MCPServer")
	}

	if got := ProxyEndpoint(server); got != "http://srv.ns.svc:3000/mcp" {
		t.Fatalf("ProxyEndpoint = %q", got)
	}
}

func TestSharedProxyConfigMapMeta(t *testing.T) {
	meta := SharedProxyConfigMapMeta(IngressConfig{
		Mode:           IngressModeManagedProxy,
		ProxyNamespace: "kmcp-system",
	})
	if meta.Name != DefaultProxyConfigMapName || meta.Namespace != "kmcp-system" {
		t.Fatalf("meta = %s/%s", meta.Namespace, meta.Name)
	}
	if meta.Labels[RoleLabelKey] != RoleIngressProxy {
		t.Fatalf("meta labels = %v", meta.Labels)
	}
}

func TestSplitRouterURL(t *testing.T) {
	host, port, err := splitRouterURL("http://atenet-router.ate-system.svc:80")
	if err != nil || host != "atenet-router.ate-system.svc" || port != 80 {
		t.Fatalf("splitRouterURL = %q %d %v", host, port, err)
	}
	host, port, err = splitRouterURL("https://router.example")
	if err != nil || host != "router.example" || port != 443 {
		t.Fatalf("splitRouterURL https default = %q %d %v", host, port, err)
	}
	if _, _, err = splitRouterURL("http://"); err == nil {
		t.Fatal("expected error for URL without host")
	}
}

func TestEndpointSliceNameTruncation(t *testing.T) {
	server := validHTTPServer()
	server.Name = strings.Repeat("a", 250)
	if name := EndpointSliceName(server); len(name) > 253 {
		t.Fatalf("EndpointSliceName too long: %d", len(name))
	}
}
