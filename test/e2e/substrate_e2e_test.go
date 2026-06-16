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
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/kagent-dev/kmcp/test/utils"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// Substrate e2e tests are gated behind KMCP_SUBSTRATE_E2E=true and assume the
// environment prepared per SUBSTRATE_PLAN.md:
//   - a kind cluster with substrate installed (the kagent-dev/substrate fork)
//   - kmcp deployed with substrate.enabled=true (managed-proxy ingress mode)
//     and a default WorkerPool
//   - the test namespace (default kmcp-test) with the WorkerPool applied
//   - KMCP_SUBSTRATE_TEST_IMAGE set to a digest-pinned image that can run
//     `npx @modelcontextprotocol/server-everything` (e.g. a pinned node image)
//
// Optional environment:
//   - KMCP_NAMESPACE: namespace of the kmcp installation (default kmcp-system)
//   - KMCP_SUBSTRATE_PROXY_DEPLOYMENT: name of the shared ingress proxy
//     Deployment (default kmcp-substrate-ingress-proxy)
var _ = ginkgo.Describe("substrate runtime", ginkgo.Ordered, func() {
	var (
		testNamespace   string
		testImage       string
		kmcpNamespace   string
		proxyDeployment string
	)

	ginkgo.BeforeAll(func() {
		if os.Getenv("KMCP_SUBSTRATE_E2E") != "true" {
			ginkgo.Skip("set KMCP_SUBSTRATE_E2E=true to run substrate e2e tests")
		}
		testImage = os.Getenv("KMCP_SUBSTRATE_TEST_IMAGE")
		if testImage == "" || !strings.Contains(testImage, "@") {
			ginkgo.Skip("set KMCP_SUBSTRATE_TEST_IMAGE to a digest-pinned image")
		}
		testNamespace = os.Getenv("KMCP_SUBSTRATE_TEST_NAMESPACE")
		if testNamespace == "" {
			testNamespace = "kmcp-test"
		}
		kmcpNamespace = os.Getenv("KMCP_NAMESPACE")
		if kmcpNamespace == "" {
			kmcpNamespace = "kmcp-system"
		}
		proxyDeployment = os.Getenv("KMCP_SUBSTRATE_PROXY_DEPLOYMENT")
		if proxyDeployment == "" {
			proxyDeployment = "kmcp-substrate-ingress-proxy"
		}
	})

	applyMCPServer := func(name, transportType string) {
		var transportBlock, argsList string
		if transportType == "http" {
			transportBlock = "httpTransport:\n    targetPort: 3001\n    path: /mcp"
			argsList = `["-y", "@modelcontextprotocol/server-everything", "streamableHttp"]`
		} else {
			transportBlock = "stdioTransport: {}"
			argsList = `["-y", "@modelcontextprotocol/server-everything"]`
		}
		manifest := fmt.Sprintf(`apiVersion: kagent.dev/v1alpha1
kind: MCPServer
metadata:
  name: %s
  namespace: %s
spec:
  runtime: substrate
  substrate: {}
  transportType: %s
  %s
  deployment:
    image: %s
    cmd: npx
    args: %s
    port: 3000
`, name, testNamespace, transportType, transportBlock, testImage, argsList)

		cmd := exec.Command("kubectl", "apply", "-f", "-")
		cmd.Stdin = strings.NewReader(manifest)
		_, err := utils.Run(cmd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to apply MCPServer manifest")
	}

	waitReady := func(name string) {
		ginkgo.By("waiting for the MCPServer to become Ready (the golden snapshot bake is the slow part)")
		gomega.Eventually(func(g gomega.Gomega) {
			cmd := exec.Command("kubectl", "get", "mcpserver", name, "-n", testNamespace,
				"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
			out, err := utils.Run(cmd)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(strings.TrimSpace(out)).To(gomega.Equal("True"))
		}, 5*time.Minute, 10*time.Second).Should(gomega.Succeed())
	}

	assertSubstrateStatus := func(name string) {
		jsonPath := "{.status.substrate.actorID} {.status.substrate.actorHost} " +
			"{.status.substrate.routerURL} {.status.substrate.mcpPath}"
		cmd := exec.Command("kubectl", "get", "mcpserver", name, "-n", testNamespace, "-o", "jsonpath="+jsonPath)
		out, err := utils.Run(cmd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		fields := strings.Fields(out)
		gomega.Expect(fields).To(gomega.HaveLen(4), "status.substrate should publish routing coordinates: %q", out)
		gomega.Expect(fields[1]).To(gomega.HavePrefix(fields[0]+"."), "actorHost should be derived from actorID")
		gomega.Expect(fields[3]).To(gomega.Equal("/mcp"))
	}

	// assertSharedIngressObjects checks the managed-proxy plumbing: the
	// selector-less Service, the controller-managed EndpointSlice carrying the
	// proxy pod IPs, and the server's route in the shared ConfigMap.
	assertSharedIngressObjects := func(name string) {
		ginkgo.By("checking the selector-less Service")
		cmd := exec.Command("kubectl", "get", "service", name, "-n", testNamespace,
			"-o", "jsonpath={.spec.selector}")
		out, err := utils.Run(cmd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(strings.TrimSpace(out)).To(gomega.BeEmpty(), "Service must have no selector")

		ginkgo.By("checking the controller-managed EndpointSlice has ready endpoints")
		cmd = exec.Command("kubectl", "get", "endpointslice", name+"-proxy", "-n", testNamespace,
			"-o", "jsonpath={.endpoints[*].addresses[0]}")
		out, err = utils.Run(cmd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(strings.Fields(out)).NotTo(gomega.BeEmpty(), "EndpointSlice should carry proxy pod IPs")

		ginkgo.By("checking the server's route in the shared proxy ConfigMap")
		cmd = exec.Command("kubectl", "get", "configmap", proxyDeployment, "-n", kmcpNamespace,
			"-o", "jsonpath={.data.local\\.yaml}")
		out, err = utils.Run(cmd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(out).To(gomega.ContainSubstring(fmt.Sprintf("%s.%s.svc.cluster.local", name, testNamespace)))
	}

	// mcpRoundTrip exercises a full MCP handshake through the shared ingress
	// proxy, addressing the server by its Service hostname (the proxy routes
	// on the Host header; the in-cluster Service path is asserted separately).
	mcpRoundTrip := func(name string, localPort int) {
		ginkgo.By("port-forwarding to the shared ingress proxy")
		portForward := exec.Command("kubectl", "port-forward",
			fmt.Sprintf("deployment/%s", proxyDeployment), fmt.Sprintf("%d:8080", localPort), "-n", kmcpNamespace)
		gomega.Expect(portForward.Start()).To(gomega.Succeed())
		defer func() {
			if portForward.Process != nil {
				_ = portForward.Process.Kill()
			}
		}()

		gomega.Eventually(func() error {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d", localPort))
			if err != nil {
				return err
			}
			_ = resp.Body.Close()
			return nil
		}, 30*time.Second, 1*time.Second).Should(gomega.Succeed())

		// Pin the Host header so requests through the local port-forward to
		// the shared proxy still match the server's route.
		serviceHost := fmt.Sprintf("%s.%s.svc", name, testNamespace)

		ginkgo.By("waiting for the server's route to be live on the proxy (ConfigMap sync takes up to ~1 minute)")
		gomega.Eventually(func(g gomega.Gomega) {
			req, err := http.NewRequest(http.MethodPost,
				fmt.Sprintf("http://localhost:%d/mcp", localPort), strings.NewReader("{}"))
			g.Expect(err).NotTo(gomega.HaveOccurred())
			req.Host = serviceHost
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			resp, err := http.DefaultClient.Do(req)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			defer func() { _ = resp.Body.Close() }()
			// Any status but 404 means the proxy matched the route; the MCP
			// round trip below verifies the path end to end.
			g.Expect(resp.StatusCode).NotTo(gomega.Equal(http.StatusNotFound),
				"the proxy has not hot-reloaded the new route yet")
		}, 5*time.Minute, 5*time.Second).Should(gomega.Succeed())

		ginkgo.By("running an MCP initialize + tools/list round trip (resumes the actor if suspended)")
		httpClient := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.Host = serviceHost
			return http.DefaultTransport.RoundTrip(req)
		})}
		mcpClient, err := client.NewStreamableHttpClient(fmt.Sprintf("http://localhost:%d/mcp", localPort),
			transport.WithHTTPBasicClient(httpClient))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		// Generous timeout: the first request through the atenet router may
		// buffer while the actor resumes from its snapshot.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				ClientInfo:      mcp.Implementation{Name: "kmcp-substrate-e2e", Version: "1.0.0"},
			},
		})
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "MCP initialize failed")

		tools, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "MCP tools/list failed")
		gomega.Expect(tools.Tools).NotTo(gomega.BeEmpty())
	}

	// serviceContractRoundTrip proves the normal in-cluster Service URL works
	// end to end (Service -> EndpointSlice -> shared proxy -> router -> actor)
	// by POSTing an MCP initialize from a throwaway pod.
	serviceContractRoundTrip := func(name string) {
		ginkgo.By("running an in-cluster MCP initialize against the Service URL")
		url := fmt.Sprintf("http://%s.%s.svc:3000/mcp", name, testNamespace)
		initialize := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{` +
			`"protocolVersion":"2025-03-26",` +
			`"capabilities":{},` +
			`"clientInfo":{"name":"kmcp-substrate-e2e-curl","version":"1.0.0"}}}`
		gomega.Eventually(func(g gomega.Gomega) {
			cmd := exec.Command("kubectl", "run", fmt.Sprintf("%s-curl", name),
				"-n", testNamespace, "--rm", "--attach", "--restart=Never", "--quiet",
				"--image=curlimages/curl:8.7.1", "--command", "--",
				"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
				"--max-time", "120",
				"-X", "POST", url,
				"-H", "content-type: application/json",
				"-H", "accept: application/json, text/event-stream",
				"-d", initialize)
			out, err := utils.Run(cmd)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(strings.TrimSpace(out)).To(gomega.Equal("200"),
				"in-cluster initialize through the Service URL should return 200")
		}, 5*time.Minute, 10*time.Second).Should(gomega.Succeed())
	}

	deleteAndAssertCleanup := func(name string) {
		ginkgo.By("deleting the MCPServer and waiting for substrate cleanup")
		cmd := exec.Command("kubectl", "delete", "mcpserver", name, "-n", testNamespace, "--wait=true", "--timeout=5m")
		_, err := utils.Run(cmd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		gomega.Eventually(func(g gomega.Gomega) {
			cmd := exec.Command("kubectl", "get", "actortemplate", name, "-n", testNamespace)
			_, err := utils.Run(cmd)
			g.Expect(err).To(gomega.HaveOccurred(), "ActorTemplate should be garbage-collected")
		}, 2*time.Minute, 5*time.Second).Should(gomega.Succeed())

		ginkgo.By("checking the route is removed from the shared proxy ConfigMap")
		gomega.Eventually(func(g gomega.Gomega) {
			cmd := exec.Command("kubectl", "get", "configmap", proxyDeployment, "-n", kmcpNamespace,
				"-o", "jsonpath={.data.local\\.yaml}")
			out, err := utils.Run(cmd)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(out).NotTo(gomega.ContainSubstring(fmt.Sprintf("%s.%s.svc", name, testNamespace)))
		}, 2*time.Minute, 5*time.Second).Should(gomega.Succeed())
	}

	ginkgo.It("serves streamable HTTP through the shared ingress proxy", func() {
		const name = "substrate-e2e-http"
		applyMCPServer(name, "http")
		waitReady(name)
		assertSubstrateStatus(name)
		assertSharedIngressObjects(name)
		mcpRoundTrip(name, 18091)
		serviceContractRoundTrip(name)

		// Scale-from-zero: suspend the actor (when kubectl-ate is available)
		// and prove the next request resumes it transparently.
		if _, err := exec.LookPath("kubectl-ate"); err == nil {
			cmd := exec.Command("kubectl", "get", "mcpserver", name, "-n", testNamespace,
				"-o", "jsonpath={.status.substrate.actorID}")
			actorID, err := utils.Run(cmd)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("suspending the actor and re-running the MCP round trip")
			cmd = exec.Command("kubectl", "ate", "suspend", "actor", strings.TrimSpace(actorID))
			_, err = utils.Run(cmd)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to suspend actor")
			mcpRoundTrip(name, 18092)
		}

		deleteAndAssertCleanup(name)
	})

	ginkgo.It("serves stdio MCP servers through the in-actor adapter", func() {
		const name = "substrate-e2e-stdio"
		applyMCPServer(name, "stdio")
		waitReady(name)
		assertSubstrateStatus(name)
		mcpRoundTrip(name, 18093)
		deleteAndAssertCleanup(name)
	})
})

// roundTripperFunc adapts a function to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
