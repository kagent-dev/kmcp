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
	"context"
	"crypto/tls"
	"fmt"

	"github.com/agent-substrate/substrate/proto/ateapipb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
)

// ControlAPI is the subset of the ate-api control plane the MCPServer
// substrate controller depends on. *Client implements it; tests fake it.
type ControlAPI interface {
	// GetActor returns the actor with the given id, or a NotFound gRPC error.
	GetActor(ctx context.Context, actorID string) (*ateapipb.Actor, error)
	// CreateActor instantiates an actor from an ActorTemplate.
	CreateActor(ctx context.Context, actorID, tmplNS, tmplName string) (*ateapipb.Actor, error)
	// SuspendActor snapshots a running actor and frees its worker. Required
	// before deletion: ate-api only deletes suspended actors.
	SuspendActor(ctx context.Context, actorID string) error
	// DeleteActor removes the actor. Deletion is asynchronous: poll GetActor
	// until it returns NotFound.
	DeleteActor(ctx context.Context, actorID string) error
}

// Client wraps the ate-api Control gRPC service.
type Client struct {
	ateapipb.ControlClient
	conn *grpc.ClientConn
	cfg  Config
}

var _ ControlAPI = (*Client)(nil)

// Dial connects to the ate-api server and blocks until the connection is
// ready or Config.DialTimeout elapses.
func Dial(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.AteAPIEndpoint == "" {
		return nil, fmt.Errorf("substrate: ate-api endpoint is required")
	}
	dialTimeout := cfg.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = DefaultDialTimeout
	}
	dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if cfg.Insecure {
		// Kind/local ate-api uses pod-issued certs; skip verification.
		tlsCfg.InsecureSkipVerify = true
	}
	conn, err := grpc.NewClient(cfg.AteAPIEndpoint,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	if err != nil {
		return nil, fmt.Errorf("substrate: dial ate-api %q: %w", cfg.AteAPIEndpoint, err)
	}
	// NewClient stays idle until Connect() or an RPC; waitConnReady enforces DialTimeout.
	conn.Connect()
	if err := waitConnReady(dialCtx, conn); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("substrate: dial ate-api %q: %w", cfg.AteAPIEndpoint, err)
	}

	return &Client{
		ControlClient: ateapipb.NewControlClient(conn),
		conn:          conn,
		cfg:           cfg,
	}, nil
}

func waitConnReady(ctx context.Context, conn *grpc.ClientConn) error {
	for {
		switch s := conn.GetState(); s {
		case connectivity.Ready:
			return nil
		case connectivity.Shutdown:
			return fmt.Errorf("connection shut down")
		default:
			if !conn.WaitForStateChange(ctx, s) {
				if err := ctx.Err(); err != nil {
					return err
				}
				return fmt.Errorf("connection closed before ready")
			}
		}
	}
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) callCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := c.cfg.CallTimeout
	if timeout <= 0 {
		timeout = DefaultCallTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

func (c *Client) GetActor(ctx context.Context, actorID string) (*ateapipb.Actor, error) {
	ctx, cancel := c.callCtx(ctx)
	defer cancel()
	resp, err := c.ControlClient.GetActor(ctx, &ateapipb.GetActorRequest{ActorId: actorID})
	if err != nil {
		return nil, err
	}
	return resp.GetActor(), nil
}

func (c *Client) CreateActor(ctx context.Context, actorID, tmplNS, tmplName string) (*ateapipb.Actor, error) {
	ctx, cancel := c.callCtx(ctx)
	defer cancel()
	resp, err := c.ControlClient.CreateActor(ctx, &ateapipb.CreateActorRequest{
		ActorId:                actorID,
		ActorTemplateNamespace: tmplNS,
		ActorTemplateName:      tmplName,
	})
	if err != nil {
		return nil, err
	}
	return resp.GetActor(), nil
}

func (c *Client) ResumeActor(ctx context.Context, actorID string) (*ateapipb.Actor, error) {
	ctx, cancel := c.callCtx(ctx)
	defer cancel()
	resp, err := c.ControlClient.ResumeActor(ctx, &ateapipb.ResumeActorRequest{ActorId: actorID})
	if err != nil {
		return nil, err
	}
	return resp.GetActor(), nil
}

func (c *Client) SuspendActor(ctx context.Context, actorID string) error {
	ctx, cancel := c.callCtx(ctx)
	defer cancel()
	_, err := c.ControlClient.SuspendActor(ctx, &ateapipb.SuspendActorRequest{ActorId: actorID})
	return err
}

func (c *Client) DeleteActor(ctx context.Context, actorID string) error {
	ctx, cancel := c.callCtx(ctx)
	defer cancel()
	_, err := c.ControlClient.DeleteActor(ctx, &ateapipb.DeleteActorRequest{ActorId: actorID})
	return err
}
