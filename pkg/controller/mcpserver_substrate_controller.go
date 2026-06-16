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
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	atev1alpha1 "github.com/agent-substrate/substrate/api/v1alpha1"
	"github.com/agent-substrate/substrate/proto/ateapipb"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kagentdevv1alpha1 "github.com/kagent-dev/kmcp/api/v1alpha1"
	"github.com/kagent-dev/kmcp/pkg/substrate"
)

const (
	// substrateNotReadyRequeue is the poll interval while waiting on the
	// golden snapshot or actor state transitions.
	substrateNotReadyRequeue = 15 * time.Second

	// substrateDeleteTimeout bounds how long actor deletion may take before
	// the condition surfaces a timeout (deletion keeps retrying regardless).
	substrateDeleteTimeout = 5 * time.Minute
)

// MCPServerSubstrateReconciler reconciles MCPServers with runtime=substrate:
// it generates an ate.dev ActorTemplate, creates the backing actor through
// the ate-api control plane, publishes routing coordinates in status, and
// (in managed-proxy mode) routes the server's Service through the shared
// in-cluster ingress proxy.
type MCPServerSubstrateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// APIReader reads objects without the manager cache; the default
	// WorkerPool may live in a namespace outside the watched set.
	APIReader client.Reader
	Ate       substrate.ControlAPI
	Defaults  substrate.Defaults
	Ingress   substrate.IngressConfig
	Recorder  events.EventRecorder
}

// +kubebuilder:rbac:groups=ate.dev,resources=actortemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ate.dev,resources=actortemplates/status,verbs=get
// +kubebuilder:rbac:groups=ate.dev,resources=workerpools,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=discovery.k8s.io,resources=endpointslices,verbs=get;list;watch;create;update;patch;delete

func (r *MCPServerSubstrateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("mcpserver-substrate")

	server := &kagentdevv1alpha1.MCPServer{}
	if err := r.Get(ctx, req.NamespacedName, server); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !server.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, server)
	}

	if effectiveRuntime(server) != kagentdevv1alpha1.MCPServerRuntimeSubstrate {
		// Runtime flipped away from substrate: clean up the actor before
		// handing the resource over to the kubernetes controller.
		if controllerutil.ContainsFinalizer(server, substrate.FinalizerName) {
			return r.reconcileRuntimeFlip(ctx, server)
		}
		return ctrl.Result{}, nil
	}

	// A finalizer update does not change the generation, so an event would be
	// filtered by the predicates; continue reconciling in the same pass.
	if controllerutil.AddFinalizer(server, substrate.FinalizerName) {
		if err := r.Update(ctx, server); err != nil {
			return ctrl.Result{}, fmt.Errorf("add finalizer: %w", err)
		}
	}

	server.Status.ObservedGeneration = server.Generation

	// Terminal configuration errors: report and wait for a spec change.
	if err := substrate.ValidateForSubstrate(server); err != nil {
		setAcceptedCondition(server, false, kagentdevv1alpha1.MCPServerReasonUnsupportedField, err.Error())
		setReadyCondition(server, false, kagentdevv1alpha1.MCPServerReasonNotAvailable,
			"Configuration is not valid for the substrate runtime")
		r.updateStatus(ctx, server)
		return ctrl.Result{}, nil
	}
	setAcceptedCondition(server, true, kagentdevv1alpha1.MCPServerReasonAccepted,
		"MCPServer configuration is valid for the substrate runtime")
	if ignored := substrate.IgnoredFields(server); len(ignored) > 0 && r.Recorder != nil {
		r.Recorder.Eventf(server, nil, corev1.EventTypeNormal, "SubstrateIgnoredFields", "Reconcile",
			"spec.deployment fields ignored on the substrate runtime: %s", strings.Join(ignored, ", "))
	}

	workerPool, err := r.resolveWorkerPool(ctx, server)
	if err != nil {
		setResolvedRefsCondition(server, false, kagentdevv1alpha1.MCPServerReasonWorkerPoolNotFound, err.Error())
		setReadyCondition(server, false, kagentdevv1alpha1.MCPServerReasonNotAvailable, "WorkerPool not resolved")
		r.updateStatus(ctx, server)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	templateReady, result, err := r.ensureActorTemplate(ctx, server, workerPool)
	if err != nil || !templateReady {
		return result, err
	}

	actorReady, result, err := r.ensureActor(ctx, server)
	if err != nil || !actorReady {
		return result, err
	}

	proxyReady, err := r.reconcileIngress(ctx, server)
	if err != nil {
		setProgrammedCondition(server, false, kagentdevv1alpha1.MCPServerReasonDeploymentFailed, err.Error())
		r.updateStatus(ctx, server)
		return ctrl.Result{}, err
	}
	setProgrammedCondition(server, true, kagentdevv1alpha1.MCPServerReasonProgrammed,
		"Substrate resources created successfully")

	if r.Ingress.Mode == substrate.IngressModeManagedProxy && !proxyReady {
		setReadyCondition(server, false, kagentdevv1alpha1.MCPServerReasonPodsNotReady,
			"Shared ingress proxy has no ready pods")
		r.updateStatus(ctx, server)
		return ctrl.Result{RequeueAfter: substrateNotReadyRequeue}, nil
	}

	setReadyCondition(server, true, kagentdevv1alpha1.MCPServerReasonAvailable, "Substrate actor is routable")
	r.updateStatus(ctx, server)
	logger.V(1).Info("substrate MCPServer ready", "actorID", substrate.ActorID(server))
	return ctrl.Result{}, nil
}

