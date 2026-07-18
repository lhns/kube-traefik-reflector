# kube-traefik-reflector

IMG ?= ghcr.io/lhns/kube-traefik-reflector:dev
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)

CONTROLLER_GEN_VERSION ?= v0.16.5
CONTROLLER_GEN ?= go run sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)

.PHONY: help
help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

##@ Develop

.PHONY: fmt
fmt: ## Run gofmt.
	gofmt -w .

.PHONY: vet
vet: ## Run go vet.
	go vet ./...

.PHONY: test
test: ## Run unit tests.
	go test ./... -count=1

.PHONY: build
build: ## Build the manager binary.
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/manager ./cmd

.PHONY: run
run: ## Run the manager against the current kubecontext.
	go run ./cmd

##@ Generate

.PHONY: manifests
manifests: ## Regenerate the RBAC ClusterRole from +kubebuilder:rbac markers.
	$(CONTROLLER_GEN) rbac:roleName=traefik-reflector paths="./internal/controller/..." output:rbac:dir=config/rbac

##@ Docker

.PHONY: docker-build
docker-build: ## Build the container image.
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_DATE=$(BUILD_DATE) -t $(IMG) .

.PHONY: docker-push
docker-push: ## Push the container image.
	docker push $(IMG)
