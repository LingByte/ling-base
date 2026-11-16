#!/usr/bin/env node
/**
 * Fast GitHub Pages export for ling-base docs.
 *
 * Default engine: lightweight MDX/Markdown → HTML (seconds).
 * Optional: DOCS_STATIC_ENGINE=next for the heavy Next.js export.
 */
import {
  existsSync,
  mkdirSync,
  readdirSync,
  readFileSync,
  renameSync,
  rmSync,
  statSync,
  writeFileSync,
} from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"
import { spawnSync } from "node:child_process"

const root = path.dirname(path.dirname(fileURLToPath(import.meta.url)))
const contentRoot = path.join(root, "content/docs")
const outRoot = path.join(root, "out")
const basePath = (process.env.NEXT_PUBLIC_BASE_PATH || "").replace(/\/$/, "")
const asset = (p) => `${basePath}${p.startsWith("/") ? p : `/${p}`}`

function log(msg) {
  console.log(`[static] ${msg}`)
}

function walkMdx(dir, acc = []) {
  for (const name of readdirSync(dir)) {
    const full = path.join(dir, name)
    const st = statSync(full)
    if (st.isDirectory()) walkMdx(full, acc)
    else if (name.endsWith(".mdx") || name.endsWith(".md")) acc.push(full)
  }
  return acc
}

function parseFrontmatter(raw) {
  if (!raw.startsWith("---")) {
    return { meta: {}, body: raw }
  }
  const end = raw.indexOf("\n---", 3)
  if (end === -1) return { meta: {}, body: raw }
  const fm = raw.slice(3, end).trim()
  const body = raw.slice(end + 4).replace(/^\s+/, "")
  const meta = {}
  for (const line of fm.split("\n")) {
    const m = line.match(/^([A-Za-z0-9_-]+):\s*(.*)$/)
    if (!m) continue
    meta[m[1]] = m[2].replace(/^["']|["']$/g, "").trim()
  }
  return { meta, body }
}

function escapeHtml(s) {
  return s
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
}

function inlineMarkdown(text) {
  let s = escapeHtml(text)
  s = s.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_, label, href) => {
    const url = href.startsWith("http") ? href : asset(href)
    return `<a href="${escapeHtml(url)}">${label}</a>`
  })
  s = s.replace(/`([^`]+)`/g, "<code>$1</code>")
  s = s.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
  s = s.replace(/\*([^*]+)\*/g, "<em>$1</em>")
  return s
}

function markdownToHtml(md) {
  const lines = md.replace(/\r\n/g, "\n").split("\n")
  const html = []
  let i = 0
  let inUl = false
  let inOl = false
  let inCode = false
  let codeLang = ""
  let code = []

  const closeLists = () => {
    if (inUl) {
      html.push("</ul>")
      inUl = false
    }
    if (inOl) {
      html.push("</ol>")
      inOl = false
    }
  }

  while (i < lines.length) {
    const line = lines[i]

    if (line.startsWith("```")) {
      if (!inCode) {
        closeLists()
        inCode = true
        codeLang = line.slice(3).trim()
        code = []
      } else {
        html.push(
          `<pre><code class="language-${escapeHtml(codeLang)}">${escapeHtml(code.join("\n"))}</code></pre>`
        )
        inCode = false
        codeLang = ""
        code = []
      }
      i += 1
      continue
    }

    if (inCode) {
      code.push(line)
      i += 1
      continue
    }

    if (!line.trim()) {
      closeLists()
      i += 1
      continue
    }

    const heading = line.match(/^(#{1,6})\s+(.*)$/)
    if (heading) {
      closeLists()
      const level = heading[1].length
      html.push(`<h${level}>${inlineMarkdown(heading[2])}</h${level}>`)
      i += 1
      continue
    }

    const ul = line.match(/^[-*]\s+(.*)$/)
    if (ul) {
      if (inOl) {
        html.push("</ol>")
        inOl = false
      }
      if (!inUl) {
        html.push("<ul>")
        inUl = true
      }
      html.push(`<li>${inlineMarkdown(ul[1])}</li>`)
      i += 1
      continue
    }

    const ol = line.match(/^\d+\.\s+(.*)$/)
    if (ol) {
      if (inUl) {
        html.push("</ul>")
        inUl = false
      }
      if (!inOl) {
        html.push("<ol>")
        inOl = true
      }
      html.push(`<li>${inlineMarkdown(ol[1])}</li>`)
      i += 1
      continue
    }

    closeLists()
    html.push(`<p>${inlineMarkdown(line)}</p>`)
    i += 1
  }

  closeLists()
  if (inCode) {
    html.push(`<pre><code>${escapeHtml(code.join("\n"))}</code></pre>`)
  }
  return html.join("\n")
}

