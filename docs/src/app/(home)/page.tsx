import Link from 'next/link';
import Image from 'next/image';
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
  Terminal,
  Layers,
  CheckCircle2,
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
    color: 'text-blue-500',
    bg: 'bg-blue-500/10',
    border: 'hover:border-blue-500/40',
    href: '/docs/relay',
  },
  {
    icon: Mic,
    label: '语音处理',
    count: '36',
    desc: 'TTS / ASR / Realtime',
    color: 'text-purple-500',
    bg: 'bg-purple-500/10',
    border: 'hover:border-purple-500/40',
    href: '/docs/voice',
  },
  {
    icon: ShieldCheck,
    label: '安全认证',
    count: '4',
    desc: 'TOTP / Passkey / JWT',
    color: 'text-emerald-500',
    bg: 'bg-emerald-500/10',
    border: 'hover:border-emerald-500/40',
    href: '/docs/security',
  },
  {
    icon: Settings2,
    label: '基础设施',
    count: '6',
    desc: '限流 / 熔断 / 重试 / 缓存',
    color: 'text-orange-500',
    bg: 'bg-orange-500/10',
    border: 'hover:border-orange-500/40',
    href: '/docs/infrastructure',
  },
  {
    icon: Plug,
    label: '第三方服务',
    count: '20+',
    desc: 'OCR / 审核 / 存储 / MQ',
    color: 'text-rose-500',
    bg: 'bg-rose-500/10',
    border: 'hover:border-rose-500/40',
    href: '/docs/providers',
  },
  {
    icon: Wrench,
    label: '通用工具',
    count: '50+',
    desc: '日志 / 压缩 / IP / i18n',
    color: 'text-slate-500',
    bg: 'bg-slate-500/10',
    border: 'hover:border-slate-500/40',
    href: '/docs/common',
  },
];

const features = [
  { icon: Package, title: '多模块按需引入', desc: '业务只 import 用到的模块，不会拉入无关 SDK' },
  { icon: Zap, title: '生产级基础设施', desc: 'HTTP 连接池 / 超时 / 重试 / 熔断 / 降级 / 可观测性' },
  { icon: Sparkles, title: '40+ AI Provider', desc: '统一接口调用 Chat / Embed / Image / Audio / Responses' },
];

const stats = [
  { value: '175+', label: 'Modules' },
  { value: '40+', label: 'AI Providers' },
  { value: '36', label: 'Voice Engines' },
  { value: '9', label: 'Storage Backends' },
];

