# ==============================================================================
# Define global Makefile variables for easy reference later

COMMON_SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
# Project root directory
PROJ_ROOT_DIR := $(strip $(abspath $(shell cd $(COMMON_SELF_DIR)/ && pwd -P)))
# Directory for build artifacts and temporary files
OUTPUT_DIR := $(PROJ_ROOT_DIR)/_output
INTEGRATION_PROJECT ?= gin-enterprise-template-integration
INTEGRATION_DATA_DIR := $(OUTPUT_DIR)/integration
INTEGRATION_POSTGRES_PORT ?= 55432
INTEGRATION_KAFKA_PORT ?= 29092
INTEGRATION_REDIS_PORT ?= 56380
# Path to Protobuf files
APIROOT=$(PROJ_ROOT_DIR)/pkg/api
ROOT_PACKAGE=github.com/clin211/gin-enterprise-template
DOCKERFILE_DIR=$(PROJ_ROOT_DIR)/build/docker
REGISTRY_PREFIX ?= gin-enterprise-template

# Set the unit test coverage threshold. The value of 1 here is just for demonstration, as the project currently lacks comprehensive unit tests.
# For production environments, it is recommended to set a higher threshold, e.g., 60.
COVERAGE ?= 20
PROTOC_GEN_GO_VERSION ?= v1.36.11
PROTOC_GEN_GO_GRPC_VERSION ?= v1.6.2
GRPC_GATEWAY_VERSION ?= v2.29.0
PROTOC_GEN_DEFAULTS_VERSION ?= v0.0.2
PROTOC_INJECT_TAG_VERSION ?= v1.4.0
WIRE_VERSION ?= v0.6.0
GOLANGCI_LINT_VERSION ?= v2.12.2

# ==============================================================================
# Define version-related variables

## Specify the version package used by the application. Values will be injected into variables in this package using `-ldflags -X`.
VERSION_PACKAGE=github.com/clin211/gin-enterprise-template/pkg/version

## Check if current directory is a Git repository
IS_GIT_REPO := $(shell git rev-parse --is-inside-work-tree 2>/dev/null)
ifeq ($(IS_GIT_REPO),)
    # Non-Git repository, use fallback values
    GIT_TREE_STATE := "not_a_git_repo"
    VERSION := v0.0.0
    GIT_COMMIT := "unknown"
else
    # Define the VERSION semantic version (if not already set)
    ifeq ($(origin VERSION), undefined)
        VERSION := $(shell git describe --tags --abbrev=0 --match='v*' 2>/dev/null || echo v0.0.0)
    endif

    # Check if the code repository is in a dirty state (default is dirty)
    GIT_TREE_STATE := "dirty"
    ifeq (, $(shell git status --porcelain 2>/dev/null))
        GIT_TREE_STATE := "clean"
    endif

    # Get current Git commit hash
    GIT_COMMIT := $(shell git rev-parse HEAD)
endif

GO_LDFLAGS += \
    -X $(VERSION_PACKAGE).gitVersion=$(VERSION) \
    -X $(VERSION_PACKAGE).gitCommit=$(GIT_COMMIT) \
    -X $(VERSION_PACKAGE).gitTreeState=$(GIT_TREE_STATE) \
    -X $(VERSION_PACKAGE).buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ') \
	-X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn

ifeq ($(GOOS),windows)
    GO_OUT_EXT := .exe
endif

