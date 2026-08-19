// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package apidocs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
)

func TestMount_Default(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := Mount(r, Options{
		Title:   "Test API",
		Version: "1.0.0",
	})

	if api == nil {
		t.Fatal("Mount returned nil API")
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("openapi.json status=%d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("docs status=%d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "api-reference") {
		t.Fatal("docs should use Scalar by default")
	}
	if !strings.Contains(body, "Test API") {
		t.Fatal("docs should contain title")
	}
	if !strings.Contains(body, "apidocs-topbar") {
		t.Fatal("docs should have topbar")
	}
	if !strings.Contains(body, "导出 JSON") {
		t.Fatal("docs should have JSON export button")
	}
}

func TestMount_SwaggerTheme(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{Title: "Swagger Test", Theme: ThemeSwagger})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "swagger-ui") {
		t.Fatal("docs should use Swagger UI")
	}
	if !strings.Contains(body, "unpkg.com/swagger-ui-dist") {
		t.Fatal("Swagger should load from public CDN by default")
	}
}

func TestMount_RedocTheme(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{Title: "Redoc Test", Theme: ThemeRedoc})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "redoc") {
		t.Fatal("docs should use Redoc")
	}
}

func TestMount_StoplightTheme(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{Title: "Stoplight Test", Theme: ThemeStoplight})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "elements-api") {
		t.Fatal("docs should use Stoplight Elements")
	}
}

func TestMount_DarkMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{Title: "Dark Test", DarkMode: true})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "&#34;darkMode&#34;:true") {
		t.Fatal("docs should have darkMode=true in Scalar config")
	}
	if !strings.Contains(body, `color-scheme" content="dark light"`) {
		t.Fatal("docs should have dark color-scheme meta")
	}
}

func TestMount_CustomCSS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{Title: "Custom CSS", CSS: "body { background: red; }"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs/assets/docs.css", nil)
	r.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "background: red") {
		t.Fatal("custom CSS should be served")
	}
}

func TestMount_CustomLogo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title:           "Custom Logo",
		Logo:            []byte("fake-png-data"),
		LogoContentType: "image/png",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs/assets/logo", nil)
	r.ServeHTTP(w, req)
	if w.Body.String() != "fake-png-data" {
		t.Fatal("custom logo should be served")
	}
	if w.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("logo content-type=%s, want image/png", w.Header().Get("Content-Type"))
	}
}

func TestMount_MetaEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{Title: "Meta Test", Version: "2.0.0"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("meta status=%d", w.Code)
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["name"] != "Meta Test" {
		t.Fatalf("unexpected name: %v", body["name"])
	}
}

func TestMount_DisableMeta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	f := false
	Mount(r, Options{Title: "No Meta", EnableMeta: &f})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("meta should be disabled, got %d", w.Code)
	}
}

