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

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kagentdevv1alpha1 "github.com/kagent-dev/kmcp/api/v1alpha1"
)

var _ = ginkgo.Describe("MCPServer Controller", func() {
	ginkgo.Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		mcpserver := &kagentdevv1alpha1.MCPServer{}

		ginkgo.BeforeEach(func() {
			ginkgo.By("creating the custom resource for the Kind MCPServer")
			err := k8sClient.Get(ctx, typeNamespacedName, mcpserver)
			if err != nil && errors.IsNotFound(err) {
				resource := &kagentdevv1alpha1.MCPServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: kagentdevv1alpha1.MCPServerSpec{
						Deployment: kagentdevv1alpha1.MCPServerDeployment{
							Image: "docker.io/mcp/everything",
							Port:  3000,
							Cmd:   "npx",
							Args:  []string{"-y", "@modelcontextprotocol/server-filesystem", "/"},
						},
						TransportType: "stdio",
					},
				}
				gomega.Expect(k8sClient.Create(ctx, resource)).To(gomega.Succeed())
			}
		})

		ginkgo.AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &kagentdevv1alpha1.MCPServer{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("Cleanup the specific resource instance MCPServer")
			gomega.Expect(k8sClient.Delete(ctx, resource)).To(gomega.Succeed())
		})
		ginkgo.It("should successfully reconcile the resource", func() {
			ginkgo.By("Reconciling the created resource")
			scheme := k8sClient.Scheme()
			err := kagentdevv1alpha1.AddToScheme(scheme)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			controllerReconciler := &MCPServerReconciler{
				Client: k8sClient,
				Scheme: scheme,
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

		})
	})

	ginkgo.Context("When testing available replicas functionality", func() {
		const testResourceName = "test-replicas-resource"
		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      testResourceName,
			Namespace: "default",
		}

		ginkgo.BeforeEach(func() {
			ginkgo.By("creating test MCPServer resource")
			resource := &kagentdevv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testResourceName,
					Namespace: "default",
				},
				Spec: kagentdevv1alpha1.MCPServerSpec{
					Deployment: kagentdevv1alpha1.MCPServerDeployment{
						Image: "docker.io/mcp/everything",
						Port:  3000,
						Cmd:   "npx",
						Args:  []string{"-y", "@modelcontextprotocol/server-filesystem", "/"},
					},
					TransportType: "stdio",
				},
			}
			gomega.Expect(k8sClient.Create(ctx, resource)).To(gomega.Succeed())
		})

		ginkgo.AfterEach(func() {
			ginkgo.By("cleaning up test resources")
			// Clean up MCPServer
			resource := &kagentdevv1alpha1.MCPServer{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				gomega.Expect(k8sClient.Delete(ctx, resource)).To(gomega.Succeed())
			}

			// Clean up deployment
			deployment := &appsv1.Deployment{}
			err = k8sClient.Get(ctx, typeNamespacedName, deployment)
			if err == nil {
				gomega.Expect(k8sClient.Delete(ctx, deployment)).To(gomega.Succeed())
			}
		})

		ginkgo.It("should set Available condition to false when deployment has no available replicas", func() {
			// Setup controller and create deployment
			controllerReconciler := setupController()
			createDeployment(ctx, controllerReconciler, typeNamespacedName)

			// Update deployment status to have no available replicas
			updateDeploymentStatus(ctx, typeNamespacedName, 3, 0, 0)

			// Reconcile and verify Ready condition is false
			reconcileAndVerifyCondition(ctx, controllerReconciler, typeNamespacedName,
				metav1.ConditionFalse,
				string(kagentdevv1alpha1.MCPServerReasonNotAvailable),
				"0/3 replicas available")
		})

		ginkgo.It("should set Available condition to true when deployment has all replicas available", func() {
			// Setup controller and create deployment
			controllerReconciler := setupController()
			createDeployment(ctx, controllerReconciler, typeNamespacedName)

			// Update deployment status to have all replicas available
			updateDeploymentStatus(ctx, typeNamespacedName, 2, 2, 2)

			// Reconcile and verify Ready condition is true
			reconcileAndVerifyCondition(ctx, controllerReconciler, typeNamespacedName,
				metav1.ConditionTrue,
				string(kagentdevv1alpha1.MCPServerReasonAvailable),
				"Deployment is ready and all pods are running")
		})
	})

	ginkgo.Context("Volume Mounting", func() {
		ginkgo.It("should create deployment with ConfigMap and Secret references", func() {
			ginkgo.By("Creating MCPServer with volume references")
			serverWithVolumes := &kagentdevv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-server-with-volumes",
					Namespace: "default",
				},
				Spec: kagentdevv1alpha1.MCPServerSpec{
					TransportType: kagentdevv1alpha1.TransportTypeStdio,
					Deployment: kagentdevv1alpha1.MCPServerDeployment{
						Image: "test-image:latest",
						Port:  8080,
						SecretRefs: []corev1.LocalObjectReference{
							{Name: "test-secret"},
						},
						ConfigMapRefs: []corev1.LocalObjectReference{
							{Name: "test-configmap"},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "custom-volume",
								MountPath: "/custom",
								ReadOnly:  false,
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "custom-volume",
								VolumeSource: corev1.VolumeSource{
									EmptyDir: &corev1.EmptyDirVolumeSource{},
								},
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, serverWithVolumes)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("Reconciling the MCPServer with volumes")
			scheme := k8sClient.Scheme()
			err = kagentdevv1alpha1.AddToScheme(scheme)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			controllerReconciler := &MCPServerReconciler{
				Client: k8sClient,
				Scheme: scheme,
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-server-with-volumes",
					Namespace: "default",
				},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("Verifying deployment was created with volumes")
			deployment := &appsv1.Deployment{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-server-with-volumes",
				Namespace: "default",
			}, deployment)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Check that the deployment has the expected volumes
			// config, binary, cm-test-configmap, custom-volume
			gomega.Expect(deployment.Spec.Template.Spec.Volumes).To(gomega.HaveLen(4))

			// Check that the container has the expected volume mounts
			container := deployment.Spec.Template.Spec.Containers[0]
			// config, binary, cm-test-configmap, custom-volume
			gomega.Expect(container.VolumeMounts).To(gomega.HaveLen(4))

			// Verify that custom volume mount is present
			foundCustomMount := false
			for _, mount := range container.VolumeMounts {
				if mount.Name == "custom-volume" && mount.MountPath == "/custom" {
					foundCustomMount = true
					break
				}
			}
			gomega.Expect(foundCustomMount).To(gomega.BeTrue(), "Custom volume mount not found in container")

			// Cleanup
			err = k8sClient.Delete(ctx, serverWithVolumes)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})
})

