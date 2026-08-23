'use client';

import { CheckCircle2, XCircle } from 'lucide-react';

export function ResultBox({ result, error, children }: { result: unknown; error: string | null; children?: React.ReactNode }) {
  if (error) {
    return (
      <div className="mt-4 flex items-start gap-2 rounded-lg bg-red-500/10 p-3 text-sm text-red-500">
        <XCircle className="size-4 shrink-0 mt-0.5" />
        <pre className="whitespace-pre-wrap break-all">{error}</pre>
      </div>
    );
  }
  if (!result && !children) return null;
  return (
    <div className="mt-4 rounded-lg bg-fd-secondary/50 p-3">
      <div className="mb-2 flex items-center gap-2 text-sm font-medium text-emerald-500">
        <CheckCircle2 className="size-4" />
        执行成功
      </div>
      {children}
      {result != null && (
        <pre className="overflow-x-auto text-xs text-fd-foreground">
          {typeof result === 'string' ? result : JSON.stringify(result, null, 2)}
        </pre>
      )}
    </div>
  );
}

export function RunButton({ onClick, loading, label, variant = 'primary' }: {
  onClick: () => void;
  loading: boolean;
  label?: string;
  variant?: 'primary' | 'secondary';
}) {
  const base = 'inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition disabled:opacity-50';
  const styles = variant === 'primary'
    ? `${base} bg-fd-primary text-fd-primary-foreground hover:bg-fd-primary/90`
    : `${base} border border-fd-border hover:bg-fd-accent`;

  return (
    <button onClick={onClick} disabled={loading} className={styles}>
      {loading ? (
        <span className="size-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
      ) : (
        <span className="size-4">▶</span>
      )}
      {label || '运行'}
    </button>
  );
}

export const inputClass = 'w-full rounded-lg border border-fd-border bg-fd-background px-3 py-2 text-sm';

export function FieldLabel({ children }: { children: React.ReactNode }) {
  return <span className="mb-1 block text-xs font-medium text-fd-muted-foreground">{children}</span>;
}

export function ApiKeyNotice() {
  return (
    <p className="rounded-lg border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-xs text-amber-600 dark:text-amber-400">
      本地 <code className="mx-0.5">pnpm dev</code> 会走服务端代理（无 CORS）。可在页面填写 Key，或在{' '}
      <code className="mx-0.5">docs/.env.local</code> 配置 <code className="mx-0.5">OPENROUTER_API_KEY</code>
      。Key 不会写入日志。
    </p>
  );
}

export function RealtimeKeyNotice({ provider }: { provider: 'aliyun_omni' | 'openai_realtime' }) {
  const envKey = provider === 'aliyun_omni' ? 'DASHSCOPE_API_KEY' : 'OPENAI_API_KEY';
  return (
    <p className="rounded-lg border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-xs text-amber-600 dark:text-amber-400">
      使用 <code className="mx-0.5">node server.mjs</code> 启动（<code className="mx-0.5">pnpm dev</code>）以启用
      WebSocket 代理。在页面填写 Key，或在 <code className="mx-0.5">docs/.env.local</code> 配置{' '}
      <code className="mx-0.5">{envKey}</code>。Key 不会写入日志。
    </p>
  );
}
