'use client';

import { useState, useCallback, useRef, useEffect } from 'react';
import { Mic, MicOff, Phone, PhoneOff, Trash2 } from 'lucide-react';
import { VoiceRealtimeSession } from '@/lib/playground/realtime-client';
import {
  REALTIME_PROVIDERS,
  type RealtimeProviderId,
} from '@/lib/playground/realtime-providers';
import { FieldLabel, inputClass, RealtimeKeyNotice } from '../shared';

interface Msg {
  role: 'user' | 'assistant';
  content: string;
}

export function RealtimeDemo() {
  const [provider, setProvider] = useState<RealtimeProviderId>('aliyun_omni');
  const preset = REALTIME_PROVIDERS[provider];
  const [apiKey, setApiKey] = useState('');
  const [model, setModel] = useState(preset.defaultModel);
  const [voice, setVoice] = useState(preset.defaultVoice);
  const [instructions, setInstructions] = useState(
    'You are a helpful assistant. Reply concisely in the same language as the user.',
  );
  const [messages, setMessages] = useState<Msg[]>([]);
  const [streamingAssistant, setStreamingAssistant] = useState<string | null>(null);
  const [status, setStatus] = useState('未连接');
  const [connected, setConnected] = useState(false);
  const [micOn, setMicOn] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const sessionRef = useRef<VoiceRealtimeSession | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, streamingAssistant]);

  const onProviderChange = (id: RealtimeProviderId) => {
    const p = REALTIME_PROVIDERS[id];
    setProvider(id);
    setModel(p.defaultModel);
    setVoice(p.defaultVoice);
    if (connected) void disconnect();
  };

  const disconnect = useCallback(async () => {
    sessionRef.current?.disconnect();
    sessionRef.current = null;
    setConnected(false);
    setMicOn(false);
    setStreamingAssistant(null);
    setStatus('已断开');
  }, []);

  const connect = useCallback(async () => {
    setError(null);
    const session = new VoiceRealtimeSession();
    session.setCallbacks({
      onStatus: setStatus,
      onError: (m) => setError(m),
      onConnected: () => setConnected(true),
      onDisconnected: () => {
        setConnected(false);
        setMicOn(false);
      },
      onBargeIn: () => setStreamingAssistant(null),
      onUserTranscript: (text) => {
        setMessages((prev) => {
          const last = prev[prev.length - 1];
          if (last?.role === 'user' && last.content === text) return prev;
          return [...prev, { role: 'user', content: text }];
        });
      },
      onAssistantText: (text, final) => {
        if (!text.trim()) return;
        if (final) {
          setStreamingAssistant(null);
          setMessages((prev) => {
            const last = prev[prev.length - 1];
            if (last?.role === 'assistant' && last.content === text) return prev;
            return [...prev, { role: 'assistant', content: text }];
          });
        } else {
          setStreamingAssistant(text);
        }
      },
    });
    sessionRef.current = session;
    try {
      await session.connect({ provider, apiKey, model, voice, instructions });
    } catch (e) {
      setError(String(e));
      sessionRef.current = null;
      setConnected(false);
    }
  }, [provider, apiKey, model, voice, instructions]);

  const toggleMic = useCallback(async () => {
    const session = sessionRef.current;
    if (!session?.connected) return;
    setError(null);
    try {
      if (micOn) {
        session.stopMic();
        setMicOn(false);
        setStatus('麦克风已关闭');
      } else {
        await session.startMic();
        setMicOn(true);
      }
    } catch (e) {
      setError(String(e));
    }
  }, [micOn]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">voice/realtime 实时语音对话</h3>
        <p className="text-sm text-fd-muted-foreground">
          WebSocket + PCM 音频，对齐 ling-base voice/realtime（默认阿里云 Qwen-Omni）
        </p>
      </div>

      <RealtimeKeyNotice provider={provider} />

      <div className="grid grid-cols-2 gap-2">
        <div>
          <FieldLabel>Provider</FieldLabel>
          <select
            value={provider}
            onChange={(e) => onProviderChange(e.target.value as RealtimeProviderId)}
            className={inputClass}
            disabled={connected}
          >
            {Object.values(REALTIME_PROVIDERS).map((p) => (
              <option key={p.id} value={p.id}>{p.label}</option>
            ))}
          </select>
        </div>
        <div>
          <FieldLabel>API Key（可选，或用 .env.local）</FieldLabel>
          <input
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder={preset.keyEnv}
            className={inputClass}
            disabled={connected}
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div>
          <FieldLabel>Model</FieldLabel>
          <input
            list={`models-${provider}`}
            value={model}
            onChange={(e) => setModel(e.target.value)}
            className={inputClass}
            disabled={connected}
          />
          <datalist id={`models-${provider}`}>
            {preset.models.map((m) => <option key={m} value={m} />)}
          </datalist>
        </div>
        <div>
          <FieldLabel>Voice</FieldLabel>
          <input
            list={`voices-${provider}`}
            value={voice}
            onChange={(e) => setVoice(e.target.value)}
            className={inputClass}
            disabled={connected}
          />
          <datalist id={`voices-${provider}`}>
            {preset.voices.map((v) => <option key={v} value={v} />)}
          </datalist>
        </div>
      </div>

      <input
        value={instructions}
        onChange={(e) => setInstructions(e.target.value)}
        placeholder="System instructions"
        className={`${inputClass} text-xs`}
        disabled={connected}
      />

      <div className="flex flex-wrap items-center gap-2">
        {!connected ? (
          <button type="button" onClick={() => void connect()}
            className="inline-flex items-center gap-2 rounded-lg bg-fd-primary px-4 py-2 text-sm font-medium text-fd-primary-foreground">
            <Phone className="size-4" /> 连接
          </button>
        ) : (
          <>
            <button type="button" onClick={() => void disconnect()}
              className="inline-flex items-center gap-2 rounded-lg border border-fd-border px-4 py-2 text-sm hover:bg-fd-accent">
              <PhoneOff className="size-4" /> 断开
            </button>
            <button type="button" onClick={() => void toggleMic()}
              className={`inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm ${
                micOn ? 'bg-red-500/20 text-red-500' : 'border border-fd-border hover:bg-fd-accent'
              }`}>
              {micOn ? <MicOff className="size-4" /> : <Mic className="size-4" />}
              {micOn ? '关闭麦克风' : '开启麦克风'}
            </button>
          </>
        )}
        <span className="text-xs text-fd-muted-foreground">{status}</span>
      </div>

      <div className="flex h-72 flex-col rounded-xl border border-fd-border bg-fd-secondary/20">
        <div className="flex-1 space-y-3 overflow-y-auto p-3">
          {messages.length === 0 && !streamingAssistant && (
            <p className="pt-8 text-center text-sm text-fd-muted-foreground">
              连接后开启麦克风，直接说话即可（服务端 VAD 自动检测轮次）
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
          {streamingAssistant && (
            <div className="flex justify-start">
              <div className="max-w-[85%] rounded-2xl border border-fd-border bg-fd-background px-3 py-2 text-sm whitespace-pre-wrap">
                {streamingAssistant}
                <span className="animate-pulse">▌</span>
              </div>
            </div>
          )}
          <div ref={bottomRef} />
        </div>

        {error && <p className="px-3 text-xs text-red-500">{error}</p>}

        <div className="flex items-center gap-2 border-t border-fd-border p-2">
          <button type="button" onClick={() => { setMessages([]); setStreamingAssistant(null); }}
            className="rounded-lg p-2 text-fd-muted-foreground hover:bg-fd-accent" title="清空记录">
            <Trash2 className="size-4" />
          </button>
          <p className="text-xs text-fd-muted-foreground">
            音频由模型实时返回（24kHz PCM），无需浏览器 TTS
          </p>
        </div>
      </div>
    </div>
  );
}
