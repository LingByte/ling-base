'use client';

import { usePlayground } from './drawer-context';
import { Play } from 'lucide-react';
import type { DemoId } from '@/lib/playground/types';
import { DEMO_META } from '@/lib/playground/types';

export function DemoButton({ demo, label }: { demo: DemoId; label?: string }) {
  const { openDemo } = usePlayground();
  const meta = DEMO_META[demo];
  return (
    <button
      type="button"
      onClick={() => openDemo(demo)}
      className="inline-flex items-center gap-1.5 rounded-lg bg-fd-primary/10 px-3 py-1.5 text-sm font-medium text-fd-primary transition hover:bg-fd-primary/20"
    >
      <Play className="size-3.5" />
      {label || meta?.title || '在线演示'}
    </button>
  );
}
