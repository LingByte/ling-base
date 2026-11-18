package recognizerstd

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"strconv"
	"strings"

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
)

// ConfigReader reads string/int fields from a credential map.
type ConfigReader struct {
	config map[string]interface{}
}

// NewConfigReader creates a ConfigReader.
func NewConfigReader(config map[string]interface{}) *ConfigReader {
	return &ConfigReader{config: config}
}

// String returns the first non-empty string value for the given keys; the last
// argument may be used as a default when it is not a config key.
func (r *ConfigReader) String(keysAndDefault ...string) string {
	if len(keysAndDefault) == 0 {
		return ""
	}
	defaultValue := ""
	keys := keysAndDefault
	if len(keysAndDefault) > 1 {
		lastKey := keysAndDefault[len(keysAndDefault)-1]
		if _, exists := r.config[lastKey]; !exists {
			defaultValue = lastKey
			keys = keysAndDefault[:len(keysAndDefault)-1]
		}
	}
	for _, key := range keys {
		if val, ok := r.config[key].(string); ok && val != "" {
			return val
		}
	}
	return defaultValue
}

// Int returns the first matching int value or defaultValue.
func (r *ConfigReader) Int(keys ...interface{}) int {
	var defaultValue int
	var keyStrings []string
	for _, k := range keys {
		switch v := k.(type) {
		case string:
			keyStrings = append(keyStrings, v)
		case int:
			defaultValue = v
		}
	}
	for _, key := range keyStrings {
		if val, ok := r.config[key]; ok {
			switch v := val.(type) {
			case int:
				return v
			case int64:
				return int(v)
			case float64:
				return int(v)
			case string:
				if intVal, err := strconv.Atoi(v); err == nil {
					return intVal
				}
			}
		}
	}
	return defaultValue
}

// NewTranscriberConfigFromMap builds a vendor TranscriberConfig from tenant credentials.
func NewTranscriberConfigFromMap(
	provider string,
	config map[string]interface{},
	language string,
) (base.TranscriberConfig, error) {
	switch provider {
	case "tencent", "qcloud":
		return buildQCloudConfig(config)
	case "google":
		return buildGoogleConfig(config, language)
	case "funasr":
		return buildFunASRConfig(config)
	case "funasr_realtime":
		return buildFunASRRealtimeConfig(config)
	case "volcengine":
		return buildVolcengineConfig(config)
	case "volcllmasr", "volcengine_llm":
		return buildVolcengineLLMConfig(config)
	case "gladia":
		return buildGladiaConfig(config)
	case "deepgram":
		return buildDeepgramConfig(config, language)
	case "aws":
		return buildAwsConfig(config, language)
	case "baidu":
		return buildBaiduConfig(config)
	case "voiceapi":
		return buildVoiceAPIConfig(config)
	case "whisper":
		return buildWhisperConfig(config)
	case "local":
		return buildLocalConfig(config)
	default:
		return nil, fmt.Errorf("unsupported ASR provider: %s", provider)
	}
}

func buildQCloudConfig(config map[string]interface{}) (*asrvqcloud.QCloudASROption, error) {
	cfg := NewConfigReader(config)
	appID := cfg.String("app_id", "appId")
	secretID := cfg.String("secret_id", "secretId")
	secretKey := cfg.String("secret_key", "secretKey", "secret")
	if appID == "" || secretID == "" || secretKey == "" {
		return nil, fmt.Errorf("腾讯云ASR配置不完整：缺少appId、secretId或secretKey (配置: appId=%s, secretId=%s, secretKey=%s)",
			appID, secretID, secretKey)
	}
	opt := asrvqcloud.NewQcloudASROption(appID, secretID, secretKey)
	if mt := cfg.String("modelType", "model_type"); mt != "" {
		opt.ModelType = mt
	}
	if v := cfg.Int("vadSilenceTime", "vad_silence_time", 0); v > 0 {
		opt.VadSilenceTime = v
	}
	if _, ok := config["needVad"]; ok {
		opt.NeedVad = cfg.Int("needVad", 0)
	} else if _, ok := config["need_vad"]; ok {
		opt.NeedVad = cfg.Int("need_vad", 0)
		if opt.NeedVad == 0 {
			opt.NeedVad = -1
		}
	}
	return &opt, nil
}

