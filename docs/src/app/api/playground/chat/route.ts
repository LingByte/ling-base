import { NextResponse } from 'next/server';

export const runtime = 'nodejs';

interface ChatBody {
  apiKey?: string;
  baseUrl?: string;
  model?: string;
  message?: string;
  system?: string;
  messages?: { role: string; content: string }[];
  stream?: boolean;
}

function resolveEndpoint(baseUrl?: string): string {
  const base = (baseUrl?.trim() || process.env.PLAYGROUND_BASE_URL || 'https://openrouter.ai/api/v1').replace(
    /\/$/,
    '',
  );
  return `${base}/chat/completions`;
}

function resolveApiKey(bodyKey?: string): string | null {
  const key = bodyKey?.trim() || process.env.OPENROUTER_API_KEY?.trim() || process.env.PLAYGROUND_API_KEY?.trim();
  return key || null;
}

export async function POST(req: Request) {
  try {
    const body = (await req.json()) as ChatBody;
    const { baseUrl, model, message, system, messages: history, stream } = body;
    const apiKey = resolveApiKey(body.apiKey);

    if (!apiKey) {
      return NextResponse.json(
        {
          error:
            '请填写 API Key，或在 docs/.env.local 配置 OPENROUTER_API_KEY（本地开发代理会自动使用）',
        },
        { status: 400 },
      );
    }

    const messages =
      history && history.length > 0
        ? history
        : [
            ...(system ? [{ role: 'system', content: system }] : []),
            { role: 'user', content: message ?? '' },
          ];

    if (!messages.some((m) => m.role === 'user' && m.content?.trim())) {
      return NextResponse.json({ error: '请填写消息内容' }, { status: 400 });
    }

    const upstream = await fetch(resolveEndpoint(baseUrl), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${apiKey}`,
        'HTTP-Referer': 'https://lingbyte.github.io/ling-base',
        'X-Title': 'ling-base docs playground',
      },
      body: JSON.stringify({
        model: model || 'openai/gpt-4o-mini',
        messages,
        stream: Boolean(stream),
      }),
    });

    if (!upstream.ok) {
      const errText = await upstream.text();
      return NextResponse.json({ error: errText || upstream.statusText }, { status: upstream.status });
    }

    if (stream) {
      return new Response(upstream.body, {
        headers: {
          'Content-Type': 'text/event-stream; charset=utf-8',
          'Cache-Control': 'no-cache',
        },
      });
    }

    const data = await upstream.json();
    return NextResponse.json({
      text: data.choices?.[0]?.message?.content ?? '',
      usage: data.usage,
      model: data.model,
      id: data.id,
    });
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    return NextResponse.json({ error: msg }, { status: 500 });
  }
}
