package transportadapter

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kagent-dev/kmcp/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
)

const (
	transportAdapterContainerImage = "ghcr.io/agentgateway/agentgateway:0.7.4-musl"
	busyboxImage                   = "busybox:1.37.0-musl"
)

// Translator is the interface for translating MCPServer objects to TransportAdapter objects.
type Translator interface {
	TranslateTransportAdapterOutputs(server *v1alpha1.MCPServer) ([]client.Object, error)
}

type transportAdapterTranslator struct {
	scheme *runtime.Scheme
}

func NewTransportAdapterTranslator(scheme *runtime.Scheme) Translator {
	return &transportAdapterTranslator{
		scheme: scheme,
	}
}

func (t *transportAdapterTranslator) TranslateTransportAdapterOutputs(
	server *v1alpha1.MCPServer,
) ([]client.Object, error) {
	serviceAccount, err := t.translateTransportAdapterServiceAccount(server)
	if err != nil {
		return nil, fmt.Errorf("failed to translate TransportAdapter service account: %w", err)
	}
	deployment, err := t.translateTransportAdapterDeployment(server)
	if err != nil {
		return nil, fmt.Errorf("failed to translate TransportAdapter deployment: %w", err)
	}
	service, err := t.translateTransportAdapterService(server)
	if err != nil {
		return nil, fmt.Errorf("failed to translate TransportAdapter service: %w", err)
	}
	configMap, err := t.translateTransportAdapterConfigMap(server)
	if err != nil {
		return nil, fmt.Errorf("failed to translate TransportAdapter config map: %w", err)
	}
	return []client.Object{
		serviceAccount,
		deployment,
		service,
		configMap,
	}, nil
}

func (t *transportAdapterTranslator) translateTransportAdapterDeployment(
	server *v1alpha1.MCPServer,
) (*appsv1.Deployment, error) {
	image := server.Spec.Deployment.Image
	if image == "" && server.Spec.Deployment.Cmd == "uvx" {
		image = "ghcr.io/astral-sh/uv:debian"
	}
	if image == "" && server.Spec.Deployment.Cmd == "npx" {
		image = "node:24-alpine3.21"
	}
	if image == "" {
		return nil, fmt.Errorf("image must be specified for MCPServer %s or the command must be 'uvx' or 'npx'", server.Name)
	}

	// Create environment variables from secrets for envFrom
	secretEnvFrom := t.createSecretEnvFrom(server.Spec.Deployment.SecretRefs)

	var template corev1.PodSpec
	switch server.Spec.TransportType {
	case v1alpha1.TransportTypeStdio:
		// copy the binary into the container when running with stdio
		initContainers := []corev1.Container{{
			Name:            "copy-binary",
			Image:           transportAdapterContainerImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{},
			Args: []string{
				"--copy-self",
				"/adapterbin/agentgateway",
			},
			VolumeMounts: []corev1.VolumeMount{{
				Name:      "binary",
				MountPath: "/adapterbin",
			}},
			SecurityContext: getSecurityContext(),
		}}

		template = corev1.PodSpec{
			ServiceAccountName: server.Name,
			InitContainers:     initContainers,
			Containers: []corev1.Container{{
				Name:            "mcp-server",
				Image:           image,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command: []string{
					"/adapterbin/agentgateway",
				},
				Args: []string{
					"-f",
					"/config/local.yaml",
				},
				Env:     convertEnvVars(server.Spec.Deployment.Env),
				EnvFrom: secretEnvFrom,
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "config",
						MountPath: "/config",
					},
					{
						Name:      "binary",
						MountPath: "/adapterbin",
					},
				},
				SecurityContext: getSecurityContext(),
			}},
			Volumes: []corev1.Volume{
				{
					Name: "config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: server.Name, // ConfigMap name matches the MCPServer name
							},
						},
					},
				},
				{
					Name: "binary",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{}, // EmptyDir for the binary
					},
				},
			},
		}
	case v1alpha1.TransportTypeHTTP:
		var cmd []string
		if server.Spec.Deployment.Cmd != "" {
			cmd = []string{server.Spec.Deployment.Cmd}
		}
		template = corev1.PodSpec{
			ServiceAccountName: server.Name,
			Containers: []corev1.Container{
				{
					Name:            "mcp-server",
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         cmd,
					Args:            server.Spec.Deployment.Args,
					Env:             convertEnvVars(server.Spec.Deployment.Env),
					EnvFrom:         secretEnvFrom,
					SecurityContext: getSecurityContext(),
				}},
			Volumes: []corev1.Volume{
				{
					Name: "config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: server.Name, // ConfigMap name matches the MCPServer name
							},
						},
					},
				},
			},
		}
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      server.Name,
			Namespace: server.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":     server.Name,
					"app.kubernetes.io/instance": server.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name":       server.Name,
						"app.kubernetes.io/instance":   server.Name,
						"app.kubernetes.io/managed-by": "kmcp",
					},
				},
				Spec: template,
			},
		},
	}

	// Set owner reference first
	if err := controllerutil.SetOwnerReference(server, deployment, t.scheme); err != nil {
		return nil, err
	}

	configYaml, err := t.translateTransportAdapterConfigAsYAML(server)
	if err != nil {
		return nil, err
	}
	// Add hash annotation based on MCPServer spec to initiate a restart on changes to the MCPServer spec
	t.addMCPServerConfigHashAnnotation(deployment, configYaml)

	return deployment, nil
}

