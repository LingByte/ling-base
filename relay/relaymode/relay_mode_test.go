// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package constant

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPath2RelayMode(t *testing.T) {
	tests := []struct {
		name string
		path string
		want int
	}{
		{"chat completions", "/v1/chat/completions", RelayModeChatCompletions},
		{"chat completions pg", "/pg/chat/completions", RelayModeChatCompletions},
		{"chat completions with query", "/v1/chat/completions?foo=1", RelayModeChatCompletions},
		{"completions", "/v1/completions", RelayModeCompletions},
		{"embeddings", "/v1/embeddings", RelayModeEmbeddings},
		{"embeddings suffix", "/some/custom/embeddings", RelayModeEmbeddings},
		{"moderations", "/v1/moderations", RelayModeModerations},
		{"images generations", "/v1/images/generations", RelayModeImagesGenerations},
		{"images edits", "/v1/images/edits", RelayModeImagesEdits},
		{"edits", "/v1/edits", RelayModeEdits},
		{"responses", "/v1/responses", RelayModeResponses},
		{"responses compact", "/v1/responses/compact", RelayModeResponsesCompact},
		{"alpha search", "/v1/alpha/search", RelayModeAlphaSearch},
		{"alpha search with query", "/v1/alpha/search?foo=1", RelayModeAlphaSearch},
		{"audio speech", "/v1/audio/speech", RelayModeAudioSpeech},
		{"audio transcription", "/v1/audio/transcriptions", RelayModeAudioTranscription},
		{"audio translation", "/v1/audio/translations", RelayModeAudioTranslation},
		{"rerank", "/v1/rerank", RelayModeRerank},
		{"realtime", "/v1/realtime", RelayModeRealtime},
		{"realtime pg", "/pg/realtime", RelayModeRealtime},
		{"gemini v1beta", "/v1beta/models/gemini-pro:generateContent", RelayModeGemini},
		{"gemini v1 models", "/v1/models/gemini-pro", RelayModeGemini},
		{"unknown", "/v1/unknown", RelayModeUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Path2RelayMode(tt.path))
		})
	}
}

func TestPath2RelayModeMidjourney(t *testing.T) {
	tests := []struct {
		name string
		path string
		want int
	}{
		{"imagine", "/mj/submit/imagine", RelayModeMidjourneyImagine},
		{"describe", "/mj/submit/describe", RelayModeMidjourneyDescribe},
		{"blend", "/mj/submit/blend", RelayModeMidjourneyBlend},
		{"change", "/mj/submit/change", RelayModeMidjourneyChange},
		{"simple-change", "/mj/submit/simple-change", RelayModeMidjourneyChange},
		{"action", "/mj/submit/action", RelayModeMidjourneyAction},
		{"modal", "/mj/submit/modal", RelayModeMidjourneyModal},
		{"shorten", "/mj/submit/shorten", RelayModeMidjourneyShorten},
		{"video", "/mj/submit/video", RelayModeMidjourneyVideo},
		{"edits", "/mj/submit/edits", RelayModeMidjourneyEdits},
		{"notify", "/mj/notify", RelayModeMidjourneyNotify},
		{"swap face", "/mj/insight-face/swap", RelayModeSwapFace},
		{"upload", "/submit/upload-discord-images", RelayModeMidjourneyUpload},
		{"task fetch", "/mj/task/1234/fetch", RelayModeMidjourneyTaskFetch},
		{"image seed", "/mj/task/1234/image-seed", RelayModeMidjourneyTaskImageSeed},
		{"list by condition", "/mj/task/list-by-condition", RelayModeMidjourneyTaskFetchByCondition},
		{"unknown mj", "/mj/unknown", RelayModeUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Path2RelayModeMidjourney(tt.path))
		})
	}
}

func TestPath2RelayMode_MidjourneyDispatch(t *testing.T) {
	// Paths starting with /mj are dispatched to Path2RelayModeMidjourney.
	assert.Equal(t, RelayModeMidjourneyImagine, Path2RelayMode("/mj/submit/imagine"))
	assert.Equal(t, RelayModeMidjourneyAction, Path2RelayMode("/mj/submit/action"))
}

func TestPath2RelaySuno(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"fetch POST", http.MethodPost, "/suno/fetch", RelayModeSunoFetch},
		{"fetch by id GET", http.MethodGet, "/suno/fetch/123", RelayModeSunoFetchByID},
		{"submit", http.MethodPost, "/suno/submit/music", RelayModeSunoSubmit},
		{"submit lyrics", http.MethodPost, "/suno/submit/lyrics", RelayModeSunoSubmit},
		{"unknown", http.MethodGet, "/suno/unknown", RelayModeUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Path2RelaySuno(tt.method, tt.path))
		})
	}
}