export default function HomePage() {
  return (
    <div className="flex flex-1 flex-col">
      {/* Hero */}
      <section className="relative overflow-hidden px-6 pt-32 pb-20">
        {/* Subtle background glow */}
        <div className="absolute left-1/2 top-20 -z-10 h-[500px] w-[800px] -translate-x-1/2 rounded-full bg-gradient-to-r from-blue-500/8 via-purple-500/5 to-transparent blur-3xl" />

        <div className="mx-auto flex max-w-4xl flex-col items-center text-center">
          {/* Logo — no background, just the image */}
          <Image
            src="/logo.png"
            alt="ling-base"
            width={96}
            height={96}
            className="mb-8 rounded-2xl shadow-xl shadow-black/20"
            priority
          />

          {/* Badge */}
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-fd-border bg-fd-secondary/50 px-3 py-1 text-xs font-medium text-fd-muted-foreground">
            <Layers className="size-3.5" />
            Go 多模块基础库
          </div>

          <h1 className="mb-4 text-6xl font-bold tracking-tight">
            ling-base
          </h1>
          <p className="mb-3 max-w-2xl text-lg text-fd-muted-foreground">
            覆盖 AI 中继、语音处理、安全认证、基础设施、第三方服务对接
          </p>

          {/* Stats row */}
          <div className="mb-10 mt-6 flex items-center gap-8">
            {stats.map((s) => (
              <div key={s.label} className="text-center">
                <div className="text-2xl font-bold text-fd-foreground">{s.value}</div>
                <div className="text-xs text-fd-muted-foreground">{s.label}</div>
              </div>
            ))}
          </div>

          {/* CTA */}
          <div className="flex flex-wrap items-center justify-center gap-3">
            <Link
              href="/docs"
              className="group inline-flex items-center gap-2 rounded-lg bg-fd-primary px-6 py-2.5 text-sm font-medium text-fd-primary-foreground transition hover:bg-fd-primary/90"
            >
              <BookOpen className="size-4" />
              查看文档
              <ArrowRight className="size-4 transition group-hover:translate-x-0.5" />
            </Link>
            <a
              href="https://github.com/LingByte/ling-base"
              target="_blank"
              rel="noopener"
              className="inline-flex items-center gap-2 rounded-lg border border-fd-border px-6 py-2.5 text-sm font-medium transition hover:bg-fd-accent"
            >
              <GithubIcon className="size-4" />
              GitHub
            </a>
          </div>

          {/* Install command */}
          <div className="mt-8 flex items-center gap-3 rounded-lg border border-fd-border bg-fd-secondary/30 px-4 py-2.5 font-mono text-sm">
            <span className="text-green-500">$</span>
            <span className="text-fd-muted-foreground">go get</span>
            <span className="text-blue-400">github.com/LingByte/ling-base/relay</span>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="mx-auto w-full max-w-5xl px-6 pb-20">
        <div className="grid grid-cols-1 gap-5 md:grid-cols-3">
          {features.map((f) => (
            <div
              key={f.title}
              className="group rounded-xl border border-fd-border bg-fd-card p-6 transition hover:border-fd-primary/30"
            >
              <div className="mb-4 flex h-10 w-10 items-center justify-center rounded-lg bg-fd-primary/10">
                <f.icon className="size-5 text-fd-primary" />
              </div>
              <h3 className="mb-2 font-semibold">{f.title}</h3>
              <p className="text-sm leading-relaxed text-fd-muted-foreground">{f.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Categories */}
      <section className="mx-auto w-full max-w-5xl px-6 pb-24">
        <div className="mb-10 text-center">
          <h2 className="mb-2 text-3xl font-bold">模块导航</h2>
          <p className="text-sm text-fd-muted-foreground">按分类浏览 175+ 个模块</p>
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {categories.map((cat) => (
            <Link
              key={cat.label}
              href={cat.href}
              className={`group relative overflow-hidden rounded-xl border border-fd-border bg-fd-card p-6 transition hover:shadow-md ${cat.border}`}
            >
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className={`flex h-11 w-11 items-center justify-center rounded-xl ${cat.bg}`}>
                    <cat.icon className={`size-5 ${cat.color}`} />
                  </div>
                  <div>
                    <h3 className="font-semibold">{cat.label}</h3>
                    <p className="text-xs text-fd-muted-foreground">{cat.count} modules</p>
                  </div>
                </div>
                <ArrowRight className="size-4 text-fd-muted-foreground opacity-0 transition group-hover:opacity-100" />
              </div>

              <p className="mt-4 text-sm text-fd-muted-foreground">{cat.desc}</p>
            </Link>
          ))}
        </div>
      </section>

      {/* lingcli banner */}
      <section className="mx-auto w-full max-w-5xl px-6 pb-24">
        <div className="relative overflow-hidden rounded-2xl border border-fd-border bg-fd-card p-8">
          <div className="flex flex-col items-center gap-6 text-center md:flex-row md:text-left">
            <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-xl bg-fd-primary/10">
              <Terminal className="size-7 text-fd-primary" />
            </div>
            <div className="flex-1">
              <h3 className="mb-1 text-lg font-bold">lingcli 项目脚手架</h3>
              <p className="text-sm text-fd-muted-foreground">
                一键生成完整 Go 项目骨架，集成 ling-base 模块。支持 web-api / grpc-service / cli-tool / library / worker 五种模板。
              </p>
            </div>
            <Link
              href="/docs/lingcli"
              className="inline-flex shrink-0 items-center gap-2 rounded-lg border border-fd-border px-5 py-2.5 text-sm font-medium transition hover:bg-fd-accent"
            >
              了解更多
              <ArrowRight className="size-4" />
            </Link>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="mt-auto border-t border-fd-border py-8 text-center">
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
