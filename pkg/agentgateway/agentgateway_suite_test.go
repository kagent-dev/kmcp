package agentgateway_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAgentgateway(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agentgateway Suite")
}
