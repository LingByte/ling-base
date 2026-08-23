'use client';

import { useState, useCallback } from 'react';
import { ResultBox, RunButton, inputClass, FieldLabel, ApiKeyNotice } from '../shared';
import {
  chatCompletionJson,
  chatCompletionStream,
  createEmbedding,
} from '@/lib/playground/ai-client';

function ApiConfigFields({ apiKey, setApiKey, baseUrl, setBaseUrl, model, setModel, modelPlaceholder }: {
  apiKey: string;
  setApiKey: (v: string) => void;
  baseUrl: string;
  setBaseUrl: (v: string) => void;
  model: string;
  setModel: (v: string) => void;
  modelPlaceholder: string;
}) {
  return (
    <>
      <ApiKeyNotice />
      <label className="block">
        <FieldLabel>API Key</FieldLabel>
        <input type="password" value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="可选：sk-or-...（或用 .env.local）" className={inputClass} />
      </label>
      <label className="block">
        <FieldLabel>Base URL（默认 OpenRouter）</FieldLabel>
        <input value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} placeholder="https://openrouter.ai/api/v1" className={`${inputClass} font-mono text-xs`} />
      </label>
      <label className="block">
        <FieldLabel>Model</FieldLabel>
        <input value={model} onChange={(e) => setModel(e.target.value)} placeholder={modelPlaceholder} className={inputClass} />
      </label>
    </>
  );
}

export function ChatDemo() {
  const [apiKey, setApiKey] = useState('');
  const [baseUrl, setBaseUrl] = useState('');
  const [model, setModel] = useState('openai/gpt-4o-mini');
  const [message, setMessage] = useState('用一句话介绍 ling-base 是什么');
  const [system, setSystem] = useState('You are a helpful assistant.');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try {
      setResult(await chatCompletionJson({ apiKey, baseUrl: baseUrl || undefined, model, message, system }));
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [apiKey, baseUrl, model, message, system]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">Chat 对话</h3>
        <p className="text-sm text-fd-muted-foreground">relay.Chat — 浏览器直连 OpenAI 兼容 API</p>
      </div>
      <ApiConfigFields apiKey={apiKey} setApiKey={setApiKey} baseUrl={baseUrl} setBaseUrl={setBaseUrl} model={model} setModel={setModel} modelPlaceholder="openai/gpt-4o-mini" />
      <label className="block">
        <FieldLabel>System Prompt</FieldLabel>
        <input value={system} onChange={(e) => setSystem(e.target.value)} className={inputClass} />
      </label>
      <label className="block">
        <FieldLabel>User Message</FieldLabel>
        <textarea value={message} onChange={(e) => setMessage(e.target.value)} rows={3} className={inputClass} />
      </label>
      <RunButton onClick={run} loading={loading} label="发送 Chat 请求" />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function ChatStreamDemo() {
  const [apiKey, setApiKey] = useState('');
  const [baseUrl, setBaseUrl] = useState('');
  const [model, setModel] = useState('openai/gpt-4o-mini');
  const [message, setMessage] = useState('写一首关于 Go 语言的短诗');
  const [streamText, setStreamText] = useState('');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null); setStreamText('');
    try {
      const r = await chatCompletionStream(
        { apiKey, baseUrl: baseUrl || undefined, model, message },
        setStreamText,
      );
      setResult(r);
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [apiKey, baseUrl, model, message]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">流式 Chat</h3>
        <p className="text-sm text-fd-muted-foreground">relay.ChatStream — SSE 流式输出</p>
      </div>
      <ApiConfigFields apiKey={apiKey} setApiKey={setApiKey} baseUrl={baseUrl} setBaseUrl={setBaseUrl} model={model} setModel={setModel} modelPlaceholder="openai/gpt-4o-mini" />
      <label className="block">
        <FieldLabel>User Message</FieldLabel>
        <textarea value={message} onChange={(e) => setMessage(e.target.value)} rows={3} className={inputClass} />
      </label>
      <RunButton onClick={run} loading={loading} label="流式调用" />
      {streamText && (
        <div className="rounded-lg border border-fd-border bg-fd-secondary/30 p-3 text-sm whitespace-pre-wrap">{streamText}</div>
      )}
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function EmbedDemo() {
  const [apiKey, setApiKey] = useState('');
  const [baseUrl, setBaseUrl] = useState('');
  const [model, setModel] = useState('openai/text-embedding-3-small');
  const [input, setInput] = useState('ling-base is a Go multi-module foundation library');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try {
      setResult(await createEmbedding({ apiKey, baseUrl: baseUrl || undefined, model, input }));
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [apiKey, baseUrl, model, input]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">Embeddings</h3>
        <p className="text-sm text-fd-muted-foreground">relay.Embed — 文本向量化</p>
      </div>
      <ApiConfigFields apiKey={apiKey} setApiKey={setApiKey} baseUrl={baseUrl} setBaseUrl={setBaseUrl} model={model} setModel={setModel} modelPlaceholder="openai/text-embedding-3-small" />
      <label className="block">
        <FieldLabel>输入文本</FieldLabel>
        <textarea value={input} onChange={(e) => setInput(e.target.value)} rows={3} className={inputClass} />
      </label>
      <RunButton onClick={run} loading={loading} label="生成 Embedding" />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function LimiterDemo() {
  const [rate, setRate] = useState(5);
  const [burst, setBurst] = useState(10);
  const [requests, setRequests] = useState(20);
  const [result, setResult] = useState<unknown>(null);
  const [loading, setLoading] = useState(false);

  const simulate = useCallback(() => {
    setLoading(true);
    const tokens = burst;
    let available = tokens;
    const lastRefill = Date.now();
    const interval = 1000 / rate;
    const log: string[] = [];
    let allowed = 0;
    let rejected = 0;

    for (let i = 0; i < requests; i++) {
      const now = Date.now() + i * 50;
      const elapsed = now - lastRefill;
      const refill = Math.floor(elapsed / interval);
      if (refill > 0) {
        available = Math.min(burst, available + refill);
      }
      if (available > 0) {
        available--;
        allowed++;
        log.push(`#${i + 1} ✅ 通过 (剩余 ${available})`);
      } else {
        rejected++;
        log.push(`#${i + 1} ❌ 限流`);
      }
    }

    setResult({ rate, burst, requests, allowed, rejected, log: log.slice(0, 15) });
    setLoading(false);
  }, [rate, burst, requests]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">令牌桶限流</h3>
        <p className="text-sm text-fd-muted-foreground">common/limiter — 模拟 QPS 限流行为</p>
      </div>
      <div className="grid grid-cols-3 gap-3">
        <label>
          <FieldLabel>速率 (QPS)</FieldLabel>
          <input type="number" value={rate} onChange={(e) => setRate(Number(e.target.value))} className={inputClass} min={1} />
        </label>
        <label>
          <FieldLabel>桶容量</FieldLabel>
          <input type="number" value={burst} onChange={(e) => setBurst(Number(e.target.value))} className={inputClass} min={1} />
        </label>
        <label>
          <FieldLabel>请求数</FieldLabel>
          <input type="number" value={requests} onChange={(e) => setRequests(Number(e.target.value))} className={inputClass} min={1} max={100} />
        </label>
      </div>
      <RunButton onClick={simulate} loading={loading} label="模拟限流" />
      <ResultBox result={result} error={null} />
    </div>
  );
}
