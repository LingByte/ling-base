package recognizerstd

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"

	base "github.com/LingByte/ling-base/providers/voice/recognizer"
	asrvaws "github.com/LingByte/ling-base/providers/voice/recognizer/aws"
	asrvbaidu "github.com/LingByte/ling-base/providers/voice/recognizer/baidu"
	asrvdeepgram "github.com/LingByte/ling-base/providers/voice/recognizer/deepgram"
	asrvfunasr "github.com/LingByte/ling-base/providers/voice/recognizer/funasr"
	asrvgladia "github.com/LingByte/ling-base/providers/voice/recognizer/gladia"
	asrvgoogle "github.com/LingByte/ling-base/providers/voice/recognizer/google"
	asrvlocal "github.com/LingByte/ling-base/providers/voice/recognizer/local"
	asrvqcloud "github.com/LingByte/ling-base/providers/voice/recognizer/qcloud"
	asrvvoiceapi "github.com/LingByte/ling-base/providers/voice/recognizer/voiceapi"
	asrvvolcengine "github.com/LingByte/ling-base/providers/voice/recognizer/volcengine"
	asrvvolcenginellm "github.com/LingByte/ling-base/providers/voice/recognizer/volcengine_llm"
	asrvwhisper "github.com/LingByte/ling-base/providers/voice/recognizer/whisper"
	"github.com/sirupsen/logrus"
)

// NewTranscriberFactory returns a factory with default vendor creators registered.
func NewTranscriberFactory() *base.DefaultFactory {
	factory := base.NewFactory()
	registerDefaultCreators(factory)
	return factory
}

func registerDefaultCreators(f *base.DefaultFactory) {
	f.RegisterCreator(base.VendorQCloud, func(config base.TranscriberConfig) (base.Engine, error) {
		qcloudConfig, ok := config.(*asrvqcloud.QCloudASROption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for qcloud")
		}
		return asrvqcloud.NewQcloudASR(*qcloudConfig), nil
	})

	f.RegisterCreator(base.VendorGoogle, func(config base.TranscriberConfig) (base.Engine, error) {
		googleConfig, ok := config.(*asrvgoogle.GoogleASROption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for google")
		}
		return asrvgoogle.NewGoogleASR(*googleConfig), nil
	})

	f.RegisterCreator(base.VendorFunASR, func(config base.TranscriberConfig) (base.Engine, error) {
		funasrConfig, ok := config.(*asrvfunasr.FunASROption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for funasr")
		}
		return asrvfunasr.NewFunASR(*funasrConfig), nil
	})

	f.RegisterCreator(base.VendorVolcengineLLM, func(config base.TranscriberConfig) (base.Engine, error) {
		volcengineLLMConfig, ok := config.(*asrvvolcenginellm.VolcengineLLMOption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for volcengine_llm")
		}
		asr := asrvvolcenginellm.NewVolcengineLLM(*volcengineLLMConfig)
		return &asr, nil
	})

	f.RegisterCreator(base.VendorXfyunMul, func(config base.TranscriberConfig) (base.Engine, error) {
		return nil, fmt.Errorf("xfyun_mul requires pipeline.SessionHandler, use xunfei vendor directly")
	})

	f.RegisterCreator(base.VendorGladia, func(config base.TranscriberConfig) (base.Engine, error) {
		gladiaConfig, ok := config.(*asrvgladia.GladiaASROption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for gladia")
		}
		return asrvgladia.NewGladiaASR(*gladiaConfig), nil
	})

	f.RegisterCreator(base.VendorFunASRRealtime, func(config base.TranscriberConfig) (base.Engine, error) {
		return nil, fmt.Errorf("funasr_realtime is not yet implemented in ling-base recognizer")
	})

	f.RegisterCreator(base.VendorLocal, func(config base.TranscriberConfig) (base.Engine, error) {
		localConfig, ok := config.(*asrvlocal.LocalASROption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for local")
		}
		return asrvlocal.NewLocalASR(*localConfig), nil
	})

	f.RegisterCreator(base.VendorVolcengine, func(config base.TranscriberConfig) (base.Engine, error) {
		opt, ok := config.(*asrvvolcengine.VolcengineOption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for volcengine")
		}
		return asrvvolcengine.NewVolcengineASR(*opt), nil
	})

	f.RegisterCreator(base.VendorDeepgram, func(config base.TranscriberConfig) (base.Engine, error) {
		deepgramConfig, ok := config.(*asrvdeepgram.DeepgramASROption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for deepgram")
		}
		return asrvdeepgram.NewDeepgramASR(*deepgramConfig), nil
	})

	f.RegisterCreator(base.VendorAWS, func(config base.TranscriberConfig) (base.Engine, error) {
		awsConfig, ok := config.(*asrvaws.AWSASROption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for aws")
		}
		return asrvaws.NewAWSASR(*awsConfig), nil
	})

	f.RegisterCreator(base.VendorBaidu, func(config base.TranscriberConfig) (base.Engine, error) {
		baiduConfig, ok := config.(*asrvbaidu.BaiduASROption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for baidu")
		}
		return asrvbaidu.NewBaiduASR(*baiduConfig), nil
	})

	f.RegisterCreator(base.VendorVoiceAPI, func(config base.TranscriberConfig) (base.Engine, error) {
		voiceapiConfig, ok := config.(*asrvvoiceapi.VoiceAPIASROption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for voiceapi")
		}
		return asrvvoiceapi.NewVoiceAPIASR(*voiceapiConfig), nil
	})

	f.RegisterCreator(base.VendorWhisper, func(config base.TranscriberConfig) (base.Engine, error) {
		whisperConfig, ok := config.(*asrvwhisper.WhisperASROption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for whisper")
		}
		return asrvwhisper.NewWhisperASR(*whisperConfig), nil
	})

	logrus.WithFields(logrus.Fields{
		"vendors": f.GetSupportedVendors(),
	}).Info("transcriber factory initialized")
}