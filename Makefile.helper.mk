
# SRO-specific options

SPECIALRESOURCE  ?= driver-container-base
NAMESPACE        ?= openshift-special-resource-operator
PULLPOLICY       ?= IfNotPresent
TAG              ?= $(shell git branch --show-current)
IMAGE            ?= quay.io/openshift-psap/special-resource-operator:$(TAG)
CSPLIT           ?= csplit - --prefix="" --suppress-matched --suffix-format="%04d.yaml"  /---/ '{*}' --silent
YAMLFILES        ?= $(shell  find manifests charts -name "*.yaml"  -not \( -path "charts/lustre/lustre-aws-fsx-0.0.1/csi-driver/*" -prune \)  -not \( -path "charts/*/shipwright-*/*" -prune \) -not \( -path "charts/experimental/*" -prune \) )
PLATFORM         ?= ""
SUFFIX           ?= $(shell if [ ${PLATFORM} == "k8s" ]; then echo "-${PLATFORM}"; fi)

export PATH := go/bin:$(PATH)

patch:
	cp .patches/options.patch.go vendor/github.com/google/go-containerregistry/pkg/crane/.
	cp .patches/getter.patch.go vendor/helm.sh/helm/v3/pkg/getter/.
#	cp .patches/zapr.patch.go vendor/github.com/go-logr/zapr/.

kube-lint: kube-linter
	$(KUBELINTER) lint $(YAMLFILES)

lint: golangci-lint
	$(GOLANGCILINT) run -v --timeout 5m0s

verify: fmt vet
unit:
	@echo "TODO UNIT TEST"

go-deploy-manifests: manifests-gen
	go run test/deploy/deploy.go -path ./manifests

go-undeploy-manifests:
	go run test/undeploy/undeploy.go -path ./manifests

test-e2e-upgrade: go-deploy-manifests

test-e2e:
	for d in basic; do \
          KUBERNETES_CONFIG="$(KUBECONFIG)" go test -v -timeout 40m ./test/e2e/$$d -ginkgo.v -ginkgo.noColor -ginkgo.failFast || exit; \
        done


# Download kube-linter locally if necessary
KUBELINTER = $(shell pwd)/bin/kube-linter
kube-linter:
	$(call go-get-tool,$(KUBELINTER),golang.stackrox.io/kube-linter/cmd/kube-linter@v0.0.0-20210328011908-cb34f2cc447f)

# Download golangci-lint locally if necessary
GOLANGCILINT = $(shell pwd)/bin/golangci-lint
golangci-lint:
	$(call go-get-tool,$(GOLANGCILINT),github.com/golangci/golangci-lint/cmd/golangci-lint@v1.33.0)

# Additional bundle options for ART
DEFAULT_CHANNEL="4.9"
CHANNELS="4.9"

update-bundle:
	mv $$(find bundle -name image-references) bundle/image-references
	rm -rf bundle/4.*/manifests bundle/4.*/metadata
	$(MAKE) bundle DEFAULT_CHANNEL=$(DEFAULT_CHANNEL) VERSION=$(VERSION) IMAGE=$(IMAGE)
	mv bundle/manifests/special-resource-operator.clusterserviceversion.yaml bundle/manifests/special-resource-operator.v$(VERSION).clusterserviceversion.yaml
	mv bundle/manifests bundle/$(DEFAULT_CHANNEL)/manifests
	mv bundle/metadata bundle/$(DEFAULT_CHANNEL)/metadata
	sed 's#bundle/##g' bundle.Dockerfile | head -n -1 > bundle/$(DEFAULT_CHANNEL)/bundle.Dockerfile
	mv bundle/image-references bundle/$(DEFAULT_CHANNEL)/manifests/image-references
