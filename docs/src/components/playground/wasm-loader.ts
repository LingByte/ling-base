'use client';

import { useEffect, useState, useCallback } from 'react';

type WasmState = 'idle' | 'loading' | 'ready' | 'error';

let wasmState: WasmState = 'idle';
let wasmPromise: Promise<void> | null = null;

export function useWasm() {
  const [state, setState] = useState<WasmState>(wasmState);

  useEffect(() => {
    if (wasmState === 'ready' || wasmState === 'loading') {
      setState(wasmState);
      if (wasmState === 'loading' && wasmPromise) {
        wasmPromise.then(() => setState('ready')).catch(() => setState('error'));
      }
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
          wasmState = 'error';
          setState('error');
          console.error('WASM load error:', err);
        });
    }
  }, []);

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
          script.src = '/wasm_exec.js';
          script.onload = () => res();
          script.onerror = () => rej(new Error('Failed to load wasm_exec.js'));
          document.head.appendChild(script);
        });

    scriptPromise
      .then(async () => {
        // 2. 等待 Go 构造函数可用
        const GoClass = (window as any).Go;
        if (!GoClass) {
          reject(new Error('Go class not found after loading wasm_exec.js'));
          return;
        }

        const go = new GoClass();

        // 3. 加载 WASM
        const resp = await fetch('/lingbase.wasm');
        if (!resp.ok) {
          reject(new Error(`Failed to fetch WASM: ${resp.status}`));
          return;
        }

        const wasmBuffer = await resp.arrayBuffer();
        const wasmModule = await WebAssembly.instantiate(wasmBuffer, go.importObject);

        // 4. 运行 Go 程序（不 await — go.run 返回的 promise 在程序退出时才 resolve）
        go.run(wasmModule.instance);

        // 5. 等待 lingbaseWasmReady 标志
        const waitForReady = () => {
          if ((window as any).lingbaseWasmReady === true) {
            resolve();
            return;
          }
          setTimeout(waitForReady, 50);
        };
        waitForReady();
      })
      .catch(reject);
  });
}

export function callWasm(fnName: string, ...args: any[]): Promise<any> {
  return new Promise((resolve, reject) => {
    if (wasmState !== 'ready') {
      if (wasmPromise) {
        wasmPromise
          .then(() => {
            const fn = (window as any)[fnName];
            if (!fn) {
              reject(new Error(`WASM function ${fnName} not found`));
              return;
            }
            try {
              const result = fn(...args);
              const jsonStr = typeof result === 'string' ? result : String(result);
              resolve(JSON.parse(jsonStr));
            } catch (e) {
              reject(e);
            }
          })
          .catch(() => reject(new Error('WASM not loaded')));
      } else {
        reject(new Error('WASM not loaded'));
      }
      return;
    }

    const fn = (window as any)[fnName];
    if (!fn) {
      reject(new Error(`WASM function ${fnName} not found`));
      return;
    }
    try {
      const result = fn(...args);
      const jsonStr = typeof result === 'string' ? result : String(result);
      resolve(JSON.parse(jsonStr));
    } catch (e) {
      reject(e);
    }
  });
}
