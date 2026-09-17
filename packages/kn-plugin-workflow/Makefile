# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#  http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.

BIN              := kn-workflow
BIN_DARWIN_AMD64 ?= $(BIN)-darwin-amd64
BIN_DARWIN_ARM64 ?= $(BIN)-darwin-arm64
BIN_LINUX        ?= $(BIN)-linux-amd64
BIN_WINDOWS      ?= $(BIN)-windows-amd64.exe

BIN_PATH         := ./dist
TEST_PATH        := ./dist-tests
MAIN_PATH        := cmd/main.go

METADATA_PATH                 := github.com/apache/incubator-kie-tools/packages/kn-plugin-workflow/pkg/metadata
SET_QUARKUS_PLATFORM_GROUP_ID := $(METADATA_PATH).QuarkusPlatformGroupId=$(QUARKUS_PLATFORM_GROUP_ID)
SET_QUARKUS_VERSION           := $(METADATA_PATH).QuarkusVersion=$(QUARKUS_VERSION)
SET_DEV_MODE_IMAGE            := $(METADATA_PATH).DevModeImage=$(DEV_MODE_IMAGE_URL)
SET_BUILDER_IMAGE             := $(METADATA_PATH).BuilderImage=$(BUILD_IMAGE_URL)
SET_VERSION                   := $(METADATA_PATH).PluginVersion=$(PLUGIN_VERSION)
SET_KOGITO_VERSION            := $(METADATA_PATH).KogitoVersion=$(KOGITO_VERSION)
LDFLAGS                       := "-X $(SET_QUARKUS_PLATFORM_GROUP_ID) -X $(SET_QUARKUS_VERSION) -X $(SET_VERSION) -X $(SET_DEV_MODE_IMAGE) -X $(SET_KOGITO_VERSION) -X $(SET_BUILDER_IMAGE)"

KIND_VERSION ?= v0.20.0
OLM_VERSION = v0.31.0
KIND_CLUSTER ?= kind
KUBE_RBAC_PROXY_SRC := quay.io/brancz/kube-rbac-proxy:v0.22.0
KUBE_RBAC_PROXY_DST := gcr.io/kubebuilder/kube-rbac-proxy:v0.13.0

# Optional: set the following variables to install the operator from a custom OLM CatalogSource
# (product/OSL builds) instead of the public operatorhubio-catalog.
# Only CATALOG_INDEX_IMAGE is required to activate the custom-catalog install path.
#
# When CATALOG_INDEX_IMAGE is set, kind-preload-images will also:
#   - Pull OPERATOR_IMAGE (proxy ref) and retag it as OPERATOR_IMAGE_RHT (registry.redhat.io ref
#     embedded in the CSV), then load it into kind so containerd resolves it locally by digest.
#   - Pull KUBE_RBAC_PROXY_PRODUCT (proxy ref) and retag it as KUBE_RBAC_PROXY_PRODUCT_RHT
#     (registry.redhat.io ref embedded in the CSV), then load it into kind.
#
# OPERATOR_IMAGE            - proxy-accessible source image  (registry-proxy.engineering.redhat.com/...)
# OPERATOR_IMAGE_RHT        - registry.redhat.io target ref  (must match what the CSV embeds)
# KUBE_RBAC_PROXY_PRODUCT   - proxy-accessible source image  (registry-proxy.engineering.redhat.com/...)
# KUBE_RBAC_PROXY_PRODUCT_RHT - registry.redhat.io target ref (must match what the CSV embeds)
CATALOG_INDEX_IMAGE ?=
OPERATOR_BUNDLE_IMAGE ?=
OPERATOR_IMAGE ?=
OPERATOR_IMAGE_RHT ?=
KUBE_RBAC_PROXY_PRODUCT ?=
KUBE_RBAC_PROXY_PRODUCT_RHT ?=

ARCH := $(shell uname -m)
ifeq ($(ARCH),arm64)
	GOARCH = arm64
	BIN_BUILD_DARWIN=$(BIN_DARWIN_ARM64)
else
	GOARCH = amd64
	BIN_BUILD_DARWIN=$(BIN_DARWIN_AMD64)
endif

build-all: build-linux-amd64 build-darwin-amd64 build-darwin-arm64 build-win32-amd64

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=$(GOARCH) go build -ldflags $(LDFLAGS) -o $(BIN_PATH)/$(BIN_BUILD_DARWIN) $(MAIN_PATH)

build-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags $(LDFLAGS) -o $(BIN_PATH)/$(BIN_DARWIN_AMD64) $(MAIN_PATH)

build-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags $(LDFLAGS) -o $(BIN_PATH)/$(BIN_DARWIN_ARM64) $(MAIN_PATH)

build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags $(LDFLAGS) -o $(BIN_PATH)/$(BIN_LINUX) $(MAIN_PATH)

build-win32-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags $(LDFLAGS) -o $(BIN_PATH)/$(BIN_WINDOWS) $(MAIN_PATH)

clean:
	go clean
	rm -rf $(BIN_PATH) $(TEST_PATH)

.PHONY: test-e2e
test-e2e:
	@$(MAKE) install-kind
	@$(MAKE) create-cluster
	@$(MAKE) install-operator-framework
	@$(MAKE) kind-preload-images
	@$(MAKE) go-test-e2e
	@$(MAKE) go-test-e2e-report


.PHONY: install-kind
install-kind:
	command -v kind >/dev/null || go install sigs.k8s.io/kind@$(KIND_VERSION)

KIND_CLUSTER_CONFIG     := e2e-tests/kind-cluster-config.yaml
CONTAINERD_CERTS_SRC    := e2e-tests/containerd-certs.d
CONTAINERD_CERTS_DST    := /etc/containerd/certs.d

