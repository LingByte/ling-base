'use client';

import { useEffect, useState, useCallback } from 'react';

let wasmLoaded = false;
let wasmLoading = false;
let wasmPromise: Promise<void> | null = null;

export function useWasm() {
  const [loaded, setLoaded] = useState(wasmLoaded);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (wasmLoaded) return;
    if (wasmLoading && wasmPromise) {
      wasmPromise.then(() => setLoaded(true)).catch(setError);
      return;
    }

    wasmLoading = true;
    wasmPromise = loadWasm();
    wasmPromise.then(() => {
      wasmLoaded = true;
      setLoaded(true);
    }).catch((err) => {
      setError(err instanceof Error ? err.message : String(err));
    });
  }, []);

  return { loaded, error };
}

async function loadWasm(): Promise<void> {
  // wasm_exec.js 注册了 Go 类到全局
  // 需要动态加载脚本
  await new Promise<void>((resolve, reject) => {
    const existing = document.getElementById('wasm-exec-script');
    if (existing) {
      resolve();
      return;
    }
    const script = document.createElement('script');
    script.id = 'wasm-exec-script';
    script.src = '/wasm_exec.js';
    script.onload = () => resolve();
    script.onerror = () => reject(new Error('Failed to load wasm_exec.js'));
    document.head.appendChild(script);
  });

  // 等待 Go 构造函数可用
  const go = new (window as any).Go();

  const resp = await fetch('/lingbase.wasm');
  if (!resp.ok) throw new Error(`Failed to fetch WASM: ${resp.status}`);

  const wasmModule = await WebAssembly.instantiateStreaming(resp, go.importObject);
  // 运行 Go WASM（不 await，因为它会阻塞直到 main 结束）
  go.run(wasmModule.instance);
}

export async function callWasm(fnName: string, ...args: any[]): Promise<any> {
  if (!wasmLoaded) {
    await wasmPromise;
  }
  const fn = (window as any)[fnName];
  if (!fn) throw new Error(`WASM function ${fnName} not found`);
  const result = fn(...args);
  const json = typeof result === 'string' ? result : JSON.stringify(result);
  return JSON.parse(json);
}
