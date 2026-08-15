package google

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	p := New("/path/to/creds.json")
	assert.Equal(t, "google", p.Name())
}

func TestProvider_NewFromEnv(t *testing.T) {
	p := NewFromEnv()
	assert.NotNil(t, p)
	assert.Equal(t, "google", p.Name())
}

func TestProvider_New_PreservesFields(t *testing.T) {
	p := New("/custom/creds.json")
	assert.Equal(t, "/custom/creds.json", p.CredentialsFile)
}

func TestProvider_NewFromEnv_UsesEnv(t *testing.T) {
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/env/creds.json")
	p := NewFromEnv()
	assert.Equal(t, "/env/creds.json", p.CredentialsFile)
}

func TestProvider_ClientLazy_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := New("/path/to/creds.json")
	_, err := p.clientLazy(ctx)
	assert.Error(t, err)
}

func TestProvider_Recognize_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := New("/path/to/creds.json")
	_, err := p.Recognize(ctx, []byte("img"), nil)
	assert.Error(t, err)
}

func TestProvider_ClientLazy_NoCredentials(t *testing.T) {
	// Ensure no credentials file and no env var so the client creation
	// falls back to default credentials, which fails without ADC setup.
	os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
	p := New("")
	_, err := p.clientLazy(context.Background())
	assert.Error(t, err)
}

func TestProvider_ClientLazy_Cached(t *testing.T) {
	// When client creation fails the client is not cached; verify that a
	// second call still returns an error (not a panic).
	p := New("/path/to/creds.json")
	_, err := p.clientLazy(context.Background())
	require.Error(t, err)
	_, err = p.clientLazy(context.Background())
	assert.Error(t, err)
}
