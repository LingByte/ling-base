// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package apidocs provides a reusable wrapper around Huma (OpenAPI 3.1)
// for quickly mounting API documentation on a Gin engine.
//
// Features:
//   - One-line mount: apidocs.Mount(r, apidocs.Options{...})
//   - Multiple UI themes: Scalar (default), Swagger UI, Redoc, Stoplight
//   - Custom CSS / logo / branding
//   - Security scheme helpers (Bearer, APIKey, OAuth2)
//   - Meta endpoint for doc discovery
//   - OpenAPI JSON/YAML export
//
// # Quick start
//
//	r := gin.New()
//	api := apidocs.Mount(r, apidocs.Options{
//	    Title:   "My API",
//	    Version: "1.0.0",
//	})
//	// Register routes with huma.Register(api, ...)
//	huma.Register(api, huma.Operation{
//	    Method: http.MethodGet,
//	    Path:   "/hello",
//	}, func(ctx context.Context, _ *struct{}) (*HelloOutput, error) {
//	    return &HelloOutput{Body: struct{ Msg string `json:"msg"` }{Msg: "hi"}}, nil
//	})
package apidocs

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

// Mount wires Huma OpenAPI 3.1 onto the Gin engine with a customizable
// docs UI. Returns the huma.API for registering typed operations.
//
// One line is enough:
//
//	api := apidocs.Mount(r, apidocs.Options{Title: "My API", Version: "1.0.0"})
//
// Custom theme + logo + security:
//
//	api := apidocs.Mount(r, apidocs.Options{
//	    Title:   "My API",
//	    Version: "1.0.0",
//	    Theme:   apidocs.ThemeSwagger,
//	    Logo:    myLogoPNG,        // []byte
//	    CSS:     myCustomCSS,      // string
//	    SecuritySchemes: map[string]apidocs.SecurityScheme{
//	        "BearerAuth": {Type: "http", Scheme: "bearer"},
//	    },
//	})
func Mount(r *gin.Engine, opts Options) huma.API {
	opts.applyDefaults()

	cfg := huma.DefaultConfig(opts.Title, opts.Version)
	// Disable built-in docs — we serve our own customizable UI.
	cfg.DocsPath = ""
	cfg.OpenAPIPath = "/openapi"
	cfg.SchemasPath = "/schemas"

	if opts.Description != "" {
		cfg.Info.Description = opts.Description
	}
	if len(opts.Servers) > 0 {
		cfg.Servers = opts.Servers
	} else if opts.APIPrefix != "" {
		cfg.Servers = []*huma.Server{{
			URL:         "/",
			Description: "API prefix: " + opts.APIPrefix,
		}}
	}

	api := humagin.New(r, cfg)

	// Register security schemes directly on the OpenAPI spec.
	if len(opts.SecuritySchemes) > 0 {
		if api.OpenAPI().Components == nil {
			api.OpenAPI().Components = &huma.Components{}
		}
		if api.OpenAPI().Components.SecuritySchemes == nil {
			api.OpenAPI().Components.SecuritySchemes = make(map[string]*huma.SecurityScheme)
		}
		for name, scheme := range opts.SecuritySchemes {
			api.OpenAPI().Components.SecuritySchemes[name] = scheme.toHuma()
		}
	}

	// Mount the docs UI.
	ui := newUIRenderer(opts)
	ui.mount(r)

	// Mount meta endpoint if enabled.
	if opts.EnableMeta != nil && *opts.EnableMeta {
		mountMeta(api, opts)
	}

	return api
}

// mountMeta registers a /api/v1/meta endpoint that returns doc metadata.
func mountMeta(api huma.API, opts Options) {
	type metaBody struct {
		Name        string `json:"name" example:"My API" doc:"API name"`
		Version     string `json:"version" example:"1.0.0" doc:"API version"`
		DocsPath    string `json:"docs_path" doc:"Interactive docs UI path"`
		OpenAPIJSON string `json:"openapi_json" example:"/openapi.json" doc:"OpenAPI 3.1 JSON"`
		OpenAPIYAML string `json:"openapi_yaml" example:"/openapi.yaml" doc:"OpenAPI 3.1 YAML"`
		Theme       string `json:"theme" doc:"UI theme"`
	}
	type metaOutput struct {
		Body metaBody
	}

	metaPath := opts.MetaPath
	if metaPath == "" {
		metaPath = "/api/v1/meta"
	}

	huma.Register(api, huma.Operation{
		OperationID:   "get-api-meta",
		Method:        http.MethodGet,
		Path:          metaPath,
		Summary:       "文档元数据",
		Description:   "返回 OpenAPI 导出地址与文档入口，便于集成方发现文档。",
		Tags:          []string{"元信息"},
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, _ *struct{}) (*metaOutput, error) {
		return &metaOutput{
			Body: metaBody{
				Name:        opts.Title,
				Version:     opts.Version,
				DocsPath:    opts.DocsPath,
				OpenAPIJSON: "/openapi.json",
				OpenAPIYAML: "/openapi.yaml",
				Theme:       string(opts.Theme),
			},
		}, nil
	})
}

// SecurityScheme defines an OpenAPI security scheme.
type SecurityScheme struct {
	// Type is the scheme type: "http", "apiKey", "oauth2", "openIdConnect".
	Type string

	// Scheme is the HTTP auth scheme: "bearer", "basic".
	Scheme string

	// In is where the API key is located: "header", "query", "cookie".
	In string

	// Name is the header/query parameter name for apiKey.
	Name string

	// BearerFormat is the format hint for bearer tokens, e.g. "JWT".
	BearerFormat string

	// Description is a human-readable description.
	Description string
}

func (s SecurityScheme) toHuma() *huma.SecurityScheme {
	return &huma.SecurityScheme{
		Type:         s.Type,
		Scheme:       s.Scheme,
		In:           s.In,
		Name:         s.Name,
		BearerFormat: s.BearerFormat,
		Description:  s.Description,
	}
}

// EnsureTag ensures a tag exists in the OpenAPI spec with a description.
func EnsureTag(api huma.API, name, description string) {
	if description == "" {
		description = name
	}
	// Check if tag already exists.
	for _, t := range api.OpenAPI().Tags {
		if t.Name == name {
			return
		}
	}
	api.OpenAPI().Tags = append(api.OpenAPI().Tags, &huma.Tag{
		Name:        name,
		Description: description,
	})
}

// suppress unused import
var _ = context.Background
var _ = strings.TrimSpace
