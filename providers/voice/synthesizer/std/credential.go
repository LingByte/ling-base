package synthesizerstd

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	base "github.com/LingByte/ling-base/providers/voice/synthesizer"
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
)

// TTSCredentialConfig TTS凭证配置结构（灵活的键值对配置）
type TTSCredentialConfig map[string]interface{}

// getString 从配置中获取字符串值
func (c TTSCredentialConfig) getString(key string) string {
	if val, ok := c[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		// 尝试转换为字符串
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// getBool 从配置中获取 bool 值。
func (c TTSCredentialConfig) getBool(key string, def bool) bool {
	if val, ok := c[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return def
}

// getInt64 从配置中获取 int64 值
func (c TTSCredentialConfig) getInt64(key string) int64 {
	if val, ok := c[key]; ok {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		case string:
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return i
			}
		}
	}
	return 0
}

// NewAudioSynthesisEngineFromCredential 根据凭证配置创建TTS服务
func NewAudioSynthesisEngineFromCredential(config TTSCredentialConfig) (AudioSynthesisEngine, error) {
	if config == nil || len(config) == 0 {
		return nil, fmt.Errorf("TTS配置为空")
	}

	provider := strings.ToLower(strings.TrimSpace(config.getString("provider")))
	if provider == "" {
		return nil, fmt.Errorf("TTS provider 未配置")
	}

	var providerName string
	var options map[string]any

	switch provider {
	case "qiniu":
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		if apiKey == "" {
			return nil, fmt.Errorf("七牛云TTS配置不完整：缺少apiKey")
		}
		providerName = base.TTS_QINIU
		baseURL := config.getString("baseUrl")
		if baseURL == "" {
			baseURL = config.getString("base_url") // 兼容下划线格式
		}
		ttsConfig := ttsqiniu.NewQiniuTTSConfig(apiKey, baseURL)
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(ttsConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化七牛云配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化七牛云配置失败: %w", err)
		}

	case "qcloud", "tencent":
		appID := config.getString("appId")
		if appID == "" {
			appID = config.getString("app_id") // 兼容下划线格式
		}
		secretID := config.getString("secretId")
		if secretID == "" {
			secretID = config.getString("secret_id") // 兼容下划线格式
		}
		secretKey := config.getString("secretKey")
		if secretKey == "" {
			secretKey = config.getString("secret_key") // 兼容下划线格式
		}

		if appID == "" || secretID == "" || secretKey == "" {
			return nil, fmt.Errorf("腾讯云TTS配置不完整：缺少appId、secretId或secretKey")
		}

		providerName = base.TTS_QCLOUD
		voiceTypeStr := config.getString("voiceType")
		if voiceTypeStr == "" {
			voiceTypeStr = config.getString("voice_type") // 兼容下划线格式
		}
		if voiceTypeStr == "" {
			voiceTypeStr = "601002" // 默认值
		}
		voiceType := config.getInt64("voiceType")
		if voiceType == 0 {
			if v, err := strconv.ParseInt(voiceTypeStr, 10, 64); err == nil {
				voiceType = v
			} else {
				voiceType = 601002 // 默认值
			}
		}
		codec := config.getString("codec")
		if codec == "" {
			codec = "pcm"
		}
		// 支持配置采样率（默认16000）
		sampleRate := config.getInt64("sampleRate")
		if sampleRate == 0 {
			sampleRate = config.getInt64("sample_rate")
		}
		if sampleRate == 0 {
			sampleRate = 16000 // 默认16kHz
		}
		qcloudConfig := ttsqcloud.NewQcloudTTSConfig(appID, secretID, secretKey, voiceType, codec, int(sampleRate))
		// 读取语言配置（腾讯云通过音色类型区分语言，但保留此字段用于配置和缓存）
		language := config.getString("language")
		if language != "" {
			qcloudConfig.Language = language
		}
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(qcloudConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化腾讯云配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化腾讯云配置失败: %w", err)
		}

	case "baidu":
		token := config.getString("token")
		if token == "" {
			// 兼容旧字段名
			token = config.getString("secretKey")
			if token == "" {
				token = config.getString("tok")
			}
		}
		if token == "" {
			return nil, fmt.Errorf("百度TTS配置不完整：缺少token")
		}
		providerName = base.TTS_BAIDU
		baiduConfig := ttsbaidu.NewBaiduTTSOption(token)
		// 读取语言配置（语言代码应该已经是平台特定的格式，如 zh, en, jp, kr）
		language := config.getString("language")
		if language == "" {
			language = config.getString("lan") // 兼容 lan 字段
		}
		if language != "" {
			// 直接使用传入的语言代码（应该已经是百度格式，如 zh, en, jp, kr）
			baiduConfig.Lan = language
		}
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(baiduConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化百度配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化百度配置失败: %w", err)
		}

	case "azure":
		subscriptionKey := config.getString("subscriptionKey")
		if subscriptionKey == "" {
			subscriptionKey = config.getString("subscription_key") // 兼容下划线格式
		}
		region := config.getString("region")
		if subscriptionKey == "" || region == "" {
			return nil, fmt.Errorf("Azure TTS配置不完整：缺少subscriptionKey或region")
		}
		providerName = base.TTS_AZURE
		voice := config.getString("voice")
		if voice == "" {
			voice = "zh-CN-XiaoxiaoNeural" // 默认值
		}
		azureConfig := ttsazure.NewAzureConfig(subscriptionKey, region)
		azureConfig.Voice = voice
		// 读取语言配置（用于 SSML 中的 xml:lang）
		language := config.getString("language")
		if language == "" {
			// 如果没有指定语言，尝试从 voice 名称中提取（例如：zh-CN-XiaoxiaoNeural -> zh-CN）
			if parts := strings.Split(voice, "-"); len(parts) >= 2 {
				language = parts[0] + "-" + parts[1]
			} else {
				language = "zh-CN" // 默认值
			}
		}
		// 将语言信息存储到配置中（Azure 服务会使用它来设置 SSML 的 xml:lang）
		azureConfig.Language = language
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(azureConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化Azure配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化Azure配置失败: %w", err)
		}
		if c := config.getString("codec"); c != "" {
			options["codec"] = c
		}
		if sr := config.getInt64("sampleRate"); sr > 0 {
			options["sample_rate"] = sr
		} else if sr := config.getInt64("sample_rate"); sr > 0 {
			options["sample_rate"] = sr
		}

	case "xunfei":
		appID := config.getString("appId")
		if appID == "" {
			appID = config.getString("app_id") // 兼容下划线格式
		}
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		apiSecret := config.getString("apiSecret")
		if apiSecret == "" {
			apiSecret = config.getString("api_secret") // 兼容下划线格式
		}
		if appID == "" || apiKey == "" || apiSecret == "" {
			return nil, fmt.Errorf("科大讯飞TTS配置不完整：缺少appId、apiKey或apiSecret")
		}
		providerName = base.TTS_XUNFEI
		xunfeiConfig := ttsxunfei.NewXunfeiTTSConfig(appID, apiKey, apiSecret)
		if resID := config.getString("resId"); resID != "" {
			xunfeiConfig.ResID = resID
		} else if resID := config.getString("res_id"); resID != "" {
			xunfeiConfig.ResID = resID
		}
		if vcn := config.getString("vcn"); vcn != "" {
			xunfeiConfig.Vcn = vcn
		}
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(xunfeiConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化科大讯飞配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化科大讯飞配置失败: %w", err)
		}

	case "openai":
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI TTS配置不完整：缺少apiKey")
		}
		providerName = base.TTS_OPENAI
		baseURL := config.getString("baseUrl")
		if baseURL == "" {
			baseURL = config.getString("base_url") // 兼容下划线格式
		}
		if baseURL == "" {
			baseURL = "https://api.openai.com"
		}
		oc := ttsopenai.NewOpenAIConfig(apiKey)
		oc.BaseURL = baseURL
		if v := config.getString("voice"); v != "" {
			oc.Voice = v
		}
		if m := config.getString("model"); m != "" {
			oc.Model = m
		}
		if rf := config.getString("response_format"); rf != "" {
			oc.Codec = rf
		} else if c := config.getString("codec"); c != "" {
			oc.Codec = c
		}
		if sp := config.getString("speed"); sp != "" {
			if f, err := strconv.ParseFloat(sp, 64); err == nil {
				oc.Speed = f
			}
		}
		if sr := config.getInt64("sampleRate"); sr > 0 {
			oc.SampleRate = int(sr)
		} else if sr := config.getInt64("sample_rate"); sr > 0 {
			oc.SampleRate = int(sr)
		}
		configBytes, err := json.Marshal(oc)
		if err != nil {
			return nil, fmt.Errorf("序列化OpenAI TTS配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化OpenAI TTS配置失败: %w", err)
		}

	case "google":
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		if apiKey == "" {
			return nil, fmt.Errorf("Google TTS配置不完整：缺少apiKey")
		}
		providerName = base.TTS_GOOGLE
		projectID := config.getString("projectId")
		if projectID == "" {
			projectID = config.getString("project_id") // 兼容下划线格式
		}
		languageCode := config.getString("languageCode")
		if languageCode == "" {
			languageCode = config.getString("language_code") // 兼容下划线格式
		}
		if languageCode == "" {
			languageCode = config.getString("language") // 也支持 language 字段
		}
		if languageCode == "" {
			languageCode = "en-US" // 默认值
		}
		googleConfig := ttsgoogle.NewGoogleTTSOption(languageCode)
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(googleConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化Google配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化Google配置失败: %w", err)
		}
		// 保留 api_key 和 project_id（如果需要）
		if apiKey != "" {
			options["api_key"] = apiKey
		}
		if projectID != "" {
			options["project_id"] = projectID
		}

	case "aws":
		accessKeyID := config.getString("accessKeyId")
		if accessKeyID == "" {
			accessKeyID = config.getString("access_key_id") // 兼容下划线格式
		}
		secretAccessKey := config.getString("secretAccessKey")
		if secretAccessKey == "" {
			secretAccessKey = config.getString("secret_access_key") // 兼容下划线格式
		}
		if accessKeyID == "" || secretAccessKey == "" {
			return nil, fmt.Errorf("AWS TTS配置不完整：缺少accessKeyId或secretAccessKey")
		}
		providerName = base.TTS_AWS
		region := config.getString("region")
		if region == "" {
			region = "us-east-1" // 默认值
		}
		options = map[string]any{
			"access_key_id":     accessKeyID,
			"secret_access_key": secretAccessKey,
			"region":            region,
		}

	case "volcengine":
		appID := config.getString("appId")
		if appID == "" {
			appID = config.getString("app_id") // 兼容下划线格式
		}
		accessToken := config.getString("accessToken")
		if accessToken == "" {
			accessToken = config.getString("access_token") // 兼容下划线格式
		}
		if accessToken == "" {
			accessToken = config.getString("token") // 兼容 token 字段
		}
		cluster := config.getString("cluster")
		if cluster == "" {
			cluster = "volcano_tts" // 默认集群
		}
		if appID == "" || accessToken == "" {
			return nil, fmt.Errorf("火山引擎TTS配置不完整：缺少appId或accessToken")
		}
		providerName = base.TTS_VOLCENGINE
		voiceType := config.getString("voiceType")
		if voiceType == "" {
			voiceType = config.getString("voice_type") // 兼容下划线格式
		}
		if voiceType == "" {
			voiceType = "BV700_streaming" // 默认值
		}
		rate := config.getInt64("rate")
		if rate == 0 {
			rate = config.getInt64("sampleRate") // 尝试 sampleRate 字段
		}
		if rate == 0 {
			rate = config.getInt64("sample_rate") // 尝试 sample_rate 字段
		}
		if rate == 0 {
			rate = 16000 // 默认采样率改为16000（Volcengine标准）
		}
		encoding := config.getString("encoding")
		if encoding == "" {
			encoding = "pcm"
		}
		speedRatio := float32(1.0)
		if speedStr := config.getString("speedRatio"); speedStr != "" {
			if f, err := strconv.ParseFloat(speedStr, 32); err == nil {
				speedRatio = float32(f)
			}
		}
		volcengineConfig := ttsvolcengine.NewVolcengineTTSOption(appID, accessToken, cluster)
		volcengineConfig.VoiceType = voiceType
		volcengineConfig.Rate = int(rate)
		volcengineConfig.Encoding = encoding
		volcengineConfig.SpeedRatio = speedRatio
		volcengineConfig.Streaming = config.getBool("streaming", true)
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(volcengineConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化火山引擎配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化火山引擎配置失败: %w", err)
		}

	case "volcengine_clone":
		appID := config.getString("appId")
		if appID == "" {
			appID = config.getString("app_id")
		}
		accessToken := config.getString("accessToken")
		if accessToken == "" {
			accessToken = config.getString("access_token")
		}
		if accessToken == "" {
			accessToken = config.getString("token")
		}
		assetID := config.getString("assetId")
		if assetID == "" {
			assetID = config.getString("asset_id")
		}
		if assetID == "" {
			assetID = config.getString("voiceType")
		}
		if assetID == "" {
			assetID = config.getString("voice_type")
		}
		if appID == "" || accessToken == "" {
			return nil, fmt.Errorf("火山引擎克隆TTS配置不完整：缺少appId或accessToken")
		}
		if assetID == "" {
			return nil, fmt.Errorf("火山引擎克隆TTS配置不完整：缺少assetId")
		}
		providerName = base.TTS_VOLCENGINE_CLONE
		cluster := config.getString("cluster")
		if cluster == "" || cluster == "volcano_tts" {
			cluster = volcengineCloneCluster
		}
		encoding := config.getString("encoding")
		if encoding == "" {
			encoding = "pcm"
		}
		rate := int(config.getInt64("rate"))
		if rate == 0 {
			rate = int(config.getInt64("sampleRate"))
		}
		if rate == 0 {
			rate = int(config.getInt64("sample_rate"))
		}
		if rate == 0 {
			rate = 16000
		}
		sourceRate := int(config.getInt64("sourceRate"))
		if sourceRate == 0 {
			sourceRate = int(config.getInt64("source_rate"))
		}
		speedRatio := 1.0
		if speedStr := config.getString("speedRatio"); speedStr != "" {
			if f, err := strconv.ParseFloat(speedStr, 64); err == nil {
				speedRatio = f
			}
		}
		resourceID := config.getString("resourceId")
		if resourceID == "" {
			resourceID = config.getString("resource_id")
		}
		if resourceID == "" {
			resourceID = "seed-icl-2.0"
		}
		modelType := int(config.getInt64("modelType"))
		if modelType == 0 {
			modelType = int(config.getInt64("model_type"))
		}
		if modelType == 0 {
			modelType = 4
		}
		cloneOpt := VolcengineCloneOption{
			AppID:       appID,
			AccessToken: accessToken,
			Cluster:     cluster,
			ResourceID:  resourceID,
			ModelType:   modelType,
			AssetID:     assetID,
			Encoding:    encoding,
			Rate:        rate,
			SourceRate:  sourceRate,
			SpeedRatio:  speedRatio,
			Streaming:   config.getBool("streaming", true),
		}
		configBytes, err := json.Marshal(cloneOpt)
		if err != nil {
			return nil, fmt.Errorf("序列化火山引擎克隆配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化火山引擎克隆配置失败: %w", err)
		}

	case "minimax":
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		if apiKey == "" {
			return nil, fmt.Errorf("Minimax TTS配置不完整：缺少apiKey")
		}
		providerName = base.TTS_MINIMAX
		model := config.getString("model")
		if model == "" {
			model = "speech-2.5-turbo-preview" // 默认值
		}
		voiceID := config.getString("voiceId")
		if voiceID == "" {
			voiceID = config.getString("voice_id") // 兼容下划线格式
		}
		if voiceID == "" {
			voiceID = "male-qn-qingse" // 默认值
		}
		speedRatio := float64(1.0)
		if speedStr := config.getString("speedRatio"); speedStr != "" {
			if f, err := strconv.ParseFloat(speedStr, 64); err == nil {
				speedRatio = f
			}
		}
		volume := float64(1.0)
		if volStr := config.getString("volume"); volStr != "" {
			if f, err := strconv.ParseFloat(volStr, 64); err == nil {
				volume = f
			}
		}
		pitch := float64(0.0)
		if pitchStr := config.getString("pitch"); pitchStr != "" {
			if f, err := strconv.ParseFloat(pitchStr, 64); err == nil {
				pitch = f
			}
		}
		emotion := config.getString("emotion")
		if emotion == "" {
			emotion = "neutral" // 默认值
		}
		languageBoost := config.getString("languageBoost")
		if languageBoost == "" {
			languageBoost = config.getString("language_boost") // 兼容下划线格式
		}
		if languageBoost == "" {
			languageBoost = "auto" // 默认值
		}
		trainingTimes := int(1)
		if tt := config.getInt64("trainingTimes"); tt > 0 {
			trainingTimes = int(tt)
		}
		sampleRate := config.getInt64("sampleRate")
		if sampleRate == 0 {
			sampleRate = config.getInt64("sample_rate") // 兼容下划线格式
		}
		if sampleRate == 0 {
			sampleRate = 8000 // 默认值
		}
		bitrate := config.getInt64("bitrate")
		if bitrate == 0 {
			bitrate = 16 // 默认值
		}
		format := config.getString("format")
		if format == "" {
			format = "pcm" // 默认值
		}
		channels := config.getInt64("channels")
		if channels == 0 {
			channels = 1 // 默认值
		}
		frameDuration := config.getString("frameDuration")
		if frameDuration == "" {
			frameDuration = config.getString("frame_duration") // 兼容下划线格式
		}
		if frameDuration == "" {
			frameDuration = "20ms" // 默认值
		}
		minimaxConfig := ttsminimax.NewMinimaxOption(apiKey)
		minimaxConfig.Model = model
		minimaxConfig.VoiceID = voiceID
		minimaxConfig.SpeedRatio = speedRatio
		minimaxConfig.Volume = volume
		minimaxConfig.Pitch = pitch
		minimaxConfig.Emotion = emotion
		minimaxConfig.LanguageBoost = languageBoost
		minimaxConfig.TrainingTimes = trainingTimes
		minimaxConfig.SampleRate = int(sampleRate)
		minimaxConfig.Bitrate = int(bitrate)
		minimaxConfig.Format = format
		minimaxConfig.Channels = int(channels)
		minimaxConfig.FrameDuration = frameDuration
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(minimaxConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化Minimax配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化Minimax配置失败: %w", err)
		}

	case "elevenlabs":
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		if apiKey == "" {
			return nil, fmt.Errorf("ElevenLabs TTS配置不完整：缺少apiKey")
		}
		providerName = base.TTS_ELEVENLABS
		voiceID := config.getString("voiceId")
		if voiceID == "" {
			voiceID = config.getString("voice_id") // 兼容下划线格式
		}
		if voiceID == "" {
			voiceID = "21m00Tcm4TlvDq8ikWAM" // 默认值
		}
		modelID := config.getString("modelId")
		if modelID == "" {
			modelID = config.getString("model_id") // 兼容下划线格式
		}
		if modelID == "" {
			modelID = "eleven_monolingual_v1" // 默认值
		}
		languageCode := config.getString("languageCode")
		if languageCode == "" {
			languageCode = config.getString("language_code") // 兼容下划线格式
		}
		if languageCode == "" {
			languageCode = config.getString("language") // 也支持 language 字段
		}
		stability := float64(0.5)
		if stabStr := config.getString("stability"); stabStr != "" {
			if f, err := strconv.ParseFloat(stabStr, 64); err == nil {
				stability = f
			}
		}
		similarityBoost := float64(0.75)
		if simStr := config.getString("similarityBoost"); simStr != "" {
			if f, err := strconv.ParseFloat(simStr, 64); err == nil {
				similarityBoost = f
			}
		}
		style := float64(0.0)
		if styleStr := config.getString("style"); styleStr != "" {
			if f, err := strconv.ParseFloat(styleStr, 64); err == nil {
				style = f
			}
		}
		useSpeakerBoost := true
		if boostStr := config.getString("useSpeakerBoost"); boostStr != "" {
			useSpeakerBoost = boostStr == "true" || boostStr == "1"
		}
		elevenlabsConfig := ttselevenlabs.NewElevenLabsConfig(apiKey, voiceID)
		elevenlabsConfig.ModelID = modelID
		elevenlabsConfig.Stability = stability
		elevenlabsConfig.SimilarityBoost = similarityBoost
		elevenlabsConfig.Style = style
		elevenlabsConfig.UseSpeakerBoost = useSpeakerBoost
		// 设置语言代码（如果提供）
		if languageCode != "" {
			elevenlabsConfig.LanguageCode = languageCode
		}
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(elevenlabsConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化ElevenLabs配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化ElevenLabs配置失败: %w", err)
		}

	case "local":
		command := config.getString("command")
		if command == "" {
			command = "say" // 默认值
		}
		providerName = base.TTS_LOCAL
		voice := config.getString("voice")
		sampleRate := config.getInt64("sampleRate")
		if sampleRate == 0 {
			sampleRate = config.getInt64("sample_rate") // 兼容下划线格式
		}
		if sampleRate == 0 {
			sampleRate = 16000 // 默认值
		}
		channels := config.getInt64("channels")
		if channels == 0 {
			channels = 1 // 默认值
		}
		bitDepth := config.getInt64("bitDepth")
		if bitDepth == 0 {
			bitDepth = config.getInt64("bit_depth") // 兼容下划线格式
		}
		if bitDepth == 0 {
			bitDepth = 16 // 默认值
		}
		codec := config.getString("codec")
		if codec == "" {
			codec = "wav" // 默认值
		}
		outputDir := config.getString("outputDir")
		if outputDir == "" {
			outputDir = config.getString("output_dir") // 兼容下划线格式
		}
		if outputDir == "" {
			outputDir = "/tmp" // 默认值
		}
		localConfig := ttslocal.NewLocalTTSConfig(command)
		localConfig.Voice = voice
		localConfig.SampleRate = int(sampleRate)
		localConfig.Channels = int(channels)
		localConfig.BitDepth = int(bitDepth)
		localConfig.Codec = codec
		localConfig.OutputDir = outputDir
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(localConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化本地TTS配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化本地TTS配置失败: %w", err)
		}

	case "fishspeech":
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		if apiKey == "" {
			return nil, fmt.Errorf("FishSpeech TTS配置不完整：缺少apiKey")
		}
		providerName = base.TTS_FISHSPEECH
		referenceID := config.getString("referenceId")
		if referenceID == "" {
			referenceID = config.getString("reference_id") // 兼容下划线格式
		}
		if referenceID == "" {
			referenceID = "default" // 默认值
		}
		latency := config.getString("latency")
		if latency == "" {
			latency = "normal" // 默认值
		}
		version := config.getString("version")
		if version == "" {
			version = "s1" // 默认值
		}
		sampleRate := config.getInt64("sampleRate")
		if sampleRate == 0 {
			sampleRate = config.getInt64("sample_rate") // 兼容下划线格式
		}
		if sampleRate == 0 {
			sampleRate = 24000 // 默认值
		}
		codec := config.getString("codec")
		if codec == "" {
			codec = "wav" // 默认值
		}
		fishspeechConfig := ttsfishspeech.NewFishSpeechConfig(apiKey, referenceID)
		fishspeechConfig.Latency = latency
		fishspeechConfig.Version = version
		fishspeechConfig.SampleRate = int(sampleRate)
		fishspeechConfig.Codec = codec
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(fishspeechConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化FishSpeech配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化FishSpeech配置失败: %w", err)
		}

	case "fishaudio":
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		if apiKey == "" {
			return nil, fmt.Errorf("Fish Audio TTS配置不完整：缺少apiKey")
		}
		providerName = base.TTS_FISHAUDIO
		referenceID := config.getString("referenceId")
		if referenceID == "" {
			referenceID = config.getString("reference_id") // 兼容下划线格式
		}
		model := config.getString("model")
		if model == "" {
			model = "s1" // 默认值
		}
		format := config.getString("format")
		if format == "" {
			format = "mp3" // 默认值
		}
		sampleRate := config.getInt64("sampleRate")
		if sampleRate == 0 {
			sampleRate = config.getInt64("sample_rate") // 兼容下划线格式
		}
		if sampleRate == 0 {
			if format == "opus" {
				sampleRate = 48000
			} else {
				sampleRate = 44100 // 默认值
			}
		}
		temperature := 0.7
		if tempStr := config.getString("temperature"); tempStr != "" {
			if f, err := strconv.ParseFloat(tempStr, 64); err == nil {
				temperature = f
			}
		}
		topP := 0.7
		if topPStr := config.getString("topP"); topPStr != "" {
			if f, err := strconv.ParseFloat(topPStr, 64); err == nil {
				topP = f
			}
		}
		latency := config.getString("latency")
		if latency == "" {
			latency = "normal" // 默认值
		}
		chunkLength := config.getInt64("chunkLength")
		if chunkLength == 0 {
			chunkLength = config.getInt64("chunk_length") // 兼容下划线格式
		}
		if chunkLength == 0 {
			chunkLength = 300 // 默认值
		}
		normalize := true
		if normStr := config.getString("normalize"); normStr != "" {
			normalize = normStr == "true" || normStr == "1"
		}
		mp3Bitrate := config.getInt64("mp3Bitrate")
		if mp3Bitrate == 0 {
			mp3Bitrate = config.getInt64("mp3_bitrate") // 兼容下划线格式
		}
		if mp3Bitrate == 0 {
			mp3Bitrate = 128 // 默认值
		}
		fishaudioConfig := ttsfishaudio.NewFishAudioConfig(apiKey, referenceID)
		fishaudioConfig.Model = model
		fishaudioConfig.Format = format
		fishaudioConfig.SampleRate = int(sampleRate)
		fishaudioConfig.Temperature = temperature
		fishaudioConfig.TopP = topP
		fishaudioConfig.Latency = latency
		fishaudioConfig.ChunkLength = int(chunkLength)
		fishaudioConfig.Normalize = normalize
		fishaudioConfig.MPEGBitrate = int(mp3Bitrate)
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(fishaudioConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化Fish Audio配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化Fish Audio配置失败: %w", err)
		}

	case "coqui":
		url := config.getString("url")
		if url == "" {
			return nil, fmt.Errorf("Coqui TTS配置不完整：缺少url")
		}
		providerName = base.TTS_COQUI
		language := config.getString("language")
		if language == "" {
			language = "en_US" // 默认值
		}
		speaker := config.getString("speaker")
		if speaker == "" {
			speaker = "p226" // 默认值
		}
		sampleRate := config.getInt64("sampleRate")
		if sampleRate == 0 {
			sampleRate = config.getInt64("sample_rate") // 兼容下划线格式
		}
		if sampleRate == 0 {
			sampleRate = 16000 // 默认值
		}
		channels := config.getInt64("channels")
		if channels == 0 {
			channels = 1 // 默认值
		}
		bitDepth := config.getInt64("bitDepth")
		if bitDepth == 0 {
			bitDepth = config.getInt64("bit_depth") // 兼容下划线格式
		}
		if bitDepth == 0 {
			bitDepth = 16 // 默认值
		}
		coquiConfig := ttscoqui.NewCoquiTTSOption(url)
		coquiConfig.Language = language
		coquiConfig.Speaker = speaker
		coquiConfig.SampleRate = int(sampleRate)
		coquiConfig.Channels = int(channels)
		coquiConfig.BitDepth = int(bitDepth)
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(coquiConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化Coqui配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化Coqui配置失败: %w", err)
		}

	case "local_gospeech":
		provider := config.getString("provider")
		if provider == "" {
			provider = "melotts" // 默认使用 MeloTTS
		}
		modelPath := config.getString("modelPath")
		if modelPath == "" {
			modelPath = config.getString("model_path") // 兼容下划线格式
		}
		if modelPath == "" {
			return nil, fmt.Errorf("本地go-speech TTS配置不完整：缺少modelPath")
		}

		providerName = base.TTS_LOCAL_GOSPEECH
		language := config.getString("language")
		if language == "" {
			language = "zh-CN" // 默认值
		}
		speaker := config.getString("speaker")
		if speaker == "" {
			speaker = "default" // 默认值
		}
		sampleRate := config.getInt64("sampleRate")
		if sampleRate == 0 {
			sampleRate = config.getInt64("sample_rate") // 兼容下划线格式
		}
		if sampleRate == 0 {
			sampleRate = 16000 // 默认值
		}
		channels := config.getInt64("channels")
		if channels == 0 {
			channels = 1 // 默认值
		}
		bitDepth := config.getInt64("bitDepth")
		if bitDepth == 0 {
			bitDepth = config.getInt64("bit_depth") // 兼容下划线格式
		}
		if bitDepth == 0 {
			bitDepth = 16 // 默认值
		}
		speed := float32(1.0)
		if speedStr := config.getString("speed"); speedStr != "" {
			if f, err := strconv.ParseFloat(speedStr, 32); err == nil {
				speed = float32(f)
			}
		}
		pitch := float32(1.0)
		if pitchStr := config.getString("pitch"); pitchStr != "" {
			if f, err := strconv.ParseFloat(pitchStr, 32); err == nil {
				pitch = float32(f)
			}
		}
		volume := float32(1.0)
		if volumeStr := config.getString("volume"); volumeStr != "" {
			if f, err := strconv.ParseFloat(volumeStr, 32); err == nil {
				volume = float32(f)
			}
		}
		enableCache := true
		if cacheStr := config.getString("enableCache"); cacheStr != "" {
			enableCache = cacheStr == "true" || cacheStr == "1"
		}
		cacheExpiry := 24 * time.Hour
		if expiryStr := config.getString("cacheExpiry"); expiryStr != "" {
			if d, err := time.ParseDuration(expiryStr); err == nil {
				cacheExpiry = d
			}
		}

		localGoSpeechConfig := ttslocal.NewLocalGoSpeechConfig(ttslocal.LocalGoSpeechProvider(provider), modelPath)
		localGoSpeechConfig.Language = language
		localGoSpeechConfig.Speaker = speaker
		localGoSpeechConfig.SampleRate = int(sampleRate)
		localGoSpeechConfig.Channels = int(channels)
		localGoSpeechConfig.BitDepth = int(bitDepth)
		localGoSpeechConfig.Speed = speed
		localGoSpeechConfig.Pitch = pitch
		localGoSpeechConfig.Volume = volume
		localGoSpeechConfig.EnableCache = enableCache
		localGoSpeechConfig.CacheExpiry = cacheExpiry

		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(localGoSpeechConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化本地go-speech配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化本地go-speech配置失败: %w", err)
		}

	case "aliyun":
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		if apiKey == "" {
			return nil, fmt.Errorf("阿里云TTS配置不完整：缺少apiKey")
		}
		providerName = base.TTS_ALIYUN
		baseURL := config.getString("baseUrl")
		if baseURL == "" {
			baseURL = config.getString("base_url") // 兼容下划线格式
		}
		model := config.getString("model")
		if model == "" {
			model = "qwen3-tts-flash-realtime" // 默认值
		}
		voice := config.getString("voice")
		if voice == "" {
			voice = "Cherry" // 默认值
		}
		languageType := config.getString("languageType")
		if languageType == "" {
			languageType = config.getString("language_type") // 兼容下划线格式
		}
		if languageType == "" {
			languageType = "Auto" // 默认值
		}
		mode := config.getString("mode")
		if mode == "" {
			mode = "server_commit" // 默认值
		}
		sampleRate := config.getInt64("sampleRate")
		if sampleRate == 0 {
			sampleRate = config.getInt64("sample_rate") // 兼容下划线格式
		}
		if sampleRate == 0 {
			sampleRate = 24000 // 默认值
		}
		channels := config.getInt64("channels")
		if channels == 0 {
			channels = 1 // 默认值
		}
		bitDepth := config.getInt64("bitDepth")
		if bitDepth == 0 {
			bitDepth = config.getInt64("bit_depth") // 兼容下划线格式
		}
		if bitDepth == 0 {
			bitDepth = 16 // 默认值
		}
		frameDuration := config.getString("frameDuration")
		if frameDuration == "" {
			frameDuration = config.getString("frame_duration") // 兼容下划线格式
		}
		if frameDuration == "" {
			frameDuration = "20ms" // 默认值
		}
		dialTimeoutMs := config.getInt64("dialTimeoutMs")
		if dialTimeoutMs == 0 {
			dialTimeoutMs = config.getInt64("dial_timeout_ms") // 兼容下划线格式
		}
		if dialTimeoutMs == 0 {
			dialTimeoutMs = 10000 // 默认值
		}
		instructions := config.getString("instructions")
		optimizeInstructions := false
		if optimizeStr := config.getString("optimizeInstructions"); optimizeStr != "" {
			optimizeInstructions = optimizeStr == "true" || optimizeStr == "1"
		}
		aliyunConfig := ttsaliyun.NewAliyunTTSConfig(apiKey)
		aliyunConfig.BaseURL = baseURL
		aliyunConfig.Model = model
		aliyunConfig.Voice = voice
		aliyunConfig.LanguageType = languageType
		aliyunConfig.Mode = mode
		aliyunConfig.SampleRate = int(sampleRate)
		aliyunConfig.Channels = int(channels)
		aliyunConfig.BitDepth = int(bitDepth)
		aliyunConfig.FrameDuration = frameDuration
		aliyunConfig.DialTimeoutMs = int(dialTimeoutMs)
		aliyunConfig.Instructions = instructions
		aliyunConfig.OptimizeInstructions = optimizeInstructions
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(aliyunConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化阿里云TTS配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化阿里云TTS配置失败: %w", err)
		}

	default:
		return nil, fmt.Errorf("不支持的TTS provider: %s", provider)
	}


	return newEngineFromCredential(providerName, options)
}