// resolveWorkerPool returns the WorkerPool the actor should run on: the spec
// reference (same namespace) when set, the controller default otherwise.
func (r *MCPServerSubstrateReconciler) resolveWorkerPool(
	ctx context.Context,
	server *kagentdevv1alpha1.MCPServer,
) (types.NamespacedName, error) {
	key := r.Defaults.DefaultWorkerPool
	if server.Spec.Substrate != nil && server.Spec.Substrate.WorkerPoolRef != nil &&
		server.Spec.Substrate.WorkerPoolRef.Name != "" {
		key = types.NamespacedName{Namespace: server.Namespace, Name: server.Spec.Substrate.WorkerPoolRef.Name}
	}
	if key.Name == "" {
		return types.NamespacedName{}, fmt.Errorf(
			"no WorkerPool: spec.substrate.workerPoolRef is unset and the controller has no default WorkerPool configured")
	}
	if key.Namespace == "" {
		key.Namespace = server.Namespace
	}
	var pool atev1alpha1.WorkerPool
	if err := r.APIReader.Get(ctx, key, &pool); err != nil {
		if apierrors.IsNotFound(err) {
			return types.NamespacedName{}, fmt.Errorf("WorkerPool %s not found", key)
		}
		return types.NamespacedName{}, fmt.Errorf("get WorkerPool %s: %w", key, err)
	}
	return key, nil
}

