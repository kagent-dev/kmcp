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

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kagent.dev/kmcp/api/v1alpha1"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"kagent.dev/kmcp/test/utils"
	"sigs.k8s.io/yaml"
)

// namespace where the project is deployed in
const namespace = "kmcp-system"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, and deploying the controller using Helm.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("deploying the controller-manager using Helm")
		cmd = exec.Command("helm", "install", "kmcp", "helm/kmcp",
			"--namespace", namespace,
			"--wait", "--timeout=5m",
			"--set", fmt.Sprintf("image.repository=%s", getImageRepository(projectImage)),
			"--set", fmt.Sprintf("image.tag=%s", getImageTag(projectImage)))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager using Helm")
	})

	// After all tests have been executed, clean up by undeploying the controller using Helm
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager using Helm")
		cmd = exec.Command("helm", "uninstall", "kmcp", "--namespace", namespace)
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				// Validate the pod's status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks

		// TODO: Customize the e2e test suite with scenarios specific to your project.
		// Consider applying sample/CR(s) and check their status and/or verifying
		// the reconciliation by using the metrics, i.e.:
		// metricsOutput := getMetricsOutput()
		// Expect(metricsOutput).To(ContainSubstring(
		//    fmt.Sprintf(`controller_runtime_reconcile_total{controller="%s",result="success"} 1`,
		//    strings.ToLower(<Kind>),
		// ))
	})

	Context("MCPServer CRD", func() {
		It("deploy a working MCP server", func() {
			mcpServerName := "test-mcp-client-server"
			var portForwardCmd *exec.Cmd
			localPort := 8080

			By("creating an MCPServer for client testing")
			mcpServer := &v1alpha1.MCPServer{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kagent.dev/v1alpha1",
					Kind:       "MCPServer",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      mcpServerName,
					Namespace: namespace,
				},
				Spec: v1alpha1.MCPServerSpec{
					Deployment: v1alpha1.MCPServerDeployment{
						Image: "docker.io/mcp/everything",
						Port:  3000,
						Cmd:   "npx",
						Args:  []string{"-y", "@modelcontextprotocol/server-filesystem", "/"},
					},
					TransportType: "stdio",
				},
			}

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(mcpServerToYAML(mcpServer))
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create MCPServer")

			By("waiting for the deployment to be ready")
			Eventually(func(g Gomega) {
				deployment := getDeployment(mcpServerName, namespace)
				g.Expect(deployment).NotTo(BeNil())
				g.Expect(deployment.Status.ReadyReplicas).To(Equal(int32(1)))
			}, 3*time.Minute).Should(Succeed())

			By("waiting for the service to be ready")
			Eventually(func(g Gomega) {
				service := getService(mcpServerName, namespace)
				g.Expect(service).NotTo(BeNil())
				g.Expect(service.Spec.Ports).To(HaveLen(1))
				g.Expect(service.Spec.Ports[0].Port).To(Equal(int32(3000)))
			}).Should(Succeed())

			By("setting up kubectl port-forward to access the MCP server")
			portForwardCmd = exec.Command("kubectl", "port-forward",
				fmt.Sprintf("service/%s", mcpServerName),
				fmt.Sprintf("%d:3000", localPort),
				"-n", namespace)

			err = portForwardCmd.Start()
			Expect(err).NotTo(HaveOccurred(), "Failed to start port-forward")

			// Wait for port-forward to be ready
			Eventually(func() error {
				resp, err := http.Get(fmt.Sprintf("http://localhost:%d", localPort))
				if err != nil {
					return err
				}
				_ = resp.Body.Close()
				return nil
			}, 30*time.Second, 1*time.Second).Should(Succeed())

			By("creating MCP client and testing connection")
			mcpClient, err := client.NewStreamableHttpClient(fmt.Sprintf("http://localhost:%d/mcp", localPort))
			Expect(err).NotTo(HaveOccurred(), "Failed to create MCP client")

			ctx := context.Background()

			By("initializing the MCP client")
			initResponse, err := mcpClient.Initialize(ctx, mcp.InitializeRequest{
				Params: mcp.InitializeParams{
					ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
					ClientInfo: mcp.Implementation{
						Name:    "kmcp-e2e-test",
						Version: "1.0.0",
					},
				},
			})
			Expect(err).NotTo(HaveOccurred(), "Failed to initialize MCP client")
			Expect(initResponse).NotTo(BeNil())

			By("listing available tools from the MCP server")
			toolsResponse, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
			Expect(err).NotTo(HaveOccurred(), "Failed to list tools")
			Expect(toolsResponse).NotTo(BeNil())
			Expect(toolsResponse.Tools).NotTo(BeEmpty(), "Expected at least one tool to be available")

			// Log the available tools for debugging
			for _, tool := range toolsResponse.Tools {
				_, _ = fmt.Fprintf(GinkgoWriter, "Available tool: %s - %s\n", tool.Name, tool.Description)
			}

			By("cleaning up port-forward")
			if portForwardCmd != nil && portForwardCmd.Process != nil {
				err = portForwardCmd.Process.Kill()
				Expect(err).NotTo(HaveOccurred(), "Failed to kill port-forward process")
			}

			By("cleaning up the MCPServer")
			cmd = exec.Command("kubectl", "delete", "mcpserver", mcpServerName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

// Helper functions for resource verification
func getDeployment(name, namespace string) *appsv1.Deployment {
	cmd := exec.Command("kubectl", "get", "deployment", name, "-n", namespace, "-o", "json")
	output, err := utils.Run(cmd)
	if err != nil {
		return nil
	}

	var deployment appsv1.Deployment
	if err := json.Unmarshal([]byte(output), &deployment); err != nil {
		return nil
	}
	return &deployment
}

func getService(name, namespace string) *corev1.Service {
	cmd := exec.Command("kubectl", "get", "service", name, "-n", namespace, "-o", "json")
	output, err := utils.Run(cmd)
	if err != nil {
		return nil
	}

	var service corev1.Service
	if err := json.Unmarshal([]byte(output), &service); err != nil {
		return nil
	}
	return &service
}

func mcpServerToYAML(mcpServer interface{}) string {
	yamlBytes, err := yaml.Marshal(mcpServer)
	if err != nil {
		return ""
	}
	return string(yamlBytes)
}

// getImageRepository extracts the repository part from a full image name
// e.g., "example.com/kmcp:v0.0.1" -> "example.com/kmcp"
func getImageRepository(image string) string {
	if idx := strings.LastIndex(image, ":"); idx != -1 {
		return image[:idx]
	}
	return image
}

// getImageTag extracts the tag part from a full image name
// e.g., "example.com/kmcp:v0.0.1" -> "v0.0.1"
func getImageTag(image string) string {
	if idx := strings.LastIndex(image, ":"); idx != -1 {
		return image[idx+1:]
	}
	return "latest"
}
