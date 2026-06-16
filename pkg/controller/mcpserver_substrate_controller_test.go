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
	"strings"
	"testing"
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
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	mcpv1 "github.com/kagent-dev/kmcp/api/v1alpha1"
	"github.com/kagent-dev/kmcp/pkg/substrate"
)

const pinnedTestImage = "ghcr.io/example/mcp@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// fakeControlAPI is an in-memory ate-api control plane.
type fakeControlAPI struct {
	actors        map[string]*ateapipb.Actor
	createCalls   []string
	deleteCalls   []string
	defaultStatus ateapipb.Actor_Status
}

func newFakeControlAPI() *fakeControlAPI {
	return &fakeControlAPI{
		actors:        map[string]*ateapipb.Actor{},
		defaultStatus: ateapipb.Actor_STATUS_RUNNING,
	}
}

func (f *fakeControlAPI) GetActor(_ context.Context, actorID string) (*ateapipb.Actor, error) {
	actor, ok := f.actors[actorID]
	if !ok {
		return nil, grpcstatus.Error(codes.NotFound, "actor not found")
	}
	return actor, nil
}

func (f *fakeControlAPI) CreateActor(_ context.Context, actorID, _, _ string) (*ateapipb.Actor, error) {
	f.createCalls = append(f.createCalls, actorID)
	actor := &ateapipb.Actor{ActorId: actorID, Status: f.defaultStatus}
	f.actors[actorID] = actor
	return actor, nil
}

func (f *fakeControlAPI) SuspendActor(_ context.Context, actorID string) error {
	actor, ok := f.actors[actorID]
	if !ok {
		return grpcstatus.Error(codes.NotFound, "actor not found")
	}
	actor.Status = ateapipb.Actor_STATUS_SUSPENDED
	return nil
}

func (f *fakeControlAPI) DeleteActor(_ context.Context, actorID string) error {
	actor, ok := f.actors[actorID]
	if !ok {
		return grpcstatus.Error(codes.NotFound, "actor not found")
	}
	if actor.GetStatus() != ateapipb.Actor_STATUS_SUSPENDED && actor.GetStatus() != ateapipb.Actor_STATUS_UNSPECIFIED {
		return grpcstatus.Errorf(codes.FailedPrecondition, "actor %s is not suspended", actorID)
	}
	f.deleteCalls = append(f.deleteCalls, actorID)
	delete(f.actors, actorID)
	return nil
}

func substrateTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := mcpv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := atev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return scheme
}

func substrateMCPServer() *mcpv1.MCPServer {
	return &mcpv1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "srv", Generation: 1},
		Spec: mcpv1.MCPServerSpec{
			Runtime:       mcpv1.MCPServerRuntimeSubstrate,
			Substrate:     &mcpv1.SubstrateSpec{},
			TransportType: mcpv1.TransportTypeHTTP,
			HTTPTransport: &mcpv1.HTTPTransport{TargetPort: 3001, TargetPath: "/mcp"},
			Deployment: mcpv1.MCPServerDeployment{
				Image: pinnedTestImage,
				Cmd:   "node",
				Port:  3000,
			},
		},
	}
}

func workerPool() *atev1alpha1.WorkerPool {
	return &atev1alpha1.WorkerPool{
		ObjectMeta: metav1.ObjectMeta{Namespace: "pools", Name: "wp"},
	}
}

type substrateTestEnv struct {
	client     client.Client
	ate        *fakeControlAPI
	reconciler *MCPServerSubstrateReconciler
}

const proxyNamespace = "kmcp-system"

func newSubstrateTestEnv(t *testing.T, mode substrate.IngressMode, objects ...client.Object) *substrateTestEnv {
	t.Helper()
	scheme := substrateTestScheme(t)
	kube := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(&mcpv1.MCPServer{}, &atev1alpha1.ActorTemplate{}).
		Build()
	ate := newFakeControlAPI()
	return &substrateTestEnv{
		client: kube,
		ate:    ate,
		reconciler: &MCPServerSubstrateReconciler{
			Client:    kube,
			Scheme:    scheme,
			APIReader: kube,
			Ate:       ate,
			Defaults: substrate.Defaults{
				DefaultWorkerPool: types.NamespacedName{Namespace: "pools", Name: "wp"},
			},
			Ingress: substrate.IngressConfig{
				Mode:           mode,
				ProxyNamespace: proxyNamespace,
			},
		},
	}
}

