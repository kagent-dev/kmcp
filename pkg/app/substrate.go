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

package app

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kagent-dev/kmcp/pkg/controller"
	"github.com/kagent-dev/kmcp/pkg/substrate"
)

// SubstrateConfig configures the optional Agent Substrate runtime support.
// Substrate support is enabled if and only if AteAPIEndpoint is set.
type SubstrateConfig struct {
	AteAPIEndpoint string
	Insecure       bool
	DialTimeout    time.Duration
	CallTimeout    time.Duration

	AtenetRouterURL string
	ActorHostSuffix string

	DefaultWorkerPoolNamespace string
	DefaultWorkerPoolName      string
	SnapshotsLocationPrefix    string
	PauseImage                 string

	RunscAMD64URL      string
	RunscAMD64SHA256   string
	RunscARM64URL      string
	RunscARM64SHA256   string
	AdapterAMD64URL    string
	AdapterAMD64SHA256 string
	AdapterARM64URL    string
	AdapterARM64SHA256 string

	IngressMode           string
	IngressProxyNamespace string
	IngressProxyConfigMap string
	IngressProxyPort      uint
}

// Enabled reports whether substrate support should be turned on.
func (c *SubstrateConfig) Enabled() bool {
	return c.AteAPIEndpoint != ""
}

func (c *SubstrateConfig) SetFlags(commandLine *flag.FlagSet) {
	commandLine.StringVar(&c.AteAPIEndpoint, "substrate-ate-api-endpoint", "",
		"gRPC endpoint of the substrate ate-api server (e.g. dns:///api.ate-system.svc:443). "+
			"Setting this enables substrate runtime support for MCPServers.")
	commandLine.BoolVar(&c.Insecure, "substrate-ate-api-insecure", false,
		"Skip TLS certificate verification when connecting to ate-api (kind/local installs use pod-issued certs).")
	commandLine.DurationVar(&c.DialTimeout, "substrate-ate-api-dial-timeout", substrate.DefaultDialTimeout,
		"Timeout for the initial connection to ate-api.")
	commandLine.DurationVar(&c.CallTimeout, "substrate-ate-api-call-timeout", substrate.DefaultCallTimeout,
		"Timeout for individual ate-api RPCs.")
	commandLine.StringVar(&c.AtenetRouterURL, "substrate-atenet-router-url", substrate.DefaultAtenetRouterURL,
		"URL of the substrate atenet router that fronts actors.")
	commandLine.StringVar(&c.ActorHostSuffix, "substrate-actor-host-suffix", substrate.DefaultActorHostSuffix,
		"DNS suffix the atenet router matches actor Host headers against.")
	commandLine.StringVar(&c.DefaultWorkerPoolNamespace, "substrate-default-workerpool-namespace", "",
		"Namespace of the WorkerPool used when an MCPServer does not reference one.")
	commandLine.StringVar(&c.DefaultWorkerPoolName, "substrate-default-workerpool-name", "",
		"Name of the WorkerPool used when an MCPServer does not reference one.")
	commandLine.StringVar(&c.SnapshotsLocationPrefix, "substrate-snapshots-location-prefix",
		substrate.DefaultSnapshotsLocationPrefix,
		"Object-storage prefix for actor snapshots; the per-server default is <prefix>/<namespace>/<name>.")
	commandLine.StringVar(&c.PauseImage, "substrate-pause-image", substrate.DefaultPauseImage,
		"Digest-pinned pause image used as the actor's root sandbox container.")
	commandLine.StringVar(&c.RunscAMD64URL, "substrate-runsc-amd64-url", "",
		"URL of the amd64 gVisor runsc binary fetched by atelet.")
	commandLine.StringVar(&c.RunscAMD64SHA256, "substrate-runsc-amd64-sha256", "",
		"SHA256 of the amd64 runsc binary.")
	commandLine.StringVar(&c.RunscARM64URL, "substrate-runsc-arm64-url", "",
		"URL of the arm64 gVisor runsc binary fetched by atelet.")
	commandLine.StringVar(&c.RunscARM64SHA256, "substrate-runsc-arm64-sha256", "",
		"SHA256 of the arm64 runsc binary.")
	commandLine.StringVar(&c.AdapterAMD64URL, "substrate-adapter-amd64-url", "",
		"URL of the amd64 agentgateway binary downloaded by the actor bootstrap when not bundled in the image.")
	commandLine.StringVar(&c.AdapterAMD64SHA256, "substrate-adapter-amd64-sha256", "",
		"SHA256 of the amd64 agentgateway binary.")
	commandLine.StringVar(&c.AdapterARM64URL, "substrate-adapter-arm64-url", "",
		"URL of the arm64 agentgateway binary downloaded by the actor bootstrap when not bundled in the image.")
	commandLine.StringVar(&c.AdapterARM64SHA256, "substrate-adapter-arm64-sha256", "",
		"SHA256 of the arm64 agentgateway binary.")
	commandLine.StringVar(&c.IngressMode, "substrate-ingress-mode", string(substrate.IngressModeManagedProxy),
		"How substrate MCPServers are exposed: 'managed-proxy' routes every server's Service through the shared "+
			"agentgateway Deployment shipped by the kmcp helm chart; 'none' creates no ingress objects (consumers "+
			"use the MCPServer status.substrate fields directly).")
	commandLine.StringVar(&c.IngressProxyNamespace, "substrate-ingress-proxy-namespace", "",
		"Namespace of the shared ingress proxy Deployment (and its route ConfigMap). "+
			"Defaults to the POD_NAMESPACE environment variable.")
	commandLine.StringVar(&c.IngressProxyConfigMap, "substrate-ingress-proxy-configmap",
		substrate.DefaultProxyConfigMapName,
		"Name of the shared route ConfigMap mounted by the ingress proxy Deployment.")
	commandLine.UintVar(&c.IngressProxyPort, "substrate-ingress-proxy-port", uint(substrate.DefaultProxyPort),
		"Listen port of the shared ingress proxy.")
}

