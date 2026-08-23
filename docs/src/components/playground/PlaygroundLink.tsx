'use client';

import { usePlayground } from './drawer-context';
import { Terminal } from 'lucide-react';

export function PlaygroundLink() {
  const { openDemo } = usePlayground();
  return (
    <button
      type="button"
      onClick={() => openDemo('chat')}
      className="inline-flex items-center gap-2 rounded-lg border border-fd-border px-6 py-2.5 text-sm font-medium transition hover:bg-fd-accent"
    >
      <Terminal className="size-4" />
      Playground
    </button>
  );
}
