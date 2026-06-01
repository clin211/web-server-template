---
name: feature-implement
description: 根据规划文档完整实现一个功能（model + store + biz + handler），遵循 gin-enterprise-template 项目规范
---

你现在是一位**资深的 Go 后端工程师**，熟悉 gin-enterprise-template 项目的所有分层规范、命名风格、查询构建方式、错误处理习惯。

## 任务目标

根据用户提供的**规划文档**，完整实现文档中描述的功能模块。

**核心原则**（来自 @.claude/constitution.md）：

1. **简单性优先**：只实现 spec 中明确要求的功能，不过度工程化
2. **标准库优先**：优先使用 Go 标准库和 `pkg/` 下已有工具
3. **测试先行**：优先编写集成测试，使用真实依赖或 fake object

## 项目信息

- **包名路径**：`github.com/clin211/gin-enterprise-template`
- **Model 目录**：`internal/apiserver/model/`
- **数据库**：PostgreSQL + GORM（自动迁移）

## 执行流程

### 1. 阅读规划文档

先读取用户提到的规划文件，理解完整需求：

- 请求/响应结构
- 业务规则和边界情况
- 数据库变更

### 2. 确认数据库表结构

读取 schema 文件（如 `@configs/schema/xxx.sql`），确认：

- 新增/修改的表结构、字段、索引
- 外键、枚举、json 字段等特殊类型

### 3. 更新 GORM Model 生成配置

编辑 `cmd/gen-gorm-model/gen_gorm_model.go`，在 `GenerateTemplateModels` 函数中添加：

```go
// 模块名模块表
g.GenerateModelAs("table_name", "TableNameM")
```

运行生成命令：

```bash
make models
```

### 4. 实现完整功能（严格分层）

**开发过程中务必遵循** `@.claude/commands/arch-review.md` 中的架构规范，以减少后续重构时间。

**目录规范**：

| 层级       | 路径                                          | 说明                     |
| ---------- | --------------------------------------------- | ------------------------ |
| Model      | `internal/apiserver/model/`                   | GORM 生成的模型文件      |
| Store      | `internal/apiserver/store/`                   | 数据访问接口与实现       |
| Biz        | `internal/apiserver/biz/v1/模块名/`           | 业务逻辑（按动作分文件） |
| Handler    | `internal/apiserver/handler/模块名.go`        | HTTP 处理器              |
| Validation | `internal/apiserver/pkg/validation/模块名.go` | 请求校验                 |
| Conversion | `internal/apiserver/pkg/conversion/模块名.go` | Model <-> Proto 转换     |
| Errno      | `internal/pkg/errno/code.go`                  | 错误码定义（集中管理）    |
| Proto      | `pkg/api/apiserver/v1/模块名.proto`           | 请求/响应消息定义        |
| Known      | `internal/pkg/known/`                         | 项目常量定义             |

**常量定义**：
- 业务相关的常量（如角色名、状态值、配置键）放在 `internal/pkg/known/` 中
- **按模块分文件管理**：例如 `known.go`（通用常量）、`role.go`（角色相关）
- 新增常量时，在 `internal/pkg/known/` 下创建对应模块的 `xxx.go` 文件
- 示例：`known.RoleUser`、`known.RoleAdmin`、`known.AdminUsername`

**实现顺序**：

#### Step 1: 定义 Proto API

1. 在 `pkg/api/apiserver/v1/` 目录下创建 `模块名.proto` 定义请求/响应消息
2. 在 `pkg/api/apiserver/v1/apiserver.proto` 中定义 API 接口（service 方法）
3. 运行 `make protoc` 生成 pb 代码

**示例 proto 文件**（`pkg/api/apiserver/v1/xxx.proto`）：

```protobuf
syntax = "proto3";

package apiserver.v1;

import "google/api/annotations.proto";
import "google/protobuf/empty.proto";
import "github.com/onexstack/protobuf/github.com/onexstack/defaults/defaults.proto";

option go_package = "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1;v1";

// Xxx 模块请求/响应消息定义
message CreateXxxRequest { ... }
message CreateXxxResponse { string xxx_id = 1; }

// ...
```

**apiserver.proto 示例**：

