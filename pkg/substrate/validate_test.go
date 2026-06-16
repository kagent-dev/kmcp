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
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kagent-dev/kmcp/api/v1alpha1"
)

const pinnedImage = "ghcr.io/example/mcp@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func validHTTPServer() *v1alpha1.MCPServer {
	return &v1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "srv"},
		Spec: v1alpha1.MCPServerSpec{
			Runtime:       v1alpha1.MCPServerRuntimeSubstrate,
			Substrate:     &v1alpha1.SubstrateSpec{},
			TransportType: v1alpha1.TransportTypeHTTP,
			HTTPTransport: &v1alpha1.HTTPTransport{TargetPort: 3001, TargetPath: "/mcp"},
			Deployment: v1alpha1.MCPServerDeployment{
				Image: pinnedImage,
				Cmd:   "node",
				Args:  []string{"server.js"},
			},
		},
	}
}

func validStdioServer() *v1alpha1.MCPServer {
	server := validHTTPServer()
	server.Spec.TransportType = v1alpha1.TransportTypeStdio
	server.Spec.HTTPTransport = nil
	server.Spec.Deployment.Cmd = "npx"
	server.Spec.Deployment.Args = []string{"-y", "@modelcontextprotocol/server-everything"}
	return server
}

func TestValidateForSubstrate(t *testing.T) {
	replicasTwo := int32(2)
	replicasOne := int32(1)

	tests := []struct {
		name    string
		mutate  func(*v1alpha1.MCPServer)
		wantErr string
	}{
		{name: "valid http", mutate: func(*v1alpha1.MCPServer) {}},
		{name: "valid stdio", mutate: func(s *v1alpha1.MCPServer) {
			*s = *validStdioServer()
		}},
		{name: "unpinned image", mutate: func(s *v1alpha1.MCPServer) {
			s.Spec.Deployment.Image = "ghcr.io/example/mcp:latest"
		}, wantErr: "digest-pinned"},
		{name: "empty image", mutate: func(s *v1alpha1.MCPServer) {
			s.Spec.Deployment.Image = ""
		}, wantErr: "digest-pinned"},
		{name: "configMapRefs", mutate: func(s *v1alpha1.MCPServer) {
			s.Spec.Deployment.ConfigMapRefs = []corev1.LocalObjectReference{{Name: "cm"}}
		}, wantErr: "configMapRefs"},
		{name: "volumes", mutate: func(s *v1alpha1.MCPServer) {
			s.Spec.Deployment.Volumes = []corev1.Volume{{Name: "v"}}
		}, wantErr: "volumes"},
		{name: "volumeMounts", mutate: func(s *v1alpha1.MCPServer) {
			s.Spec.Deployment.VolumeMounts = []corev1.VolumeMount{{Name: "v", MountPath: "/v"}}
		}, wantErr: "volumeMounts"},
		{name: "sidecars", mutate: func(s *v1alpha1.MCPServer) {
			s.Spec.Deployment.Sidecars = []corev1.Container{{Name: "s"}}
		}, wantErr: "sidecars"},
		{name: "initContainer", mutate: func(s *v1alpha1.MCPServer) {
			s.Spec.Deployment.InitContainer = &v1alpha1.InitContainerConfig{}
		}, wantErr: "initContainer"},
		{name: "replicas > 1", mutate: func(s *v1alpha1.MCPServer) {
			s.Spec.Deployment.Replicas = &replicasTwo
		}, wantErr: "replicas"},
		{name: "replicas == 1 allowed", mutate: func(s *v1alpha1.MCPServer) {
			s.Spec.Deployment.Replicas = &replicasOne
		}},
		{name: "http tls", mutate: func(s *v1alpha1.MCPServer) {
			s.Spec.HTTPTransport.TLS = &v1alpha1.HTTPTransportTLS{SecretRef: "tls"}
		}, wantErr: "tls"},
		{name: "http without cmd", mutate: func(s *v1alpha1.MCPServer) {
			s.Spec.Deployment.Cmd = ""
		}, wantErr: "cmd is required for http"},
		{name: "stdio without cmd", mutate: func(s *v1alpha1.MCPServer) {
			*s = *validStdioServer()
			s.Spec.Deployment.Cmd = ""
		}, wantErr: "cmd is required for stdio"},
		{name: "http without target port", mutate: func(s *v1alpha1.MCPServer) {
			s.Spec.HTTPTransport = nil
		}, wantErr: "targetPort"},
		{name: "http target port 80 collides with adapter", mutate: func(s *v1alpha1.MCPServer) {
			s.Spec.HTTPTransport.TargetPort = 80
		}, wantErr: "must not be 80"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := validHTTPServer()
			tt.mutate(server)
			err := ValidateForSubstrate(server)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestIgnoredFields(t *testing.T) {
	server := validHTTPServer()
	if got := IgnoredFields(server); len(got) != 0 {
		t.Fatalf("expected no ignored fields, got %v", got)
	}

	server.Spec.Deployment.Resources = &corev1.ResourceRequirements{}
	server.Spec.Deployment.NodeSelector = map[string]string{"zone": "a"}
	server.Spec.Deployment.ServiceAccountName = "sa"
	got := IgnoredFields(server)
	want := []string{"resources", "nodeSelector", "serviceAccountName"}
	if len(got) != len(want) {
		t.Fatalf("ignored fields = %v, want %v", got, want)
	}
	for _, field := range want {
		found := false
		for _, g := range got {
			if g == field {
				found = true
			}
		}
		if !found {
			t.Fatalf("ignored fields %v missing %q", got, field)
		}
	}
}
