# syntax=docker/dockerfile:1.7

# 0) Build args (overridable in CI)
ARG BUILDER_IMAGE=golang:1.25.0
ARG RUNTIME_IMAGE=gcr.io/distroless/base-debian12:nonroot
ARG UID=65532
ARG GID=65532

# 1) Builder stage
FROM golang:1.25.3 AS builder
ENV GOTOOLCHAIN=auto
ARG GOPROXY=https://goproxy.cn,direct
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
FROM scratch AS runtime
WORKDIR /app
# 复制 CA 证书以支持 HTTPS（如不需要可移除该行）
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
# 仅复制二进制
COPY --from=builder /app/gin-enterprise-template-apiserver /app/gin-enterprise-template-apiserver

# 使用非 root（数值 UID 即可）
USER 10001

# 与 configs/gin-enterprise-template-apiserver.yaml 中 http.addr 端口保持一致
EXPOSE 5555

# 健康检查（k8s 之外的 docker 运行时也能感知服务可用性）
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/app/gin-enterprise-template-apiserver", "--help"]

ENTRYPOINT ["/app/gin-enterprise-template-apiserver"]
# 预计用 Compose volumes 挂载配置文件，保留默认命令
CMD ["-c", "/app/configs/gin-enterprise-template-apiserver.yaml"]