```protobuf
// Xxx xxx
// @router /xxx/:id [GET]
rpc GetXxx(GetXxxRequest) returns (GetXxxResponse) {
    option (google.api.http) = {
        get: "/v1/xxx/{id}"
    };
    option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
        summary: "xxx"
        description: "xxxx"
        tags: "xxx"
    };
}
```

#### Step 2: 实现 Store 层

创建 `internal/apiserver/store/模块名.go`，参考 `user.go` 模式：

```go
package store

import (
    "context"

    storelogger "github.com/clin211/gin-enterprise-template/pkg/logger/slog/store"
    genericstore "github.com/clin211/gin-enterprise-template/pkg/store"
    "github.com/clin211/gin-enterprise-template/pkg/store/where"

    "github.com/clin211/gin-enterprise-template/internal/apiserver/model"
)

// XxxStore 定义了 xxx 模块在 store 层所实现的方法.
type XxxStore interface {
    Create(ctx context.Context, obj *model.XxxM) error
    Update(ctx context.Context, obj *model.XxxM) error
    Delete(ctx context.Context, opts *where.Options) error
    Get(ctx context.Context, opts *where.Options) (*model.XxxM, error)
    List(ctx context.Context, opts *where.Options) (int64, []*model.XxxM, error)

    XxxExpansion
}

// XxxExpansion 定义了 xxx 操作的附加方法.
// nolint: iface
type XxxExpansion interface {
    // 根据需要添加扩展方法
}

// xxxStore 是 XxxStore 接口的实现。
type xxxStore struct {
    *genericstore.Store[model.XxxM]
    core *datastore
}

// 确保 xxxStore 实现了 XxxStore 接口。
var _ XxxStore = (*xxxStore)(nil)

// newXxxStore 创建 xxxStore 的实例。
func newXxxStore(store *datastore) *xxxStore {
    return &xxxStore{
        Store: genericstore.NewStore[model.XxxM](store, storelogger.NewLogger()),
        core:  store,
    }
}
```

更新 `internal/apiserver/store/store.go`：

1. 在 `IStore` 接口添加 `Xxx() XxxStore`
2. 在 `datastore` 添加方法返回实例

#### Step 3: 实现 Biz 层

创建 `internal/apiserver/biz/v1/模块名/模块名.go` 定义接口：

```go
package 模块名

import (
    "context"

    "github.com/clin211/gin-enterprise-template/internal/apiserver/store"
    v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// XxxBiz 定义处理 xxx 请求所需的方法.
type XxxBiz interface {
    Create(ctx context.Context, rq *v1.CreateXxxRequest) (*v1.CreateXxxResponse, error)
    Update(ctx context.Context, rq *v1.UpdateXxxRequest) (*v1.UpdateXxxResponse, error)
    Delete(ctx context.Context, rq *v1.DeleteXxxRequest) (*v1.DeleteXxxResponse, error)
    Get(ctx context.Context, rq *v1.GetXxxRequest) (*v1.GetXxxResponse, error)
    List(ctx context.Context, rq *v1.ListXxxRequest) (*v1.ListXxxResponse, error)

    XxxExpansion
}

// XxxExpansion 定义 xxx 操作的扩展方法.
type XxxExpansion interface {
    // 根据需要添加扩展方法
}

// xxxBiz 是 XxxBiz 接口的实现.
type xxxBiz struct {
    store store.IStore
    // 可根据需要添加其他依赖：authz、producer、scheduler 等
}

// 确保 xxxBiz 实现了 XxxBiz 接口.
var _ XxxBiz = (*xxxBiz)(nil)

func New(store store.IStore) *xxxBiz {
    return &xxxBiz{store: store}
}
```

**Biz 文件组织方式**（每个动作一个文件）：

```
biz/v1/xxx/
├── xxx.go          # 接口定义和 New 函数
├── create.go       # 创建逻辑
├── update.go       # 更新逻辑
├── delete.go       # 删除逻辑
├── get.go          # 获取详情
├── list.go         # 列表查询
└── xxx_expand.go   # 扩展方法（如有）
```

**查询构建**（必须使用 `where.Options`）：

```go
opts := where.NewWhere().
    Q("field = ?", value).
    Q("status IN ?", []string{"pending", "processing"}).
    Order("created_at DESC").
    L(10)

count, items, err := s.store.Xxx().List(ctx, opts)
```

