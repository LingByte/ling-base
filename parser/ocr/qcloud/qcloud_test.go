package qcloud

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	p := New("id", "key", "ap-guangzhou")
	assert.Equal(t, "qcloud", p.Name())
}

func TestProvider_NewFromEnv(t *testing.T) {
	p := NewFromEnv()
	assert.NotNil(t, p)
	assert.Equal(t, "qcloud", p.Name())
}

func TestProvider_New_PreservesFields(t *testing.T) {
	p := New("myid", "mykey", "ap-beijing")
	assert.Equal(t, "myid", p.SecretID)
	assert.Equal(t, "mykey", p.SecretKey)
	assert.Equal(t, "ap-beijing", p.Region)
}

func TestProvider_NewFromEnv_DefaultRegion(t *testing.T) {
	os.Unsetenv("TENCENTCLOUD_REGION")
	p := NewFromEnv()
	assert.Equal(t, "ap-guangzhou", p.Region)
}

func TestProvider_NewFromEnv_EnvRegion(t *testing.T) {
	t.Setenv("TENCENTCLOUD_REGION", "ap-beijing")
	p := NewFromEnv()
	assert.Equal(t, "ap-beijing", p.Region)
}

func TestProvider_ClientLazy_MissingCredentials(t *testing.T) {
	p := New("", "", "")
	_, err := p.clientLazy()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestProvider_Recognize_MissingCredentials(t *testing.T) {
	p := New("", "", "")
	_, err := p.Recognize(context.Background(), []byte("img"), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestProvider_ClientLazy_EnvFallback(t *testing.T) {
	t.Setenv("TENCENTCLOUD_SECRET_ID", "envid")
	t.Setenv("TENCENTCLOUD_SECRET_KEY", "envkey")
	p := New("", "", "")
	_, err := p.clientLazy()
	require.NoError(t, err)
}

func TestProvider_ClientLazy_DefaultRegion(t *testing.T) {
	t.Setenv("TENCENTCLOUD_SECRET_ID", "envid")
	t.Setenv("TENCENTCLOUD_SECRET_KEY", "envkey")
	p := New("id", "key", "")
	c, err := p.clientLazy()
	require.NoError(t, err)
	require.NotNil(t, c)
}

func TestProvider_ClientLazy_Cached(t *testing.T) {
	t.Setenv("TENCENTCLOUD_SECRET_ID", "envid")
	t.Setenv("TENCENTCLOUD_SECRET_KEY", "envkey")
	p := New("id", "key", "ap-guangzhou")
	c1, err := p.clientLazy()
	require.NoError(t, err)
	c2, err := p.clientLazy()
	require.NoError(t, err)
	assert.Same(t, c1, c2)
}

func TestProvider_Recognize_WithClient(t *testing.T) {
	t.Setenv("TENCENTCLOUD_SECRET_ID", "envid")
	t.Setenv("TENCENTCLOUD_SECRET_KEY", "envkey")
	p := New("id", "key", "ap-guangzhou")
	// Recognize will attempt a real API call; with fake creds it errors.
	_, err := p.Recognize(context.Background(), []byte("img"), nil)
	assert.Error(t, err)
}
