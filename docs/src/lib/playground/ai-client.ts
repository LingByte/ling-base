import { asset } from '@/lib/shared';

/** AI Playground：优先走同源服务端代理（绕过 CORS），仅在代理不可用时直连 */

export interface ChatMessage {
  role: 'system' | 'user' | 'assistant';
  content: string;
}

export interface ChatRequest {
  apiKey: string;
  baseUrl?: string;
  model: string;
  message?: string;
  messages?: ChatMessage[];
  system?: string;
  stream?: boolean;
}

export interface EmbedRequest {
  apiKey: string;
  baseUrl?: string;
  model: string;
  input: string;
}

function resolveBase(baseUrl?: string): string {
  return (baseUrl?.trim() || 'https://openrouter.ai/api/v1').replace(/\/$/, '');
}

function providerHeaders(apiKey: string): HeadersInit {
  return {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${apiKey.trim()}`,
    'HTTP-Referer': typeof window !== 'undefined' ? window.location.origin : 'https://lingbyte.github.io/ling-base',
    'X-Title': 'ling-base docs playground',
  };
}

async function tryProxy(path: string, body: Record<string, unknown>): Promise<Response | null> {
  try {
    const res = await fetch(asset(path), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    // 静态导出站点没有 API 路由时会 404/405
    if (res.status === 404 || res.status === 405) return null;
    return res;
  } catch {
    return null;
  }
}

function buildMessages(req: ChatRequest): ChatMessage[] {
  if (req.messages && req.messages.length > 0) return req.messages;
  return [
    ...(req.system ? [{ role: 'system' as const, content: req.system }] : []),
    { role: 'user' as const, content: req.message ?? '' },
  ];
}

export async function chatCompletionJson(req: ChatRequest) {
  const messages = buildMessages(req);
  if (!messages.some((m) => m.role === 'user' && m.content.trim())) {
    throw new Error('请填写消息内容');
  }

  const payload = {
    apiKey: req.apiKey || undefined,
    baseUrl: req.baseUrl || undefined,
    model: req.model,
    messages,
    stream: false,
  };

  const proxied = await tryProxy('/api/playground/chat', payload);
  if (proxied) {
    if (!proxied.ok) {
      const err = await proxied.json().catch(() => ({ error: proxied.statusText }));
      throw new Error(err.error || proxied.statusText);
    }
    return proxied.json();
  }

  // 回退：浏览器直连（需服务商支持 CORS，如 OpenRouter）
  if (!req.apiKey?.trim()) {
    throw new Error('静态站点无服务端代理，请填写 API Key，并使用支持 CORS 的端点（推荐 OpenRouter）');
  }

  const res = await fetch(`${resolveBase(req.baseUrl)}/chat/completions`, {
    method: 'POST',
    headers: providerHeaders(req.apiKey),
    body: JSON.stringify({
      model: req.model || 'openai/gpt-4o-mini',
      messages,
      stream: false,
    }),
  });
  if (!res.ok) {
    throw new Error((await res.text()) || res.statusText);
  }
  const data = await res.json();
  return {
    text: data.choices?.[0]?.message?.content ?? '',
    usage: data.usage,
    model: data.model,
    id: data.id,
  };
}

export async function chatCompletionStream(req: ChatRequest, onDelta: (text: string) => void) {
  const messages = buildMessages(req);
  if (!messages.some((m) => m.role === 'user' && m.content.trim())) {
    throw new Error('请填写消息内容');
  }

  const payload = {
    apiKey: req.apiKey || undefined,
    baseUrl: req.baseUrl || undefined,
    model: req.model,
    messages,
    stream: true,
  };

  let res = await tryProxy('/api/playground/chat', payload);

  if (!res) {
    if (!req.apiKey?.trim()) {
      throw new Error('静态站点无服务端代理，请填写 API Key，并使用支持 CORS 的端点（推荐 OpenRouter）');
    }
    res = await fetch(`${resolveBase(req.baseUrl)}/chat/completions`, {
      method: 'POST',
      headers: providerHeaders(req.apiKey),
      body: JSON.stringify({
        model: req.model || 'openai/gpt-4o-mini',
        messages,
        stream: true,
      }),
    });
  }

  if (!res.ok) {
    const errText = await res.text();
    try {
      const j = JSON.parse(errText);
      throw new Error(j.error || errText);
    } catch (e) {
      if (e instanceof Error && e.message !== errText) throw e;
      throw new Error(errText || res.statusText);
    }
  }

  const reader = res.body?.getReader();
  if (!reader) throw new Error('No response body');

  const decoder = new TextDecoder();
  let buffer = '';
  let text = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() ?? '';
    for (const line of lines) {
      if (!line.startsWith('data: ')) continue;
      const data = line.slice(6).trim();
      if (!data || data === '[DONE]') continue;
      try {
        const json = JSON.parse(data);
        const delta = json.choices?.[0]?.delta?.content;
        if (delta) {
          text += delta;
          onDelta(text);
        }
      } catch {
        // ignore
      }
    }
  }
  return { streamed: true, length: text.length, text };
}

export async function createEmbedding(req: EmbedRequest) {
  if (!req.input?.trim()) throw new Error('请填写文本');

  const payload = {
    apiKey: req.apiKey || undefined,
    baseUrl: req.baseUrl || undefined,
    model: req.model,
    input: req.input,
  };

  const proxied = await tryProxy('/api/playground/embed', payload);
  if (proxied) {
    if (!proxied.ok) {
      const err = await proxied.json().catch(() => ({ error: proxied.statusText }));
      throw new Error(err.error || proxied.statusText);
    }
    return proxied.json();
  }

  if (!req.apiKey?.trim()) {
    throw new Error('静态站点无服务端代理，请填写 API Key，并使用支持 CORS 的端点（推荐 OpenRouter）');
  }

  const res = await fetch(`${resolveBase(req.baseUrl)}/embeddings`, {
    method: 'POST',
    headers: providerHeaders(req.apiKey),
    body: JSON.stringify({
      model: req.model || 'openai/text-embedding-3-small',
      input: req.input.trim(),
    }),
  });
  if (!res.ok) throw new Error((await res.text()) || res.statusText);
  const data = await res.json();
  const embedding: number[] = data.data?.[0]?.embedding ?? [];
  return {
    dimensions: embedding.length,
    preview: embedding.slice(0, 8).map((v: number) => Number(v.toFixed(6))),
    usage: data.usage,
    model: data.model,
  };
}
