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
	"kagent.dev/kmcp/pkg/agentgateway"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kagentdevv1alpha1 "kagent.dev/kmcp/api/v1alpha1"
)

// MCPServerReconciler reconciles a MCPServer object
type MCPServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kagent.dev,resources=mcpservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kagent.dev,resources=mcpservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kagent.dev,resources=mcpservers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MCPServer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.0/pkg/reconcile
func (r *MCPServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Fetch the MCPServer instance
	mcpServer := &kagentdevv1alpha1.MCPServer{}
	if err := r.Get(ctx, req.NamespacedName, mcpServer); err != nil {
		// If the resource is not found, we can ignore the error since it will be requeued later
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	t := agentgateway.NewAgentGatewayTranslator(r.Scheme)
	outputs, err := t.TranslateAgentGatewayOutputs(mcpServer)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to translate MCPServer outputs")
		r.reconcileStatus(ctx, mcpServer, err)
		return ctrl.Result{}, err
	}

	err = r.reconcileOutputs(ctx, outputs)
	if err != nil {
		log.FromContext(ctx).Error(err, "Failed to reconcile outputs")
		r.reconcileStatus(ctx, mcpServer, err)
		return ctrl.Result{}, err
	}

	r.reconcileStatus(ctx, mcpServer, nil)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MCPServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kagentdevv1alpha1.MCPServer{}).
		Named("mcpserver").
		Complete(r)
}

func (r *MCPServerReconciler) reconcileOutputs(ctx context.Context, outputs *agentgateway.AgentGatewayOutputs) error {
	// upsert the outputs to the cluster
	if outputs.Deployment != nil {
		if err := upsertOutput(ctx, r.Client, outputs.Deployment); err != nil {
			return err
		}
	}
	if outputs.Service != nil {
		if err := upsertOutput(ctx, r.Client, outputs.Service); err != nil {
			return err
		}
	}
	if outputs.ConfigMap != nil {
		if err := upsertOutput(ctx, r.Client, outputs.ConfigMap); err != nil {
			return err
		}
	}

	return nil
}

func (r *MCPServerReconciler) reconcileStatus(ctx context.Context, server *kagentdevv1alpha1.MCPServer, err error) {
	// TODO: Implement status reconciliation logic
	// log for now
	log.FromContext(ctx).Info("Reconcile status", "server", server.Name, "error", err)
}

func upsertOutput(ctx context.Context, kube client.Client, output client.Object) error {
	existing := output.DeepCopyObject().(client.Object)
	if err := kube.Get(ctx, client.ObjectKeyFromObject(existing), existing); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		// If not found, create it
		if err := kube.Create(ctx, output); err != nil {
			return err
		}
	} else {
		// If found, update it
		output.SetResourceVersion(existing.GetResourceVersion())
		if err := kube.Update(ctx, output); err != nil {
			return err
		}
	}
	return nil
}
