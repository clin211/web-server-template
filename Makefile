TEST_FLAGS ?= $(if $(TEST_POSTGRES_DSN),,-short)
# ==============================================================================
# Define global Makefile variables for easy reference later

COMMON_SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
# Project root directory
PROJ_ROOT_DIR := $(strip $(abspath $(shell cd $(COMMON_SELF_DIR)/ && pwd -P)))
# Directory for build artifacts and temporary files
OUTPUT_DIR := $(PROJ_ROOT_DIR)/_output
CONFIG ?= configs/configs.yaml
RUN_SERVICES := apiserver worker scheduler
RUN_BIN_DIR = $(OUTPUT_DIR)/platforms/$(GOOS)/$(GOARCH)
INTEGRATION_PROJECT ?= gin-enterprise-template-integration
INTEGRATION_POSTGRES_PORT ?= 55432
INTEGRATION_KAFKA_PORT ?= 29092
INTEGRATION_REDIS_PORT ?= 56380
# Path to Protobuf files
APIROOT=$(PROJ_ROOT_DIR)/pkg/api
ROOT_PACKAGE=github.com/clin211/gin-enterprise-template
DOCKERFILE_DIR=$(PROJ_ROOT_DIR)/build/docker
REGISTRY_PREFIX ?= gin-enterprise-template

