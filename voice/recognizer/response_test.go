package recognizer

import (
	"testing"
)

func TestParseResponseServerFullResponse(t *testing.T) {
	// Build a server full response message
	header := NewDefaultHeader().
		SetMessageType(MessageTypeServerFullResponse).
		SetMessageTypeFlags(FlagPosSequence).
		SetSerializationType(SerializationJSON).
		SetCompressionType(CompressionGZIP)

	headerBytes := header.Serialize()

	// Payload: sequence (4 bytes) + payload size (4 bytes) + gzip-compressed JSON
	payload := []byte(`{"result":{"text":"hello world"}}`)
	compressed := GzipCompress(payload)

	// Sequence = 1
	seqBytes := []byte{0x00, 0x00, 0x00, 0x01}
	// Payload size
	sizeBytes := make([]byte, 4)
	sizeBytes[3] = byte(len(compressed))

	msg := append(headerBytes, seqBytes...)
	msg = append(msg, sizeBytes...)
	msg = append(msg, compressed...)

	resp := ParseResponse(msg)
	if resp == nil {
		t.Fatal("ParseResponse returned nil")
	}
	if resp.PayloadSequence != 1 {
		t.Errorf("PayloadSequence = %d, want 1", resp.PayloadSequence)
	}
	if resp.PayloadMsg == nil {
		t.Fatal("PayloadMsg should not be nil")
	}
	if resp.PayloadMsg.Result.Text != "hello world" {
		t.Errorf("Text = %q, want %q", resp.PayloadMsg.Result.Text, "hello world")
	}
}

func TestParseResponseServerErrorResponse(t *testing.T) {
	header := NewDefaultHeader().
		SetMessageType(MessageTypeServerErrorResponse).
		SetMessageTypeFlags(FlagNoSequence).
		SetSerializationType(SerializationJSON).
		SetCompressionType(CompressionGZIP)

	headerBytes := header.Serialize()

	// Error code = 1001
	codeBytes := []byte{0x00, 0x00, 0x03, 0xE9}

	payload := []byte(`{"error":"bad request"}`)
	compressed := GzipCompress(payload)
	sizeBytes := make([]byte, 4)
	sizeBytes[3] = byte(len(compressed))

	msg := append(headerBytes, codeBytes...)
	msg = append(msg, sizeBytes...)
	msg = append(msg, compressed...)

	resp := ParseResponse(msg)
	if resp == nil {
		t.Fatal("ParseResponse returned nil")
	}
	if resp.Code != 1001 {
		t.Errorf("Code = %d, want 1001", resp.Code)
	}
}

func TestParseResponseLastPackage(t *testing.T) {
	header := NewDefaultHeader().
		SetMessageType(MessageTypeServerFullResponse).
		SetMessageTypeFlags(FlagNegWithSequence).
		SetSerializationType(SerializationJSON).
		SetCompressionType(CompressionGZIP)

	headerBytes := header.Serialize()

	// Negative sequence = -1 (0xFFFFFFFF)
	seqBytes := []byte{0xFF, 0xFF, 0xFF, 0xFF}

	payload := []byte(`{"result":{"text":"final"}}`)
	compressed := GzipCompress(payload)
	sizeBytes := make([]byte, 4)
	sizeBytes[3] = byte(len(compressed))

	msg := append(headerBytes, seqBytes...)
	msg = append(msg, sizeBytes...)
	msg = append(msg, compressed...)

	resp := ParseResponse(msg)
	if resp == nil {
		t.Fatal("ParseResponse returned nil")
	}
	if !resp.IsLastPackage {
		t.Error("IsLastPackage should be true for FlagNegWithSequence")
	}
}
