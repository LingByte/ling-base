'use client';

import { Terminal } from 'lucide-react';
import { usePlayground } from './drawer-context';
import type { DemoId } from '@/lib/playground/types';
import { DEMO_META } from '@/lib/playground/types';

export function PlaygroundBanner({ demos }: { demos: DemoId[] }) {
  const { openDemo } = usePlayground();

  if (demos.length === 0) return null;

  return (
    <div className="not-prose mb-6 rounded-xl border border-fd-primary/20 bg-fd-primary/5 p-4">
      <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-fd-primary">
        <Terminal className="size-4" />
        在线 Playground
      </div>
      <p className="mb-3 text-sm text-fd-muted-foreground">
        在浏览器中直接体验本页相关 API，无需本地安装 Go 环境。
      </p>
      <div className="flex flex-wrap gap-2">
        {demos.map((id) => {
          const meta = DEMO_META[id];
          return (
            <button
              key={id}
              type="button"
              onClick={() => openDemo(id)}
              className="inline-flex flex-col items-start rounded-lg border border-fd-border bg-fd-background px-3 py-2 text-left transition hover:border-fd-primary/40 hover:bg-fd-accent"
            >
              <span className="text-sm font-medium">{meta.title}</span>
              <span className="text-xs text-fd-muted-foreground">{meta.description}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