// ingressConfig validates and resolves the ingress flags.
func (c *SubstrateConfig) ingressConfig() (substrate.IngressConfig, error) {
	mode := substrate.IngressMode(c.IngressMode)
	switch mode {
	case substrate.IngressModeManagedProxy, substrate.IngressModeNone:
	default:
		return substrate.IngressConfig{}, fmt.Errorf(
			"invalid --substrate-ingress-mode %q (supported: managed-proxy, none)", c.IngressMode)
	}
	namespace := c.IngressProxyNamespace
	if namespace == "" {
		namespace = os.Getenv("POD_NAMESPACE")
	}
	if mode == substrate.IngressModeManagedProxy && namespace == "" {
		return substrate.IngressConfig{}, fmt.Errorf(
			"--substrate-ingress-proxy-namespace (or POD_NAMESPACE) is required in managed-proxy mode")
	}
	if c.IngressProxyPort > 65535 {
		return substrate.IngressConfig{}, fmt.Errorf("invalid --substrate-ingress-proxy-port %d", c.IngressProxyPort)
	}
	return substrate.IngressConfig{
		Mode:               mode,
		ProxyNamespace:     namespace,
		ProxyConfigMapName: c.IngressProxyConfigMap,
		ProxyPort:          uint16(c.IngressProxyPort),
	}, nil
}

func (c *SubstrateConfig) clientConfig() substrate.Config {
	return substrate.Config{
		AteAPIEndpoint: c.AteAPIEndpoint,
		Insecure:       c.Insecure,
		DialTimeout:    c.DialTimeout,
		CallTimeout:    c.CallTimeout,
	}
}

func (c *SubstrateConfig) defaults() substrate.Defaults {
	return substrate.Defaults{
		AtenetRouterURL: c.AtenetRouterURL,
		ActorHostSuffix: c.ActorHostSuffix,
		PauseImage:      c.PauseImage,
		RunscAMD64:      substrate.BinaryConfig{URL: c.RunscAMD64URL, SHA256: c.RunscAMD64SHA256},
		RunscARM64:      substrate.BinaryConfig{URL: c.RunscARM64URL, SHA256: c.RunscARM64SHA256},
		AdapterAMD64:    substrate.BinaryConfig{URL: c.AdapterAMD64URL, SHA256: c.AdapterAMD64SHA256},
		AdapterARM64:    substrate.BinaryConfig{URL: c.AdapterARM64URL, SHA256: c.AdapterARM64SHA256},
		DefaultWorkerPool: types.NamespacedName{
			Namespace: c.DefaultWorkerPoolNamespace,
			Name:      c.DefaultWorkerPoolName,
		},
		SnapshotsLocationPrefix: c.SnapshotsLocationPrefix,
	}
}

// setupSubstrateController dials ate-api (failing fast on connection errors)
// and registers the substrate MCPServer controller with the manager.
func setupSubstrateController(ctx context.Context, mgr ctrl.Manager, cfg *SubstrateConfig) error {
	ingress, err := cfg.ingressConfig()
	if err != nil {
		return err
	}
	ateClient, err := substrate.Dial(ctx, cfg.clientConfig())
	if err != nil {
		return fmt.Errorf("connect to substrate ate-api: %w", err)
	}
	reconciler := &controller.MCPServerSubstrateReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		APIReader: mgr.GetAPIReader(),
		Ate:       ateClient,
		Defaults:  cfg.defaults(),
		Ingress:   ingress,
		Recorder:  mgr.GetEventRecorder("mcpserver-substrate"),
	}
	if err := reconciler.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("create substrate controller: %w", err)
	}
	return nil
}
