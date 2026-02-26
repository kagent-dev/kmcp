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

package mcp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/kagent-dev/kmcp/api/v1alpha1"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// MCPHandler handles MCP requests for dynamic MCP server discovery and invocation.
// This is the MCP-server-to-server equivalent of kagent-controller's A2A MCP endpoint.
type MCPHandler struct {
	kubeClient  client.Client
	httpHandler *mcpsdk.StreamableHTTPHandler
	server      *mcpsdk.Server
	sessions    sync.Map // cached MCP client sessions keyed by "namespace/name"
}

// Input/output types for MCP tools

type ListMCPServersInput struct {
	Namespace string `json:"namespace,omitempty" jsonschema:"Optional namespace filter"`
}

type ListMCPServersOutput struct {
	Servers []MCPServerSummary `json:"servers"`
}

type MCPServerSummary struct {
	Ref       string `json:"ref"`       // "namespace/name"
	Status    string `json:"status"`    // "Ready" / "NotReady"
	Port      int    `json:"port"`      // Service port
	Transport string `json:"transport"` // "stdio" / "http"
}

type ListToolsInput struct {
	Server string `json:"server" jsonschema:"MCP server reference in namespace/name format"`
}

type ListToolsOutput struct {
	Server string     `json:"server"`
	Tools  []ToolInfo `json:"tools"`
}

type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema,omitempty"`
}

type CallToolInput struct {
	Server    string         `json:"server" jsonschema:"MCP server reference in namespace/name format"`
	Tool      string         `json:"tool" jsonschema:"Tool name to invoke"`
	Arguments map[string]any `json:"arguments,omitempty" jsonschema:"Tool arguments as JSON object"`
}

type CallToolOutput struct {
	Server  string `json:"server"`
	Tool    string `json:"tool"`
	Content any    `json:"content"`
	IsError bool   `json:"isError"`
}

// NewMCPHandler creates a new MCP handler exposing list_mcp_servers, list_tools, and call_tool.
func NewMCPHandler(kubeClient client.Client) (*MCPHandler, error) {
	handler := &MCPHandler{
		kubeClient: kubeClient,
	}

	impl := &mcpsdk.Implementation{
		Name:    "kmcp-mcp-servers",
		Version: "0.1.0",
	}
	server := mcpsdk.NewServer(impl, nil)
	handler.server = server

	mcpsdk.AddTool[ListMCPServersInput, ListMCPServersOutput](
		server,
		&mcpsdk.Tool{
			Name:        "list_mcp_servers",
			Description: "List MCPServer deployments managed by kmcp in the cluster",
		},
		handler.handleListMCPServers,
	)

	mcpsdk.AddTool[ListToolsInput, ListToolsOutput](
		server,
		&mcpsdk.Tool{
			Name:        "list_tools",
			Description: "Connect to a specific MCPServer and return its available tools",
		},
		handler.handleListTools,
	)

	mcpsdk.AddTool[CallToolInput, CallToolOutput](
		server,
		&mcpsdk.Tool{
			Name:        "call_tool",
			Description: "Invoke a specific tool on a specific MCPServer",
		},
		handler.handleCallTool,
	)

	handler.httpHandler = mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server {
			return server
		},
		nil,
	)

	return handler, nil
}

