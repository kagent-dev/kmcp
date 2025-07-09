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

// MCPServerSpec defines the desired state of MCPServer.
type MCPServerSpec struct {
	// Configuration to Deploy the MCP Server using a docker container
	Deployment MCPServerDeployment `json:"deployment"`

	// TransportType defines the type of mcp server being run
	// +kubebuilder:validation:Enum=stdio;http
	TransportType TransportType `json:"transportType,omitempty"`
}

// StdioTransport defines the configuration for a standard input/output transport.
type StdioTransport struct{}

// HTTPTransport defines the configuration for a Streamable HTTP transport.
type HTTPTransport struct{}

// MCPServerStatus defines the observed state of MCPServer.
type MCPServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

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