func TestMount_SecuritySchemes_Bearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title: "Security Test",
		SecuritySchemes: map[string]SecurityScheme{
			"BearerAuth": {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
			"ApiKey":     {Type: "apiKey", In: "header", Name: "X-API-Key"},
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	r.ServeHTTP(w, req)

	var spec map[string]any
	json.Unmarshal(w.Body.Bytes(), &spec)
	components := spec["components"].(map[string]any)
	schemes := components["securitySchemes"].(map[string]any)
	if _, ok := schemes["BearerAuth"]; !ok {
		t.Fatal("BearerAuth not found")
	}
	if _, ok := schemes["ApiKey"]; !ok {
		t.Fatal("ApiKey not found")
	}
}

func TestMount_SecuritySchemes_OAuth2(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title: "OAuth2 Test",
		SecuritySchemes: map[string]SecurityScheme{
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
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	r.ServeHTTP(w, req)

	var spec map[string]any
	json.Unmarshal(w.Body.Bytes(), &spec)
	components := spec["components"].(map[string]any)
	schemes := components["securitySchemes"].(map[string]any)
	oauth2Scheme, ok := schemes["OAuth2"].(map[string]any)
	if !ok {
		t.Fatal("OAuth2 scheme not found")
	}
	if oauth2Scheme["type"] != "oauth2" {
		t.Fatalf("unexpected type: %v", oauth2Scheme["type"])
	}
	flows, ok := oauth2Scheme["flows"].(map[string]any)
	if !ok {
		t.Fatal("OAuth2 flows not found")
	}
	if _, ok := flows["authorizationCode"]; !ok {
		t.Fatal("authorizationCode flow not found")
	}
}

func TestMount_GlobalSecurity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title:   "Global Security",
		Version: "1.0.0",
		SecuritySchemes: map[string]SecurityScheme{
			"BearerAuth": {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
		},
		GlobalSecurity: []map[string][]string{
			{"BearerAuth": {}},
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	r.ServeHTTP(w, req)

	var spec map[string]any
	json.Unmarshal(w.Body.Bytes(), &spec)
	security, ok := spec["security"].([]any)
	if !ok || len(security) == 0 {
		t.Fatal("global security not set in OpenAPI spec")
	}
}

func TestMount_CustomDocsPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{Title: "Custom Path", DocsPath: "/api/docs"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("custom docs path status=%d", w.Code)
	}
}

func TestMount_WithHumaOperation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := Mount(r, Options{Title: "Operation Test", Version: "1.0.0"})

	type helloOutput struct {
		Body struct {
			Msg string `json:"msg"`
		}
	}

	huma.Register(api, huma.Operation{
		OperationID: "hello",
		Method:      http.MethodGet,
		Path:        "/hello",
		Summary:     "Hello",
	}, func(ctx context.Context, _ *struct{}) (*helloOutput, error) {
		return &helloOutput{Body: struct {
			Msg string `json:"msg"`
		}{Msg: "hello world"}}, nil
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("hello status=%d", w.Code)
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["msg"] != "hello world" {
		t.Fatalf("unexpected msg: %v", body["msg"])
	}
}

func TestEnsureTag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := Mount(r, Options{Title: "Tag Test"})

	EnsureTag(api, "users", "用户管理")
	EnsureTag(api, "users", "用户管理") // duplicate

	found := false
	for _, tag := range api.OpenAPI().Tags {
		if tag.Name == "users" {
			found = true
		}
	}
	if !found {
		t.Fatal("tag not found")
	}
}

func TestMount_DefaultLogo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{Title: "Default Logo"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs/assets/logo", nil)
	r.ServeHTTP(w, req)
	if w.Header().Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("default logo content-type=%s", w.Header().Get("Content-Type"))
	}
}

// ──────────────────────────────────────────────
// CDN mode tests
// ──────────────────────────────────────────────

func TestCDN_PublicMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title: "CDN Public",
		CDN:   CDNConfig{Mode: CDNModePublic},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "unpkg.com/@scalar/api-reference") {
		t.Fatal("public CDN mode should load from unpkg")
	}
}

func TestCDN_SelfHostedMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title: "CDN SelfHosted",
		CDN: CDNConfig{
			Mode:    CDNModeSelfHosted,
			BaseURL: "https://assets.internal.company.com/openapi-ui",
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "assets.internal.company.com/openapi-ui/scalar/api-reference.js") {
		t.Fatalf("self-hosted mode should load from BaseURL, body: %s", body[:min(300, len(body))])
	}
	if strings.Contains(body, "unpkg.com/@scalar") {
		t.Fatal("self-hosted mode should NOT load from unpkg")
	}
}

func TestCDN_CustomMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title: "CDN Custom",
		CDN: CDNConfig{
			Mode:     CDNModeCustom,
			ScalarJS: "https://cdn.mycompany.com/scalar.js",
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "cdn.mycompany.com/scalar.js") {
		t.Fatal("custom CDN mode should use specified URL")
	}
}

func TestCDN_SelfHosted_Swagger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title: "CDN SelfHosted Swagger",
		Theme: ThemeSwagger,
		CDN: CDNConfig{
			Mode:    CDNModeSelfHosted,
			BaseURL: "https://assets.internal.company.com/ui",
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "assets.internal.company.com/ui/swagger-ui/swagger-ui-bundle.js") {
		t.Fatal("self-hosted Swagger should load from BaseURL")
	}
	if !strings.Contains(body, "assets.internal.company.com/ui/swagger-ui/swagger-ui.css") {
		t.Fatal("self-hosted Swagger CSS should load from BaseURL")
	}
}

// ──────────────────────────────────────────────
// TopBar tests
// ──────────────────────────────────────────────

func TestTopBar_HideTopBar(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title:  "No TopBar",
		TopBar: TopBarConfig{HideTopBar: true},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if strings.Contains(body, "apidocs-topbar") {
		t.Fatal("topbar should be hidden")
	}
}

