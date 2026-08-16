package recognizer

import (
	"bytes"
	"testing"
)

func TestGzipCompressDecompress(t *testing.T) {
	original := []byte("hello world, this is a test message for gzip compress/decompress")
	compressed := GzipCompress(original)
	if len(compressed) == 0 {
		t.Fatal("GzipCompress returned empty data")
	}

	decompressed := GzipDecompress(compressed)
	if !bytes.Equal(decompressed, original) {
		t.Errorf("GzipDecompress(GzipCompress(x)) != x\n got: %q\nwant: %q", string(decompressed), string(original))
	}
}

func TestGzipDecompressInvalidInput(t *testing.T) {
	result := GzipDecompress([]byte("not gzip data"))
	if result != nil {
		t.Errorf("GzipDecompress(invalid) should return nil, got %q", string(result))
	}
}

func TestGzipCompressEmpty(t *testing.T) {
	compressed := GzipCompress([]byte{})
	if len(compressed) == 0 {
		t.Fatal("GzipCompress(empty) should still return gzip header bytes")
	}
	decompressed := GzipDecompress(compressed)
	if len(decompressed) != 0 {
		t.Errorf("GzipDecompress(GzipCompress(empty)) should be empty, got %d bytes", len(decompressed))
	}
}

func TestProtocolHeaderSerialize(t *testing.T) {
	header := NewDefaultHeader()
	serialized := header.Serialize()
	if len(serialized) != 4 {
		t.Errorf("Serialize() length = %d, want 4", len(serialized))
	}

	// First byte: protocol version (0x1) << 4 | 0x1 = 0x11
	if serialized[0] != 0x11 {
		t.Errorf("byte[0] = 0x%02x, want 0x11", serialized[0])
	}

	// Second byte: messageType (0x1) << 4 | flags (0x1) = 0x11
	if serialized[1] != 0x11 {
		t.Errorf("byte[1] = 0x%02x, want 0x11", serialized[1])
	}

	// Third byte: serialization (0x1) << 4 | compression (0x1) = 0x11
	if serialized[2] != 0x11 {
		t.Errorf("byte[2] = 0x%02x, want 0x11", serialized[2])
	}

	// Fourth byte: reserved data 0x00
	if serialized[3] != 0x00 {
		t.Errorf("byte[3] = 0x%02x, want 0x00", serialized[3])
	}
}

func TestProtocolHeaderBuilders(t *testing.T) {
	header := NewDefaultHeader().
		SetMessageType(MessageTypeClientAudioOnlyRequest).
		SetMessageTypeFlags(FlagNegWithSequence).
		SetSerializationType(SerializationRaw).
		SetCompressionType(CompressionNone).
		SetReservedData([]byte{0xFF})

	serialized := header.Serialize()
	if serialized[1] != byte(byte(MessageTypeClientAudioOnlyRequest)<<4|byte(FlagNegWithSequence)) {
		t.Errorf("byte[1] = 0x%02x", serialized[1])
	}
	if serialized[2] != byte(byte(SerializationRaw)<<4|byte(CompressionNone)) {
		t.Errorf("byte[2] = 0x%02x", serialized[2])
	}
	if serialized[3] != 0xFF {
		t.Errorf("byte[3] = 0x%02x, want 0xFF", serialized[3])
	}
}

func TestGzipCompressRoundTripLarge(t *testing.T) {
	// Test with larger data to ensure gzip works correctly
	original := make([]byte, 4096)
	for i := range original {
		original[i] = byte(i % 256)
	}

	compressed := GzipCompress(original)
	decompressed := GzipDecompress(compressed)

	if !bytes.Equal(decompressed, original) {
		t.Error("Gzip round-trip failed for large data")
	}
}

func TestDefaultSampleRate(t *testing.T) {
	if DefaultSampleRate != 16000 {
		t.Errorf("DefaultSampleRate = %d, want 16000", DefaultSampleRate)
	}
}
