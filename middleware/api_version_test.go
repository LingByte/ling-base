package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAPIVersionMiddleware_DefaultV1(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(APIVersionMiddleware())
	router.GET("/", func(c *gin.Context) {
		v, ok := c.Get("api_version")
		assert.True(t, ok)
		assert.Equal(t, APIVersionV1, v)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, APIVersionV1, w.Header().Get(HeaderAPIVersion))
}

func TestAPIVersionMiddleware_ExplicitV1(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(APIVersionMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set(HeaderAPIVersion, "v1")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIVersionMiddleware_NormalizeV1(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(APIVersionMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set(HeaderAPIVersion, "1")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIVersionMiddleware_Unsupported(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(APIVersionMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set(HeaderAPIVersion, "v2")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, APIVersionV1, w.Header().Get(HeaderAPIVersion))
}

func TestDeprecationHeaders_WithSunset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(DeprecationHeaders("2027-01-01"))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "true", w.Header().Get("Deprecation"))
	assert.Equal(t, "2027-01-01", w.Header().Get("Sunset"))
}

func TestDeprecationHeaders_NoSunset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(DeprecationHeaders(""))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "true", w.Header().Get("Deprecation"))
	assert.Empty(t, w.Header().Get("Sunset"))
}
