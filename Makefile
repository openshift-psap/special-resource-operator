include Makefile.specialresource.mk
include Makefile.helm.mk
# For SRO specific options see:
include Makefile.sro.mk

# Current Operator version
VERSION ?= 0.0.1
# Default bundle image tag
BUNDLE_IMG ?= quay.io/openshift-psap/special-resource-operator-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
TAG ?= $(shell git branch --show-current)
IMG ?= quay.io/openshift-psap/special-resource-operator:$(TAG)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:crdVersions=v1,trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# GENERATED all: manager
all: $(SPECIALRESOURCE)

# Run tests
test: # generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: patch generate fmt vet
	go build -mod=vendor -o /tmp/bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run -mod=vendor ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

namespace: patch manifests kustomize
	$(KUSTOMIZE) build config/namespace | kubectl apply -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: namespace
	$(KUSTOMIZE) build config/default$(SUFFIX) | kubectl apply -f -
	$(shell sleep 5)
	$(KUSTOMIZE) build config/cr | kubectl apply -f -

# If the CRD is deleted before the CRs the CRD finalizer will hang forever
# The specialresource finalizer will not execute either
undeploy: kustomize
	if [ ! -z "$$(kubectl get crd | grep specialresource)" ]; then         \
		kubectl delete --ignore-not-found sr --all;                    \
	fi;
	# Give SRO time to reconcile
	sleep 10
	$(KUSTOMIZE) build config/namespace | kubectl delete --ignore-not-found -f -
	$(KUSTOMIZE) build config/default$(SUFFIX) | kubectl delete --ignore-not-found -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Populate manifests dir, and SRO specific customizations
manifests-gen: manifests kustomize configure
	cd $@; rm -f *.yaml
	cd $@; ( $(KUSTOMIZE) build ../config/namespace && echo "---" && $(KUSTOMIZE) build ../config/default$(SUFFIX) ) | $(CSPLIT)
	cd manifests; bash ../scripts/rename.sh
	cd manifests; $(KUSTOMIZE) build ../config/cr > 0017_specialresource_special-resource-preamble.yaml

# SRO specific configuration to set namespace of all manifests
configure:
	# TODO kustomize cannot set name of namespace according to settings, hack TODO
	cd config/namespace && sed -i 's/name: .*/name: $(NAMESPACE)/g' namespace.yaml
	cd config/namespace && $(KUSTOMIZE) edit set namespace $(NAMESPACE)
	cd config/default && $(KUSTOMIZE) edit set namespace $(NAMESPACE)
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet --mod=vendor ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the container image
local-image-build: patch helm-lint helm-repo-index test generate manifests-gen
	podman build -t $(IMG) -f Dockerfile.ubi8 --no-cache .

# Push the container image
local-image-push:
	podman push $(IMG)

# Download controller-gen locally if necessary
CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen:
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.5.0)

# Download kustomize locally if necessary
KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize:
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests kustomize
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --verbose --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	podman build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: opm
OPM = ./bin/opm
opm:
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.15.1/$(OS)-$(ARCH)-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= quay.io/openshift-psap/sro-catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool podman --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push the catalog image.
	$(MAKE) local-image-push IMG=$(CATALOG_IMG)

