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
	"errors"
	"net"
	"net/http"
	"time"

	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// Server wraps the MCPHandler in an HTTP server that implements manager.Runnable.
type Server struct {
	handler  *MCPHandler
	bindAddr string
}

// NewServer creates a new MCP HTTP server.
func NewServer(handler *MCPHandler, bindAddr string) *Server {
	return &Server{
		handler:  handler,
		bindAddr: bindAddr,
	}
}

// Start implements manager.Runnable. It starts the HTTP server and blocks
// until the context is cancelled, then gracefully shuts down.
func (s *Server) Start(ctx context.Context) error {
	log := ctrllog.FromContext(ctx).WithName("mcp-server")

	mux := http.NewServeMux()
	mux.Handle("/mcp", s.handler)
	mux.Handle("/mcp/", s.handler)

	srv := &http.Server{
		Addr:              s.bindAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("Starting MCP server", "addr", s.bindAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("Shutting down MCP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
