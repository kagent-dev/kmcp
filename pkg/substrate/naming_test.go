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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kagent-dev/kmcp/api/v1alpha1"
)

func mcpServer(namespace, name string) *v1alpha1.MCPServer {
	return &v1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, UID: "0000-1111"},
	}
}

func TestActorID(t *testing.T) {
	if got := ActorID(mcpServer("kmcp-test", "everything")); got != "mcp-kmcp-test-everything" {
		t.Fatalf("ActorID = %q", got)
	}

	long := ActorID(mcpServer("a-very-long-namespace-name-here", "an-even-longer-mcp-server-name-that-overflows"))
	if len(long) > 63 {
		t.Fatalf("ActorID too long: %d chars", len(long))
	}
	if !dns1123Label.MatchString(long) {
		t.Fatalf("ActorID %q is not a DNS-1123 label", long)
	}
}

func TestActorHost(t *testing.T) {
	if got := ActorHost("mcp-ns-name", ""); got != "mcp-ns-name.actors.resources.substrate.ate.dev" {
		t.Fatalf("ActorHost = %q", got)
	}
	if got := ActorHost("a", "custom.suffix"); got != "a.custom.suffix" {
		t.Fatalf("ActorHost with suffix = %q", got)
	}
}

func TestSnapshotsLocation(t *testing.T) {
	server := mcpServer("ns", "name")
	if got := SnapshotsLocation(server, ""); got != DefaultSnapshotsLocationPrefix+"/ns/name" {
		t.Fatalf("default location = %q", got)
	}
	if got := SnapshotsLocation(server, "s3://bucket/"); got != "s3://bucket/ns/name" {
		t.Fatalf("prefix location = %q", got)
	}

	server.Spec.Substrate = &v1alpha1.SubstrateSpec{
		SnapshotsConfig: &v1alpha1.SubstrateSnapshotsConfig{Location: "gs://explicit/path"},
	}
	if got := SnapshotsLocation(server, "s3://bucket"); got != "gs://explicit/path" {
		t.Fatalf("explicit location = %q", got)
	}
}

func TestDefaultsFallbacks(t *testing.T) {
	var d Defaults
	if got := d.RouterURL(); got != DefaultAtenetRouterURL {
		t.Fatalf("RouterURL = %q", got)
	}
	if got := d.HostSuffix(); got != DefaultActorHostSuffix {
		t.Fatalf("HostSuffix = %q", got)
	}
	d.AtenetRouterURL = "http://router.example:8080"
	d.ActorHostSuffix = "custom.suffix"
	if d.RouterURL() != "http://router.example:8080" || d.HostSuffix() != "custom.suffix" {
		t.Fatalf("overrides not honored: %q %q", d.RouterURL(), d.HostSuffix())
	}
	if !strings.HasPrefix(DefaultPauseImage, "registry.k8s.io/pause") || !strings.Contains(DefaultPauseImage, "@sha256:") {
		t.Fatalf("DefaultPauseImage must be digest-pinned: %q", DefaultPauseImage)
	}
}
