# apidocs — API 文档基础库

基于 [Huma](https://github.com/danielgtaylor/huma) (OpenAPI 3.1) 封装的 API 文档库，
一行代码挂载到 Gin 引擎，支持多种 UI 主题和自定义样式。

## 特性

- **一行挂载**：`apidocs.Mount(r, apidocs.Options{...})`
- **多种 UI**：Scalar（默认）、Swagger UI、Redoc、Stoplight Elements
- **自定义样式**：CSS / Logo / 暗黑模式 / 品牌名称
- **安全方案**：Bearer JWT / API Key / OAuth2
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

    // 注册路由
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
    // 访问 http://localhost:8080/docs 查看文档
}
```

## UI 主题

```go
// Scalar（默认，现代风格，支持暗黑模式）
apidocs.Mount(r, apidocs.Options{Theme: apidocs.ThemeScalar})

// Swagger UI（经典风格）
apidocs.Mount(r, apidocs.Options{Theme: apidocs.ThemeSwagger})

// Redoc（三栏布局，适合大量 API）
apidocs.Mount(r, apidocs.Options{Theme: apidocs.ThemeRedoc})

// Stoplight Elements（设计优先）
apidocs.Mount(r, apidocs.Options{Theme: apidocs.ThemeStoplight})
```

## 自定义样式

```go
apidocs.Mount(r, apidocs.Options{
    Title:   "My API",
    Version: "1.0.0",
    Theme:   apidocs.ThemeScalar,
    DarkMode: true,                    // 暗黑模式
    CSS:     ".topbar { background: #1a1a2e; }", // 自定义 CSS
    Logo:    logoPNG,                  // []byte, 自定义 Logo
    LogoContentType: "image/png",
    CustomHeadHTML: `<meta name="author" content="MyCorp">`,
})
```

## 安全方案

```go
apidocs.Mount(r, apidocs.Options{
    Title:   "My API",
    Version: "1.0.0",
    SecuritySchemes: map[string]apidocs.SecurityScheme{
        "BearerAuth": {
            Type:         "http",
            Scheme:       "bearer",
            BearerFormat: "JWT",
            Description:  "JWT Bearer token",
        },
        "ApiKey": {
            Type:        "apiKey",
            In:          "header",
            Name:        "X-API-Key",
            Description: "API Key in header",
        },
    },
})
```

## 配置项

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| Title | string | "API" | API 标题 |
| Version | string | "1.0.0" | API 版本 |
| Description | string | "" | API 描述（支持 Markdown） |
| DocsPath | string | "/docs" | 文档 UI 路径 |
| APIPrefix | string | "" | API 前缀（影响 server URL 显示） |
| MetaPath | string | "/api/v1/meta" | 元信息端点路径 |
| EnableMeta | *bool | true | 是否注册元信息端点 |
| Theme | Theme | ThemeScalar | UI 主题 |
| DarkMode | bool | false | 暗黑模式 |
| Logo | []byte | 内置 SVG | Logo 图片 |
| LogoContentType | string | "image/png" | Logo MIME 类型 |
| CSS | string | 内置 CSS | 自定义 CSS |
| SecuritySchemes | map | nil | OpenAPI 安全方案 |
| Servers | []*huma.Server | 自动 | OpenAPI servers |
| ScalarConfig | map | nil | Scalar 专属配置 |
| CustomHeadHTML | string | "" | 注入 `<head>` 的 HTML |

## 自动生成的端点

| 路径 | 说明 |
|------|------|
| `<DocsPath>` | 文档 UI 页面 |
| `<DocsPath>/assets/docs.css` | CSS 样式文件 |
| `<DocsPath>/assets/logo` | Logo 图片 |
| `/openapi.json` | OpenAPI 3.1 JSON |
| `/openapi.yaml` | OpenAPI 3.1 YAML |
| `/api/v1/meta` | 文档元信息（可禁用） |

## License

MIT
