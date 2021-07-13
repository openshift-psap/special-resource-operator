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
	cd manifests; $(KUSTOMIZE) build ../config/cr > 0016_specialresource_special-resource-preamble.yaml

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
	podman build -f Dockerfile.ubi8 --no-cache . -t $(IMG)

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