func buildGoogleConfig(config map[string]interface{}, language string) (*asrvgoogle.GoogleASROption, error) {
	cfg := NewConfigReader(config)
	credentialsJSON := cfg.String("credentialsJson", "credentials_json", "credentials")
	if credentialsJSON == "" {
		credentialsJSON = cfg.String("apiKey", "api_key")
	}
	if credentialsJSON == "" {
		return nil, fmt.Errorf("Google ASR配置不完整：缺少 credentials_json 或 apiKey")
	}
	languageCode := cfg.String("languageCode", "language_code", language)
	if languageCode == "" {
		languageCode = "zh-CN"
	}
	opt := asrvgoogle.NewGoogleASROption(credentialsJSON)
	opt.LanguageCode = languageCode
	if enc := cfg.String("encoding"); enc != "" {
		opt.Encoding = enc
	}
	if sr := cfg.Int("sampleRate", "sample_rate", 0); sr > 0 {
		opt.SampleRate = sr
	}
	return &opt, nil
}

func buildFunASRConfig(config map[string]interface{}) (*asrvfunasr.FunASROption, error) {
	cfg := NewConfigReader(config)
	url := cfg.String("url", "wss://dashscope.aliyuncs.com/api-ws/v1/inference")
	opt := asrvfunasr.NewFunASROption(url)
	if apiKey := cfg.String("apiKey", "api_key"); apiKey != "" {
		// FunASR option has no ApiKey field; callers may embed in URL query if needed.
		_ = apiKey
	}
	return &opt, nil
}

func buildFunASRRealtimeConfig(config map[string]interface{}) (base.TranscriberConfig, error) {
	return nil, fmt.Errorf("funasr_realtime is not yet implemented in ling-base recognizer")
}

func buildVolcengineConfig(config map[string]interface{}) (*asrvvolcengine.VolcengineOption, error) {
	cfg := NewConfigReader(config)
	url := cfg.String("url", "wss://openspeech.bytedance.com/api/v2/asr")
	appID := cfg.String("appId", "app_id")
	token := cfg.String("token")
	cluster := cfg.String("cluster", "volcengine_input_common")
	format := cfg.String("format", "raw")
	if appID == "" || token == "" {
		return nil, fmt.Errorf("火山引擎ASR配置不完整：缺少appId或token")
	}
	opt := asrvvolcengine.NewVolcengineOption(appID, token, cluster, format)
	opt.Url = url
	if v := cfg.Int("endWindowSize", "end_window_size", "vadSilenceTime", "vad_silence_time", 0); v > 0 {
		opt.EndWindowSize = v
	}
	return &opt, nil
}

func buildVolcengineLLMConfig(config map[string]interface{}) (*asrvvolcenginellm.VolcengineLLMOption, error) {
	cfg := NewConfigReader(config)
	token := cfg.String("token")
	appID := cfg.String("appId", "app_id")
	if token == "" || appID == "" {
		return nil, fmt.Errorf("火山引擎LLM ASR配置不完整：缺少token或appId")
	}
	opt := asrvvolcenginellm.NewVolcengineLLMOption(token, appID)
	if v := cfg.Int("endWindowSize", "end_window_size", "vadSilenceTime", "vad_silence_time", 0); v > 0 {
		opt.EndWindowSize = v
	}
	if rid := cfg.String("resourceId", "resource_id"); rid != "" {
		opt.ResourceId = rid
	}
	return &opt, nil
}

func buildGladiaConfig(config map[string]interface{}) (*asrvgladia.GladiaASROption, error) {
	cfg := NewConfigReader(config)
	apiKey := cfg.String("apiKey", "api_key")
	encoding := cfg.String("encoding", "WAV/PCM")
	if apiKey == "" {
		return nil, fmt.Errorf("Gladia ASR配置不完整：缺少apiKey")
	}
	opt := asrvgladia.NewGladiaASROption(apiKey)
	opt.Encoding = encoding
	return &opt, nil
}

