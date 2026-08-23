import { asset } from '@/lib/shared';
import type { RealtimeProviderId } from './realtime-providers';
import { INPUT_SAMPLE_RATE, OUTPUT_SAMPLE_RATE } from './realtime-providers';

export interface RealtimeConnectConfig {
  provider: RealtimeProviderId;
  apiKey?: string;
  model?: string;
  voice?: string;
  instructions?: string;
}

export interface RealtimeCallbacks {
  onStatus?: (status: string) => void;
  onUserTranscript?: (text: string) => void;
  onAssistantText?: (text: string, final: boolean) => void;
  onError?: (message: string) => void;
  onConnected?: (info: { provider: string; model: string; voice: string }) => void;
  onDisconnected?: () => void;
  /** User started speaking — clear in-flight assistant UI */
  onBargeIn?: () => void;
}

function wsUrl(): string {
  const proto = typeof window !== 'undefined' && window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const host = typeof window !== 'undefined' ? window.location.host : 'localhost:3000';
  return `${proto}//${host}${asset('/api/playground/realtime')}`;
}

async function messageToText(data: unknown): Promise<string> {
  if (typeof data === 'string') return data;
  if (data instanceof Blob) return data.text();
  if (data instanceof ArrayBuffer) return new TextDecoder().decode(data);
  if (ArrayBuffer.isView(data)) {
    return new TextDecoder().decode(data as ArrayBufferView<ArrayBuffer>);
  }
  return String(data);
}

function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunk));
  }
  return btoa(binary);
}

function floatTo16BitPCM(samples: Float32Array): ArrayBuffer {
  const buffer = new ArrayBuffer(samples.length * 2);
  const view = new DataView(buffer);
  for (let i = 0; i < samples.length; i++) {
    const s = Math.max(-1, Math.min(1, samples[i]!));
    view.setInt16(i * 2, s < 0 ? s * 0x8000 : s * 0x7fff, true);
  }
  return buffer;
}

class PcmPlayer {
  private ctx: AudioContext;
  private nextTime = 0;
  private sources = new Set<AudioBufferSourceNode>();

  constructor() {
    this.ctx = new AudioContext({ sampleRate: OUTPUT_SAMPLE_RATE });
  }

  async resume() {
    if (this.ctx.state === 'suspended') await this.ctx.resume();
  }

  play(pcm: ArrayBuffer) {
    const int16 = new Int16Array(pcm);
    const float32 = new Float32Array(int16.length);
    for (let i = 0; i < int16.length; i++) float32[i] = int16[i]! / 32768;
    const buf = this.ctx.createBuffer(1, float32.length, OUTPUT_SAMPLE_RATE);
    buf.copyToChannel(float32, 0);
    const src = this.ctx.createBufferSource();
    src.buffer = buf;
    src.connect(this.ctx.destination);
    src.onended = () => this.sources.delete(src);
    this.sources.add(src);
    const now = this.ctx.currentTime;
    const start = Math.max(now, this.nextTime);
    src.start(start);
    this.nextTime = start + buf.duration;
  }

  /** Stop all scheduled/playing chunks immediately (barge-in). */
  flush() {
    for (const src of this.sources) {
      try { src.stop(); } catch { /* already stopped */ }
      src.disconnect();
    }
    this.sources.clear();
    this.nextTime = this.ctx.currentTime;
  }

  close() {
    this.flush();
    void this.ctx.close();
  }
}

type WireMsg = {
  type?: string;
  delta?: string;
  transcript?: string;
  provider?: string;
  model?: string;
  voice?: string;
  error?: { message?: string };
};

export class VoiceRealtimeSession {
  private ws: WebSocket | null = null;
  private player = new PcmPlayer();
  private micCtx: AudioContext | null = null;
  private micStream: MediaStream | null = null;
  private micProcessor: ScriptProcessorNode | null = null;
  private micSource: MediaStreamAudioSourceNode | null = null;
  private micGain: GainNode | null = null;
  private resampleBuf: number[] = [];
  private assistantBuf = '';
  private callbacks: RealtimeCallbacks = {};
  private intentionalClose = false;
  /** After user interrupts, ignore stale assistant chunks until a new response starts. */
  private interrupted = false;

  private bargeIn() {
    this.interrupted = true;
    this.assistantBuf = '';
    this.player.flush();
    this.ws?.send(JSON.stringify({ type: 'response.cancel' }));
    this.callbacks.onBargeIn?.();
    this.callbacks.onStatus?.('已打断 — 正在听你说…');
  }

  setCallbacks(cb: RealtimeCallbacks) {
    this.callbacks = cb;
  }

