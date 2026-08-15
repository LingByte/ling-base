package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LingByte/ling-base/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRequestIDMiddleware_WithHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/", func(c *gin.Context) {
		id := ReqIDFromGin(c)
		assert.Equal(t, "req-from-header", id)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set(logger.HeaderXReqID, "req-from-header")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "req-from-header", w.Header().Get(logger.HeaderXReqID))
}

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/", func(c *gin.Context) {
		id := ReqIDFromGin(c)
		assert.NotEmpty(t, id)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	generated := w.Header().Get(logger.HeaderXReqID)
	assert.NotEmpty(t, generated)
}

func TestRequestIDMiddleware_WhitespaceHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/", func(c *gin.Context) {
		id := ReqIDFromGin(c)
		assert.NotEmpty(t, id)
		// Should have generated a new one since header was whitespace
		assert.NotEqual(t, "   ", id)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set(logger.HeaderXReqID, "   ")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReqIDFromGin_NilContext(t *testing.T) {
	assert.Equal(t, "", ReqIDFromGin(nil))
}
