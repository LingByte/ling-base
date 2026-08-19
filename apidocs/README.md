# apidocs — API 文档基础库

基于 [Huma](https://github.com/danielgtaylor/huma) (OpenAPI 3.1) 封装的 API 文档库，
一行代码挂载到 Gin 引擎，支持多种 UI 主题、CDN/自托管资源、深度自定义样式。

## 特性

- **一行挂载**：`apidocs.Mount(r, apidocs.Options{...})`
- **4 种 UI**：Scalar（默认）、Swagger UI、Redoc、Stoplight Elements
- **CDN + 自托管**：公网 CDN / 自托管 BaseURL / 逐个 URL 自定义（支持离线/内网）
- **深度自定义**：CSS / JS / Logo / topbar / 环境标识 / 暗黑模式
- **安全方案**：Bearer JWT / API Key / OAuth2 (4 种 flow) / OpenID Connect
- **OpenAPI 元信息**：Contact / License / TermsOfService / ExternalDocs / GlobalSecurity
- **环境控制**：`EnabledFunc` 按环境开关文档（生产关闭 / 开发开启）
- **元信息端点**：自动注册 `/api/v1/meta` 返回文档入口
- **OpenAPI 导出**：JSON / YAML 下载

## 安装

```bash
go get github.com/LingByte/ling-base/apidocs
```

## 快速开始

```go
package main

import (
    "context"
    "net/http"

    "github.com/LingByte/ling-base/apidocs"
    "github.com/danielgtaylor/huma/v2"
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.New()
    api := apidocs.Mount(r, apidocs.Options{
        Title:   "My API",
        Version: "1.0.0",
    })

    type helloOutput struct {
        Body struct {
            Msg string `json:"msg"`
        }
    }
    huma.Register(api, huma.Operation{
        OperationID: "hello",
        Method:      http.MethodGet,
        Path:        "/hello",
        Summary:     "打招呼",
    }, func(ctx context.Context, _ *struct{}) (*helloOutput, error) {
        return &helloOutput{Body: struct {
            Msg string `json:"msg"`
        }{Msg: "hello"}}, nil
    })

    r.Run(":8080")
    // 访问 http://localhost:8080/docs
}
```

## UI 主题

```go
apidocs.Mount(r, apidocs.Options{Theme: apidocs.ThemeScalar})    // 默认，现代风格
apidocs.Mount(r, apidocs.Options{Theme: apidocs.ThemeSwagger})   // 经典风格
apidocs.Mount(r, apidocs.Options{Theme: apidocs.ThemeRedoc})     // 三栏布局
apidocs.Mount(r, apidocs.Options{Theme: apidocs.ThemeStoplight}) // 设计优先
```

## CDN 模式（离线/内网支持）

### 公网 CDN（默认）

```go
apidocs.Mount(r, apidocs.Options{
    CDN: apidocs.CDNConfig{Mode: apidocs.CDNModePublic},
})
```

### 自托管（内网/离线）

```go
apidocs.Mount(r, apidocs.Options{
    CDN: apidocs.CDNConfig{
        Mode:    apidocs.CDNModeSelfHosted,
        BaseURL: "https://assets.internal.company.com/openapi-ui",
    },
})
```

资源路径规则：
- Scalar: `<BaseURL>/scalar/api-reference.js`
- Swagger: `<BaseURL>/swagger-ui/swagger-ui-bundle.js` + `swagger-ui.css`
- Redoc: `<BaseURL>/redoc/redoc.standalone.js`
- Stoplight: `<BaseURL>/stoplight/elements.min.js` + `styles.min.css`

### 逐个 URL 自定义

```go
apidocs.Mount(r, apidocs.Options{
    CDN: apidocs.CDNConfig{
        Mode:     apidocs.CDNModeCustom,
        ScalarJS: "https://cdn.mycompany.com/scalar.js",
        SwaggerJS: "https://cdn.mycompany.com/swagger.js",
    },
})
```

## 自定义样式

```go
apidocs.Mount(r, apidocs.Options{
    Title:    "My API",
    Version:  "1.0.0",
    DarkMode: true,
    CSS:      ".topbar { background: #1a1a2e; }",
    Logo:     logoPNG,               // []byte
    LogoContentType: "image/png",
    CustomJS: "console.log('docs loaded');",
    CustomHeadHTML: `<meta name="author" content="MyCorp">`,
})
```

## 自定义 TopBar

```go
apidocs.Mount(r, apidocs.Options{
    Title: "My API",
    TopBar: apidocs.TopBarConfig{
        Subtitle:    "API Reference",           // 默认 "接口文档"
        EnvLabel:    "STAGING",                 // 环境标识
        EnvLabelColor: "#ef4444",               // 红色
        ExtraButtons: `<a href="/">返回首页</a>`,
        // HideTopBar:       true,              // 完全隐藏 topbar
        // CustomHTML:       "<header>...</header>", // 完全自定义
        // ShowExportButtons: &f,               // false 隐藏导出按钮
    },
})
```

## 安全方案

### 环境控制（生产关闭文档）

```go
// 用环境变量控制
apidocs.Mount(r, apidocs.Options{
    Title: "My API",
    EnabledFunc: func() bool {
        return os.Getenv("APP_ENV") != "prod"
    },
})
```

```go
// 用 config 模块控制
apidocs.Mount(r, apidocs.Options{
    EnabledFunc: func() bool { return cfg.Env != "prod" },
})
```

`EnabledFunc` 返回 false 时：
- `/docs` 文档 UI 不挂载
- `/api/v1/meta` 元信息端点不挂载
- `/openapi.json` 和 `/openapi.yaml` 不暴露（除非 `ExposeSpec: &true`）
- `huma.API` 仍正常返回，`huma.Register` 注册的路由正常工作

```go
// 生产环境关闭文档 UI，但保留 OpenAPI spec 给工具用
tr := true
apidocs.Mount(r, apidocs.Options{
    EnabledFunc: func() bool { return os.Getenv("APP_ENV") != "prod" },
    ExposeSpec:  &tr,
})
```

### Bearer JWT

```go
apidocs.Mount(r, apidocs.Options{
    SecuritySchemes: map[string]apidocs.SecurityScheme{
        "BearerAuth": {
            Type:         "http",
            Scheme:       "bearer",
            BearerFormat: "JWT",
        },
    },
    GlobalSecurity: []map[string][]string{
        {"BearerAuth": {}},
    },
})
```

### API Key

```go
SecuritySchemes: map[string]apidocs.SecurityScheme{
    "ApiKey": {Type: "apiKey", In: "header", Name: "X-API-Key"},
}
```

### OAuth2

```go
SecuritySchemes: map[string]apidocs.SecurityScheme{
    "OAuth2": {
        Type: "oauth2",
        Flows: &huma.OAuthFlows{
            AuthorizationCode: &huma.OAuthFlow{
                AuthorizationURL: "https://example.com/oauth/authorize",
                TokenURL:         "https://example.com/oauth/token",
                Scopes: map[string]string{
                    "read":  "read access",
                    "write": "write access",
                },
            },
        },
    },
}
```

## OpenAPI 元信息

```go
apidocs.Mount(r, apidocs.Options{
    Title:          "My API",
    Version:        "1.0.0",
    Description:    "## My API\n\n支持 **Markdown**。",
    TermsOfService: "https://example.com/terms",
    Contact: &huma.Contact{
        Name:  "API Support",
        Email: "support@example.com",
        URL:   "https://example.com/support",
    },
    License: &huma.License{
        Name: "MIT",
        URL:  "https://opensource.org/licenses/MIT",
    },
    ExternalDocs: &huma.ExternalDocs{
        URL:         "https://example.com/docs",
        Description: "更多文档",
    },
})
```

## 主题专属配置

```go
apidocs.Mount(r, apidocs.Options{
    Theme: apidocs.ThemeScalar,
    ScalarConfig: map[string]any{
        "hideModels":  true,
        "layout":      "classic",
    },
})

apidocs.Mount(r, apidocs.Options{
    Theme: apidocs.ThemeSwagger,
    SwaggerConfig: map[string]any{
        "docExpansion": "none",
        "filter":       true,
    },
})
```

## 配置项

### Options

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| Title | string | "API" | API 标题 |
| Version | string | "1.0.0" | API 版本 |
| Description | string | "" | API 描述（Markdown） |
| Contact | *huma.Contact | nil | 联系信息 |
| License | *huma.License | nil | 许可证 |
| TermsOfService | string | "" | 服务条款 URL |
| DocsPath | string | "/docs" | 文档 UI 路径 |
| APIPrefix | string | "" | API 前缀 |
| MetaPath | string | "/api/v1/meta" | 元信息端点路径 |
| EnableMeta | *bool | true | 是否注册元信息端点 |
| EnabledFunc | func() bool | nil(始终开启) | 环境控制：返回 false 关闭文档 UI + meta + spec |
| ExposeSpec | *bool | false | EnabledFunc=false 时是否仍暴露 /openapi.json |
| Theme | Theme | ThemeScalar | UI 主题 |
| DarkMode | bool | false | 暗黑模式 |
| Logo | []byte | 内置 SVG | Logo 图片 |
| LogoContentType | string | "image/png" | Logo MIME |
| CSS | string | 内置 CSS | 自定义 CSS |
| CustomJS | string | "" | 注入 `<body>` 末尾的 JS |
| CustomHeadHTML | string | "" | 注入 `<head>` 的 HTML |
| CDN | CDNConfig | 公网 CDN | 资源加载方式 |
| TopBar | TopBarConfig | 默认 topbar | 顶部导航栏 |
| SecuritySchemes | map | nil | 安全方案 |
| GlobalSecurity | []map | nil | 全局安全要求 |
| Servers | []*huma.Server | 自动 | OpenAPI servers |
| ExternalDocs | *huma.ExternalDocs | nil | 外部文档链接 |
| ScalarConfig | map | nil | Scalar 专属配置 |
| SwaggerConfig | map | nil | Swagger 专属配置 |
| RedocConfig | map | nil | Redoc 专属配置 |
| StoplightConfig | map | nil | Stoplight 专属配置 |

### CDNConfig

| 选项 | 说明 |
|------|------|
| Mode | `CDNModePublic` / `CDNModeSelfHosted` / `CDNModeCustom` |
| BaseURL | 自托管基址（Mode=SelfHosted 时） |
| ScalarJS / SwaggerJS / SwaggerCSS / RedocJS / StoplightJS / StoplightCSS | 逐个覆盖 |

### TopBarConfig

| 选项 | 说明 |
|------|------|
| HideTopBar | 完全隐藏 topbar |
| CustomHTML | 完全自定义 topbar HTML |
| Subtitle | 副标题（默认"接口文档"） |
| ShowExportButtons | 是否显示导出按钮（默认 true） |
| ExtraButtons | 额外按钮 HTML |
| EnvLabel | 环境标识（DEV/STAGING/PROD） |
| EnvLabelColor | 环境标识颜色 |

## 自动生成的端点

| 路径 | 说明 |
|------|------|
| `<DocsPath>` | 文档 UI |
| `<DocsPath>/assets/docs.css` | CSS |
| `<DocsPath>/assets/logo` | Logo |
| `/openapi.json` | OpenAPI 3.1 JSON |
| `/openapi.yaml` | OpenAPI 3.1 YAML |
| `/api/v1/meta` | 文档元信息（可禁用） |

## License

MIT