// Helper functions to reduce code duplication

func setupController() *MCPServerReconciler {
	scheme := k8sClient.Scheme()
	err := kagentdevv1alpha1.AddToScheme(scheme)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return &MCPServerReconciler{
		Client: k8sClient,
		Scheme: scheme,
	}
}

func createDeployment(ctx context.Context, controllerReconciler *MCPServerReconciler,
	typeNamespacedName types.NamespacedName) {
	ginkgo.By("reconciling to create deployment")
	_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: typeNamespacedName,
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func updateDeploymentStatus(ctx context.Context, typeNamespacedName types.NamespacedName,
	replicas, availableReplicas, readyReplicas int32) {
	ginkgo.By("updating deployment status")
	deployment := &appsv1.Deployment{}
	gomega.Expect(k8sClient.Get(ctx, typeNamespacedName, deployment)).To(gomega.Succeed())

	deployment.Status = appsv1.DeploymentStatus{
		Replicas:          replicas,
		AvailableReplicas: availableReplicas,
		ReadyReplicas:     readyReplicas,
	}
	gomega.Expect(k8sClient.Status().Update(ctx, deployment)).To(gomega.Succeed())
}

func reconcileAndVerifyCondition(ctx context.Context, controllerReconciler *MCPServerReconciler,
	typeNamespacedName types.NamespacedName, expectedStatus metav1.ConditionStatus,
	expectedReason, expectedMessageSubstring string) {
	ginkgo.By("reconciling again to check ready condition")
	_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: typeNamespacedName,
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	ginkgo.By("verifying Ready condition")
	updatedMCPServer := &kagentdevv1alpha1.MCPServer{}
	gomega.Expect(k8sClient.Get(ctx, typeNamespacedName, updatedMCPServer)).To(gomega.Succeed())

	var readyCondition *metav1.Condition
	for _, condition := range updatedMCPServer.Status.Conditions {
		if condition.Type == string(kagentdevv1alpha1.MCPServerConditionReady) {
			readyCondition = &condition
			break
		}
	}
	gomega.Expect(readyCondition).NotTo(gomega.BeNil())
	gomega.Expect(readyCondition.Status).To(gomega.Equal(expectedStatus))
	gomega.Expect(readyCondition.Reason).To(gomega.Equal(expectedReason))
	gomega.Expect(readyCondition.Message).To(gomega.ContainSubstring(expectedMessageSubstring))
}
