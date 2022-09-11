
# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/suffiks/suffiks:latest
KIND_CLUSTER ?= kind

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: manifests build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager ./cmd/suffiks/main.go

run: manifests generate ## Run a controller from your host.
	go run --workfile=off ./cmd/suffiks/main.go

docker-build: #test ## Build docker image with the manager.
	docker build -t ${IMG} .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

kind: docker-build
	kind load docker-image --name ${KIND_CLUSTER} ${IMG}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

##@ extensions
gen-extensions:
	mkdir -p extension/protogen
	protoc \
		-I extension/proto/thirdparty \
		-I extension/proto/ \
		--go_opt=Mk8s.io/api/core/v1/generated.proto=k8s.io/api/core/v1 \
		--go_opt=Mk8s.io/apimachinery/pkg/api/resource/generated.proto=k8s.io/apimachinery/pkg/api/resource \
		--go_opt=Mk8s.io/apimachinery/pkg/apis/meta/v1/generated.proto=k8s.io/apimachinery/pkg/apis/meta/v1 \
		--go_opt=Mk8s.io/apimachinery/pkg/runtime/generated.proto=k8s.io/apimachinery/pkg/runtime \
		--go_opt=Mk8s.io/apimachinery/pkg/runtime/schema/generated.proto=k8s.io/apimachinery/pkg/runtime/schema \
		--go_opt=Mk8s.io/apimachinery/pkg/util/intstr/generated.proto=k8s.io/apimachinery/pkg/util/intstr \
		./extension/proto/extension.proto \
		--go_out=. \
		--go-grpc_out=.

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN): ## Ensure that the directory exists
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.9.2
ENVTEST_VERSION ?= 1.25.0

KUSTOMIZE_BUNDLE = kustomize_$(KUSTOMIZE_VERSION)_$(shell go env GOOS)_$(shell go env GOARCH).tar.gz
KUSTOMIZE_URL = https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2F$(KUSTOMIZE_VERSION)/$(KUSTOMIZE_BUNDLE)
.PHONY: kustomize
kustomize: $(KUSTOMIZE)
$(KUSTOMIZE): ## Download kustomize locally if necessary.
	curl -sLO $(KUSTOMIZE_URL)
	tar xf $(KUSTOMIZE_BUNDLE) --directory $(LOCALBIN)
	rm $(KUSTOMIZE_BUNDLE)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN)
$(CONTROLLER_GEN): ## Download controller-gen locally if necessary.
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST)
$(ENVTEST): ## Download envtest-setup locally if necessary.
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION)

.PHONY: clientset
clientset: TMPDIR := $(shell mktemp -d)
clientset: ## Generate clientset
	go run k8s.io/code-generator/cmd/client-gen \
		--clientset-name versioned \
		--input-base=github.com/suffiks/suffiks/apis --input-dirs=. --input="suffiks/v1" \
		--output-package=github.com/suffiks/suffiks/pkg/client/clientset_generated/ --output-base=$(TMPDIR) \
		--go-header-file hack/boilerplate.go.txt
	cp -r $(TMPDIR)/github.com/suffiks/suffiks/pkg .
# Application is special, so reference the base package
	sed -i 's|github.com/suffiks/suffiks/apis/suffiks/v1|github.com/suffiks/suffiks/base|' pkg/client/clientset_generated/versioned/typed/suffiks/v1/application.go
	sed -i 's|github.com/suffiks/suffiks/apis/suffiks/v1|github.com/suffiks/suffiks/base|' pkg/client/clientset_generated/versioned/typed/suffiks/v1/fake/fake_application.go
	rm -rf $(TMPDIR)
