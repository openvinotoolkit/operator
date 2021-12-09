SHELL = /bin/bash

IMG ?= registry.toolbox.iotg.sclab.intel.com/cpp/openvino-operator


BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
IS_OPENSHIFT =? false 

# Build-time variables to inject into binaries
export GIT_COMMIT = $(shell git rev-parse HEAD)
export K8S_VERSION = 1.20.2

# Build settings
export TOOLS_DIR = tools/bin
export SCRIPTS_DIR = tools/scripts
REPO = $(shell go list -m)
BUILD_DIR = build
GO_ASMFLAGS = -asmflags "all=-trimpath=$(shell dirname $(PWD))"
GO_GCFLAGS = -gcflags "all=-trimpath=$(shell dirname $(PWD))"
GO_BUILD_ARGS = \
  $(GO_GCFLAGS) $(GO_ASMFLAGS)

export GO111MODULE = on
export CGO_ENABLED = 0
export PATH := $(PWD)/$(BUILD_DIR):$(PATH)
TAG ?= latest

##@ Development

.PHONY: fix
fix: ## Fixup files in the repo.
	go mod tidy
	go fmt ./...

.PHONY: clean
clean: ## Cleanup build artifacts and tool binaries.
	rm -rf $(BUILD_DIR) dist $(TOOLS_DIR)

##@ Build

.PHONY: build
build: ## Build operator-sdk, ansible-operator, and helm-operator.
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_ARGS) -o $(BUILD_DIR)/openvino-operator ./cmd

run: build # Run against the configured Kubernetes cluster in ~/.kube/config
	./$(BUILD_DIR)/openvino-operator run

docker-build: ## Build docker image with the manager.
	docker build -t ${IMG}:${TAG} -f docker/Dockerfile .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}:${TAG} 


BUNDLE_REPOSITORY ?= registry.toolbox.iotg.sclab.intel.com/cpp/openvino-operator-bundle
CATALOG_REPOSITORY ?= registry.toolbox.iotg.sclab.intel.com/cpp/openvino-operator-bundle

bundle_build:
ifeq ($(IS_OPENSHIFT), true)
	echo "Building openshift bundle"
	sed -i "s|registry.connect.redhat.com/intel/ovms-operator:0.2.0|$(IMG):$(TAG)|" bundle/manifests/openvino-operator.clusterserviceversion.yaml
	sed -i "s|gcr.io/kubebuilder/kube-rbac-proxy:v0.8.0|registry.redhat.io/openshift4/ose-kube-rbac-proxy:v4.8.0|" bundle/manifests/openvino-operator.clusterserviceversion.yaml
	docker build -t $(BUNDLE_REPOSITORY):$(TAG) -f bundle/Dockerfile bundle
	sed -i "s|$(IMG):$(TAG)|registry.connect.redhat.com/intel/ovms-operator:0.2.0|" bundle/manifests/openvino-operator.clusterserviceversion.yaml
	sed -i "s|registry.redhat.io/openshift4/ose-kube-rbac-proxy:v4.8.0|gcr.io/kubebuilder/kube-rbac-proxy:v0.8.0|" bundle/manifests/openvino-operator.clusterserviceversion.yaml
else
	echo "Building kubernetes bundle"
	sed -i "s|registry.connect.redhat.com/intel/ovms-operator:0.2.0|$(IMG):$(TAG)|" bundle/manifests/openvino-operator.clusterserviceversion.yaml
	docker build -t $(BUNDLE_REPOSITORY)-k8s:$(TAG) -f bundle/Dockerfile bundle
	sed -i "s|$(IMG):$(TAG)|registry.connect.redhat.com/intel/ovms-operator:0.2.0|" bundle/manifests/openvino-operator.clusterserviceversion.yaml
endif

bundle_push:
ifeq ($(IS_OPENSHIFT), true)
	docker push $(BUNDLE_REPOSITORY):$(TAG)
else
	docker push $(BUNDLE_REPOSITORY)-k8s:$(TAG)