// readyProxyPod fakes one ready shared ingress proxy pod.
func readyProxyPod(name, ip string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: proxyNamespace,
			Labels:    map[string]string{substrate.RoleLabelKey: substrate.RoleIngressProxy},
		},
		Status: corev1.PodStatus{
			PodIP: ip,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
		},
	}
}

func (e *substrateTestEnv) reconcile(t *testing.T) ctrl.Result {
	t.Helper()
	result, err := e.reconciler.Reconcile(context.Background(),
		ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "ns", Name: "srv"}})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	return result
}

// reconcileSteady reconciles until no requeue is requested, with a safety bound.
func (e *substrateTestEnv) reconcileSteady(t *testing.T) {
	t.Helper()
	for i := 0; i < 5; i++ {
		if result := e.reconcile(t); result.RequeueAfter == 0 {
			return
		}
	}
}

func (e *substrateTestEnv) getServer(t *testing.T) *mcpv1.MCPServer {
	t.Helper()
	server := &mcpv1.MCPServer{}
	if err := e.client.Get(context.Background(),
		types.NamespacedName{Namespace: "ns", Name: "srv"}, server); err != nil {
		t.Fatalf("get MCPServer: %v", err)
	}
	return server
}

func (e *substrateTestEnv) markTemplateReady(t *testing.T) {
	t.Helper()
	tmpl := &atev1alpha1.ActorTemplate{}
	if err := e.client.Get(context.Background(),
		types.NamespacedName{Namespace: "ns", Name: "srv"}, tmpl); err != nil {
		t.Fatalf("get ActorTemplate: %v", err)
	}
	tmpl.Status.Phase = atev1alpha1.PhaseReady
	tmpl.Status.GoldenActorID = "golden-1"
	if err := e.client.Status().Update(context.Background(), tmpl); err != nil {
		t.Fatalf("update ActorTemplate status: %v", err)
	}
}

func condition(server *mcpv1.MCPServer, condType mcpv1.MCPServerConditionType) *metav1.Condition {
	for i := range server.Status.Conditions {
		if server.Status.Conditions[i].Type == string(condType) {
			return &server.Status.Conditions[i]
		}
	}
	return nil
}

func assertCondition(
	t *testing.T,
	server *mcpv1.MCPServer,
	condType mcpv1.MCPServerConditionType,
	status metav1.ConditionStatus,
	reason mcpv1.MCPServerConditionReason,
) {
	t.Helper()
	cond := condition(server, condType)
	if cond == nil {
		t.Fatalf("condition %s not set; conditions: %+v", condType, server.Status.Conditions)
	}
	if cond.Status != status || cond.Reason != string(reason) {
		t.Fatalf("condition %s = %s/%s, want %s/%s (message: %s)",
			condType, cond.Status, cond.Reason, status, reason, cond.Message)
	}
}

func TestSubstrateReconcileWaitsForTemplate(t *testing.T) {
	env := newSubstrateTestEnv(t, substrate.IngressModeNone, substrateMCPServer(), workerPool())

	env.reconcileSteady(t)

	server := env.getServer(t)
	if !controllerutil.ContainsFinalizer(server, substrate.FinalizerName) {
		t.Fatal("finalizer not added")
	}
	assertCondition(t, server, mcpv1.MCPServerConditionAccepted, metav1.ConditionTrue, mcpv1.MCPServerReasonAccepted)
	assertCondition(t, server, mcpv1.MCPServerConditionActorTemplateReady, metav1.ConditionFalse,
		mcpv1.MCPServerReasonActorTemplatePending)
	assertCondition(t, server, mcpv1.MCPServerConditionActorReady, metav1.ConditionFalse,
		mcpv1.MCPServerReasonActorNotCreated)
	assertCondition(t, server, mcpv1.MCPServerConditionReady, metav1.ConditionFalse, mcpv1.MCPServerReasonNotAvailable)

	// The actor must not be created before the golden snapshot is ready.
	if len(env.ate.createCalls) != 0 {
		t.Fatalf("CreateActor called before template ready: %v", env.ate.createCalls)
	}

	// The generated ActorTemplate must exist with the owner ref and label.
	tmpl := &atev1alpha1.ActorTemplate{}
	if err := env.client.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "srv"}, tmpl); err != nil {
		t.Fatalf("get ActorTemplate: %v", err)
	}
	if tmpl.Labels[substrate.MCPServerLabelKey] != "srv" {
		t.Fatalf("ActorTemplate labels = %v", tmpl.Labels)
	}
}

