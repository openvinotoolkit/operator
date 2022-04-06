#
# Copyright (c) 2022 Intel Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

SHELL = /bin/bash
# The operator image name to be used. By default it point to internal repo. 
# Can be swaped with the public operator for building the final bundle image
OPERATOR_IMAGE ?= registry.toolbox.iotg.sclab.intel.com/cpp/openvino-operator

BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
# TARGET_PLATFORM can be k8s or openshift
TARGET_PLATFORM =? k8s

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

ifneq ($(PLATFORM_KUBECONFIG), )
export KUBECONFIG=$(PLATFORM_KUBECONFIG)
endif

# TAG by default includes the git commit. It can be set manualy to any user friendly name like release name. 
# the catalog imamge includes also the tag in a format <branch>-latest
IMAGE_TAG ?= $(shell git rev-parse --short HEAD)

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
	go mod download
	go build $(GO_BUILD_ARGS) -o $(BUILD_DIR)/openvino-operator ./cmd

run: build # Run against the configured Kubernetes cluster in ~/.kube/config
	./$(BUILD_DIR)/openvino-operator run

.PHONY: test-unit
TEST_PKGS = $(shell go list ./...)
test-unit: ## Run unit tests
	go test -coverprofile=coverage.out -covermode=count -short $(TEST_PKGS)

docker-build: ## Build docker image with the manager.
	docker build -t ${OPERATOR_IMAGE}:${IMAGE_TAG} --build-arg https_proxy=$(https_proxy) --build-arg http_proxy=$(http_proxy) -f docker/Dockerfile .

docker-push: ## Push docker image with the manager.
	docker push ${OPERATOR_IMAGE}:${IMAGE_TAG} 


BUNDLE_REPOSITORY ?= registry.toolbox.iotg.sclab.intel.com/cpp/openvino-operator-bundle
CATALOG_REPOSITORY ?= registry.toolbox.iotg.sclab.intel.com/cpp/openvino-operator-catalog

bundle_build:
ifeq ($(TARGET_PLATFORM), openshift)
	echo "Building openshift bundle"
	sed -i "s|registry.connect.redhat.com/intel/ovms-operator:0.2.0|$(OPERATOR_IMAGE):$(IMAGE_TAG)|" bundle/manifests/openvino-operator.clusterserviceversion.yaml
	sed -i "s|gcr.io/kubebuilder/kube-rbac-proxy:v0.8.0|registry.redhat.io/openshift4/ose-kube-rbac-proxy:v4.8.0|" bundle/manifests/openvino-operator.clusterserviceversion.yaml
	docker build -t $(BUNDLE_REPOSITORY):$(IMAGE_TAG) -f bundle/Dockerfile bundle
	sed -i "s|$(OPERATOR_IMAGE):$(IMAGE_TAG)|registry.connect.redhat.com/intel/ovms-operator:0.2.0|" bundle/manifests/openvino-operator.clusterserviceversion.yaml
	sed -i "s|registry.redhat.io/openshift4/ose-kube-rbac-proxy:v4.8.0|gcr.io/kubebuilder/kube-rbac-proxy:v0.8.0|" bundle/manifests/openvino-operator.clusterserviceversion.yaml
else
	echo "Building kubernetes bundle"
	sed -i "s|registry.connect.redhat.com/intel/ovms-operator:0.2.0|$(OPERATOR_IMAGE):$(IMAGE_TAG)|" bundle/manifests/openvino-operator.clusterserviceversion.yaml
	docker build -t $(BUNDLE_REPOSITORY)-k8s:$(IMAGE_TAG) -f bundle/Dockerfile bundle
	sed -i "s|$(OPERATOR_IMAGE):$(IMAGE_TAG)|registry.connect.redhat.com/intel/ovms-operator:0.2.0|" bundle/manifests/openvino-operator.clusterserviceversion.yaml
endif

bundle_push:
ifeq ($(TARGET_PLATFORM), openshift)
	docker push $(BUNDLE_REPOSITORY):$(IMAGE_TAG)
else
	docker push $(BUNDLE_REPOSITORY)-k8s:$(IMAGE_TAG)
endif

catalog_build:
ifeq ($(TARGET_PLATFORM), openshift)
	docker -v | grep -q podman ; if [ $$? -eq 0 ]; then \
	opm index add --bundles $(BUNDLE_REPOSITORY):$(IMAGE_TAG) --from-index registry.redhat.io/redhat/certified-operator-index:v4.8 -c podman --tag $(CATALOG_REPOSITORY):$(IMAGE_TAG) ;\
	else sudo opm index add --bundles $(BUNDLE_REPOSITORY):$(IMAGE_TAG) --from-index registry.redhat.io/redhat/certified-operator-index:v4.8 -c docker --tag $(CATALOG_REPOSITORY):$(IMAGE_TAG) ;\
    fi
	docker tag $(CATALOG_REPOSITORY):$(IMAGE_TAG) $(CATALOG_REPOSITORY):$(BRANCH)-latest 
else
	sudo opm index add --bundles $(BUNDLE_REPOSITORY)-k8s:$(IMAGE_TAG)  -c docker --tag $(CATALOG_REPOSITORY)-k8s:$(IMAGE_TAG)
	docker tag $(CATALOG_REPOSITORY)-k8s:$(IMAGE_TAG) $(CATALOG_REPOSITORY)-k8s:$(BRANCH)-latest 	
endif

catalog_push:
ifeq ($(TARGET_PLATFORM), openshift)
	docker push $(CATALOG_REPOSITORY):$(IMAGE_TAG)
	docker push $(CATALOG_REPOSITORY):$(BRANCH)-latest 
else
	docker push $(CATALOG_REPOSITORY)-k8s:$(IMAGE_TAG)
	docker push $(CATALOG_REPOSITORY)-k8s:$(BRANCH)-latest 
