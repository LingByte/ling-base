package synthesizerstd

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"fmt"

	base "github.com/LingByte/ling-base/providers/voice/synthesizer"
	ttsaws "github.com/LingByte/ling-base/providers/voice/synthesizer/aws"
	ttsaliyun "github.com/LingByte/ling-base/providers/voice/synthesizer/aliyun"
	ttsazure "github.com/LingByte/ling-base/providers/voice/synthesizer/azure"
	ttsbaidu "github.com/LingByte/ling-base/providers/voice/synthesizer/baidu"
	ttscoqui "github.com/LingByte/ling-base/providers/voice/synthesizer/coqui"
	ttselevenlabs "github.com/LingByte/ling-base/providers/voice/synthesizer/elevenlabs"
	ttsfishaudio "github.com/LingByte/ling-base/providers/voice/synthesizer/fishaudio"
	ttsfishspeech "github.com/LingByte/ling-base/providers/voice/synthesizer/fishspeech"
	ttsgoogle "github.com/LingByte/ling-base/providers/voice/synthesizer/google"
	ttslocal "github.com/LingByte/ling-base/providers/voice/synthesizer/local"
	ttsminimax "github.com/LingByte/ling-base/providers/voice/synthesizer/minimax"
	ttsopenai "github.com/LingByte/ling-base/providers/voice/synthesizer/openai"
	ttsqcloud "github.com/LingByte/ling-base/providers/voice/synthesizer/qcloud"
	ttsqiniu "github.com/LingByte/ling-base/providers/voice/synthesizer/qiniu"
	ttsvolcengine "github.com/LingByte/ling-base/providers/voice/synthesizer/volcengine"
	ttsxunfei "github.com/LingByte/ling-base/providers/voice/synthesizer/xunfei"

	"github.com/aws/aws-sdk-go-v2/service/polly/types"
)

func newEngineFromCredential(providerName string, options map[string]any) (base.Engine, error) {
	if len(options) == 0 {
		return nil, fmt.Errorf("empty TTS options for %s", providerName)
	}
	raw, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("marshal TTS options: %w", err)
	}

	switch providerName {
	case base.TTS_QINIU:
		var opt ttsqiniu.QiniuTTSConfig
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttsqiniu.NewQiniuService(opt), nil
	case base.TTS_QCLOUD:
		var opt ttsqcloud.QCloudTTSConfig
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttsqcloud.NewQCloudService(opt), nil
	case base.TTS_BAIDU:
		var opt ttsbaidu.BaiduTTSConfig
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttsbaidu.NewBaiduService(opt), nil
	case base.TTS_AZURE:
		var opt ttsazure.AzureConfig
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttsazure.NewAzureService(opt), nil
	case base.TTS_XUNFEI:
		var opt ttsxunfei.XunfeiTTSConfig
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttsxunfei.NewXunfeiService(opt), nil
	case base.TTS_OPENAI:
		var opt ttsopenai.OpenAIConfig
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttsopenai.NewOpenAIService(opt), nil
	case base.TTS_GOOGLE:
		var opt ttsgoogle.GoogleTTSOption
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttsgoogle.NewGoogleService(opt), nil
	case base.TTS_AWS:
		region, _ := options["region"].(string)
		if region == "" {
			region = "us-east-1"
		}
		opt := ttsaws.NewAmazonTTSOption(region, types.OutputFormatPcm, types.VoiceIdJoanna)
		if sr, ok := options["sample_rate"].(float64); ok && sr > 0 {
			opt.SampleRate = int(sr)
		}
		return ttsaws.NewAmazonService(opt), nil
	case base.TTS_VOLCENGINE:
		var opt ttsvolcengine.VolcengineTTSOption
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttsvolcengine.NewVolcengineService(opt), nil
	case base.TTS_VOLCENGINE_CLONE:
		return newVolcengineCloneEngineFromMap(options)
	case base.TTS_MINIMAX:
		var opt ttsminimax.MinimaxOption
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttsminimax.NewMinimaxService(opt), nil
	case base.TTS_ELEVENLABS:
		var opt ttselevenlabs.ElevenLabsConfig
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttselevenlabs.NewElevenLabsService(opt), nil
	case base.TTS_LOCAL:
		var opt ttslocal.LocalTTSConfig
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttslocal.NewLocalService(opt), nil
	case base.TTS_FISHSPEECH:
		var opt ttsfishspeech.FishSpeechConfig
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttsfishspeech.NewFishSpeechService(opt), nil
	case base.TTS_FISHAUDIO:
		var opt ttsfishaudio.FishAudioConfig
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttsfishaudio.NewFishAudioService(opt), nil
	case base.TTS_COQUI:
		var opt ttscoqui.CoquiTTSOption
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttscoqui.NewCoquiService(opt), nil
	case base.TTS_LOCAL_GOSPEECH:
		var opt ttslocal.LocalGoSpeechConfig
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttslocal.NewLocalGoSpeechService(&opt)
	case base.TTS_ALIYUN:
		var opt ttsaliyun.AliyunTTSConfig
		if err := json.Unmarshal(raw, &opt); err != nil {
			return nil, err
		}
		return ttsaliyun.NewAliyunService(opt), nil
	default:
		return nil, fmt.Errorf("unsupported TTS provider: %s", providerName)
	}
}