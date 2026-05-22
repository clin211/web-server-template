# Project gin-enterprise-template

> **安全提示（必读）**
>
> 本仓库 `configs/gin-enterprise-template-apiserver.yaml` 中的密钥与密码字段是占位符（`CHANGE_ME_*`），并非可用值。
> **首次部署到任何环境（开发/测试/生产）前必须替换：**
>
> - `jwt.secret`：用 `openssl rand -hex 32` 生成（≥ 32 字符）
> - `postgresql.password` / `redis.password`：改为强密码或通过环境变量注入
>
> 程序启动时会校验默认/弱 secret 并直接拒绝运行（参见 `pkg/options/jwt_options.go`）。
>
> 完整 fork 后清单见 [`docs/FORK-CHECKLIST.md`](./docs/FORK-CHECKLIST.md)。
> 改进路线见 [`docs/template-improvements.md`](./docs/template-improvements.md)。

gin-enterprise-template 是一个基于 Go 语言开发的通用企业级后端 skeleton，采用简洁架构设计，具有代码质量高、扩展能力强、符合 Go 编码及最佳实践等特点。

gin-enterprise-template 具有以下特性：

- 软件架构：采用简洁架构设计，确保项目结构清晰、易维护；
- 高频 Go 包：使用了 Go 项目开发中常用的包，如 gin、otel、gorm、gin、uuid、cobra、viper、pflag、resty、govalidator、slog、protobuf、casbin、onexstack 等；
- 目录结构：遵循 [project-layout](https://github.com/golang-standards/project-layout) 规范，采用标准化的目录结构；
- 认证与授权：实现了基于 JWT 的认证和基于 Casbin 的授权功能；
- 错误处理：设计了独立的错误包及错误码管理机制；
- 构建与管理：使用高质量的 Makefile 对项目进行管理；
- 代码质量：通过 golangci-lint 工具对代码进行静态检查，确保代码质量；
- 测试覆盖：包含单元测试、性能测试、模糊测试和示例测试等多种测试案例；
- 丰富的 Web 功能：支持 Trace ID、优雅关停、中间件、跨域处理、异常恢复等功能；
- 多种数据交换格式：支持 JSON 和 Protobuf 数据格式的交换；
- 开发规范：遵循多种开发规范，包括代码规范、版本规范、接口规范、日志规范、错误规范以及提交规范等；
- API 设计：接口设计遵循 RESTful API 规范；
- 项目具有 Dockerfile，并且 Dockerfile 符合最佳实践；

## Getting Started

### Prerequisites

在开始之前，请确保您的开发环境中安装了以下工具：

**必需工具：**

- [Go](https://golang.org/dl/) 1.25.3 或更高版本
- [Git](https://git-scm.com/) 版本控制工具
- [Make](https://www.gnu.org/software/make/) 构建工具

**可选工具：**

- [Docker](https://www.docker.com/) 容器化部署
- [golangci-lint](https://golangci-lint.run/) 代码静态检查

**验证安装：**

```bash
$ go version  
go version go1.25.3 linux/amd64  
$ make --version  
GNU Make 4.3  
```

### Building

> 提示：项目配置文件配置项 `metadata.makefileMode` 不能为 `none`，如果为 `none` 需要自行构建。

在项目根目录下，执行以下命令构建项目：

1. 安装依赖工具和包

```bash
make deps  # 安装项目所需的开发工具  
go mod tidy # 下载 Go 模块依赖  
```

1. 生成代码

```bash
make protoc # generate gRPC code  
go get cloud.google.com/go/compute@latest cloud.google.com/go/compute/metadata@latest  
go mod tidy # tidy dependencies  
go generate ./... # run all go:generate directives  
```

1. 构建应用

```bash
make build # build all binary files locate in cmd/  
```

**构建结果：**

```bash
_output/platforms/  
├── linux/  
│   └── amd64/  
│       └── gin-enterprise-template-apiserver  # apiserver 服务二进制文件  
└── darwin/  
    └── amd64/  
        └── gin-enterprise-template-apiserver  
```

### Running

启动服务有多种方式：

1. 使用构建的二进制文件运行

  ```bash  
  # 启动 apiserver 服务  
  $ _output/platforms/linux/amd64/gin-enterprise-template-apiserver --config configs/gin-enterprise-template-apiserver.yaml  
  # 服务将在以下端口启动：  
  # - HTTP API: http://localhost:5555
  # - Health Check: http://localhost:5555/healthz  
  # - Metrics: http://localhost:5555/metrics  
  $ curl http://localhost:5555/healthz # 测试：打开另外一个终端，调用健康检查接口  
  ```

1. 使用 Docker 运行

```bash
# 构建镜像  
$ make image
$ docker run --name gin-enterprise-template-apiserver -v configs/gin-enterprise-template-apiserver.yaml:/etc/gin-enterprise-template-apiserver.yaml -p 5555:5555 gin-enterprise-template/gin-enterprise-template-apiserver:latest -c /etc/gin-enterprise-template-apiserver.yaml
```

**配置文件示例：**  

gin-enterprise-template-apiserver 配置文件 `configs/gin-enterprise-template-apiserver.yaml`：

```yaml
addr: 0.0.0.0:5555 # 服务监听地址
timeout: 30s # 服务端超时
otel:
  endpoint: 127.0.0.1:4327
  service-name: gin-enterprise-template-apiserver
  output-mode: otel
  level: debug
  add-source: true
  use-prometheus-endpoint: true
  slog: # 改配置项只有 output-mod 为 slog 时生效
    format: text
    time-format: "2006-01-02 15:04:05"
    output: stdout
```  

## 快速参考手册

### 构建和部署命令

#### 本地开发环境

```bash
# 1. 启动依赖服务（PostgreSQL, Redis, OTEL）
docker compose -f docker-compose.env.yml up -d

# 2. 构建应用
make build BINS=gin-enterprise-template-apiserver

# 3. 构建镜像
make image PLATFORM=linux_amd64 VERSION=v0.0.5-alpha IMAGES=gin-enterprise-template-apiserver

# 4. 启动应用容器
cd build/docker/gin-enterprise-template-apiserver
docker compose up -d

# 5. 查看日志
docker compose logs -f

# 6. 测试健康检查
curl localhost:5555/healthz
```

#### 生产环境部署

```bash
# 1. 准备生产配置
cp configs/gin-enterprise-template-apiserver.prod.yaml.example configs/gin-enterprise-template-apiserver.prod.yaml
vim configs/gin-enterprise-template-apiserver.prod.yaml  # 修改数据库地址、密码等

# 2. 构建生产镜像
make build BINS=gin-enterprise-template-apiserver
make image PLATFORM=linux_amd64 VERSION=v1.0.0 IMAGES=gin-enterprise-template-apiserver

# 3. 部署
cd build/docker/gin-enterprise-template-apiserver
VERSION=v1.0.0 docker compose -f docker-compose.prod.yml up -d

# 4. 验证
curl localhost:5555/healthz
docker logs gin-enterprise-template-apiserver
```

### 常用运维命令

#### 查看状态

```bash
# 查看运行中的容器
docker ps

# 查看所有容器（包括停止的）
docker ps -a

# 查看特定容器
docker ps | grep gin-enterprise-template-apiserver

# 查看容器详细信息
docker inspect gin-enterprise-template-apiserver

# 查看资源使用
docker stats gin-enterprise-template-apiserver
```

#### 日志管理

```bash
# 实时查看日志
docker logs -f gin-enterprise-template-apiserver

# 查看最近 50 行
docker logs --tail 50 gin-enterprise-template-apiserver

# 查看最近 30 分钟的日志
docker logs --since 30m gin-enterprise-template-apiserver

# 查看特定时间段
docker logs --since "2025-11-09T10:00:00" gin-enterprise-template-apiserver
```

#### 容器操作

```bash
# 启动容器
docker start gin-enterprise-template-apiserver

# 停止容器
docker stop gin-enterprise-template-apiserver

# 重启容器
docker restart gin-enterprise-template-apiserver

# 删除容器
docker rm gin-enterprise-template-apiserver

# 强制删除运行中的容器
docker rm -f gin-enterprise-template-apiserver
```

#### Docker Compose 操作

```bash
# 启动服务
docker compose up -d

# 停止服务
docker compose down

# 重启服务
docker compose restart

# 查看服务状态
docker compose ps

# 查看日志
docker compose logs -f

# 重新构建并启动
docker compose up -d --build
```

### 镜像管理

```bash
# 查看本地镜像
docker images | grep gin-enterprise-template-apiserver

# 删除镜像
docker rmi gin-enterprise-template/gin-enterprise-template-apiserver:v0.0.5-alpha

# 清理未使用的镜像
docker image prune

# 导出镜像
docker save gin-enterprise-template/gin-enterprise-template-apiserver:v0.0.5-alpha -o gin-enterprise-template-apiserver.tar

# 导入镜像
docker load -i gin-enterprise-template-apiserver.tar

# 标记镜像
docker tag gin-enterprise-template/gin-enterprise-template-apiserver:v0.0.5-alpha gin-enterprise-template/gin-enterprise-template-apiserver:latest
```

### 网络调试

```bash
# 查看容器网络配置
docker inspect gin-enterprise-template-apiserver | grep -A 20 "Networks"

# 查看 Docker 网络
docker network ls

# 查看网络详情
docker network inspect gin-enterprise-template_net

# 测试容器内网络连接（如果容器有 shell）
docker exec -it gin-enterprise-template-apiserver ping host.docker.internal

# 从宿主机测试端口
telnet localhost 5555
nc -zv localhost 5555
```

### API 测试

```bash
# 健康检查
curl -i localhost:5555/healthz

# 创建用户
curl -X POST http://localhost:5555/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "Test@123",
    "email": "test@example.com"
  }'

# 用户登录
curl -X POST http://localhost:5555/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "Test@123"
  }'

# 查看指标
curl localhost:5555/metrics

# 查看 pprof
curl localhost:5555/debug/pprof/
```

### 故障排查

#### 问题：容器启动后立即退出

```bash
# 1. 查看日志
docker logs gin-enterprise-template-apiserver

# 2. 检查退出代码
docker inspect gin-enterprise-template-apiserver | grep -A 5 "State"

# 3. 尝试交互式启动（如果可能）
docker run -it --rm gin-enterprise-template/gin-enterprise-template-apiserver:v0.0.5-alpha /bin/sh
```

#### 问题：无法连接数据库

```bash
# 1. 检查数据库是否运行
docker ps | grep postgres

# 2. 从宿主机测试数据库连接
telnet localhost 54321

# 3. 检查配置文件
cat configs/gin-enterprise-template-apiserver.docker.yaml | grep -A 5 postgresql

# 4. 查看应用日志
docker logs gin-enterprise-template-apiserver | grep -i "database\|postgres"
```

#### 问题：端口冲突

```bash
# 1. 查看端口占用
lsof -i :5555
netstat -an | grep 5555

# 2. 停止占用端口的进程
kill -9 <PID>

# 3. 修改 docker-compose.yml 使用其他端口（左侧为宿主机端口，右侧为容器内端口）
# ports:
#   - "5557:5555"
```

#### 问题：磁盘空间不足

```bash
# 查看 Docker 占用空间
docker system df

# 清理未使用的资源
docker system prune

# 清理所有未使用的镜像
docker image prune -a

# 清理构建缓存
docker builder prune
```

### 配置文件位置

| 文件 | 用途 |
|------|------|
| `configs/gin-enterprise-template-apiserver.yaml` | 本地开发配置（非 Docker） |
| `configs/gin-enterprise-template-apiserver.docker.yaml` | Docker 开发环境配置 |
| `configs/gin-enterprise-template-apiserver.prod.yaml.example` | 生产环境配置模板 |
| `build/docker/gin-enterprise-template-apiserver/Dockerfile` | 镜像构建文件 |
| `build/docker/gin-enterprise-template-apiserver/docker-compose.yml` | 开发环境 Compose |
| `build/docker/gin-enterprise-template-apiserver/docker-compose.prod.yml` | 生产环境 Compose |
| `docker-compose.env.yml` | 依赖服务 Compose |

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `VERSION` | 镜像版本 | `latest` |
| `TZ` | 时区 | `Asia/Shanghai` |
| `GOPROXY` | Go 代理 | `https://goproxy.cn,direct` |

### 端口映射

| 服务 | 容器端口 | 宿主机端口 |
|------|---------|-----------|
| gin-enterprise-template-apiserver | 5555 | 5555 |
| PostgreSQL | 5432 | 54321 |
| Redis | 6379 | 56379 |
| OTEL Collector | 4327 | 4327 |
| OTEL Collector (HTTP) | 4328 | 4328 |
| OTEL Health | 13133 | 13133 |

### 性能优化建议

#### 镜像构建优化

```bash
# 使用缓存加速构建
make image PLATFORM=linux_amd64 VERSION=vX.X.X IMAGES=gin-enterprise-template-apiserver

# 并行构建多个平台
make build.multiarch BINS=gin-enterprise-template-apiserver
```

#### 资源限制

在生产环境 `docker-compose.prod.yml` 中配置：

```yaml
deploy:
  resources:
    limits:
      cpus: '2.0'
      memory: 2G
    reservations:
      cpus: '0.5'
      memory: 512M
```

#### 日志限制

```yaml
logging:
  driver: json-file
  options:
    max-size: "50m"
    max-file: "5"
    compress: "true"
```

### 安全检查清单

- [ ] 修改默认 JWT 密钥
- [ ] 使用强密码（数据库、Redis）
- [ ] 配置文件权限设置为 600
- [ ] 启用 TLS/SSL（生产环境）
- [ ] 定期更新基础镜像
- [ ] 限制容器资源使用
- [ ] 配置防火墙规则
- [ ] 启用日志审计
- [ ] 定期备份数据

### 监控指标

```bash
# Prometheus 指标
curl localhost:5555/metrics

# 容器资源使用
docker stats gin-enterprise-template-apiserver

# 健康检查
while true; do curl -s localhost:5555/healthz | jq -r .timestamp; sleep 5; done
```

## 附录

### 项目结构

```bash
gin-enterprise-template/  
├── cmd/                     # 应用程序入口  
│   └── gin-enterprise-template-apiserver/       # apiserver 服务  
│       └── main.go          # 主函数  
├── internal/                # 私有应用程序代码  
│   └── apiserver/             # apiserver 内部包  
│       ├── biz/             # 业务逻辑层  
│       ├── handler/         # gin 处理器  
│       ├── model/           # GORM 数据模型  
│       ├── pkg/             # 内部工具包  
│       └── store/           # 数据访问层  
├── pkg/                     # 公共库代码  
│   ├── api/                 # API 定义  
├── examples/                # 示例代码  
│   └── client/              # 客户端示例  
├── configs/                 # 配置文件  
├── docs/                    # 项目文档  
├── build/                   # 构建配置  
│   └── docker/              # Docker 文件  
├── scripts/                 # 构建和部署脚本  
├── third_party/             # 第三方依赖  
├── Makefile                 # 构建配置  
├── go.mod                   # Go 模块文件  
├── go.sum                   # Go 模块校验文件  
└── README.md                # 项目说明文档  
```

### 相关链接

- [项目文档](docs/)
- [问题追踪](github.com/clin211/gin-enterprise-template/issues)
- [讨论区](github.com/clin211/gin-enterprise-template/discussions)
- [项目看板](github.com/clin211/gin-enterprise-template/projects)
- [发布页面](github.com/clin211/gin-enterprise-template/releases)

### 支持

如果这个项目对您有帮助，请考虑给我们一个 ⭐️ 来支持项目发展！

[![Star History Chart](https://api.star-history.com/svg?repos=github.com/clin211/gin-enterprise-template&type=Date)](https://star-history.com/#github.com/clin211/gin-enterprise-template&Date)