func TestSubstrateReconcileReadyAfterTemplate(t *testing.T) {
	env := newSubstrateTestEnv(t, substrate.IngressModeNone, substrateMCPServer(), workerPool())

	env.reconcileSteady(t)
	env.markTemplateReady(t)
	env.reconcileSteady(t)

	server := env.getServer(t)
	assertCondition(t, server, mcpv1.MCPServerConditionActorTemplateReady, metav1.ConditionTrue,
		mcpv1.MCPServerReasonReady)
	assertCondition(t, server, mcpv1.MCPServerConditionActorReady, metav1.ConditionTrue, mcpv1.MCPServerReasonActorRunning)
	assertCondition(t, server, mcpv1.MCPServerConditionProgrammed, metav1.ConditionTrue, mcpv1.MCPServerReasonProgrammed)
	assertCondition(t, server, mcpv1.MCPServerConditionReady, metav1.ConditionTrue, mcpv1.MCPServerReasonAvailable)

	if len(env.ate.createCalls) != 1 || env.ate.createCalls[0] != "mcp-ns-srv" {
		t.Fatalf("CreateActor calls = %v", env.ate.createCalls)
	}

	status := server.Status.Substrate
	if status == nil {
		t.Fatal("status.substrate not set")
	}
	if status.ActorID != "mcp-ns-srv" ||
		status.ActorHost != "mcp-ns-srv.actors.resources.substrate.ate.dev" ||
		status.RouterURL != substrate.DefaultAtenetRouterURL ||
		status.MCPPath != "/mcp" ||
		status.ActorTemplateRef != "srv" {
		t.Fatalf("status.substrate = %+v", status)
	}
	if status.ProxyEndpoint != "" {
		t.Fatalf("proxy endpoint set with proxy disabled: %q", status.ProxyEndpoint)
	}

	// No proxy objects without the ingress proxy.
	deployment := &appsv1.Deployment{}
	err := env.client.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "srv"}, deployment)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected no proxy deployment, got err=%v", err)
	}
}

func TestSubstrateSuspendedActorIsReady(t *testing.T) {
	env := newSubstrateTestEnv(t, substrate.IngressModeNone, substrateMCPServer(), workerPool())
	env.ate.defaultStatus = ateapipb.Actor_STATUS_SUSPENDED

	env.reconcileSteady(t)
	env.markTemplateReady(t)
	env.reconcileSteady(t)

	server := env.getServer(t)
	assertCondition(t, server, mcpv1.MCPServerConditionActorReady, metav1.ConditionTrue,
		mcpv1.MCPServerReasonActorSuspended)
	assertCondition(t, server, mcpv1.MCPServerConditionReady, metav1.ConditionTrue, mcpv1.MCPServerReasonAvailable)
}