func buildDeepgramConfig(config map[string]interface{}, language string) (*asrvdeepgram.DeepgramASROption, error) {
	cfg := NewConfigReader(config)
	apiKey := cfg.String("apiKey", "api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("Deepgram ASR配置不完整：缺少apiKey")
	}
	model := cfg.String("model", "nova-2")
	lang := cfg.String("language", language)
	if lang == "" {
		lang = "en-US"
	}
	opt := asrvdeepgram.NewDeepgramASROption(apiKey)
	opt.Model = model
	opt.Language = lang
	return &opt, nil
}

func buildAwsConfig(config map[string]interface{}, language string) (*asrvaws.AWSASROption, error) {
	cfg := NewConfigReader(config)
	accessKey := cfg.String("accessKeyId", "access_key_id", "appId", "app_id")
	secretKey := cfg.String("secretAccessKey", "secret_access_key", "secretKey", "secret_key")
	region := cfg.String("region", "us-east-1")
	lang := cfg.String("language", language)
	if lang == "" {
		lang = "en-US"
	}
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("AWS ASR配置不完整：缺少 accessKeyId 或 secretAccessKey")
	}
	opt := asrvaws.NewAWSASROption(accessKey, secretKey, region)
	opt.LanguageCode = lang
	return &opt, nil
}

func buildBaiduConfig(config map[string]interface{}) (*asrvbaidu.BaiduASROption, error) {
	cfg := NewConfigReader(config)
	appKey := cfg.String("appKey", "app_key", "apiKey", "api_key")
	secretKey := cfg.String("secretKey", "secret_key")
	appID := cfg.Int("appId", "app_id", 0)
	devPid := cfg.Int("devPid", "dev_pid", 1537)
	format := cfg.String("format", "pcm")
	sampleRate := cfg.Int("sampleRate", "sample_rate", 16000)
	if appKey == "" {
		return nil, fmt.Errorf("百度ASR配置不完整：缺少appKey")
	}
	opt := asrvbaidu.NewBaiduASROption(appKey, secretKey)
	if appID > 0 {
		opt.AppID = strconv.Itoa(appID)
	}
	opt.DevPid = devPid
	opt.Format = format
	opt.Rate = sampleRate
	return &opt, nil
}

func buildVoiceAPIConfig(config map[string]interface{}) (*asrvvoiceapi.VoiceAPIASROption, error) {
	cfg := NewConfigReader(config)
	url := cfg.String("url", "")
	if url == "" {
		return nil, fmt.Errorf("VoiceAPI ASR配置不完整：缺少url")
	}
	apiKey := cfg.String("apiKey", "api_key")
	opt := asrvvoiceapi.NewVoiceAPIASROption(url, apiKey)
	return &opt, nil
}

func buildWhisperConfig(config map[string]interface{}) (*asrvwhisper.WhisperASROption, error) {
	cfg := NewConfigReader(config)
	url := cfg.String("url", "")
	model := cfg.String("model", "whisper-1")
	if url == "" {
		return nil, fmt.Errorf("Whisper ASR配置不完整：缺少url")
	}
	apiKey := cfg.String("apiKey", "api_key")
	opt := asrvwhisper.NewWhisperASROption(url, apiKey)
	opt.Model = model
	return &opt, nil
}

func buildLocalConfig(config map[string]interface{}) (*asrvlocal.LocalASROption, error) {
	cfg := NewConfigReader(config)
	command := cfg.String("command")
	modelPath := cfg.String("modelPath", "model_path", "model")
	opt := asrvlocal.NewLocalASROption(command)
	opt.Model = modelPath
	if strings.TrimSpace(opt.Command) == "" && strings.TrimSpace(opt.Model) != "" {
		opt.Command = asrvlocal.DetectLocalASRCommand()
	}
	return &opt, nil
}