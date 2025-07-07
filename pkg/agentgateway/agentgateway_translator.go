package agentgateway

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"kagent.dev/kmcp/api/v1alpha1"
	"sigs.k8s.io/yaml"
)

const (
	copyBinaryContainerImage = "ttl.sh/h1751392143:24h"
	everythingContainerImage = "docker.io/mcp/everything"
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
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{
						Name:    "copy-binary",
						Image:   copyBinaryContainerImage,
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
						Name:  "tool",
						Image: everythingContainerImage,
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
				},
			},
		},
	}, nil
}

func (t *agentGatewayTranslator) translateAgentGatewayService(server *v1alpha1.MCPServer) (*corev1.Service, error) {
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
				Port:     int32(server.Spec.Port),
				TargetPort: intstr.IntOrString{
					IntVal: int32(server.Spec.Port),
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
			SSE:     nil,
			Stdio:   nil,
			OpenAPI: nil,
		},
		//Filters: nil,
	}

	switch server.Spec.TransportType {
	case v1alpha1.TransportTypeStdio:
		if server.Spec.StdioTransport == nil {
			return nil, fmt.Errorf("StdioTransport must be specified for Stdio transport type")
		}
		mcpTarget.Spec.Stdio = &StdioTargetSpec{
			Cmd:  server.Spec.StdioTransport.Cmd,
			Args: server.Spec.StdioTransport.Args,
			Env:  server.Spec.StdioTransport.Env,
		}
	case v1alpha1.TransportTypeHTTP:
		if server.Spec.HTTPTransport == nil {
			return nil, fmt.Errorf("HTTPTransport must be specified for HTTP transport type")
		}
		mcpTarget.Spec.OpenAPI = &OpenAPITargetSpec{
			//Host:   "localhost",
			//Port:   server.Spec.HTTPTransport.Port,
			//Schema: nil,
		}
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", server.Spec.TransportType)
	}

	config := &LocalConfig{
		Binds: []LocalBind{
			{
				Port: server.Spec.Port,
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
