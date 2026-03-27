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
	"testing"
)

func TestFilterValidNamespaces(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty list",
			input:    []string{""},
			expected: nil,
		},
		{
			name:     "single valid namespace",
			input:    []string{"default"},
			expected: []string{"default"},
		},
		{
			name:     "multiple valid namespaces",
			input:    []string{"ns1", "ns2", "ns3"},
			expected: []string{"ns1", "ns2", "ns3"},
		},
		{
			name:     "filters out empty strings",
			input:    []string{"ns1", "", "ns2"},
			expected: []string{"ns1", "ns2"},
		},
		{
			name:     "filters out whitespace-only strings",
			input:    []string{"ns1", "  ", "ns2"},
			expected: []string{"ns1", "ns2"},
		},
		{
			name:     "trims whitespace from valid namespaces",
			input:    []string{" ns1 ", "ns2"},
			expected: []string{"ns1", "ns2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterValidNamespaces(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("got %v (len %d), want %v (len %d)", result, len(result), tt.expected, len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("index %d: got %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestConfigureNamespaceWatching(t *testing.T) {
	t.Run("empty list returns nil (cluster-wide)", func(t *testing.T) {
		result := configureNamespaceWatching([]string{})
		if result != nil {
			t.Errorf("expected nil for cluster-wide watching, got %v", result)
		}
	})

	t.Run("single namespace returns map with one entry", func(t *testing.T) {
		result := configureNamespaceWatching([]string{"default"})
		if len(result) != 1 {
			t.Errorf("expected 1 entry, got %d", len(result))
		}
		if _, ok := result["default"]; !ok {
			t.Errorf("expected key 'default' in map, got %v", result)
		}
	})

	t.Run("multiple namespaces returns map with all entries", func(t *testing.T) {
		result := configureNamespaceWatching([]string{"ns1", "ns2", "ns3"})
		if len(result) != 3 {
			t.Errorf("expected 3 entries, got %d", len(result))
		}
		for _, ns := range []string{"ns1", "ns2", "ns3"} {
			if _, ok := result[ns]; !ok {
				t.Errorf("expected key %q in map, got %v", ns, result)
			}
		}
	})
}
