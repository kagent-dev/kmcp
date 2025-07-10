package agentgateway_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"kagent.dev/kmcp/api/v1alpha1"
	"kagent.dev/kmcp/pkg/agentgateway"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
)

var _ = Describe("AgentgatewayTranslator", func() {
	It("handles stdio", func() {
		scheme := runtime.NewScheme()
		err := v1alpha1.AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())

		// Add apiextensions scheme to handle CRDs
		err = apiextensionsv1.AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())

		// Apply CRDs to cluster
		kube, err := client.New(config.GetConfigOrDie(), client.Options{Scheme: scheme})
		Expect(err).NotTo(HaveOccurred())

		// Read and apply the CRD file
		crdData, err := os.ReadFile(getProjectRoot() + "/config/crd/bases/kagent.dev_mcpservers.yaml")
		Expect(err).NotTo(HaveOccurred())

		var crd apiextensionsv1.CustomResourceDefinition
		err = yaml.Unmarshal(crdData, &crd)
		Expect(err).NotTo(HaveOccurred())

		err = kube.Create(context.TODO(), &crd)
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		} else if err == nil {
			// sleep to ensure CRD is ready
			time.Sleep(2 * time.Second)
		}

		agt := agentgateway.NewAgentGatewayTranslator(scheme)

		server := &v1alpha1.MCPServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-stdio-server",
				Namespace: "default",
			},
			Spec: v1alpha1.MCPServerSpec{
				Deployment: v1alpha1.MCPServerDeployment{
					Image: "docker.io/mcp/everything",
					Port:  3000,
					Cmd:   "npx",
					Args: []string{
						"-y",
						"@modelcontextprotocol/server-filesystem",
						"/",
					},
				},
				TransportType: v1alpha1.TransportTypeStdio,
			},
		}

		// kube apply the server
		err = kube.Create(context.TODO(), server)
		Expect(err).NotTo(HaveOccurred())

		outputs, err := agt.TranslateAgentGatewayOutputs(server)
		Expect(err).NotTo(HaveOccurred())
		Expect(outputs).NotTo(BeNil())

		writeYamlToFile(outputs.Deployment, "agentgateway_deployment_stdio.yaml")
		writeYamlToFile(outputs.Service, "agentgateway_service_stdio.yaml")
		writeYamlToFile(outputs.ConfigMap, "agentgateway_configmap_stdio.yaml")
	})
	It("handles http", func() {
		scheme := runtime.NewScheme()
		v1alpha1.AddToScheme(scheme)
		agt := agentgateway.NewAgentGatewayTranslator(scheme)

		outputs, err := agt.TranslateAgentGatewayOutputs(&v1alpha1.MCPServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-http-server",
				Namespace: "default",
			},
			Spec: v1alpha1.MCPServerSpec{
				Deployment: v1alpha1.MCPServerDeployment{
					Image: "docker.io/mcp/everything",
					Port:  3000,
					Cmd:   "npm",
					Args: []string{
						"run",
						"start:streamableHttp",
					},
					Env: nil,
				},
				TransportType: v1alpha1.TransportTypeHTTP,
				HTTPTransport: &v1alpha1.HTTPTransport{TargetPort: 3001},
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(outputs).NotTo(BeNil())

		writeYamlToFile(outputs.Deployment, "agentgateway_deployment_http.yaml")
		writeYamlToFile(outputs.Service, "agentgateway_service_http.yaml")
		writeYamlToFile(outputs.ConfigMap, "agentgateway_configmap_http.yaml")
	})
})

func writeYamlToFile(data interface{}, filename string) {
	yamlData, err := yaml.Marshal(data)
	Expect(err).NotTo(HaveOccurred())
	err = os.WriteFile(filename, yamlData, 0644)
	Expect(err).NotTo(HaveOccurred())
}

// absolute path to go.mod file for current dir
func getProjectRoot() string {
	out, err := exec.Command("go", "env", "GOMOD").CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	return filepath.Dir(strings.TrimSpace(string(out)))
}
