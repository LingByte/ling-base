package synthesizerstd

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	base "github.com/LingByte/ling-base/providers/voice/synthesizer"
	"github.com/LingByte/ling-base/providers/voice/voiceclone"
)

const volcengineCloneCluster = "volcano_icl"

// VolcengineCloneOption configures Volcengine voice-clone TTS (volcano_icl).
type VolcengineCloneOption struct {
	AppID       string  `json:"appId"`
	AccessToken string  `json:"accessToken"`
	Cluster     string  `json:"cluster"`
	ResourceID  string  `json:"resourceId"`
	ModelType   int     `json:"modelType"`
	AssetID     string  `json:"assetId"`
	Encoding    string  `json:"encoding"`
	Rate        int     `json:"rate"`
	SourceRate  int     `json:"sourceRate"`
	SpeedRatio  float64 `json:"speedRatio"`
	Streaming   bool    `json:"streaming"`
}

// VolcengineCloneEngine wraps voiceclone.VoiceCloneService as base.Engine.
type VolcengineCloneEngine struct {
	svc voiceclone.VoiceCloneService
	opt VolcengineCloneOption
	mu  sync.Mutex
}

func newVolcengineCloneEngineFromMap(options map[string]any) (base.Engine, error) {
	data, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}
	var opt VolcengineCloneOption
	if err := json.Unmarshal(data, &opt); err != nil {
		return nil, err
	}
	return newVolcengineCloneEngine(opt)
}

func newVolcengineCloneEngine(opt VolcengineCloneOption) (base.Engine, error) {
	if opt.AppID == "" || opt.AccessToken == "" {
		return nil, fmt.Errorf("volcengine_clone: appId and accessToken are required")
	}
	if opt.AssetID == "" {
		return nil, fmt.Errorf("volcengine_clone: assetId is required")
	}
	cluster := opt.Cluster
	if cluster == "" {
		cluster = volcengineCloneCluster
	}
	encoding := opt.Encoding
	if encoding == "" {
		encoding = "pcm"
	}
	sourceRate := opt.SourceRate
	if sourceRate <= 0 {
		sourceRate = opt.Rate
	}
	if sourceRate <= 0 {
		sourceRate = 24000
	}
	outRate := opt.Rate
	if outRate <= 0 {
		outRate = 16000
	}
	speed := opt.SpeedRatio
	if speed <= 0 {
		speed = 1.0
	}
	resourceID := opt.ResourceID
	if resourceID == "" {
		resourceID = "seed-icl-2.0"
	}
	modelType := opt.ModelType
	if modelType == 0 {
		modelType = 4
	}
	svc, err := voiceclone.NewFactory().CreateService(&voiceclone.Config{
		Provider: voiceclone.ProviderVolcengine,
		Options: map[string]interface{}{
			"app_id":      opt.AppID,
			"token":       opt.AccessToken,
			"cluster":     cluster,
			"resource_id": resourceID,
			"model_type":  modelType,
			"encoding":    encoding,
			"sample_rate": sourceRate,
			"speed_ratio": speed,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("volcengine_clone: %w", err)
	}
	opt.Cluster = cluster
	opt.Encoding = encoding
	opt.SourceRate = sourceRate
	opt.Rate = outRate
	opt.SpeedRatio = speed
	return &VolcengineCloneEngine{svc: svc, opt: opt}, nil
}

func (e *VolcengineCloneEngine) Provider() base.Provider {
	return base.ProviderVolcengineClone
}

func (e *VolcengineCloneEngine) Format() base.StreamFormat {
	e.mu.Lock()
	rate := e.opt.Rate
	e.mu.Unlock()
	if rate <= 0 {
		rate = 16000
	}
	return base.StreamFormat{
		SampleRate:    rate,
		BitDepth:      16,
		Channels:      1,
		Codec:         "pcm",
		FrameDuration: 20 * time.Millisecond,
	}
}

func (e *VolcengineCloneEngine) CacheKey(text string) string {
	e.mu.Lock()
	asset := e.opt.AssetID
	e.mu.Unlock()
	return fmt.Sprintf("volcengine_clone-%s-%s.pcm", asset, base.HashText(text))
}

func (e *VolcengineCloneEngine) Synthesize(ctx context.Context, handler base.Handler, text string) error {
	if e == nil || e.svc == nil || text == "" {
		return nil
	}
	e.mu.Lock()
	opt := e.opt
	e.mu.Unlock()
	req := &voiceclone.SynthesizeRequest{
		AssetID:  opt.AssetID,
		Text:     text,
		Language: "zh",
	}
	return e.svc.SynthesizeStream(ctx, req, &voicecloneStreamAdapter{fn: handler.OnMessage})
}

func (e *VolcengineCloneEngine) Close() error {
	return nil
}

type voicecloneStreamAdapter struct {
	fn func([]byte)
}

func (a *voicecloneStreamAdapter) OnMessage(data []byte) {
	if a != nil && a.fn != nil && len(data) > 0 {
		a.fn(data)
	}
}

func (a *voicecloneStreamAdapter) OnTimestamp(voiceclone.SentenceTimestamp) {}