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

	"github.com/onsi/gomega"

	ginkgo "github.com/onsi/ginkgo/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
})
