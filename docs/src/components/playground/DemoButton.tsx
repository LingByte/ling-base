'use client';

import { usePlayground } from './drawer-context';
import { Play } from 'lucide-react';

type DemoId = 'totp' | 'compress';

export function DemoButton({ demo, label }: { demo: DemoId; label?: string }) {
  const { openDemo } = usePlayground();
  return (
    <button
      onClick={() => openDemo(demo)}
      className="inline-flex items-center gap-1.5 rounded-lg bg-fd-primary/10 px-3 py-1.5 text-sm font-medium text-fd-primary transition hover:bg-fd-primary/20"
    >
      <Play className="size-3.5" />
      {label || '在线演示'}
    </button>
  );
}
