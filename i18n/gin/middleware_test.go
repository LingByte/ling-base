// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package gin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LingByte/ling-base/i18n"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestMiddleware_AcceptLanguage(t *testing.T) {
	m := i18n.NewManager(&i18n.Config{
		SupportedLocales: []i18n.Locale{i18n.LocaleEn, i18n.LocaleZhCN},
		DefaultLocale:    i18n.LocaleEn,
	})

	r := gin.New()
	r.Use(Middleware(m))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"locale": GetLocale(c)})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "zh-CN")
}

func TestMiddleware_QueryParam(t *testing.T) {
	m := i18n.NewManager(&i18n.Config{
		SupportedLocales: []i18n.Locale{i18n.LocaleEn, i18n.LocaleZhCN},
		DefaultLocale:    i18n.LocaleEn,
	})

	r := gin.New()
	r.Use(Middleware(m))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"locale": GetLocale(c)})
	})

	req := httptest.NewRequest("GET", "/test?locale=zh-CN", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "zh-CN")
}

func TestMiddleware_Default(t *testing.T) {
	m := i18n.NewManager(&i18n.Config{
		SupportedLocales: []i18n.Locale{i18n.LocaleEn},
		DefaultLocale:    i18n.LocaleEn,
	})

	r := gin.New()
	r.Use(Middleware(m))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"locale": GetLocale(c)})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "en")
}

func TestT(t *testing.T) {
	m := i18n.NewManager(&i18n.Config{
		SupportedLocales: []i18n.Locale{i18n.LocaleEn, i18n.LocaleZhCN},
		DefaultLocale:    i18n.LocaleEn,
	})
	m.SetTranslation(i18n.LocaleEn, "test.key", "Test Value")
	m.SetTranslation(i18n.LocaleZhCN, "test.key", "测试值")

	r := gin.New()
	r.Use(Middleware(m))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"result": T(c, "test.key")})
	})

	// English
	req := httptest.NewRequest("GET", "/test?locale=en", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Body.String(), "Test Value")

	// Chinese
	req = httptest.NewRequest("GET", "/test?locale=zh-CN", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Body.String(), "测试值")
}

func TestResponseJSON(t *testing.T) {
	m := i18n.NewManager(&i18n.Config{
		SupportedLocales: []i18n.Locale{i18n.LocaleEn},
		DefaultLocale:    i18n.LocaleEn,
	})
	m.SetTranslation(i18n.LocaleEn, "success", "Success")

	r := gin.New()
	r.Use(Middleware(m))
	r.GET("/test", func(c *gin.Context) {
		ResponseJSON(c, http.StatusOK, "success", gin.H{"id": 1})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Success")
}

func TestErrorJSON(t *testing.T) {
	m := i18n.NewManager(&i18n.Config{
		SupportedLocales: []i18n.Locale{i18n.LocaleEn},
		DefaultLocale:    i18n.LocaleEn,
	})
	m.SetTranslation(i18n.LocaleEn, "error.not_found", "Not found")

	r := gin.New()
	r.Use(Middleware(m))
	r.GET("/test", func(c *gin.Context) {
		ErrorJSON(c, http.StatusNotFound, "error.not_found", assert.AnError)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "Not found")
}

func TestSetLocale(t *testing.T) {
	m := i18n.NewManager(nil)
	r := gin.New()
	r.Use(Middleware(m))
	r.GET("/test", func(c *gin.Context) {
		SetLocale(c, i18n.LocaleJaJP)
		c.JSON(http.StatusOK, gin.H{"locale": GetLocale(c)})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Contains(t, w.Body.String(), "ja-JP")
}

func TestGetManager_NoMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		if GetManager(c) != nil {
			t.Error("expected nil manager without middleware")
		}
		// T should return the key when no manager is set
		if v := T(c, "some.key"); v != "some.key" {
			t.Errorf("expected key fallback, got %q", v)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
