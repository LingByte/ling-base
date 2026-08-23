'use client';

import { isStaticDeploy } from '@/lib/shared';
import { RealtimeTextChatDemo } from './realtime-text-demo';
import { RealtimeVoiceDemo } from './realtime-voice-demo';

/** 本地 dev：voice/realtime WebSocket；GitHub Pages：文本多轮对话 */
export function RealtimeDemo() {
  if (isStaticDeploy) return <RealtimeTextChatDemo />;
  return <RealtimeVoiceDemo />;
}