// handleListMCPServers lists MCPServer CRs in the cluster.
func (h *MCPHandler) handleListMCPServers(ctx context.Context, req *mcpsdk.CallToolRequest, input ListMCPServersInput) (*mcpsdk.CallToolResult, ListMCPServersOutput, error) {
	log := ctrllog.FromContext(ctx).WithName("mcp-handler").WithValues("tool", "list_mcp_servers")

	serverList := &v1alpha1.MCPServerList{}
	listOpts := []client.ListOption{}
	if input.Namespace != "" {
		listOpts = append(listOpts, client.InNamespace(input.Namespace))
	}

	if err := h.kubeClient.List(ctx, serverList, listOpts...); err != nil {
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: fmt.Sprintf("Failed to list MCP servers: %v", err)},
			},
			IsError: true,
		}, ListMCPServersOutput{}, nil
	}

	servers := make([]MCPServerSummary, 0, len(serverList.Items))
	for _, server := range serverList.Items {
		status := "NotReady"
		for _, condition := range server.Status.Conditions {
			if condition.Type == string(v1alpha1.MCPServerConditionReady) && condition.Status == metav1.ConditionTrue {
				status = "Ready"
				break
			}
		}

		transport := string(server.Spec.TransportType)
		if transport == "" {
			transport = "stdio"
		}

		port := int(server.Spec.Deployment.Port)
		if port == 0 {
			port = 3000
		}

		servers = append(servers, MCPServerSummary{
			Ref:       server.Namespace + "/" + server.Name,
			Status:    status,
			Port:      port,
			Transport: transport,
		})
	}

	log.Info("Listed MCP servers", "count", len(servers))

	output := ListMCPServersOutput{Servers: servers}

	var fallbackText strings.Builder
	if len(servers) == 0 {
		fallbackText.WriteString("No MCP servers found.")
	} else {
		for i, s := range servers {
			if i > 0 {
				fallbackText.WriteByte('\n')
			}
			fmt.Fprintf(&fallbackText, "%s [%s] port=%d transport=%s", s.Ref, s.Status, s.Port, s.Transport)
		}
	}

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{Text: fallbackText.String()},
		},
	}, output, nil
}

// handleListTools connects to an MCPServer and returns its tool catalog.
func (h *MCPHandler) handleListTools(ctx context.Context, req *mcpsdk.CallToolRequest, input ListToolsInput) (*mcpsdk.CallToolResult, ListToolsOutput, error) {
	log := ctrllog.FromContext(ctx).WithName("mcp-handler").WithValues("tool", "list_tools", "server", input.Server)

	server, err := h.getMCPServer(ctx, input.Server)
	if err != nil {
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: err.Error()},
			},
			IsError: true,
		}, ListToolsOutput{}, nil
	}

	session, err := h.getOrCreateSession(ctx, server)
	if err != nil {
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: fmt.Sprintf("Failed to connect to MCP server %s: %v", input.Server, err)},
			},
			IsError: true,
		}, ListToolsOutput{}, nil
	}

	result, err := session.ListTools(ctx, &mcpsdk.ListToolsParams{})
	if err != nil {
		// Connection may be stale; evict and retry once
		h.sessions.Delete(input.Server)
		session, err = h.getOrCreateSession(ctx, server)
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: fmt.Sprintf("Failed to reconnect to MCP server %s: %v", input.Server, err)},
				},
				IsError: true,
			}, ListToolsOutput{}, nil
		}
		result, err = session.ListTools(ctx, &mcpsdk.ListToolsParams{})
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: fmt.Sprintf("Failed to list tools on MCP server %s: %v", input.Server, err)},
				},
				IsError: true,
			}, ListToolsOutput{}, nil
		}
	}

	tools := make([]ToolInfo, 0, len(result.Tools))
	for _, t := range result.Tools {
		tools = append(tools, ToolInfo{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}

	log.Info("Listed tools", "server", input.Server, "count", len(tools))

	output := ListToolsOutput{
		Server: input.Server,
		Tools:  tools,
	}

	var fallbackText strings.Builder
	if len(tools) == 0 {
		fmt.Fprintf(&fallbackText, "No tools found on MCP server %s.", input.Server)
	} else {
		fmt.Fprintf(&fallbackText, "Tools on %s:\n", input.Server)
		for _, t := range tools {
			fmt.Fprintf(&fallbackText, "- %s: %s\n", t.Name, t.Description)
		}
	}

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{Text: fallbackText.String()},
		},
	}, output, nil
}

