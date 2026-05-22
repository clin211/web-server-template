# Fork 后必做事项清单

> 本清单适用于**手动 fork** 本模板（而非通过 forge 一键生成）的场景。
> forge 生成项目时大部分替换由 forge 自动完成，但仍建议过一遍本清单确认无遗漏。

## 1. 安全：替换所有默认密钥（P0，部署前必须）

- [ ] `configs/<binary-name>.yaml` 中 `jwt.secret`：用 `openssl rand -hex 32` 生成
- [ ] `configs/<binary-name>.yaml` 中 `postgresql.password`：改为强密码
- [ ] `configs/<binary-name>.yaml` 中 `redis.password`：改为强密码
- [ ] **建议**：把 `configs/<binary-name>.yaml` 加入 `.gitignore`，仅提交 `configs/<binary-name>.yaml.example`
- [ ] **建议**：通过环境变量注入敏感字段（参考 `.env.example`）

> 默认配置文件名由 `cmd/<binary-name>/app/server.go` 根据 `os.Args[0]` 动态推导，
> 改名二进制后无需手动改源码；初始仓库内对应的实际文件名是
> `configs/gin-enterprise-template-apiserver.yaml`。
- [ ] 启动时确认 `Validate()` 没有报 `secret matches a known insecure/default value`

## 2. 项目身份信息

- [ ] `go.mod` 第一行：`module github.com/your-org/your-project`
- [ ] `Makefile` 中 `ROOT_PACKAGE` 变量：与 `go.mod` 模块路径一致
- [ ] `Makefile` 中 `REGISTRY_PREFIX`：改为你的 Docker registry 命名空间
- [ ] `Makefile` 中 `VERSION_PACKAGE`：跟随模块路径
- [ ] `LICENSE`：替换版权所有人与组织名
- [ ] `README.md` 中所有 `gin-enterprise-template` 字符串替换为你的项目名
- [ ] `.git/config` 中 remote.origin.url 指向你自己的仓库

## 3. 服务名 / 二进制名

- [ ] `cmd/gin-enterprise-template-apiserver/`：目录重命名为 `cmd/<your-service>-apiserver/`
- [ ] `cmd/<your-service>-apiserver/main.go` 的 import 路径同步更新
- [ ] `cmd/<your-service>-apiserver/app/server.go` 中 cobra `Use` 字段
- [ ] `internal/apiserver/httpserver.go` 中 `otelgin.Middleware(...)` 与 `metrics.Initialize(...)` 的 service name
- [ ] `Dockerfile` 中 `make build BINS=` 与 `COPY` 路径
- [ ] `build/docker/<service>/`：目录重命名

> 提示：`server.go` 已使用 `os.Args[0]` 动态推导 `defaultHomeDir` / `defaultConfigName`，二进制改名后无需手动改源码。

## 4. 数据库与配置

- [ ] `configs/template.sql` → 改名为 `configs/<your-db-name>.sql`
- [ ] `configs/<binary-name>.yaml` 中 `postgresql.database` 改为目标库名
- [ ] 启动前先创建该数据库：`createdb <your-db-name>`
- [ ] 检查 `internal/apiserver/model/*.gen.go` 是否需要重新跑 `gen-gorm-model`

## 5. 端口与对外暴露

- [ ] `configs/<binary-name>.yaml` 中 `http.addr` 端口
- [ ] `Dockerfile` 中 `EXPOSE`
- [ ] `build/docker/<service>/docker-compose*.yml` 中 ports 映射
- [ ] `README.md` 中所有提到端口的位置（curl/healthz/metrics）

## 6. CI / CD

- [ ] `.github/workflows/`（如果有）调整工作流名称、镜像标签
- [ ] Docker registry credentials（GitHub secrets / 公司 vault）
- [ ] 部署目标的环境变量映射（k8s ConfigMap / Secret）

## 7. 文档与品牌

- [ ] `README.md` 顶部项目描述
- [ ] `docs/` 目录下所有引用旧项目名的位置
- [ ] `.claude/CLAUDE.md`（如果保留）的项目说明
- [ ] OpenAPI 中的 service name / version

## 8. 推荐删除（如果用不到）

- [ ] gRPC 相关文件（`pkg/api/.../*_grpc.pb.go`、`api/openapi/.../*.swagger.json`）
- [ ] `cmd/gen-gorm-model/`（除非你需要从数据库反向生成 model）
- [ ] `.claude/`（特定 IDE 配置）
- [ ] `docker-compose.env.yml` 中你不需要的依赖（如 OTEL Collector）

## 9. 安全 / 合规验收

- [ ] 跑 `golangci-lint run` 确认 lint 全过
- [ ] 跑 `go test ./...` 确认测试通过
- [ ] 跑 `gosec` 或类似工具扫描已知漏洞
- [ ] 检查 `git log` 中是否有不慎提交的密钥

---

**完成后**：把本文件本身也删除（你不需要再 fork 一次），或者改为内部使用文档。
