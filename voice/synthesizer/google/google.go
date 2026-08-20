// Package synthesizer implements the Google Cloud Text-to-Speech adapter for ling-base.
package synthesizer

import (
	"context"
	"fmt"
	"sync"

	base "github.com/LingByte/ling-base/voice/synthesizer"
	"github.com/sirupsen/logrus"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
)

// GoogleTTSOption Google TTS 配置
type GoogleTTSOption struct {
	LanguageCode  string                         `json:"languageCode" yaml:"language_code"`
	SsmlGender    texttospeechpb.SsmlVoiceGender `json:"ssmlGender" yaml:"ssml_gender"`
	AudioEncoding texttospeechpb.AudioEncoding   `json:"audioEncoding" yaml:"audio_encoding" default:"LINEAR16"`
	SampleRate    int                            `json:"sampleRate" yaml:"sample_rate" default:"16000"`
	Channels      int                            `json:"channels" yaml:"channels" default:"1"`
	BitDepth      int                            `json:"bitDepth" yaml:"bit_depth" default:"16"`
	FrameDuration string                         `json:"frameDuration" yaml:"frame_duration" default:"20ms"`
}

// GetProvider returns the TTS provider type
func (c *GoogleTTSOption) GetProvider() base.Provider {
	return base.ProviderGoogle
}

func (opt *GoogleTTSOption) String() string {
	return fmt.Sprintf("GoogleTTSOption{LanguageCode: %s, SsmlGender: %d, AudioEncoding: %d, SampleRate: %d, Channels: %d, BitDepth: %d}",
		opt.LanguageCode, opt.SsmlGender, opt.AudioEncoding, opt.SampleRate, opt.Channels, opt.BitDepth)
}

// NewGoogleTTSOption 创建 Google TTS 配置
func NewGoogleTTSOption(languageCode string) GoogleTTSOption {
	return GoogleTTSOption{
		LanguageCode:  languageCode,
		SsmlGender:    texttospeechpb.SsmlVoiceGender_NEUTRAL,
		AudioEncoding: texttospeechpb.AudioEncoding_LINEAR16,
		Channels:      1,
		SampleRate:    16000,
		BitDepth:      16,
		FrameDuration: "20ms",
	}
}

// GoogleService Google TTS 服务
type GoogleService struct {
	opt GoogleTTSOption
	mu  sync.Mutex
}

// NewGoogleService 创建 Google TTS 服务
func NewGoogleService(opt GoogleTTSOption) *GoogleService {
	return &GoogleService{opt: opt}
}

func (gs *GoogleService) Close() error {
	return nil
}

func (gs *GoogleService) Format() base.StreamFormat {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return base.StreamFormat{
		Channels:      gs.opt.Channels,
		SampleRate:    gs.opt.SampleRate,
		BitDepth:      gs.opt.BitDepth,
		Codec:         "pcm",
		FrameDuration: base.NormalizeFramePeriod(gs.opt.FrameDuration),
	}
}

func (gs *GoogleService) Provider() base.Provider {
	return base.ProviderGoogle
}

func (gs *GoogleService) CacheKey(text string) string {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return fmt.Sprintf("google.tts-%s-%d-%s.pcm", gs.opt.LanguageCode, gs.opt.AudioEncoding, base.HashText(text))
}

func (gs *GoogleService) Synthesize(ctx context.Context, handler base.Handler, text string) error {
	gs.mu.Lock()
	opt := gs.opt
	gs.mu.Unlock()

	if text == "" {
		handler.OnMessage(nil)
		return nil
	}

	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		logrus.WithError(err).Error("google tts: create client failed")
		return nil
	}
	defer client.Close()

	req := texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: text},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: opt.LanguageCode,
			SsmlGender:   opt.SsmlGender,
		},
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: opt.AudioEncoding,
		},
	}

	resp, err := client.SynthesizeSpeech(ctx, &req)
	if err != nil {
		logrus.WithError(err).Error("google tts: synthesize failed")
		return err
	}

	if len(resp.AudioContent) > 0 {
		handler.OnMessage(resp.AudioContent)
	}

	logrus.WithFields(logrus.Fields{
		"languageCode": opt.LanguageCode,
		"audioSize":    len(resp.AudioContent),
	}).Info("google tts: synthesis complete")

	return nil
}

// Compile-time guard.
var _ base.Engine = (*GoogleService)(nil)
