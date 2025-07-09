package agentgateway_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kagent.dev/kmcp/api/v1alpha1"
	"kagent.dev/kmcp/pkg/agentgateway"
	"os"
	"sigs.k8s.io/yaml"
)

var _ = Describe("AgentgatewayTranslator", func() {
	It("handles stdio", func() {
		agt := agentgateway.NewAgentGatewayTranslator()

		outputs, err := agt.TranslateAgentGatewayOutputs(&v1alpha1.MCPServer{
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
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(outputs).NotTo(BeNil())

		writeYamlToFile(outputs.Deployment, "agentgateway_deployment_stdio.yaml")
		writeYamlToFile(outputs.Service, "agentgateway_service_stdio.yaml")
		writeYamlToFile(outputs.ConfigMap, "agentgateway_configmap_stdio.yaml")
	})
	It("handles http", func() {
		agt := agentgateway.NewAgentGatewayTranslator()

		outputs, err := agt.TranslateAgentGatewayOutputs(&v1alpha1.MCPServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-http-server",
				Namespace: "default",
			},
			Spec: v1alpha1.MCPServerSpec{
				Deployment: v1alpha1.MCPServerDeployment{
					Image: "docker.io/mcp/everything",
					Port:  3001,
					Cmd:   "npm",
					Args: []string{
						"run",
						"start:streamableHttp",
					},
					Env: nil,
				},
				TransportType: v1alpha1.TransportTypeStdio,
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
