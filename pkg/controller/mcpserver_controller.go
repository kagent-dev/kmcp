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
	"fmt"

	"github.com/kagent-dev/kmcp/pkg/controller/internal/agentgateway"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kagentdevv1alpha1 "github.com/kagent-dev/kmcp/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MCPServerReconciler reconciles a MCPServer object
type MCPServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kagent.dev,resources=mcpservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kagent.dev,resources=mcpservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kagent.dev,resources=mcpservers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

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

	t := agentgateway.NewAgentGatewayTranslator(r.Scheme, r.Client)
	outputs, err := t.TranslateAgentGatewayOutputs(ctx, mcpServer)
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
		For(&kagentdevv1alpha1.MCPServer{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&appsv1.Deployment{}, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Owns(&corev1.Service{}, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Owns(&corev1.ConfigMap{}, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(func(
				ctx context.Context,
				o client.Object,
			) []reconcile.Request {
				mcpServers := &kagentdevv1alpha1.MCPServerList{}
				if err := r.List(ctx, mcpServers); err != nil {
					log.FromContext(ctx).Error(err,
						"failed to list mcp servers for secret event")
					return []reconcile.Request{}
				}

				var requests []reconcile.Request
				for _, server := range mcpServers.Items {
					if auth := server.Spec.Authn; auth != nil && auth.JWT != nil && auth.JWT.JWKS != nil {
						if auth.JWT.JWKS.Name == o.GetName() && server.Namespace == o.GetNamespace() {
							requests = append(requests, reconcile.Request{
								NamespacedName: types.NamespacedName{
									Name:      server.Name,
									Namespace: server.Namespace,
								},
							})
						}
					}
				}
				return requests
			}),
		).
		Named("mcpserver").
		Complete(r)
}

func (r *MCPServerReconciler) reconcileOutputs(ctx context.Context, outputs *agentgateway.Outputs) error {
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

func (r *MCPServerReconciler) reconcileStatus(
	ctx context.Context,
	server *kagentdevv1alpha1.MCPServer,
	reconcileErr error,
) {
	// Update ObservedGeneration
	server.Status.ObservedGeneration = server.Generation

	// Set Accepted condition based on validation
	if err := r.validateMCPServer(server); err != nil {
		setAcceptedCondition(server, false, kagentdevv1alpha1.MCPServerReasonInvalidConfig, err.Error())
		// If validation fails, set other conditions as unknown/false
		setResolvedRefsCondition(
			server,
			false,
			kagentdevv1alpha1.MCPServerReasonImageNotFound,
			"Configuration validation failed",
		)
		setProgrammedCondition(
			server,
			false,
			kagentdevv1alpha1.MCPServerReasonDeploymentFailed,
			"Configuration validation failed",
		)
		setReadyCondition(
			server,
			false,
			kagentdevv1alpha1.MCPServerReasonPodsNotReady,
			"Configuration validation failed",
		)
	} else {
		setAcceptedCondition(
			server,
			true,
			kagentdevv1alpha1.MCPServerReasonAccepted,
			"MCPServer configuration is valid",
		)

		// Set ResolvedRefs condition (for now, assume image exists - could be enhanced later)
		setResolvedRefsCondition(
			server,
			true,
			kagentdevv1alpha1.MCPServerReasonResolvedRefs,
			"All references resolved successfully",
		)

		// Set Programmed condition based on reconcile result
		if reconcileErr != nil {
			setProgrammedCondition(
				server,
				false,
				kagentdevv1alpha1.MCPServerReasonDeploymentFailed,
				reconcileErr.Error(),
			)
			setReadyCondition(server,
				false,
				kagentdevv1alpha1.MCPServerReasonPodsNotReady,
				"Resources failed to be created",
			)
		} else {
			setProgrammedCondition(server,
				true,
				kagentdevv1alpha1.MCPServerReasonProgrammed,
				"All resources created successfully",
			)

			// Check Ready condition by examining deployment status
			r.checkReadyCondition(ctx, server)
		}
	}

	// Update the status
	if err := r.Status().Update(ctx, server); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update MCPServer status")
	}
}

// validateMCPServer validates the MCPServer configuration
func (r *MCPServerReconciler) validateMCPServer(server *kagentdevv1alpha1.MCPServer) error {
	// Check if transport type is supported
	if server.Spec.TransportType != kagentdevv1alpha1.TransportTypeStdio &&
		server.Spec.TransportType != kagentdevv1alpha1.TransportTypeHTTP {
		return fmt.Errorf("unsupported transport type: %s", server.Spec.TransportType)
	}

	// Check if required fields are present
	if server.Spec.Deployment.Image == "" {
		return fmt.Errorf("deployment.image is required")
	}

	// Additional validation could be added here
	return nil
}

// checkReadyCondition checks if the MCPServer is ready by examining the deployment status
func (r *MCPServerReconciler) checkReadyCondition(ctx context.Context, server *kagentdevv1alpha1.MCPServer) {
	// Get the deployment
	deployment := &appsv1.Deployment{}
	deploymentName := server.Name
	if err := r.Get(ctx, client.ObjectKey{Name: deploymentName, Namespace: server.Namespace}, deployment); err != nil {
		if client.IgnoreNotFound(err) == nil {
			setReadyCondition(server, false, kagentdevv1alpha1.MCPServerReasonPodsNotReady, "Deployment not found")
		} else {
			setReadyCondition(
				server,
				false,
				kagentdevv1alpha1.MCPServerReasonPodsNotReady,
				fmt.Sprintf("Error getting deployment: %s", err.Error()),
			)
		}
		return
	}

	// Check if deployment is available
	if deployment.Status.ReadyReplicas > 0 && deployment.Status.ReadyReplicas == deployment.Status.Replicas {
		setReadyCondition(
			server,
			true,
			kagentdevv1alpha1.MCPServerReasonReady,
			"Deployment is ready and all pods are running",
		)
	} else {
		message := fmt.Sprintf("Deployment not ready: %d/%d replicas ready",
			deployment.Status.ReadyReplicas, deployment.Status.Replicas)
		setReadyCondition(server, false, kagentdevv1alpha1.MCPServerReasonPodsNotReady, message)
	}
}

// setCondition sets the given condition on the MCPServer status.
func setCondition(
	server *kagentdevv1alpha1.MCPServer,
	conditionType kagentdevv1alpha1.MCPServerConditionType,
	status metav1.ConditionStatus,
	reason kagentdevv1alpha1.MCPServerConditionReason,
	message string,
) {
	now := metav1.Now()
	condition := metav1.Condition{
		Type:               string(conditionType),
		Status:             status,
		LastTransitionTime: now,
		Reason:             string(reason),
		Message:            message,
		ObservedGeneration: server.Generation,
	}

	// Find existing condition
	for i, existingCondition := range server.Status.Conditions {
		if existingCondition.Type == string(conditionType) {
			// Only update LastTransitionTime if status changed
			if existingCondition.Status != status {
				server.Status.Conditions[i] = condition
			} else {
				// Update other fields but keep the original LastTransitionTime
				condition.LastTransitionTime = existingCondition.LastTransitionTime
				server.Status.Conditions[i] = condition
			}
			return
		}
	}

	// Add new condition
	server.Status.Conditions = append(server.Status.Conditions, condition)
}

// setAcceptedCondition sets the Accepted condition on the MCPServer.
func setAcceptedCondition(
	server *kagentdevv1alpha1.MCPServer,
	accepted bool,
	reason kagentdevv1alpha1.MCPServerConditionReason,
	message string,
) {
	status := metav1.ConditionTrue
	if !accepted {
		status = metav1.ConditionFalse
	}
	setCondition(server, kagentdevv1alpha1.MCPServerConditionAccepted, status, reason, message)
}

// setResolvedRefsCondition sets the ResolvedRefs condition on the MCPServer.
func setResolvedRefsCondition(
	server *kagentdevv1alpha1.MCPServer,
	resolved bool,
	reason kagentdevv1alpha1.MCPServerConditionReason,
	message string,
) {
	status := metav1.ConditionTrue
	if !resolved {
		status = metav1.ConditionFalse
	}
	setCondition(server, kagentdevv1alpha1.MCPServerConditionResolvedRefs, status, reason, message)
}

// setProgrammedCondition sets the Programmed condition on the MCPServer.
func setProgrammedCondition(
	server *kagentdevv1alpha1.MCPServer,
	programmed bool,
	reason kagentdevv1alpha1.MCPServerConditionReason,
	message string,
) {
	status := metav1.ConditionTrue
	if !programmed {
		status = metav1.ConditionFalse
	}
	setCondition(server, kagentdevv1alpha1.MCPServerConditionProgrammed, status, reason, message)
}

// setReadyCondition sets the Ready condition on the MCPServer.
func setReadyCondition(
	server *kagentdevv1alpha1.MCPServer,
	ready bool,
	reason kagentdevv1alpha1.MCPServerConditionReason,
	message string,
) {
	status := metav1.ConditionTrue
	if !ready {
		status = metav1.ConditionFalse
	}
	setCondition(server, kagentdevv1alpha1.MCPServerConditionReady, status, reason, message)
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
