package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDefaultSecurityConfig(t *testing.T) {
	cfg := DefaultSecurityConfig()
	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.CSRFSecret)
	assert.Equal(t, "csrf_token", cfg.CSRFTokenName)
	assert.True(t, cfg.XSSProtection)
	assert.True(t, cfg.ContentTypeNosniff)
	assert.Equal(t, "DENY", cfg.XFrameOptions)
	assert.True(t, cfg.MaxRequestSize > 0)
	assert.True(t, cfg.HSTSMaxAge > 0)
}

func TestSecurityMiddleware_DefaultConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityMiddleware(nil))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.NotEmpty(t, w.Header().Get("Strict-Transport-Security"))
	assert.NotEmpty(t, w.Header().Get("Referrer-Policy"))
}

func TestSecurityMiddleware_RequestTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &SecurityConfig{MaxRequestSize: 10}
	router := gin.New()
	router.Use(SecurityMiddleware(cfg))
	router.POST("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader("this is too long"))
	req.ContentLength = 18
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestSecurityMiddleware_OriginBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &SecurityConfig{AllowedOrigins: []string{"https://allowed.com"}}
	router := gin.New()
	router.Use(SecurityMiddleware(cfg))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSecurityMiddleware_OriginAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &SecurityConfig{AllowedOrigins: []string{"https://allowed.com"}}
	router := gin.New()
	router.Use(SecurityMiddleware(cfg))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://allowed.com")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSecurityMiddleware_OriginWildcard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &SecurityConfig{AllowedOrigins: []string{"https://example.*"}}
	router := gin.New()
	router.Use(SecurityMiddleware(cfg))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSecurityMiddleware_NoOriginHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &SecurityConfig{AllowedOrigins: []string{"https://allowed.com"}}
	router := gin.New()
	router.Use(SecurityMiddleware(cfg))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSecurityMiddleware_CustomHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &SecurityConfig{
		XSSProtection:      false,
		ContentTypeNosniff: false,
		XFrameOptions:      "SAMEORIGIN",
		HSTSMaxAge:         0,
		ReferrerPolicy:     "no-referrer",
	}
	router := gin.New()
	router.Use(SecurityMiddleware(cfg))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("X-XSS-Protection"))
	assert.Empty(t, w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "SAMEORIGIN", w.Header().Get("X-Frame-Options"))
	assert.Empty(t, w.Header().Get("Strict-Transport-Security"))
	assert.Equal(t, "no-referrer", w.Header().Get("Referrer-Policy"))
}

func TestXSSProtectionMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(XSSProtectionMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?q=<script>alert(1)</script>", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInputValidationMiddleware_CleanInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(InputValidationMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?q=hello", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInputValidationMiddleware_SQLInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(InputValidationMiddleware())
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/?q=1' OR 1=1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInputValidationMiddleware_ScriptTag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(InputValidationMiddleware())
	router.POST("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	body := "data=<script>alert(1)</script>"
	req, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSecureCompare(t *testing.T) {
	assert.True(t, SecureCompare("abc", "abc"))
	assert.False(t, SecureCompare("abc", "abd"))
	assert.False(t, SecureCompare("abc", "abcd"))
	assert.False(t, SecureCompare("", "a"))
	assert.True(t, SecureCompare("", ""))
}

func TestSanitizeString(t *testing.T) {
	assert.Equal(t, "hello", SanitizeString("hello"))
	assert.Equal(t, "alert", SanitizeString("<script>alert</script>"))
	assert.NotContains(t, SanitizeString("<img src=x>"), "<")
}

func TestValidateEmail(t *testing.T) {
	assert.True(t, ValidateEmail("user@example.com"))
	assert.True(t, ValidateEmail("user.name+tag@sub.example.co"))
	assert.False(t, ValidateEmail("invalid"))
	assert.False(t, ValidateEmail("user@"))
	assert.False(t, ValidateEmail("@example.com"))
}

func TestValidatePassword_Valid(t *testing.T) {
	assert.NoError(t, ValidatePassword("Abc123!@#"))
}

func TestValidatePassword_TooShort(t *testing.T) {
	err := ValidatePassword("Ab1!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "8 characters")
}

func TestValidatePassword_NoUpper(t *testing.T) {
	err := ValidatePassword("abc123!@#")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "uppercase")
}

func TestValidatePassword_NoLower(t *testing.T) {
	err := ValidatePassword("ABC123!@#")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lowercase")
}

func TestValidatePassword_NoNumber(t *testing.T) {
	err := ValidatePassword("Abcdef!@#")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "number")
}

func TestValidatePassword_NoSpecial(t *testing.T) {
	err := ValidatePassword("Abc12345")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "special character")
}

func TestGenerateRandomKey(t *testing.T) {
	key1 := generateRandomKey(32)
	key2 := generateRandomKey(32)
	assert.NotEmpty(t, key1)
	assert.NotEmpty(t, key2)
	assert.NotEqual(t, key1, key2)
}

func TestCSRFMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := DefaultSecurityConfig()
	cfg.CSRFSecure = false // allow HTTP in tests

	router := gin.New()
	router.Use(CSRFMiddleware(cfg))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	// CSRF middleware sets X-CSRF-Token header
	assert.NotEmpty(t, w.Header().Get("X-CSRF-Token"))
}

func TestXSSProtectionMiddleware_PostForm(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(XSSProtectionMiddleware())
	router.POST("/", func(c *gin.Context) {
		_ = c.Request.ParseForm()
		c.Status(http.StatusOK)
	})

	body := "name=<script>alert(1)</script>"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInputValidationMiddleware_PostFormClean(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(InputValidationMiddleware())
	router.POST("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	body := "name=hello&email=test@example.com"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInputValidationMiddleware_PostFormInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(InputValidationMiddleware())
	router.POST("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	body := "q=UNION SELECT * FROM users"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