// ensureActorTemplate creates or updates the generated ActorTemplate and
// reports whether its golden snapshot is ready.
func (r *MCPServerSubstrateReconciler) ensureActorTemplate(
	ctx context.Context,
	server *kagentdevv1alpha1.MCPServer,
	workerPool types.NamespacedName,
) (bool, ctrl.Result, error) {
	tb := &substrate.TemplateBuilder{Client: r.Client, Scheme: r.Scheme, Defaults: r.Defaults}
	desired, err := tb.Build(ctx, server, workerPool)
	if err != nil {
		var notFound *substrate.SecretNotFoundError
		if errors.As(err, &notFound) {
			setResolvedRefsCondition(server, false, kagentdevv1alpha1.MCPServerReasonSecretNotFound, err.Error())
			setReadyCondition(server, false, kagentdevv1alpha1.MCPServerReasonNotAvailable, "Secret references not resolved")
			r.updateStatus(ctx, server)
			return false, ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return false, ctrl.Result{}, err
	}
	setResolvedRefsCondition(server, true, kagentdevv1alpha1.MCPServerReasonResolvedRefs,
		"All references resolved successfully")

	existing := &atev1alpha1.ActorTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: desired.Name, Namespace: desired.Namespace},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, existing, func() error {
		existing.Labels = desired.Labels
		existing.Annotations = desired.Annotations
		existing.OwnerReferences = desired.OwnerReferences
		existing.Spec = desired.Spec
		return nil
	}); err != nil {
		setProgrammedCondition(server, false, kagentdevv1alpha1.MCPServerReasonDeploymentFailed,
			fmt.Sprintf("Failed to reconcile ActorTemplate: %v", err))
		r.updateStatus(ctx, server)
		return false, ctrl.Result{}, fmt.Errorf("reconcile ActorTemplate %s/%s: %w", desired.Namespace, desired.Name, err)
	}

	r.fillSubstrateStatus(server)

	switch existing.Status.Phase {
	case atev1alpha1.PhaseReady:
		setCondition(server, kagentdevv1alpha1.MCPServerConditionActorTemplateReady, metav1.ConditionTrue,
			kagentdevv1alpha1.MCPServerReasonReady, "ActorTemplate golden snapshot is ready")
		return true, ctrl.Result{}, nil
	case atev1alpha1.PhaseFailed:
		setCondition(server, kagentdevv1alpha1.MCPServerConditionActorTemplateReady, metav1.ConditionFalse,
			kagentdevv1alpha1.MCPServerReasonActorTemplateFailed, "ActorTemplate failed to bake a golden snapshot")
		setReadyCondition(server, false, kagentdevv1alpha1.MCPServerReasonNotAvailable, "ActorTemplate failed")
		r.updateStatus(ctx, server)
		return false, ctrl.Result{RequeueAfter: substrateNotReadyRequeue}, nil
	default:
		setCondition(server, kagentdevv1alpha1.MCPServerConditionActorTemplateReady, metav1.ConditionFalse,
			kagentdevv1alpha1.MCPServerReasonActorTemplatePending, "Waiting for the ActorTemplate golden snapshot")
		setCondition(server, kagentdevv1alpha1.MCPServerConditionActorReady, metav1.ConditionFalse,
			kagentdevv1alpha1.MCPServerReasonActorNotCreated, "Waiting for the ActorTemplate before creating the actor")
		setReadyCondition(server, false, kagentdevv1alpha1.MCPServerReasonNotAvailable, "ActorTemplate is not ready yet")
		r.updateStatus(ctx, server)
		return false, ctrl.Result{RequeueAfter: substrateNotReadyRequeue}, nil
	}
}

// ensureActor creates the backing actor if missing and reports whether it is
// routable. A suspended actor is routable: the atenet router buffers the
// request and resumes it on demand, which is the scale-from-zero path.
func (r *MCPServerSubstrateReconciler) ensureActor(
	ctx context.Context,
	server *kagentdevv1alpha1.MCPServer,
) (bool, ctrl.Result, error) {
	actorID := substrate.ActorID(server)
	actor, err := r.Ate.GetActor(ctx, actorID)
	if err != nil {
		if grpcstatus.Code(err) != codes.NotFound {
			setCondition(server, kagentdevv1alpha1.MCPServerConditionActorReady, metav1.ConditionUnknown,
				kagentdevv1alpha1.MCPServerReasonActorCreateFailed, fmt.Sprintf("Failed to query actor: %v", err))
			r.updateStatus(ctx, server)
			return false, ctrl.Result{}, fmt.Errorf("get actor %q: %w", actorID, err)
		}
		actor, err = r.Ate.CreateActor(ctx, actorID, server.Namespace, substrate.ActorTemplateName(server))
		if err != nil {
			setCondition(server, kagentdevv1alpha1.MCPServerConditionActorReady, metav1.ConditionFalse,
				kagentdevv1alpha1.MCPServerReasonActorCreateFailed, fmt.Sprintf("Failed to create actor: %v", err))
			r.updateStatus(ctx, server)
			return false, ctrl.Result{}, fmt.Errorf("create actor %q: %w", actorID, err)
		}
	}

	switch actor.GetStatus() {
	case ateapipb.Actor_STATUS_RUNNING:
		setCondition(server, kagentdevv1alpha1.MCPServerConditionActorReady, metav1.ConditionTrue,
			kagentdevv1alpha1.MCPServerReasonActorRunning, "Actor is running")
	case ateapipb.Actor_STATUS_RESUMING:
		setCondition(server, kagentdevv1alpha1.MCPServerConditionActorReady, metav1.ConditionTrue,
			kagentdevv1alpha1.MCPServerReasonActorResuming, "Actor is resuming")
	case ateapipb.Actor_STATUS_SUSPENDED, ateapipb.Actor_STATUS_UNSPECIFIED:
		// Suspended is the normal serverless idle state; the router resumes
		// the actor when a request arrives.
		setCondition(server, kagentdevv1alpha1.MCPServerConditionActorReady, metav1.ConditionTrue,
			kagentdevv1alpha1.MCPServerReasonActorSuspended, "Actor is suspended and will resume on demand")
	case ateapipb.Actor_STATUS_SUSPENDING:
		setCondition(server, kagentdevv1alpha1.MCPServerConditionActorReady, metav1.ConditionFalse,
			kagentdevv1alpha1.MCPServerReasonActorSuspending, "Actor is suspending")
		setReadyCondition(server, false, kagentdevv1alpha1.MCPServerReasonNotAvailable, "Actor is suspending")
		r.updateStatus(ctx, server)
		return false, ctrl.Result{RequeueAfter: substrateNotReadyRequeue}, nil
	}
	return true, ctrl.Result{}, nil
}

