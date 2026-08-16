package recognizer

import (
	"encoding/json"
	"testing"
)

func TestNewFullClientRequest(t *testing.T) {
	cfg := DefaultConfig().
		WithAuth(AuthConfig{ResourceId: "test-resource", AccessKey: "test-key", AppKey: "test-app"}).
		WithUser(UserConfig{UID: "test-uid"})

	req := NewFullClientRequest(cfg)
	if len(req) < 4 {
		t.Fatalf("NewFullClientRequest returned %d bytes, want >= 4", len(req))
	}

	// Header should start with 0x11 (protocol version 1 << 4 | 1)
	if req[0] != 0x11 {
		t.Errorf("header byte[0] = 0x%02x, want 0x11", req[0])
	}
}

func TestNewAudioOnlyRequest(t *testing.T) {
	audio := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	req := NewAudioOnlyRequest(1, audio)
	if len(req) < 4 {
		t.Fatalf("NewAudioOnlyRequest returned %d bytes, want >= 4", len(req))
	}

	// Header byte[1] should have MessageTypeClientAudioOnlyRequest (0x2) in high nibble
	if req[1]>>4 != 0x02 {
		t.Errorf("header byte[1] high nibble = 0x%01x, want 0x2", req[1]>>4)
	}
}

func TestNewAudioOnlyRequestNegativeSeq(t *testing.T) {
	audio := []byte{0x01, 0x02}
	req := NewAudioOnlyRequest(-1, audio)
	if len(req) < 4 {
		t.Fatalf("NewAudioOnlyRequest returned %d bytes, want >= 4", len(req))
	}

	// For negative sequence, flags should be FlagNegWithSequence (0b0011)
	if req[1]&0x0F != byte(FlagNegWithSequence) {
		t.Errorf("header byte[1] low nibble = 0x%01x, want 0x3 (FlagNegWithSequence)", req[1]&0x0F)
	}
}

func TestBuildAuthHeader(t *testing.T) {
	auth := AuthConfig{
		ResourceId: "test-resource",
		AccessKey:  "test-access-key",
		AppKey:     "test-app-key",
	}
	header := BuildAuthHeader(auth)

	if header.Get("X-Api-Resource-Id") != "test-resource" {
		t.Errorf("X-Api-Resource-Id = %q", header.Get("X-Api-Resource-Id"))
	}
	if header.Get("X-Api-Access-Key") != "test-access-key" {
		t.Errorf("X-Api-Access-Key = %q", header.Get("X-Api-Access-Key"))
	}
	if header.Get("X-Api-App-Key") != "test-app-key" {
		t.Errorf("X-Api-App-Key = %q", header.Get("X-Api-App-Key"))
	}
	if header.Get("X-Api-Request-Id") == "" {
		t.Error("X-Api-Request-Id should not be empty")
	}
}

func TestRequestPayloadJSON(t *testing.T) {
	payload := RequestPayload{
		User: UserMeta{UID: "test-uid", Platform: "test"},
		Audio: AudioMeta{Format: "pcm", Rate: 16000, Bits: 16, Channel: 1},
		Request: RequestMeta{ModelName: "bigmodel", EnableITN: true, EnablePUNC: true},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded RequestPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.User.UID != "test-uid" {
		t.Errorf("User.UID = %q, want %q", decoded.User.UID, "test-uid")
	}
	if decoded.Audio.Rate != 16000 {
		t.Errorf("Audio.Rate = %d, want 16000", decoded.Audio.Rate)
	}
	if decoded.Request.ModelName != "bigmodel" {
		t.Errorf("Request.ModelName = %q, want %q", decoded.Request.ModelName, "bigmodel")
	}
}
