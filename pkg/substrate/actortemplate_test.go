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
	"errors"
	"testing"

	atev1alpha1 "github.com/agent-substrate/substrate/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kagent-dev/kmcp/api/v1alpha1"
)

func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := atev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return scheme
}

func buildTestTemplate(t *testing.T) (*v1alpha1.MCPServer, *atev1alpha1.ActorTemplate) {
	t.Helper()
	scheme := newTestScheme(t)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "creds"},
		Data:       map[string][]byte{"API_KEY": []byte("k"), "ANOTHER": []byte("v")},
	}
	builder := &TemplateBuilder{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build(),
		Scheme: scheme,
		Defaults: Defaults{
			SnapshotsLocationPrefix: "s3://snaps",
			RunscAMD64:              BinaryConfig{URL: "gs://gvisor/runsc-amd64", SHA256: "ccc"},
			AdapterAMD64:            BinaryConfig{URL: "https://example.com/agw", SHA256: "aaa"},
		},
	}

	server := validHTTPServer()
	server.Spec.Deployment.Env = map[string]string{"Z_LAST": "z", "A_FIRST": "a"}
	server.Spec.Deployment.SecretRefs = []corev1.LocalObjectReference{{Name: "creds"}}
	server.Spec.Deployment.Labels = map[string]string{"team": "x"}

	tmpl, err := builder.Build(context.Background(), server, types.NamespacedName{Namespace: "pools", Name: "wp"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return server, tmpl
}

func TestTemplateBuilderBuild(t *testing.T) {
	server, tmpl := buildTestTemplate(t)

	if tmpl.Name != server.Name || tmpl.Namespace != server.Namespace {
		t.Fatalf("template name/namespace = %s/%s", tmpl.Namespace, tmpl.Name)
	}
	if tmpl.Labels[MCPServerLabelKey] != server.Name || tmpl.Labels["team"] != "x" {
		t.Fatalf("labels = %v", tmpl.Labels)
	}
	if len(tmpl.OwnerReferences) != 1 || tmpl.OwnerReferences[0].Kind != "MCPServer" ||
		!*tmpl.OwnerReferences[0].Controller {
		t.Fatalf("owner refs = %+v", tmpl.OwnerReferences)
	}
	if tmpl.Spec.PauseImage != DefaultPauseImage {
		t.Fatalf("pause image = %q", tmpl.Spec.PauseImage)
	}
	if tmpl.Spec.WorkerPoolRef.Namespace != "pools" || tmpl.Spec.WorkerPoolRef.Name != "wp" {
		t.Fatalf("workerPoolRef = %+v", tmpl.Spec.WorkerPoolRef)
	}
	if tmpl.Spec.SnapshotsConfig.Location != "s3://snaps/ns/srv" {
		t.Fatalf("snapshots location = %q", tmpl.Spec.SnapshotsConfig.Location)
	}
	if tmpl.Spec.Runsc.AMD64 == nil || tmpl.Spec.Runsc.AMD64.URL != "gs://gvisor/runsc-amd64" ||
		tmpl.Spec.Runsc.ARM64 != nil {
		t.Fatalf("runsc config = %+v", tmpl.Spec.Runsc)
	}

	if len(tmpl.Spec.Containers) != 1 {
		t.Fatalf("expected exactly one container, got %d", len(tmpl.Spec.Containers))
	}
	container := tmpl.Spec.Containers[0]
	if container.Image != pinnedImage {
		t.Fatalf("container image = %q", container.Image)
	}
	if len(container.Command) != 3 || container.Command[0] != "/bin/sh" || container.Command[1] != "-c" {
		t.Fatalf("container command = %v", container.Command[:2])
	}
}

func TestTemplateBuilderEnvExpansion(t *testing.T) {
	_, tmpl := buildTestTemplate(t)
	container := tmpl.Spec.Containers[0]

	// Env must be sorted and include both literal vars and expanded secret keys.
	names := make([]string, 0, len(container.Env))
	for _, env := range container.Env {
		names = append(names, env.Name)
	}
	want := []string{"ANOTHER", "API_KEY", "A_FIRST", "Z_LAST"}
	if len(names) != len(want) {
		t.Fatalf("env names = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("env names = %v, want %v", names, want)
		}
	}
	for _, env := range container.Env {
		if env.Name == "API_KEY" {
			if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil ||
				env.ValueFrom.SecretKeyRef.Name != "creds" || env.ValueFrom.SecretKeyRef.Key != "API_KEY" {
				t.Fatalf("API_KEY env = %+v", env)
			}
		}
		if env.Name == "A_FIRST" && env.Value != "a" {
			t.Fatalf("A_FIRST env = %+v", env)
		}
	}
}

func TestTemplateBuilderMissingSecret(t *testing.T) {
	scheme := newTestScheme(t)
	builder := &TemplateBuilder{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme: scheme,
	}

	server := validHTTPServer()
	server.Spec.Deployment.SecretRefs = []corev1.LocalObjectReference{{Name: "missing"}}

	_, err := builder.Build(context.Background(), server, types.NamespacedName{Namespace: "pools", Name: "wp"})
	var notFound *SecretNotFoundError
	if !errors.As(err, &notFound) || notFound.Name != "missing" {
		t.Fatalf("want SecretNotFoundError for %q, got %v", "missing", err)
	}
}
