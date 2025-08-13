package agentgateway

import (
	"context"
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
	agentGatewayContainerImage = "ghcr.io/agentgateway/agentgateway:0.7.4-musl"
)

type Outputs struct {
	// AgentGateway Deployment
	Deployment *appsv1.Deployment
	// AgentGateway Service
	Service *corev1.Service
	// AgentGateway Configmap
	ConfigMap *corev1.ConfigMap
}

// Translator is the interface for translating MCPServer objects to AgentGateway objects.
type Translator interface {
	TranslateAgentGatewayOutputs(ctx context.Context, server *v1alpha1.MCPServer) (*Outputs, error)
}

// agentGatewayTranslator is the implementation of the Translator interface.
type agentGatewayTranslator struct {
	scheme *runtime.Scheme
	client client.Client
}

// NewAgentGatewayTranslator creates a new instance of the agentGatewayTranslator.
func NewAgentGatewayTranslator(scheme *runtime.Scheme, client client.Client) Translator {
	return &agentGatewayTranslator{
		scheme: scheme,
		client: client,
	}
}

// TranslateAgentGatewayOutputs translates an MCPServer object to AgentGateway objects.
func (t *agentGatewayTranslator) TranslateAgentGatewayOutputs(
	ctx context.Context,
	server *v1alpha1.MCPServer,
) (*Outputs, error) {
	deployment, err := t.translateAgentGatewayDeployment(server)
	if err != nil {
		return nil, fmt.Errorf("failed to translate AgentGateway deployment: %w", err)
	}
	service, err := t.translateAgentGatewayService(server)
	if err != nil {
		return nil, fmt.Errorf("failed to translate AgentGateway service: %w", err)
	}
	configMap, err := t.translateAgentGatewayConfigMap(ctx, server)
	if err != nil {
		return nil, fmt.Errorf("failed to translate AgentGateway config map: %w", err)
	}
	return &Outputs{
		Deployment: deployment,
		Service:    service,
		ConfigMap:  configMap,
	}, nil
}

func (t *agentGatewayTranslator) translateAgentGatewayDeployment(
	server *v1alpha1.MCPServer,
) (*appsv1.Deployment, error) {
	image := server.Spec.Deployment.Image
	if image == "" {
		return nil, fmt.Errorf("deployment image must be specified for MCPServer %s", server.Name)
	}

	// Create environment variables from secrets for envFrom
	secretEnvFrom := t.createSecretEnvFrom(server.Spec.Deployment.SecretRefs)

	var template corev1.PodSpec
	switch server.Spec.TransportType {
	case v1alpha1.TransportTypeStdio:
		// copy the binary into the container when running with stdio
		template = corev1.PodSpec{
			InitContainers: []corev1.Container{{
				Name:            "copy-binary",
				Image:           agentGatewayContainerImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{},
				Args: []string{
					"--copy-self",
					"/agentbin/agentgateway",
				},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "binary",
					MountPath: "/agentbin",
				}},
				SecurityContext: getSecurityContext(),
			}},
			Containers: []corev1.Container{{
				Name:            "mcp-server",
				Image:           image,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command: []string{
					"/agentbin/agentgateway",
				},
				Args: []string{
					"-f",
					"/config/local.yaml",
				},
				EnvFrom: secretEnvFrom,
				VolumeMounts: func() []corev1.VolumeMount {
					mounts := []corev1.VolumeMount{
						{
							Name:      "config",
							MountPath: "/config",
						},
						{
							Name:      "binary",
							MountPath: "/agentbin",
						},
					}
					// Add JWKS secret mount if file-based JWT authentication is configured
					if isFileBasedJWTAuth(server) {
						mounts = append(mounts, corev1.VolumeMount{
							Name:      "jwks",
							MountPath: "/jwks",
						})
					}
					return mounts
				}(),
				SecurityContext: getSecurityContext(),
			}},
			Volumes: func() []corev1.Volume {
				volumes := []corev1.Volume{
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
				}
				// Add JWKS secret volume if file-based JWT authentication is configured
				if isFileBasedJWTAuth(server) {
					volumes = append(volumes, corev1.Volume{
						Name: "jwks",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: server.Spec.Authn.JWT.JWKS.Name,
							},
						},
					})
				}
				return volumes
			}(),
		}
	case v1alpha1.TransportTypeHTTP:
		// run the gateway as a sidecar when running with HTTP transport
		var cmd []string
		if server.Spec.Deployment.Cmd != "" {
			cmd = []string{server.Spec.Deployment.Cmd}
		}
		template = corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "agent-gateway",
					Image:           agentGatewayContainerImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{},
					Args: []string{
						"--copy-self",
						"/agentbin/agentgateway",
					},
					VolumeMounts: func() []corev1.VolumeMount {
						mounts := []corev1.VolumeMount{{
							Name:      "config",
							MountPath: "/config",
						}}
						// Add JWKS secret mount if file-based JWT authentication is configured
						if isFileBasedJWTAuth(server) {
							mounts = append(mounts, corev1.VolumeMount{
								Name:      "jwks",
								MountPath: "/jwks",
							})
						}
						return mounts
					}(),
					SecurityContext: getSecurityContext(),
				},
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
			Volumes: func() []corev1.Volume {
				volumes := []corev1.Volume{
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
				}
				// Add JWKS secret volume if file-based JWT authentication is configured
				if isFileBasedJWTAuth(server) {
					volumes = append(volumes, corev1.Volume{
						Name: "jwks",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: server.Spec.Authn.JWT.JWKS.Name,
							},
						},
					})
				}
				return volumes
			}(),
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

	return deployment, controllerutil.SetOwnerReference(server, deployment, t.scheme)
}