// addMCPServerSpecHashAnnotation adds a hash annotation to the deployment's pod template
// based on the MCPServer config yaml. This ensures pod restarts when the MCPServer configuration changes.
func (t *transportAdapterTranslator) addMCPServerConfigHashAnnotation(
	deployment *appsv1.Deployment,
	mcpServerConfigYaml string,
) {
	hash := sha256.Sum256([]byte(mcpServerConfigYaml))
	truncatedHash := hex.EncodeToString(hash[:])[:8]
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kmcp.kagent.dev/mcpserver-config-hash"] = truncatedHash
}

func (t *transportAdapterTranslator) translateTransportAdapterServiceAccount(
	server *v1alpha1.MCPServer,
) (*corev1.ServiceAccount, error) {
	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      server.Name,
			Namespace: server.Namespace,
		},
	}
	return serviceAccount, controllerutil.SetOwnerReference(server, serviceAccount, t.scheme)
}

// createSecretEnvFrom creates envFrom references from secret references
func (t *transportAdapterTranslator) createSecretEnvFrom(
	secretRefs []corev1.LocalObjectReference,
) []corev1.EnvFromSource {
	envFrom := make([]corev1.EnvFromSource, 0, len(secretRefs))

	for _, secretRef := range secretRefs {
		// Skip empty secret references
		if secretRef.Name == "" {
			continue
		}

		envFrom = append(envFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretRef.Name,
				},
			},
		})
	}

	return envFrom
}

// getSecurityContext returns a SecurityContext that meets Pod Security Standards "restricted" policy
func getSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: &[]bool{false}[0],
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		RunAsNonRoot: &[]bool{true}[0],
		RunAsUser:    &[]int64{1000}[0],
		RunAsGroup:   &[]int64{1000}[0],
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func convertEnvVars(env map[string]string) []corev1.EnvVar {
	if env == nil {
		return nil
	}
	envVars := make([]corev1.EnvVar, 0, len(env))
	for key, value := range env {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}
	sort.Slice(envVars, func(i, j int) bool {
		return envVars[i].Name < envVars[j].Name
	})
	return envVars
}

func (t *transportAdapterTranslator) translateTransportAdapterService(server *v1alpha1.MCPServer) (*corev1.Service, error) {
	port := server.Spec.Deployment.Port
	if port == 0 {
		return nil, fmt.Errorf("deployment port must be specified for MCPServer %s", server.Name)
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      server.Name,
			Namespace: server.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:     "http",
				Protocol: "TCP",
				Port:     int32(port),
				TargetPort: intstr.IntOrString{
					IntVal: int32(port),
				},
			}},
			Selector: map[string]string{
				"app.kubernetes.io/name":     server.Name,
				"app.kubernetes.io/instance": server.Name,
			},
		},
	}

	return service, controllerutil.SetOwnerReference(server, service, t.scheme)
}

func (t *transportAdapterTranslator) translateTransportAdapterConfigAsYAML(server *v1alpha1.MCPServer) (string, error) {
	config, err := t.translateTransportAdapterConfig(server)
	if err != nil {
		return "", fmt.Errorf("failed to translate MCP server config: %w", err)
	}

	configYaml, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal MCP server config to YAML: %w", err)
	}

	return string(configYaml), nil
}

func (t *transportAdapterTranslator) translateTransportAdapterConfigMap(server *v1alpha1.MCPServer) (*corev1.ConfigMap, error) {
	configYaml, err := t.translateTransportAdapterConfigAsYAML(server)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCP server config to YAML: %w", err)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      server.Name,
			Namespace: server.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		Data: map[string]string{
			"local.yaml": configYaml,
		},
	}

	return configMap, controllerutil.SetOwnerReference(server, configMap, t.scheme)
}

func (t *transportAdapterTranslator) translateTransportAdapterConfig(server *v1alpha1.MCPServer) (*LocalConfig, error) {
	mcpTarget := MCPTarget{
		Name: server.Name,
	}

	port := server.Spec.Deployment.Port
	if port == 0 {
		return nil, fmt.Errorf("deployment port must be specified for MCPServer %s", server.Name)
	}

	switch server.Spec.TransportType {
	case v1alpha1.TransportTypeStdio:
		mcpTarget.Stdio = &StdioTargetSpec{
			Cmd:  server.Spec.Deployment.Cmd,
			Args: server.Spec.Deployment.Args,
			Env:  server.Spec.Deployment.Env,
		}
	case v1alpha1.TransportTypeHTTP:
		httpTransportConfig := server.Spec.HTTPTransport
		if httpTransportConfig == nil || httpTransportConfig.TargetPort == 0 {
			return nil, fmt.Errorf("HTTP transport requires a target port")
		}
		mcpTarget.SSE = &SSETargetSpec{
			Host: "localhost",
			Port: httpTransportConfig.TargetPort,
			Path: httpTransportConfig.TargetPath,
		}
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", server.Spec.TransportType)
	}

	config := &LocalConfig{
		Config: struct{}{},
		Binds: []LocalBind{
			{
				Port: port,
				Listeners: []LocalListener{
					{
						Name:     "default",
						Protocol: "HTTP",
						Routes: []LocalRoute{{
							RouteName: "mcp",
							Matches: []RouteMatch{
								{
									Path: PathMatch{
										PathPrefix: "/sse",
									},
								},
								{
									Path: PathMatch{
										PathPrefix: "/mcp",
									},
								},
							},
							Backends: []RouteBackend{{
								Weight: 100,
								MCP: &MCPBackend{
									Targets: []MCPTarget{mcpTarget},
								},
							}},
						}},
					},
				},
			},
		},
	}

	return config, nil
}
