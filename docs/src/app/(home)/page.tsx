import Link from 'next/link';

export default function HomePage() {
  return (
    <div className="flex flex-1 flex-col items-center justify-center text-center">
      <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-blue-500 to-purple-600 text-3xl font-bold text-white">
        L
      </div>
      <h1 className="mb-3 text-4xl font-bold">ling-base</h1>
      <p className="mb-2 max-w-xl text-lg text-fd-muted-foreground">
        Go 多模块基础库，175+ modules
      </p>
      <p className="mb-8 max-w-xl text-sm text-fd-muted-foreground">
        覆盖 AI 中继、语音处理、安全认证、基础设施、第三方服务对接等场景
      </p>
      <div className="flex gap-3">
        <Link
          href="/docs"
          className="inline-flex items-center justify-center rounded-md bg-fd-primary px-6 py-2 text-sm font-medium text-fd-primary-foreground hover:bg-fd-primary/80"
        >
          查看文档
        </Link>
        <a
          href="https://github.com/LingByte/ling-base"
          target="_blank"
          rel="noopener"
          className="inline-flex items-center justify-center rounded-md border px-6 py-2 text-sm font-medium hover:bg-fd-accent"
        >
          GitHub
        </a>
      </div>

      <div className="mt-12 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
        {[
          { icon: '🤖', label: 'AI 中继', count: '40+' },
          { icon: '🎙️', label: '语音处理', count: '36' },
          { icon: '🔐', label: '安全认证', count: '4' },
          { icon: '⚙️', label: '基础设施', count: '6' },
          { icon: '🔌', label: '第三方服务', count: '20+' },
          { icon: '🛠️', label: '通用工具', count: '50+' },
        ].map((item) => (
          <div
            key={item.label}
            className="flex flex-col items-center rounded-xl border border-fd-border p-4"
          >
            <span className="mb-1 text-2xl">{item.icon}</span>
            <span className="text-sm font-medium">{item.label}</span>
            <span className="text-xs text-fd-muted-foreground">{item.count} modules</span>
          </div>
        ))}
      </div>
    </div>
  );
}