# Keep the existing 20% coverage threshold; CI includes real dependency fault tests.
# For production environments, it is recommended to set a higher threshold, e.g., 60.
COVERAGE ?= 20
PROTOC_GEN_GO_VERSION ?= v1.36.11
PROTOC_GEN_GO_GRPC_VERSION ?= v1.6.2
GRPC_GATEWAY_VERSION ?= v2.29.0
PROTOC_GEN_DEFAULTS_VERSION ?= v0.0.2
PROTOC_INJECT_TAG_VERSION ?= v1.4.0
WIRE_VERSION ?= v0.7.0
GOLANGCI_LINT_VERSION ?= v2.12.2
GOOSE_VERSION := v3.28.0
GOOSE := $(OUTPUT_DIR)/tools/goose
GOOSE_BUILD_TAGS := no_clickhouse,no_libsql,no_mssql,no_mysql,no_sqlite3,no_vertica,no_ydb

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
# 默认构建各进程入口，排除 gen-gorm-model 等工具
COMMANDS ?= $(foreach dir,$(wildcard $(PROJ_ROOT_DIR)/cmd/*),$(if $(wildcard $(dir)/main.go),$(dir)))
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
  BINS             The binary files to build. Defaults to all application entrypoints.
                   This option can be used with the following command: make build
                   Example: make build BINS=<BinaryName>
  IMAGES           Backend images to make. Defaults to all files in the `cmd` directory.
                   This option is available when using command: make image
                   Example: make image IMAGES=<BinaryName>
  VERSION          Version information to embed into the binary.
  CONFIG           Config file for make run or make run.<service>. Defaults to configs/configs.yaml.
  GOOSE_DBSTRING   Target PostgreSQL connection string for make migrate or make migrate.status.
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
.PHONY: run run.apiserver run.worker run.scheduler
run: $(foreach service,$(RUN_SERVICES),build.$(PLATFORM).gin-enterprise-template-$(service)) ## Build and run all three services; Ctrl+C stops them together.
	@set -e; cd "$(PROJ_ROOT_DIR)"; \
	for service in $(RUN_SERVICES); do \
		binary="$(RUN_BIN_DIR)/gin-enterprise-template-$$service$(GO_OUT_EXT)"; \
		if [ ! -x "$$binary" ]; then \
			echo "Binary missing or not executable: $$binary. Run make build first." >&2; \
			exit 1; \
		fi; \
	done; \
	pids=""; \
	trap 'for pid in $$pids; do kill "$$pid" 2>/dev/null || :; done; wait' 0; \
	trap 'exit 130' INT; \
	trap 'exit 143' TERM; \
	for service in $(RUN_SERVICES); do \
		"$(RUN_BIN_DIR)/gin-enterprise-template-$$service$(GO_OUT_EXT)" --config "$(CONFIG)" & \
		pids="$$pids $$!"; \
	done; \
	for pid in $$pids; do wait "$$pid"; done

run.apiserver: ## Build and run the current API server.
run.worker: ## Build and run the current Worker.
run.scheduler: ## Build and run the current Scheduler.
run.apiserver run.worker run.scheduler: run.%: build.$(PLATFORM).gin-enterprise-template-%
	@cd "$(PROJ_ROOT_DIR)" && exec "$(RUN_BIN_DIR)/gin-enterprise-template-$*$(GO_OUT_EXT)" --config "$(CONFIG)"

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
verify.test: $(if $(TEST_POSTGRES_DSN),tools.goose) ## Run race and coverage; TEST_POSTGRES_DSN includes integration paths.
	@mkdir -p $(OUTPUT_DIR)
	@GOOSE_BIN="$(GOOSE)" go test -race -cover -coverpkg="$$(go list ./... | egrep -v 'tools|vendor|third_party' | paste -sd, -)" -coverprofile=$(OUTPUT_DIR)/coverage.out -timeout=10m -shuffle=on $(TEST_FLAGS) `go list ./...|egrep -v 'tools|vendor|third_party'`
	@go tool cover -func=$(OUTPUT_DIR)/coverage.out | awk -v target=$(COVERAGE) -f $(PROJ_ROOT_DIR)/scripts/coverage.awk

.PHONY: verify.generated
verify.generated: ## Regenerate in an isolated copy and compare with current generated files.
	@./scripts/check-generated.sh

.PHONY: verify.lint
verify.lint: ## Run static analysis for new changes without changing source files.
	@if [ -n "$(LINT_NEW_FROM_REV)" ]; then \
		golangci-lint run -c $(PROJ_ROOT_DIR)/.golangci.yaml --new-from-rev="$(LINT_NEW_FROM_REV)" $(PROJ_ROOT_DIR)/...; \
	else \
		golangci-lint run -c $(PROJ_ROOT_DIR)/.golangci.yaml --new $(PROJ_ROOT_DIR)/...; \
	fi

.PHONY: verify.build
verify.build: ## Build API, Worker and Scheduler binaries.
	@$(MAKE) build BINS="gin-enterprise-template-apiserver gin-enterprise-template-worker gin-enterprise-template-scheduler"
	@python3 scripts/check-runtime-migrations.py --binary-dir $(OUTPUT_DIR)/platforms/$(GOOS)/$(GOARCH) --sql-dir migrations

.PHONY: integration.up
integration.up: ## Start isolated PostgreSQL and Kafka integration dependencies.
	@COMPOSE_PROJECT_NAME=$(INTEGRATION_PROJECT) \
	 POSTGRES_DB=enterprise_template_test \
	 POSTGRES_PASSWORD=integration-postgres \
	 POSTGRES_PORT=$(INTEGRATION_POSTGRES_PORT) \
	 POSTGRES_DATA_PATH=pgdata \
	 KAFKA_PORT=$(INTEGRATION_KAFKA_PORT) \
	 REDIS_PORT=$(INTEGRATION_REDIS_PORT) \
	 REDIS_PASSWORD=integration-redis \
	 REDIS_DATA_PATH=redisdata \
	 docker compose -f docker-compose.env.yml up -d --wait postgres redis redpanda
	@COMPOSE_PROJECT_NAME=$(INTEGRATION_PROJECT) \
	 POSTGRES_DB=enterprise_template_test \
	 POSTGRES_PASSWORD=integration-postgres \
	 POSTGRES_PORT=$(INTEGRATION_POSTGRES_PORT) \
	 POSTGRES_DATA_PATH=pgdata \
	 KAFKA_PORT=$(INTEGRATION_KAFKA_PORT) \
	 REDIS_PORT=$(INTEGRATION_REDIS_PORT) \
	 REDIS_PASSWORD=integration-redis \
	 REDIS_DATA_PATH=redisdata \
	 docker compose -f docker-compose.env.yml run --rm topic-init

.PHONY: integration.test
integration.test: tools.goose ## Run PostgreSQL and Kafka integration tests.
	@GOOSE_BIN="$(GOOSE)" TEST_POSTGRES_DSN="$${TEST_POSTGRES_DSN:-postgres://postgres:integration-postgres@127.0.0.1:$(INTEGRATION_POSTGRES_PORT)/enterprise_template_test?sslmode=disable}" \
	 TEST_KAFKA_BROKERS="$${TEST_KAFKA_BROKERS:-127.0.0.1:$(INTEGRATION_KAFKA_PORT)}" \
	 TEST_KAFKA_TOPIC="$${TEST_KAFKA_TOPIC:-events}" \
	 TEST_KAFKA_DLQ_TOPIC="$${TEST_KAFKA_DLQ_TOPIC:-events.dlq}" \
	 TEST_REDIS_ADDR="$${TEST_REDIS_ADDR:-127.0.0.1:$(INTEGRATION_REDIS_PORT)}" \
	 TEST_REDIS_PASSWORD="$${TEST_REDIS_PASSWORD:-integration-redis}" \
	 go test -race -count=1 -v ./internal/apiserver/store ./internal/messaging/consumer ./pkg/token -run Integration

.PHONY: integration.smoke
integration.smoke: build integration.migrate ## Start the API against integration dependencies and verify liveness/readiness.
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
	 POSTGRES_DATA_PATH=pgdata \
	 KAFKA_PORT=$(INTEGRATION_KAFKA_PORT) \
	 REDIS_PORT=$(INTEGRATION_REDIS_PORT) \
	 REDIS_PASSWORD=integration-redis \
	 REDIS_DATA_PATH=redisdata \
	 docker compose -f docker-compose.env.yml down -v --remove-orphans

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
		--openapiv2_opt=allow_merge=true,merge_file_name=apiserver,disable_default_errors=true,allow_delete_body=true,logtostderr=true \
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
		--openapiv2_opt=allow_merge=true,merge_file_name=apiserver,disable_default_errors=true,allow_delete_body=true,logtostderr=true \
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
deps: tools.goose ## Install build, migration and codegen tools.
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

.PHONY: migrate migrate.status
migrate: ## Apply pending migrations to GOOSE_DBSTRING before starting services.
migrate.status: ## Show migration status for GOOSE_DBSTRING.
migrate migrate.status: tools.goose
	@test -n "$$GOOSE_DBSTRING" || { echo 'Export GOOSE_DBSTRING for the same database used by the services.' >&2; exit 1; }
	@GOOSE_DRIVER=postgres "$(GOOSE)" -env=none -dir "$(PROJ_ROOT_DIR)/migrations" $(if $(filter migrate.status,$@),status,up)

.PHONY: integration.migrate
integration.migrate: tools.goose ## Apply Goose SQL to the isolated integration database.
	@GOOSE_DRIVER=postgres GOOSE_DBSTRING="postgres://postgres:integration-postgres@127.0.0.1:$(INTEGRATION_POSTGRES_PORT)/enterprise_template_test?sslmode=disable" $(GOOSE) -env=none -dir migrations up

.PHONY: tools.goose
tools.goose: $(GOOSE) ## Install the pinned official Goose CLI (PostgreSQL only).

$(GOOSE): Makefile
	@mkdir -p $(OUTPUT_DIR)/tools
	@go mod download github.com/pressly/goose/v3@$(GOOSE_VERSION)
	@go -C "$$(go env GOMODCACHE)/github.com/pressly/goose/v3@$(GOOSE_VERSION)" build -mod=readonly -tags '$(GOOSE_BUILD_TAGS)' -ldflags '-X main.version=$(GOOSE_VERSION)' -o "$(GOOSE)" ./cmd/goose

.PHONY: dev.worker dev.scheduler
dev.worker: ## Run the PostgreSQL task Worker; Kafka remains opt-in.
	@go run ./cmd/gin-enterprise-template-worker --config configs/configs.yaml
dev.scheduler: ## Run persistent Scheduler.
	@go run ./cmd/gin-enterprise-template-scheduler --config configs/configs.yaml
