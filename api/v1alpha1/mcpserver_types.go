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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MCPServerTransportType defines the type of transport for the MCP server.
type TransportType string

const (
	// TransportTypeStdio indicates that the MCP server uses standard input/output for communication.
	TransportTypeStdio TransportType = "stdio"

	// TransportTypeHTTP indicates that the MCP server uses Streamable HTTP for communication.
	TransportTypeHTTP TransportType = "http"
)

// MCPServerConditionType represents the condition types for MCPServer status.
type MCPServerConditionType string

const (
	// MCPServerConditionAccepted indicates that the MCPServer has been accepted for processing.
	// This condition indicates that the MCPServer configuration is syntactically and semantically valid,
	// and the controller can generate some configuration for the underlying infrastructure.
	//
	// Possible reasons for this condition to be True are:
	//
	// * "Accepted"
	//
	// Possible reasons for this condition to be False are:
	//
	// * "InvalidConfig"
	// * "UnsupportedTransport"
	//
	// Controllers may raise this condition with other reasons,
	// but should prefer to use the reasons listed above to improve
	// interoperability.
	MCPServerConditionAccepted MCPServerConditionType = "Accepted"

	// MCPServerConditionResolvedRefs indicates whether the controller was able to
	// resolve all the object references for the MCPServer.
	//
	// Possible reasons for this condition to be True are:
	//
	// * "ResolvedRefs"
	//
	// Possible reasons for this condition to be False are:
	//
	// * "ImageNotFound"
	//
	// Controllers may raise this condition with other reasons,
	// but should prefer to use the reasons listed above to improve
	// interoperability.
	MCPServerConditionResolvedRefs MCPServerConditionType = "ResolvedRefs"

	// MCPServerConditionProgrammed indicates that the controller has successfully
	// programmed the underlying infrastructure with the MCPServer configuration.
	// This means that all required Kubernetes resources (Deployment, Service, ConfigMap)
	// have been created and configured.
	//
	// Possible reasons for this condition to be True are:
	//
	// * "Programmed"
	//
	// Possible reasons for this condition to be False are:
	//
	// * "DeploymentFailed"
	// * "ServiceFailed"
	// * "ConfigMapFailed"
	//
	// Controllers may raise this condition with other reasons,
	// but should prefer to use the reasons listed above to improve
	// interoperability.
	MCPServerConditionProgrammed MCPServerConditionType = "Programmed"

	// MCPServerConditionReady indicates that the MCPServer is ready to serve traffic.
	// This condition indicates that the underlying Deployment has running pods
	// that are ready to accept connections.
	//
	// Possible reasons for this condition to be True are:
	//
	// * "Ready"
	//
	// Possible reasons for this condition to be False are:
	//
	// * "PodsNotReady"
	//
	// Controllers may raise this condition with other reasons,
	// but should prefer to use the reasons listed above to improve
	// interoperability.
	MCPServerConditionReady MCPServerConditionType = "Ready"
)

// MCPServerConditionReason represents the reasons for MCPServer conditions.
type MCPServerConditionReason string

const (
	// Accepted condition reasons
	MCPServerReasonAccepted             MCPServerConditionReason = "Accepted"
	MCPServerReasonInvalidConfig        MCPServerConditionReason = "InvalidConfig"
	MCPServerReasonUnsupportedTransport MCPServerConditionReason = "UnsupportedTransport"

	// ResolvedRefs condition reasons
	MCPServerReasonResolvedRefs  MCPServerConditionReason = "ResolvedRefs"
	MCPServerReasonImageNotFound MCPServerConditionReason = "ImageNotFound"

	// Programmed condition reasons
	MCPServerReasonProgrammed       MCPServerConditionReason = "Programmed"
	MCPServerReasonDeploymentFailed MCPServerConditionReason = "DeploymentFailed"
	MCPServerReasonServiceFailed    MCPServerConditionReason = "ServiceFailed"
	MCPServerReasonConfigMapFailed  MCPServerConditionReason = "ConfigMapFailed"

	// Ready condition reasons
	MCPServerReasonReady        MCPServerConditionReason = "Ready"
	MCPServerReasonPodsNotReady MCPServerConditionReason = "PodsNotReady"
)