  get connected() {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  async connect(cfg: RealtimeConnectConfig): Promise<void> {
    this.disconnect();
    this.intentionalClose = false;

    return new Promise((resolve, reject) => {
      const ws = new WebSocket(wsUrl());
      this.ws = ws;
      let settled = false;

      ws.onopen = () => {
        ws.send(JSON.stringify({
          provider: cfg.provider,
          apiKey: cfg.apiKey?.trim() || undefined,
          model: cfg.model,
          voice: cfg.voice,
          instructions: cfg.instructions,
        }));
      };

      ws.onmessage = (ev) => {
        void (async () => {
          let msg: WireMsg;
          try {
            const text = await messageToText(ev.data);
            msg = JSON.parse(text) as WireMsg;
          } catch {
            return;
          }

          switch (msg.type) {
            case 'proxy.connected':
              this.callbacks.onConnected?.({
                provider: msg.provider ?? cfg.provider,
                model: msg.model ?? cfg.model ?? '',
                voice: msg.voice ?? cfg.voice ?? '',
              });
              this.callbacks.onStatus?.('已连接 — 等待 session 就绪…');
              void this.player.resume();
              if (!settled) {
                settled = true;
                resolve();
              }
              break;

            case 'session.created':
            case 'session.updated':
              this.callbacks.onStatus?.('会话就绪 — 点击麦克风开始说话');
              break;

            case 'input_audio_buffer.speech_started':
              this.bargeIn();
              break;

            case 'response.created':
            case 'response.output_item.added':
              this.interrupted = false;
              this.assistantBuf = '';
              break;

            case 'response.cancelled':
              this.interrupted = true;
              this.assistantBuf = '';
              this.player.flush();
              this.callbacks.onBargeIn?.();
              break;

            case 'input_audio_buffer.speech_stopped':
              this.callbacks.onStatus?.('处理中…');
              break;

            case 'conversation.item.input_audio_transcription.completed':
              if (msg.transcript) this.callbacks.onUserTranscript?.(msg.transcript);
              break;

            case 'response.audio_transcript.delta':
              if (this.interrupted) break;
              if (msg.delta) {
                this.assistantBuf += msg.delta;
                this.callbacks.onAssistantText?.(this.assistantBuf, false);
              }
              break;

            case 'response.audio_transcript.done': {
              if (this.interrupted) break;
              const finalText = (msg.transcript || this.assistantBuf).trim();
              if (finalText) this.callbacks.onAssistantText?.(finalText, true);
              this.assistantBuf = '';
              break;
            }

            case 'response.audio.delta':
            case 'response.output_audio.delta':
              if (this.interrupted) break;
              if (msg.delta) {
                const bin = atob(msg.delta);
                const bytes = new Uint8Array(bin.length);
                for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
                this.player.play(bytes.buffer);
              }
              break;

            case 'response.done':
              this.callbacks.onStatus?.(this.micStream ? '麦克风已开启 — 继续说话' : '会话就绪');
              break;

            case 'error':
              this.callbacks.onError?.(msg.error?.message ?? 'Unknown error');
              if (!settled) {
                settled = true;
                reject(new Error(msg.error?.message ?? 'error'));
              }
              break;
          }
        })();
      };

      ws.onerror = () => {
        const err =
          'WebSocket 连接失败。请使用 pnpm dev（node server.mjs）启动，静态导出站点不支持 voice/realtime。';
        this.callbacks.onError?.(err);
        if (!settled) {
          settled = true;
          reject(new Error(err));
        }
      };

      ws.onclose = () => {
        this.stopMic();
        this.callbacks.onStatus?.('已断开');
        this.callbacks.onDisconnected?.();
        if (!settled && !this.intentionalClose) {
          settled = true;
          reject(new Error('连接在就绪前关闭'));
        }
      };
    });
  }

  async startMic(): Promise<void> {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) throw new Error('未连接');
    if (this.micStream) return;

    this.micStream = await navigator.mediaDevices.getUserMedia({
      audio: { channelCount: 1, echoCancellation: true, noiseSuppression: true },
    });
    this.micCtx = new AudioContext();
    const ratio = this.micCtx.sampleRate / INPUT_SAMPLE_RATE;

    this.micSource = this.micCtx.createMediaStreamSource(this.micStream);
    this.micProcessor = this.micCtx.createScriptProcessor(4096, 1, 1);
    // ScriptProcessor must stay in the graph; mute so mic isn't played back (echo).
    this.micGain = this.micCtx.createGain();
    this.micGain.gain.value = 0;

    this.micProcessor.onaudioprocess = (e) => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
      const input = e.inputBuffer.getChannelData(0);
      for (let i = 0; i < input.length; i += ratio) {
        this.resampleBuf.push(input[Math.floor(i)]!);
      }
      const chunkSamples = 1600;
      while (this.resampleBuf.length >= chunkSamples) {
        const slice = this.resampleBuf.splice(0, chunkSamples);
        const pcm = floatTo16BitPCM(new Float32Array(slice));
        this.ws.send(JSON.stringify({
          type: 'input_audio_buffer.append',
          audio: arrayBufferToBase64(pcm),
        }));
      }
    };

    this.micSource.connect(this.micProcessor);
    this.micProcessor.connect(this.micGain);
    this.micGain.connect(this.micCtx.destination);
    this.callbacks.onStatus?.('麦克风已开启 — 直接说话，VAD 自动检测');
  }

  stopMic() {
    this.micProcessor?.disconnect();
    this.micSource?.disconnect();
    this.micGain?.disconnect();
    this.micStream?.getTracks().forEach((t) => t.stop());
    this.micProcessor = null;
    this.micSource = null;
    this.micGain = null;
    this.micStream = null;
    if (this.micCtx) {
      void this.micCtx.close();
      this.micCtx = null;
    }
    this.resampleBuf = [];
  }

  cancel() {
    this.bargeIn();
  }

  disconnect() {
    this.intentionalClose = true;
    this.stopMic();
    this.player.close();
    this.player = new PcmPlayer();
    this.ws?.close();
    this.ws = null;
  }
}
