package realtime

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"strings"
	"testing"
)

func TestParseFrameTooShort(t *testing.T) {
	_, err := ParseFrame([]byte{0x11})
	if err == nil {
		t.Fatal("expected error for short frame")
	}
	if !strings.Contains(err.Error(), "too short") {
		t.Errorf("err = %v, want 'too short'", err)
	}
}

func TestParseFrameIncompleteHeader(t *testing.T) {
	// Header size claims 8 bytes (0x21 = version 2, header 8 bytes) but only 4 bytes.
	_, err := ParseFrame([]byte{0x21, 0x10, 0x10, 0x00})
	if err == nil {
		t.Fatal("expected error for incomplete header")
	}
}

func TestMarshalAndParseJSONEvent(t *testing.T) {
	payload := map[string]any{"hello": "world"}
	frame, err := MarshalJSONEvent(eventStartSession, "session-123", payload)
	if err != nil {
		t.Fatalf("MarshalJSONEvent: %v", err)
	}
	f, err := ParseFrame(frame)
	if err != nil {
		t.Fatalf("ParseFrame: %v", err)
	}
	if f.Event != eventStartSession {
		t.Errorf("Event = %d, want %d", f.Event, eventStartSession)
	}
	if f.SessionID != "session-123" {
		t.Errorf("SessionID = %s, want session-123", f.SessionID)
	}
	if !bytes.Contains(f.Payload, []byte("hello")) {
		t.Errorf("Payload = %s, want hello", string(f.Payload))
	}
}

func TestMarshalAndParseAudioTask(t *testing.T) {
	pcm := bytes.Repeat([]byte{0x01, 0x02}, 100)
	frame, err := MarshalAudioTask("session-abc", pcm)
	if err != nil {
		t.Fatalf("MarshalAudioTask: %v", err)
	}
	f, err := ParseFrame(frame)
	if err != nil {
		t.Fatalf("ParseFrame: %v", err)
	}
	if f.Event != eventTaskRequest {
		t.Errorf("Event = %d, want %d", f.Event, eventTaskRequest)
	}
	if !bytes.Equal(f.Payload, pcm) {
		t.Errorf("Payload len = %d, want %d (roundtrip via gzip)", len(f.Payload), len(pcm))
	}
}

func TestParseFrameErrorType(t *testing.T) {
	// Build an error frame manually.
	header := buildHeader(msgTypeError, 0, serializationJSON, compressionNone)
	buf := bytes.NewBuffer(header)
	_ = binary.Write(buf, binary.BigEndian, uint32(42)) // error code
	_ = binary.Write(buf, binary.BigEndian, uint32(5))  // payload size
	buf.WriteString("error")
	f, err := ParseFrame(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseFrame: %v", err)
	}
	if f.MsgType != msgTypeError {
		t.Errorf("MsgType = %d, want %d", f.MsgType, msgTypeError)
	}
	if f.ErrorCode != 42 {
		t.Errorf("ErrorCode = %d, want 42", f.ErrorCode)
	}
	if string(f.Payload) != "error" {
		t.Errorf("Payload = %s, want error", string(f.Payload))
	}
}

func TestParseFrameGzipDecompress(t *testing.T) {
	original := []byte("compressed payload data")
	var compressed bytes.Buffer
	w := gzip.NewWriter(&compressed)
	_, _ = w.Write(original)
	_ = w.Close()

	header := buildHeader(msgTypeFullServer, flagWithEvent, serializationJSON, compressionGzip)
	buf := bytes.NewBuffer(header)
	_ = binary.Write(buf, binary.BigEndian, int32(eventChatResponse))
	_ = binary.Write(buf, binary.BigEndian, uint32(len("sid")))
	buf.WriteString("sid")
	_ = binary.Write(buf, binary.BigEndian, uint32(len(compressed.Bytes())))
	buf.Write(compressed.Bytes())

	f, err := ParseFrame(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseFrame: %v", err)
	}
	if !bytes.Equal(f.Payload, original) {
		t.Errorf("Payload = %s, want %s", string(f.Payload), string(original))
	}
}

func TestParseFrameMissingEventID(t *testing.T) {
	header := buildHeader(msgTypeFullServer, flagWithEvent, serializationJSON, compressionNone)
	// Only header, no event ID.
	_, err := ParseFrame(header)
	if err == nil {
		t.Fatal("expected error for missing event id")
	}
	if !strings.Contains(err.Error(), "event id") {
		t.Errorf("err = %v, want 'event id'", err)
	}
}

func TestParseFrameMissingPayloadSize(t *testing.T) {
	header := buildHeader(msgTypeFullServer, 0, serializationJSON, compressionNone)
	// No event flag, no payload size.
	_, err := ParseFrame(header)
	if err == nil {
		t.Fatal("expected error for missing payload size")
	}
}

func TestGzipRoundtrip(t *testing.T) {
	original := bytes.Repeat([]byte{0xAB, 0xCD}, 50)
	compressed := GzipCompress(original)
	decompressed, err := GzipDecompress(compressed)
	if err != nil {
		t.Fatalf("GzipDecompress: %v", err)
	}
	if !bytes.Equal(decompressed, original) {
		t.Errorf("roundtrip mismatch")
	}
}

func TestGzipDecompressInvalid(t *testing.T) {
	_, err := GzipDecompress([]byte{0x00, 0x01, 0x02})
	if err == nil {
		t.Fatal("expected error for invalid gzip data")
	}
}

func TestShouldWriteSessionID(t *testing.T) {
	if shouldWriteSessionID(eventStartConnection) {
		t.Error("StartConnection should not write session ID")
	}
	if shouldWriteSessionID(eventFinishConnection) {
		t.Error("FinishConnection should not write session ID")
	}
	if !shouldWriteSessionID(eventStartSession) {
		t.Error("StartSession should write session ID")
	}
	if !shouldWriteSessionID(eventTaskRequest) {
		t.Error("TaskRequest should write session ID")
	}
}

func TestFrameIsAudioServer(t *testing.T) {
	f := &Frame{MsgType: msgTypeAudioOnlyServer, Payload: []byte{1, 2}}
	if !f.IsAudioServer() {
		t.Error("expected IsAudioServer true")
	}
	f2 := &Frame{MsgType: msgTypeFullServer, Payload: []byte{1, 2}}
	if f2.IsAudioServer() {
		t.Error("expected IsAudioServer false for full server frame")
	}
	f3 := &Frame{MsgType: msgTypeAudioOnlyServer, Payload: nil}
	if f3.IsAudioServer() {
		t.Error("expected IsAudioServer false for empty payload")
	}
}
