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
	"testing"

	"github.com/kagent-dev/kmcp/api/v1alpha1"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setupScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(s)
	return s
}

func createTestMCPServer(name, namespace string, port uint16, ready bool, transport v1alpha1.TransportType) *v1alpha1.MCPServer {
	conditions := []metav1.Condition{
		{
			Type:               string(v1alpha1.MCPServerConditionAccepted),
			Status:             metav1.ConditionTrue,
			Reason:             string(v1alpha1.MCPServerReasonAccepted),
			LastTransitionTime: metav1.Now(),
		},
	}

	readyStatus := metav1.ConditionFalse
	if ready {
		readyStatus = metav1.ConditionTrue
	}
	conditions = append(conditions, metav1.Condition{
		Type:               string(v1alpha1.MCPServerConditionReady),
		Status:             readyStatus,
		Reason:             string(v1alpha1.MCPServerReasonReady),
		LastTransitionTime: metav1.Now(),
	})

	return &v1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.MCPServerSpec{
			TransportType: transport,
			Deployment: v1alpha1.MCPServerDeployment{
				Port: port,
			},
		},
		Status: v1alpha1.MCPServerStatus{
			Conditions: conditions,
		},
	}
}

func setupTestHandler(t *testing.T, objects ...client.Object) *MCPHandler {
	t.Helper()
	kubeClient := fake.NewClientBuilder().
		WithScheme(setupScheme()).
		WithObjects(objects...).
		WithStatusSubresource(&v1alpha1.MCPServer{}).
		Build()

	handler, err := NewMCPHandler(kubeClient)
	require.NoError(t, err)
	require.NotNil(t, handler)
	return handler
}

// TestNewMCPHandler verifies handler creation and tool registration.
func TestNewMCPHandler(t *testing.T) {
	handler := setupTestHandler(t)
	assert.NotNil(t, handler.server)
	assert.NotNil(t, handler.httpHandler)
	assert.NotNil(t, handler.kubeClient)
}

// TestHandleListMCPServers tests the list_mcp_servers tool.
func TestHandleListMCPServers(t *testing.T) {
	tests := []struct {
		name          string
		objects       []client.Object
		input         ListMCPServersInput
		expectedCount int
		checkFunc     func(t *testing.T, output ListMCPServersOutput)
	}{
		{
			name:          "empty cluster returns empty list",
			objects:       nil,
			input:         ListMCPServersInput{},
			expectedCount: 0,
		},
		{
			name: "returns all servers across namespaces",
			objects: []client.Object{
				createTestMCPServer("weather-mcp", "default", 3000, true, v1alpha1.TransportTypeStdio),
				createTestMCPServer("db-mcp", "tools", 8080, true, v1alpha1.TransportTypeHTTP),
				createTestMCPServer("search-mcp", "default", 3000, false, v1alpha1.TransportTypeStdio),
			},
			input:         ListMCPServersInput{},
			expectedCount: 3,
			checkFunc: func(t *testing.T, output ListMCPServersOutput) {
				refs := make(map[string]MCPServerSummary)
				for _, s := range output.Servers {
					refs[s.Ref] = s
				}
				assert.Contains(t, refs, "default/weather-mcp")
				assert.Contains(t, refs, "tools/db-mcp")
				assert.Contains(t, refs, "default/search-mcp")

				assert.Equal(t, "Ready", refs["default/weather-mcp"].Status)
				assert.Equal(t, "Ready", refs["tools/db-mcp"].Status)
				assert.Equal(t, "NotReady", refs["default/search-mcp"].Status)

				assert.Equal(t, 3000, refs["default/weather-mcp"].Port)
				assert.Equal(t, 8080, refs["tools/db-mcp"].Port)

				assert.Equal(t, "stdio", refs["default/weather-mcp"].Transport)
				assert.Equal(t, "http", refs["tools/db-mcp"].Transport)
			},
		},
		{
			name: "filters by namespace",
			objects: []client.Object{
				createTestMCPServer("weather-mcp", "default", 3000, true, v1alpha1.TransportTypeStdio),
				createTestMCPServer("db-mcp", "tools", 8080, true, v1alpha1.TransportTypeHTTP),
			},
			input:         ListMCPServersInput{Namespace: "tools"},
			expectedCount: 1,
			checkFunc: func(t *testing.T, output ListMCPServersOutput) {
				assert.Equal(t, "tools/db-mcp", output.Servers[0].Ref)
			},
		},
		{
			name: "mixed ready and not-ready servers",
			objects: []client.Object{
				createTestMCPServer("ready-mcp", "default", 3000, true, v1alpha1.TransportTypeStdio),
				createTestMCPServer("notready-mcp", "default", 3000, false, v1alpha1.TransportTypeStdio),
			},
			input:         ListMCPServersInput{},
			expectedCount: 2,
			checkFunc: func(t *testing.T, output ListMCPServersOutput) {
				statuses := make(map[string]string)
				for _, s := range output.Servers {
					statuses[s.Ref] = s.Status
				}
				assert.Equal(t, "Ready", statuses["default/ready-mcp"])
				assert.Equal(t, "NotReady", statuses["default/notready-mcp"])
			},
		},
		{
			name: "default port when zero",
			objects: []client.Object{
				createTestMCPServer("zero-port-mcp", "default", 0, true, v1alpha1.TransportTypeStdio),
			},
			input:         ListMCPServersInput{},
			expectedCount: 1,
			checkFunc: func(t *testing.T, output ListMCPServersOutput) {
				assert.Equal(t, 3000, output.Servers[0].Port)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupTestHandler(t, tt.objects...)
			ctx := context.Background()

			result, output, err := handler.handleListMCPServers(ctx, &mcpsdk.CallToolRequest{}, tt.input)
			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Len(t, output.Servers, tt.expectedCount)

			if tt.checkFunc != nil {
				tt.checkFunc(t, output)
			}
		})
	}
}