func TestTopBar_CustomHTML(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title: "Custom TopBar",
		TopBar: TopBarConfig{
			CustomHTML: `<header class="my-header"><h1>My Custom Header</h1></header>`,
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "my-header") {
		t.Fatal("custom topbar HTML should be rendered")
	}
	if !strings.Contains(body, "My Custom Header") {
		t.Fatal("custom topbar content should be rendered")
	}
	if strings.Contains(body, "apidocs-topbar") {
		t.Fatal("default topbar should not be rendered when CustomHTML is set")
	}
}

func TestTopBar_EnvBadge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title: "Env Test",
		TopBar: TopBarConfig{
			EnvLabel:      "STAGING",
			EnvLabelColor: "#ef4444",
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "apidocs-env-badge") {
		t.Fatal("env badge should be rendered")
	}
	if !strings.Contains(body, "STAGING") {
		t.Fatal("env label should be rendered")
	}
	if !strings.Contains(body, "#ef4444") {
		t.Fatal("env label color should be applied")
	}
}

func TestTopBar_ExtraButtons(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title: "Extra Buttons",
		TopBar: TopBarConfig{
			ExtraButtons: `<a class="apidocs-btn-ghost" href="/">返回首页</a>`,
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "返回首页") {
		t.Fatal("extra buttons should be rendered")
	}
}

func TestTopBar_CustomSubtitle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title:  "Subtitle Test",
		TopBar: TopBarConfig{Subtitle: "API Reference"},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "API Reference") {
		t.Fatal("custom subtitle should be rendered")
	}
	if strings.Contains(body, "接口文档") {
		t.Fatal("default subtitle should not be rendered when custom is set")
	}
}

func TestTopBar_HideExportButtons(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	f := false
	Mount(r, Options{
		Title:  "No Export",
		TopBar: TopBarConfig{ShowExportButtons: &f},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if strings.Contains(body, "导出 JSON") {
		t.Fatal("export buttons should be hidden")
	}
}

// ──────────────────────────────────────────────
// CustomJS test
// ──────────────────────────────────────────────

func TestCustomJS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title:    "Custom JS",
		CustomJS: "console.log('docs loaded');",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "console.log('docs loaded')") {
		t.Fatal("custom JS should be injected")
	}
}

// ──────────────────────────────────────────────
// Contact / License / TermsOfService
// ──────────────────────────────────────────────

func TestMount_ContactLicense(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title:          "Contact Test",
		Version:        "1.0.0",
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
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	r.ServeHTTP(w, req)

	var spec map[string]any
	json.Unmarshal(w.Body.Bytes(), &spec)
	info := spec["info"].(map[string]any)
	if info["termsOfService"] != "https://example.com/terms" {
		t.Fatalf("unexpected termsOfService: %v", info["termsOfService"])
	}
	contact, ok := info["contact"].(map[string]any)
	if !ok || contact["name"] != "API Support" {
		t.Fatalf("unexpected contact: %v", info["contact"])
	}
	license, ok := info["license"].(map[string]any)
	if !ok || license["name"] != "MIT" {
		t.Fatalf("unexpected license: %v", info["license"])
	}
}

// ──────────────────────────────────────────────
// Theme-specific config tests
// ──────────────────────────────────────────────

func TestSwaggerConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title: "Swagger Config",
		Theme: ThemeSwagger,
		SwaggerConfig: map[string]any{
			"docExpansion": "none",
			"filter":       true,
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "docExpansion") {
		t.Fatal("Swagger config should be injected")
	}
	if !strings.Contains(body, `"none"`) {
		t.Fatal("Swagger config value should be rendered")
	}
}

func TestScalarConfig_Merge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title: "Scalar Merge",
		ScalarConfig: map[string]any{
			"hideModels":  true, // override default false
			"customField": "test",
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	// User override should take effect.
	if !strings.Contains(body, "&#34;hideModels&#34;:true") {
		t.Fatal("Scalar config merge should override defaults")
	}
	if !strings.Contains(body, "customField") {
		t.Fatal("Scalar config should include custom fields")
	}
}

// ──────────────────────────────────────────────
// Benchmarks
// ──────────────────────────────────────────────

func BenchmarkMount(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	for i := 0; i < b.N; i++ {
		r := gin.New()
		Mount(r, Options{Title: "Bench", Version: "1.0.0"})
	}
}

func BenchmarkDocsRequest(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	Mount(r, Options{Title: "Bench", Version: "1.0.0"})

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}
