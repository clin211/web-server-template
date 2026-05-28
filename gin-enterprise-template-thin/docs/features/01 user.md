# User 模块业务逻辑文档

## 目录

- [User 模块业务逻辑文档](#user-模块业务逻辑文档)
  - [目录](#目录)
  - [1. 模块概述](#1-模块概述)
    - [1.1 职责定位](#11-职责定位)
    - [1.2 依赖组件](#12-依赖组件)
    - [1.3 目录结构](#13-目录结构)
  - [2. 接口定义](#2-接口定义)
    - [2.1 UserBiz 核心接口](#21-userbiz-核心接口)
    - [2.2 UserExpansion 扩展接口](#22-userexpansion-扩展接口)
  - [3. 核心功能](#3-核心功能)
    - [3.1 创建用户 (Create)](#31-创建用户-create)
      - [业务流程](#业务流程)
      - [关键逻辑](#关键逻辑)
      - [错误处理](#错误处理)
    - [3.2 获取用户 (Get)](#32-获取用户-get)
      - [业务流程](#业务流程-1)
      - [权限说明](#权限说明)
    - [3.3 用户列表 (List)](#33-用户列表-list)
      - [业务流程](#业务流程-2)
      - [核心特性](#核心特性)
      - [并发安全说明](#并发安全说明)
    - [3.4 更新用户 (Update)](#34-更新用户-update)
      - [业务流程](#业务流程-3)
      - [更新逻辑](#更新逻辑)
    - [3.5 删除用户 (Delete)](#35-删除用户-delete)
      - [业务流程](#业务流程-4)
      - [权限说明](#权限说明-1)
    - [3.6 用户登录 (Login)](#36-用户登录-login)
      - [业务流程](#业务流程-5)
      - [令牌说明](#令牌说明)
      - [错误处理](#错误处理-1)
    - [3.7 修改密码 (ChangePassword)](#37-修改密码-changepassword)
      - [业务流程](#业务流程-6)
      - [密码处理](#密码处理)
    - [3.8 刷新令牌 (RefreshToken)](#38-刷新令牌-refreshtoken)
      - [业务流程](#业务流程-7)
      - [返回数据结构](#返回数据结构)
      - [中间件说明](#中间件说明)
      - [注意事项](#注意事项)
  - [4. 数据流转](#4-数据流转)
    - [4.1 请求流程](#41-请求流程)
    - [4.2 响应流程](#42-响应流程)
    - [4.3 模型转换](#43-模型转换)
  - [5. 错误处理](#5-错误处理)
    - [5.1 错误包装规范](#51-错误包装规范)
    - [5.2 常见错误码](#52-常见错误码)
    - [5.3 日志记录](#53-日志记录)
  - [6. 安全机制](#6-安全机制)
    - [6.1 密码安全](#61-密码安全)
    - [6.2 令牌机制](#62-令牌机制)
    - [6.3 授权机制](#63-授权机制)
    - [6.4 权限控制](#64-权限控制)
  - [7. 并发控制](#7-并发控制)
    - [7.1 List 接口并发处理](#71-list-接口并发处理)
    - [7.2 并发安全措施](#72-并发安全措施)
  - [附录](#附录)
    - [A. 相关文件路径](#a-相关文件路径)
    - [B. 依赖包](#b-依赖包)
    - [C. 常量定义](#c-常量定义)
    - [D. API 使用示例](#d-api-使用示例)
      - [刷新令牌接口](#刷新令牌接口)

---

## 1. 模块概述

### 1.1 职责定位

User 模块位于 `internal/apiserver/biz/v1/user/`，属于**业务逻辑层 (Biz Layer)**。

| 层级 | 职责 |
|------|------|
| Handler | 处理 HTTP 请求/响应，参数校验 |
| **Biz** | **业务逻辑编排，事务协调** |
| Store | 数据库访问，CRUD 操作 |
| Model | 数据模型定义 |

### 1.2 依赖组件

```go
type userBiz struct {
    store store.IStore     // 数据访问层接口
    authz *authz.Authz     // Casbin 授权引擎
}
```

### 1.3 目录结构

```
internal/apiserver/biz/v1/user/
├── user.go           # 接口定义与构造函数
├── create.go         # 创建用户业务逻辑
├── get.go            # 获取单个用户
├── list.go           # 用户列表（游标分页）
├── update.go         # 更新用户信息
├── delete.go         # 删除用户
├── login.go          # 用户登录
├── changepassword.go # 修改密码
└── refreshtoken.go   # 刷新令牌
```

---

## 2. 接口定义

### 2.1 UserBiz 核心接口

```go
// UserBiz 定义处理用户请求所需的方法
type UserBiz interface {
    Create(ctx context.Context, rq *v1.CreateUserRequest) (*v1.CreateUserResponse, error)
    Update(ctx context.Context, rq *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error)
    Delete(ctx context.Context, rq *v1.DeleteUserRequest) (*v1.DeleteUserResponse, error)
    Get(ctx context.Context, rq *v1.GetUserRequest) (*v1.GetUserResponse, error)
    List(ctx context.Context, rq *v1.ListUserRequest) (*v1.ListUserResponse, error)

    UserExpansion  // 嵌入扩展接口
}
```

### 2.2 UserExpansion 扩展接口

```go
// UserExpansion 定义用户操作的扩展方法（认证相关）
type UserExpansion interface {
    Login(ctx context.Context, rq *v1.LoginRequest) (*v1.LoginResponse, error)
    // RefreshToken 返回与 Login 相同的数据结构
    RefreshToken(ctx context.Context, rq *v1.RefreshTokenRequest) (*v1.LoginResponse, error)
    ChangePassword(ctx context.Context, rq *v1.ChangePasswordRequest) (*v1.ChangePasswordResponse, error)
}
```

---

## 3. 核心功能

### 3.1 创建用户 (Create)

**文件位置**: `internal/apiserver/biz/v1/user/create.go`

#### 业务流程

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ 1. 参数拷贝      │ -> │ 2. 唯一性检查    │ -> │ 3. 创建用户记录  │
│   (copier.Copy) │    │    username     │    │    (store)      │
└─────────────────┘    │    email        │    └─────────────────┘
                       │    phone        │           │
                       └─────────────────┘           │
                                                      v
                                              ┌─────────────────┐
                                              │ 4. 分配默认角色  │
                                              │    (Casbin)     │
                                              └─────────────────┘
```

#### 关键逻辑

| 步骤 | 说明 | 代码位置 |
|------|------|----------|
| 参数拷贝 | 使用 `copier.Copy` 将请求转换为模型 | `create.go:19-23` |
| 用户名检查 | 查询数据库确保用户名唯一 | `create.go:26-29` |
| 邮箱检查 | 可选字段，存在时检查唯一性 | `create.go:32-37` |
| 手机号检查 | 可选字段，存在时检查唯一性 | `create.go:40-45` |
| 分配角色 | 新用户默认分配 `user` 角色 | `create.go:51-54` |

#### 错误处理

```go
// 用户已存在错误
errno.ErrUserAlreadyExists

// 添加角色失败错误
errno.ErrAddRole
```

---

### 3.2 获取用户 (Get)

**文件位置**: `internal/apiserver/biz/v1/user/get.go`

#### 业务流程

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ 1. 构建查询条件   │ -> │ 2. 查询数据库     │ -> │ 3. 模型转换      │
│    where.T(ctx)  │    │    store.User()│    │    Proto 转换    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

#### 权限说明

- `where.T(ctx)` 会自动从上下文中提取用户身份
- 普通用户只能查询自己的信息
- 管理员可以查询所有用户

---

### 3.3 用户列表 (List)

**文件位置**: `internal/apiserver/biz/v1/user/list.go`

#### 业务流程

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ 1. 解析分页令牌  │ -> │ 2. 构建查询条件  │ -> │ 3. 查询数据库    │
│    page_token    │    │    cursor 分页   │    │    store.User() │
└─────────────────┘    │    pageSize限制 │    └─────────────────┘
                       └─────────────────┘           │
                                                      v
                                              ┌─────────────────┐
                                              │ 4. 并发数据处理  │
                                              │    errgroup     │
                                              │    goroutine    │
                                              └─────────────────┘
                                                      │
                                                      v
                                              ┌─────────────────┐
                                              │ 5. 生成分页令牌  │
                                              │    next_token   │
                                              └─────────────────┘
```

#### 核心特性

| 特性 | 实现方式 | 代码位置 |
|------|----------|----------|
| 游标分页 | 基于 `id` 字段游标查询 | `list.go:21-32` |
| 并发处理 | `errgroup` + `goroutine` | `list.go:58-90` |
| 并发限制 | `MaxErrGroupConcurrency` | `list.go:61` |
| PageSize 限制 | 默认 20，最大 100 | `list.go:35-42` |
| 权限过滤 | 非 admin 只能看到自己 | `list.go:48-50` |

#### 并发安全说明

```go
// 使用 sync.Map 保证并发写入安全
var m sync.Map

// 使用 errgroup 控制并发数量，并传播错误
eg, ctx := errgroup.WithContext(ctx)
eg.SetLimit(known.MaxErrGroupConcurrency)
```

---

### 3.4 更新用户 (Update)

**文件位置**: `internal/apiserver/biz/v1/user/update.go`

#### 业务流程

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ 1. 获取当前用户   │ -> │ 2. 唯一性检查    │ -> │ 3. 更新数据库     │
│   where.T(ctx)  │    │    username     │    │    store.User() │
└─────────────────┘    │    email        │    └─────────────────┘
                       │    phone        │
                       └─────────────────┘
```

#### 更新逻辑

| 字段 | 更新条件 | 唯一性检查 |
|------|----------|------------|
| Username | 请求值 ≠ 当前值 | ✓ |
| Email | 请求值 ≠ 当前值（或当前为空） | ✓ |
| Phone | 请求值 ≠ 当前值（或当前为空） | ✓ |
| Nickname | 请求值非空 | ✗ |

---

### 3.5 删除用户 (Delete)

**文件位置**: `internal/apiserver/biz/v1/user/delete.go`

#### 业务流程

```
┌─────────────────┐    ┌─────────────────┐
│ 1. 删除用户记录   │ -> │ 2. 移除角色策略   │
│    store.User() │    │    Casbin       │
└─────────────────┘    └─────────────────┘
```

#### 权限说明

```go
// 只有 root 用户可以删除用户，并且可以删除其他用户
// 所以这里不用 where.T()，因为 where.T() 会查询 root 用户自己
b.store.User().Delete(ctx, where.F("userID", rq.GetUserID()))
```

---

### 3.6 用户登录 (Login)

**文件位置**: `internal/apiserver/biz/v1/user/login.go`

#### 业务流程

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ 1. 查询用户      │ -> │ 2. 验证密码      │ -> │ 3. 签发双令牌     │
│    by username  │    │    bcrypt.Compare│   │    access_token │
└─────────────────┘    └─────────────────┘    │    refresh_token│
                                              └─────────────────┘
```

#### 令牌说明

| 令牌类型 | 用途 | 过期时间 |
|----------|------|----------|
| access_token | API 访问令牌 | 配置可调 |
| refresh_token | 刷新令牌 | 配置可调 |

#### 错误处理

```go
errno.ErrUserNotFound    // 用户不存在
errno.ErrPasswordInvalid // 密码错误
errno.ErrSignToken       // 令牌签名失败
```

---

### 3.7 修改密码 (ChangePassword)

**文件位置**: `internal/apiserver/biz/v1/user/changepassword.go`

#### 业务流程

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ 1. 获取当前用户   │ -> │ 2. 验证旧密码    │ -> │ 3. 加密新密码     │
│    where.T(ctx) │    │    bcrypt.Compare│   │    bcrypt.Hash  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                      │
                                                      v
                                              ┌─────────────────┐
                                              │ 4. 更新数据库     │
                                              └─────────────────┘
```

#### 密码处理

```go
// 密码比对
authn.Compare(存储的加密密码, 用户输入的明文密码)

// 密码加密
authn.Encrypt(用户输入的明文密码)
```

---

### 3.8 刷新令牌 (RefreshToken)

**文件位置**: `internal/apiserver/biz/v1/user/refreshtoken.go`

#### 业务流程

```sh
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ 1. Refresh Token │ -> │ 2. 验证令牌类型  │ -> │ 3. 获取用户信息   │
│   (Header)      │    │ token_type=     │    │    by userID    │
│                 │    │ "refresh"       │    └─────────────────┘
└─────────────────┘    └─────────────────┘           │
                       ┌─────────────────┐           │
                       │ RefreshAuthn    │ <─────────┘
                       │ Middleware      │
                       └─────────────────┘           │
                                                      v
                                              ┌─────────────────┐
                                              │ 4. 重新签发令牌   │
                                              │    access_token │
                                              │    refresh_token│
                                              └─────────────────┘
```

#### 返回数据结构

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIs...",
    "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
    "expireAt": "2025-12-30T19:06:49+08:00"
  }
}
```

**注意**: 返回结构与 `/v1/auth/login` 完全一致，便于前端统一处理。

#### 中间件说明

| 中间件 | 用途 | 接受的 Token 类型 |
|--------|------|-------------------|
| `AuthnMiddleware` | 普通 API 认证 | Access Token (`token_type="access"`) |
| `RefreshAuthnMiddleware` | 刷新令牌专用 | Refresh Token (`token_type="refresh"`) |

#### 注意事项

- 刷新令牌通过 Authorization header 传递：`Authorization: Bearer <refresh_token>`
- Request Body 可以为空（使用 `HandleNoBodyRequest` 处理）
- 刷新后会同时返回新的 access_token 和 refresh_token
- 旧令牌在刷新后即失效

---

## 4. 数据流转

### 4.1 请求流程

```sh
HTTP Request
    │
    v
┌─────────────┐
│  Handler    │  参数校验、协议转换
└─────────────┘
    │
    v
┌─────────────┐
│  Biz Layer  │  业务逻辑编排 ← 当前模块
└─────────────┘
    │
    v
┌─────────────┐
│  Store      │  数据库访问
└─────────────┘
    │
    v
┌─────────────┐
│  Database   │  PostgreSQL
└─────────────┘
```

### 4.2 响应流程

```sh
Database
    │
    v
Store (model.UserM)
    │
    v
Biz (模型转换)
    │
    v
Handler (*v1.User)
    │
    v
HTTP Response (JSON/Protobuf)
```

### 4.3 模型转换

```go
// Model -> Proto
conversion.UserModelToUserV1(userM) *v1.User

// Proto -> Model (使用 copier)
copier.Copy(&userM, rq)
```

---

## 5. 错误处理

### 5.1 错误包装规范

```go
// 必须使用 %w 包装底层错误
return nil, fmt.Errorf("failed to copy request: %w", err)
```

### 5.2 常见错误码

| 错误码 | 说明 | 使用场景 |
|--------|------|----------|
| `ErrUserAlreadyExists` | 用户已存在 | Create/Update 时用户名/邮箱/手机号冲突 |
| `ErrUserNotFound` | 用户不存在 | Login 时用户名不存在 |
| `ErrPasswordInvalid` | 密码无效 | Login/ChangePassword 时密码错误 |
| `ErrSignToken` | 令牌签名失败 | Login/RefreshToken 时签名出错 |
| `ErrAddRole` | 添加角色失败 | Create 时 Casbin 策略添加失败 |
| `ErrRemoveRole` | 移除角色失败 | Delete 时 Casbin 策略移除失败 |

### 5.3 日志记录

```go
// 错误日志
slog.ErrorContext(ctx, "Failed to copy request to model", "error", err)

// 警告日志
slog.WarnContext(ctx, "Username already exists", "username", userM.Username)

// 信息日志
slog.InfoContext(ctx, "Get users from backend storage", "count", len(users))
```

---

## 6. 安全机制

### 6.1 密码安全

| 机制 | 实现方式 |
|------|----------|
| 密码加密 | bcrypt 哈希算法 |
| 密码验证 | Constant-time 比较 |
| 明文传输禁止 | 数据库存储加密密码 |

### 6.2 令牌机制

```go
// 双令牌设计
token.Sign(userID) -> (accessToken, refreshToken, accessExpire, refreshExpire)
```

| 特性 | Access Token | Refresh Token |
|------|--------------|---------------|
| 用途 | API 访问 | 刷新令牌 |
| 有效期 | 较短（如 2 小时） | 较长（如 7 天） |
| 存储位置 | HTTP Header / Cookie | HTTP Only Cookie |
| Token Type | `access` | `refresh` |
| 认证中间件 | `AuthnMiddleware` | `RefreshAuthnMiddleware` |

### 6.3 授权机制

```go
// Casbin 策略管理
b.authz.AddGroupingPolicy(userID, roleName)      // 添加角色
b.authz.RemoveGroupingPolicy(userID, roleName)   // 移除角色
```

### 6.4 权限控制

```go
// where.T(ctx) 自动注入当前用户上下文
// 非管理员用户只能操作自己的数据
if contextx.Username(ctx) != known.AdminUsername {
    whr.T(ctx)  // 自动添加 userID 过滤条件
}
```

---

## 7. 并发控制

### 7.1 List 接口并发处理

```go
// 使用 errgroup 管理并发 goroutine
eg, ctx := errgroup.WithContext(ctx)
eg.SetLimit(known.MaxErrGroupConcurrency)  // 限制最大并发数

// 使用 sync.Map 保证并发写入安全
var m sync.Map

for _, user := range userList {
    eg.Go(func() error {
        // 业务处理
        userv1 := conversion.UserModelToUserV1(user)
        m.Store(user.ID, userv1)
        return nil
    })
}

// 等待所有 goroutine 完成
if err := eg.Wait(); err != nil {
    return nil, err
}
```

### 7.2 并发安全措施

| 措施 | 说明 |
|------|------|
| `errgroup.SetLimit()` | 限制最大并发 goroutine 数量 |
| `sync.Map` | 并发安全的 key-value 存储 |
| `context.Context` | 传播取消信号，防止 goroutine 泄漏 |
| `select ctx.Done()` | 响应上下文取消 |

---

## 附录

### A. 相关文件路径

| 类型 | 路径 |
|------|------|
| Biz 层 | `internal/apiserver/biz/v1/user/*.go` |
| Store 层 | `internal/apiserver/store/user.go` |
| Model 层 | `internal/apiserver/model/user.go` |
| Handler 层 | `internal/apiserver/handler/user.go` |
| Proto 定义 | `pkg/api/apiserver/v1/apiserver.proto` |
| 认证中间件 | `internal/pkg/middleware/gin/authn.go` |
| 核心处理函数 | `pkg/core/core.go` |

**新增文件说明**:
- `authn.go`: 包含 `AuthnMiddleware`（处理 Access Token）和 `RefreshAuthnMiddleware`（处理 Refresh Token）
- `core.go`: 新增 `HandleNoBodyRequest` 函数，用于处理不需要请求体的接口

### B. 依赖包

| 包 | 用途 |
|----|------|
| `github.com/jinzhu/copier` | 结构体拷贝 |
| `github.com/clin211/gin-enterprise-template/pkg/authn` | 认证工具 |
| `github.com/clin211/gin-enterprise-template/pkg/authz` | 授权 (Casbin) |
| `github.com/clin211/gin-enterprise-template/pkg/token` | JWT 令牌（支持 Access/Refresh Token） |
| `github.com/clin211/gin-enterprise-template/pkg/core` | 核心处理函数（`HandleNoBodyRequest` 等） |
| `github.com/clin211/gin-enterprise-template/pkg/store/where` | 查询条件构建 |
| `github.com/clin211/gin-enterprise-template/internal/pkg/middleware/gin` | 认证中间件 |
| `golang.org/x/sync/errgroup` | 并发控制 |
| `log/slog` | 结构化日志 |

### C. 常量定义

| 常量 | 值 | 说明 |
|------|---|------|
| `known.RoleUser` | `"user"` | 默认用户角色 |
| `known.AdminUsername` | `"root"` | 管理员用户名 |
| `known.MaxErrGroupConcurrency` | 配置定义 | 最大并发数 |
| `token.TokenTypeAccess` | `"access"` | Access Token 类型标识 |
| `token.TokenTypeRefresh` | `"refresh"` | Refresh Token 类型标识 |

### D. API 使用示例

#### 刷新令牌接口

**请求**:
```bash
PUT /v1/auth/refresh-token
Authorization: Bearer <refresh_token>
Content-Type: application/json
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expireAt": "2025-12-30T19:06:49+08:00"
  }
}
```

**核心函数使用**:
```go
// Handler 中使用 HandleNoBodyRequest 处理无 body 请求
func (h *Handler) RefreshToken(c *gin.Context) {
    core.HandleNoBodyRequest(c, h.biz.UserV1().RefreshToken)
}
```
