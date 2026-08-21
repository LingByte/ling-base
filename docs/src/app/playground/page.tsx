import { Playground } from '@/components/playground/Playground';
import { Terminal, AlertTriangle } from 'lucide-react';

export const metadata = {
  title: 'Playground — ling-base',
  description: '在浏览器中直接运行 ling-base 模块，无需后端',
};

export default function PlaygroundPage() {
  return (
    <div className="flex flex-1 flex-col">
      <section className="px-6 pt-24 pb-8">
        <div className="mx-auto max-w-3xl">
          <div className="mb-6 flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-fd-primary/10">
              <Terminal className="size-6 text-fd-primary" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">Playground</h1>
              <p className="text-sm text-fd-muted-foreground">
                在浏览器中直接运行 ling-base 模块
              </p>
            </div>
          </div>

          <div className="mb-6 flex items-start gap-2 rounded-lg border border-amber-500/30 bg-amber-500/5 p-4 text-sm">
            <AlertTriangle className="size-4 shrink-0 text-amber-500 mt-0.5" />
            <div className="text-fd-muted-foreground">
              所有代码在浏览器中通过 WebAssembly 运行，无需后端。
              首次加载需要下载 ~6MB WASM 模块，之后会被浏览器缓存。
              当前支持 TOTP、密码哈希、数据压缩。
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto w-full max-w-3xl px-6 pb-24">
        <Playground />
      </section>
    </div>
  );
}
