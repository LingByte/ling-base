package aws

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	p := New("ak", "sk", "us-east-1")
	assert.Equal(t, "aws", p.Name())
}

func TestProvider_NewFromEnv(t *testing.T) {
	p := NewFromEnv()
	assert.NotNil(t, p)
	assert.Equal(t, "aws", p.Name())
}

func TestProvider_New_PreservesFields(t *testing.T) {
	p := New("myak", "mysk", "us-west-2")
	assert.Equal(t, "myak", p.AccessKeyID)
	assert.Equal(t, "mysk", p.SecretAccessKey)
	assert.Equal(t, "us-west-2", p.Region)
}

func TestProvider_Recognize_InvalidCredentials(t *testing.T) {
	// With invalid static credentials, the client creation should succeed
	// but the actual API call will fail. We only test that Recognize
	// doesn't panic with empty bytes.
	p := New("invalid", "invalid", "us-east-1")
	_, err := p.Recognize(context.Background(), []byte("img"), nil)
	assert.Error(t, err)
}

func TestProvider_ClientLazy_DefaultRegion(t *testing.T) {
	p := New("ak", "sk", "")
	c, err := p.clientLazy(context.Background())
	require.NoError(t, err)
	require.NotNil(t, c)
}

func TestProvider_ClientLazy_EnvRegion(t *testing.T) {
	t.Setenv("AWS_REGION", "eu-west-1")
	p := New("ak", "sk", "")
	c, err := p.clientLazy(context.Background())
	require.NoError(t, err)
	require.NotNil(t, c)
}

func TestProvider_ClientLazy_DefaultCredentialChain(t *testing.T) {
	p := New("", "", "us-east-1")
	c, err := p.clientLazy(context.Background())
	require.NoError(t, err)
	require.NotNil(t, c)
}

func TestProvider_ClientLazy_Cached(t *testing.T) {
	p := New("ak", "sk", "us-east-1")
	c1, err := p.clientLazy(context.Background())
	require.NoError(t, err)
	c2, err := p.clientLazy(context.Background())
	require.NoError(t, err)
	assert.Same(t, c1, c2)
}

func TestProvider_Recognize_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := New("ak", "sk", "us-east-1")
	_, err := p.Recognize(ctx, []byte("img"), nil)
	assert.Error(t, err)
}

func TestProvider_Recognize_EmptyBytes(t *testing.T) {
	p := New("ak", "sk", "us-east-1")
	_, err := p.Recognize(context.Background(), []byte{}, nil)
	assert.Error(t, err)
}
