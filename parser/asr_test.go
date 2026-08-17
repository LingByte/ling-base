package parser

import (
	"context"
	"fmt"
	"testing"
	"time"

	base "github.com/LingByte/ling-base/recognizer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEngine is a minimal recognizer.Engine for testing ASRParser.
type mockEngine struct {
	audio   []byte
	result  string
	err     error
	started bool
}

func (m *mockEngine) Init(tr base.ResultFunc, er base.ErrorFunc) {
	if m.err != nil {
		er(m.err, true)
		return
	}
	// Simulate immediate final result
	tr(m.result, true, 0, "test-dialog")
}

func (m *mockEngine) Vendor() string                       { return "mock" }
func (m *mockEngine) ConnAndReceive(dialogID string) error { m.started = true; return nil }
func (m *mockEngine) Activity() bool                       { return m.started }
func (m *mockEngine) RestartClient()                       {}
func (m *mockEngine) SendAudioBytes(data []byte) error {
	m.audio = append(m.audio, data...)
	return nil
}
func (m *mockEngine) SendEnd() error  { return nil }
func (m *mockEngine) StopConn() error { return nil }

func TestASRParser_NoEngine(t *testing.T) {
	p := &ASRParser{}
	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeWAV,
		FileName: "clip.wav",
		Content:  []byte("RIFF...."),
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no recognizer engine")
}

func TestASRParser_WithMockEngine(t *testing.T) {
	engine := &mockEngine{result: "hello world"}
	p := NewASRParser(engine)

	samples := make([]int16, 16000) // 1s silence
	wav := buildTestWAV(t, samples, 16000)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := p.Parse(ctx, &ParseRequest{
		FileType: FileTypeWAV,
		FileName: "silence.wav",
		Content:  wav,
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)
	assert.Equal(t, FileTypeWAV, res.FileType)
	assert.Equal(t, "hello world", res.Text)
}

func TestASRParser_MockEngineError(t *testing.T) {
	engine := &mockEngine{err: fmt.Errorf("connection failed")}
	p := NewASRParser(engine)

	samples := make([]int16, 1600)
	wav := buildTestWAV(t, samples, 16000)

	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeWAV,
		FileName: "clip.wav",
		Content:  wav,
	}, nil)
	require.Error(t, err)
}

func TestASRParser_SetEngine(t *testing.T) {
	p := &ASRParser{}
	assert.Nil(t, p.engine)

	engine := &mockEngine{result: "test"}
	p.SetEngine(engine)
	assert.NotNil(t, p.engine)
}

func TestASRParser_SupportedTypes(t *testing.T) {
	p := &ASRParser{}
	types := p.SupportedTypes()
	assert.Contains(t, types, FileTypeWAV)
	assert.Contains(t, types, FileTypeMP3)
}

func TestASRParser_Provider(t *testing.T) {
	p := &ASRParser{}
	assert.Equal(t, "asr", p.Provider())
}
