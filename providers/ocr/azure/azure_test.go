package azure

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	p := New("https://example.cognitiveservices.azure.com/", "key")
	assert.Equal(t, "azure", p.Name())
}

func TestProvider_NewFromEnv(t *testing.T) {
	p := NewFromEnv()
	assert.NotNil(t, p)
	assert.Equal(t, "azure", p.Name())
}

func TestProvider_New_PreservesFields(t *testing.T) {
	p := New("https://my-cv.cognitiveservices.azure.com/", "mykey")
	assert.Equal(t, "https://my-cv.cognitiveservices.azure.com/", p.Endpoint)
	assert.Equal(t, "mykey", p.SubscriptionKey)
}

func TestProvider_Recognize_MissingEndpoint(t *testing.T) {
	p := New("", "key")
	_, err := p.Recognize(context.Background(), []byte("img"), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint not configured")
}

func TestProvider_Recognize_MissingKey(t *testing.T) {
	p := New("https://example.cognitiveservices.azure.com/", "")
	_, err := p.Recognize(context.Background(), []byte("img"), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subscription key not configured")
}

func TestProvider_APIVersion(t *testing.T) {
	p := New("ep", "key")
	assert.Equal(t, "4.0", p.apiVersion())
	p.APIVersion = "3.2"
	assert.Equal(t, "3.2", p.apiVersion())
}

func TestProvider_EndpointURL_EnvFallback(t *testing.T) {
	t.Setenv("AZURE_COMPUTER_VISION_ENDPOINT", "https://env-cv.cognitiveservices.azure.com/")
	p := New("", "key")
	u, err := p.endpointURL()
	require.NoError(t, err)
	assert.Equal(t, "https://env-cv.cognitiveservices.azure.com", u)
}

func TestProvider_EndpointURL_TrimsTrailingSlash(t *testing.T) {
	p := New("https://my-cv.cognitiveservices.azure.com/", "key")
	u, err := p.endpointURL()
	require.NoError(t, err)
	assert.Equal(t, "https://my-cv.cognitiveservices.azure.com", u)
}

func TestProvider_EndpointURL_Missing(t *testing.T) {
	p := New("", "key")
	_, err := p.endpointURL()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint not configured")
}

func TestProvider_APIKey_EnvFallback(t *testing.T) {
	t.Setenv("AZURE_COMPUTER_VISION_KEY", "envkey")
	p := New("https://my-cv.cognitiveservices.azure.com/", "")
	k, err := p.apiKey()
	require.NoError(t, err)
	assert.Equal(t, "envkey", k)
}

func TestProvider_APIKey_Missing(t *testing.T) {
	p := New("https://my-cv.cognitiveservices.azure.com/", "")
	_, err := p.apiKey()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subscription key not configured")
}

func TestProvider_Recognize_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := New("https://example.cognitiveservices.azure.com/", "key")
	_, err := p.Recognize(ctx, []byte("img"), nil)
	assert.Error(t, err)
}

func TestProvider_Recognize_CustomAPIVersion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := New("https://example.cognitiveservices.azure.com/", "key")
	p.APIVersion = "3.2"
	_, err := p.Recognize(ctx, []byte("img"), nil)
	assert.Error(t, err)
}
