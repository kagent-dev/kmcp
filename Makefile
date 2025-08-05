
# Image configuration
DOCKER_REGISTRY ?= ghcr.io
BASE_IMAGE_REGISTRY ?= cgr.dev
DOCKER_REPO ?= kagent-dev/kmcp
HELM_REPO ?= oci://ghcr.io/kagent-dev

BUILD_DATE := $(shell date -u '+%Y-%m-%d')
GIT_COMMIT := $(shell git rev-parse --short HEAD || echo "unknown")
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/-dirty//' | grep v || echo "v0.0.1+$(GIT_COMMIT)")

# Local architecture detection to build for the current platform
LOCALARCH ?= $(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')

# Docker buildx configuration
BUILDKIT_VERSION = v0.23.0
BUILDX_NO_DEFAULT_ATTESTATIONS=1
BUILDX_BUILDER_NAME ?= kmcp-builder-$(BUILDKIT_VERSION)
DOCKER_BUILDER ?= docker buildx
DOCKER_BUILD_ARGS ?= --builder $(BUILDX_BUILDER_NAME) --pull --load --platform linux/$(LOCALARCH)

# Image configuration
CONTROLLER_IMAGE_NAME ?= controller
CONTROLLER_IMAGE_TAG ?= $(VERSION)
CONTROLLER_IMG ?= $(DOCKER_REGISTRY)/$(DOCKER_REPO)/$(CONTROLLER_IMAGE_NAME):$(CONTROLLER_IMAGE_TAG)

# Image URL to use all building/pushing image targets (backward compatibility)
IMG ?= $(CONTROLLER_IMG)
DIST_FOLDER ?= dist

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: helm-lint
helm-lint:
	helm lint helm/kmcp

.PHONY: helm-crd
helm-crd: manifests ## Generate Helm CRD template from the generated CRD definition.
	@echo "Generating Helm CRD template from config/crd/bases/kagent.dev_mcpservers.yaml"
	@mkdir -p helm/kmcp/templates/crds
	@echo '{{- if .Values.crd.create }}' > helm/kmcp/templates/crds/mcpserver-crd.yaml
	@awk '/^  name: mcpservers.kagent.dev$$/ { \
		print; \
		print "  labels:"; \
		print "    {{- include \"kmcp.labels\" . | nindent 4 }}"; \
		next; \
	} \
	{ print }' config/crd/bases/kagent.dev_mcpservers.yaml >> helm/kmcp/templates/crds/mcpserver-crd.yaml
	@echo '{{- end }}' >> helm/kmcp/templates/crds/mcpserver-crd.yaml
	@echo "Helm CRD template generated at helm/kmcp/templates/crds/mcpserver-crd.yaml"

.PHONY: helm-package
helm-package:
	mkdir -p $(DIST_FOLDER)
	@echo "Packaging Helm chart with version $(VERSION)..."
	@cp helm/kmcp/Chart.yaml helm/kmcp/Chart.yaml.bak
	@sed "s/^version: .*/version: $(VERSION)/" helm/kmcp/Chart.yaml.bak > helm/kmcp/Chart.yaml
	@helm package helm/kmcp --version $(VERSION) -d $(DIST_FOLDER)
	@mv helm/kmcp/Chart.yaml.bak helm/kmcp/Chart.yaml
	@echo "Helm package created: $(DIST_FOLDER)/kmcp-$(VERSION).tgz"

.PHONY: helm-cleanup
helm-cleanup: ## Clean up Helm chart packages
	rm -f $(DIST_FOLDER)/*.tgz

.PHONY: helm-build
helm-build: helm-lint helm-package ## Build and package the Helm chart
	@echo "Helm chart built successfully"

.PHONY: helm-publish
helm-publish: helm-package ## Publish Helm chart to OCI registry
	@echo "Publishing Helm chart to $(HELM_REPO)/kmcp/helm..."
	@helm push $(DIST_FOLDER)/kmcp-$(VERSION).tgz $(HELM_REPO)/kmcp/helm
	@echo "Helm chart published successfully"

.PHONY: helm-test
helm-test:
	@echo "Running Helm chart unit tests..."
	@command -v helm >/dev/null 2>&1 || { \
		echo "Helm is not installed. Please install Helm manually."; \
		exit 1; \
	}
	@helm plugin list | grep -q 'unittest' || { \
		echo "Installing helm-unittest plugin..."; \
		helm plugin install https://github.com/quintush/helm-unittest; \
	}
	@helm unittest helm/kmcp
	@echo "Helm chart unit tests completed successfully"

.PHONY: manifests-all
manifests-all: manifests ## Generate Kustomize from source definitions.
	@echo "All manifests (Kustomize and Helm) updated successfully"

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: goimports ## Run go fmt against code.
	go fmt ./...
	$(GOIMPORTS) -local "github.com/kgateway-dev/kmcp" -w .

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet setup-envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

# TODO(user): To use a different vendor for e2e tests, modify the setup under 'tests/e2e'.
# The default setup assumes Kind is pre-installed and builds/loads the Manager Docker image locally.
# Prometheus and CertManager are installed by default; skip with:
# - PROMETHEUS_INSTALL_SKIP=true
# - CERT_MANAGER_INSTALL_SKIP=true
.PHONY: test-e2e
test-e2e: manifests generate fmt vet ## Run the e2e tests. Expected an isolated environment using Kind.
	@command -v kind >/dev/null 2>&1 || { \
		echo "Kind is not installed. Please install Kind manually."; \
		exit 1; \
	}
	@kind get clusters | grep -q 'kind' || { \
		echo "No Kind cluster is running. Please start a Kind cluster before running the e2e tests."; \
		exit 1; \
	}
	go test ./test/e2e/ -v -ginkgo.v -ginkgo.show-node-events

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: lint-config
lint-config: golangci-lint ## Verify golangci-lint linter configuration
	$(GOLANGCI_LINT) config verify

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: build-cli
build-cli: fmt vet ## Build kmcp CLI binary.
	mkdir -p $(DIST_FOLDER)
	go build -ldflags="-X 'github.com/kagent-dev/kmcp/cmd/kmcp/cmd.Version=$(VERSION)'" -o $(DIST_FOLDER)/kmcp cmd/kmcp/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	- $(DOCKER_BUILDER) create --name $(BUILDX_BUILDER_NAME)
	$(DOCKER_BUILDER) use $(BUILDX_BUILDER_NAME)
	$(DOCKER_BUILDER) build $(DOCKER_BUILD_ARGS) -t ${CONTROLLER_IMG} .
	- $(DOCKER_BUILDER) rm $(BUILDX_BUILDER_NAME)

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(DOCKER_BUILDER) push ${CONTROLLER_IMG}


##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOIMPORTS ?= $(LOCALBIN)/goimports

## Tool Versions
KUSTOMIZE_VERSION ?= v5.5.0
CONTROLLER_TOOLS_VERSION ?= v0.17.1
#ENVTEST_VERSION is the version of controller-runtime release branch to fetch the envtest setup script (i.e. release-0.20)
ENVTEST_VERSION ?= $(shell go list -m -f "{{ .Version }}" sigs.k8s.io/controller-runtime | awk -F'[v.]' '{printf "release-%d.%d", $$2, $$3}')
#ENVTEST_K8S_VERSION is the version of Kubernetes to use for setting up ENVTEST binaries (i.e. 1.31)
ENVTEST_K8S_VERSION ?= $(shell go list -m -f "{{ .Version }}" k8s.io/api | awk -F'[v.]' '{printf "1.%d", $$3}')
GOLANGCI_LINT_VERSION ?= v1.63.4
GOIMPORTS_VERSION ?= v0.26.0


.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: setup-envtest
setup-envtest: envtest ## Download the binaries required for ENVTEST in the local bin directory.
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@$(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: goimports
goimports: $(GOIMPORTS) ## Download goimports locally if necessary.
$(GOIMPORTS): $(LOCALBIN)
	$(call go-install-tool,$(GOIMPORTS),golang.org/x/tools/cmd/goimports,$(GOIMPORTS_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
