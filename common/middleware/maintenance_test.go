package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setEnv(t *testing.T, key, val string) {
	t.Helper()
	old, ok := os.LookupEnv(key)
	os.Setenv(key, val)
	t.Cleanup(func() {
		if ok {
			os.Setenv(key, old)
		} else {
			os.Unsetenv(key)
		}
	})
}

func TestMaintenanceMode_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setEnv(t, EnvMaintenanceMode, "false")

	router := gin.New()
	router.Use(MaintenanceMode(nil))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMaintenanceMode_Enabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setEnv(t, EnvMaintenanceMode, "true")

	router := gin.New()
	router.Use(MaintenanceMode(nil))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestMaintenanceMode_EnabledWithAllowedPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setEnv(t, EnvMaintenanceMode, "true")
	SetMaintenanceAllowedPaths([]string{"/health"}, []string{"/api/docs"})
	defer SetMaintenanceAllowedPaths(nil, nil)

	router := gin.New()
	router.Use(MaintenanceMode(nil))
	router.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMaintenanceMode_EnabledWithAllowedPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setEnv(t, EnvMaintenanceMode, "true")
	SetMaintenanceAllowedPaths([]string{"/health"}, []string{"/api/docs"})
	defer SetMaintenanceAllowedPaths(nil, nil)

	router := gin.New()
	router.Use(MaintenanceMode(nil))
	router.GET("/api/docs/swagger", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/docs/swagger", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMaintenanceMode_Bypass(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setEnv(t, EnvMaintenanceMode, "true")

	bypass := func(c *gin.Context) bool {
		return c.GetHeader("X-Admin") == "1"
	}
	router := gin.New()
	router.Use(MaintenanceMode(bypass))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	// With bypass header
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("X-Admin", "1")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Without bypass header
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusServiceUnavailable, w2.Code)
}

func TestMaintenanceMode_CustomMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setEnv(t, EnvMaintenanceMode, "true")
	setEnv(t, EnvMaintenanceMessage, "Custom maintenance message")

	router := gin.New()
	router.Use(MaintenanceMode(nil))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "Custom maintenance message")
}

func TestMaintenanceEnabled(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"1", true},
		{"true", true},
		{"yes", true},
		{"on", true},
		{"TRUE", true},
		{"0", false},
		{"false", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			setEnv(t, EnvMaintenanceMode, tt.val)
			assert.Equal(t, tt.want, maintenanceEnabled())
		})
	}
}

func TestMaintenancePathAllowed(t *testing.T) {
	SetMaintenanceAllowedPaths([]string{"/health"}, []string{"/api/docs"})
	defer SetMaintenanceAllowedPaths(nil, nil)

	assert.True(t, maintenancePathAllowed("/health"))
	assert.True(t, maintenancePathAllowed("/api/docs/swagger"))
	assert.False(t, maintenancePathAllowed("/api/users"))
}

func TestMaintenancePathAllowed_EmptyConfig(t *testing.T) {
	SetMaintenanceAllowedPaths(nil, nil)
	assert.False(t, maintenancePathAllowed("/health"))
	assert.False(t, maintenancePathAllowed("/anything"))
}
