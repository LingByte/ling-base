/** voice/realtime provider presets — mirrors Go adapters */

export type RealtimeProviderId = 'aliyun_omni' | 'openai_realtime';

export interface RealtimeProviderPreset {
  id: RealtimeProviderId;
  label: string;
  defaultModel: string;
  defaultVoice: string;
  keyEnv: string;
  voices: string[];
  models: string[];
}

export const REALTIME_PROVIDERS: Record<RealtimeProviderId, RealtimeProviderPreset> = {
  aliyun_omni: {
    id: 'aliyun_omni',
    label: '阿里云 Qwen-Omni (DashScope)',
    defaultModel: 'qwen3.5-omni-flash-realtime-2026-03-15',
    defaultVoice: 'Tina',
    keyEnv: 'DASHSCOPE_API_KEY',
    models: ['qwen3.5-omni-flash-realtime-2026-03-15'],
    voices: ['Tina', 'Ethan', 'Serena', 'Harvey', 'Maia', 'Evan'],
  },
  openai_realtime: {
    id: 'openai_realtime',
    label: 'OpenAI Realtime',
    defaultModel: 'gpt-4o-realtime-preview',
    defaultVoice: 'alloy',
    keyEnv: 'OPENAI_API_KEY',
    models: ['gpt-4o-realtime-preview', 'gpt-4o-mini-realtime-preview'],
    voices: ['alloy', 'ash', 'ballad', 'coral', 'echo', 'sage', 'shimmer', 'verse'],
  },
};

export const INPUT_SAMPLE_RATE = 16000;
export const OUTPUT_SAMPLE_RATE = 24000;