GO_BUILD_FLAGS += -ldflags "$(GO_LDFLAGS)"
# 只构建 gin-enterprise-template-apiserver，排除 gen-gorm-model 等工具
COMMANDS ?= $(filter-out $(PROJ_ROOT_DIR)/cmd/gen-gorm-model, $(filter-out %.md, $(wildcard $(PROJ_ROOT_DIR)/cmd/*)))
BINS ?= $(foreach cmd,${COMMANDS},$(notdir $(cmd)))
IMAGES ?= $(filter-out tools, $(foreach dir, $(COMMANDS), $(notdir $(if $(wildcard $(dir)/*.go), $(dir),))))

# 编译的操作系统可以是 linux/windows/darwin
PLATFORMS ?= darwin_amd64 darwin_arm64 windows_amd64 windows_arm64 linux_amd64 linux_arm64

# 设置一个指定的操作系统
ifeq ($(origin PLATFORM), undefined)
    ifeq ($(origin GOOS), undefined)
        GOOS := $(shell go env GOOS)
    endif
    ifeq ($(origin GOARCH), undefined)
        GOARCH := $(shell go env GOARCH)
    endif
    PLATFORM := $(GOOS)_$(GOARCH)
    # 构建镜像时，使用 linux 作为默认的 OS
    IMAGE_PLAT := linux_$(GOARCH)
else
    GOOS := $(word 1, $(subst _, ,$(PLATFORM)))
    GOARCH := $(word 2, $(subst _, ,$(PLATFORM)))
    IMAGE_PLAT := $(PLATFORM)
endif

# ==============================================================================
# Define the default target as `all`
.DEFAULT_GOAL := all

# Define the Makefile `all` phony target. When `make` is executed, it will default to executing the `all` target.
.PHONY: all
all: deps protoc tidy format generate build cover

# ==============================================================================
# Usage

define USAGE_OPTIONS

Options:
  BINS             The binary files to build. Defaults to gin-enterprise-template-apiserver only.
                   This option can be used with the following command: make build
                   Example: make build BINS=<BinaryName>
  IMAGES           Backend images to make. Defaults to all files in the `cmd` directory.
                   This option is available when using command: make image
                   Example: make image IMAGES=<BinaryName>
  VERSION          Version information to embed into the binary.
  V                Set to 1 to enable verbose build output. Default is 0.
endef
export USAGE_OPTIONS

# ==============================================================================
# Define other required phony targets
#

.PHONY: build.multiarch
build.multiarch: $(foreach p,$(PLATFORMS),$(addprefix build., $(addprefix $(p)., $(BINS)))) ## Build all applications with all supported arch.

.PHONY: build
build: $(addprefix build., $(addprefix $(PLATFORM)., $(BINS))) ## Build all binaries for the selected platform.

build.%: ## 编译 Go 源码.
	$(eval COMMAND := $(word 2,$(subst ., ,$*)))
	$(eval PLATFORM := $(word 1,$(subst ., ,$*)))
	$(eval OS := $(word 1,$(subst _, ,$(PLATFORM))))
	$(eval ARCH := $(word 2,$(subst _, ,$(PLATFORM))))
	@echo "===========> Building binary $(COMMAND) $(VERSION) for $(OS) $(ARCH)"
	@mkdir -p $(OUTPUT_DIR)/platforms/$(OS)/$(ARCH)
	@CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build $(GO_BUILD_FLAGS) \
		-o $(OUTPUT_DIR)/platforms/$(OS)/$(ARCH)/$(COMMAND)$(GO_OUT_EXT) \
		$(ROOT_PACKAGE)/cmd/$(COMMAND)
.PHONY: image
image: $(addprefix image., $(addprefix $(PLATFORM)., $(IMAGES))) ## Build all docker images for the selected platform.

image.%: build.% ## 构建指定的 Docker 镜像
	$(eval IMAGE := $(word 2,$(subst ., ,$*)))
	$(eval IMAGE_TAG := $(subst +,-,$(VERSION)))
	@echo "===========> Building docker image $(IMAGE) $(IMAGE_TAG) for $(IMAGE_PLAT)"
	$(eval PLATFORM := $(word 1,$(subst ., ,$*)))
	$(eval ARCH := $(word 2,$(subst _, ,$(PLATFORM))))
	$(eval OS := $(word 1,$(subst _, ,$(PLATFORM))))
	$(eval DOCKERFILE := Dockerfile)
	@DOCKER_BUILDKIT=1 docker build \
		--build-arg OS=$(OS) \
		--build-arg ARCH=$(ARCH) \
		--file $(DOCKERFILE_DIR)/$(IMAGE)/$(DOCKERFILE) \
		--tag $(REGISTRY_PREFIX)/$(IMAGE):$(IMAGE_TAG) \
		$(PROJ_ROOT_DIR)

.PHONY: format
format: format.proto ## Format source code with gofmt (-s -w) and protobuf with buf.
	@gofmt -s -w ./

.PHONY: verify
verify: verify.format verify.generated verify.vet verify.test verify.lint verify.build ## Run the non-mutating production quality gate.

.PHONY: verify.format
verify.format: ## Check Go and Proto formatting without modifying the worktree.
	@test -z "$$(gofmt -s -l $$(find . -name '*.go' -not -path './third_party/*' -not -path './_output/*'))"
	@buf format --diff $(PROJ_ROOT_DIR) --path pkg/api/apiserver/v1

.PHONY: verify.vet
verify.vet: ## Run the standard Go static analyzer.
	@go vet ./...

.PHONY: verify.test
verify.test: ## Run unit tests with race detection and coverage.
	@mkdir -p $(OUTPUT_DIR)
	@go test -race -cover -coverprofile=$(OUTPUT_DIR)/coverage.out -timeout=10m -shuffle=on -short `go list ./...|egrep -v 'tools|vendor|third_party'`
	@go tool cover -func=$(OUTPUT_DIR)/coverage.out | awk -v target=$(COVERAGE) -f $(PROJ_ROOT_DIR)/scripts/coverage.awk

.PHONY: verify.generated
verify.generated: ## Regenerate Proto/OpenAPI/Wire and fail on drift; requires a clean generated-file worktree.
	@test -z "$$(git status --porcelain -- pkg/api/apiserver/v1 api/openapi internal/apiserver/wire_gen.go)" || \
		(echo "generated files have local changes; commit or stash them before verification"; exit 1)
	@$(MAKE) protoc
	@go generate ./internal/apiserver
	@git diff --exit-code -- pkg/api/apiserver/v1 api/openapi internal/apiserver/wire_gen.go

.PHONY: verify.lint
verify.lint: ## Run static analysis for new changes without changing source files.
	@if [ -n "$(LINT_NEW_FROM_REV)" ]; then \
		golangci-lint run -c $(PROJ_ROOT_DIR)/.golangci.yaml --new-from-rev="$(LINT_NEW_FROM_REV)" $(PROJ_ROOT_DIR)/...; \
	else \
		golangci-lint run -c $(PROJ_ROOT_DIR)/.golangci.yaml --new $(PROJ_ROOT_DIR)/...; \
	fi

.PHONY: verify.build
verify.build: ## Build the API server binary.
	@$(MAKE) build BINS=gin-enterprise-template-apiserver

.PHONY: integration.up
integration.up: ## Start isolated PostgreSQL and Kafka integration dependencies.
	@mkdir -p $(INTEGRATION_DATA_DIR)/pgdata
	@COMPOSE_PROJECT_NAME=$(INTEGRATION_PROJECT) \
	 POSTGRES_DB=enterprise_template_test \
	 POSTGRES_PASSWORD=integration-postgres \
	 POSTGRES_PORT=$(INTEGRATION_POSTGRES_PORT) \
	 POSTGRES_DATA_PATH=$(INTEGRATION_DATA_DIR)/pgdata \
	 KAFKA_PORT=$(INTEGRATION_KAFKA_PORT) \
	 REDIS_PORT=$(INTEGRATION_REDIS_PORT) \
	 REDIS_PASSWORD=integration-redis \
	 docker compose -f docker-compose.env.yml up -d --wait postgres redis redpanda
	@COMPOSE_PROJECT_NAME=$(INTEGRATION_PROJECT) \
	 POSTGRES_DB=enterprise_template_test \
	 POSTGRES_PASSWORD=integration-postgres \
	 POSTGRES_PORT=$(INTEGRATION_POSTGRES_PORT) \
	 POSTGRES_DATA_PATH=$(INTEGRATION_DATA_DIR)/pgdata \
	 KAFKA_PORT=$(INTEGRATION_KAFKA_PORT) \
	 REDIS_PORT=$(INTEGRATION_REDIS_PORT) \
	 REDIS_PASSWORD=integration-redis \
	 docker compose -f docker-compose.env.yml run --rm topic-init

.PHONY: integration.test
integration.test: ## Run PostgreSQL and Kafka integration tests.
	@TEST_POSTGRES_DSN="$${TEST_POSTGRES_DSN:-postgres://postgres:integration-postgres@127.0.0.1:$(INTEGRATION_POSTGRES_PORT)/enterprise_template_test?sslmode=disable}" \
	 TEST_KAFKA_BROKERS="$${TEST_KAFKA_BROKERS:-127.0.0.1:$(INTEGRATION_KAFKA_PORT)}" \
	 TEST_KAFKA_TOPIC="$${TEST_KAFKA_TOPIC:-events}" \
	 TEST_KAFKA_DLQ_TOPIC="$${TEST_KAFKA_DLQ_TOPIC:-events.dlq}" \
	 TEST_REDIS_ADDR="$${TEST_REDIS_ADDR:-127.0.0.1:$(INTEGRATION_REDIS_PORT)}" \
	 TEST_REDIS_PASSWORD="$${TEST_REDIS_PASSWORD:-integration-redis}" \
	 go test -race -count=1 -v ./internal/apiserver/store ./internal/messaging/consumer ./pkg/token -run Integration

.PHONY: integration.smoke
integration.smoke: build ## Start the API against integration dependencies and verify liveness/readiness.
	@INTEGRATION_POSTGRES_PORT=$(INTEGRATION_POSTGRES_PORT) \
	 INTEGRATION_KAFKA_PORT=$(INTEGRATION_KAFKA_PORT) \
	 INTEGRATION_REDIS_PORT=$(INTEGRATION_REDIS_PORT) \
	 ./scripts/smoke-test.sh $(OUTPUT_DIR)/platforms/$(OS)/$(ARCH)/gin-enterprise-template-apiserver

.PHONY: integration.down
integration.down: ## Stop and remove isolated integration dependencies.
	@COMPOSE_PROJECT_NAME=$(INTEGRATION_PROJECT) \
	 POSTGRES_DB=enterprise_template_test \
	 POSTGRES_PASSWORD=integration-postgres \
	 POSTGRES_PORT=$(INTEGRATION_POSTGRES_PORT) \
	 POSTGRES_DATA_PATH=$(INTEGRATION_DATA_DIR)/pgdata \
	 KAFKA_PORT=$(INTEGRATION_KAFKA_PORT) \
	 REDIS_PORT=$(INTEGRATION_REDIS_PORT) \
	 REDIS_PASSWORD=integration-redis \
	 docker compose -f docker-compose.env.yml down -v --remove-orphans
	@$(RM) -r $(INTEGRATION_DATA_DIR)

.PHONY: format.proto
format.proto: ## Format protobuf files in pkg/api/apiserver/v1 with buf.
	@buf format -w $(PROJ_ROOT_DIR) --path pkg/api/apiserver/v1

.PHONY: add-copyright
add-copyright: ## Add copyright headers (skip third_party, vendor, _output).
	@addlicense -v -f $(PROJ_ROOT_DIR)/scripts/boilerplate.txt $(PROJ_ROOT_DIR) --skip-dirs=third_party,vendor,$(OUTPUT_DIR)

.PHONY: tidy
tidy: ## Sync dependencies and update go.mod/go.sum (go mod tidy).
	@go mod tidy

.PHONY: clean
clean: ## Remove build artifacts and temp files (_output/).
	@-rm -vrf $(OUTPUT_DIR)

.PHONY: protoc
protoc: ## Compile Protobuf files.
	@echo "===========> Generate protobuf files"
	@mkdir -p $(PROJ_ROOT_DIR)/api/openapi
	@protoc                                              \
		--proto_path=$(APIROOT)                          \
		--proto_path=$(PROJ_ROOT_DIR)/third_party/protobuf    \
		--go_out=paths=source_relative:$(APIROOT)        \
		--go-grpc_out=paths=source_relative:$(APIROOT)   \
		--grpc-gateway_out=allow_delete_body=true,paths=source_relative:$(APIROOT) \
		--openapiv2_out=$(PROJ_ROOT_DIR)/api/openapi \
		--openapiv2_opt=allow_delete_body=true,logtostderr=true \
		--defaults_out=paths=source_relative:$(APIROOT) \
		$(shell find $(APIROOT)/* -name *.proto)
	@find $(APIROOT)/* -name "*.pb.go" -exec protoc-go-inject-tag -input={} \;

.PHONY: protoc.%
protoc.%: ## Compile Protobuf files.
	@echo "===========> Generate protobuf files"
	@mkdir -p $(PROJ_ROOT_DIR)/api/openapi
	@protoc                                              \
		--proto_path=$(APIROOT)                          \
		--proto_path=$(PROJ_ROOT_DIR)/third_party/protobuf    \
		--go_out=paths=source_relative:$(APIROOT)        \
		--go-grpc_out=paths=source_relative:$(APIROOT)   \
		--openapiv2_out=$(PROJ_ROOT_DIR)/api/openapi \
		--openapiv2_opt=allow_delete_body=true,logtostderr=true \
		--defaults_out=paths=source_relative:$(APIROOT) \
		$(shell find $(APIROOT)/$* -name *.proto)
	@find $(APIROOT)/$* -name "*.pb.go" -exec protoc-go-inject-tag -input={} \;

.PHONY: models
models: ## Generate models for all packages using gorm.io/gen.
	@echo "===========> Generating models"
	@cd cmd/gen-gorm-model && go run gen_gorm_model.go

.PHONY: test
test: ## Run unit tests (-race -cover -shuffle -short).
	@echo "===========> Running unit tests"
	@mkdir -p $(OUTPUT_DIR)
	@go test -race -cover \
		-coverprofile=$(OUTPUT_DIR)/coverage.out \
		-timeout=10m -shuffle=on -short \
		-v `go list ./...|egrep -v 'tools|vendor|third_party'`

.PHONY: cover
cover: test ## Run tests and enforce coverage threshold (COVERAGE%).
	@echo "===========> Running code coverage tests"
	@go tool cover -func=$(OUTPUT_DIR)/coverage.out | awk -v target=$(COVERAGE) -f $(PROJ_ROOT_DIR)/scripts/coverage.awk

.PHONY: lint
lint: ## Static analysis with golangci-lint (uses .golangci.yaml).
	@echo "===========> Running golangci to lint source codes"
	@golangci-lint run -c $(PROJ_ROOT_DIR)/.golangci.yaml $(PROJ_ROOT_DIR)/...

.PHONY: deps
deps: ## Install build and codegen tools.
	@echo "===========> Installing dependent tools, may take a while ..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@$(GRPC_GATEWAY_VERSION)
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@$(GRPC_GATEWAY_VERSION)
	@go install github.com/onexstack/protoc-gen-defaults@$(PROTOC_GEN_DEFAULTS_VERSION)
	@go install github.com/favadi/protoc-go-inject-tag@$(PROTOC_INJECT_TAG_VERSION)
	@go install github.com/google/wire/cmd/wire@$(WIRE_VERSION)
	@go install github.com/onexstack/addlicense@v0.0.3
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found, installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | \
			sh -s -- -b "$(go env GOPATH)/bin" $(GOLANGCI_LINT_VERSION) ; \
	fi

.PHONY: generate
generate: ## Generate code for all packages using go:generate directives (recursively ./...)
	@go generate ./...

help: Makefile  ## Show available targets and usage.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<TARGETS> <OPTIONS>\033[0m\n\n\033[35mTargets:\033[0m\n"} /^[0-9A-Za-z._-]+:.*?##/ { printf "  \033[36m%-45s\033[0m %s\n", $$1, $$2 } /^\$$\([0-9A-Za-z_-]+\):.*?##/ { gsub("_","-", $$1); printf "  \033[36m%-45s\033[0m %s\n", tolower(substr($$1, 3, length($$1)-7)), $$2 } /^##@/{ printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' Makefile #$(MAKEFILE_LIST)
	@echo "$$USAGE_OPTIONS"