endif

catalog_build:
ifeq ($(IS_OPENSHIFT), true)
	docker -v | grep -q podman ; if [ $$? -eq 0 ]; then \
	opm index add --bundles $(BUNDLE_REPOSITORY):$(TAG) --from-index registry.redhat.io/redhat/community-operator-index:v4.8 -c podman --tag $(CATALOG_REPOSITORY):$(TAG) ;\
	else sudo opm index add --bundles $(BUNDLE_REPOSITORY):$(TAG) --from-index registry.redhat.io/redhat/community-operator-index:v4.8 -c docker --tag $(CATALOG_REPOSITORY):$(TAG) ;\
    fi
	docker tag $(CATALOG_REPOSITORY):$(TAG) $(CATALOG_REPOSITORY):$(BRANCH)-latest 
else
	sudo opm index add --bundles $(BUNDLE_REPOSITORY)-k8s:$(TAG) --from-index quay.io/operatorhubio/catalog:latest -c docker --tag $(CATALOG_REPOSITORY)-k8s:$(TAG)
	docker tag $(CATALOG_REPOSITORY)-k8s:$(TAG) $(CATALOG_REPOSITORY)-k8s:$(BRANCH)-latest 	
endif

catalog_push:
ifeq ($(IS_OPENSHIFT), true)
	docker push $(CATALOG_REPOSITORY):$(TAG)
	docker push $(CATALOG_REPOSITORY):$(BRANCH)-latest 
else
	docker push $(CATALOG_REPOSITORY)-k8s:$(TAG)
	docker push $(CATALOG_REPOSITORY)-k8s:$(BRANCH)-latest 
endif


install: kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}:${TAG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

all:  build_all_images deploy_catalog deploy_operator

build_all_images: docker-build docker-push bundle_build bundle_push catalog_build catalog_push

build_bundle_catalog_images: bundle_build bundle_push catalog_build catalog_push

cluster_clean:
ifeq ($(IS_OPENSHIFT), true)
	echo "Skipping cleanup"
else
	kubectl delete --ignore-not-found=true ns operator
	kubectl get ns olm ; if [ $$? -eq 0 ]; then operator-sdk olm uninstall ; fi
	operator-sdk olm install
endif

deploy_catalog:
ifeq ($(IS_OPENSHIFT), true)
	kubectl delete --ignore-not-found=true -n openshift-marketplace --ignore-not-found=true CatalogSource my-test-catalog
	sleep 10
	cat tests/catalog-source.yaml | sed "s|catalog:latest|$(CATALOG_REPOSITORY):$(TAG)|" | sed "s|olm|openshift-marketplace|"| kubectl apply -f -
	sleep 20
else
	kubectl delete --ignore-not-found=true -n olm --ignore-not-found=true CatalogSource my-test-catalog
	sleep 10
	cat tests/catalog-source.yaml | sed "s|catalog:latest|$(CATALOG_REPOSITORY)-k8s:$(TAG)|" | kubectl apply -f -
	sleep 20
endif

deploy_operator:
ifeq ($(IS_OPENSHIFT), true)
	echo "Skipping deployment, TDB"
else
	kubectl create ns operator || true
	kubectl apply -f tests/operator-group.yaml
	kubectl apply -f tests/operator-subscription.yaml
	sleep 15
	kubectl get clusterserviceversion --all-namespaces
endif



OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH := $(shell uname -m | sed 's/x86_64/amd64/')

.PHONY: kustomize
KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
ifeq (,$(wildcard $(KUSTOMIZE)))
ifeq (,$(shell which kustomize 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(KUSTOMIZE)) ;\
	curl -sSLo - https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v3.5.4/kustomize_v3.5.4_$(OS)_$(ARCH).tar.gz | \
	tar xzf - -C bin/ ;\
	}
else
KUSTOMIZE = $(shell which kustomize)
endif
endif


.DEFAULT_GOAL := help
.PHONY: help
help: ## Show this help screen.
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
