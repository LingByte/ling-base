import { NextResponse } from 'next/server';

export const runtime = 'nodejs';

interface EmbedBody {
  apiKey?: string;
  baseUrl?: string;
  model?: string;
  input: string;
}

function resolveEndpoint(baseUrl?: string): string {
  const base = (baseUrl?.trim() || process.env.PLAYGROUND_BASE_URL || 'https://openrouter.ai/api/v1').replace(
    /\/$/,
    '',
  );
  return `${base}/embeddings`;
}

function resolveApiKey(bodyKey?: string): string | null {
  const key = bodyKey?.trim() || process.env.OPENROUTER_API_KEY?.trim() || process.env.PLAYGROUND_API_KEY?.trim();
  return key || null;
}

export async function POST(req: Request) {
  try {
    const body = (await req.json()) as EmbedBody;
    const { baseUrl, model, input } = body;
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
    if (!input?.trim()) {
      return NextResponse.json({ error: '请填写文本' }, { status: 400 });
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
        model: model || 'openai/text-embedding-3-small',
        input: input.trim(),
      }),
    });

    if (!upstream.ok) {
      const errText = await upstream.text();
      return NextResponse.json({ error: errText || upstream.statusText }, { status: upstream.status });
    }

    const data = await upstream.json();
    const embedding: number[] = data.data?.[0]?.embedding ?? [];
    return NextResponse.json({
      dimensions: embedding.length,
      preview: embedding.slice(0, 8).map((v: number) => Number(v.toFixed(6))),
      usage: data.usage,
      model: data.model,
    });
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    return NextResponse.json({ error: msg }, { status: 500 });
  }
}
