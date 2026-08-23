import type { DemoId } from './types';

/** common/* 子模块 → 演示映射（51 个包全覆盖） */
const COMMON_MODULE_DEMOS: Record<string, DemoId[]> = {
  totp: ['totp'],
  compress: ['compress'],
  hash: ['hash'],
  password: ['password'],
  validate: ['validate'],
  jwtutil: ['jwt'],
  qrcode: ['qrcode', 'barcode'],
  barcode: ['barcode', 'qrcode'],
  captcha: ['captcha'],
  crypto: ['crypto', 'hash'],
  random: ['random', 'idgen'],
  convert: ['convert'],
  parser: ['convert'],
  phone: ['phone'],
  pinyin: ['pinyin'],
  bloom: ['bloom'],
  idgen: ['idgen', 'random'],
  nltime: ['nltime', 'timeutil'],
  timeutil: ['timeutil', 'nltime'],
  mathutil: ['mathutil'],
  netutil: ['netutil'],
  i18n: ['i18n'],
  limiter: ['limiter'],
  circuitbreaker: ['circuitbreaker', 'limiter'],
  retry: ['retry', 'circuitbreaker'],
  cache: ['bloom', 'limiter'],
  lock: ['limiter'],
  logger: ['validate'],
  config: ['convert'],
  middleware: ['jwt'],
  response: ['validate'],
  stats: ['mathutil'],
  metrics: ['mathutil', 'limiter'],
  scheduler: ['nltime'],
  cron: ['nltime'],
  queue: ['limiter'],
  pool: ['limiter'],
  eventbus: ['limiter'],
  mq: ['limiter'],
  notification: ['validate'],
  search: ['embed'],
  migration: ['convert'],
  opentelemetry: ['limiter'],
  tracing: ['limiter'],
  geoip: ['netutil', 'phone'],
  audioutil: ['realtime', 'chat'],
  videoutil: ['realtime'],
  imageutil: ['qrcode', 'captcha'],
  root: ['chat'],
  constants: ['validate'],
  system: ['idgen'],
  passkey: ['jwt'],
};

const EXACT: Record<string, DemoId[]> = {
  'relay/chat': ['chat', 'chat-stream', 'realtime'],
  'relay/streaming': ['chat-stream', 'realtime'],
  'relay/embeddings': ['embed'],
  'relay/image': ['chat'],
  'relay/audio': ['realtime', 'chat'],
  'relay/rerank': ['embed'],
  'relay/responses': ['chat-stream', 'realtime'],
  'relay/tasks': ['chat'],
  'relay/realtime': ['realtime', 'chat-stream'],
  'relay/providers': ['chat', 'embed', 'realtime'],
  'relay/production': ['limiter', 'circuitbreaker', 'retry'],
  'relay/index': ['chat', 'embed', 'realtime'],

  'agentkit/index': ['chat', 'realtime'],
  'agentkit/relaymodel': ['chat', 'realtime'],

  'voice/index': ['realtime', 'chat'],
  'voice/synthesizer': ['realtime'],
  'voice/recognizer': ['realtime'],
  'voice/realtime': ['realtime', 'chat-stream'],

  'security/totp': ['totp'],
  'security/password': ['password'],
  'security/jwt': ['jwt'],
  'security/passkey': ['jwt'],
  'security/index': ['totp', 'password', 'jwt', 'captcha'],

  'infrastructure/limiter': ['limiter'],
  'infrastructure/circuitbreaker': ['circuitbreaker'],
  'infrastructure/retry': ['retry'],
  'infrastructure/cache': ['bloom', 'limiter'],
  'infrastructure/lock': ['limiter'],
  'infrastructure/index': ['limiter', 'circuitbreaker', 'retry'],

  'providers/stores': ['qrcode'],
  'providers/ocr': ['chat'],
  'providers/censor': ['validate', 'captcha'],
  'providers/mq': ['limiter'],
  'providers/index': ['chat'],

  'common/captcha': ['captcha'],
  'common/qrcode': ['qrcode', 'barcode'],
  'common/barcode': ['barcode', 'qrcode'],
  'common/mathutil': ['mathutil'],
  'common/netutil': ['netutil'],
  'common/i18n': ['i18n'],
  'common/timeutil': ['timeutil'],

  'bootstrap/index': ['chat'],
  'apidocs/index': ['chat'],
  'version/index': ['idgen'],
  'pentest/index': ['jwt', 'crypto', 'captcha'],
  'lingcli/index': ['chat'],
  'lingcli/templates': ['chat'],
  'lingcli/modules': ['chat'],
  'example/index': ['chat', 'realtime'],
};

const PREFIX: [string, DemoId[]][] = [
  ['relay/', ['chat', 'realtime']],
  ['agentkit/', ['chat', 'realtime']],
  ['voice/', ['realtime']],
  ['security/', ['totp', 'jwt', 'captcha']],
  ['infrastructure/', ['limiter', 'circuitbreaker', 'retry']],
  ['providers/', ['chat']],
  ['bootstrap/', ['chat']],
  ['apidocs/', ['chat']],
  ['lingcli/', ['chat']],
  ['example/', ['chat', 'realtime']],
  ['pentest/', ['jwt', 'crypto']],
  ['version/', ['idgen']],
];

export function getDemosForSlug(slug: string[] | undefined): DemoId[] {
  if (!slug || slug.length === 0) {
    return ['chat', 'realtime', 'embed', 'captcha', 'qrcode'];
  }

  const path = slug.join('/');
  if (EXACT[path]) return dedupe(EXACT[path]);

  if (slug[0] === 'common' && slug[1]) {
    const moduleDemos = COMMON_MODULE_DEMOS[slug[1]];
    if (moduleDemos) return dedupe(moduleDemos);
    return ['hash', 'validate', 'idgen'];
  }

  for (const [prefix, demos] of PREFIX) {
    if (path.startsWith(prefix)) return dedupe(demos);
  }

  return ['chat'];
}

function dedupe(ids: DemoId[]): DemoId[] {
  return [...new Set(ids)];
}
