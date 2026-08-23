'use client';

import { useEffect, useState, useCallback } from 'react';
import { basePath } from '@/lib/shared';

type WasmState = 'idle' | 'loading' | 'ready' | 'error';

let wasmState: WasmState = 'idle';
let wasmPromise: Promise<void> | null = null;
let goExited = false;

export function useWasm(enabled = true) {
  const [state, setState] = useState<WasmState>(wasmState);

  useEffect(() => {
    if (!enabled) return;
    if (wasmState === 'ready') {
      setState('ready');
      return;
    }
    if (wasmState === 'loading' && wasmPromise) {
      wasmPromise.then(() => setState('ready')).catch(() => setState('error'));
      return;
    }
    if (wasmState === 'idle') {
      wasmState = 'loading';
      setState('loading');
      wasmPromise = loadWasm();
      wasmPromise
        .then(() => {
          wasmState = 'ready';
          setState('ready');
        })
        .catch((err) => {
          console.error('WASM load error:', err);
          wasmState = 'error';
          setState('error');
        });
    }
  }, [enabled]);

  return {
    loaded: state === 'ready',
    loading: state === 'loading',
    error: state === 'error' ? 'WASM 加载失败' : null,
  };
}

function loadWasm(): Promise<void> {
  return new Promise<void>((resolve, reject) => {
    // 1. 动态加载 wasm_exec.js
    const existingScript = document.getElementById('wasm-exec-script');
    const scriptPromise = existingScript
      ? Promise.resolve()
      : new Promise<void>((res, rej) => {
          const script = document.createElement('script');
          script.id = 'wasm-exec-script';
          script.src = `${basePath}/wasm_exec.js`;
          script.onload = () => res();
          script.onerror = () => rej(new Error('Failed to load wasm_exec.js'));
          document.head.appendChild(script);
        });

    scriptPromise
      .then(async () => {
        const GoClass = (window as any).Go;
        if (!GoClass) {
          reject(new Error('Go class not found after loading wasm_exec.js'));
          return;
        }

        const go = new GoClass();

        // 监听 Go 程序退出
        go.run = ((originalRun: any) => {
          return function (this: any, instance: any) {
            const p = originalRun.call(this, instance);
            p.catch?.((e: any) => {
              console.error('Go program exited:', e);
              goExited = true;
            });
            p.then?.(() => {
              console.warn('Go program exited normally');
              goExited = true;
            });
            return p;
          };
        })(go.run.bind(go));

        // 2. 加载 WASM
        const resp = await fetch(`${basePath}/lingbase.wasm`);
        if (!resp.ok) {
          reject(new Error(`Failed to fetch WASM: ${resp.status}`));
          return;
        }

        const wasmBuffer = await resp.arrayBuffer();
        const wasmModule = await WebAssembly.instantiate(wasmBuffer, go.importObject);

        // 3. 运行 Go 程序
        go.run(wasmModule.instance);

        // 4. 等待 lingbaseWasmReady 标志
        const waitForReady = (timeout = 10000) => {
          const start = Date.now();
          return new Promise<void>((res, rej) => {
            const check = () => {
              if (goExited) {
                rej(new Error('Go program exited before ready'));
                return;
              }
              if ((window as any).lingbaseWasmReady === true) {
                res();
                return;
              }
              if (Date.now() - start > timeout) {
                rej(new Error('WASM ready timeout'));
                return;
              }
              setTimeout(check, 50);
            };
            check();
          });
        };

        await waitForReady();
        resolve();
      })
      .catch(reject);
  });
}

export function callWasm(fnName: string, ...args: any[]): Promise<any> {
  return new Promise((resolve, reject) => {
    const doCall = () => {
      if (goExited) {
        reject(new Error('Go program has already exited'));
        return;
      }

      const fn = (window as any)[fnName];
      if (!fn) {
        reject(new Error(`WASM function ${fnName} not found`));
        return;
      }

      try {
        const result = fn(...args);

        // 处理 undefined 返回值
        if (result === undefined || result === null) {
          reject(new Error(`WASM function ${fnName} returned undefined (program may have exited)`));
          return;
        }

        const jsonStr = typeof result === 'string' ? result : String(result);
        try {
          resolve(JSON.parse(jsonStr));
        } catch {
          reject(new Error(`Invalid JSON from ${fnName}: ${jsonStr}`));
        }
      } catch (e) {
        // "Go program has already exited" 会在这里被捕获
        reject(e instanceof Error ? e : new Error(String(e)));
      }
    };

    if (wasmState === 'ready') {
      doCall();
    } else if (wasmPromise) {
      wasmPromise.then(() => doCall()).catch(() => reject(new Error('WASM not loaded')));
    } else {
      reject(new Error('WASM not loaded'));
    }
  });
}
