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

package controller

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestUpsertOutputReplicas(t *testing.T) {
	tests := []struct {
		name    string
		desired *int32
		want    int32
	}{
		{name: "unset keeps the live count", desired: nil, want: 3},
		{name: "explicit value overrides the live count", desired: int32Ptr(2), want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			meta := metav1.ObjectMeta{Name: "mcp", Namespace: "default"}
			kube := fake.NewClientBuilder().WithObjects(&appsv1.Deployment{
				ObjectMeta: meta,
				Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(3)},
			}).Build()

			desired := &appsv1.Deployment{
				ObjectMeta: meta,
				Spec:       appsv1.DeploymentSpec{Replicas: tt.desired},
			}
			if err := upsertOutput(ctx, kube, desired); err != nil {
				t.Fatalf("upsertOutput: %v", err)
			}

			got := &appsv1.Deployment{}
			if err := kube.Get(ctx, client.ObjectKeyFromObject(desired), got); err != nil {
				t.Fatalf("get deployment: %v", err)
			}
			if got.Spec.Replicas == nil || *got.Spec.Replicas != tt.want {
				t.Fatalf("replicas = %v, want %d", got.Spec.Replicas, tt.want)
			}
		})
	}
}

func int32Ptr(i int32) *int32 { return &i }
