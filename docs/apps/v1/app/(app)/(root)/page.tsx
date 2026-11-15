import { type Metadata } from "next"
import Link from "next/link"

import { siteConfig } from "@/lib/config"
import { Announcement } from "@/components/announcement"
import {
  PageActions,
  PageHeader,
  PageHeaderDescription,
  PageHeaderHeading,
} from "@/components/page-header"
import { Button } from "@/styles/radix-luma/ui/button"

const title = "Go 多模块基础库"
const metadataTitle = `${siteConfig.name} - ${title}`
const description = siteConfig.description

const websiteJsonLd = {
  "@context": "https://schema.org",
  "@type": "WebSite",
  "@id": `${siteConfig.url}/#website`,
  url: siteConfig.url,
  name: siteConfig.name,
  alternateName: ["ling-base", "LingByte ling-base"],
  description: siteConfig.description,
  inLanguage: "zh-CN",
  sameAs: [siteConfig.links.github],
}

export const dynamic = "force-static"
export const revalidate = false

export const metadata: Metadata = {
  title: {
    absolute: metadataTitle,
  },
  description,
  alternates: {
    canonical: "/",
  },
  openGraph: {
    type: "website",
    url: siteConfig.url,
    title: metadataTitle,
    description,
    siteName: siteConfig.name,
    images: [
      {
        url: `/og?title=${encodeURIComponent(
          title
        )}&description=${encodeURIComponent(description)}`,
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: metadataTitle,
    description,
    images: [
      {
        url: `/og?title=${encodeURIComponent(
          title
        )}&description=${encodeURIComponent(description)}`,
      },
    ],
  },
}

const modules = [
  {
    title: "relay",
    href: "/docs/relay",
    desc: "统一 Client 调用 Chat / Embed / Image / Audio，约 40 个 channel 适配器。",
  },
  {
    title: "providers/voice",
    href: "/docs/voice",
    desc: "ASR / TTS / Realtime / VAD / 声纹 / 克隆，按厂商独立 module。",
  },
  {
    title: "common",
    href: "/docs/common",
    desc: "缓存、限流、熔断、锁、MQ、JWT 等通用能力，按路径 go get。",
  },
  {
    title: "lingcli",
    href: "/docs/lingcli",
    desc: "项目脚手架，一键生成 web-api / worker 等骨架并接入 ling-base。",
  },
]

export default function IndexPage() {
  return (
    <div className="flex flex-1 flex-col">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify(websiteJsonLd).replace(/</g, "\\u003c"),
        }}
      />
      <PageHeader className="md:**:[.container]:pb-8 lg:**:[.container]:pb-12">
        <Announcement />
        <PageHeaderHeading className="max-w-4xl">ling-base</PageHeaderHeading>
        <PageHeaderDescription>{description}</PageHeaderDescription>
        <PageActions>
          <Button asChild className="h-[35px]">
            <Link href="/docs">快速开始</Link>
          </Button>
          <Button asChild variant="secondary">
            <Link href={siteConfig.links.github} target="_blank" rel="noreferrer">
              GitHub
            </Link>
          </Button>
        </PageActions>
      </PageHeader>

      <div className="container-wrapper flex-1">
        <div className="container space-y-16 py-10 md:py-16">
          <section className="space-y-4">
            <h2 className="text-2xl font-semibold tracking-tight">安装</h2>
            <p className="max-w-2xl text-muted-foreground">
              每个能力是独立 Go module。业务只引入需要的路径：
            </p>
            <pre className="overflow-x-auto rounded-xl border bg-muted/40 p-4 font-mono text-sm">
              <code>{`go get github.com/LingByte/ling-base/relay
go get github.com/LingByte/ling-base/common/logger
go get github.com/LingByte/ling-base/bootstrap`}</code>
            </pre>
          </section>

          <section className="space-y-6">
            <div className="space-y-2">
              <h2 className="text-2xl font-semibold tracking-tight">模块</h2>
              <p className="max-w-2xl text-muted-foreground">
                约 238 个独立 go.mod。本地用仓库根目录 go.work 联调，发布后按路径 go get。
              </p>
            </div>
            <div className="divide-y rounded-xl border">
              {modules.map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className="flex flex-col gap-1 px-5 py-4 transition-colors hover:bg-muted/40 sm:flex-row sm:items-baseline sm:gap-6"
                >
                  <code className="shrink-0 font-mono text-sm font-medium">
                    {item.title}
                  </code>
                  <span className="text-sm text-muted-foreground">{item.desc}</span>
                </Link>
              ))}
            </div>
          </section>
        </div>
      </div>
    </div>
  )
}
