package agentgateway

import (
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"kagent.dev/kmcp/api/v1alpha1"
	"sigs.k8s.io/yaml"
)

const (
	agentGatewayContainerImage = "ttl.sh/h1751392143:24h"
)

type AgentGatewayOutputs struct {
	// AgentGateway Deployment
	Deployment *appsv1.Deployment
	// AgentGateway Service
	Service *corev1.Service
	// AgentGateway Configmap
	ConfigMap *corev1.ConfigMap
}

type AgentGatewayTranslator interface {
	TranslateAgentGatewayOutputs(server *v1alpha1.MCPServer) (*AgentGatewayOutputs, error)
}

type agentGatewayTranslator struct {
}

func NewAgentGatewayTranslator() AgentGatewayTranslator {
	return &agentGatewayTranslator{}
}

func (t *agentGatewayTranslator) TranslateAgentGatewayOutputs(server *v1alpha1.MCPServer) (*AgentGatewayOutputs, error) {
	deployment, err := t.translateAgentGatewayDeployment(server)
	if err != nil {
		return nil, fmt.Errorf("failed to translate AgentGateway deployment: %w", err)
	}
	service, err := t.translateAgentGatewayService(server)
	if err != nil {
		return nil, fmt.Errorf("failed to translate AgentGateway service: %w", err)
	}
	configMap, err := t.translateAgentGatewayConfigMap(server)
	if err != nil {
		return nil, fmt.Errorf("failed to translate AgentGateway config map: %w", err)
	}
	return &AgentGatewayOutputs{
		Deployment: deployment,
		Service:    service,
		ConfigMap:  configMap,
	}, nil
}

func (t *agentGatewayTranslator) translateAgentGatewayDeployment(server *v1alpha1.MCPServer) (*appsv1.Deployment, error) {
	image := server.Spec.Deployment.Image
	if image == "" {
		return nil, fmt.Errorf("deployment image must be specified for MCPServer %s", server.Name)
	}

	var template corev1.PodSpec
	switch server.Spec.TransportType {
	case v1alpha1.TransportTypeStdio:
		// copy the binary into the container when running with stdio
		template = corev1.PodSpec{
			InitContainers: []corev1.Container{{
				Name:    "copy-binary",
				Image:   agentGatewayContainerImage,
				Command: []string{"sh"},
				Args: []string{
					"-c",
					"cp /usr/bin/agentgateway /agentbin/agentgateway",
				},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "binary",
					MountPath: "/agentbin",
				}},
			}},
			Containers: []corev1.Container{{
				Name:  "mcp-server",
				Image: image,
				Command: []string{
					"sh",
					"-c",
					"/agentbin/agentgateway -f /config/local.yaml",
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "config",
						MountPath: "/config",
					},
					{
						Name:      "binary",
						MountPath: "/agentbin",
					},
				},
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
		// run the gateway as a sidecar when running with HTTP transport
		template = corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "agent-gateway",
					Image:   agentGatewayContainerImage,
					Command: []string{"sh"},
					Args: []string{
						"-c",
						"/agentbin/agentgateway -f /config/local.yaml",
					},
					VolumeMounts: []corev1.VolumeMount{{
						Name:      "config",
						MountPath: "/config",
					}},
				},
				{
					Name:    "mcp-server",
					Image:   image,
					Command: []string{server.Spec.Deployment.Cmd},
					Args:    server.Spec.Deployment.Args,
					Env:     convertEnvVars(server.Spec.Deployment.Env),
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

	return &appsv1.Deployment{
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
	}, nil
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

func (t *agentGatewayTranslator) translateAgentGatewayService(server *v1alpha1.MCPServer) (*corev1.Service, error) {
	port := server.Spec.Deployment.Port
	if port == 0 {
		return nil, fmt.Errorf("deployment port must be specified for MCPServer %s", server.Name)
	}
	return &corev1.Service{
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
	}, nil
}

func (t *agentGatewayTranslator) translateAgentGatewayConfigMap(server *v1alpha1.MCPServer) (*corev1.ConfigMap, error) {
	config, err := t.translateAgentGatewayConfig(server)
	if err != nil {
		return nil, fmt.Errorf("failed to translate MCP server config: %w", err)
	}

	if config == nil {
		return nil, nil // No config needed
	}

	configYaml, err := yaml.Marshal(config)
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
			"local.yaml": string(configYaml), // Assuming ToYAML() is a method that converts LocalConfig to YAML
		},
	}

	return configMap, nil
}

func (t *agentGatewayTranslator) translateAgentGatewayConfig(server *v1alpha1.MCPServer) (*LocalConfig, error) {
	if server.Spec.TransportType != v1alpha1.TransportTypeStdio {
		return nil, nil // Only Stdio transport is supported for now
	}

	mcpTarget := MCPTarget{
		Name: server.Name,
		Spec: MCPTargetSpec{
			SSE:   nil,
			Stdio: nil,
		},
		//Filters: nil,
	}

	switch server.Spec.TransportType {
	case v1alpha1.TransportTypeStdio:
		mcpTarget.Spec.Stdio = &StdioTargetSpec{
			Cmd:  server.Spec.Deployment.Cmd,
			Args: server.Spec.Deployment.Args,
			Env:  server.Spec.Deployment.Env,
		}
	case v1alpha1.TransportTypeHTTP:
		mcpTarget.Spec.SSE = &SSETargetSpec{
			Host: "localhost",
			Port: uint32(server.Spec.Deployment.Port),
			//Path TODO do we need this
		}
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", server.Spec.TransportType)
	}

	port := server.Spec.Deployment.Port
	if port == 0 {
		return nil, fmt.Errorf("deployment port must be specified for MCPServer %s", server.Name)
	}

	config := &LocalConfig{
		Config: struct{}{},
		Binds: []LocalBind{
			{
				Port: port,
				Listeners: []LocalListener{
					{
						Name: "default",
						//GatewayName: nil,
						//Hostname:    nil,
						Protocol: "HTTP",
						//TLS:         nil,
						Routes: []LocalRoute{{
							RouteName: "mcp",
							//RuleName:  "",
							//Hostnames: nil,
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
							//Policies: nil,
							Backends: []RouteBackend{{
								Weight: 100,
								Backend: Backend{
									MCP: &MCPBackend{
										Name:    mcpTarget.Name,
										Targets: []MCPTarget{mcpTarget},
									},
								},
								//Filters: nil, TODO
							}},
						}},
						//TCPRoutes:   nil,
					},
				},
			},
		},
	}

	return config, nil
}