endif


install: kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${OPERATOR_IMAGE}:${IMAGE_TAG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

all:  build_all_images deploy_catalog deploy_operator

build_all_images: docker-build docker-push bundle_build bundle_push catalog_build catalog_push

build_bundle_catalog_images: bundle_build bundle_push catalog_build catalog_push

cluster_clean:
ifeq ($(TARGET_PLATFORM), openshift)
	echo "Skipping cleanup"
else
	kubectl delete --ignore-not-found=true ns operator
	kubectl get ns olm ; if [ $$? -eq 0 ]; then operator-sdk olm uninstall ; fi
	operator-sdk olm install --timeout 10m0s
endif

deploy_catalog:
ifeq ($(TARGET_PLATFORM), openshift)
	kubectl delete --ignore-not-found=true -n openshift-marketplace --ignore-not-found=true CatalogSource my-test-catalog
	sleep 10
	cat tests/catalog-source.yaml | sed "s|catalog:latest|$(CATALOG_REPOSITORY):$(IMAGE_TAG)|" | sed "s|olm|openshift-marketplace|"| kubectl apply -f -
	sleep 20
else
	kubectl delete --ignore-not-found=true -n olm --ignore-not-found=true CatalogSource my-test-catalog
	sleep 10
	cat tests/catalog-source.yaml | sed "s|catalog:latest|$(CATALOG_REPOSITORY)-k8s:$(IMAGE_TAG)|" | kubectl apply -f -
	sleep 20
endif

deploy_operator:
ifeq ($(TARGET_PLATFORM), openshift)
	echo "Skipping deployment, TDB"
else
	kubectl create ns operator || true
	kubectl apply -f tests/operator-group.yaml
	kubectl apply -f tests/operator-subscription.yaml
	sleep 15
	kubectl get clusterserviceversion --all-namespaces
endif

platform-build-software: PLATFORM_BUILD_DIR PLATFORM_BUILD_MODE $(PLATFORM_BUILD_MODE)
	@echo ========== Target builds platform software  ===============
	@echo Args:
	@echo PLATFORM_BUILD_DIR: installation files directory [$(PLATFORM_BUILD_DIR)]
	@echo OPERATOR_IMAGE: [$(OPERATOR_IMAGE)]
	@echo IMAGE_TAG: [$(IMAGE_TAG)]
	@echo TARGET_PLATFORM: [$(TARGET_PLATFORM)]
	@echo PLATFORM_PACKAGE_DIR: packages files directory
	@echo PLATFORM_BUILD_MODE: build targets
	@echo PLATFORM_OPTS: yaml file with all platform opts for installation
	@echo  $(OPERATOR_IMAGE):$(IMAGE_TAG)
	@mkdir -p  $(PLATFORM_PACKAGE_DIR)/dependencies/images
	@docker save $(OPERATOR_IMAGE):$(IMAGE_TAG) | pigz > $(PLATFORM_PACKAGE_DIR)/dependencies/images/openvino_operator_$(PRODUCT_BUILD).tar.gz && test $${PIPESTATUS[0]} -eq 0
	@cp $(PLATFORM_PACKAGE_DIR)/dependencies/images/openvino_operator_$(PRODUCT_BUILD).tar.gz $(PLATFORM_PACKAGE_DIR)/package.tar.gz

platform-install-software: PLATFORM_KUBECONFIG PLATFORM_INSTALLER_DIR PLATFORM_INSTALLATION_MODE $(PLATFORM_INSTALLATION_MODE)
	@echo ========== Target installs platform software on the top of the kubernetes ===============
	@echo Args:
	@echo PLATFORM_KUBECONFIG: Kubernetes config with permissions to install platform
	@echo PLATFORM_INSTALLER_DIR: installation files directory
	@echo PLATFORM_INSTALLATION_MODE: installation mode
	@echo PLATFORM_OPTS: yaml file with all platform opts for installation
	@echo Returns: kubernetes installation with PLATFORM_KUBECONFIG configuration file
	@echo =========================================================================================

PLATFORM_%:
	@ if [ "${PLATFORM_${*}}" = "" ]; then \
	echo "Environment variable PLATFORM_$ is not set, please set one before run"; \
	exit 1; \
	fi

style:
	docker run --rm -v $$(pwd):/app -w /app -e https_proxy=$(https_proxy) golangci/golangci-lint:v1.44.0 golangci-lint run -E stylecheck --disable-all -v --timeout 3m0s

lint:
	docker run --rm -v $$(pwd):/app -w /app -e https_proxy=$(https_proxy) golangci/golangci-lint:v1.44.0 golangci-lint run --skip-dirs ../go/pkg/mod -v --timeout 3m0s

VIRTUALENV_EXE := python3 -m virtualenv -p python3
VIRTUALENV_DIR := .venv
ACTIVATE="$(VIRTUALENV_DIR)/bin/activate"

venv:$(ACTIVATE)
	@echo -n "Using venv "
	@. $(ACTIVATE); python3 --version

$(ACTIVATE):
	@echo "Updating virtualenv dependencies in: $(VIRTUALENV_DIR)..."
	@test -d $(VIRTUALENV_DIR) || $(VIRTUALENV_EXE) $(VIRTUALENV_DIR)
	@. $(ACTIVATE); pip install --upgrade pip
	@touch $(ACTIVATE)

sdl-check: venv
	@echo "Checking license headers in files..."
	@. $(ACTIVATE); bash -c "python3 lib_search.py . > missing_headers.txt"
	@if ! grep -FRq "All files have headers" missing_headers.txt; then\
        echo "Files with missing headers";\
        cat missing_headers.txt;\
		exit 1;\
	fi
	@rm missing_headers.txt

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