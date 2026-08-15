# common/response

统一 HTTP JSON 响应封装、应用错误处理、状态码规范和 i18n 消息 key 常量。

## 设计理念

- **统一信封**：所有 API 响应遵循 `{"code", "msg", "data"}` 格式
- **稳定错误码**：字符串 `Code`（如 `"NOT_FOUND"`）供客户端分支判断，不随版本变更语义
- **数字业务码**：`200`（成功）、`1000-1999`（客户端错误）、`2000-2999`（系统错误）
- **i18n 优先**：错误存储消息 key，在渲染时根据请求 locale 解析为本地化文本
- **解耦**：核心模块不依赖 gin；gin 集成在 `common/response/gin` 子模块

## 响应格式

### 成功

```json
{"code": 200, "msg": "success", "data": {...}}
```

### 错误

```json
{"code": 1001, "msg": "未找到", "error": "NOT_FOUND", "data": null, "details": null}
```

## 快速使用

### 服务层 — 返回 AppError

```go
import "github.com/LingByte/ling-base/common/response"

func GetUser(id string) (*User, error) {
    user, err := db.Find(id)
    if err != nil {
        return nil, response.WrapErr(response.CodeInternal, err)
    }
    if user == nil {
        return nil, response.NewI18n(response.CodeNotFound, response.KeyNotFound)
    }
    return user, nil
}
```

### HTTP 层 — gin 集成

```go
import ginresp "github.com/LingByte/ling-base/common/response/gin"

func Handler(c *gin.Context) {
    user, err := service.GetUser(id)
    if err != nil {
        ginresp.WriteError(c, err)  // 自动转换为 AppError 信封
        return
    }
    ginresp.Success(c, user)
}
```

### i18n 集成

```go
import (
    "github.com/LingByte/ling-base/i18n"
    ginresp "github.com/LingByte/ling-base/common/response/gin"
    i18ngin "github.com/LingByte/ling-base/i18n/gin"
)

func setupRouter(manager *i18n.Manager) *gin.Engine {
    r := gin.Default()
    
    // i18n 中间件 — 从 Accept-Language 解析 locale
    r.Use(i18ngin.Middleware(manager))
    
    // 设置 response 的消息解析器
    ginresp.Resolver = ginresp.ResolverFunc(func(key string, args ...any) string {
        locale := i18ngin.GetLocale(c)  // 从 context 获取
        return manager.T(locale, key, args...)
    })
    
    return r
}
```

## 错误码规范

### 字符串 Code（稳定标识符）

| Code | HTTP | 数字码 | 说明 |
|------|------|--------|------|
| `BAD_REQUEST` | 400 | 1000 | 请求参数错误 |
| `VALIDATION_FAILED` | 400 | 1011 | 数据校验失败 |
| `UNAUTHORIZED` | 401 | 1002 | 未授权 |
| `AUTH_FAILED` | 401 | 1002 | 认证失败 |
| `CREDENTIAL_INVALID` | 401 | 1002 | 凭证无效 |
| `FORBIDDEN` | 403 | 1003 | 禁止访问 |
| `TENANT_MISMATCH` | 403 | 1006 | 租户不匹配 |
| `NOT_FOUND` | 404 | 1001 | 资源未找到 |
| `CONFLICT` | 409 | 1004 | 资源冲突 |
| `DUPLICATE` | 409 | 1010 | 资源重复 |
| `RATE_LIMITED` | 429 | 1005 | 请求频率限制 |
| `QUOTA_EXCEEDED` | 402 | 1007 | 配额超限 |
| `UPSTREAM_TIMEOUT` | 504 | 1008 | 上游超时 |
| `SERVICE_UNAVAILABLE` | 503 | 1009 | 服务不可用 |
| `PROVIDER_ERROR` | 503 | 2002 | 供应商错误 |
| `INTERNAL` | 500 | 2000 | 内部错误 |

### 数字业务码分段

| 范围 | 用途 |
|------|------|
| 200 | 成功 |
| 1000-1099 | 通用业务错误 |
| 1100-1199 | 认证错误 |
| 1200-1299 | 租户错误 |
| 1300-1399 | 权限错误 |
| 2000-2999 | 系统错误 |
| 3000+ | 应用自定义（预留） |