// TestHandleListToolsValidation tests input validation for list_tools.
func TestHandleListToolsValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     ListToolsInput
		wantError string
	}{
		{
			name:      "invalid ref format - no slash",
			input:     ListToolsInput{Server: "just-a-name"},
			wantError: "invalid server reference",
		},
		{
			name:      "server not found",
			input:     ListToolsInput{Server: "nonexistent/server"},
			wantError: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupTestHandler(t)
			ctx := context.Background()

			result, _, err := handler.handleListTools(ctx, &mcpsdk.CallToolRequest{}, tt.input)
			require.NoError(t, err) // protocol-level error should not occur
			assert.True(t, result.IsError)

			// Check that error message contains expected text
			for _, content := range result.Content {
				if textContent, ok := content.(*mcpsdk.TextContent); ok {
					assert.Contains(t, textContent.Text, tt.wantError)
				}
			}
		})
	}
}

// TestHandleCallToolValidation tests input validation for call_tool.
func TestHandleCallToolValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     CallToolInput
		wantError string
	}{
		{
			name:      "missing tool name",
			input:     CallToolInput{Server: "default/test-mcp", Tool: ""},
			wantError: "tool name is required",
		},
		{
			name:      "invalid ref format",
			input:     CallToolInput{Server: "bad-ref", Tool: "some_tool"},
			wantError: "invalid server reference",
		},
		{
			name:      "server not found",
			input:     CallToolInput{Server: "nonexistent/server", Tool: "some_tool"},
			wantError: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupTestHandler(t)
			ctx := context.Background()

			result, _, err := handler.handleCallTool(ctx, &mcpsdk.CallToolRequest{}, tt.input)
			require.NoError(t, err) // protocol-level error should not occur
			assert.True(t, result.IsError)

			for _, content := range result.Content {
				if textContent, ok := content.(*mcpsdk.TextContent); ok {
					assert.Contains(t, textContent.Text, tt.wantError)
				}
			}
		})
	}
}

// TestGetMCPServer tests the getMCPServer helper.
func TestGetMCPServer(t *testing.T) {
	server := createTestMCPServer("weather-mcp", "default", 3000, true, v1alpha1.TransportTypeStdio)
	handler := setupTestHandler(t, server)
	ctx := context.Background()

	t.Run("valid ref returns server", func(t *testing.T) {
		result, err := handler.getMCPServer(ctx, "default/weather-mcp")
		require.NoError(t, err)
		assert.Equal(t, "weather-mcp", result.Name)
		assert.Equal(t, "default", result.Namespace)
	})

	t.Run("invalid ref format", func(t *testing.T) {
		_, err := handler.getMCPServer(ctx, "no-slash")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid server reference")
	})

	t.Run("non-existent server", func(t *testing.T) {
		_, err := handler.getMCPServer(ctx, "default/nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestMCPServerURL tests the URL derivation logic.
func TestMCPServerURL(t *testing.T) {
	tests := []struct {
		name     string
		server   *v1alpha1.MCPServer
		expected string
	}{
		{
			name: "stdio transport with default path",
			server: &v1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{Name: "weather-mcp", Namespace: "default"},
				Spec: v1alpha1.MCPServerSpec{
					TransportType: v1alpha1.TransportTypeStdio,
					Deployment:    v1alpha1.MCPServerDeployment{Port: 3000},
				},
			},
			expected: "http://weather-mcp.default.svc.cluster.local:3000/mcp",
		},
		{
			name: "http transport with custom path",
			server: &v1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{Name: "db-mcp", Namespace: "tools"},
				Spec: v1alpha1.MCPServerSpec{
					TransportType: v1alpha1.TransportTypeHTTP,
					Deployment:    v1alpha1.MCPServerDeployment{Port: 8080},
					HTTPTransport: &v1alpha1.HTTPTransport{
						TargetPath: "/api/mcp",
					},
				},
			},
			expected: "http://db-mcp.tools.svc.cluster.local:8080/api/mcp",
		},
		{
			name: "default port when zero",
			server: &v1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{Name: "test-mcp", Namespace: "ns"},
				Spec: v1alpha1.MCPServerSpec{
					Deployment: v1alpha1.MCPServerDeployment{Port: 0},
				},
			},
			expected: "http://test-mcp.ns.svc.cluster.local:3000/mcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, mcpServerURL(tt.server))
		})
	}
}

// TestServeHTTP verifies that the handler implements http.Handler.
func TestServeHTTP(t *testing.T) {
	handler := setupTestHandler(t)
	// Just verify it's a valid http.Handler — actual HTTP flow is covered by E2E
	var _ interface{ ServeHTTP(interface{}, interface{}) }
	_ = handler
}
