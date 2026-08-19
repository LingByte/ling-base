// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package apidocs

import (
	"github.com/danielgtaylor/huma/v2"
)

// Theme is the docs UI theme.
type Theme string

const (
	// ThemeScalar uses Scalar API Reference (modern, light/dark, customizable).
	// Default theme. CDN: https://unpkg.com/@scalar/api-reference
	ThemeScalar Theme = "scalar"

	// ThemeSwagger uses Swagger UI (classic, widely recognized).
	// CDN: https://unpkg.com/swagger-ui-dist
	ThemeSwagger Theme = "swagger"

	// ThemeRedoc uses Redoc (clean, responsive, three-panel layout).
	// CDN: https://cdn.jsdelivr.net/npm/redoc
	ThemeRedoc Theme = "redoc"

	// ThemeStoplight uses Stoplight Elements (modern, good for design-first).
	// CDN: https://unpkg.com/@stoplight/elements
	ThemeStoplight Theme = "stoplight"
)

// Options configures the API docs surface.
type Options struct {
	// Title is the API title shown in the docs UI.
	// Default: "API"
	Title string

	// Version is the API version.
	// Default: "1.0.0"
	Version string

	// Description is the OpenAPI info description.
	// Supports Markdown.
	Description string

	// DocsPath is the path where the docs UI is served.
	// Default: "/docs"
	DocsPath string

	// APIPrefix is the API prefix for server URL display, e.g. "/api".
	// Optional. If set, the server URL shows the prefix.
	APIPrefix string

	// MetaPath is the path for the meta endpoint.
	// Default: "/api/v1/meta"
	// Set to "" to disable the meta endpoint.
	MetaPath string

	// EnableMeta controls whether the meta endpoint is registered.
	// Default: true
	EnableMeta *bool

	// Theme is the docs UI theme.
	// Default: ThemeScalar
	Theme Theme

	// DarkMode controls the docs UI dark mode.
	// Default: false (light mode)
	DarkMode bool

	// Logo is a PNG/JPG/SVG image served at <docsPath>/assets/logo.
	// If nil, a default logo is used.
	Logo []byte

	// LogoContentType is the MIME type for the logo, e.g. "image/png".
	// Default: "image/png"
	LogoContentType string

	// CSS is custom CSS injected into the docs page.
	// If empty, a default theme CSS is used.
	CSS string

	// SecuritySchemes registers OpenAPI security schemes.
	// Key is the scheme name (e.g. "BearerAuth"), value is the scheme config.
	SecuritySchemes map[string]SecurityScheme

	// Servers overrides the OpenAPI servers list.
	// If nil, a default server is generated from APIPrefix.
	Servers []*huma.Server

	// ScalarConfig is additional Scalar-specific configuration.
	// Only used when Theme == ThemeScalar.
	// See: https://github.com/scalar/scalar/blob/main/docs/configuration.md
	ScalarConfig map[string]any

	// CustomHeadHTML is injected into the <head> of the docs page.
	// Use for additional meta tags, preconnect, etc.
	CustomHeadHTML string
}

func (o *Options) applyDefaults() {
	if o.Title == "" {
		o.Title = "API"
	}
	if o.Version == "" {
		o.Version = "1.0.0"
	}
	if o.DocsPath == "" {
		o.DocsPath = "/docs"
	}
	if o.Theme == "" {
		o.Theme = ThemeScalar
	}
	if o.LogoContentType == "" {
		o.LogoContentType = "image/png"
	}
	if o.EnableMeta == nil {
		t := true
		o.EnableMeta = &t
	}
}
