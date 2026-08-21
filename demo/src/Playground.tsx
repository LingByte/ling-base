import { useState, useRef } from 'react'

interface Message {
  role: 'system' | 'user' | 'assistant'
  content: string
}

const PROVIDERS = [
  { name: 'OpenAI', baseURL: 'https://api.openai.com', models: ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-3.5-turbo'] },
  { name: 'Qiniu', baseURL: 'https://llmapi.qiniu.io', models: ['gpt-5.4-mini', 'gpt-5.4', 'grok-4.5'] },
  { name: 'DeepSeek', baseURL: 'https://api.deepseek.com', models: ['deepseek-chat', 'deepseek-coder', 'deepseek-reasoner'] },
  { name: 'Moonshot', baseURL: 'https://api.moonshot.cn', models: ['moonshot-v1-8k', 'moonshot-v1-32k', 'moonshot-v1-128k'] },
  { name: 'Zhipu', baseURL: 'https://open.bigmodel.cn', models: ['glm-4', 'glm-4-flash', 'glm-4v'] },
  { name: 'Custom', baseURL: '', models: [] },
]

export function Playground() {
  const [provider, setProvider] = useState(PROVIDERS[0])
  const [apiKey, setApiKey] = useState('')
  const [model, setModel] = useState('gpt-4o-mini')
  const [systemPrompt, setSystemPrompt] = useState('You are a helpful assistant.')
  const [input, setInput] = useState('Hello! What is ling-base?')
  const [messages, setMessages] = useState<Message[]>([])
  const [streaming, setStreaming] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [useStream, setUseStream] = useState(true)
  const abortRef = useRef<AbortController | null>(null)

  async function send() {
    if (!apiKey) {
      setError('请填写 API Key')
      return
    }
    if (!input.trim()) return

    setError('')
    setLoading(true)

    const userMsg: Message = { role: 'user', content: input }
    const allMsgs: Message[] = [
      { role: 'system', content: systemPrompt },
      ...messages,
      userMsg,
    ]

    setMessages((prev) => [...prev, userMsg])
    setInput('')

    const baseURL = provider.baseURL || apiKey.split('|')[1] || ''
    const key = apiKey.split('|')[0] || apiKey
    const url = `${baseURL}/v1/chat/completions`

    try {
      if (useStream) {
        await streamChat(url, key, model, allMsgs)
      } else {
        await chat(url, key, model, allMsgs)
      }
    } catch (e) {
      if (e instanceof Error && e.name === 'AbortError') {
        // user cancelled
      } else {
        setError(e instanceof Error ? e.message : String(e))
      }
    } finally {
      setLoading(false)
      setStreaming(false)
      abortRef.current = null
    }
  }

  async function chat(url: string, key: string, model: string, msgs: Message[]) {
    const resp = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${key}`,
      },
      body: JSON.stringify({ model, messages: msgs, stream: false }),
    })
    if (!resp.ok) {
      const body = await resp.text()
      throw new Error(`HTTP ${resp.status}: ${body}`)
    }
    const data = await resp.json()
    const content = data.choices?.[0]?.message?.content || '(empty response)'
    setMessages((prev) => [...prev, { role: 'assistant', content }])
  }

  async function streamChat(url: string, key: string, model: string, msgs: Message[]) {
    const ac = new AbortController()
    abortRef.current = ac
    setStreaming(true)

    const resp = await fetch(url, {
      method: 'POST',
      signal: ac.signal,
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${key}`,
      },
      body: JSON.stringify({ model, messages: msgs, stream: true }),
    })
    if (!resp.ok) {
      const body = await resp.text()
      throw new Error(`HTTP ${resp.status}: ${body}`)
    }

    const reader = resp.body?.getReader()
    if (!reader) throw new Error('No response body')

    const decoder = new TextDecoder()
    let buffer = ''
    let assistantContent = ''

    // Add empty assistant message that we'll update
    setMessages((prev) => [...prev, { role: 'assistant', content: '' }])

    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (!line.startsWith('data: ')) continue
        const data = line.slice(6).trim()
        if (data === '[DONE]') continue
        try {
          const json = JSON.parse(data)
          const delta = json.choices?.[0]?.delta?.content
          if (delta) {
            assistantContent += delta
            setMessages((prev) => {
              const copy = [...prev]
              copy[copy.length - 1] = { role: 'assistant', content: assistantContent }
              return copy
            })
          }
        } catch {
          // skip invalid JSON
        }
      }
    }
  }

  function stop() {
    abortRef.current?.abort()
  }

  function clear() {
    setMessages([])
    setError('')
  }

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-[360px_1fr]">
      {/* Config panel */}
      <div className="space-y-4 rounded-xl border border-neutral-800 bg-neutral-900 p-5">
        <h3 className="font-bold text-white">配置</h3>

        {/* Provider */}
        <div>
          <label className="mb-1 block text-sm text-neutral-400">Provider</label>
          <select
            value={provider.name}
            onChange={(e) => {
              const p = PROVIDERS.find((x) => x.name === e.target.value)!
              setProvider(p)
              if (p.models.length > 0) setModel(p.models[0])
            }}
            className="w-full rounded-lg bg-neutral-800 px-3 py-2 text-sm text-white outline-none"
          >
            {PROVIDERS.map((p) => (
              <option key={p.name} value={p.name}>{p.name}</option>
            ))}
          </select>
        </div>

        {/* API Key */}
        <div>
          <label className="mb-1 block text-sm text-neutral-400">
            API Key
            <span className="ml-2 text-xs text-neutral-600">仅存浏览器，不会上传</span>
          </label>
          <input
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder="sk-..."
            className="w-full rounded-lg bg-neutral-800 px-3 py-2 text-sm text-white outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>

        {/* Custom base URL */}
        {provider.name === 'Custom' && (
          <div>
            <label className="mb-1 block text-sm text-neutral-400">Base URL</label>
            <input
              type="text"
              value={provider.baseURL}
              onChange={(e) => setProvider({ ...provider, baseURL: e.target.value })}
              placeholder="https://api.example.com"
              className="w-full rounded-lg bg-neutral-800 px-3 py-2 text-sm text-white outline-none"
            />
          </div>
        )}

        {/* Model */}
        <div>
          <label className="mb-1 block text-sm text-neutral-400">Model</label>
          {provider.models.length > 0 ? (
            <select
              value={model}
              onChange={(e) => setModel(e.target.value)}
              className="w-full rounded-lg bg-neutral-800 px-3 py-2 text-sm text-white outline-none"
            >
              {provider.models.map((m) => (
                <option key={m} value={m}>{m}</option>
              ))}
            </select>
          ) : (
            <input
              type="text"
              value={model}
              onChange={(e) => setModel(e.target.value)}
              className="w-full rounded-lg bg-neutral-800 px-3 py-2 text-sm text-white outline-none"
            />
          )}
        </div>

        {/* System prompt */}
        <div>
          <label className="mb-1 block text-sm text-neutral-400">System Prompt</label>
          <textarea
            value={systemPrompt}
            onChange={(e) => setSystemPrompt(e.target.value)}
            rows={3}
            className="w-full rounded-lg bg-neutral-800 px-3 py-2 text-sm text-white outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>

        {/* Stream toggle */}
        <label className="flex items-center gap-2 text-sm text-neutral-400">
          <input
            type="checkbox"
            checked={useStream}
            onChange={(e) => setUseStream(e.target.checked)}
            className="rounded"
          />
          流式输出 (SSE)
        </label>

        <div className="rounded-lg bg-neutral-950 p-3 text-xs text-neutral-500">
          <p className="mb-1 font-semibold text-neutral-400">💡 说明</p>
          <p>Playground 直接从浏览器调用 provider API，不经过任何中间服务器。
          API Key 仅存在浏览器内存中，不会发送到 ling-base 或任何第三方。</p>
        </div>
      </div>

      {/* Chat panel */}
      <div className="flex flex-col rounded-xl border border-neutral-800 bg-neutral-900">
        {/* Messages */}
        <div className="flex-1 overflow-auto p-5 space-y-4 min-h-[400px] max-h-[600px]">
          {messages.length === 0 && (
            <div className="flex h-full items-center justify-center text-neutral-600">
              <p>输入消息开始对话 →</p>
            </div>
          )}
          {messages.map((msg, i) => (
            <div
              key={i}
              className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
            >
              <div
                className={`max-w-[80%] rounded-2xl px-4 py-3 text-sm ${
                  msg.role === 'user'
                    ? 'bg-blue-600 text-white'
                    : msg.role === 'assistant'
                    ? 'bg-neutral-800 text-neutral-200'
                    : 'bg-neutral-850 text-neutral-400 text-xs'
                }`}
              >
                {msg.role === 'system' && <span className="text-neutral-500">[system] </span>}
                <pre className="whitespace-pre-wrap break-words font-sans">{msg.content || '...'}</pre>
              </div>
            </div>
          ))}
          {streaming && (
            <div className="flex items-center gap-2 text-xs text-neutral-500">
              <span className="animate-pulse">●</span> 生成中...
            </div>
          )}
        </div>

        {/* Error */}
        {error && (
          <div className="mx-5 mb-3 rounded-lg bg-red-950 border border-red-800 px-4 py-2 text-sm text-red-300">
            {error}
          </div>
        )}

        {/* Input */}
        <div className="border-t border-neutral-800 p-4">
          <div className="flex gap-2">
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  if (!loading) send()
                }
              }}
              rows={2}
              placeholder="输入消息... (Enter 发送, Shift+Enter 换行)"
              className="flex-1 rounded-lg bg-neutral-800 px-3 py-2 text-sm text-white outline-none focus:ring-1 focus:ring-blue-500"
            />
            <div className="flex flex-col gap-2">
              {loading ? (
                <button
                  onClick={stop}
                  className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
                >
                  停止
                </button>
              ) : (
                <button
                  onClick={send}
                  disabled={!input.trim()}
                  className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-40"
                >
                  发送
                </button>
              )}
              <button
                onClick={clear}
                className="rounded-lg bg-neutral-800 px-4 py-2 text-sm text-neutral-400 hover:text-white"
              >
                清空
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
