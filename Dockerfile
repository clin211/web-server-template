# syntax=docker/dockerfile:1.7

# 0) Build args (overridable in CI)
ARG BUILDER_IMAGE=golang:1.26.6
ARG RUNTIME_IMAGE=gcr.io/distroless/base-debian12:nonroot
ARG UID=65532
ARG GID=65532

# 1) Builder stage
FROM ${BUILDER_IMAGE} AS builder
ENV GOTOOLCHAIN=auto
ARG GOPROXY=https://proxy.golang.org|direct
ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=sum.golang.org
ARG OS=linux
ARG ARCH=amd64
WORKDIR /workspace

# Optional: install build-time tools (as needed)
# RUN apt-get update && apt-get install -y --no-install-recommends upx && rm -rf /var/lib/apt/lists/*

# Work directory
WORKDIR /workspace

# Install minimal tools required to download files securely
# RUN apk add --no-cache curl ca-certificates

# Download the static tini binary (example for amd64); use a different file for other architectures
# Then make it executable so it can be used as PID 1 in the final image
# 移除未使用的 tini 下载步骤
# RUN curl -fsSL -o /usr/bin/tini https://github.com/krallin/tini/releases/download/v0.19.0/tini-static-amd64 \
#  && chmod +x /usr/bin/tini

# Use Go modules cache - The most important cache optimization
# Copy go.mod and go.sum first to leverage Docker layer cache
COPY go.mod go.sum ./

# Use cache mount to cache Go modules
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    # 这里继续执行 go mod download
    go mod download

# Copy source code
COPY . .

# Go build parameters (disable CGO for static linking with scratch base)
ENV CGO_ENABLED=0 GOOS=${OS} GOARCH=${ARCH} GO111MODULE=on GOCACHE=/root/.cache/go-build GOMODCACHE=/go/pkg/mod

# Build with cache mount for make build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    make build BINS=gin-enterprise-template-apiserver

# 将 Makefile 的构建产物复制到固定位置，供 runtime 阶段 COPY
RUN mkdir -p /app && cp -v _output/platforms/${OS}/${ARCH}/gin-enterprise-template-apiserver /app/gin-enterprise-template-apiserver

# 2) Runtime stage：极小基础镜像
FROM ${RUNTIME_IMAGE} AS runtime
ARG UID
ARG GID
WORKDIR /app
COPY --from=builder /app/gin-enterprise-template-apiserver /app/gin-enterprise-template-apiserver
COPY configs/configs.container.yaml /app/configs/configs.yaml

# distroless nonroot 镜像内置 65532:65532。
USER ${UID}:${GID}

# 与 configs/configs.yaml 中 http.addr 保持一致。
EXPOSE 5558

# distroless 不包含 shell/curl；部署平台应通过 /healthz 配置 HTTP 探针。

ENTRYPOINT ["/app/gin-enterprise-template-apiserver"]
CMD ["-c", "/app/configs/configs.yaml"]
