package aliyun

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	p := New("ak", "sk", "endpoint")
	assert.Equal(t, "aliyun", p.Name())
}

func TestProvider_NewFromEnv(t *testing.T) {
	p := NewFromEnv()
	assert.NotNil(t, p)
	assert.Equal(t, "aliyun", p.Name())
}

func TestProvider_New_PreservesFields(t *testing.T) {
	p := New("myak", "mysk", "my-endpoint")
	assert.Equal(t, "myak", p.AccessKeyID)
	assert.Equal(t, "mysk", p.AccessKeySecret)
	assert.Equal(t, "my-endpoint", p.Endpoint)
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
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_ID", "envak")
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET", "envsk")
	p := New("", "", "")
	_, err := p.clientLazy()
	require.NoError(t, err)
}

func TestProvider_ClientLazy_DefaultEndpoint(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_ID", "envak")
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET", "envsk")
	p := New("ak", "sk", "")
	c, err := p.clientLazy()
	require.NoError(t, err)
	require.NotNil(t, c)
}

func TestProvider_ClientLazy_Cached(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_ID", "envak")
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET", "envsk")
	p := New("ak", "sk", "ep")
	c1, err := p.clientLazy()
	require.NoError(t, err)
	c2, err := p.clientLazy()
	require.NoError(t, err)
	assert.Same(t, c1, c2)
}

func TestProvider_Recognize_NilImageBytes(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_ID", "envak")
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET", "envsk")
	p := New("ak", "sk", "ocr-api.cn-hangzhou.aliyuncs.com")
	// Recognize will attempt a real API call; with fake creds it errors.
	_, err := p.Recognize(context.Background(), []byte("img"), nil)
	assert.Error(t, err)
}
