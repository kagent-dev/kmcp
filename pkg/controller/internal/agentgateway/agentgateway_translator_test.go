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

package agentgateway

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestConvertEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected []corev1.EnvVar
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input",
			input:    map[string]string{},
			expected: []corev1.EnvVar{},
		},
		{
			name: "single env var",
			input: map[string]string{
				"GRAFANA_URL": "http://grafana.grafana:3000",
			},
			expected: []corev1.EnvVar{
				{
					Name:  "GRAFANA_URL",
					Value: "http://grafana.grafana:3000",
				},
			},
		},
		{
			name: "multiple env vars (sorted by name)",
			input: map[string]string{
				"ZZ_LAST":     "last",
				"AA_FIRST":    "first",
				"BB_MIDDLE":   "middle",
				"GRAFANA_URL": "http://grafana.grafana:3000",
			},
			expected: []corev1.EnvVar{
				{
					Name:  "AA_FIRST",
					Value: "first",
				},
				{
					Name:  "BB_MIDDLE",
					Value: "middle",
				},
				{
					Name:  "GRAFANA_URL",
					Value: "http://grafana.grafana:3000",
				},
				{
					Name:  "ZZ_LAST",
					Value: "last",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertEnvVars(tt.input)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("convertEnvVars() = %v, expected %v", result, tt.expected)
			}

			// Additional check: ensure no empty env vars are created
			for _, envVar := range result {
				if envVar.Name == "" {
					t.Errorf("convertEnvVars() created an empty env var name, this indicates the original bug is still present")
				}
			}

			// Verify length is correct
			if len(result) != len(tt.input) {
				t.Errorf("convertEnvVars() returned %d env vars, expected %d", len(result), len(tt.input))
			}
		})
	}
}
