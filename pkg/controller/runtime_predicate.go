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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kagentdevv1alpha1 "github.com/kagent-dev/kmcp/api/v1alpha1"
)

// effectiveRuntime returns the runtime an MCPServer is provisioned with,
// defaulting to kubernetes when unset.
func effectiveRuntime(server *kagentdevv1alpha1.MCPServer) kagentdevv1alpha1.MCPServerRuntime {
	if server.Spec.Runtime == "" {
		return kagentdevv1alpha1.MCPServerRuntimeKubernetes
	}
	return server.Spec.Runtime
}

// mcpServerRuntimePredicate filters MCPServer events to a single runtime so
// each runtime's controller only reconciles its own resources. Update events
// match when either the old or the new object matches, so a runtime flip is
// seen by both controllers (the losing one gets a chance to clean up).
func mcpServerRuntimePredicate(rt kagentdevv1alpha1.MCPServerRuntime) predicate.Predicate {
	matches := func(obj client.Object) bool {
		server, ok := obj.(*kagentdevv1alpha1.MCPServer)
		if !ok {
			return false
		}
		return effectiveRuntime(server) == rt
	}
	return predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return matches(e.Object) },
		UpdateFunc:  func(e event.UpdateEvent) bool { return matches(e.ObjectOld) || matches(e.ObjectNew) },
		DeleteFunc:  func(e event.DeleteEvent) bool { return matches(e.Object) },
		GenericFunc: func(e event.GenericEvent) bool { return matches(e.Object) },
	}
}
