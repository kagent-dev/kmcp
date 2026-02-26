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
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerStartAndShutdown verifies the MCP server starts and gracefully shuts down.
func TestServerStartAndShutdown(t *testing.T) {
	handler := setupTestHandler(t)

	// Find a free port
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	bindAddr := fmt.Sprintf(":%d", port)
	server := NewServer(handler, bindAddr)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	// Wait for server to start
	require.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/mcp", port))
		if err != nil {
			return false
		}
		resp.Body.Close()
		return true
	}, 3*time.Second, 50*time.Millisecond, "server should start accepting connections")

	// Cancel context to trigger shutdown
	cancel()

	select {
	case err := <-errCh:
		assert.NoError(t, err, "server should shut down cleanly")
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

// TestNewServer verifies server construction.
func TestNewServer(t *testing.T) {
	handler := setupTestHandler(t)
	server := NewServer(handler, ":8083")

	assert.Equal(t, ":8083", server.bindAddr)
	assert.Equal(t, handler, server.handler)
}
