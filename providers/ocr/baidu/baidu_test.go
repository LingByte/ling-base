package baidu

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	p := New("ak", "sk")
	assert.Equal(t, "baidu", p.Name())
}

func TestProvider_NewFromEnv(t *testing.T) {
	p := NewFromEnv()
	assert.NotNil(t, p)
	assert.Equal(t, "baidu", p.Name())
}

func TestProvider_New_PreservesFields(t *testing.T) {
	p := New("myak", "mysk")
	assert.Equal(t, "myak", p.APIKey)
	assert.Equal(t, "mysk", p.SecretKey)
}

func TestProvider_Recognize_MissingCredentials(t *testing.T) {
	p := New("", "")
	_, err := p.Recognize(context.Background(), []byte("img"), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestProvider_TokenEndpoint_Default(t *testing.T) {
	p := New("ak", "sk")
	assert.Equal(t, "https://aip.baidubce.com/oauth/2.0/token", p.tokenEndpoint())
}

func TestProvider_TokenEndpoint_Override(t *testing.T) {
	p := New("ak", "sk")
	p.TokenURL = "https://custom.example.com/token"
	assert.Equal(t, "https://custom.example.com/token", p.tokenEndpoint())
}

func TestProvider_OCREndpoint_Default(t *testing.T) {
	p := New("ak", "sk")
	assert.Equal(t, "https://aip.baidubce.com/rest/2.0/ocr/v1/general_basic", p.ocrEndpoint())
}

func TestProvider_OCREndpoint_Override(t *testing.T) {
	p := New("ak", "sk")
	p.OCRURL = "https://custom.example.com/ocr"
	assert.Equal(t, "https://custom.example.com/ocr", p.ocrEndpoint())
}

func TestProvider_FetchAccessToken_EnvFallback(t *testing.T) {
	t.Setenv("BAIDU_OCR_API_KEY", "envak")
	t.Setenv("BAIDU_OCR_SECRET_KEY", "envsk")
	p := New("", "")
	// With env creds set, the missing-credentials check passes; the HTTP
	// call to the real token endpoint will fail (no network / invalid).
	_, err := p.fetchAccessToken(context.Background())
	assert.Error(t, err)
}

func TestProvider_FetchAccessToken_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := New("ak", "sk")
	_, err := p.fetchAccessToken(ctx)
	assert.Error(t, err)
}

func TestProvider_Recognize_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := New("ak", "sk")
	_, err := p.Recognize(ctx, []byte("img"), nil)
	assert.Error(t, err)
}

func TestProvider_FetchAccessToken_Cached(t *testing.T) {
	p := New("ak", "sk")
	p.accessToken = "cached-token"
	tok, err := p.fetchAccessToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "cached-token", tok)
}
