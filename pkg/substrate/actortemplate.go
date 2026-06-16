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
	"fmt"
	"sort"

	atev1alpha1 "github.com/agent-substrate/substrate/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kagent-dev/kmcp/api/v1alpha1"
)

// SecretNotFoundError marks a missing Secret referenced by
// spec.deployment.secretRefs, so the controller can map it to
// ResolvedRefs=False instead of a generic failure.
type SecretNotFoundError struct {
	Name string
	Err  error
}

func (e *SecretNotFoundError) Error() string {
	return fmt.Sprintf("secret %q referenced by spec.deployment.secretRefs not found", e.Name)
}

func (e *SecretNotFoundError) Unwrap() error { return e.Err }

// TemplateBuilder translates MCPServers into ate.dev ActorTemplates.
type TemplateBuilder struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Defaults Defaults
}

// Build produces the desired ActorTemplate for the MCPServer: a single
// container running the user's image with a /bin/sh -c bootstrap that fronts
// the MCP server with agentgateway on port 80 (the only port the atenet
// router forwards to).
func (b *TemplateBuilder) Build(
	ctx context.Context,
	server *v1alpha1.MCPServer,
	workerPool types.NamespacedName,
) (*atev1alpha1.ActorTemplate, error) {
	startupScript, err := BuildStartupScript(server, b.Defaults)
	if err != nil {
		return nil, err
	}
	env, err := b.buildContainerEnv(ctx, server)
	if err != nil {
		return nil, err
	}

	pauseImage := b.Defaults.PauseImage
	if pauseImage == "" {
		pauseImage = DefaultPauseImage
	}

	labels := map[string]string{
		managedByLabel:    managedByKmcp,
		MCPServerLabelKey: server.Name,
	}
	for k, v := range server.Spec.Deployment.Labels {
		labels[k] = v
	}

	tmpl := &atev1alpha1.ActorTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ActorTemplateName(server),
			Namespace:   server.Namespace,
			Labels:      labels,
			Annotations: server.Spec.Deployment.Annotations,
		},
		Spec: atev1alpha1.ActorTemplateSpec{
			PauseImage: pauseImage,
			Runsc:      b.runscConfig(),
			Containers: []atev1alpha1.Container{{
				Name:    "mcp-server",
				Image:   server.Spec.Deployment.Image,
				Command: []string{"/bin/sh", "-c", startupScript},
				Env:     env,
			}},
			WorkerPoolRef: corev1.ObjectReference{
				Namespace: workerPool.Namespace,
				Name:      workerPool.Name,
			},
			SnapshotsConfig: atev1alpha1.SnapshotsConfig{
				Location: SnapshotsLocation(server, b.Defaults.SnapshotsLocationPrefix),
			},
		},
	}
	if err := controllerutil.SetControllerReference(server, tmpl, b.Scheme); err != nil {
		return nil, fmt.Errorf("set ActorTemplate owner ref: %w", err)
	}
	return tmpl, nil
}

func (b *TemplateBuilder) runscConfig() atev1alpha1.RunscConfig {
	cfg := atev1alpha1.RunscConfig{}
	if b.Defaults.RunscAMD64.URL != "" {
		cfg.AMD64 = &atev1alpha1.RunscPlatformConfig{
			URL:        b.Defaults.RunscAMD64.URL,
			SHA256Hash: b.Defaults.RunscAMD64.SHA256,
		}
	}
	if b.Defaults.RunscARM64.URL != "" {
		cfg.ARM64 = &atev1alpha1.RunscPlatformConfig{
			URL:        b.Defaults.RunscARM64.URL,
			SHA256Hash: b.Defaults.RunscARM64.SHA256,
		}
	}
	return cfg
}

// buildContainerEnv converts the deployment env map plus expanded secretRefs
// into the ActorTemplate container env. Secret values are not inlined: each
// key becomes a secretKeyRef that substrate's ate-api resolves at
// actor-create time (value rotation needs no template change; key changes
// re-bake the golden snapshot).
func (b *TemplateBuilder) buildContainerEnv(
	ctx context.Context,
	server *v1alpha1.MCPServer,
) ([]corev1.EnvVar, error) {
	env := make([]corev1.EnvVar, 0, len(server.Spec.Deployment.Env))
	for name, value := range server.Spec.Deployment.Env {
		env = append(env, corev1.EnvVar{Name: name, Value: value})
	}

	for _, ref := range server.Spec.Deployment.SecretRefs {
		if ref.Name == "" {
			continue
		}
		var secret corev1.Secret
		key := types.NamespacedName{Namespace: server.Namespace, Name: ref.Name}
		if err := b.Client.Get(ctx, key, &secret); err != nil {
			if apierrors.IsNotFound(err) {
				return nil, &SecretNotFoundError{Name: ref.Name, Err: err}
			}
			return nil, fmt.Errorf("get secret %q: %w", ref.Name, err)
		}
		for dataKey := range secret.Data {
			env = append(env, corev1.EnvVar{
				Name: dataKey,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: ref.Name},
						Key:                  dataKey,
					},
				},
			})
		}
	}

	sort.Slice(env, func(i, j int) bool { return env[i].Name < env[j].Name })
	return env, nil
}
