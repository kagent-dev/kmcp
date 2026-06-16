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
	_ "embed"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"

	"sigs.k8s.io/yaml"

	"github.com/kagent-dev/kmcp/api/v1alpha1"
	"github.com/kagent-dev/kmcp/pkg/controller/transportadapter"
)

// adapterBindPort is the port the in-actor agentgateway binds. The atenet
// router only forwards to port 80 on the actor, so both transports are
// normalized behind an adapter listening there.
const adapterBindPort = 80

//go:embed templates/mcp_startup.sh.tmpl
var startupScriptTmplContent string

var startupScriptTmpl = template.Must(template.New("mcp_startup").Parse(startupScriptTmplContent))

type startupScriptData struct {
	AMD64URL          string
	AMD64SHA256       string
	ARM64URL          string
	ARM64SHA256       string
	LocalConfigBase64 string
	BackgroundCmd     string
}

// BuildStartupScript renders the /bin/sh -c bootstrap executed inside the
// actor container: it ensures the agentgateway binary is present (downloading
// it once before the golden snapshot is taken), writes the inlined
// agentgateway config, starts the MCP server in the background for http
// transport, and execs agentgateway bound to port 80.
func BuildStartupScript(server *v1alpha1.MCPServer, defaults Defaults) (string, error) {
	cfg, err := buildActorLocalConfig(server)
	if err != nil {
		return "", err
	}
	cfgJSON, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal in-actor agentgateway config: %w", err)
	}

	data := startupScriptData{
		AMD64URL:          defaults.AdapterAMD64.URL,
		AMD64SHA256:       defaults.AdapterAMD64.SHA256,
		ARM64URL:          defaults.AdapterARM64.URL,
		ARM64SHA256:       defaults.AdapterARM64.SHA256,
		LocalConfigBase64: base64.StdEncoding.EncodeToString(cfgJSON),
	}
	if server.Spec.TransportType == v1alpha1.TransportTypeHTTP {
		data.BackgroundCmd = backgroundCommand(server)
	}

	var sb strings.Builder
	if err := startupScriptTmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("render startup script: %w", err)
	}
	return sb.String(), nil
}

// buildActorLocalConfig builds the agentgateway config served inside the
// actor. It mirrors the kubernetes-runtime translator config, with the bind
// port fixed to 80 (the only port the atenet router forwards to).
func buildActorLocalConfig(server *v1alpha1.MCPServer) (*transportadapter.LocalConfig, error) {
	target := transportadapter.MCPTarget{Name: server.Name}

	switch server.Spec.TransportType {
	case v1alpha1.TransportTypeStdio:
		target.Stdio = &transportadapter.StdioTargetSpec{
			Cmd:  server.Spec.Deployment.Cmd,
			Args: server.Spec.Deployment.Args,
			Env:  server.Spec.Deployment.Env,
		}
	case v1alpha1.TransportTypeHTTP:
		httpTransport := server.Spec.HTTPTransport
		if httpTransport == nil || httpTransport.TargetPort == 0 {
			return nil, fmt.Errorf("HTTP transport requires a target port")
		}
		// HTTPTransport is documented as streamable HTTP, so front it with a
		// streamable HTTP ("mcp") target, not a legacy SSE one.
		target.MCP = &transportadapter.SSETargetSpec{
			Host: "localhost",
			Port: httpTransport.TargetPort,
			Path: httpTransport.TargetPath,
		}
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", server.Spec.TransportType)
	}

	return &transportadapter.LocalConfig{
		Config: struct{}{},
		Binds: []transportadapter.LocalBind{{
			Port: adapterBindPort,
			Listeners: []transportadapter.LocalListener{{
				Name:     "default",
				Protocol: transportadapter.LocalListenerProtocolHTTP,
				Routes: []transportadapter.LocalRoute{{
					RouteName: "mcp",
					Matches: []transportadapter.RouteMatch{
						{Path: transportadapter.PathMatch{PathPrefix: "/sse"}},
						{Path: transportadapter.PathMatch{PathPrefix: MCPPath}},
					},
					Backends: []transportadapter.RouteBackend{{
						Weight: 100,
						MCP: &transportadapter.MCPBackend{
							Targets: []transportadapter.MCPTarget{target},
						},
					}},
				}},
			}},
		}},
	}, nil
}

// backgroundCommand renders the user's MCP server command for backgrounding
// in the startup script. Environment variables arrive via the container env
// (set on the ActorTemplate), so only cmd and args are rendered.
func backgroundCommand(server *v1alpha1.MCPServer) string {
	parts := make([]string, 0, 1+len(server.Spec.Deployment.Args))
	if server.Spec.Deployment.Cmd != "" {
		parts = append(parts, shellQuote(server.Spec.Deployment.Cmd))
	}
	for _, arg := range server.Spec.Deployment.Args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

// shellQuote single-quotes s for POSIX sh, escaping embedded single quotes.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
