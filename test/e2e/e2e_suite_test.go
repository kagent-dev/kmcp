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
	"fmt"
	"os/exec"
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"

	"kagent.dev/kmcp/test/utils"
)

var (

	// projectImage is the name of the image which will be build and loaded
	// with the code source changes to be tested.
	projectImage = "ghcr.io/kagent-dev/kmcp/controller:v0.0.1"
)

// TestE2E runs the end-to-end (e2e) test suite for the project. These tests execute in an isolated,
// temporary environment to validate project changes with the the purposed to be used in CI jobs.
// The default setup requires Kind, builds/loads the Manager Docker image locally, and installs
// CertManager and Prometheus.
func TestE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "Starting kmcp integration test suite\n")
	ginkgo.RunSpecs(t, "e2e suite")
}

var _ = ginkgo.BeforeSuite(func() {

	ginkgo.By("building the manager(Operator) image")
	cmd := exec.Command("make", "docker-build", fmt.Sprintf("CONTROLLER_IMG=%s", projectImage))
	_, err := utils.Run(cmd)
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred(), "Failed to build the manager(Operator) image")

	// TODO(user): If you want to change the e2e test vendor from Kind, ensure the image is
	// built and available before running the tests. Also, remove the following block.
	ginkgo.By("loading the manager(Operator) image on Kind")
	err = utils.LoadImageToKindClusterWithName(projectImage)
	gomega.ExpectWithOffset(1, err).NotTo(gomega.HaveOccurred(), "Failed to load the manager(Operator) image into Kind")

})
