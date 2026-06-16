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
	"fmt"
	"strings"

	"go.uber.org/multierr"

	"github.com/kagent-dev/kmcp/api/v1alpha1"
)

// ValidateForSubstrate rejects MCPServer configurations that cannot be
// expressed on the substrate runtime without changing workload behavior.
// Substrate ActorTemplates have no volumes, ConfigMap mounts, init containers,
// sidecars, or replica counts; fields that merely tune Kubernetes packaging
// and scheduling are ignored instead (see IgnoredFields).
func ValidateForSubstrate(server *v1alpha1.MCPServer) error {
	var errs error
	d := server.Spec.Deployment

	if d.Image == "" || !strings.Contains(d.Image, "@") {
		errs = multierr.Append(errs, fmt.Errorf(
			"spec.deployment.image must be set and digest-pinned (e.g. @sha256:...): "+
				"substrate invalidates snapshots on image changes"))
	}
	if len(d.ConfigMapRefs) > 0 {
		errs = multierr.Append(errs, fmt.Errorf(
			"spec.deployment.configMapRefs is not supported on the substrate runtime (ActorTemplates have no volume mounts)"))
	}
	if len(d.Volumes) > 0 {
		errs = multierr.Append(errs, fmt.Errorf(
			"spec.deployment.volumes is not supported on the substrate runtime (ActorTemplates have no volumes)"))
	}
	if len(d.VolumeMounts) > 0 {
		errs = multierr.Append(errs, fmt.Errorf(
			"spec.deployment.volumeMounts is not supported on the substrate runtime (ActorTemplates have no volume mounts)"))
	}
	if len(d.Sidecars) > 0 {
		errs = multierr.Append(errs, fmt.Errorf(
			"spec.deployment.sidecars is not supported on the substrate runtime"))
	}
	if d.InitContainer != nil {
		errs = multierr.Append(errs, fmt.Errorf(
			"spec.deployment.initContainer is not supported on the substrate runtime "+
				"(the transport adapter is bootstrapped inside the actor)"))
	}
	if d.Replicas != nil && *d.Replicas > 1 {
		errs = multierr.Append(errs, fmt.Errorf(
			"spec.deployment.replicas > 1 is not supported on the substrate runtime "+
				"(each MCPServer is backed by a single actor)"))
	}
	if server.Spec.HTTPTransport != nil && server.Spec.HTTPTransport.TLS != nil {
		errs = multierr.Append(errs, fmt.Errorf(
			"spec.httpTransport.tls is not supported on the substrate runtime "+
				"(the atenet router speaks plain HTTP to the actor)"))
	}

	switch server.Spec.TransportType {
	case v1alpha1.TransportTypeStdio:
		if d.Cmd == "" {
			errs = multierr.Append(errs, fmt.Errorf("spec.deployment.cmd is required for stdio transport"))
		}
	case v1alpha1.TransportTypeHTTP:
		if d.Cmd == "" {
			errs = multierr.Append(errs, fmt.Errorf(
				"spec.deployment.cmd is required for http transport on the substrate runtime "+
					"(the actor bootstrap script replaces the image entrypoint)"))
		}
		if server.Spec.HTTPTransport == nil || server.Spec.HTTPTransport.TargetPort == 0 {
			errs = multierr.Append(errs, fmt.Errorf("spec.httpTransport.targetPort is required for http transport"))
		} else if server.Spec.HTTPTransport.TargetPort == adapterBindPort {
			errs = multierr.Append(errs, fmt.Errorf(
				"spec.httpTransport.targetPort must not be %d on the substrate runtime: "+
					"the in-actor transport adapter binds it (the atenet router only routes to port %d)",
				adapterBindPort, adapterBindPort))
		}
	default:
		errs = multierr.Append(errs, fmt.Errorf("unsupported transport type: %s", server.Spec.TransportType))
	}

	return errs
}

// IgnoredFields lists deployment fields the substrate runtime silently drops:
// they configure Kubernetes pod packaging and scheduling, which substrate's
// WorkerPool owns. Returned for surfacing as a warning event.
func IgnoredFields(server *v1alpha1.MCPServer) []string {
	var ignored []string
	d := server.Spec.Deployment
	if d.Resources != nil {
		ignored = append(ignored, "resources")
	}
	if d.SecurityContext != nil {
		ignored = append(ignored, "securityContext")
	}
	if d.PodSecurityContext != nil {
		ignored = append(ignored, "podSecurityContext")
	}
	if len(d.Tolerations) > 0 {
		ignored = append(ignored, "tolerations")
	}
	if d.Affinity != nil {
		ignored = append(ignored, "affinity")
	}
	if len(d.NodeSelector) > 0 {
		ignored = append(ignored, "nodeSelector")
	}
	if d.ImagePullPolicy != "" {
		ignored = append(ignored, "imagePullPolicy")
	}
	if len(d.ImagePullSecrets) > 0 {
		ignored = append(ignored, "imagePullSecrets")
	}
	if d.ServiceAccount != nil {
		ignored = append(ignored, "serviceAccount")
	}
	if d.ServiceAccountName != "" {
		ignored = append(ignored, "serviceAccountName")
	}
	return ignored
}
