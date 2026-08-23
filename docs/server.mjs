/**
 * Next.js custom server with voice/realtime WebSocket proxy.
 * Supports openai_realtime + aliyun_omni (DashScope Qwen-Omni).
 */
import { createServer } from 'http';
import { parse } from 'url';
import next from 'next';
import { WebSocketServer, WebSocket } from 'ws';

const dev = process.env.NODE_ENV !== 'production';
const hostname = process.env.HOSTNAME || 'localhost';
const port = parseInt(process.env.PORT || '3000', 10);
const basePath = process.env.NEXT_PUBLIC_BASE_PATH || '';

const app = next({ dev, hostname, port });
const handle = app.getRequestHandler();

const REALTIME_PATHS = new Set([
  '/api/playground/realtime',
  `${basePath}/api/playground/realtime`.replace(/\/+/g, '/'),
]);

/** @type {Record<string, {
 *   label: string;
 *   resolveApiKey: (cfg: { apiKey?: string }) => string;
 *   keyHint: string;
 *   defaultModel: string;
 *   defaultVoice: string;
 *   buildUrl: (model: string) => string;
 *   headers: (apiKey: string) => Record<string, string>;
 *   buildSession: (voice: string, instructions: string) => object;
 * }>} */
const PROVIDERS = {
  aliyun_omni: {
    label: '阿里云 Qwen-Omni (DashScope)',
    resolveApiKey: (cfg) =>
      (cfg.apiKey?.trim()
        || process.env.DASHSCOPE_API_KEY?.trim()
        || process.env.ALIYUN_API_KEY?.trim()
        || process.env.PLAYGROUND_DASHSCOPE_API_KEY?.trim()
        || ''),
    keyHint: 'DASHSCOPE_API_KEY',
    defaultModel: 'qwen3.5-omni-flash-realtime-2026-03-15',
    defaultVoice: 'Tina',
    buildUrl: (model) =>
      `wss://dashscope.aliyuncs.com/api-ws/v1/realtime?model=${encodeURIComponent(model)}`,
    headers: (apiKey) => ({
      Authorization: `Bearer ${apiKey}`,
      'X-DashScope-OmniRealtime': 'true',
    }),
    buildSession: (voice, instructions) => ({
      type: 'session.update',
      session: {
        voice,
        modalities: ['text', 'audio'],
        input_audio_format: 'pcm',
        output_audio_format: 'pcm',
        input_audio_transcription: { model: 'gummy-realtime-v1' },
        turn_detection: {
          type: 'server_vad',
          threshold: 0.8,
          prefix_padding_ms: 500,
          silence_duration_ms: 1000,
          create_response: true,
          interrupt_response: true,
        },
        instructions,
      },
    }),
  },
  openai_realtime: {
    label: 'OpenAI Realtime',
    resolveApiKey: (cfg) =>
      (cfg.apiKey?.trim()
        || process.env.OPENAI_API_KEY?.trim()
        || process.env.PLAYGROUND_OPENAI_API_KEY?.trim()
        || ''),
    keyHint: 'OPENAI_API_KEY',
    defaultModel: 'gpt-4o-realtime-preview',
    defaultVoice: 'alloy',
    buildUrl: (model) =>
      `wss://api.openai.com/v1/realtime?model=${encodeURIComponent(model)}`,
    headers: (apiKey) => ({
      Authorization: `Bearer ${apiKey}`,
      'OpenAI-Beta': 'realtime=v1',
    }),
    buildSession: (voice, instructions) => ({
      type: 'session.update',
      session: {
        voice,
        modalities: ['text', 'audio'],
        input_audio_format: 'pcm16',
        output_audio_format: 'pcm16',
        input_audio_transcription: { model: 'whisper-1' },
        turn_detection: {
          type: 'server_vad',
          threshold: 0.5,
          prefix_padding_ms: 300,
          silence_duration_ms: 500,
          create_response: true,
          interrupt_response: true,
        },
        instructions,
      },
    }),
  },
};

const DEFAULT_PROVIDER = 'aliyun_omni';
const DEFAULT_INSTRUCTIONS =
  'You are a helpful assistant. Reply concisely in the same language as the user.';

