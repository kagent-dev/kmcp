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
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"gopkg.in/yaml.v3"

	"github.com/kagent-dev/kmcp/api/v1alpha1"
	"github.com/kagent-dev/kmcp/test/utils"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// namespace where the project is deployed in
const namespace = "kmcp-system"

var _ = ginkgo.Describe("Manager", ginkgo.Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, and deploying the controller using Helm.
	ginkgo.BeforeAll(func() {
		var cmd *exec.Cmd
		var err error

		ginkgo.By("creating manager namespace")
		cmd = exec.Command("kubectl", "create", "ns", namespace)
		_, err = utils.Run(cmd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to create namespace")

		ginkgo.By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to label namespace with restricted policy")

		ginkgo.By("deploying the CRDs using Helm")
		cmd = exec.Command("helm", "install", "kmcp-crds", "helm/kmcp-crds",
			"--namespace", namespace,
			"--wait", "--timeout=5m")
		_, err = utils.Run(cmd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to deploy the CRDs using Helm")

		ginkgo.By("deploying the controller-manager using Helm")
		cmd = exec.Command("helm", "install", "kmcp", "helm/kmcp",
			"--namespace", namespace,
			"--wait", "--timeout=5m",
			"--set", fmt.Sprintf("image.repository=%s", getImageRepository(projectImage)),
			"--set", fmt.Sprintf("image.tag=%s", getImageTag(projectImage)))
		_, err = utils.Run(cmd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to deploy the controller-manager using Helm")
	})

	// After all tests have been executed, clean up by undeploying the controller using Helm
	// and deleting the namespace.
	ginkgo.AfterAll(func() {
		ginkgo.By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		ginkgo.By("undeploying the controller-manager using Helm")
		cmd = exec.Command("helm", "uninstall", "kmcp", "--namespace", namespace)
		_, _ = utils.Run(cmd)

		ginkgo.By("cleaning up the knowledge-assistant project directory")
		cmd = exec.Command("rm", "-rf", "knowledge-assistant")
		_, _ = utils.Run(cmd)

		ginkgo.By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)

		ginkgo.By("cleaning up the CRD")
		cmd = exec.Command("kubectl", "delete", "crd", "mcpservers.kagent.dev", "--ignore-not-found=true")
		_, _ = utils.Run(cmd)

		ginkgo.By("cleaning up Helm packages")
		cmd = exec.Command("make", "helm-cleanup")
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	ginkgo.AfterEach(func() {
		specReport := ginkgo.CurrentSpecReport()
		if specReport.Failed() {
			ginkgo.By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			ginkgo.By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			ginkgo.By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			ginkgo.By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	gomega.SetDefaultEventuallyTimeout(2 * time.Minute)
	gomega.SetDefaultEventuallyPollingInterval(time.Second)

	ginkgo.Context("Manager", func() {
		ginkgo.It("should run successfully", func() {
			ginkgo.By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g gomega.Gomega) {
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
				g.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(gomega.HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(gomega.ContainSubstring("controller-manager"))

				// Validate the pod's status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(output).To(gomega.Equal("Running"), "Incorrect controller-manager pod status")
			}
			gomega.Eventually(verifyControllerUp).Should(gomega.Succeed())
		})
	})

	ginkgo.Context("MCPServer CRD", func() {
		ginkgo.It("build and deploy a working MCP server with mounted secrets", func() {
			mcpServerName := "knowledge-assistant"
			var portForwardCmd *exec.Cmd
			localPort := 8080
			projectDir := "knowledge-assistant"

			ginkgo.By("building the kmcp CLI")
			cmd := exec.Command("make", "build-cli")
			_, err := utils.Run(cmd)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to build kmcp CLI")

			ginkgo.By("creating a knowledge-assistant project using kmcp CLI")
			cmd = exec.Command(
				"dist/kmcp",
				"init",
				"python",
				projectDir,
				"--non-interactive",
				"--force",
				"--namespace",
				namespace,
			)
			_, err = utils.Run(cmd)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to create knowledge-assistant project")

			ginkgo.By("updating kmcp.yaml to enable staging secrets")
			cmd = exec.Command("sed",
				"-i.bak",
				"/staging:/,/enabled:/ s/enabled: false/enabled: true/",
				fmt.Sprintf("%s/kmcp.yaml", projectDir))
			_, err = utils.Run(cmd)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to update kmcp.yaml to enable staging secrets")

			// clean up kmcp yaml backup file
			cmd = exec.Command("rm", "-f", fmt.Sprintf("%s/kmcp.yaml.bak", projectDir))
			_, _ = utils.Run(cmd)

			ginkgo.By("creating a dummy .env.staging file for testing secrets")
			envFilePath := fmt.Sprintf("%s/.env.staging", projectDir)
			envContent := []byte("DATABASE_URL=postgres://user:pass@host:port/db\n" +
				"OPENAI_API_KEY=dummy-key\n" +
				"WEATHER_API_KEY=dummy-key\n")
			err = os.WriteFile(envFilePath, envContent, 0644)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to create dummy .env.staging file")

			ginkgo.By("creating Kubernetes secret from existing .env.staging file")

			cmd = exec.Command(
				"dist/kmcp",
				"secrets",
				"sync",
				"staging",
				"--from-file",
				envFilePath,
				"--project-dir",
				projectDir,
			)
			_, err = utils.Run(cmd)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to create secret from .env.local file")

			ginkgo.By("building the Docker image for the knowledge-assistant project")
			cmd = exec.Command("dist/kmcp",
				"build",
				"--verbose",
				"--project-dir",
				projectDir,
				"--kind-load-cluster",
				"kind",
			)
			_, err = utils.Run(cmd)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to build Docker image")

			ginkgo.By("deploying the knowledge-assistant MCP server using kmcp CLI")
			cmd = exec.Command(
				"dist/kmcp",
				"deploy",
				"-f",
				fmt.Sprintf("%s/kmcp.yaml", projectDir),
				"-n",
				namespace,
				"--environment",
				"staging",
				"--no-inspector",
			)
			_, err = utils.Run(cmd)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to deploy knowledge-assistant MCP server")

			ginkgo.By("waiting for the deployment to be ready")
			gomega.Eventually(func(g gomega.Gomega) {
				deployment := getDeployment(mcpServerName, namespace)
				g.Expect(deployment).NotTo(gomega.BeNil())
				g.Expect(deployment.Status.ReadyReplicas).To(gomega.Equal(int32(1)))
			}, 3*time.Minute).Should(gomega.Succeed())

			ginkgo.By("waiting for the service to be ready")
			gomega.Eventually(func(g gomega.Gomega) {
				service := getService(mcpServerName, namespace)
				g.Expect(service).NotTo(gomega.BeNil())
				g.Expect(service.Spec.Ports).To(gomega.HaveLen(1))
				g.Expect(service.Spec.Ports[0].Port).To(gomega.Equal(int32(3000)))
			}).Should(gomega.Succeed())

			ginkgo.By("verifying that environment variables are loaded via envFrom")
			gomega.Eventually(func(g gomega.Gomega) {
				// Get the pod name
				cmd := exec.Command("kubectl", "get", "pods", "-l",
					fmt.Sprintf("app.kubernetes.io/name=%s", mcpServerName), "-n", namespace,
					"-o", "jsonpath={.items[0].metadata.name}")
				podName, err := utils.Run(cmd)
				g.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to get pod name")
				g.Expect(podName).NotTo(gomega.BeEmpty(), "Pod name should not be empty")

				// Verify that environment variables are set in the container
				expectedVars := []string{"DATABASE_URL", "OPENAI_API_KEY", "WEATHER_API_KEY"}
				for _, envVar := range expectedVars {
					cmd = exec.Command("kubectl", "exec", strings.TrimSpace(podName), "-n", namespace, "--", "sh", "-c",
						fmt.Sprintf("echo $%s", envVar))
					output, err := utils.Run(cmd)
					g.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("Failed to check environment variable %s", envVar))
					g.Expect(strings.TrimSpace(output)).NotTo(gomega.BeEmpty(),
						fmt.Sprintf("Environment variable %s should be set", envVar))
				}
			}, 30*time.Second, 1*time.Second).Should(gomega.Succeed())

			ginkgo.By("setting up kubectl port-forward to access the MCP server")
			portForwardCmd = exec.Command("kubectl", "port-forward",
				fmt.Sprintf("service/%s", mcpServerName),
				fmt.Sprintf("%d:3000", localPort),
				"-n", namespace)

			err = portForwardCmd.Start()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to start port-forward")

			// Wait for port-forward to be ready
			gomega.Eventually(func() error {
				resp, err := http.Get(fmt.Sprintf("http://localhost:%d", localPort))
				if err != nil {
					return err
				}
				_ = resp.Body.Close()
				return nil
			}, 30*time.Second, 1*time.Second).Should(gomega.Succeed())

			ginkgo.By("creating MCP client and testing connection")
			mcpClient, err := client.NewStreamableHttpClient(fmt.Sprintf("http://localhost:%d/mcp", localPort))
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to create MCP client")

			ctx := context.Background()

			ginkgo.By("initializing the MCP client")
			initResponse, err := mcpClient.Initialize(ctx, mcp.InitializeRequest{
				Params: mcp.InitializeParams{
					ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
					ClientInfo: mcp.Implementation{
						Name:    "kmcp-e2e-test",
						Version: "1.0.0",
					},
				},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to initialize MCP client")
			gomega.Expect(initResponse).NotTo(gomega.BeNil())

			ginkgo.By("listing available tools from the MCP server")
			toolsResponse, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to list tools")
			gomega.Expect(toolsResponse).NotTo(gomega.BeNil())
			gomega.Expect(toolsResponse.Tools).NotTo(gomega.BeEmpty(), "Expected at least one tool to be available")

			// Log the available tools for debugging
			for _, tool := range toolsResponse.Tools {
				_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Available tool: %s - %s\n", tool.Name, tool.Description)
			}

			ginkgo.By("cleaning up port-forward")
			if portForwardCmd != nil && portForwardCmd.Process != nil {
				err = portForwardCmd.Process.Kill()
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to kill port-forward process")
			}

			ginkgo.By("cleaning up the MCPServer")
			cmd = exec.Command("kubectl", "delete", "mcpserver", mcpServerName, "-n", namespace)
			_, err = utils.Run(cmd)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("deploy a working MCP server with auth policy enabled", func() {
			mcpServerName := "everything"
			var portForwardCmd *exec.Cmd
			localPort := 8080

			ginkgo.By("creating a secret with the JWKS")
			// Read the JWKS content from the test data file
			jwksContent, err := os.ReadFile("test/testdata/jwt/pub-key")
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to read JWKS file")

			// Create the secret using kubectl with literal content (not base64 encoded)
			cmd := exec.Command("kubectl", "create", "secret", "generic", "example-jwks",
				"--from-literal=jwks="+string(jwksContent),
				"-n", namespace)
			_, err = utils.Run(cmd)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to create JWKS secret")

			ginkgo.By("creating an MCPServer for client testing")
			mcpServer := &v1alpha1.MCPServer{
				TypeMeta: v1.TypeMeta{
					APIVersion: "kagent.dev/v1alpha1",
					Kind:       "MCPServer",
				},
				ObjectMeta: v1.ObjectMeta{
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
					Authn: &v1alpha1.MCPServerAuthentication{
						JWT: &v1alpha1.MCPServerJWTAuthentication{
							Issuer:    "agentgateway.dev",
							Audiences: []string{"test.agentgateway.dev"},
							JWKS: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "example-jwks",
								},
								Key: "jwks",
							},
						},
					},
					Authz: &v1alpha1.MCPServerAuthorization{
						Rules: &[]string{
							// Allow anyone to call 'read_file'
							"mcp.tool.name == 'read_file'",
							// Only the test-user can call 'write_file'
							"jwt.sub == 'test-user' && mcp.tool.name == 'write_file'",
							// Any authenticated user with the claim `nested.key == value` can access 'list_directory'
							"mcp.tool.name == \"list_directory\" && jwt.nested.key == \"value\"",
						},
					},
				},
			}

			cmd = exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(mcpServerToYAML(mcpServer))
			_, err = utils.Run(cmd)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to apply MCP server")

			ginkgo.By("waiting for the deployment to be ready")
			gomega.Eventually(func(g gomega.Gomega) {
				deployment := getDeployment(mcpServerName, namespace)
				g.Expect(deployment).NotTo(gomega.BeNil())
				g.Expect(deployment.Status.ReadyReplicas).To(gomega.Equal(int32(1)))
			}, 3*time.Minute).Should(gomega.Succeed())

			ginkgo.By("waiting for the service to be ready")
			gomega.Eventually(func(g gomega.Gomega) {
				service := getService(mcpServerName, namespace)
				g.Expect(service).NotTo(gomega.BeNil())
				g.Expect(service.Spec.Ports).To(gomega.HaveLen(1))
				g.Expect(service.Spec.Ports[0].Port).To(gomega.Equal(int32(3000)))
			}).Should(gomega.Succeed())

			ginkgo.By("setting up kubectl port-forward to access the MCP server")
			portForwardCmd = exec.Command("kubectl", "port-forward",
				fmt.Sprintf("service/%s", mcpServerName),
				fmt.Sprintf("%d:3000", localPort),
				"-n", namespace)

			err = portForwardCmd.Start()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to start port-forward")

			// Wait for port-forward to be ready
			gomega.Eventually(func() error {
				resp, err := http.Get(fmt.Sprintf("http://localhost:%d", localPort))
				if err != nil {
					return err
				}
				_ = resp.Body.Close()
				return nil
			}, 30*time.Second, 1*time.Second).Should(gomega.Succeed())

			ginkgo.By("creating MCP client and testing connection")
			mcpClient, err := client.NewStreamableHttpClient(fmt.Sprintf("http://localhost:%d/mcp", localPort))
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to create MCP client")

			ctx := context.Background()

			ginkgo.By("initializing the MCP client")
			initResponse, err := mcpClient.Initialize(ctx, mcp.InitializeRequest{
				Params: mcp.InitializeParams{
					ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
					ClientInfo: mcp.Implementation{
						Name:    "kmcp-e2e-test",
						Version: "1.0.0",
					},
				},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to initialize MCP client")
			gomega.Expect(initResponse).NotTo(gomega.BeNil())

			ginkgo.By("listing available tools from the MCP server")
			toolsResponse, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to list tools")
			gomega.Expect(toolsResponse).NotTo(gomega.BeNil())
			gomega.Expect(toolsResponse.Tools).NotTo(gomega.BeEmpty(), "Expected at least one tool to be available")

			// we expect only the `read_file` tool to be available
			gomega.Expect(toolsResponse.Tools).To(gomega.HaveLen(1))
			gomega.Expect(toolsResponse.Tools[0].Name).To(gomega.Equal("read_file"))

			ginkgo.By("reading JWT tokens from example files")
			// Read the JWT tokens from the test data files
			example1Token, err := os.ReadFile("test/testdata/jwt/example1.key")
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to read example1.key file")
			example1TokenStr := strings.TrimSpace(string(example1Token))

			example2Token, err := os.ReadFile("test/testdata/jwt/example2.key")
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to read example2.key file")
			example2TokenStr := strings.TrimSpace(string(example2Token))

			ginkgo.By("testing list_directory tool access with JWT tokens")
			// Test that example1.key can access list_directory (should succeed)
			// and example2.key cannot access list_directory (should fail)
			// this is because our authz rules dictate that only a user with the claim
			// `nested.key == value` can access the list_directory tool
			// example1.key has the claim `nested.key == value` whereas example2.key does not

			// Test with example1.key - should succeed
			_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Testing list_directory access with example1.key (should succeed)\n")

			// Test with example1.key - should succeed
			_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Testing list_directory access with example1.key (should succeed)\n")
			_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Attempting to call list_directory tool with example1 token...\n")

			// Create MCP client with JWT authentication for example1
			mcpClient1, err := client.NewStreamableHttpClient(
				fmt.Sprintf("http://localhost:%d/mcp", localPort),
				transport.WithHTTPHeaders(map[string]string{
					"Authorization": "Bearer " + example1TokenStr,
				}),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to create MCP client with example1 token")

			// Initialize the client with example1 token
			ctx1 := context.Background()
			initResponse1, err := mcpClient1.Initialize(ctx1, mcp.InitializeRequest{
				Params: mcp.InitializeParams{
					ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
					ClientInfo: mcp.Implementation{
						Name:    "kmcp-e2e-test-example1",
						Version: "1.0.0",
					},
				},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to initialize MCP client with example1 token")
			gomega.Expect(initResponse1).NotTo(gomega.BeNil())

			// list tools when using example1 token
			toolsResponse, err = mcpClient1.ListTools(ctx1, mcp.ListToolsRequest{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to list tools")
			gomega.Expect(toolsResponse).NotTo(gomega.BeNil())
			gomega.Expect(toolsResponse.Tools).NotTo(gomega.BeEmpty(), "Expected at least one tool to be available")

			// we expect 3 tools to be available, `read_file`, `list_directory`, and `write_file`
			gomega.Expect(toolsResponse.Tools).To(gomega.HaveLen(3))

			// Test list_directory tool with example1 token - should succeed
			callResponse1, err := mcpClient1.CallTool(ctx1, mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "list_directory",
					Arguments: map[string]interface{}{
						"path": "/",
					},
				},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "list_directory should succeed with example1 token")
			gomega.Expect(callResponse1).NotTo(gomega.BeNil())
			_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "✓ list_directory succeeded with example1 token\n")

			// Test with example2.key - should fail
			_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Testing list_directory access with example2.key (should fail)\n")
			_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Attempting to call list_directory tool with example2.key...\n")

			// Create MCP client with JWT authentication for example2
			mcpClient2, err := client.NewStreamableHttpClient(
				fmt.Sprintf("http://localhost:%d/mcp", localPort),
				transport.WithHTTPHeaders(map[string]string{
					"Authorization": "Bearer " + example2TokenStr,
				}),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to create MCP client with example2 token")

			// Initialize the client with example2 token
			ctx2 := context.Background()
			initResponse2, err := mcpClient2.Initialize(ctx2, mcp.InitializeRequest{
				Params: mcp.InitializeParams{
					ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
					ClientInfo: mcp.Implementation{
						Name:    "kmcp-e2e-test-example2",
						Version: "1.0.0",
					},
				},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to initialize MCP client with example2 token")
			gomega.Expect(initResponse2).NotTo(gomega.BeNil())

			// list tools when using example2 token
			toolsResponse, err = mcpClient2.ListTools(ctx1, mcp.ListToolsRequest{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to list tools")
			gomega.Expect(toolsResponse).NotTo(gomega.BeNil())
			gomega.Expect(toolsResponse.Tools).NotTo(gomega.BeEmpty(), "Expected at least one tool to be available")

			// we expect 2 tools to be available, `read_file` and `write_file`
			gomega.Expect(toolsResponse.Tools).To(gomega.HaveLen(2))

			// Test list_directory tool with example2 token (should fail)
			_, err = mcpClient2.CallTool(ctx2, mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "list_directory",
					Arguments: map[string]interface{}{
						"path": "/",
					},
				},
			})
			// We expect this to fail due to our authz rules
			gomega.Expect(err).To(gomega.HaveOccurred(), "list_directory should fail with example2 token")
			// error message should be `not allowed`
			_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "✓ list_directory failed with example2 token as expected: %v\n", err)

			ginkgo.By("cleaning up port-forward")
			if portForwardCmd != nil && portForwardCmd.Process != nil {
				err = portForwardCmd.Process.Kill()
				gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to kill port-forward process")
			}

			ginkgo.By("cleaning up the MCPServer")
			cmd = exec.Command("kubectl", "delete", "mcpserver", mcpServerName, "-n", namespace)
			_, err = utils.Run(cmd)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
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

func mcpServerToYAML(mcpServer interface{}) string {
	var yamlData map[string]interface{}

	// First marshal to JSON to get the proper structure
	jsonBytes, err := json.Marshal(mcpServer)
	if err != nil {
		return ""
	}

	// Then unmarshal to map to ensure we have the right structure
	err = json.Unmarshal(jsonBytes, &yamlData)
	if err != nil {
		return ""
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(yamlData)
	if err != nil {
		return ""
	}
	return string(yamlBytes)
}
