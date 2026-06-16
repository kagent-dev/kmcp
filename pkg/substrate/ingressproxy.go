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
	"net/url"
	"os"
	"sort"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	"github.com/kagent-dev/kmcp/api/v1alpha1"
	"github.com/kagent-dev/kmcp/pkg/controller/transportadapter"
)

// IngressMode selects how substrate MCPServers are exposed to in-cluster
// clients.
type IngressMode string

const (
	// IngressModeManagedProxy routes every substrate MCPServer Service
	// through one shared agentgateway Deployment per installation: the
	// controller writes a route block per server into a shared ConfigMap and
	// points each server's selector-less Service at the proxy pods through a
	// controller-managed EndpointSlice.
	IngressModeManagedProxy IngressMode = "managed-proxy"

	// IngressModeNone creates no ingress objects; an embedding platform
	// reaches actors directly using the status.substrate coordinates.
	IngressModeNone IngressMode = "none"
)

const (
	// DefaultProxyPort is the single port the shared ingress proxy listens
	// on. Requests are told apart by Host header, not port, so per-server
	// Services remap their spec port onto it via the EndpointSlice.
	DefaultProxyPort uint16 = 8080

	// DefaultProxyConfigMapName names the shared proxy route ConfigMap when
	// not overridden (helm derives it from the release name).
	DefaultProxyConfigMapName = "kmcp-substrate-ingress-proxy"

	// ProxyConfigKey is the file name agentgateway reads inside the shared
	// ConfigMap volume.
	ProxyConfigKey = "local.yaml"

	// EndpointSliceManagedBy marks controller-managed EndpointSlices so the
	// Kubernetes endpoint controller leaves them alone.
	EndpointSliceManagedBy = "kmcp.kagent.dev"

	// servicePortName names the per-server Service port; the EndpointSlice
	// port must carry the same name to bind to it.
	servicePortName = "mcp"

	appNameLabel     = "app.kubernetes.io/name"
	appInstanceLabel = "app.kubernetes.io/instance"
	managedByLabel   = "app.kubernetes.io/managed-by"
	managedByKmcp    = "kmcp"
)

// IngressConfig is the installation-level ingress topology the controller
// renders for substrate MCPServers. It mirrors the helm
// substrate.ingress values.
type IngressConfig struct {
	// Mode is the ingress topology (managed-proxy or none).
	Mode IngressMode

	// ProxyNamespace is where the shared proxy Deployment (and its route
	// ConfigMap) live — the kmcp installation namespace.
	ProxyNamespace string

	// ProxyConfigMapName is the name of the shared route ConfigMap mounted
	// by the proxy Deployment.
	ProxyConfigMapName string

	// ProxyPort is the proxy's single listen port.
	ProxyPort uint16
}

// ConfigMapName returns the configured shared ConfigMap name or the default.
func (c IngressConfig) ConfigMapName() string {
	if c.ProxyConfigMapName != "" {
		return c.ProxyConfigMapName
	}
	return DefaultProxyConfigMapName
}

// Port returns the configured proxy listen port or the default.
func (c IngressConfig) Port() uint16 {
	if c.ProxyPort != 0 {
		return c.ProxyPort
	}
	return DefaultProxyPort
}

// ProxyRoute is one MCPServer's entry in the shared proxy configuration.
type ProxyRoute struct {
	// Name and Namespace identify the MCPServer (and its Service hostname).
	Name      string
	Namespace string

	// ActorHost is the atenet router Host header the route rewrites to.
	ActorHost string
}

// BuildSharedProxyConfig renders the shared agentgateway config: one bind on
// the proxy port and one route per substrate MCPServer. Each route matches
// the server's Service hostnames (the port is ignored when matching) and
// rewrites the Host header to the actor host before forwarding to the atenet
// router — the rewrite is what triggers resume-from-suspend routing. Routes
// are sorted so the output is deterministic.
func BuildSharedProxyConfig(routes []ProxyRoute, defaults Defaults, port uint16) (string, error) {
	routerHost, routerPort, err := splitRouterURL(defaults.RouterURL())
	if err != nil {
		return "", err
	}

	sorted := make([]ProxyRoute, len(routes))
	copy(sorted, routes)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Namespace != sorted[j].Namespace {
			return sorted[i].Namespace < sorted[j].Namespace
		}
		return sorted[i].Name < sorted[j].Name
	})

	localRoutes := make([]transportadapter.LocalRoute, 0, len(sorted))
	for _, route := range sorted {
		localRoutes = append(localRoutes, transportadapter.LocalRoute{
			RouteName: fmt.Sprintf("%s/%s", route.Namespace, route.Name),
			Hostnames: []string{
				fmt.Sprintf("%s.%s", route.Name, route.Namespace),
				fmt.Sprintf("%s.%s.svc", route.Name, route.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.local", route.Name, route.Namespace),
			},
			Matches: []transportadapter.RouteMatch{
				{Path: transportadapter.PathMatch{PathPrefix: "/"}},
			},
			Policies: &transportadapter.FilterOrPolicy{
				URLRewrite: &transportadapter.URLRewrite{
					Authority: &transportadapter.HostRedirect{Full: route.ActorHost},
				},
				// Belt and braces: also set the Host header in case the
				// gateway resolves the upstream from the backend hostname
				// without applying the rewritten authority.
				RequestHeaderModifier: &transportadapter.HeaderModifier{
					Set: map[string]string{"Host": route.ActorHost},
				},
			},
			Backends: []transportadapter.RouteBackend{{
				Weight: 100,
				Host:   fmt.Sprintf("%s:%d", routerHost, routerPort),
			}},
		})
	}

	cfg := &transportadapter.LocalConfig{Config: struct{}{}}
	// With no routes there is nothing to serve; an empty binds list keeps the
	// config valid instead of risking an empty listener.
	if len(localRoutes) > 0 {
		cfg.Binds = []transportadapter.LocalBind{{
			Port: port,
			Listeners: []transportadapter.LocalListener{{
				Name:     "default",
				Protocol: transportadapter.LocalListenerProtocolHTTP,
				Routes:   localRoutes,
			}},
		}}
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal shared proxy config: %w", err)
	}
	return string(out), nil
}

