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

	// MCPServerConditionActorTemplateReady indicates that the generated
	// ate.dev ActorTemplate has baked its golden snapshot and is ready to
	// instantiate actors. Only set when runtime is substrate.
	//
	// Possible reasons for this condition to be True are:
	//
	// * "Ready"
	//
	// Possible reasons for this condition to be False are:
	//
	// * "ActorTemplatePending"
	// * "ActorTemplateFailed"
	MCPServerConditionActorTemplateReady MCPServerConditionType = "ActorTemplateReady"

	// MCPServerConditionActorReady indicates that the substrate actor backing
	// this MCPServer exists and is routable through the atenet router. A
	// suspended actor is considered ready: the router resumes it on demand
	// when a request arrives. Only set when runtime is substrate.
	//
	// Possible reasons for this condition to be True are:
	//
	// * "ActorRunning"
	// * "ActorResuming"
	// * "ActorSuspended"
	//
	// Possible reasons for this condition to be False are:
	//
	// * "ActorNotCreated"
	// * "ActorSuspending"
	// * "ActorCreateFailed"
	MCPServerConditionActorReady MCPServerConditionType = "ActorReady"
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
	MCPServerReasonAvailable    MCPServerConditionReason = "Available"
	MCPServerReasonNotAvailable MCPServerConditionReason = "NotAvailable"

	// Substrate runtime condition reasons
	MCPServerReasonUnsupportedField     MCPServerConditionReason = "UnsupportedField"
	MCPServerReasonImageNotPinned       MCPServerConditionReason = "ImageNotPinned"
	MCPServerReasonWorkerPoolNotFound   MCPServerConditionReason = "WorkerPoolNotFound"
	MCPServerReasonSecretNotFound       MCPServerConditionReason = "SecretNotFound"
	MCPServerReasonActorTemplatePending MCPServerConditionReason = "ActorTemplatePending"
	MCPServerReasonActorTemplateFailed  MCPServerConditionReason = "ActorTemplateFailed"
	MCPServerReasonActorNotCreated      MCPServerConditionReason = "ActorNotCreated"
	MCPServerReasonActorCreateFailed    MCPServerConditionReason = "ActorCreateFailed"
	MCPServerReasonActorRunning         MCPServerConditionReason = "ActorRunning"
	MCPServerReasonActorResuming        MCPServerConditionReason = "ActorResuming"
	MCPServerReasonActorSuspended       MCPServerConditionReason = "ActorSuspended"
	MCPServerReasonActorSuspending      MCPServerConditionReason = "ActorSuspending"
	MCPServerReasonActorDeleting        MCPServerConditionReason = "ActorDeleting"
	MCPServerReasonDeleteTimeout        MCPServerConditionReason = "DeleteTimeout"
)

// MCPServerRuntime selects the provisioning stack for the MCP server workload.
// +kubebuilder:validation:Enum=kubernetes;substrate
type MCPServerRuntime string

const (
	// MCPServerRuntimeKubernetes provisions the MCP server as a standard
	// Kubernetes Deployment + Service. This is the default.
	MCPServerRuntimeKubernetes MCPServerRuntime = "kubernetes"

	// MCPServerRuntimeSubstrate provisions the MCP server as a serverless
	// Agent Substrate actor (scale-to-zero with resume-on-request).
	MCPServerRuntimeSubstrate MCPServerRuntime = "substrate"
)