function fileToRoute(file) {
  const rel = path.relative(contentRoot, file).replace(/\\/g, "/")
  // content/docs/(root)/index.mdx → /docs
  // content/docs/relay/chat.mdx → /docs/relay/chat
  let route = rel.replace(/\.mdx?$/, "")
  route = route.replace(/\(root\)\//g, "")
  if (route.endsWith("/index")) route = route.slice(0, -"/index".length)
  if (route === "index" || route === "") return "/docs"
  return `/docs/${route}`.replace(/\/+/g, "/")
}

function pageShell({ title, description, nav, body, canonical }) {
  return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>${escapeHtml(title)}</title>
  <meta name="description" content="${escapeHtml(description || "")}" />
  <link rel="canonical" href="${escapeHtml(canonical)}" />
  <link rel="stylesheet" href="${asset("/assets/docs.css")}" />
</head>
<body>
  <header class="top">
    <a class="brand" href="${asset("/")}">ling-base</a>
    <nav class="top-nav">
      <a href="${asset("/docs/")}">文档</a>
      <a href="https://github.com/LingByte/ling-base">GitHub</a>
    </nav>
  </header>
  <div class="layout">
    <aside class="side">
      <div class="side-title">文档</div>
      <ul>${nav}</ul>
    </aside>
    <main class="content">
      ${body}
    </main>
  </div>
</body>
</html>
`
}

function homeShell({ nav }) {
  return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>ling-base</title>
  <meta name="description" content="Go 多模块基础库。按需引入 AI 中继、语音、安全与基础设施。" />
  <link rel="stylesheet" href="${asset("/assets/docs.css")}" />
</head>
<body>
  <header class="top">
    <a class="brand" href="${asset("/")}">ling-base</a>
    <nav class="top-nav">
      <a href="${asset("/docs/")}">文档</a>
      <a href="https://github.com/LingByte/ling-base">GitHub</a>
    </nav>
  </header>
  <main class="hero">
    <p class="eyebrow">LingByte</p>
    <h1>ling-base</h1>
    <p class="lede">Go 多模块基础库。按需引入 AI 中继、语音、安全与基础设施，不把无关 SDK 拉进依赖树。</p>
    <p class="actions">
      <a class="btn" href="${asset("/docs/")}">阅读文档</a>
      <a class="btn ghost" href="https://github.com/LingByte/ling-base">GitHub</a>
    </p>
    <section class="toc">
      <h2>目录</h2>
      <ul>${nav}</ul>
    </section>
  </main>
</body>
</html>
`
}

const CSS = `:root {
  color-scheme: light dark;
  --bg: #f7f5f1;
  --fg: #1c1917;
  --muted: #57534e;
  --line: #e7e5e4;
  --card: #ffffff;
  --accent: #0f766e;
  --code-bg: #1c1917;
  --code-fg: #f5f5f4;
  --font: ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif;
  --mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #0c0a09;
    --fg: #fafaf9;
    --muted: #a8a29e;
    --line: #292524;
    --card: #1c1917;
    --accent: #2dd4bf;
    --code-bg: #000;
    --code-fg: #e7e5e4;
  }
}
* { box-sizing: border-box; }
body {
  margin: 0;
  font-family: var(--font);
  background: radial-gradient(1200px 600px at 10% -10%, #ccfbf1 0%, transparent 50%), var(--bg);
  color: var(--fg);
  line-height: 1.6;
}
a { color: var(--accent); text-decoration: none; }
a:hover { text-decoration: underline; }
.top {
  display: flex; align-items: center; justify-content: space-between;
  padding: 1rem 1.25rem; border-bottom: 1px solid var(--line);
  position: sticky; top: 0; backdrop-filter: blur(10px); background: color-mix(in oklab, var(--bg) 85%, transparent);
}
.brand { font-weight: 700; color: var(--fg); font-size: 1.1rem; letter-spacing: -0.02em; }
.top-nav { display: flex; gap: 1rem; }
.layout { display: grid; grid-template-columns: 240px 1fr; gap: 0; min-height: calc(100vh - 64px); }
.side { border-right: 1px solid var(--line); padding: 1.25rem; }
.side-title { font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.08em; color: var(--muted); margin-bottom: 0.75rem; }
.side ul, .toc ul { list-style: none; padding: 0; margin: 0; display: grid; gap: 0.35rem; }
.side a, .toc a { color: var(--fg); }
.content, .hero { padding: 2rem 1.5rem 3rem; max-width: 820px; }
.hero { margin: 0 auto; }
.eyebrow { color: var(--muted); text-transform: uppercase; letter-spacing: 0.12em; font-size: 0.75rem; }
.hero h1 { font-size: clamp(2.5rem, 6vw, 4rem); letter-spacing: -0.04em; margin: 0.2rem 0 0.75rem; }
.lede { color: var(--muted); font-size: 1.15rem; max-width: 40rem; }
.actions { display: flex; gap: 0.75rem; margin: 1.5rem 0 2.5rem; }
.btn {
  display: inline-flex; align-items: center; justify-content: center;
  padding: 0.7rem 1rem; border-radius: 999px; background: var(--accent); color: #042f2e; font-weight: 600;
}
.btn.ghost { background: transparent; color: var(--fg); border: 1px solid var(--line); }
.content h1, .content h2, .content h3 { letter-spacing: -0.02em; line-height: 1.25; }
.content pre {
  background: var(--code-bg); color: var(--code-fg); padding: 1rem; border-radius: 0.75rem;
  overflow: auto; font-family: var(--mono); font-size: 0.9rem;
}
.content code { font-family: var(--mono); font-size: 0.92em; }
.content :not(pre) > code { background: color-mix(in oklab, var(--line) 70%, transparent); padding: 0.1em 0.35em; border-radius: 0.35em; }
.content p, .content li { color: color-mix(in oklab, var(--fg) 92%, var(--muted)); }
@media (max-width: 860px) {
  .layout { grid-template-columns: 1fr; }
  .side { border-right: 0; border-bottom: 1px solid var(--line); }
}
`

function writePage(route, html) {
  const clean = route === "/" ? "" : route.replace(/^\//, "")
  const dir = path.join(outRoot, clean)
  mkdirSync(dir, { recursive: true })
  writeFileSync(path.join(dir, "index.html"), html)
}

function buildFast() {
  const started = Date.now()
  log("engine=fast (MDX → HTML)")
  if (!existsSync(contentRoot)) {
    throw new Error(`missing content dir: ${contentRoot}`)
  }

  rmSync(outRoot, { recursive: true, force: true })
  mkdirSync(path.join(outRoot, "assets"), { recursive: true })
  writeFileSync(path.join(outRoot, "assets", "docs.css"), CSS)

  const files = walkMdx(contentRoot).sort()
  const pages = files.map((file) => {
    const raw = readFileSync(file, "utf8")
    const { meta, body } = parseFrontmatter(raw)
    const route = fileToRoute(file)
    return {
      file,
      route,
      title: meta.title || route,
      description: meta.description || "",
      body,
    }
  })

  const nav = pages
    .map(
      (p) =>
        `<li><a href="${asset(p.route.endsWith("/") ? p.route : `${p.route}/`)}">${escapeHtml(p.title)}</a></li>`
    )
    .join("")

  writePage("/", homeShell({ nav }))

  for (const page of pages) {
    const body = `<article>
  <h1>${escapeHtml(page.title)}</h1>
  ${page.description ? `<p class="lede">${escapeHtml(page.description)}</p>` : ""}
  ${markdownToHtml(page.body)}
</article>`
    const html = pageShell({
      title: `${page.title} · ling-base`,
      description: page.description,
      nav,
      body,
      canonical: `https://lingbyte.github.io${asset(page.route)}/`,
    })
    writePage(page.route === "/docs" ? "/docs" : page.route, html)
    log(`wrote ${page.route}`)
  }

  // Convenience redirect-like docs index already at /docs/
  writeFileSync(
    path.join(outRoot, "404.html"),
    pageShell({
      title: "Not found · ling-base",
      description: "Page not found",
      nav,
      body: `<h1>页面不存在</h1><p><a href="${asset("/docs/")}">返回文档</a></p>`,
      canonical: `https://lingbyte.github.io${asset("/")}`,
    })
  )

  log(`done: ${pages.length} docs pages in ${Date.now() - started}ms → ${outRoot}`)
}

// ── Optional heavy Next export (kept for debugging) ──────────────────────────

const STASH_DIRS = [
  "app/typeset.css",
  "app/rss.xml",
  "app/r",
  "app/api",
  "app/og",
  "app/(view)",
  "app/(app)/llm",
  "app/(app)/(create)/create",
  "app/(app)/(create)/init",
  "app/(app)/(typeset)/typeset",
  "app/(app)/(styles)",
  "app/(app)/blocks",
  "app/(app)/charts",
  "app/(app)/colors",
  "app/(app)/examples",
  "app/(app)/(root)/cards",
]

const stashRoot = path.join(root, ".static-stash")

function stashRoutes() {
  rmSync(stashRoot, { recursive: true, force: true })
  mkdirSync(stashRoot, { recursive: true })
  for (const rel of STASH_DIRS) {
    const src = path.join(root, rel)
    if (!existsSync(src)) continue
    const dest = path.join(stashRoot, rel)
    mkdirSync(path.dirname(dest), { recursive: true })
    renameSync(src, dest)
  }
}

function restoreRoutes() {
  if (!existsSync(stashRoot)) return
  for (const rel of STASH_DIRS) {
    const src = path.join(stashRoot, rel)
    if (!existsSync(src)) continue
    const dest = path.join(root, rel)
    if (existsSync(dest)) rmSync(dest, { recursive: true, force: true })
    mkdirSync(path.dirname(dest), { recursive: true })
    renameSync(src, dest)
  }
  rmSync(stashRoot, { recursive: true, force: true })
}

function run(command, args, env = {}) {
  const result = spawnSync(command, args, {
    cwd: root,
    stdio: "inherit",
    env: { ...process.env, ...env },
    shell: process.platform === "win32",
  })
  if (result.status !== 0) {
    throw new Error(`${command} ${args.join(" ")} failed with code ${result.status}`)
  }
}

function buildNext() {
  log("engine=next (slow; use only if you need the full App Router shell)")
  stashRoutes()
  try {
    const stylesReady = existsSync(path.join(root, "styles", "base-nova", "ui"))
    if (stylesReady && process.env.DOCS_FORCE_REGISTRY_BUILD !== "1") {
      log("styles present; skip registry:build (set DOCS_FORCE_REGISTRY_BUILD=1 to force)")
    } else {
      run("pnpm", ["registry:build"])
    }
    run("pnpm", ["exec", "next", "build", "--webpack"], {
      NEXT_PUBLIC_STATIC_DEPLOY: "1",
      NEXT_PUBLIC_BASE_PATH: process.env.NEXT_PUBLIC_BASE_PATH || "",
      NODE_OPTIONS: [process.env.NODE_OPTIONS, "--max-old-space-size=8192"]
        .filter(Boolean)
        .join(" "),
    })
  } finally {
    restoreRoutes()
  }
}

const isMain =
  process.argv[1] &&
  path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)

if (isMain) {
  const engine = (process.env.DOCS_STATIC_ENGINE || "fast").toLowerCase()
  if (engine === "next") buildNext()
  else buildFast()
}