.PHONY: create-cluster
create-cluster: install-kind
	@if [ -n "$(CATALOG_INDEX_IMAGE)" ]; then \
		echo "Creating kind cluster with containerd v2 registry config for internal registry..."; \
		kind create cluster --config $(KIND_CLUSTER_CONFIG); \
		echo "Copying registry TLS/mirror config into kind node..."; \
		docker cp $(CONTAINERD_CERTS_SRC)/. $(KIND_CLUSTER)-control-plane:$(CONTAINERD_CERTS_DST)/; \
		echo "Copying docker credentials into kind node for authenticated registry pulls..."; \
		docker cp $(HOME)/.docker/config.json $(KIND_CLUSTER)-control-plane:/var/lib/kubelet/config.json; \
		docker exec $(KIND_CLUSTER)-control-plane systemctl restart containerd; \
		echo "Waiting for containerd to restart..."; \
		sleep 5; \
	else \
		kind create cluster; \
	fi

.PHONY: install-operator-framework
install-operator-framework:
	curl -sL https://github.com/operator-framework/operator-lifecycle-manager/releases/download/$(OLM_VERSION)/install.sh | bash -s $(OLM_VERSION)

.PHONY: go-test-e2e
go-test-e2e:
	rm -rf dist-tests-e2e
	mkdir dist-tests-e2e
	CATALOG_INDEX_IMAGE=$(CATALOG_INDEX_IMAGE) \
	OPERATOR_BUNDLE_IMAGE=$(OPERATOR_BUNDLE_IMAGE) \
	OPERATOR_IMAGE=$(OPERATOR_IMAGE) \
	go test -v ./e2e-tests/... -tags e2e_tests -run TestQuarkusRunCommand -timeout 20m 2>&1 | tee ./dist-tests-e2e/go-test-output-e2e.txt

.PHONY: go-test-e2e-report
go-test-e2e-report:
	go run github.com/jstemmer/go-junit-report/v2 \
	  -set-exit-code \
	  -in ./dist-tests-e2e/go-test-output-e2e.txt \
	  -out ./dist-tests-e2e/junit-report-it.xml

.PHONY: kind-preload-images
kind-preload-images:
	@echo "Preloading kube-rbac-proxy image into kind..."
	docker pull $(KUBE_RBAC_PROXY_SRC)
	docker tag $(KUBE_RBAC_PROXY_SRC) $(KUBE_RBAC_PROXY_DST)
	kind load docker-image --name $(KIND_CLUSTER) $(KUBE_RBAC_PROXY_DST)
	@if [ -n "$(CATALOG_INDEX_IMAGE)" ]; then \
		echo "Preloading index image $(CATALOG_INDEX_IMAGE) into kind..."; \
		docker pull $(CATALOG_INDEX_IMAGE); \
		kind load docker-image --name $(KIND_CLUSTER) $(CATALOG_INDEX_IMAGE); \
	fi
	@if [ -n "$(OPERATOR_IMAGE)" ] && [ -n "$(OPERATOR_IMAGE_RHT)" ]; then \
		echo "Preloading operator image $(OPERATOR_IMAGE) -> $(OPERATOR_IMAGE_RHT) into kind..."; \
		docker pull $(OPERATOR_IMAGE); \
		docker save $(OPERATOR_IMAGE) -o /tmp/operator-image.tar; \
		OPERATOR_RHT_BASE=$$(echo "$(OPERATOR_IMAGE_RHT)" | cut -d'@' -f1); \
		OPERATOR_RHT_DIGEST=$$(echo "$(OPERATOR_IMAGE_RHT)" | cut -d'@' -f2); \
		docker exec -i $(KIND_CLUSTER)-control-plane ctr --namespace k8s.io images import --digests --base-name $$OPERATOR_RHT_BASE - < /tmp/operator-image.tar; \
		MANIFEST_DIGEST=$$(docker exec $(KIND_CLUSTER)-control-plane ctr --namespace k8s.io images ls 2>/dev/null | grep "$$OPERATOR_RHT_BASE@" | grep "manifest.v1" | awk '{print $$1}' | head -1 | cut -d'@' -f2); \
		if [ -n "$$MANIFEST_DIGEST" ] && [ "$$MANIFEST_DIGEST" != "$$OPERATOR_RHT_DIGEST" ]; then \
			echo "Adding digest alias $$OPERATOR_RHT_DIGEST -> $$MANIFEST_DIGEST in containerd..."; \
			docker exec $(KIND_CLUSTER)-control-plane ctr --namespace k8s.io images tag \
				"$$OPERATOR_RHT_BASE@$$MANIFEST_DIGEST" \
				"$$OPERATOR_RHT_BASE@$$OPERATOR_RHT_DIGEST"; \
		fi; \
		rm -f /tmp/operator-image.tar; \
	fi
	@if [ -n "$(KUBE_RBAC_PROXY_PRODUCT)" ] && [ -n "$(KUBE_RBAC_PROXY_PRODUCT_RHT)" ]; then \
		echo "Preloading product kube-rbac-proxy $(KUBE_RBAC_PROXY_PRODUCT) -> $(KUBE_RBAC_PROXY_PRODUCT_RHT) into kind..."; \
		docker pull $(KUBE_RBAC_PROXY_PRODUCT); \
		kind load docker-image --name $(KIND_CLUSTER) $(KUBE_RBAC_PROXY_PRODUCT); \
		KUBE_RBAC_RHT_NAME=$$(echo "$(KUBE_RBAC_PROXY_PRODUCT_RHT)" | cut -d'@' -f1); \
		docker tag $(KUBE_RBAC_PROXY_PRODUCT) $$KUBE_RBAC_RHT_NAME; \
		kind load docker-image --name $(KIND_CLUSTER) $$KUBE_RBAC_RHT_NAME; \
	fi