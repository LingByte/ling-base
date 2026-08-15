package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMarkOperationLogged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	assert.False(t, OperationAlreadyLogged(c))
	MarkOperationLogged(c)
	assert.True(t, OperationAlreadyLogged(c))
}

func TestMarkOperationLogged_NilContext(t *testing.T) {
	MarkOperationLogged(nil)
	assert.False(t, OperationAlreadyLogged(nil))
}

func TestOperationAlreadyLogged_NotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	assert.False(t, OperationAlreadyLogged(c))
}
