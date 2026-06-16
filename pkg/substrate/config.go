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

// Package substrate integrates kmcp with Agent Substrate (ate.dev): it
// translates MCPServers with runtime=substrate into ActorTemplates, drives
// actor lifecycle through the ate-api control plane, and preserves the
// normal Service contract by routing each server through one shared
// in-cluster ingress proxy.
package substrate

import (
	"time"

	"k8s.io/apimachinery/pkg/types"
)

const (
	// DefaultAtenetRouterURL is the in-cluster URL of substrate's Envoy
	// router as installed by the substrate manifests.
	DefaultAtenetRouterURL = "http://atenet-router.ate-system.svc:80"

	// DefaultActorHostSuffix is the DNS suffix the atenet router matches
	// actor Host headers against.
	DefaultActorHostSuffix = "actors.resources.substrate.ate.dev"

	// DefaultPauseImage is the off-GCP pause image recommended by substrate.
	DefaultPauseImage = "registry.k8s.io/pause:3.10.2" +
		"@sha256:f548e0e8e3dc1896ca956272154dde3314e8cc4fde0a57577ee9fa1c63f5baf4"

	// DefaultSnapshotsLocationPrefix is the bucket prefix used for actor
	// snapshots when the MCPServer does not specify one.
	DefaultSnapshotsLocationPrefix = "gs://ate-snapshots"

	// DefaultDialTimeout bounds the initial connection to ate-api.
	DefaultDialTimeout = 10 * time.Second

	// DefaultCallTimeout bounds individual ate-api RPCs.
	DefaultCallTimeout = 30 * time.Second
)

// Config holds the connection settings for the ate-api control plane.
type Config struct {
	// AteAPIEndpoint is the gRPC target of the ate-api server,
	// e.g. dns:///api.ate-system.svc:443. Substrate support is enabled if
	// and only if this is set.
	AteAPIEndpoint string

	// Insecure skips TLS certificate verification. The kind/local ate-api
	// install uses pod-issued certificates that are not verifiable with
	// system roots.
	Insecure bool

	// DialTimeout bounds the initial connection.
	DialTimeout time.Duration

	// CallTimeout bounds individual RPCs.
	CallTimeout time.Duration
}

// BinaryConfig points at a per-architecture binary download with an integrity
// hash. Used for both the runsc binaries (fetched by atelet) and the
// agentgateway adapter binary (fetched by the actor bootstrap script).
type BinaryConfig struct {
	URL    string
	SHA256 string
}

// Defaults holds the controller-level defaults applied to every substrate
// MCPServer translation.
type Defaults struct {
	// AtenetRouterURL is published in MCPServer status and targeted by the
	// ingress proxy.
	AtenetRouterURL string

	// ActorHostSuffix is appended to actor IDs to form router Host headers.
	ActorHostSuffix string

	// PauseImage is the root sandbox container image (must be digest-pinned).
	PauseImage string

	// RunscAMD64 and RunscARM64 configure the gVisor runsc binary fetched by
	// atelet for golden-snapshot creation and resume.
	RunscAMD64 BinaryConfig
	RunscARM64 BinaryConfig

	// AdapterAMD64 and AdapterARM64 configure the agentgateway binary
	// downloaded by the actor bootstrap script when the workload image does
	// not bundle it.
	AdapterAMD64 BinaryConfig
	AdapterARM64 BinaryConfig

	// DefaultWorkerPool is used when an MCPServer does not reference a
	// WorkerPool explicitly.
	DefaultWorkerPool types.NamespacedName

	// SnapshotsLocationPrefix prefixes the per-MCPServer default snapshot
	// location <prefix>/<namespace>/<name>.
	SnapshotsLocationPrefix string
}

// RouterURL returns the configured atenet router URL or the default.
func (d Defaults) RouterURL() string {
	if d.AtenetRouterURL != "" {
		return d.AtenetRouterURL
	}
	return DefaultAtenetRouterURL
}

// HostSuffix returns the configured actor host suffix or the default.
func (d Defaults) HostSuffix() string {
	if d.ActorHostSuffix != "" {
		return d.ActorHostSuffix
	}
	return DefaultActorHostSuffix
}