// handleCallTool invokes a specific tool on a specific MCPServer.
func (h *MCPHandler) handleCallTool(ctx context.Context, req *mcpsdk.CallToolRequest, input CallToolInput) (*mcpsdk.CallToolResult, CallToolOutput, error) {
	log := ctrllog.FromContext(ctx).WithName("mcp-handler").WithValues("tool", "call_tool", "server", input.Server, "targetTool", input.Tool)

	if input.Tool == "" {
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: "tool name is required"},
			},
			IsError: true,
		}, CallToolOutput{}, nil
	}

	server, err := h.getMCPServer(ctx, input.Server)
	if err != nil {
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: err.Error()},
			},
			IsError: true,
		}, CallToolOutput{}, nil
	}

	session, err := h.getOrCreateSession(ctx, server)
	if err != nil {
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: fmt.Sprintf("Failed to connect to MCP server %s: %v", input.Server, err)},
			},
			IsError: true,
		}, CallToolOutput{}, nil
	}

	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      input.Tool,
		Arguments: input.Arguments,
	})
	if err != nil {
		// Connection may be stale; evict and retry once
		h.sessions.Delete(input.Server)
		session, err = h.getOrCreateSession(ctx, server)
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: fmt.Sprintf("Failed to reconnect to MCP server %s: %v", input.Server, err)},
				},
				IsError: true,
			}, CallToolOutput{}, nil
		}
		result, err = session.CallTool(ctx, &mcpsdk.CallToolParams{
			Name:      input.Tool,
			Arguments: input.Arguments,
		})
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: fmt.Sprintf("Failed to call tool %s on MCP server %s: %v", input.Tool, input.Server, err)},
				},
				IsError: true,
			}, CallToolOutput{}, nil
		}
	}

	log.Info("Called tool", "server", input.Server, "targetTool", input.Tool, "isError", result.IsError)

	// Extract text content for the fallback
	var fallbackText strings.Builder
	for _, content := range result.Content {
		if textContent, ok := content.(*mcpsdk.TextContent); ok {
			fallbackText.WriteString(textContent.Text)
		}
	}

	output := CallToolOutput{
		Server:  input.Server,
		Tool:    input.Tool,
		Content: result.StructuredContent,
		IsError: result.IsError,
	}

	// If no structured content, use the text content
	if output.Content == nil {
		output.Content = fallbackText.String()
	}

	return result, output, nil
}

// getMCPServer looks up an MCPServer CR by "namespace/name" reference.
func (h *MCPHandler) getMCPServer(ctx context.Context, ref string) (*v1alpha1.MCPServer, error) {
	ns, name, ok := strings.Cut(ref, "/")
	if !ok {
		return nil, fmt.Errorf("invalid server reference %q: must be in namespace/name format", ref)
	}

	server := &v1alpha1.MCPServer{}
	if err := h.kubeClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, server); err != nil {
		return nil, fmt.Errorf("MCPServer %s not found: %v", ref, err)
	}

	return server, nil
}

// mcpServerURL derives the in-cluster service URL for an MCPServer.
func mcpServerURL(server *v1alpha1.MCPServer) string {
	port := server.Spec.Deployment.Port
	if port == 0 {
		port = 3000
	}

	// For HTTP transport, use the configured path; for stdio, the transport adapter
	// serves MCP at the root path on the service port.
	path := "/mcp"
	if server.Spec.HTTPTransport != nil && server.Spec.HTTPTransport.TargetPath != "" {
		path = server.Spec.HTTPTransport.TargetPath
	}

	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d%s", server.Name, server.Namespace, port, path)
}

// getOrCreateSession returns a cached MCP client session or creates a new one.
func (h *MCPHandler) getOrCreateSession(ctx context.Context, server *v1alpha1.MCPServer) (*mcpsdk.ClientSession, error) {
	ref := server.Namespace + "/" + server.Name

	if cached, ok := h.sessions.Load(ref); ok {
		if session, ok := cached.(*mcpsdk.ClientSession); ok {
			return session, nil
		}
	}

	url := mcpServerURL(server)
	transport := &mcpsdk.StreamableClientTransport{
		Endpoint: url,
	}

	impl := &mcpsdk.Implementation{
		Name:    "kmcp-controller",
		Version: "0.1.0",
	}
	mcpClient := mcpsdk.NewClient(impl, nil)

	session, err := mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server at %s: %w", url, err)
	}

	h.sessions.Store(ref, session)
	return session, nil
}

// ServeHTTP implements http.Handler interface.
func (h *MCPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.httpHandler.ServeHTTP(w, r)
}