// SharedProxyConfigMapMeta returns the identity of the shared route ConfigMap
// for the given ingress configuration.
func SharedProxyConfigMapMeta(cfg IngressConfig) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      cfg.ConfigMapName(),
		Namespace: cfg.ProxyNamespace,
		Labels: map[string]string{
			managedByLabel: managedByKmcp,
			RoleLabelKey:   RoleIngressProxy,
		},
	}
}

// BuildServerIngressObjects produces the per-MCPServer ingress plumbing for
// managed-proxy mode: a Service without a selector (it must front pods in the
// kmcp installation namespace, which a selector cannot reach) and a
// controller-managed EndpointSlice carrying the shared proxy pod IPs. The
// Service exposes the port the MCPServer spec asks for while the
// EndpointSlice carries the proxy's listen port — for a selector-less Service
// the slice port is what actually gets used, which is how the remap works.
func BuildServerIngressObjects(
	server *v1alpha1.MCPServer,
	scheme *runtime.Scheme,
	proxyPort uint16,
	proxyPodIPs []string,
) (*corev1.Service, *discoveryv1.EndpointSlice, error) {
	port := server.Spec.Deployment.Port
	if port == 0 {
		port = 3000
	}

	labels := map[string]string{
		appNameLabel:      server.Name,
		appInstanceLabel:  server.Name,
		managedByLabel:    managedByKmcp,
		MCPServerLabelKey: server.Name,
		RoleLabelKey:      RoleIngressProxy,
	}

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: corev1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      server.Name,
			Namespace: server.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			// No selector: kmcp manages the endpoints itself.
			Ports: []corev1.ServicePort{{
				Name:        servicePortName,
				Protocol:    corev1.ProtocolTCP,
				Port:        int32(port),
				AppProtocol: mcpAppProtocol(),
			}},
		},
	}

	endpoints := make([]discoveryv1.Endpoint, 0, len(proxyPodIPs))
	for _, ip := range proxyPodIPs {
		endpoints = append(endpoints, discoveryv1.Endpoint{
			Addresses:  []string{ip},
			Conditions: discoveryv1.EndpointConditions{Ready: ptr.To(true)},
		})
	}

	sliceLabels := map[string]string{
		"kubernetes.io/service-name":             server.Name,
		"endpointslice.kubernetes.io/managed-by": EndpointSliceManagedBy,
		managedByLabel:                           managedByKmcp,
		MCPServerLabelKey:                        server.Name,
		RoleLabelKey:                             RoleIngressProxy,
	}
	slice := &discoveryv1.EndpointSlice{
		TypeMeta: metav1.TypeMeta{Kind: "EndpointSlice", APIVersion: discoveryv1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      EndpointSliceName(server),
			Namespace: server.Namespace,
			Labels:    sliceLabels,
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Ports: []discoveryv1.EndpointPort{{
			Name:     ptr.To(servicePortName),
			Port:     ptr.To(int32(proxyPort)),
			Protocol: ptr.To(corev1.ProtocolTCP),
		}},
		Endpoints: endpoints,
	}

	for _, obj := range []client.Object{service, slice} {
		if err := controllerutil.SetOwnerReference(server, obj, scheme); err != nil {
			return nil, nil, fmt.Errorf("set owner reference on %T: %w", obj, err)
		}
	}
	return service, slice, nil
}

// EndpointSliceName returns the name of the controller-managed EndpointSlice
// backing the MCPServer's selector-less Service.
func EndpointSliceName(server *v1alpha1.MCPServer) string {
	name := server.Name + "-proxy"
	if len(name) > 253 {
		name = name[:253]
	}
	return name
}

// ProxyEndpoint returns the in-cluster URL MCP clients use in managed-proxy
// mode — the same Service contract as the kubernetes runtime.
func ProxyEndpoint(server *v1alpha1.MCPServer) string {
	port := server.Spec.Deployment.Port
	if port == 0 {
		port = 3000
	}
	return fmt.Sprintf("http://%s.%s.svc:%d%s", server.Name, server.Namespace, port, MCPPath)
}

// splitRouterURL extracts host and port from the atenet router URL.
func splitRouterURL(routerURL string) (string, uint16, error) {
	u, err := url.Parse(routerURL)
	if err != nil {
		return "", 0, fmt.Errorf("parse atenet router URL %q: %w", routerURL, err)
	}
	host := u.Hostname()
	if host == "" {
		return "", 0, fmt.Errorf("atenet router URL %q has no host", routerURL)
	}
	port := uint16(80)
	if u.Scheme == "https" {
		port = 443
	}
	if p := u.Port(); p != "" {
		parsed, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return "", 0, fmt.Errorf("parse atenet router port %q: %w", p, err)
		}
		port = uint16(parsed)
	}
	return host, port, nil
}

// mcpAppProtocol mirrors the kubernetes-runtime Service appProtocol behavior,
// including its env-var escape hatch.
func mcpAppProtocol() *string {
	if os.Getenv("DISABLE_KGATEWAY_MCP_APP_PROTOCOL") == "true" {
		return nil
	}
	proto := "kgateway.dev/mcp"
	return &proto
}