// reconcileIngress renders the installation's ingress mode for this server.
// In managed-proxy mode it upserts the selector-less Service and the
// EndpointSlice carrying the shared proxy pod IPs, rebuilds the shared route
// ConfigMap, and reports whether the proxy has at least one ready pod.
func (r *MCPServerSubstrateReconciler) reconcileIngress(
	ctx context.Context,
	server *kagentdevv1alpha1.MCPServer,
) (bool, error) {
	// Earlier iterations deployed a proxy per MCPServer; remove leftovers.
	if err := r.deleteLegacyPerServerProxy(ctx, server); err != nil {
		return false, err
	}

	if r.Ingress.Mode != substrate.IngressModeManagedProxy {
		if err := r.deleteServerIngressObjects(ctx, server); err != nil {
			return false, err
		}
		if server.Status.Substrate != nil {
			server.Status.Substrate.ProxyEndpoint = ""
		}
		return true, nil
	}

	proxyPodIPs, err := r.readyProxyPodIPs(ctx)
	if err != nil {
		return false, err
	}

	service, slice, err := substrate.BuildServerIngressObjects(server, r.Scheme, r.Ingress.Port(), proxyPodIPs)
	if err != nil {
		return false, err
	}

	existingService := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: service.Name, Namespace: service.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, existingService, func() error {
		// Refuse to adopt a Service kmcp does not manage (e.g. a leftover
		// from the kubernetes runtime keeps its role-less labels).
		if existingService.ResourceVersion != "" &&
			existingService.Labels["app.kubernetes.io/managed-by"] != "kmcp" {
			return fmt.Errorf("service %s/%s exists and is not managed by kmcp", service.Namespace, service.Name)
		}
		existingService.Labels = service.Labels
		existingService.OwnerReferences = service.OwnerReferences
		existingService.Spec.Selector = nil
		existingService.Spec.Ports = service.Spec.Ports
		return nil
	}); err != nil {
		return false, fmt.Errorf("reconcile Service %s/%s: %w", service.Namespace, service.Name, err)
	}

	existingSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{Name: slice.Name, Namespace: slice.Namespace},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, existingSlice, func() error {
		existingSlice.Labels = slice.Labels
		existingSlice.OwnerReferences = slice.OwnerReferences
		existingSlice.AddressType = slice.AddressType
		existingSlice.Ports = slice.Ports
		existingSlice.Endpoints = slice.Endpoints
		return nil
	}); err != nil {
		return false, fmt.Errorf("reconcile EndpointSlice %s/%s: %w", slice.Namespace, slice.Name, err)
	}

	if err := r.reconcileSharedProxyConfig(ctx); err != nil {
		return false, err
	}

	if server.Status.Substrate != nil {
		server.Status.Substrate.ProxyEndpoint = substrate.ProxyEndpoint(server)
	}
	return len(proxyPodIPs) > 0, nil
}

