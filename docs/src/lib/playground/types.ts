export type DemoId =
  | 'totp'
  | 'compress'
  | 'hash'
  | 'password'
  | 'validate'
  | 'jwt'
  | 'qrcode'
  | 'barcode'
  | 'idgen'
  | 'random'
  | 'pinyin'
  | 'phone'
  | 'convert'
  | 'crypto'
  | 'captcha'
  | 'nltime'
  | 'bloom'
  | 'mathutil'
  | 'netutil'
  | 'i18n'
  | 'timeutil'
  | 'circuitbreaker'
  | 'retry'
  | 'chat'
  | 'chat-stream'
  | 'realtime'
  | 'embed'
  | 'limiter';

export type DemoKind = 'wasm' | 'api' | 'client';

export interface DemoMeta {
  id: DemoId;
  kind: DemoKind;
  title: string;
  description: string;
}

export const DEMO_META: Record<DemoId, DemoMeta> = {
  totp: { id: 'totp', kind: 'wasm', title: 'TOTP 两步验证', description: '验证码生成、校验与备份码' },
  compress: { id: 'compress', kind: 'wasm', title: '数据压缩', description: 'zstd / snappy / lz4 / gzip' },
  hash: { id: 'hash', kind: 'wasm', title: '哈希计算', description: 'MD5 / SHA-256 / HMAC-SHA256 等' },
  password: { id: 'password', kind: 'wasm', title: '密码哈希', description: 'Argon2id / Bcrypt 哈希与校验' },
  validate: { id: 'validate', kind: 'wasm', title: '数据校验', description: 'email / min / max / required 等规则' },
  jwt: { id: 'jwt', kind: 'wasm', title: 'JWT 令牌', description: '签发与验证 Access Token' },
  qrcode: { id: 'qrcode', kind: 'wasm', title: '二维码生成', description: '标准 / 花式 QR（形状、颜色、渐变）' },
  barcode: { id: 'barcode', kind: 'wasm', title: '条形码生成', description: 'Code128 / EAN-13 / DataMatrix 等' },
  idgen: { id: 'idgen', kind: 'wasm', title: 'ID 生成', description: 'UUID / Snowflake / ShortID' },
  random: { id: 'random', kind: 'wasm', title: '随机数', description: '字符串 / 密码 / 颜色 / UUID' },
  pinyin: { id: 'pinyin', kind: 'wasm', title: '拼音转换', description: '中文转拼音' },
  phone: { id: 'phone', kind: 'wasm', title: '手机号归属地', description: '号段查询省份 / 运营商' },
  convert: { id: 'convert', kind: 'wasm', title: '格式转换', description: 'JSON ↔ YAML ↔ TOML' },
  crypto: { id: 'crypto', kind: 'wasm', title: 'AES 加解密', description: 'AES-GCM Base64' },
  captcha: { id: 'captcha', kind: 'wasm', title: '验证码（6 种）', description: 'Image / Click / Slider / Math / Jigsaw / Rotate' },
  nltime: { id: 'nltime', kind: 'wasm', title: '自然语言时间', description: '解析 tomorrow / 3 days ago' },
  bloom: { id: 'bloom', kind: 'wasm', title: '布隆过滤器', description: '参数估算与成员测试' },
  mathutil: { id: 'mathutil', kind: 'wasm', title: '统计计算', description: 'mean / median / stdDev / p95' },
  netutil: { id: 'netutil', kind: 'wasm', title: 'IP 判断', description: 'private / loopback / public' },
  i18n: { id: 'i18n', kind: 'wasm', title: '国际化', description: '多语言 key 翻译' },
  timeutil: { id: 'timeutil', kind: 'wasm', title: '时间工具', description: '时区格式化 / 日界' },
  circuitbreaker: { id: 'circuitbreaker', kind: 'client', title: '熔断器', description: '失败阈值模拟' },
  retry: { id: 'retry', kind: 'client', title: '重试策略', description: '重试次数模拟' },
  chat: { id: 'chat', kind: 'client', title: 'Chat 对话', description: 'OpenAI 兼容 API 非流式调用' },
  'chat-stream': { id: 'chat-stream', kind: 'client', title: '流式 Chat', description: 'OpenAI 兼容 API 流式输出' },
  realtime: { id: 'realtime', kind: 'client', title: 'voice/realtime 语音对话', description: 'WebSocket + PCM（阿里云 Qwen-Omni / OpenAI）' },
  embed: { id: 'embed', kind: 'client', title: 'Embeddings', description: '文本向量化' },
  limiter: { id: 'limiter', kind: 'client', title: '令牌桶限流', description: '模拟限流器行为' },
};

export const WASM_DEMOS = new Set<DemoId>(
  Object.values(DEMO_META)
    .filter((d) => d.kind === 'wasm')
    .map((d) => d.id),
);

export function demoNeedsWasm(id: DemoId): boolean {
  return WASM_DEMOS.has(id);
}
