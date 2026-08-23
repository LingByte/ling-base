'use client';

import { useState, useCallback, useRef, useEffect } from 'react';
import { Send, Trash2 } from 'lucide-react';
import type { ChatMessage } from '@/lib/playground/ai-client';
import { chatCompletionStream } from '@/lib/playground/ai-client';
import { FieldLabel, inputClass, ApiKeyNotice } from '../shared';

interface Msg {
  role: 'user' | 'assistant';
  content: string;
}

/** GitHub Pages 静态站点：多轮文本对话（浏览器直连 OpenRouter） */
export function RealtimeTextChatDemo() {
  const [apiKey, setApiKey] = useState('');
  const [model, setModel] = useState('openai/gpt-4o-mini');
  const [system, setSystem] = useState(
    'You are a helpful assistant. Reply concisely in the same language as the user.',
  );
  const [input, setInput] = useState('');
  const [messages, setMessages] = useState<Msg[]>([]);
  const [streaming, setStreaming] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, streaming]);

  const send = useCallback(async (text?: string) => {
    const content = (text ?? input).trim();
    if (!content || loading) return;
    if (!apiKey.trim()) {
      setError('请填写 OpenRouter API Key（sk-or-...）');
      return;
    }

    setError(null);
    setLoading(true);
    setInput('');

    const userMsg: Msg = { role: 'user', content };
    const nextMessages = [...messages, userMsg];
    setMessages(nextMessages);
    setStreaming('');

    const apiMessages: ChatMessage[] = [
      { role: 'system', content: system },
      ...nextMessages.map((m) => ({ role: m.role, content: m.content })),
    ];

    try {
      const r = await chatCompletionStream(
        { apiKey, model, messages: apiMessages },
        setStreaming,
      );
      const assistantText = r.text || '';
      if (assistantText) {
        setMessages((prev) => [...prev, { role: 'assistant', content: assistantText }]);
      }
      setStreaming('');
    } catch (e) {
      setError(String(e));
      setMessages((prev) => prev.slice(0, -1));
      setStreaming('');
    } finally {
      setLoading(false);
    }
  }, [apiKey, model, system, messages, input, loading]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">多轮对话（GitHub Pages）</h3>
        <p className="text-sm text-fd-muted-foreground">
          静态站点通过浏览器直连 OpenRouter，支持流式多轮上下文
        </p>
      </div>

      <ApiKeyNotice />

      <div className="grid grid-cols-2 gap-2">
        <div>
          <FieldLabel>OpenRouter API Key</FieldLabel>
          <input
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder="sk-or-..."
            className={inputClass}
          />
        </div>
        <div>
          <FieldLabel>Model</FieldLabel>
          <input
            value={model}
            onChange={(e) => setModel(e.target.value)}
            placeholder="openai/gpt-4o-mini"
            className={inputClass}
          />
        </div>
      </div>

      <input
        value={system}
        onChange={(e) => setSystem(e.target.value)}
        placeholder="System prompt"
        className={`${inputClass} text-xs`}
      />

      <div className="flex h-72 flex-col rounded-xl border border-fd-border bg-fd-secondary/20">
        <div className="flex-1 space-y-3 overflow-y-auto p-3">
          {messages.length === 0 && !streaming && (
            <p className="pt-8 text-center text-sm text-fd-muted-foreground">
              填写 API Key 后开始对话
            </p>
          )}
          {messages.map((m, i) => (
            <div key={i} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              <div className={`max-w-[85%] rounded-2xl px-3 py-2 text-sm whitespace-pre-wrap ${
                m.role === 'user'
                  ? 'bg-fd-primary text-fd-primary-foreground'
                  : 'border border-fd-border bg-fd-background'
              }`}>
                {m.content}
              </div>
            </div>
          ))}
          {streaming && (
            <div className="flex justify-start">
              <div className="max-w-[85%] rounded-2xl border border-fd-border bg-fd-background px-3 py-2 text-sm whitespace-pre-wrap">
                {streaming}
                <span className="animate-pulse">▌</span>
              </div>
            </div>
          )}
          <div ref={bottomRef} />
        </div>

        {error && <p className="px-3 text-xs text-red-500">{error}</p>}

        <div className="flex items-center gap-2 border-t border-fd-border p-2">
          <button
            type="button"
            onClick={() => { setMessages([]); setStreaming(''); setError(null); }}
            className="rounded-lg p-2 text-fd-muted-foreground hover:bg-fd-accent"
            title="清空对话"
          >
            <Trash2 className="size-4" />
          </button>
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && !e.shiftKey && (e.preventDefault(), void send())}
            placeholder="输入消息，Enter 发送"
            className={`${inputClass} flex-1`}
            disabled={loading}
          />
          <button
            type="button"
            onClick={() => void send()}
            disabled={loading || !input.trim()}
            className="rounded-lg bg-fd-primary p-2 text-fd-primary-foreground disabled:opacity-40"
          >
            <Send className="size-4" />
          </button>
        </div>
      </div>
    </div>
  );
}