// readyProxyPodIPs lists the shared ingress proxy pods (by role label, in the
// kmcp installation namespace) and returns the IPs of the ready ones, sorted.
func (r *MCPServerSubstrateReconciler) readyProxyPodIPs(ctx context.Context) ([]string, error) {
	pods := &corev1.PodList{}
	if err := r.List(ctx, pods,
		client.InNamespace(r.Ingress.ProxyNamespace),
		client.MatchingLabels{substrate.RoleLabelKey: substrate.RoleIngressProxy},
	); err != nil {
		return nil, fmt.Errorf("list ingress proxy pods: %w", err)
	}
	ips := make([]string, 0, len(pods.Items))
	for i := range pods.Items {
		pod := &pods.Items[i]
		if pod.Status.PodIP == "" || pod.DeletionTimestamp != nil {
			continue
		}
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				ips = append(ips, pod.Status.PodIP)
				break
			}
		}
	}
	sort.Strings(ips)
	return ips, nil
}

// reconcileSharedProxyConfig rebuilds the shared proxy route ConfigMap from
// every live substrate MCPServer. Each reconcile renders the whole file, so
// route removal (deletion, runtime flips) is just another rebuild.
func (r *MCPServerSubstrateReconciler) reconcileSharedProxyConfig(ctx context.Context) error {
	servers := &kagentdevv1alpha1.MCPServerList{}
	if err := r.List(ctx, servers); err != nil {
		return fmt.Errorf("list MCPServers for proxy config: %w", err)
	}

	routes := make([]substrate.ProxyRoute, 0, len(servers.Items))
	for i := range servers.Items {
		server := &servers.Items[i]
		if effectiveRuntime(server) != kagentdevv1alpha1.MCPServerRuntimeSubstrate ||
			!server.DeletionTimestamp.IsZero() ||
			substrate.ValidateForSubstrate(server) != nil {
			continue
		}
		routes = append(routes, substrate.ProxyRoute{
			Name:      server.Name,
			Namespace: server.Namespace,
			ActorHost: substrate.ActorHost(substrate.ActorID(server), r.Defaults.HostSuffix()),
		})
	}

	configYAML, err := substrate.BuildSharedProxyConfig(routes, r.Defaults, r.Ingress.Port())
	if err != nil {
		return err
	}

	meta := substrate.SharedProxyConfigMapMeta(r.Ingress)
	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: meta.Name, Namespace: meta.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		configMap.Labels = meta.Labels
		configMap.Data = map[string]string{substrate.ProxyConfigKey: configYAML}
		return nil
	}); err != nil {
		return fmt.Errorf("reconcile shared proxy ConfigMap %s/%s: %w", meta.Namespace, meta.Name, err)
	}
	return nil
}