// MCPServerSpec defines the desired state of MCPServer.
// +kubebuilder:validation:XValidation:rule="!has(self.substrate) || (has(self.runtime) && self.runtime == 'substrate')",message="spec.substrate may only be set when runtime is substrate"
// +kubebuilder:validation:XValidation:rule="!has(self.runtime) || self.runtime != 'substrate' || has(self.substrate)",message="spec.substrate is required when runtime is substrate"
// +kubebuilder:validation:XValidation:rule="!has(self.runtime) || self.runtime != 'substrate' || (has(self.deployment.image) && self.deployment.image.contains('@'))",message="spec.deployment.image must be set and digest-pinned (e.g. @sha256:...) when runtime is substrate"
type MCPServerSpec struct {
	// Configuration to Deploy the MCP Server using a docker container
	Deployment MCPServerDeployment `json:"deployment"`

	// Runtime selects the provisioning stack for the MCP server workload.
	// Defaults to kubernetes when unset.
	// +optional
	// +kubebuilder:default=kubernetes
	Runtime MCPServerRuntime `json:"runtime,omitempty"`

	// Substrate configures the Agent Substrate runtime.
	// Required when runtime is substrate, and forbidden otherwise.
	// +optional
	Substrate *SubstrateSpec `json:"substrate,omitempty"`

	// TransportType defines the type of mcp server being run
	// +kubebuilder:validation:Enum=stdio;http
	TransportType TransportType `json:"transportType,omitempty"`

	// StdioTransport defines the configuration for a standard input/output transport.
	StdioTransport *StdioTransport `json:"stdioTransport,omitempty"`

	// HTTPTransport defines the configuration for a Streamable HTTP transport.
	HTTPTransport *HTTPTransport `json:"httpTransport,omitempty"`

	// Timeout defines the default connection timeout for clients connecting
	// to this MCP server. MCP servers deployed via the MCPServer CRD use a
	// sidecar gateway that spawns a new stdio process (e.g. via uvx/npx)
	// for each session. Process startup can take 2-8 seconds depending on
	// package cache state, which may exceed the default timeout used by some
	// clients. This value is propagated to the generated RemoteMCPServer
	// resources when they do not specify an explicit timeout.
	// +optional
	// +kubebuilder:default="30s"
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

// SubstrateSnapshotsConfig points at an object-storage prefix for actor
// memory snapshots.
type SubstrateSnapshotsConfig struct {
	// Location is the object-storage prefix to store actor snapshots in,
	// e.g. gs://my-bucket/path or s3://my-bucket/path.
	// +required
	// +kubebuilder:validation:Pattern=`^(gs|s3)://`
	Location string `json:"location"`
}

// SubstrateSpec configures the Agent Substrate runtime
// (generated ate.dev ActorTemplate + actor).
type SubstrateSpec struct {
	// WorkerPoolRef references an existing ate.dev WorkerPool in the
	// MCPServer's namespace that provides compute capacity for the actor.
	// When unset, the controller uses its configured default WorkerPool.
	// +optional
	WorkerPoolRef *corev1.LocalObjectReference `json:"workerPoolRef,omitempty"`

	// SnapshotsConfig configures where actor memory snapshots are stored.
	// Defaults to <controller default prefix>/<namespace>/<name> when unset.
	// +optional
	SnapshotsConfig *SubstrateSnapshotsConfig `json:"snapshotsConfig,omitempty"`
}

// StdioTransport defines the configuration for a standard input/output transport.
type StdioTransport struct{}

// HTTPTransport defines the configuration for a Streamable HTTP transport.
type HTTPTransport struct {
	// target port is the HTTP port that serves the MCP server.over HTTP
	TargetPort uint32 `json:"targetPort,omitempty"`

	// the target path where MCP is served
	TargetPath string `json:"path,omitempty"`

	// TLS defines the TLS configuration for HTTPS access to the MCP server.
	// +optional
	TLS *HTTPTransportTLS `json:"tls,omitempty"`
}

// HTTPTransportTLS defines the TLS configuration for HTTP transport.
type HTTPTransportTLS struct {
	// SecretRef is a reference to a Kubernetes Secret containing
	// the client certificate (tls.crt), key (tls.key), and optionally
	// the CA certificate (ca.crt) for mTLS authentication.
	// The Secret must be in the same namespace as the MCPServer.
	// +optional
	SecretRef string `json:"secretRef,omitempty"`

	// InsecureSkipVerify disables SSL certificate verification.
	// WARNING: This should ONLY be used in development/testing environments.
	// Production deployments MUST use proper certificates.
	// +optional
	// +kubebuilder:default=false
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
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

	// Substrate publishes routing coordinates for the substrate actor backing
	// this MCPServer. Only set when runtime is substrate. This is the contract
	// consumed by any ingress in front of substrate (kmcp's optional ingress
	// proxy, or an external proxy such as kagent's API server).
	// +optional
	Substrate *MCPServerSubstrateStatus `json:"substrate,omitempty"`
}

// MCPServerSubstrateStatus publishes routing coordinates for the substrate
// actor backing an MCPServer.
type MCPServerSubstrateStatus struct {
	// ActorID is the substrate actor id (a DNS-1123 label),
	// e.g. mcp-<namespace>-<name>.
	// +optional
	ActorID string `json:"actorID,omitempty"`

	// ActorHost is the Host/:authority header value the atenet router uses to
	// resume-then-route requests to the actor,
	// e.g. <actorID>.actors.resources.substrate.ate.dev.
	// +optional
	ActorHost string `json:"actorHost,omitempty"`

	// RouterURL is the atenet router base URL requests must be sent to with
	// Host set to ActorHost, e.g. http://atenet-router.ate-system.svc:80.
	// +optional
	RouterURL string `json:"routerURL,omitempty"`

	// MCPPath is the HTTP path MCP is served on through the router.
	// +optional
	MCPPath string `json:"mcpPath,omitempty"`

	// ActorTemplateRef is the name of the generated ate.dev ActorTemplate in
	// the MCPServer's namespace.
	// +optional
	ActorTemplateRef string `json:"actorTemplateRef,omitempty"`

	// ProxyEndpoint is the in-cluster URL of the optional ingress proxy
	// Service, e.g. http://<name>.<namespace>.svc:<port>/mcp.
	// Empty when the ingress proxy is disabled.
	// +optional
	ProxyEndpoint string `json:"proxyEndpoint,omitempty"`
}

// MCPServerDeployment
// +kubebuilder:validation:XValidation:rule="!(has(self.serviceAccount) && has(self.serviceAccountName))",message="serviceAccount and serviceAccountName are mutually exclusive"
type MCPServerDeployment struct {
	// Image defines the container image to to deploy the MCP server.
	// +optional
	Image string `json:"image,omitempty"`

	// ImagePullPolicy defines the pull policy for the container image.
	// +optional
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Port defines the port on which the MCP server will listen.
	// +optional
	// +kubebuilder:default=3000
	Port uint16 `json:"port,omitempty"`

	// Cmd defines the command to run in the container to start the mcp server.
	// +optional
	Cmd string `json:"cmd,omitempty"`

	// Args defines the arguments to pass to the command.
	// +optional
	Args []string `json:"args,omitempty"`

	// Env defines the environment variables to set in the container.
	// +optional
	Env map[string]string `json:"env,omitempty"`

	// SecretRefs defines the list of Kubernetes secrets to reference.
	// These secrets will be mounted as volumes to the MCP server container.
	// +optional
	SecretRefs []corev1.LocalObjectReference `json:"secretRefs,omitempty"`

	// ConfigMapRefs defines the list of Kubernetes configmaps to reference.
	// These configmaps will be mounted as volumes to the MCP server container.
	// +optional
	ConfigMapRefs []corev1.LocalObjectReference `json:"configMapRefs,omitempty"`

	// VolumeMounts defines the list of volume mounts for the MCP server container.
	// This allows for more flexible volume mounting configurations.
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// Volumes defines the list of volumes that can be mounted by containers.
	// This allows for custom volume configurations beyond just secrets and configmaps.
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// InitContainer defines the configuration for the init container that copies
	// the transport adapter binary. This is used for stdio transport type.
	// +optional
	InitContainer *InitContainerConfig `json:"initContainer,omitempty"`

	// ServiceAccount defines the configuration for the ServiceAccount to be created.
	// +optional
	ServiceAccount *ServiceAccountConfig `json:"serviceAccount,omitempty"`

	// ServiceAccountName is the name of an existing ServiceAccount to use.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Sidecars defines additional containers to run alongside the MCP server container.
	// These containers will share the same pod and can share volumes with the main container.
	// +optional
	Sidecars []corev1.Container `json:"sidecars,omitempty"`

	// Labels defines additional labels to add to the pod template.
	// These labels will be merged with the default labels.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations defines additional annotations to add to the pod template.
	// These annotations will be merged with the default annotations.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Resources defines the compute resource requirements for the main MCP server container.
	// Use this to specify CPU and memory requests and limits.
	// Example:
	//   resources:
	//     requests:
	//       cpu: "100m"
	//       memory: "128Mi"
	//     limits:
	//       cpu: "500m"
	//       memory: "512Mi"
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// SecurityContext defines the security context for the main MCP server container.
	// Use this to configure container-level security settings such as:
	// - runAsUser/runAsGroup: Run as specific user/group
	// - runAsNonRoot: Ensure container doesn't run as root
	// - readOnlyRootFilesystem: Make root filesystem read-only
	// - allowPrivilegeEscalation: Prevent privilege escalation
	// - capabilities: Add or drop Linux capabilities
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`

	// PodSecurityContext defines the security context for the entire pod.
	// Use this to configure pod-level security settings such as:
	// - runAsUser/runAsGroup: Default user/group for all containers
	// - fsGroup: Group ownership of mounted volumes
	// - seccompProfile: Seccomp profile for the pod
	// - sysctls: Kernel parameters to set
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// Tolerations defines the tolerations for the pod.
	// Use this to schedule pods on nodes with matching taints.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Affinity defines the affinity rules for the pod.
	// Use this to control pod placement based on node labels, pod labels,
	// or other scheduling constraints.
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// NodeSelector defines the node selector for the pod.
	// Use this to constrain pods to nodes with specific labels.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Replicas defines the number of desired pod replicas.
	// Defaults to 1 if not specified.
	// +optional
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas,omitempty"`

	// ImagePullSecrets defines the list of secrets to use for pulling container images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// InitContainerConfig defines the configuration for the init container.
type InitContainerConfig struct {
	// Image defines the full image reference for the init container.
	// If specified, this overrides the default transport adapter image.
	// Example: "myregistry.com/agentgateway/agentgateway:0.9.0-musl"
	// +optional
	Image string `json:"image,omitempty"`

	// ImagePullPolicy defines the pull policy for the init container image.
	// +optional
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Resources defines the compute resource requirements for the init container.
	// Use this to specify CPU and memory requests and limits for the init container.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// SecurityContext defines the security context for the init container.
	// If not specified, the main container's security context will be used.
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
}

// ServiceAccountConfig defines the configuration for the ServiceAccount.
type ServiceAccountConfig struct {
	// Annotations to add to the ServiceAccount.
	// This is useful for configuring AWS IRSA (IAM Roles for Service Accounts)
	// or other cloud provider integrations.
	// Example: {"eks.amazonaws.com/role-arn": "arn:aws:iam::123456789012:role/my-role"}
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels to add to the ServiceAccount.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=mcps;mcp
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Runtime",type="string",JSONPath=".spec.runtime"
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