function proxyRealtime(clientWs) {
  let upstream = null;
  let closed = false;

  const closeAll = (code, reason) => {
    if (closed) return;
    closed = true;
    try { clientWs.close(code, reason); } catch { /* ignore */ }
    try { upstream?.close(); } catch { /* ignore */ }
  };

  clientWs.once('message', (raw) => {
    let cfg;
    try {
      cfg = JSON.parse(raw.toString());
    } catch {
      clientWs.send(JSON.stringify({ type: 'error', error: { message: '首条消息须为 JSON 配置' } }));
      closeAll(1008, 'bad config');
      return;
    }

    const providerId = (cfg.provider || DEFAULT_PROVIDER).trim();
    const provider = PROVIDERS[providerId];
    if (!provider) {
      clientWs.send(JSON.stringify({
        type: 'error',
        error: { message: `未知 provider: ${providerId}，可选: ${Object.keys(PROVIDERS).join(', ')}` },
      }));
      closeAll(1008, 'bad provider');
      return;
    }

    const apiKey = provider.resolveApiKey(cfg);
    if (!apiKey) {
      clientWs.send(JSON.stringify({
        type: 'error',
        error: {
          message: `请填写 API Key，或在 docs/.env.local 配置 ${provider.keyHint}`,
        },
      }));
      closeAll(1008, 'no key');
      return;
    }

    const model = (cfg.model || provider.defaultModel).trim();
    const voice = (cfg.voice || provider.defaultVoice).trim();
    const instructions = (cfg.instructions || DEFAULT_INSTRUCTIONS).trim();
    const url = provider.buildUrl(model);

    upstream = new WebSocket(url, { headers: provider.headers(apiKey) });

    upstream.on('open', () => {
      upstream.send(JSON.stringify(provider.buildSession(voice, instructions)));
      clientWs.send(JSON.stringify({
        type: 'proxy.connected',
        provider: providerId,
        model,
        voice,
      }));
    });

    // DashScope often sends JSON as binary frames; always forward as UTF-8 text
    // so the browser can JSON.parse (Blob would otherwise be silently dropped).
    upstream.on('message', (data, isBinary) => {
      if (clientWs.readyState !== WebSocket.OPEN) return;
      if (isBinary || Buffer.isBuffer(data)) {
        const text = Buffer.isBuffer(data) ? data.toString('utf8') : Buffer.from(data).toString('utf8');
        clientWs.send(text);
      } else {
        clientWs.send(data);
      }
    });

    upstream.on('error', (err) => {
      if (clientWs.readyState === WebSocket.OPEN) {
        clientWs.send(JSON.stringify({ type: 'error', error: { message: String(err.message || err) } }));
      }
      closeAll(1011, 'upstream error');
    });

    upstream.on('close', (code, reasonBuf) => {
      const reason = reasonBuf?.toString?.() || '';
      if (clientWs.readyState === WebSocket.OPEN) {
        clientWs.send(JSON.stringify({
          type: 'error',
          error: { message: `上游 WebSocket 关闭 (code=${code}${reason ? `, ${reason}` : ''})` },
        }));
      }
      closeAll(1000, 'upstream closed');
    });

    clientWs.on('message', (data, isBinary) => {
      if (upstream?.readyState !== WebSocket.OPEN) return;
      // Client always sends JSON text; normalize binary→text for upstream
      if (isBinary || Buffer.isBuffer(data)) {
        const text = Buffer.isBuffer(data) ? data.toString('utf8') : Buffer.from(data).toString('utf8');
        upstream.send(text);
      } else {
        upstream.send(data);
      }
    });
  });

  clientWs.on('close', () => closeAll(1000, 'client closed'));
  clientWs.on('error', () => closeAll(1011, 'client error'));
}

await app.prepare();

const server = createServer((req, res) => {
  handle(req, res, parse(req.url, true));
});

const wss = new WebSocketServer({ noServer: true });
// Next.js HMR / Turbopack WS — must forward, otherwise /_next/hmr fails
const nextUpgrade = app.getUpgradeHandler();

server.on('upgrade', (request, socket, head) => {
  const { pathname } = parse(request.url);
  if (REALTIME_PATHS.has(pathname)) {
    wss.handleUpgrade(request, socket, head, (ws) => proxyRealtime(ws));
    return;
  }
  // Pass HMR and other Next.js upgrades through (do NOT socket.destroy)
  void nextUpgrade(request, socket, head);
});

server.listen(port, () => {
  console.log(`> Ready on http://${hostname}:${port}`);
  console.log(`> voice/realtime WS: ws://${hostname}:${port}/api/playground/realtime`);
  console.log(`> providers: ${Object.keys(PROVIDERS).join(', ')} (default: ${DEFAULT_PROVIDER})`);
});
