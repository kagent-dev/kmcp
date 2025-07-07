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
	It("do", func() {
		agt := agentgateway.NewAgentGatewayTranslator()

		outputs, err := agt.TranslateAgentGatewayOutputs(&v1alpha1.MCPServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-server",
				Namespace: "default",
			},
			Spec: v1alpha1.MCPServerSpec{
				TransportType: v1alpha1.TransportTypeStdio,
				StdioTransport: &v1alpha1.StdioTransport{
					Cmd: "npx",
					Args: []string{
						"-y",
						"@modelcontextprotocol/server-filesystem",
						"/",
					},
					Env: nil,
				},
				//DeploymentOverrides: nil,
				Port: 3000,
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(outputs).NotTo(BeNil())

		writeYamlToFile(outputs.Deployment, "agentgateway_deployment.yaml")
		writeYamlToFile(outputs.Service, "agentgateway_service.yaml")
		writeYamlToFile(outputs.ConfigMap, "agentgateway_configmap.yaml")
	})
})

func writeYamlToFile(data interface{}, filename string) {
	yamlData, err := yaml.Marshal(data)
	Expect(err).NotTo(HaveOccurred())
	err = os.WriteFile(filename, yamlData, 0644)
	Expect(err).NotTo(HaveOccurred())
}
