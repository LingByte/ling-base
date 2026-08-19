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

	// OpenAPI JSON should be served.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("openapi.json status=%d", w.Code)
	}

	// Docs UI should be served at /docs.
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
	Mount(r, Options{
		Title: "Swagger Test",
		Theme: ThemeSwagger,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("docs status=%d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "swagger-ui") {
		t.Fatal("docs should use Swagger UI")
	}
}

func TestMount_RedocTheme(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title: "Redoc Test",
		Theme: ThemeRedoc,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "redoc") {
		t.Fatal("docs should use Redoc")
	}
}

func TestMount_StoplightTheme(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title: "Stoplight Test",
		Theme: ThemeStoplight,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "elements-api") {
		t.Fatal("docs should use Stoplight Elements")
	}
}

func TestMount_DarkMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title:    "Dark Test",
		DarkMode: true,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	body := w.Body.String()
	// JSON in data-configuration is HTML-escaped, so " becomes &#34;
	if !strings.Contains(body, "darkMode") {
		t.Fatal("docs should have darkMode in Scalar config")
	}
	// Check the raw config value is true (escaped JSON).
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
	Mount(r, Options{
		Title: "Custom CSS",
		CSS:   "body { background: red; }",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs/assets/docs.css", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("css status=%d", w.Code)
	}
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
	if w.Code != http.StatusOK {
		t.Fatalf("logo status=%d", w.Code)
	}
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
	Mount(r, Options{
		Title:   "Meta Test",
		Version: "2.0.0",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("meta status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["name"] != "Meta Test" {
		t.Fatalf("unexpected name: %v", body["name"])
	}
	if body["version"] != "2.0.0" {
		t.Fatalf("unexpected version: %v", body["version"])
	}
	if body["docs_path"] != "/docs" {
		t.Fatalf("unexpected docs_path: %v", body["docs_path"])
	}
}

func TestMount_DisableMeta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	f := false
	Mount(r, Options{
		Title:      "No Meta",
		EnableMeta: &f,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("meta should be disabled, got status=%d", w.Code)
	}
}

func TestMount_SecuritySchemes(t *testing.T) {
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
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	components, ok := spec["components"].(map[string]any)
	if !ok {
		t.Fatal("no components in OpenAPI spec")
	}
	schemes, ok := components["securitySchemes"].(map[string]any)
	if !ok {
		t.Fatal("no securitySchemes in components")
	}
	if _, ok := schemes["BearerAuth"]; !ok {
		t.Fatal("BearerAuth scheme not found")
	}
	if _, ok := schemes["ApiKey"]; !ok {
		t.Fatal("ApiKey scheme not found")
	}
}

func TestMount_CustomDocsPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{
		Title:    "Custom Path",
		DocsPath: "/api/docs",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("custom docs path status=%d", w.Code)
	}

	// Default path should not work.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/docs", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("default docs path should 404, got %d", w.Code)
	}
}

func TestMount_WithHumaOperation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := Mount(r, Options{
		Title:   "Operation Test",
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
		Summary:     "Hello",
	}, func(ctx context.Context, _ *struct{}) (*helloOutput, error) {
		out := &helloOutput{}
		out.Body.Msg = "hello world"
		return out, nil
	})

	// Test the registered endpoint.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("hello status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["msg"] != "hello world" {
		t.Fatalf("unexpected msg: %v", body["msg"])
	}

	// Verify it appears in OpenAPI spec.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	r.ServeHTTP(w, req)
	var spec map[string]any
	json.Unmarshal(w.Body.Bytes(), &spec)
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("no paths in spec")
	}
	if _, ok := paths["/hello"]; !ok {
		t.Fatal("/hello not in OpenAPI paths")
	}
}

func TestEnsureTag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := Mount(r, Options{Title: "Tag Test"})

	EnsureTag(api, "users", "用户管理")
	EnsureTag(api, "users", "用户管理") // duplicate should be ignored

	found := false
	for _, tag := range api.OpenAPI().Tags {
		if tag.Name == "users" {
			found = true
			if tag.Description != "用户管理" {
				t.Fatalf("unexpected tag description: %s", tag.Description)
			}
		}
	}
	if !found {
		t.Fatal("tag 'users' not found")
	}
}

func TestMount_DefaultLogo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Mount(r, Options{Title: "Default Logo"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs/assets/logo", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("logo status=%d", w.Code)
	}
	if w.Body.Len() < 50 {
		t.Fatalf("default logo too small: %d bytes", w.Body.Len())
	}
	if w.Header().Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("default logo content-type=%s, want image/svg+xml", w.Header().Get("Content-Type"))
	}
}

// BenchmarkMount measures the overhead of mounting docs.
func BenchmarkMount(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	for i := 0; i < b.N; i++ {
		r := gin.New()
		Mount(r, Options{
			Title:   "Bench",
			Version: "1.0.0",
		})
	}
}

// BenchmarkDocsRequest measures the overhead of serving the docs page.
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