func TestSubstrateManagedProxyIngress(t *testing.T) {
	env := newSubstrateTestEnv(t, substrate.IngressModeManagedProxy, substrateMCPServer(), workerPool())

	env.reconcileSteady(t)
	env.markTemplateReady(t)
	env.reconcileSteady(t)

	// No ready proxy pods yet: the ingress objects exist but Ready is false.
	server := env.getServer(t)
	assertCondition(t, server, mcpv1.MCPServerConditionReady, metav1.ConditionFalse, mcpv1.MCPServerReasonPodsNotReady)
	if server.Status.Substrate.ProxyEndpoint != "http://srv.ns.svc:3000/mcp" {
		t.Fatalf("proxy endpoint = %q", server.Status.Substrate.ProxyEndpoint)
	}

	service := &corev1.Service{}
	serviceKey := types.NamespacedName{Namespace: "ns", Name: "srv"}
	if err := env.client.Get(context.Background(), serviceKey, service); err != nil {
		t.Fatalf("get service: %v", err)
	}
	if service.Spec.Selector != nil {
		t.Fatalf("service must be selector-less, got %v", service.Spec.Selector)
	}

	slice := &discoveryv1.EndpointSlice{}
	sliceKey := types.NamespacedName{Namespace: "ns", Name: "srv-proxy"}
	if err := env.client.Get(context.Background(), sliceKey, slice); err != nil {
		t.Fatalf("get EndpointSlice: %v", err)
	}
	if len(slice.Endpoints) != 0 {
		t.Fatalf("expected no endpoints without ready proxy pods, got %+v", slice.Endpoints)
	}

	// The shared ConfigMap carries the server's route.
	configMap := &corev1.ConfigMap{}
	configMapKey := types.NamespacedName{Namespace: proxyNamespace, Name: substrate.DefaultProxyConfigMapName}
	if err := env.client.Get(context.Background(), configMapKey, configMap); err != nil {
		t.Fatalf("get shared proxy ConfigMap: %v", err)
	}
	if !strings.Contains(configMap.Data[substrate.ProxyConfigKey], "srv.ns.svc.cluster.local") {
		t.Fatalf("shared proxy config missing route hostnames:\n%s", configMap.Data[substrate.ProxyConfigKey])
	}

	// A ready proxy pod appears: the slice picks up its IP, the server is Ready.
	if err := env.client.Create(context.Background(), readyProxyPod("proxy-1", "10.244.1.17")); err != nil {
		t.Fatalf("create proxy pod: %v", err)
	}
	env.reconcileSteady(t)

	server = env.getServer(t)
	assertCondition(t, server, mcpv1.MCPServerConditionReady, metav1.ConditionTrue, mcpv1.MCPServerReasonAvailable)
	if err := env.client.Get(context.Background(), sliceKey, slice); err != nil {
		t.Fatalf("get EndpointSlice: %v", err)
	}
	if len(slice.Endpoints) != 1 || slice.Endpoints[0].Addresses[0] != "10.244.1.17" {
		t.Fatalf("slice endpoints = %+v", slice.Endpoints)
	}

	// Deleting the server removes its route from the shared config.
	if err := env.client.Delete(context.Background(), env.getServer(t)); err != nil {
		t.Fatalf("delete MCPServer: %v", err)
	}
	env.ate.actors["golden-1"] = &ateapipb.Actor{ActorId: "golden-1", Status: ateapipb.Actor_STATUS_SUSPENDED}
	for i := 0; i < 6; i++ {
		got := &mcpv1.MCPServer{}
		if apierrors.IsNotFound(env.client.Get(context.Background(), serviceKey, got)) {
			break
		}
		env.reconcile(t)
	}
	if err := env.client.Get(context.Background(), configMapKey, configMap); err != nil {
		t.Fatalf("get shared proxy ConfigMap after delete: %v", err)
	}
	if strings.Contains(configMap.Data[substrate.ProxyConfigKey], "srv.ns") {
		t.Fatalf("route not removed from shared proxy config:\n%s", configMap.Data[substrate.ProxyConfigKey])
	}
}

func TestSubstrateManagedProxyAdoptsLegacyObjects(t *testing.T) {
	// Earlier iterations deployed a proxy per MCPServer; reconciling must
	// remove the role-labeled Deployment/ConfigMap and convert the Service.
	legacyLabels := map[string]string{
		"app.kubernetes.io/managed-by": "kmcp",
		substrate.RoleLabelKey:         substrate.RoleIngressProxy,
	}
	legacyDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "srv", Labels: legacyLabels},
	}
	legacyConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "srv", Labels: legacyLabels},
	}
	legacyService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "srv", Labels: legacyLabels},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app.kubernetes.io/name": "srv"},
			Ports:    []corev1.ServicePort{{Name: "http", Port: 3000}},
		},
	}

	env := newSubstrateTestEnv(t, substrate.IngressModeManagedProxy,
		substrateMCPServer(), workerPool(), legacyDeployment, legacyConfigMap, legacyService,
		readyProxyPod("proxy-1", "10.244.1.17"))

	env.reconcileSteady(t)
	env.markTemplateReady(t)
	env.reconcileSteady(t)

	deployment := &appsv1.Deployment{}
	err := env.client.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "srv"}, deployment)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("legacy per-server proxy deployment not removed: %v", err)
	}
	configMap := &corev1.ConfigMap{}
	err = env.client.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "srv"}, configMap)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("legacy per-server proxy ConfigMap not removed: %v", err)
	}
	service := &corev1.Service{}
	serviceKey := types.NamespacedName{Namespace: "ns", Name: "srv"}
	if err := env.client.Get(context.Background(), serviceKey, service); err != nil {
		t.Fatalf("get service: %v", err)
	}
	if service.Spec.Selector != nil {
		t.Fatalf("legacy service selector not cleared: %v", service.Spec.Selector)
	}
}