#### Step 4: 实现 Validation 层

在 `internal/apiserver/pkg/validation/模块名.go` 中添加验证规则：

```go
package validation

import (
    "context"

    genericvalidation "github.com/clin211/gin-enterprise-template/pkg/validation"

    "github.com/clin211/gin-enterprise-template/internal/pkg/errno"
    v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

func (v *Validator) ValidateXxxRules() genericvalidation.Rules {
    return genericvalidation.Rules{
        "FieldName": func(value any) error {
            s := value.(string)
            if s == "" {
                return errno.ErrInvalidArgument.WithMessage("fieldName cannot be empty")
            }
            return nil
        },
        // 其他字段校验...
    }
}

// ValidateCreateXxxRequest 校验创建 xxx 请求.
func (v *Validator) ValidateCreateXxxRequest(ctx context.Context, rq *v1.CreateXxxRequest) error {
    return genericvalidation.ValidateAllFields(rq, v.ValidateXxxRules())
}

// ValidateUpdateXxxRequest 校验更新 xxx 请求.
func (v *Validator) ValidateUpdateXxxRequest(ctx context.Context, rq *v1.UpdateXxxRequest) error {
    return genericvalidation.ValidateAllFields(rq, v.ValidateXxxRules())
}

// ValidateDeleteXxxRequest 校验删除 xxx 请求.
func (v *Validator) ValidateDeleteXxxRequest(ctx context.Context, rq *v1.DeleteXxxRequest) error {
    return genericvalidation.ValidateAllFields(rq, v.ValidateXxxRules())
}

// ValidateGetXxxRequest 校验获取 xxx 请求.
func (v *Validator) ValidateGetXxxRequest(ctx context.Context, rq *v1.GetXxxRequest) error {
    return genericvalidation.ValidateAllFields(rq, v.ValidateXxxRules())
}

// ValidateListXxxRequest 校验列表查询 xxx 请求.
func (v *Validator) ValidateListXxxRequest(ctx context.Context, rq *v1.ListXxxRequest) error {
    return genericvalidation.ValidateAllFields(rq, v.ValidateXxxRules())
}
```

#### Step 5: 实现 Conversion 层

创建 `internal/apiserver/pkg/conversion/模块名.go`，参考 `menu.go` 模式：

```go
package conversion

import (
    "github.com/clin211/gin-enterprise-template/pkg/core"

    "github.com/clin211/gin-enterprise-template/internal/apiserver/model"
    v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// XxxModelToXxxV1 将模型层的 XxxM 转换为 Protobuf 层的 Xxx.
func XxxModelToXxxV1(m *model.XxxM) *v1.Xxx {
    if m == nil {
        return &v1.Xxx{}
    }
    return &v1.Xxx{
        XxxID:   m.XxxID,
        Name:    m.Name,
        // ... 其他字段映射
    }
}

// XxxV1ToXxxModel 将 Protobuf 层的 Xxx 转换为模型层的 XxxM.
func XxxV1ToXxxModel(proto *v1.Xxx) *model.XxxM {
    var m model.XxxM
    _ = core.CopyWithConverters(&m, proto)
    return &m
}

// XxxModelListToXxxV1List 将模型列表转换为 proto 列表.
func XxxModelListToXxxV1List(list []*model.XxxM) []*v1.Xxx {
    result := make([]*v1.Xxx, len(list))
    for i, item := range list {
        result[i] = XxxModelToXxxV1(item)
    }
    return result
}
```

#### Step 6: 实现 Handler 层

创建 `internal/apiserver/handler/模块名.go`：