// createSecretEnvFrom creates envFrom references from secret references
func (t *agentGatewayTranslator) createSecretEnvFrom(
	secretRefs []corev1.ObjectReference,
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
	envVars := make([]corev1.EnvVar, len(env))
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

// isFileBasedJWTAuth checks if the JWT authentication is configured to use file-based JWKS
func isFileBasedJWTAuth(server *v1alpha1.MCPServer) bool {
	return server.Spec.Authn != nil &&
		server.Spec.Authn.JWT != nil &&
		server.Spec.Authn.JWT.JWKS != nil
}

func (t *agentGatewayTranslator) translateAgentGatewayService(server *v1alpha1.MCPServer) (*corev1.Service, error) {
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

func (t *agentGatewayTranslator) translateAgentGatewayConfigMap(
	ctx context.Context,
	server *v1alpha1.MCPServer,
) (*corev1.ConfigMap, error) {
	config, err := t.translateAgentGatewayConfig(ctx, server)
	if err != nil {
		return nil, fmt.Errorf("failed to translate MCP server config: %w", err)
	}

	configYAML, err := yaml.Marshal(config)
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
			"local.yaml": string(configYAML),
		},
	}

	return configMap, controllerutil.SetOwnerReference(server, configMap, t.scheme)
}

func (t *agentGatewayTranslator) translateAgentGatewayConfig(
	ctx context.Context,
	server *v1alpha1.MCPServer,
) (*LocalConfig, error) {
	if server.Spec.TransportType != v1alpha1.TransportTypeStdio {
		return nil, nil // Only Stdio transport is supported for now
	}

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

	policies := &FilterOrPolicy{}
	if authn := server.Spec.Authn; authn != nil && authn.JWT != nil {
		jwt := authn.JWT
		if jwt.JWKS != nil {
			secret := &corev1.Secret{}
			secretKey := client.ObjectKey{
				Namespace: server.Namespace,
				Name:      jwt.JWKS.Name,
			}
			if err := t.client.Get(ctx, secretKey, secret); err != nil {
				return nil, fmt.Errorf("failed to get JWKS secret %s: %w", jwt.JWKS.Name, err)
			}

			policies.JWTAuth = &JWTAuth{
				Issuer:    jwt.Issuer,
				Audiences: jwt.Audiences,
				JWKS: &JWKS{
					File: "/jwks/" + jwt.JWKS.Key,
				},
			}
		}
	}

	if authz := server.Spec.Authz; authz != nil {
		if authz.CEL != nil && len(authz.CEL.Rules) > 0 {
			policies.MCPAuthorization = &MCPAuthorization{
				Rules: authz.CEL.Rules,
			}
		}

		if authz.Server != nil {
			providerMap := make(map[string]interface{})
			if authz.Server.Provider != nil {
				// only keycloak is supported for now
				providerMap["keycloak"] = struct{}{}
			}

			// agentgateway expects a map[string]interface{}
			resourceMetadata := make(map[string]interface{})
			// Add the required resource field using the base url of the protected resource
			// and the default path prefix for the MCP server
			resourceMetadata["resource"] = fmt.Sprintf("%s/mcp", authz.Server.ResourceMetadata.BaseUrl)

			// Add scopes if they exist
			if authz.Server.ResourceMetadata.ScopesSupported != nil {
				resourceMetadata["scopesSupported"] = authz.Server.ResourceMetadata.ScopesSupported
			}

			// Add bearer methods if they exist
			if authz.Server.ResourceMetadata.BearerMethodsSupported != nil {
				resourceMetadata["bearerMethodsSupported"] = authz.Server.ResourceMetadata.BearerMethodsSupported
			}

			// Add any additional fields if they exist
			if authz.Server.ResourceMetadata.AdditionalFields != nil {
				for k, v := range authz.Server.ResourceMetadata.AdditionalFields {
					resourceMetadata[k] = v
				}
			}

			policies.MCPAuthentication = &MCPAuthentication{
				Issuer:           authz.Server.Issuer,
				Audience:         authz.Server.Audience,
				JwksURL:          authz.Server.JwksURL,
				Provider:         providerMap,
				ResourceMetadata: resourceMetadata,
			}
		}
	}

	// Add CORS policy if routeFilter is configured
	if routeFilter := server.Spec.RouteFilter; routeFilter != nil && routeFilter.CORS != nil {
		policies.CORS = &CORS{
			AllowHeaders: routeFilter.CORS.AllowHeaders,
			AllowOrigins: routeFilter.CORS.AllowOrigins,
		}
	}

	// default path matches
	pathMatches := []RouteMatch{
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
	}
	if authz := server.Spec.Authz; authz != nil && authz.Server != nil && authz.Server.Provider != nil {
		if authz.Server.Provider.Keycloak.Realm == "" {
			return nil, fmt.Errorf("keycloak realm must be specified when using keycloak as the authorization server")
		}

		// add path for public keys enabled by the realm
		pathMatches = append(pathMatches, RouteMatch{
			Path: PathMatch{
				PathPrefix: fmt.Sprintf("/realms/%s", authz.Server.Provider.Keycloak.Realm),
			},
		})

		// add path for endpoint containing metadata about the protected resource
		pathMatches = append(pathMatches, RouteMatch{
			Path: PathMatch{
				Exact: "/.well-known/oauth-protected-resource/mcp",
			},
		})

		// add path for the dynamic client registration endpoint
		pathMatches = append(pathMatches, RouteMatch{
			Path: PathMatch{
				Exact: "/.well-known/oauth-authorization-server/mcp/client-registration",
			},
		})

		// add path for the authorization server metadata endpoint
		pathMatches = append(pathMatches, RouteMatch{
			Path: PathMatch{
				Exact: "/.well-known/oauth-authorization-server/mcp",
			},
		})
	}
	return &LocalConfig{
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
							Matches:   pathMatches,
							Backends: []RouteBackend{{
								Weight: 100,
								MCP: &MCPBackend{
									Targets: []MCPTarget{mcpTarget},
								},
							}},
							Policies: policies,
						}},
					},
				},
			},
		},
	}, nil
}