## i18n 消息 Key 常量

```go
// common.*
response.KeySuccess           // "common.success"
response.KeyNotFound           // "common.not_found"
response.KeyUnauthorized       // "common.unauthorized"
response.KeyForbidden          // "common.forbidden"
response.KeyInvalidParams      // "common.invalid_params"
response.KeyConflict           // "common.conflict"
response.KeyRateLimited        // "common.rate_limited"
response.KeyInternalError      // "common.internal_error"

// auth.*
response.KeyAuthInvalidCredentials  // "auth.invalid_credentials"
response.KeyAuthMissingToken        // "auth.missing_token"
response.KeyAuthInvalidToken        // "auth.invalid_token"

// tenant.*
response.KeyTenantNotFound          // "tenant.not_found"
response.KeyTenantEmailExists       // "tenant.email_exists"

// perm.*
response.KeyPermInsufficient        // "perm.insufficient"

// validation.*
response.KeyValidationRequired      // "validation.required"
response.KeyValidationEmail         // "validation.email"
response.KeyValidationMin           // "validation.min"
```

## AppError 构造器

```go
// 从 Code 创建（i18n 在渲染时解析）
err := response.Err(response.CodeNotFound)

// 带直接消息
err := response.New(response.CodeBadRequest, "invalid email format")

// 格式化消息
err := response.Newf(response.CodeValidation, "field %s is required", "email")

// 带 i18n key 和参数
err := response.NewI18n(response.CodeNotFound, response.KeyNotFound)

// 包装底层错误
err := response.Wrap(response.CodeInternal, "db query failed", dbErr)
err := response.WrapErr(response.CodeInternal, dbErr)
err := response.WrapI18n(response.CodeNotFound, response.KeyNotFound, dbErr)
```

## Builder 方法

```go
err := response.Err(response.CodeBadRequest).
    WithStatus(422).                    // 覆盖 HTTP 状态码
    WithDetails(map[string]any{         // 附加结构化详情
        "field": "email",
        "reason": "empty",
    }).
    WithCause(originalErr)              // 附加底层错误
```

## MessageResolver

`MessageResolver` 接口将 i18n key 解析为本地化字符串：

```go
type MessageResolver interface {
    Resolve(key string, args ...any) string
}
```

内置实现：

- **StaticResolver** — map 内存查找，适合测试和简单应用
- **ChainResolver** — 多级回退链
- **NoopResolver** — 返回 key 本身（默认）
- **ResolverFunc** — 函数适配器

```go
resolver := &response.StaticResolver{
    Messages: map[string]string{
        "common.not_found": "未找到",
        "common.forbidden": "禁止访问",
    },
}
envelope := response.ErrorEnvelope(appErr, resolver)
```

## 分页

```go
// 构造分页响应
page := response.NewPage(users, total, pageNum, pageSize)

// 返回
ginresp.Success(c, page)
// → {"code": 200, "msg": "success", "data": {"list": [...], "total": 100, "page": 1, "size": 20, "total_page": 5}}
```

## Gin 集成

| 函数 | 说明 |
|------|------|
| `gin.Success(c, data)` | 200 成功响应 |
| `gin.SuccessI18n(c, key, data, args...)` | 200 带本地化消息 |
| `gin.Created(c, data)` | 201 创建成功 |
| `gin.NoContent(c)` | 204 无内容 |
| `gin.WriteError(c, err)` | 自动转换并渲染错误 |
| `gin.Fail(c, msg, data)` | 500 内部错误 |
| `gin.FailWithCode(c, code, msg, data)` | 指定 Code 的错误 |
| `gin.FailI18n(c, key, data, args...)` | 本地化错误 |
| `gin.FailAppError(c, ae)` | 直接渲染 AppError |
| `gin.AbortWithStatusJSON(c, status, err)` | 中止并返回错误 |
| `gin.Recovery()` | panic 恢复中间件 |

## errors.Is / errors.As 支持

```go
err := response.Err(response.CodeNotFound)

// 比较 Code
if errors.Is(err, response.Err(response.CodeNotFound)) {
    // handle not found
}

// 提取 AppError
var ae *response.AppError
if errors.As(err, &ae) {
    fmt.Println(ae.Code, ae.HTTPStatus)
}
```

## License

MIT
