import Link from 'next/link';
import {
  Bot,
  Mic,
  ShieldCheck,
  Settings2,
  Plug,
  Wrench,
  ArrowRight,
  BookOpen,
  Zap,
  Package,
  Sparkles,
} from 'lucide-react';

function GithubIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="currentColor" viewBox="0 0 24 24">
      <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
    </svg>
  );
}

const categories = [
  {
    icon: Bot,
    label: 'AI 中继',
    count: '40+',
    desc: 'OpenAI / Claude / Gemini 等',
    color: 'from-blue-500 to-cyan-500',
    href: '/docs/relay',
  },
  {
    icon: Mic,
    label: '语音处理',
    count: '36',
    desc: 'TTS / ASR / Realtime',
    color: 'from-purple-500 to-pink-500',
    href: '/docs/voice',
  },
  {
    icon: ShieldCheck,
    label: '安全认证',
    count: '4',
    desc: 'TOTP / Passkey / JWT',
    color: 'from-green-500 to-emerald-500',
    href: '/docs/security',
  },
  {
    icon: Settings2,
    label: '基础设施',
    count: '6',
    desc: '限流 / 熔断 / 重试 / 缓存',
    color: 'from-orange-500 to-amber-500',
    href: '/docs/infrastructure',
  },
  {
    icon: Plug,
    label: '第三方服务',
    count: '20+',
    desc: 'OCR / 审核 / 存储 / MQ',
    color: 'from-red-500 to-rose-500',
    href: '/docs/providers',
  },
  {
    icon: Wrench,
    label: '通用工具',
    count: '50+',
    desc: '日志 / 压缩 / IP / i18n',
    color: 'from-slate-500 to-gray-500',
    href: '/docs/common',
  },
];

const features = [
  {
    icon: Package,
    title: '多模块按需引入',
    desc: '业务只 import 用到的模块，不会拉入无关 SDK',
  },
  {
    icon: Zap,
    title: '生产级基础设施',
    desc: 'HTTP 连接池 / 超时 / 重试 / 熔断 / 降级 / 可观测性',
  },
  {
    icon: Sparkles,
    title: '40+ AI Provider',
    desc: '统一接口调用 Chat / Embed / Image / Audio / Responses',
  },
];

export default function HomePage() {
  return (
    <div className="flex flex-1 flex-col">
      {/* Hero */}
      <section className="relative overflow-hidden px-6 py-24">
        {/* Background gradient */}
        <div className="absolute inset-0 -z-10 bg-gradient-to-b from-blue-500/5 via-transparent to-transparent" />
        <div className="absolute left-1/2 top-0 -z-10 h-[400px] w-[600px] -translate-x-1/2 rounded-full bg-gradient-to-r from-blue-500/10 to-purple-500/10 blur-3xl" />

        <div className="mx-auto flex max-w-4xl flex-col items-center text-center">
          {/* Logo */}
          <div className="mb-8 flex h-20 w-20 items-center justify-center rounded-3xl bg-gradient-to-br from-blue-500 to-purple-600 shadow-lg shadow-blue-500/25">
            <span className="text-4xl font-bold text-white">L</span>
          </div>

          <h1 className="mb-4 text-5xl font-bold tracking-tight">
            ling-base
          </h1>
          <p className="mb-3 max-w-2xl text-xl text-fd-muted-foreground">
            Go 多模块基础库
          </p>
          <p className="mb-10 max-w-2xl text-base text-fd-muted-foreground/80">
            175+ modules · 覆盖 AI 中继、语音处理、安全认证、基础设施、第三方服务对接
          </p>

          {/* CTA */}
          <div className="flex flex-wrap items-center justify-center gap-4">
            <Link
              href="/docs"
              className="group inline-flex items-center gap-2 rounded-lg bg-fd-primary px-6 py-3 text-sm font-medium text-fd-primary-foreground transition hover:bg-fd-primary/90"
            >
              <BookOpen className="size-4" />
              查看文档
              <ArrowRight className="size-4 transition group-hover:translate-x-0.5" />
            </Link>
            <a
              href="https://github.com/LingByte/ling-base"
              target="_blank"
              rel="noopener"
              className="inline-flex items-center gap-2 rounded-lg border border-fd-border px-6 py-3 text-sm font-medium transition hover:bg-fd-accent"
            >
              <GithubIcon className="size-4" />
              GitHub
            </a>
          </div>

          {/* Install command */}
          <div className="mt-8 rounded-lg border border-fd-border bg-fd-secondary/50 px-4 py-2.5">
            <code className="text-sm text-fd-muted-foreground">
              <span className="text-green-500">$</span> go get{' '}
              <span className="text-blue-400">github.com/LingByte/ling-base/relay</span>
            </code>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="mx-auto w-full max-w-5xl px-6 pb-16">
        <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
          {features.map((f) => (
            <div
              key={f.title}
              className="rounded-xl border border-fd-border bg-fd-card p-6"
            >
              <div className="mb-4 flex h-10 w-10 items-center justify-center rounded-lg bg-fd-accent">
                <f.icon className="size-5 text-fd-accent-foreground" />
              </div>
              <h3 className="mb-2 font-semibold">{f.title}</h3>
              <p className="text-sm text-fd-muted-foreground">{f.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Categories */}
      <section className="mx-auto w-full max-w-5xl px-6 pb-24">
        <h2 className="mb-8 text-center text-2xl font-bold">模块导航</h2>
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {categories.map((cat) => (
            <Link
              key={cat.label}
              href={cat.href}
              className="group relative overflow-hidden rounded-xl border border-fd-border bg-fd-card p-6 transition hover:border-fd-primary/50 hover:shadow-lg"
            >
              {/* Gradient accent */}
              <div
                className={`absolute -right-8 -top-8 h-24 w-24 rounded-full bg-gradient-to-br ${cat.color} opacity-10 transition group-hover:opacity-20`}
              />

              <div className="mb-4 flex items-center gap-3">
                <div
                  className={`flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br ${cat.color} shadow-md`}
                >
                  <cat.icon className="size-5 text-white" />
                </div>
                <div>
                  <h3 className="font-semibold">{cat.label}</h3>
                  <p className="text-xs text-fd-muted-foreground">
                    {cat.count} modules
                  </p>
                </div>
              </div>

              <p className="text-sm text-fd-muted-foreground">{cat.desc}</p>

              <div className="mt-4 flex items-center gap-1 text-sm font-medium text-fd-primary opacity-0 transition group-hover:opacity-100">
                查看
                <ArrowRight className="size-3.5" />
              </div>
            </Link>
          ))}
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-fd-border py-8 text-center">
        <p className="text-sm text-fd-muted-foreground">
          ling-base · MIT License ·{' '}
          <a
            href="https://github.com/LingByte/ling-base"
            className="font-medium text-fd-foreground hover:underline"
          >
            GitHub
          </a>
        </p>
      </footer>
    </div>
  );
}
