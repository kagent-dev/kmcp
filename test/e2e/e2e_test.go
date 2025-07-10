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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"kagent.dev/kmcp/test/utils"
	"sigs.k8s.io/yaml"
)

// namespace where the project is deployed in
const namespace = "kmcp-system"

// serviceAccountName created for the project
const serviceAccountName = "kmcp-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "kmcp-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "kmcp-metrics-binding"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
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

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
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

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=kmcp-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("validating that the ServiceMonitor for Prometheus is applied in the namespace")
			cmd = exec.Command("kubectl", "get", "ServiceMonitor", "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "ServiceMonitor should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("waiting for the metrics endpoint to be ready")
			verifyMetricsEndpointReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpoints", metricsServiceName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("8443"), "Metrics endpoint is not ready")
			}
			Eventually(verifyMetricsEndpointReady).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("controller-runtime.metrics\tServing metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted).Should(Succeed())

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": ["curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics"],
							"securityContext": {
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccount": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			metricsOutput := getMetricsOutput()
			Expect(metricsOutput).To(ContainSubstring(
				"controller_runtime_reconcile_total",
			))
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
		FIt("should create all resources for stdio transport MCPServer", func() {
			mcpServerName := "test-stdio-server"

			By("creating an MCPServer with stdio transport")
			mcpServer := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "kagent.dev/v1alpha1",
					"kind":       "MCPServer",
					"metadata": map[string]interface{}{
						"name":      mcpServerName,
						"namespace": namespace,
					},
					"spec": map[string]interface{}{
						"deployment": map[string]interface{}{
							"image": "docker.io/mcp/everything",
							"port":  3000,
							"cmd":   "npx",
							"args": []string{
								"-y",
								"@modelcontextprotocol/server-filesystem",
								"/",
							},
						},
						"transportType": "stdio",
					},
				},
			}

			cmd := exec.Command("kubectl", "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(mcpServerToYAML(mcpServer))
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create MCPServer")

			By("verifying the Deployment is created with correct configuration")
			Eventually(func(g Gomega) {
				deployment := getDeployment(mcpServerName, namespace)
				g.Expect(deployment).NotTo(BeNil())

				// Verify deployment has correct labels
				g.Expect(deployment.Spec.Selector.MatchLabels).To(Equal(map[string]string{
					"app.kubernetes.io/name":     mcpServerName,
					"app.kubernetes.io/instance": mcpServerName,
				}))

				// Verify pod template has correct labels
				g.Expect(deployment.Spec.Template.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", mcpServerName))
				g.Expect(deployment.Spec.Template.Labels).To(HaveKeyWithValue("app.kubernetes.io/instance", mcpServerName))
				g.Expect(deployment.Spec.Template.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "kmcp"))

				// Verify stdio transport configuration: init container + main container
				g.Expect(deployment.Spec.Template.Spec.InitContainers).To(HaveLen(1))
				g.Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))

				// Verify init container copies agentgateway binary
				initContainer := deployment.Spec.Template.Spec.InitContainers[0]
				g.Expect(initContainer.Name).To(Equal("copy-binary"))
				g.Expect(initContainer.Image).To(ContainSubstring("agentgateway"))
				g.Expect(initContainer.Command).To(ContainElement("sh"))
				g.Expect(initContainer.Args).To(ContainElement(ContainSubstring("cp /usr/bin/agentgateway /agentbin/agentgateway")))

				// Verify main container configuration
				mainContainer := deployment.Spec.Template.Spec.Containers[0]
				g.Expect(mainContainer.Name).To(Equal("mcp-server"))
				g.Expect(mainContainer.Image).To(Equal("docker.io/mcp/everything"))
				g.Expect(mainContainer.Command).To(ContainElement("sh"))
				g.Expect(mainContainer.Args).To(ContainElement(ContainSubstring("/agentbin/agentgateway -f /config/local.yaml")))

				// Verify volumes
				g.Expect(deployment.Spec.Template.Spec.Volumes).To(HaveLen(2))
				volumeNames := []string{}
				for _, volume := range deployment.Spec.Template.Spec.Volumes {
					volumeNames = append(volumeNames, volume.Name)
				}
				g.Expect(volumeNames).To(ContainElements("config", "binary"))
			}).Should(Succeed())

			By("verifying the Service is created with correct configuration")
			Eventually(func(g Gomega) {
				service := getService(mcpServerName, namespace)
				g.Expect(service).NotTo(BeNil())

				// Verify service selector
				g.Expect(service.Spec.Selector).To(Equal(map[string]string{
					"app.kubernetes.io/name":     mcpServerName,
					"app.kubernetes.io/instance": mcpServerName,
				}))

				// Verify service ports
				g.Expect(service.Spec.Ports).To(HaveLen(1))
				g.Expect(service.Spec.Ports[0].Name).To(Equal("http"))
				g.Expect(service.Spec.Ports[0].Port).To(Equal(int32(3000)))
				g.Expect(service.Spec.Ports[0].TargetPort.IntVal).To(Equal(int32(3000)))
				g.Expect(service.Spec.Ports[0].Protocol).To(Equal(corev1.ProtocolTCP))
			}).Should(Succeed())

			By("verifying the ConfigMap is created with correct configuration")
			Eventually(func(g Gomega) {
				configMap := getConfigMap(mcpServerName, namespace)
				g.Expect(configMap).NotTo(BeNil())

				// Verify configmap contains local.yaml
				g.Expect(configMap.Data).To(HaveKey("local.yaml"))

				// Parse and verify the configuration content
				configYaml := configMap.Data["local.yaml"]
				g.Expect(configYaml).To(ContainSubstring("config: {}"))
				g.Expect(configYaml).To(ContainSubstring("port: 3000"))
				g.Expect(configYaml).To(ContainSubstring("stdio:"))
				g.Expect(configYaml).To(ContainSubstring("cmd: npx"))
				g.Expect(configYaml).To(ContainSubstring("args:"))
				g.Expect(configYaml).To(ContainSubstring("- -y"))
				g.Expect(configYaml).To(ContainSubstring("- '@modelcontextprotocol/server-filesystem'"))
				g.Expect(configYaml).To(ContainSubstring("- /"))
			}).Should(Succeed())

			By("cleaning up the MCPServer")
			cmd = exec.Command("kubectl", "delete", "mcpserver", mcpServerName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() string {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	metricsOutput, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
	Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
	return metricsOutput
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}

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

func getConfigMap(name, namespace string) *corev1.ConfigMap {
	cmd := exec.Command("kubectl", "get", "configmap", name, "-n", namespace, "-o", "json")
	output, err := utils.Run(cmd)
	if err != nil {
		return nil
	}

	var configMap corev1.ConfigMap
	if err := json.Unmarshal([]byte(output), &configMap); err != nil {
		return nil
	}
	return &configMap
}

func mcpServerToYAML(mcpServer *unstructured.Unstructured) string {
	yamlBytes, err := yaml.Marshal(mcpServer.Object)
	if err != nil {
		return ""
	}
	return string(yamlBytes)
}
