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
	"encoding/base64"
	"regexp"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"

	"github.com/kagent-dev/kmcp/pkg/controller/transportadapter"
)

var configBase64Pattern = regexp.MustCompile(`echo '([A-Za-z0-9+/=]+)' \| base64 -d`)

func decodeEmbeddedConfig(t *testing.T, script string) *transportadapter.LocalConfig {
	t.Helper()
	match := configBase64Pattern.FindStringSubmatch(script)
	if match == nil {
		t.Fatalf("startup script does not embed a base64 config:\n%s", script)
	}
	raw, err := base64.StdEncoding.DecodeString(match[1])
	if err != nil {
		t.Fatalf("decode embedded config: %v", err)
	}
	var cfg transportadapter.LocalConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal embedded config: %v", err)
	}
	return &cfg
}

func testDefaults() Defaults {
	return Defaults{
		AdapterAMD64: BinaryConfig{URL: "https://example.com/agw-amd64", SHA256: "aaa"},
		AdapterARM64: BinaryConfig{URL: "https://example.com/agw-arm64", SHA256: "bbb"},
	}
}

func TestBuildStartupScriptHTTP(t *testing.T) {
	server := validHTTPServer()
	server.Spec.Deployment.Args = []string{"server.js", "--flag", "it's quoted"}

	script, err := BuildStartupScript(server, testDefaults())
	if err != nil {
		t.Fatalf("BuildStartupScript: %v", err)
	}

	if !strings.Contains(script, `'node' 'server.js' '--flag' 'it'\''s quoted' &`) {
		t.Fatalf("script does not background the quoted server command:\n%s", script)
	}
	if !strings.Contains(script, `exec "$AGW" -f /etc/kmcp/local.yaml`) {
		t.Fatalf("script does not exec agentgateway:\n%s", script)
	}
	if !strings.Contains(script, "https://example.com/agw-amd64") ||
		!strings.Contains(script, "https://example.com/agw-arm64") {
		t.Fatalf("script does not reference adapter download URLs:\n%s", script)
	}

	cfg := decodeEmbeddedConfig(t, script)
	if len(cfg.Binds) != 1 || cfg.Binds[0].Port != 80 {
		t.Fatalf("in-actor config must bind port 80, got %+v", cfg.Binds)
	}
	target := cfg.Binds[0].Listeners[0].Routes[0].Backends[0].MCP.Targets[0]
	if target.MCP == nil || target.MCP.Host != "localhost" || target.MCP.Port != 3001 || target.MCP.Path != "/mcp" {
		t.Fatalf("unexpected streamable HTTP target: %+v", target.MCP)
	}
}

func TestBuildStartupScriptStdio(t *testing.T) {
	server := validStdioServer()

	script, err := BuildStartupScript(server, testDefaults())
	if err != nil {
		t.Fatalf("BuildStartupScript: %v", err)
	}

	if strings.Contains(script, " &\n") {
		t.Fatalf("stdio script must not background anything:\n%s", script)
	}

	cfg := decodeEmbeddedConfig(t, script)
	if cfg.Binds[0].Port != 80 {
		t.Fatalf("in-actor config must bind port 80, got %d", cfg.Binds[0].Port)
	}
	target := cfg.Binds[0].Listeners[0].Routes[0].Backends[0].MCP.Targets[0]
	if target.Stdio == nil || target.Stdio.Cmd != "npx" {
		t.Fatalf("unexpected stdio target: %+v", target.Stdio)
	}

	// Both /mcp and /sse must be routable.
	matches := cfg.Binds[0].Listeners[0].Routes[0].Matches
	prefixes := map[string]bool{}
	for _, m := range matches {
		prefixes[m.Path.PathPrefix] = true
	}
	if !prefixes["/mcp"] || !prefixes["/sse"] {
		t.Fatalf("expected /mcp and /sse route matches, got %+v", matches)
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"plain":    "'plain'",
		"":         "''",
		"it's":     `'it'\''s'`,
		"a b $VAR": "'a b $VAR'",
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Fatalf("shellQuote(%q) = %s, want %s", in, got, want)
		}
	}
}
