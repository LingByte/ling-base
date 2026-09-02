// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package curlutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── ParseCurlCommand tests ─────────────────────────────────────

func TestParseCurlCommand_Simple(t *testing.T) {
	req, err := ParseCurlCommand("curl https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", req.URL)
	assert.Equal(t, "GET", req.Method)
	assert.True(t, req.VerifySSL)
}

func TestParseCurlCommand_POST(t *testing.T) {
	req, err := ParseCurlCommand(`curl -X POST https://httpbin.org/post -H "Content-Type: application/json" -d '{"key":"value"}'`)
	require.NoError(t, err)
	assert.Equal(t, "https://httpbin.org/post", req.URL)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, `{"key":"value"}`, req.Body)
	assert.Equal(t, "application/json", req.Headers["Content-Type"])
}

func TestParseCurlCommand_HeadOnly(t *testing.T) {
	req, err := ParseCurlCommand("curl -I https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "HEAD", req.Method)
	assert.True(t, req.HeadOnly)
}

func TestParseCurlCommand_Insecure(t *testing.T) {
	req, err := ParseCurlCommand("curl -k https://example.com")
	require.NoError(t, err)
	assert.False(t, req.VerifySSL)
}

func TestParseCurlCommand_FollowRedirect(t *testing.T) {
	req, err := ParseCurlCommand("curl -L https://example.com")
	require.NoError(t, err)
	assert.True(t, req.FollowRedirect)
}

func TestParseCurlCommand_UserAgent(t *testing.T) {
	req, err := ParseCurlCommand(`curl -A "MyAgent/1.0" https://example.com`)
	require.NoError(t, err)
	assert.Equal(t, "MyAgent/1.0", req.Headers["User-Agent"])
}

func TestParseCurlCommand_Referer(t *testing.T) {
	req, err := ParseCurlCommand(`curl -e "https://referrer.com" https://example.com`)
	require.NoError(t, err)
	assert.Equal(t, "https://referrer.com", req.Headers["Referer"])
}

func TestParseCurlCommand_BasicAuth(t *testing.T) {
	req, err := ParseCurlCommand(`curl -u "user:pass" https://example.com`)
	require.NoError(t, err)
	assert.Contains(t, req.Headers["Authorization"], "Basic ")
}

func TestParseCurlCommand_DataRaw(t *testing.T) {
	req, err := ParseCurlCommand(`curl --data-raw '{"a":1}' https://example.com`)
	require.NoError(t, err)
	assert.Equal(t, `{"a":1}`, req.Body)
	assert.Equal(t, "POST", req.Method)
}

func TestParseCurlCommand_MultipleHeaders(t *testing.T) {
	req, err := ParseCurlCommand(`curl -H "X-Custom: val1" -H "X-Other: val2" https://example.com`)
	require.NoError(t, err)
	assert.Equal(t, "val1", req.Headers["X-Custom"])
	assert.Equal(t, "val2", req.Headers["X-Other"])
}

func TestParseCurlCommand_Compressed(t *testing.T) {
	req, err := ParseCurlCommand("curl --compressed https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "gzip, deflate", req.Headers["Accept-Encoding"])
}

func TestParseCurlCommand_MaxTime(t *testing.T) {
	req, err := ParseCurlCommand("curl --max-time 10 https://example.com")
	require.NoError(t, err)
	assert.Equal(t, 10, req.Timeout)
}

func TestParseCurlCommand_NoURL(t *testing.T) {
	_, err := ParseCurlCommand("curl -H 'X-Test: 1'")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no URL")
}

func TestParseCurlCommand_Empty(t *testing.T) {
	_, err := ParseCurlCommand("")
	require.Error(t, err)
}

func TestParseCurlCommand_SingleQuotes(t *testing.T) {
	req, err := ParseCurlCommand(`curl -d 'hello world' https://example.com`)
	require.NoError(t, err)
	assert.Equal(t, "hello world", req.Body)
}

func TestParseCurlCommand_EscapedQuote(t *testing.T) {
	req, err := ParseCurlCommand(`curl -H "X-Test: \"quoted\"" https://example.com`)
	require.NoError(t, err)
	assert.Contains(t, req.Headers["X-Test"], "quoted")
}

func TestParseCurlCommand_UnterminatedQuote(t *testing.T) {
	_, err := ParseCurlCommand(`curl -H "unterminated https://example.com`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unterminated")
}

// ─── Tokenizer tests ────────────────────────────────────────────

func TestTokenizeCurl_Simple(t *testing.T) {
	tokens, err := tokenizeCurl(`-X POST https://example.com`)
	require.NoError(t, err)
	assert.Equal(t, []string{"-X", "POST", "https://example.com"}, tokens)
}

func TestTokenizeCurl_DoubleQuotes(t *testing.T) {
	tokens, err := tokenizeCurl(`-H "Content-Type: application/json" https://example.com`)
	require.NoError(t, err)
	assert.Equal(t, []string{"-H", "Content-Type: application/json", "https://example.com"}, tokens)
}

func TestTokenizeCurl_SingleQuotes(t *testing.T) {
	tokens, err := tokenizeCurl(`-d '{"key": "value"}' https://example.com`)
	require.NoError(t, err)
	assert.Equal(t, []string{"-d", `{"key": "value"}`, "https://example.com"}, tokens)
}

func TestTokenizeCurl_EscapedChar(t *testing.T) {
	tokens, err := tokenizeCurl(`-H "X-Test: \"hello\"" https://example.com`)
	require.NoError(t, err)
	assert.Equal(t, []string{"-H", `X-Test: "hello"`, "https://example.com"}, tokens)
}

// ─── Helper function tests ──────────────────────────────────────

func TestIsBinaryContent(t *testing.T) {
	assert.True(t, isBinaryContent("image/png"))
	assert.True(t, isBinaryContent("application/pdf"))
	assert.True(t, isBinaryContent("application/zip"))
	assert.True(t, isBinaryContent("audio/mpeg"))
	assert.True(t, isBinaryContent("video/mp4"))
	assert.False(t, isBinaryContent("text/html"))
	assert.False(t, isBinaryContent("application/json"))
	assert.False(t, isBinaryContent(""))
}

func TestTLSVersionString(t *testing.T) {
	// Use the crypto/tls package constants
	assert.Equal(t, "TLS 1.0", tlsVersionString(0x0301))
	assert.Equal(t, "TLS 1.1", tlsVersionString(0x0302))
	assert.Equal(t, "TLS 1.2", tlsVersionString(0x0303))
	assert.Equal(t, "TLS 1.3", tlsVersionString(0x0304))
	assert.Contains(t, tlsVersionString(0x0999), "0x0999")
}

func TestGenerateBinaryPreview(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	preview := generateBinaryPreview(data, "image/png")
	assert.Contains(t, preview, "Content-Type: image/png")
	assert.Contains(t, preview, "Size: 8 bytes")
	assert.Contains(t, preview, "[Image file]")
	assert.Contains(t, preview, "Hex preview")
	assert.Contains(t, preview, "8950 4e47")
}

func TestBase64Encode(t *testing.T) {
	assert.Equal(t, "dXNlcjpwYXNz", base64Encode("user:pass"))
}

func TestParseInt(t *testing.T) {
	n, err := parseInt("30")
	require.NoError(t, err)
	assert.Equal(t, 30, n)

	_, err = parseInt("abc")
	assert.Error(t, err)
}

// ─── Integration tests (require network) ────────────────────────

func TestExecute_Get(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test")
	}
	resp, err := Get("https://httpbin.org/get")
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.False(t, resp.IsBinary)
	assert.Contains(t, resp.Body, "httpbin")
}

func TestExecute_Post(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test")
	}
	resp, err := Post("https://httpbin.org/post", `{"test":"value"}`)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, resp.Body, "test")
}

func TestExecute_Head(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test")
	}
	resp, err := Head("https://example.com")
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, resp.Body, "HEAD")
}

func TestExecute_RedirectChain(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test")
	}
	resp, err := Execute(&Request{
		URL:            "http://httpbin.org/redirect/2",
		Method:         "GET",
		FollowRedirect: true,
		VerifySSL:      true,
		Timeout:        30,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.RedirectChain)
}

func TestExecute_NoRedirect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test")
	}
	resp, err := Execute(&Request{
		URL:            "http://httpbin.org/redirect/1",
		Method:         "GET",
		FollowRedirect: false,
		VerifySSL:      true,
		Timeout:        30,
	})
	require.NoError(t, err)
	assert.Equal(t, 302, resp.StatusCode)
}

func TestExecute_NilRequest(t *testing.T) {
	_, err := Execute(nil)
	require.Error(t, err)
}

func TestExecute_EmptyURL(t *testing.T) {
	_, err := Execute(&Request{URL: ""})
	require.Error(t, err)
}

func TestParseAndExecute(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test")
	}
	req, err := ParseCurlCommand(`curl -L https://httpbin.org/get -H "X-Test: hello"`)
	require.NoError(t, err)
	resp, err := Execute(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, resp.Body, "X-Test")
}