// deleteServerIngressObjects removes the per-server Service and EndpointSlice
// (mode switched to none, or the runtime flipped away from substrate). Only
// objects carrying the ingress proxy role label are touched — a runtime flip
// must not delete the kubernetes runtime's Service.
func (r *MCPServerSubstrateReconciler) deleteServerIngressObjects(
	ctx context.Context,
	server *kagentdevv1alpha1.MCPServer,
) error {
	objects := []client.Object{&corev1.Service{}, &discoveryv1.EndpointSlice{}}
	names := []string{server.Name, substrate.EndpointSliceName(server)}
	for i, obj := range objects {
		key := client.ObjectKey{Name: names[i], Namespace: server.Namespace}
		if err := r.Get(ctx, key, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		if obj.GetLabels()[substrate.RoleLabelKey] != substrate.RoleIngressProxy {
			continue
		}
		if err := r.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// deleteLegacyPerServerProxy removes the per-MCPServer proxy Deployment and
// ConfigMap created by earlier iterations (the per-server Service is adopted
// in place by reconcileIngress). Only role-labeled objects are touched.
func (r *MCPServerSubstrateReconciler) deleteLegacyPerServerProxy(
	ctx context.Context,
	server *kagentdevv1alpha1.MCPServer,
) error {
	key := client.ObjectKey{Name: server.Name, Namespace: server.Namespace}
	for _, obj := range []client.Object{&appsv1.Deployment{}, &corev1.ConfigMap{}} {
		if err := r.Get(ctx, key, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		if obj.GetLabels()[substrate.RoleLabelKey] != substrate.RoleIngressProxy {
			continue
		}
		if err := r.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// fillSubstrateStatus publishes the routing coordinates any ingress needs to
// reach the actor.
func (r *MCPServerSubstrateReconciler) fillSubstrateStatus(server *kagentdevv1alpha1.MCPServer) {
	actorID := substrate.ActorID(server)
	status := &kagentdevv1alpha1.MCPServerSubstrateStatus{
		ActorID:          actorID,
		ActorHost:        substrate.ActorHost(actorID, r.Defaults.HostSuffix()),
		RouterURL:        r.Defaults.RouterURL(),
		MCPPath:          substrate.MCPPath,
		ActorTemplateRef: substrate.ActorTemplateName(server),
	}
	if server.Status.Substrate != nil {
		status.ProxyEndpoint = server.Status.Substrate.ProxyEndpoint
	}
	server.Status.Substrate = status
}

// reconcileRuntimeFlip handles runtime changing from substrate to kubernetes
// on a live resource: the actor is deleted before the finalizer is dropped so
// the kubernetes controller starts from a clean slate.
func (r *MCPServerSubstrateReconciler) reconcileRuntimeFlip(
	ctx context.Context,
	server *kagentdevv1alpha1.MCPServer,
) (ctrl.Result, error) {
	done, err := r.cleanupSubstrate(ctx, server)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !done {
		return ctrl.Result{RequeueAfter: substrateNotReadyRequeue}, nil
	}
	// The owner is not being deleted, so the generated objects are not
	// garbage-collected on a flip: drop the server's route from the shared
	// proxy config and delete the ingress objects explicitly. The
	// EndpointSlice in particular must go before the kubernetes runtime
	// recreates the Service with a selector, or its proxy IPs would mix into
	// the real endpoints.
	if r.Ingress.Mode == substrate.IngressModeManagedProxy {
		if err := r.reconcileSharedProxyConfig(ctx); err != nil {
			return ctrl.Result{}, err
		}
	}
	if err := r.deleteServerIngressObjects(ctx, server); err != nil {
		return ctrl.Result{}, err
	}
	// The generated ActorTemplate is not garbage-collected on a flip (the
	// owner is not being deleted), so remove it explicitly.
	tmpl := &atev1alpha1.ActorTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: substrate.ActorTemplateName(server), Namespace: server.Namespace},
	}
	if err := r.Delete(ctx, tmpl); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("delete generated ActorTemplate: %w", err)
	}
	server.Status.Substrate = nil
	r.updateStatus(ctx, server)
	if controllerutil.RemoveFinalizer(server, substrate.FinalizerName) {
		if err := r.Update(ctx, server); err != nil {
			return ctrl.Result{}, fmt.Errorf("remove finalizer: %w", err)
		}
	}
	return ctrl.Result{}, nil
}

// reconcileDelete drives the two-phase deletion: remove the runtime actor and
// the golden actor through ate-api, then drop the finalizer so the generated
// ActorTemplate and proxy objects are garbage-collected via owner references.
func (r *MCPServerSubstrateReconciler) reconcileDelete(
	ctx context.Context,
	server *kagentdevv1alpha1.MCPServer,
) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(server, substrate.FinalizerName) {
		return ctrl.Result{}, nil
	}

	if time.Since(server.DeletionTimestamp.Time) > substrateDeleteTimeout {
		setReadyCondition(server, false, kagentdevv1alpha1.MCPServerReasonDeleteTimeout,
			"Substrate actor deletion is taking longer than expected")
		r.updateStatus(ctx, server)
	}

	done, err := r.cleanupSubstrate(ctx, server)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !done {
		setReadyCondition(server, false, kagentdevv1alpha1.MCPServerReasonActorDeleting, "Substrate actor is being deleted")
		r.updateStatus(ctx, server)
		return ctrl.Result{RequeueAfter: substrateNotReadyRequeue}, nil
	}

	// Drop the server's route from the shared proxy config (the rebuild
	// skips servers with a deletion timestamp); the Service and
	// EndpointSlice are garbage-collected through their owner references.
	if r.Ingress.Mode == substrate.IngressModeManagedProxy {
		if err := r.reconcileSharedProxyConfig(ctx); err != nil {
			return ctrl.Result{}, err
		}
	}

	if controllerutil.RemoveFinalizer(server, substrate.FinalizerName) {
		if err := r.Update(ctx, server); err != nil {
			return ctrl.Result{}, fmt.Errorf("remove finalizer: %w", err)
		}
	}
	return ctrl.Result{}, nil
}

// cleanupSubstrate deletes the runtime actor and the golden actor recorded on
// the generated ActorTemplate. It is reconcile-safe: it returns done=false
// while deletions are still in flight.
func (r *MCPServerSubstrateReconciler) cleanupSubstrate(
	ctx context.Context,
	server *kagentdevv1alpha1.MCPServer,
) (bool, error) {
	actorDone, err := r.deleteActor(ctx, substrate.ActorID(server))
	if err != nil {
		return false, err
	}

	goldenDone := true
	tmpl := &atev1alpha1.ActorTemplate{}
	key := client.ObjectKey{Name: substrate.ActorTemplateName(server), Namespace: server.Namespace}
	if err := r.Get(ctx, key, tmpl); err != nil {
		if !apierrors.IsNotFound(err) {
			return false, fmt.Errorf("get generated ActorTemplate: %w", err)
		}
	} else if tmpl.Status.GoldenActorID != "" {
		goldenDone, err = r.deleteActor(ctx, tmpl.Status.GoldenActorID)
		if err != nil {
			return false, err
		}
	}

	return actorDone && goldenDone, nil
}

// deleteActor drives an actor towards deletion, performing at most one
// mutating ate-api step per call: running actors are suspended first (ate-api
// only deletes suspended actors), then deleted. It returns done=true once the
// actor no longer exists in the control plane; callers requeue until then.
func (r *MCPServerSubstrateReconciler) deleteActor(ctx context.Context, actorID string) (bool, error) {
	actor, err := r.Ate.GetActor(ctx, actorID)
	if err != nil {
		if grpcstatus.Code(err) == codes.NotFound {
			return true, nil
		}
		return false, fmt.Errorf("get actor %q: %w", actorID, err)
	}

	switch actor.GetStatus() {
	case ateapipb.Actor_STATUS_SUSPENDED, ateapipb.Actor_STATUS_UNSPECIFIED:
		if err := r.Ate.DeleteActor(ctx, actorID); err != nil && grpcstatus.Code(err) != codes.NotFound {
			return false, fmt.Errorf("delete actor %q: %w", actorID, err)
		}
	case ateapipb.Actor_STATUS_RUNNING, ateapipb.Actor_STATUS_RESUMING:
		if err := r.Ate.SuspendActor(ctx, actorID); err != nil && grpcstatus.Code(err) != codes.NotFound {
			return false, fmt.Errorf("suspend actor %q before deletion: %w", actorID, err)
		}
	case ateapipb.Actor_STATUS_SUSPENDING:
		// Wait for the suspension to settle.
	}
	// Deletion is asynchronous; report done only once GetActor misses.
	return false, nil
}

func (r *MCPServerSubstrateReconciler) updateStatus(ctx context.Context, server *kagentdevv1alpha1.MCPServer) {
	if err := r.Status().Update(ctx, server); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update MCPServer status")
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *MCPServerSubstrateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	b := ctrl.NewControllerManagedBy(mgr).
		For(&kagentdevv1alpha1.MCPServer{}, builder.WithPredicates(
			mcpServerRuntimePredicate(kagentdevv1alpha1.MCPServerRuntimeSubstrate),
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.LabelChangedPredicate{},
			),
		)).
		Owns(&atev1alpha1.ActorTemplate{}, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Owns(&appsv1.Deployment{}, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Owns(&corev1.Service{}, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Owns(&corev1.ConfigMap{}, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Named("mcpserver-substrate")

	if r.Ingress.Mode == substrate.IngressModeManagedProxy {
		// Proxy pod churn must propagate into every server's EndpointSlice,
		// and drift in the shared route ConfigMap (it has no owner) must be
		// repaired; both fan out to all substrate MCPServers.
		enqueueAll := handler.EnqueueRequestsFromMapFunc(r.substrateServerRequests)
		b = b.
			Owns(&discoveryv1.EndpointSlice{}, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
			Watches(&corev1.Pod{}, enqueueAll, builder.WithPredicates(r.proxyPodPredicate())).
			Watches(&corev1.ConfigMap{}, enqueueAll, builder.WithPredicates(r.sharedConfigMapPredicate()))

		// The shared proxy mounts the route ConfigMap, so the proxy pods cannot
		// start until it exists. Reconciles only create it once an MCPServer
		// appears, which would wedge a fresh install with zero servers. Ensure
		// an (empty) ConfigMap on startup so the proxy comes up immediately.
		if err := mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
			if err := r.ensureSharedProxyConfig(ctx); err != nil {
				log.FromContext(ctx).Error(err, "Failed to ensure shared proxy ConfigMap on startup")
			}
			return nil
		})); err != nil {
			return err
		}
	}

	return b.Complete(r)
}

// ensureSharedProxyConfig creates the shared proxy route ConfigMap with an
// empty configuration if it does not already exist. This lets the proxy pods
// mount it and start on a fresh install before any MCPServer is created;
// reconciles keep its contents current thereafter. The read uses APIReader so
// it does not depend on the manager cache being populated at startup.
func (r *MCPServerSubstrateReconciler) ensureSharedProxyConfig(ctx context.Context) error {
	meta := substrate.SharedProxyConfigMapMeta(r.Ingress)
	existing := &corev1.ConfigMap{}
	switch err := r.APIReader.Get(ctx, client.ObjectKey{Name: meta.Name, Namespace: meta.Namespace}, existing); {
	case err == nil:
		return nil // already present; the reconciler owns its contents
	case !apierrors.IsNotFound(err):
		return fmt.Errorf("get shared proxy ConfigMap %s/%s: %w", meta.Namespace, meta.Name, err)
	}

	configYAML, err := substrate.BuildSharedProxyConfig(nil, r.Defaults, r.Ingress.Port())
	if err != nil {
		return err
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: meta.Name, Namespace: meta.Namespace, Labels: meta.Labels},
		Data:       map[string]string{substrate.ProxyConfigKey: configYAML},
	}
	if err := r.Create(ctx, configMap); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create shared proxy ConfigMap %s/%s: %w", meta.Namespace, meta.Name, err)
	}
	return nil
}

// substrateServerRequests maps a shared-infrastructure event (proxy pod or
// shared ConfigMap) to a reconcile request per substrate MCPServer.
func (r *MCPServerSubstrateReconciler) substrateServerRequests(ctx context.Context, _ client.Object) []ctrl.Request {
	servers := &kagentdevv1alpha1.MCPServerList{}
	if err := r.List(ctx, servers); err != nil {
		log.FromContext(ctx).Error(err, "Failed to list MCPServers for proxy event fan-out")
		return nil
	}
	requests := make([]ctrl.Request, 0, len(servers.Items))
	for i := range servers.Items {
		server := &servers.Items[i]
		if effectiveRuntime(server) != kagentdevv1alpha1.MCPServerRuntimeSubstrate {
			continue
		}
		requests = append(requests, ctrl.Request{
			NamespacedName: types.NamespacedName{Namespace: server.Namespace, Name: server.Name},
		})
	}
	return requests
}

// proxyPodPredicate matches the shared ingress proxy pods.
func (r *MCPServerSubstrateReconciler) proxyPodPredicate() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		return obj.GetNamespace() == r.Ingress.ProxyNamespace &&
			obj.GetLabels()[substrate.RoleLabelKey] == substrate.RoleIngressProxy
	})
}

// sharedConfigMapPredicate matches the shared proxy route ConfigMap.
func (r *MCPServerSubstrateReconciler) sharedConfigMapPredicate() predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		return obj.GetNamespace() == r.Ingress.ProxyNamespace &&
			obj.GetName() == r.Ingress.ConfigMapName()
	})
}
