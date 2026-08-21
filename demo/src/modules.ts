// Auto-generated module catalog for ling-base
// This data describes all modules, their categories, and usage examples.

export interface ModuleInfo {
  name: string
  path: string
  category: string
  description: string
  providers?: string[]
  example?: string
  link?: string
}

export interface Category {
  name: string
  icon: string
  description: string
  modules: ModuleInfo[]
}

export const categories: Category[] = [
  {
    name: 'AI 中继',
    icon: '🤖',
    description: '统一 40+ AI provider 的中继层，支持 Chat/Embed/Image/Audio/Responses',
    modules: [
      {
        name: 'relay',
        path: 'github.com/LingByte/ling-base/relay',
        category: 'AI 中继',
        description: '生产级 AI provider 中继库，40+ channel 适配器，支持重试/熔断/降级',
        providers: ['OpenAI', 'Claude', 'Gemini', 'AWS', 'Vertex', 'Ali', 'Baidu', 'Tencent', 'Xunfei', 'Zhipu', 'Moonshot', 'DeepSeek', 'Cohere', 'Mistral', 'Perplexity', 'xAI', 'Coze', 'Dify', 'Cloudflare', 'Replicate', 'Minimax', 'Volcengine', 'Siliconflow', '...'],
        example: `import "github.com/LingByte/ling-base/relay"

client := relay.New(
    relay.WithProvider(openai.NewProvider("sk-xxx")),
    relay.WithRetry(retry.WithMaxAttempts(3)),
    relay.WithCircuitBreaker(cb),
    relay.WithFallback(relay.FallbackConfig{
        FallbackModels: []string{"gpt-3.5-turbo"},
    }),
)

// Chat
resp, err := client.Chat(ctx, &relay.ChatRequest{
    Model: "gpt-4",
    Messages: []relay.Message{{Role: "user", Content: "Hello"}},
})

// Streaming
result, err := client.ChatStream(ctx, &relay.ChatRequest{...})`,
      },
    ],
  },
  {
    name: '语音处理',
    icon: '🎙️',
    description: 'TTS 语音合成、ASR 语音识别、实时多模态对话',
    modules: [
      {
        name: 'voice/synthesizer',
        path: 'github.com/LingByte/ling-base/voice/synthesizer',
        category: '语音处理',
        description: 'TTS 语音合成，16 个 provider',
        providers: ['Aliyun', 'Baidu', 'Qcloud', 'Volcengine', 'Xunfei', 'AWS', 'Azure', 'Google', 'OpenAI', 'ElevenLabs', 'FishAudio', 'FishSpeech', 'Coqui', 'MiniMax', 'Qiniu', 'Local'],
      },
      {
        name: 'voice/recognizer',
        path: 'github.com/LingByte/ling-base/voice/recognizer',
        category: '语音处理',
        description: 'ASR 语音识别，12 个 provider',
        providers: ['Aliyun', 'Baidu', 'Qcloud', 'Volcengine', 'AWS', 'Google', 'Deepgram', 'Gladia', 'FunASR', 'Whisper', 'VoiceAPI', 'Local'],
      },
      {
        name: 'voice/realtime',
        path: 'github.com/LingByte/ling-base/voice/realtime',
        category: '语音处理',
        description: '实时多模态语音对话，8 个 provider',
        providers: ['OpenAI', 'Gemini', 'AliyunOmni', 'VolcDialogue', 'Iflytek', 'MiniMax', 'Stepfun', 'TencentSTS'],
      },
    ],
  },
  {
    name: '安全认证',
    icon: '🔐',
    description: 'TOTP 两步验证、Passkey 无密码认证、密码哈希',
    modules: [
      {
        name: 'common/totp',
        path: 'github.com/LingByte/ling-base/common/totp',
        category: '安全认证',
        description: 'TOTP 两步验证 + QR 码生成',
        example: `import "github.com/LingByte/ling-base/common/totp"

secret := totp.GenerateSecret()
uri := totp.KeyURI("user@example.com", "MyApp", secret)
qrPNG := totp.QRCodePNG(uri, 256, 256)
valid := totp.Validate(secret, "123456", 30, 6)`,
      },
      {
        name: 'common/passkey',
        path: 'github.com/LingByte/ling-base/common/passkey',
        category: '安全认证',
        description: 'Passkey/WebAuthn 无密码认证',
      },
      {
        name: 'common/password',
        path: 'github.com/LingByte/ling-base/common/password',
        category: '安全认证',
        description: '密码哈希（argon2/bcrypt/scrypt）',
      },
      {
        name: 'common/jwtutil',
        path: 'github.com/LingByte/ling-base/common/jwtutil',
        category: '安全认证',
        description: 'JWT 生成与验证',
      },
    ],
  },
  {
    name: '基础设施',
    icon: '⚙️',
    description: '限流、熔断、重试、缓存、锁、布隆过滤器',
    modules: [
      {
        name: 'common/limiter',
        path: 'github.com/LingByte/ling-base/common/limiter',
        category: '基础设施',
        description: '限流器：令牌桶/滑动窗口/并发数（tokenbucket/count/memory/redis）',
        providers: ['TokenBucket', 'BlockingBucket', 'AtomicBucket', 'Count', 'Memory', 'Redis', 'Etcd', 'Zookeeper', 'Consul'],
      },
      {
        name: 'common/circuitbreaker',
        path: 'github.com/LingByte/ling-base/common/circuitbreaker',
        category: '基础设施',
        description: '熔断器（Closed/Open/HalfOpen 三态）',
      },
      {
        name: 'common/retry',
        path: 'github.com/LingByte/ling-base/common/retry',
        category: '基础设施',
        description: '重试策略（指数退避/固定间隔 + 熔断器集成）',
      },
      {
        name: 'common/cache',
        path: 'github.com/LingByte/ling-base/common/cache',
        category: '基础设施',
        description: '缓存接口 + 多种实现',
        providers: ['LRU', 'Memory', 'Noop', 'MultiLevel', 'BigCache', 'Redis', 'Memcache', 'FreeCache', 'Ristretto'],
      },
      {
        name: 'common/lock',
        path: 'github.com/LingByte/ling-base/common/lock',
        category: '基础设施',
        description: '分布式锁',
        providers: ['Memory', 'Redis', 'RedLock', 'Etcd', 'Zookeeper', 'Consul', 'MySQL', 'Postgres'],
      },
      {
        name: 'common/bloom',
        path: 'github.com/LingByte/ling-base/common/bloom',
        category: '基础设施',
        description: '布隆过滤器',
        providers: ['Memory', 'Counting', 'Scalable', 'Redis', 'RedisBloom'],
      },
    ],
  },
  {
    name: '第三方服务',
    icon: '🔌',
    description: 'OCR、内容审核、对象存储、消息队列',
    modules: [
      {
        name: 'providers/ocr',
        path: 'github.com/LingByte/ling-base/providers/ocr',
        category: '第三方服务',
        description: 'OCR 光学字符识别，6 个 provider',
        providers: ['AWS', 'Google', 'Aliyun', 'Baidu', 'Azure', 'Qcloud'],
      },
      {
        name: 'providers/censor',
        path: 'github.com/LingByte/ling-base/providers/censor',
        category: '第三方服务',
        description: '内容审核，3 个 provider',
        providers: ['Aliyun', 'Qcloud', 'Qiniu'],
      },
      {
        name: 'stores',
        path: 'github.com/LingByte/ling-base/stores',
        category: '第三方服务',
        description: '对象存储，9 个 provider',
        providers: ['Local', 'S3', 'OSS', 'COS', 'MinIO', 'Kodo', 'TOS', 'OBS', 'KS3'],
      },
      {
        name: 'common/mq',
        path: 'github.com/LingByte/ling-base/common/mq',
        category: '第三方服务',
        description: '消息队列，5 个 broker',
        providers: ['Kafka', 'RabbitMQ', 'ActiveMQ', 'RocketMQ', 'RedisStream'],
      },
    ],
  },
  {
    name: '通用工具',
    icon: '🛠️',
    description: '日志、配置、压缩、二维码、拼音、IP 查询等 50+ 工具包',
    modules: [
      {
        name: 'common/logger',
        path: 'github.com/LingByte/ling-base/common/logger',
        category: '通用工具',
        description: '结构化日志（zap + lumberjack）',
      },
      {
        name: 'common/compress',
        path: 'github.com/LingByte/ling-base/common/compress',
        category: '通用工具',
        description: '压缩工具（zstd/snappy/lz4）',
      },
      {
        name: 'common/geoip',
        path: 'github.com/LingByte/ling-base/common/geoip',
        category: '通用工具',
        description: 'IP 地理位置查询（国内 pconline + 国际 ip-api）',
      },
      {
        name: 'common/phone',
        path: 'github.com/LingByte/ling-base/common/phone',
        category: '通用工具',
        description: '手机号归属地查询（内置离线号段库）',
      },
      {
        name: 'common/i18n',
        path: 'github.com/LingByte/ling-base/common/i18n',
        category: '通用工具',
        description: '国际化（翻译 + 格式化 + locale 检测）',
      },
      {
        name: 'common/qrcode',
        path: 'github.com/LingByte/ling-base/common/qrcode',
        category: '通用工具',
        description: '二维码生成',
      },
      {
        name: 'common/pinyin',
        path: 'github.com/LingByte/ling-base/common/pinyin',
        category: '通用工具',
        description: '拼音转换',
      },
    ],
  },
]

export const allModules = categories.flatMap(c => c.modules)