// MCPServerSpec defines the desired state of MCPServer.
type MCPServerSpec struct {
	// Configuration to Deploy the MCP Server using a docker container
	Deployment MCPServerDeployment `json:"deployment"`

	// TransportType defines the type of mcp server being run
	// +kubebuilder:validation:Enum=stdio;http
	TransportType TransportType `json:"transportType,omitempty"`

	// StdioTransport defines the configuration for a standard input/output transport.
	StdioTransport *StdioTransport `json:"stdioTransport,omitempty"`

	// HTTPTransport defines the configuration for a Streamable HTTP transport.
	HTTPTransport *HTTPTransport `json:"httpTransport,omitempty"`

	// Authn defines the authentication configuration for the MCP server.
	// This field is optional and can be used to configure JWT authentication.
	// If not specified, the MCP server will not require authentication.
	// +optional
	Authn *MCPServerAuthentication `json:"authn,omitempty"`

	// Authz defines the authorization rule configuration for the MCP server.
	// This field is optional and can be used to configure authorization rules
	// for access to the MCP server and specific tools. If not specified, the MCP server will not enforce
	// any authorization rules.
	// +optional
	Authz *MCPServerAuthorization `json:"authz,omitempty"`

	// RouteFilter defines route filtering configuration for the MCP server.
	// Currently only supports CORS filtering.
	// +optional
	RouteFilter *RouteFilter `json:"routeFilter,omitempty" yaml:"routeFilter,omitempty"`
}

// StdioTransport defines the configuration for a standard input/output transport.
type StdioTransport struct{}

// HTTPTransport defines the configuration for a Streamable HTTP transport.
type HTTPTransport struct {
	// target port is the HTTP port that serves the MCP server.over HTTP
	TargetPort uint32 `json:"targetPort,omitempty"`

	// the target path where MCP is served
	TargetPath string `json:"path,omitempty"`
}