func TestSubstrateValidationFailure(t *testing.T) {
	server := substrateMCPServer()
	server.Spec.Deployment.Image = "unpinned:latest"
	env := newSubstrateTestEnv(t, substrate.IngressModeNone, server, workerPool())

	env.reconcileSteady(t)

	got := env.getServer(t)
	assertCondition(t, got, mcpv1.MCPServerConditionAccepted, metav1.ConditionFalse, mcpv1.MCPServerReasonUnsupportedField)
	assertCondition(t, got, mcpv1.MCPServerConditionReady, metav1.ConditionFalse, mcpv1.MCPServerReasonNotAvailable)
	if len(env.ate.createCalls) != 0 {
		t.Fatalf("CreateActor called for invalid config: %v", env.ate.createCalls)
	}
}

func TestSubstrateWorkerPoolNotFound(t *testing.T) {
	env := newSubstrateTestEnv(t, substrate.IngressModeNone, substrateMCPServer())

	env.reconcile(t) // finalizer
	result := env.reconcile(t)
	if result.RequeueAfter == 0 {
		t.Fatal("expected requeue while WorkerPool is missing")
	}

	server := env.getServer(t)
	assertCondition(t, server, mcpv1.MCPServerConditionResolvedRefs, metav1.ConditionFalse,
		mcpv1.MCPServerReasonWorkerPoolNotFound)
}

func TestSubstrateDelete(t *testing.T) {
	env := newSubstrateTestEnv(t, substrate.IngressModeNone, substrateMCPServer(), workerPool())

	env.reconcileSteady(t)
	env.markTemplateReady(t)
	env.reconcileSteady(t)

	if err := env.client.Delete(context.Background(), env.getServer(t)); err != nil {
		t.Fatalf("delete MCPServer: %v", err)
	}

	// First pass issues actor deletion and requeues; the runtime actor and
	// the golden actor recorded on the template must both be deleted.
	env.ate.actors["golden-1"] = &ateapipb.Actor{ActorId: "golden-1", Status: ateapipb.Actor_STATUS_SUSPENDED}
	for i := 0; i < 5; i++ {
		server := &mcpv1.MCPServer{}
		err := env.client.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "srv"}, server)
		if apierrors.IsNotFound(err) {
			break
		}
		if err != nil {
			t.Fatalf("get MCPServer: %v", err)
		}
		env.reconcile(t)
	}

	server := &mcpv1.MCPServer{}
	err := env.client.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "srv"}, server)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("MCPServer still present after delete reconciles: %v", err)
	}
	if len(env.ate.actors) != 0 {
		t.Fatalf("actors not cleaned up: %v", env.ate.actors)
	}
	deleted := map[string]bool{}
	for _, id := range env.ate.deleteCalls {
		deleted[id] = true
	}
	if !deleted["mcp-ns-srv"] || !deleted["golden-1"] {
		t.Fatalf("delete calls = %v", env.ate.deleteCalls)
	}
}

