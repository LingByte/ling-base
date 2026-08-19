// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package apidocs

import (
	_ "embed"
	"encoding/json"
	"html"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ──────────────────────────────────────────────
// Embedded default assets
// ──────────────────────────────────────────────

//go:embed assets/default.css
var defaultCSS string

//go:embed assets/logo.svg
var defaultLogoSVG []byte

// ──────────────────────────────────────────────
// UI renderer
// ──────────────────────────────────────────────

type uiRenderer struct {
	opts       Options
	css        string
	logo       []byte
	logoCT     string
	docsPath   string
	openAPIRef string // path to openapi.json
}

func newUIRenderer(opts Options) *uiRenderer {
	css := opts.CSS
	if css == "" {
		css = defaultCSS
	}
	logo := opts.Logo
	logoCT := opts.LogoContentType
	if logo == nil {
		logo = defaultLogoSVG
		logoCT = "image/svg+xml"
	}

	docsPath := strings.TrimSuffix(opts.DocsPath, "/")

	return &uiRenderer{
		opts:       opts,
		css:        css,
		logo:       logo,
		logoCT:     logoCT,
		docsPath:   docsPath,
		openAPIRef: "/openapi.json",
	}
}

func (u *uiRenderer) mount(r *gin.Engine) {
	cssPath := u.docsPath + "/assets/docs.css"
	logoPath := u.docsPath + "/assets/logo"

	// Serve CSS.
	r.GET(cssPath, func(c *gin.Context) {
		c.Header("Content-Type", "text/css; charset=utf-8")
		c.Header("Cache-Control", "public, max-age=300")
		c.String(http.StatusOK, u.css)
	})

	// Serve logo.
	r.GET(logoPath, func(c *gin.Context) {
		c.Header("Content-Type", u.logoCT)
		c.Header("Cache-Control", "public, max-age=86400")
		c.Data(http.StatusOK, u.logoCT, u.logo)
	})

	// Build the HTML page.
	htmlPage := u.renderHTML()

	// Serve docs UI.
	handler := func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Header("Cache-Control", "no-store")
		c.String(http.StatusOK, htmlPage)
	}
	r.GET(u.docsPath, handler)
	r.GET(u.docsPath+"/", handler)
}

func (u *uiRenderer) renderHTML() string {
	escTitle := html.EscapeString(u.opts.Title)
	escCSS := html.EscapeString(u.docsPath + "/assets/docs.css")
	escLogo := html.EscapeString(u.docsPath + "/assets/logo")
	escRef := html.EscapeString(u.openAPIRef)

	var body string
	switch u.opts.Theme {
	case ThemeSwagger:
		body = u.swaggerHTML(escRef, escTitle)
	case ThemeRedoc:
		body = u.redocHTML(escRef, escTitle)
	case ThemeStoplight:
		body = u.stoplightHTML(escRef, escTitle)
	default: // ThemeScalar
		body = u.scalarHTML(escRef, escTitle)
	}

	return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <meta name="color-scheme" content="` + darkModeAttr(u.opts.DarkMode) + `" />
  <meta name="referrer" content="no-referrer" />
  <title>` + escTitle + ` · API</title>
  <link rel="stylesheet" href="` + escCSS + `" />
  <link rel="icon" href="` + escLogo + `" />
  ` + u.opts.CustomHeadHTML + `
</head>
<body class="apidocs-body">
  <header class="apidocs-topbar">
    <div class="apidocs-brand">
      <img class="apidocs-logo" src="` + escLogo + `" alt="" width="28" height="28" />
      <div class="apidocs-brand-text">
        <strong id="apidocs-name">` + escTitle + `</strong>
        <span>接口文档</span>
      </div>
    </div>
    <div class="apidocs-actions">
      <a class="apidocs-btn-ghost" href="` + escRef + `" download="openapi.json">导出 JSON</a>
      <a class="apidocs-btn-ghost" href="/openapi.yaml" download="openapi.yaml">导出 YAML</a>
    </div>
  </header>
  <main class="apidocs-main">
` + body + `
  </main>
</body>
</html>`
}

func darkModeAttr(dark bool) string {
	if dark {
		return "dark light"
	}
	return "light"
}

// ──────────────────────────────────────────────
// Scalar UI
// ──────────────────────────────────────────────

func (u *uiRenderer) scalarHTML(ref, title string) string {
	cfg := u.scalarConfig()
	cfgBytes, _ := json.Marshal(cfg)
	cfgJSON := string(cfgBytes)
	return `    <script id="api-reference" data-url="` + ref + `" data-configuration="` + html.EscapeString(cfgJSON) + `"></script>
    <script src="https://unpkg.com/@scalar/api-reference@1.44.20/dist/browser/standalone.js"
      crossorigin></script>`
}

func (u *uiRenderer) scalarConfig() map[string]any {
	cfg := map[string]any{
		"theme":              "default",
		"layout":             "modern",
		"darkMode":           u.opts.DarkMode,
		"forceDarkModeState": darkModeStr(u.opts.DarkMode),
		"hideModels":         false,
		"hideDownloadButton": true,
		"persistAuth":        true,
		"showSidebar":        true,
		"metaData": map[string]any{
			"title": u.opts.Title + " · API",
		},
		"agent": map[string]any{"disabled": true},
	}
	// Merge user-provided Scalar config (overrides defaults).
	for k, v := range u.opts.ScalarConfig {
		cfg[k] = v
	}
	return cfg
}

func darkModeStr(dark bool) string {
	if dark {
		return "dark"
	}
	return "light"
}

// ──────────────────────────────────────────────
// Swagger UI
// ──────────────────────────────────────────────

func (u *uiRenderer) swaggerHTML(ref, title string) string {
	return `    <div id="swagger-ui"></div>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.18.2/swagger-ui.css" />
    <script src="https://unpkg.com/swagger-ui-dist@5.18.2/swagger-ui-bundle.js"></script>
    <script>
      window.onload = function() {
        window.ui = SwaggerUIBundle({
          url: "` + ref + `",
          dom_id: "#swagger-ui",
          deepLinking: true,
          presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
          plugins: [SwaggerUIBundle.plugins.DownloadUrl],
          layout: "StandaloneLayout",
          ` + darkModeSwagger(u.opts.DarkMode) + `
        });
      };
    </script>`
}

func darkModeSwagger(dark bool) string {
	if dark {
		return `theme: "dark",`
	}
	return ``
}

// ──────────────────────────────────────────────
// Redoc
// ──────────────────────────────────────────────

func (u *uiRenderer) redocHTML(ref, title string) string {
	return `    <redoc spec-url="` + ref + `"></redoc>
    <script src="https://cdn.jsdelivr.net/npm/redoc@2.1.5/bundles/redoc.standalone.js"></script>`
}

// ──────────────────────────────────────────────
// Stoplight Elements
// ──────────────────────────────────────────────

func (u *uiRenderer) stoplightHTML(ref, title string) string {
	return `    <elements-api
      apiDescriptionUrl="` + ref + `"
      router="hash"
      layout="sidebar"
      tryItCredentialsPolicy="include"
    ></elements-api>
    <link rel="stylesheet" href="https://unpkg.com/@stoplight/elements@8.3.0/styles.min.css" />
    <script src="https://unpkg.com/@stoplight/elements@8.3.0/web-components.min.js"></script>`
}