// MCPServerStatus defines the observed state of MCPServer.
type MCPServerStatus struct {
	// Conditions describe the current conditions of the MCPServer.
	// Implementations should prefer to express MCPServer conditions
	// using the `MCPServerConditionType` and `MCPServerConditionReason`
	// constants so that operators and tools can converge on a common
	// vocabulary to describe MCPServer state.
	//
	// Known condition types are:
	//
	// * "Accepted"
	// * "ResolvedRefs"
	// * "Programmed"
	// * "Ready"
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the most recent generation observed for this MCPServer.
	// It corresponds to the MCPServer's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// MCPServerDeployment
type MCPServerDeployment struct {
	// Image defines the container image to to deploy the MCP server.
	Image string `json:"image,omitempty"`

	// Port defines the port on which the MCP server will listen.
	Port uint16 `json:"port,omitempty"`

	// Cmd defines the command to run in the container to start the mcp server.
	Cmd string `json:"cmd,omitempty"`

	// Args defines the arguments to pass to the command.
	Args []string `json:"args,omitempty"`

	// Env defines the environment variables to set in the container.
	Env map[string]string `json:"env,omitempty"`

	// SecretRefs defines the list of Kubernetes secrets to reference.
	// These secrets will be mounted as volumes to the MCP server container.
	// +optional
	SecretRefs []corev1.ObjectReference `json:"secretRefs,omitempty"`
}

// MCPServerAuthentication defines the authentication configuration for the MCP server.
type MCPServerAuthentication struct {
	// JWT defines the JWT authentication configuration.
	JWT *MCPServerJWTAuthentication `json:"jwt,omitempty"`
}

// MCPServerJWTAuthentication defines the JWT authentication configuration for the MCP server.
type MCPServerJWTAuthentication struct {
	// Issuer is the JWT issuer URL.
	Issuer string `json:"issuer,omitempty"`

	// Audiences is a list of audiences that the JWT must match.
	Audiences []string `json:"audiences,omitempty"`

	// JWKS references a secret containing the JSON Web Key Set.
	// The secret must contain a key with the JWKS content.
	JWKS *corev1.SecretKeySelector `json:"jwks,omitempty"`
}

// MCPServerAuthorization defines the authorization configuration for the MCP server.
type MCPServerAuthorization struct {
	// Server defines the configuration for the MCP authorization server that protects the MCP server.
	// Setting this field will configure agentgateway to use the authorization server
	// to protect the MCP server and its resources as well as adapt traffic to the MCP client to comply with the
	// MCP authorization spec before forwarding traffic to the MCP client.
	// +optional
	Server *MCPAuthorizationServer `json:"server,omitempty"`

	// CELAuthorization defines the CEL-based authorization configuration for the MCP server.
	CEL *MCPServerCELAuthorization `json:"cel,omitempty"`
}

// MCPServerCELAuthorization defines the authorization configuration for the MCP server using CEL rules.
type MCPServerCELAuthorization struct {
	// Rules are a list of CEL rules for authorizing client mcp requests.
	Rules []string `json:"rules" yaml:"rules"`
}

// MCPAuthorizationServer represents the configuration for the MCP authorization server
type MCPAuthorizationServer struct {
	Issuer           string                    `json:"issuer" yaml:"issuer"`
	Audience         string                    `json:"audience" yaml:"audience"`
	JwksURL          string                    `json:"jwksUrl" yaml:"jwksUrl"`
	Provider         *MCPClientProvider        `json:"provider,omitempty" yaml:"provider,omitempty"`
	ResourceMetadata MCPClientResourceMetadata `json:"resourceMetadata" yaml:"resourceMetadata"`
}

// MCPClientProvider represents the support identity providers currently only keycloak is supported
type MCPClientProvider struct {
	Keycloak KeycloakProvider `json:"keycloak,omitempty" yaml:"keycloak,omitempty"`
}

type KeycloakProvider struct {
	Realm string `json:"realm" yaml:"realm"`
}

// CORS defines CORS configuration for the MCP server
type CORS struct {
	// AllowHeaders is a list of HTTP headers that can be used when making the actual request
	// +optional
	AllowHeaders []string `json:"allowHeaders,omitempty" yaml:"allowHeaders,omitempty"`
	// AllowOrigins is a list of origins that are allowed to make requests
	// +optional
	AllowOrigins []string `json:"allowOrigins,omitempty" yaml:"allowOrigins,omitempty"`
}

// RouteFilter defines route filtering configuration for the MCP server
// Only CORS filtering is currently supported
type RouteFilter struct {
	// CORS defines CORS configuration for the route
	// +optional
	CORS *CORS `json:"cors,omitempty" yaml:"cors,omitempty"`
}

// MCPClientResourceMetadata represents resource metadata for MCP client authentication
type MCPClientResourceMetadata struct {
	// BaseURL denotes the protected base url of the protected resource ie: http://localhost:3000
	BaseUrl string `json:"baseUrl" yaml:"resource"`
	// Scopes supported by this resource
	// +optional
	ScopesSupported []string `json:"scopesSupported,omitempty" yaml:"scopesSupported,omitempty"`
	// Bearer methods supported by this resource
	// +optional
	BearerMethodsSupported []string `json:"bearerMethodsSupported,omitempty" yaml:"bearerMethodsSupported,omitempty"`
	// Additional resource metadata fields
	// +optional
	AdditionalFields map[string]string `json:"additionalFields,omitempty" yaml:"additionalFields,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=mcps;mcp
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:categories=kagent

// MCPServer is the Schema for the mcpservers API.
type MCPServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MCPServerSpec   `json:"spec,omitempty"`
	Status MCPServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MCPServerList contains a list of MCPServer.
type MCPServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MCPServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MCPServer{}, &MCPServerList{})
}