func TestSubstrateRuntimeFlipCleansUp(t *testing.T) {
	env := newSubstrateTestEnv(t, substrate.IngressModeNone, substrateMCPServer(), workerPool())

	env.reconcileSteady(t)
	env.markTemplateReady(t)
	env.reconcileSteady(t)

	// Flip the runtime to kubernetes.
	server := env.getServer(t)
	server.Spec.Runtime = mcpv1.MCPServerRuntimeKubernetes
	server.Spec.Substrate = nil
	if err := env.client.Update(context.Background(), server); err != nil {
		t.Fatalf("update MCPServer: %v", err)
	}

	for i := 0; i < 5; i++ {
		if result := env.reconcile(t); result.RequeueAfter == 0 {
			break
		}
	}

	server = env.getServer(t)
	if controllerutil.ContainsFinalizer(server, substrate.FinalizerName) {
		t.Fatal("finalizer not removed after runtime flip")
	}
	if len(env.ate.actors) != 0 {
		t.Fatalf("actors not cleaned up after flip: %v", env.ate.actors)
	}
	tmpl := &atev1alpha1.ActorTemplate{}
	err := env.client.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "srv"}, tmpl)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("ActorTemplate still present after flip: %v", err)
	}
	if server.Status.Substrate != nil {
		t.Fatalf("status.substrate not cleared after flip: %+v", server.Status.Substrate)
	}
}

func TestKubernetesReconcilerIgnoresSubstrateServers(t *testing.T) {
	scheme := substrateTestScheme(t)
	server := substrateMCPServer()
	kube := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(server).
		WithStatusSubresource(&mcpv1.MCPServer{}).
		Build()

	reconciler := &MCPServerReconciler{Client: kube, Scheme: scheme}
	result, err := reconciler.Reconcile(context.Background(),
		ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "ns", Name: "srv"}})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Fatalf("unexpected requeue: %+v", result)
	}

	deployment := &appsv1.Deployment{}
	err = kube.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "srv"}, deployment)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("kubernetes reconciler created a deployment for a substrate server: %v", err)
	}
}

func TestSubstrateDeleteTimeoutCondition(t *testing.T) {
	server := substrateMCPServer()
	server.Finalizers = []string{substrate.FinalizerName}
	now := metav1.NewTime(time.Now().Add(-10 * time.Minute))
	server.DeletionTimestamp = &now

	env := newSubstrateTestEnv(t, substrate.IngressModeNone, server, workerPool())
	// The runtime actor does not exist, so cleanup finishes on the first pass
	// despite the old deletion timestamp; the timeout path must not panic and
	// the finalizer must be removed.
	env.reconcile(t)

	got := &mcpv1.MCPServer{}
	err := env.client.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "srv"}, got)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("MCPServer should be gone after cleanup: %v", err)
	}
}

// TestEnsureSharedProxyConfigBootstrapsEmptyConfigMap verifies the shared proxy
// route ConfigMap is created on startup even with zero MCPServers, so the proxy
// pods can mount it and start (the controller's manager Runnable calls this).
func TestEnsureSharedProxyConfigBootstrapsEmptyConfigMap(t *testing.T) {
	env := newSubstrateTestEnv(t, substrate.IngressModeManagedProxy)
	ctx := context.Background()
	meta := substrate.SharedProxyConfigMapMeta(env.reconciler.Ingress)
	key := types.NamespacedName{Namespace: meta.Namespace, Name: meta.Name}

	if err := env.reconciler.ensureSharedProxyConfig(ctx); err != nil {
		t.Fatalf("ensureSharedProxyConfig: %v", err)
	}
	cm := &corev1.ConfigMap{}
	if err := env.client.Get(ctx, key, cm); err != nil {
		t.Fatalf("expected shared ConfigMap to be created: %v", err)
	}
	if _, ok := cm.Data[substrate.ProxyConfigKey]; !ok {
		t.Fatalf("ConfigMap missing key %q", substrate.ProxyConfigKey)
	}

	// Idempotent: a second call must not clobber existing (reconciler-owned) content.
	cm.Data[substrate.ProxyConfigKey] = "sentinel"
	if err := env.client.Update(ctx, cm); err != nil {
		t.Fatalf("seed existing content: %v", err)
	}
	if err := env.reconciler.ensureSharedProxyConfig(ctx); err != nil {
		t.Fatalf("ensureSharedProxyConfig (second): %v", err)
	}
	got := &corev1.ConfigMap{}
	if err := env.client.Get(ctx, key, got); err != nil {
		t.Fatalf("get after second ensure: %v", err)
	}
	if got.Data[substrate.ProxyConfigKey] != "sentinel" {
		t.Fatalf("ensure clobbered existing config: %q", got.Data[substrate.ProxyConfigKey])
	}
}
