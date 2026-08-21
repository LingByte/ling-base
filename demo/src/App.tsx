import { useState } from 'react'
import { categories, type ModuleInfo } from './modules'
import { Playground } from './Playground'
import { ModuleCard } from './ModuleCard'

type Tab = 'modules' | 'playground'

function App() {
  const [tab, setTab] = useState<Tab>('modules')
  const [selected, setSelected] = useState<ModuleInfo | null>(null)

  return (
    <div className="min-h-screen bg-neutral-950 text-neutral-200">
      {/* Header */}
      <header className="sticky top-0 z-50 border-b border-neutral-800 bg-neutral-950/80 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-gradient-to-br from-blue-500 to-purple-600 text-lg font-bold text-white">
              L
            </div>
            <div>
              <h1 className="text-lg font-bold text-white">ling-base</h1>
              <p className="text-xs text-neutral-500">Go 多模块基础库 · 175+ modules</p>
            </div>
          </div>
          <nav className="flex gap-1 rounded-lg bg-neutral-900 p-1">
            <button
              onClick={() => setTab('modules')}
              className={`rounded-md px-4 py-1.5 text-sm font-medium transition ${
                tab === 'modules' ? 'bg-neutral-700 text-white' : 'text-neutral-400 hover:text-white'
              }`}
            >
              模块浏览
            </button>
            <button
              onClick={() => setTab('playground')}
              className={`rounded-md px-4 py-1.5 text-sm font-medium transition ${
                tab === 'playground' ? 'bg-neutral-700 text-white' : 'text-neutral-400 hover:text-white'
              }`}
            >
              API Playground
            </button>
          </nav>
          <a
            href="https://github.com/LingByte/ling-base"
            target="_blank"
            rel="noopener"
            className="text-neutral-400 hover:text-white"
          >
            <svg className="h-6 w-6" fill="currentColor" viewBox="0 0 24 24">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
            </svg>
          </a>
        </div>
      </header>

      {/* Content */}
      <main className="mx-auto max-w-6xl px-6 py-8">
        {tab === 'modules' ? (
          <>
            {/* Hero */}
            <div className="mb-12 text-center">
              <h2 className="mb-3 text-4xl font-bold text-white">
                Go 基础工具库
              </h2>
              <p className="mx-auto max-w-2xl text-neutral-400">
                多模块按需引入，覆盖 AI 中继、语音处理、安全认证、基础设施、第三方服务对接等场景。
                业务只 import 用到的模块，不会拉入无关 SDK。
              </p>
              <div className="mt-6 flex justify-center gap-3">
                <code className="rounded-lg bg-neutral-900 px-4 py-2 text-sm text-green-400">
                  go get github.com/LingByte/ling-base/relay
                </code>
              </div>
            </div>

            {/* Categories */}
            {categories.map((cat) => (
              <section key={cat.name} className="mb-10">
                <div className="mb-4 flex items-center gap-2">
                  <span className="text-2xl">{cat.icon}</span>
                  <h3 className="text-xl font-bold text-white">{cat.name}</h3>
                  <span className="text-sm text-neutral-500">— {cat.description}</span>
                </div>
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
                  {cat.modules.map((mod) => (
                    <ModuleCard
                      key={mod.path}
                      module={mod}
                      onClick={() => setSelected(mod)}
                    />
                  ))}
                </div>
              </section>
            ))}
          </>
        ) : (
          <Playground />
        )}
      </main>

      {/* Module detail modal */}
      {selected && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
          onClick={() => setSelected(null)}
        >
          <div
            className="max-h-[80vh] w-full max-w-2xl overflow-auto rounded-2xl border border-neutral-800 bg-neutral-900 p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-4 flex items-start justify-between">
              <div>
                <h3 className="text-xl font-bold text-white">{selected.name}</h3>
                <code className="text-sm text-blue-400">{selected.path}</code>
              </div>
              <button
                onClick={() => setSelected(null)}
                className="text-neutral-500 hover:text-white"
              >
                ✕
              </button>
            </div>
            <p className="mb-4 text-neutral-300">{selected.description}</p>
            {selected.providers && (
              <div className="mb-4">
                <h4 className="mb-2 text-sm font-semibold text-neutral-400">Providers</h4>
                <div className="flex flex-wrap gap-2">
                  {selected.providers.map((p) => (
                    <span
                      key={p}
                      className="rounded-md bg-neutral-800 px-2.5 py-1 text-xs text-neutral-300"
                    >
                      {p}
                    </span>
                  ))}
                </div>
              </div>
            )}
            {selected.example && (
              <div className="mb-4">
                <h4 className="mb-2 text-sm font-semibold text-neutral-400">用法示例</h4>
                <pre className="overflow-auto rounded-lg bg-neutral-950 p-4 text-sm text-green-400">
                  <code>{selected.example}</code>
                </pre>
              </div>
            )}
            <div className="flex gap-3">
              <a
                href={`https://github.com/LingByte/ling-base/tree/main/${
                  selected.path.replace('github.com/LingByte/ling-base/', '')
                }`}
                target="_blank"
                rel="noopener"
                className="rounded-lg bg-neutral-800 px-4 py-2 text-sm text-white hover:bg-neutral-700"
              >
                查看源码
              </a>
              <button
                onClick={() => {
                  navigator.clipboard.writeText(selected.path)
                }}
                className="rounded-lg bg-neutral-800 px-4 py-2 text-sm text-white hover:bg-neutral-700"
              >
                复制 import 路径
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Footer */}
      <footer className="border-t border-neutral-800 py-6 text-center text-sm text-neutral-600">
        <p>ling-base · MIT License · <a href="https://github.com/LingByte/ling-base" className="hover:text-neutral-400">GitHub</a></p>
      </footer>
    </div>
  )
}

export default App