```go
package handler

import (
    "github.com/clin211/gin-enterprise-template/pkg/core"
    "github.com/gin-gonic/gin"
)

func init() {
    Register(func(v1 *gin.RouterGroup, handler *Handler) {
        rg := v1.Group("/xxx")
        rg.Use(handler.mws...)
        rg.POST("", handler.CreateXxx)           // 创建
        rg.PUT(":xxxID", handler.UpdateXxx)     // 更新
        rg.DELETE(":xxxID", handler.DeleteXxx)  // 删除
        rg.GET(":xxxID", handler.GetXxx)        // 获取详情
        rg.GET("", handler.ListXxx)              // 列表查询
    })
}

// CreateXxx 创建 xxx.
func (h *Handler) CreateXxx(c *gin.Context) {
    core.HandleJSONRequest(c, h.biz.XxxV1().Create, h.val.ValidateCreateXxxRequest)
}

// UpdateXxx 更新 xxx.
func (h *Handler) UpdateXxx(c *gin.Context) {
    core.HandleUriJSONRequest(c, h.biz.XxxV1().Update, h.val.ValidateUpdateXxxRequest)
}

// DeleteXxx 删除 xxx.
func (h *Handler) DeleteXxx(c *gin.Context) {
    core.HandleUriRequest(c, h.biz.XxxV1().Delete, h.val.ValidateDeleteXxxRequest)
}

// GetXxx 获取 xxx 详情.
func (h *Handler) GetXxx(c *gin.Context) {
    core.HandleUriRequest(c, h.biz.XxxV1().Get, h.val.ValidateGetXxxRequest)
}

// ListXxx 列出 xxx.
func (h *Handler) ListXxx(c *gin.Context) {
    core.HandleQueryRequest(c, h.biz.XxxV1().List, h.val.ValidateListXxxRequest)
}
```

### 5. 更新 Biz 和 Store 依赖注入

更新 `internal/apiserver/biz/biz.go`：

1. 在 import 区添加新模块的导入：
```go
import (
    // ... 其他导入
    xxxv1 "github.com/clin211/gin-enterprise-template/internal/apiserver/biz/v1/xxx"
)
```

2. 在 `IBiz` 接口添加方法：
```go
// XxxV1 获取 xxx 业务接口.
XxxV1() xxxv1.XxxBiz
```

3. 在 `biz` 结构体中添加依赖（如需要）：
```go
type biz struct {
    store store.IStore
    authz *authz.Authz
    // 添加新模块需要的依赖
}
```

4. 添加实现方法：
```go
// XxxV1 返回一个实现了 XxxBiz 接口的实例.
func (b *biz) XxxV1() xxxv1.XxxBiz {
    return xxxv1.New(b.store)
}
```

更新 `internal/apiserver/store/store.go`：

1. 在 `IStore` 接口添加方法：
```go
// Xxx 获取 xxx 存储接口.
Xxx() XxxStore
```

2. 添加实现方法：
```go
// Xxx 返回一个实现了 XxxStore 接口的实例.
func (store *datastore) Xxx() XxxStore {
    return newXxxStore(store)
}
```

### 6. 代码生成与验证

```bash
# 生成 proto 和 wire 代码
make protoc
make generate

# 代码风格检查与修复（重要！）
make lint

# 构建验证（确保编译通过）
make build
```

## 工具函数使用优先级

1. **pkg/ 下已有工具**（最高优先）
   - `pkg/store/where` - 查询构建
   - `pkg/validation` - 参数校验
   - `pkg/errorsx` - 错误处理
   - `pkg/core` - HTTP 处理助手、copier 工具

2. **github.com/samber/lo** - 列表/集合操作

3. **标准库** - `log/slog`, `context`, `fmt`

## 代码风格强制要求

| 方面     | 规范                                              |
| -------- | ------------------------------------------------ |
| 错误处理 | `fmt.Errorf("xxx: %w", err)` 包装               |
| 日志     | `log/slog` 结构化日志，记录 `traceID`、`userID`  |
| 命名     | 小写无下划线（除 model/ 和 pb 文件）             |
| 接口     | 小接口，单一职责                                  |
| 注释     | 公共 API 必须有 godoc                             |
| 魔法值   | 抽取为常量，放在 `internal/pkg/known/` 中       |

## 输出格式

按以下结构输出实现结果：

### 第一部分：需求理解

- 核心目标
- 输入/输出
- 关键边界

### 第二部分：Model 变更

- `gen_gorm_model.go` 需添加的配置

### 第三部分：完整代码

```go
// 文件：internal/apiserver/store/xxx.go
package store
// ...
```

### 第四部分：需要用户执行

- `make models` - 生成 GORM Model
- `make protoc` - 生成 Proto 代码
- `make generate` - 生成 Wire 代码
- `make lint` - 代码风格检查
- `make build` - 构建验证

---

现在请开始：先读取用户提供的规划文档。