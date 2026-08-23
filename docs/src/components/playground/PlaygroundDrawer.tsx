'use client';

import { usePlayground } from './drawer-context';
import { PlaygroundDemo } from './PlaygroundDemo';
import { X, Terminal, Loader2, AlertTriangle } from 'lucide-react';
import { useWasm } from './wasm-loader';
import { useEffect } from 'react';
import { demoNeedsWasm } from '@/lib/playground/types';

export function PlaygroundDrawer() {
  const { open, demoId, close } = usePlayground();
  const needsWasm = demoId ? demoNeedsWasm(demoId) : false;
  const { loaded, loading, error } = useWasm(needsWasm && open);

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close();
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [open, close]);

  if (!open) return null;

  const wasmBlocking = needsWasm && (loading || error || !loaded);

  return (
    <>
      <div className="fixed inset-0 z-50 bg-black/40 backdrop-blur-sm" onClick={close} />

      <div className="fixed right-0 top-0 z-50 flex h-full w-full max-w-xl flex-col border-l border-fd-border bg-fd-background shadow-2xl">
        <div className="flex items-center justify-between border-b border-fd-border px-5 py-4">
          <div className="flex items-center gap-2">
            <Terminal className="size-5 text-fd-primary" />
            <span className="font-semibold">在线演示</span>
            {needsWasm && loading && (
              <span className="flex items-center gap-1 text-xs text-fd-muted-foreground">
                <Loader2 className="size-3 animate-spin" />
                加载 WASM...
              </span>
            )}
          </div>
          <button
            type="button"
            onClick={close}
            className="rounded-lg p-1.5 text-fd-muted-foreground transition hover:bg-fd-accent hover:text-fd-foreground"
          >
            <X className="size-5" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-5">
          {needsWasm && error ? (
            <div className="flex items-start gap-2 rounded-lg bg-red-500/10 p-4 text-sm text-red-500">
              <AlertTriangle className="size-4 shrink-0 mt-0.5" />
              {error}
            </div>
          ) : wasmBlocking ? (
            <div className="flex items-center justify-center gap-2 py-20 text-sm text-fd-muted-foreground">
              <Loader2 className="size-5 animate-spin" />
              正在加载 WASM 模块（首次约 6MB）...
            </div>
          ) : demoId ? (
            <PlaygroundDemo demoId={demoId} />
          ) : null}
        </div>
      </div>
    </>
  );
}
