'use client';

import { useState, useCallback, useEffect } from 'react';
import { callWasm } from '../wasm-loader';
import { ResultBox, RunButton, inputClass, FieldLabel } from '../shared';

export function TOTPDemo() {
  const [code, setCode] = useState('');
  const [secret, setSecret] = useState('');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState<string | null>(null);

  const validate = useCallback(async () => {
    if (!code || !secret) { setError('请输入 code 和 secret'); return; }
    setLoading('validate'); setError(null); setResult(null);
    try {
      setResult(await callWasm('wasmTOTPValidate', code, secret));
    } catch (e) { setError(String(e)); } finally { setLoading(null); }
  }, [code, secret]);

  const currentCode = useCallback(async () => {
    if (!secret) { setError('请输入 secret'); return; }
    setLoading('current'); setError(null); setResult(null);
    try {
      setResult(await callWasm('wasmTOTPCurrentCode', secret));
    } catch (e) { setError(String(e)); } finally { setLoading(null); }
  }, [secret]);

  const backupCodes = useCallback(async () => {
    setLoading('backup'); setError(null); setResult(null);
    try {
      setResult(await callWasm('wasmTOTPBackupCodes', 10));
    } catch (e) { setError(String(e)); } finally { setLoading(null); }
  }, []);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">TOTP 两步验证</h3>
        <p className="text-sm text-fd-muted-foreground">验证码校验、当前验证码生成、备份码生成（浏览器 WASM）</p>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <input value={code} onChange={(e) => setCode(e.target.value)} placeholder="6 位验证码" className={inputClass} />
        <input value={secret} onChange={(e) => setSecret(e.target.value)} placeholder="Secret (Base32)" className={`${inputClass} font-mono text-xs`} />
      </div>
      <div className="flex flex-wrap gap-2">
        <RunButton onClick={currentCode} loading={loading === 'current'} label="生成当前验证码" />
        <RunButton onClick={validate} loading={loading === 'validate'} label="校验验证码" variant="secondary" />
        <RunButton onClick={backupCodes} loading={loading === 'backup'} label="生成备份码" variant="secondary" />
      </div>
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function CompressDemo() {
  const [input, setInput] = useState('Hello, ling-base! This is a test string for compression. '.repeat(10));
  const [algo, setAlgo] = useState('zstd');
  const [mode, setMode] = useState<'compress' | 'decompress'>('compress');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try {
      const fnMap: Record<string, [string, string]> = {
        zstd: ['wasmZstdCompress', 'wasmZstdDecompress'],
        snappy: ['wasmSnappyCompress', 'wasmSnappyDecompress'],
        lz4: ['wasmLZ4Compress', 'wasmLZ4Decompress'],
        gzip: ['wasmGzipCompress', 'wasmGzipDecompress'],
      };
      const [compressFn, decompressFn] = fnMap[algo];
      const fn = mode === 'compress' ? compressFn : decompressFn;
      const data = Array.from(new TextEncoder().encode(input));
      setResult(await callWasm(fn, data));
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [input, algo, mode]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">数据压缩</h3>
        <p className="text-sm text-fd-muted-foreground">zstd / snappy / lz4 / gzip — 浏览器 WASM</p>
      </div>
      <div className="flex gap-3">
        <label className="flex-1">
          <FieldLabel>算法</FieldLabel>
          <select value={algo} onChange={(e) => setAlgo(e.target.value)} className={inputClass}>
            <option value="zstd">Zstd</option>
            <option value="snappy">Snappy</option>
            <option value="lz4">LZ4</option>
            <option value="gzip">Gzip</option>
          </select>
        </label>
        <label className="flex-1">
          <FieldLabel>模式</FieldLabel>
          <select value={mode} onChange={(e) => setMode(e.target.value as 'compress' | 'decompress')} className={inputClass}>
            <option value="compress">压缩</option>
            <option value="decompress">解压</option>
          </select>
        </label>
      </div>
      <label className="block">
        <FieldLabel>输入数据</FieldLabel>
        <textarea value={input} onChange={(e) => setInput(e.target.value)} rows={4} className={`${inputClass} font-mono`} />
      </label>
      <RunButton onClick={run} loading={loading} />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function HashDemo() {
  const [input, setInput] = useState('hello, ling-base');
  const [algo, setAlgo] = useState('sha256');
  const [hmacKey, setHmacKey] = useState('secret-key');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try {
      const args = algo === 'hmac-sha256' ? [algo, input, hmacKey] : [algo, input];
      setResult(await callWasm('wasmHash', ...args));
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [algo, input, hmacKey]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">哈希计算</h3>
        <p className="text-sm text-fd-muted-foreground">common/hash — MD5 / SHA / HMAC</p>
      </div>
      <label className="block">
        <FieldLabel>算法</FieldLabel>
        <select value={algo} onChange={(e) => setAlgo(e.target.value)} className={inputClass}>
          <option value="md5">MD5</option>
          <option value="sha1">SHA-1</option>
          <option value="sha256">SHA-256</option>
          <option value="sha512">SHA-512</option>
          <option value="hmac-sha256">HMAC-SHA256</option>
        </select>
      </label>
      <label className="block">
        <FieldLabel>输入</FieldLabel>
        <input value={input} onChange={(e) => setInput(e.target.value)} className={`${inputClass} font-mono`} />
      </label>
      {algo === 'hmac-sha256' && (
        <label className="block">
          <FieldLabel>HMAC Key</FieldLabel>
          <input value={hmacKey} onChange={(e) => setHmacKey(e.target.value)} className={inputClass} />
        </label>
      )}
      <RunButton onClick={run} loading={loading} label="计算哈希" />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function PasswordDemo() {
  const [plain, setPlain] = useState('my-secret-password');
  const [hash, setHash] = useState('');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const doHash = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try {
      const r = await callWasm('wasmPasswordHash', plain) as { hash?: string };
      if (r.hash) setHash(r.hash);
      setResult(r);
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [plain]);

  const doVerify = useCallback(async () => {
    if (!hash) { setError('请先哈希或粘贴 hash'); return; }
    setLoading(true); setError(null); setResult(null);
    try {
      setResult(await callWasm('wasmPasswordVerify', plain, hash));
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [plain, hash]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">密码哈希</h3>
        <p className="text-sm text-fd-muted-foreground">common/password — Argon2id 默认</p>
      </div>
      <label className="block">
        <FieldLabel>明文密码</FieldLabel>
        <input type="password" value={plain} onChange={(e) => setPlain(e.target.value)} className={inputClass} />
      </label>
      <label className="block">
        <FieldLabel>存储 Hash（可粘贴）</FieldLabel>
        <textarea value={hash} onChange={(e) => setHash(e.target.value)} rows={2} className={`${inputClass} font-mono text-xs`} />
      </label>
      <div className="flex gap-2">
        <RunButton onClick={doHash} loading={loading} label="生成 Hash" />
        <RunButton onClick={doVerify} loading={loading} label="校验密码" variant="secondary" />
      </div>
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function ValidateDemo() {
  const [value, setValue] = useState('user@example.com');
  const [tag, setTag] = useState('required,email');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try {
      setResult(await callWasm('wasmValidate', value, tag));
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [value, tag]);

  const presets = [
    { label: 'Email', value: 'user@example.com', tag: 'required,email' },
    { label: '手机号', value: '13800138000', tag: 'required,phone' },
    { label: '长度', value: 'abc', tag: 'min=5,max=20' },
  ];

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">数据校验</h3>
        <p className="text-sm text-fd-muted-foreground">common/validate — 标签规则校验</p>
      </div>
      <div className="flex flex-wrap gap-2">
        {presets.map((p) => (
          <button key={p.label} type="button" onClick={() => { setValue(p.value); setTag(p.tag); }}
            className="rounded-md border border-fd-border px-2 py-1 text-xs hover:bg-fd-accent">{p.label}</button>
        ))}
      </div>
      <label className="block">
        <FieldLabel>值</FieldLabel>
        <input value={value} onChange={(e) => setValue(e.target.value)} className={inputClass} />
      </label>
      <label className="block">
        <FieldLabel>校验标签</FieldLabel>
        <input value={tag} onChange={(e) => setTag(e.target.value)} className={`${inputClass} font-mono`} placeholder="required,email" />
      </label>
      <RunButton onClick={run} loading={loading} label="校验" />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function JwtDemo() {
  const [secret, setSecret] = useState('my-32-byte-secret-1234567890123456');
  const [subject, setSubject] = useState('user-123');
  const [token, setToken] = useState('');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const login = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try {
      const r = await callWasm('wasmJWTLogin', secret, subject) as { accessToken?: string };
      if (r.accessToken) setToken(r.accessToken);
      setResult(r);
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [secret, subject]);

  const verify = useCallback(async () => {
    if (!token) { setError('请先生成或粘贴 token'); return; }
    setLoading(true); setError(null); setResult(null);
    try {
      setResult(await callWasm('wasmJWTVerify', secret, token));
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [secret, token]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">JWT 令牌</h3>
        <p className="text-sm text-fd-muted-foreground">common/jwtutil — 签发与验证</p>
      </div>
      <label className="block">
        <FieldLabel>Secret（≥32 字节）</FieldLabel>
        <input value={secret} onChange={(e) => setSecret(e.target.value)} className={`${inputClass} font-mono text-xs`} />
      </label>
      <label className="block">
        <FieldLabel>Subject (sub)</FieldLabel>
        <input value={subject} onChange={(e) => setSubject(e.target.value)} className={inputClass} />
      </label>
      <label className="block">
        <FieldLabel>Access Token</FieldLabel>
        <textarea value={token} onChange={(e) => setToken(e.target.value)} rows={3} className={`${inputClass} font-mono text-xs`} />
      </label>
      <div className="flex gap-2">
        <RunButton onClick={login} loading={loading} label="签发 Token" />
        <RunButton onClick={verify} loading={loading} label="验证 Token" variant="secondary" />
      </div>
      <ResultBox result={result} error={error} />
    </div>
  );
}

type QRTemplateMeta = { id: string; name: string; category: string };

const QR_TEMPLATE_TABS: { id: string; label: string }[] = [
  { id: 'simple', label: '黑白简约' },
  { id: 'classic', label: '经典绚丽' },
  { id: 'creative', label: '创意样式' },
];

export function QRCodeDemo() {
  const [mode, setMode] = useState<'standard' | 'fancy' | 'template'>('template');
  const [text, setText] = useState('https://github.com/LingByte/ling-base');
  const [size, setSize] = useState(256);
  const [moduleShape, setModuleShape] = useState('circle');
  const [finderShape, setFinderShape] = useState('rounded');
  const [level, setLevel] = useState('quartile');
  const [fgColor, setFgColor] = useState('#1e40af');
  const [bgColor, setBgColor] = useState('#ffffff');
  const [bgTransparent, setBgTransparent] = useState(false);
  const [moduleWidth, setModuleWidth] = useState(21);
  const [borderWidth, setBorderWidth] = useState(20);
  const [useGradient, setUseGradient] = useState(false);
  const [gradAngle, setGradAngle] = useState(45);
  const [gradStart, setGradStart] = useState('#1e40af');
  const [gradEnd, setGradEnd] = useState('#7c3aed');
  const [templateCategory, setTemplateCategory] = useState('simple');
  const [templates, setTemplates] = useState<QRTemplateMeta[]>([]);
  const [templateId, setTemplateId] = useState('simple-dots');
  const [dataURL, setDataURL] = useState('');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (mode !== 'template') return;
    let cancelled = false;
    (async () => {
      try {
        const r = await callWasm('wasmQRCodeTemplates', templateCategory) as { templates?: QRTemplateMeta[] };
        if (cancelled) return;
        const list = r.templates ?? [];
        setTemplates(list);
        if (list.length) setTemplateId(list[0].id);
      } catch {
        if (!cancelled) setTemplates([]);
      }
    })();
    return () => { cancelled = true; };
  }, [mode, templateCategory]);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null); setDataURL('');
    try {
      let r: { dataURL?: string };
      if (mode === 'standard') {
        r = await callWasm('wasmQRCode', text, size) as { dataURL?: string };
      } else if (mode === 'template') {
        r = await callWasm('wasmQRCodeFromTemplate', text, templateId) as { dataURL?: string };
      } else {
        const opts: Record<string, unknown> = {
          module: moduleShape,
          finder: finderShape,
          fgColor: fgColor,
          bgColor: bgColor,
          bgTransparent,
          level,
          moduleWidth,
          borderWidth,
        };
        if (useGradient) {
          opts.gradient = {
            angle: gradAngle,
            stops: [
              { t: 0, color: gradStart },
              { t: 1, color: gradEnd },
            ],
          };
        }
        r = await callWasm('wasmQRCodeFancy', text, JSON.stringify(opts)) as { dataURL?: string };
      }
      if (r.dataURL) setDataURL(r.dataURL);
      setResult(mode === 'template' ? { mode, text, templateId } : { mode, text });
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [mode, text, size, templateId, moduleShape, finderShape, level, fgColor, bgColor, bgTransparent, moduleWidth, borderWidth, useGradient, gradAngle, gradStart, gradEnd]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">二维码生成</h3>
        <p className="text-sm text-fd-muted-foreground">common/qrcode — 标准 / 花式参数 / 命名模版</p>
      </div>

      <div className="flex flex-wrap gap-2">
        {([
          ['standard', '标准 QR'],
          ['fancy', '自定义花式'],
          ['template', '模版库'],
        ] as const).map(([m, label]) => (
          <button key={m} type="button" onClick={() => setMode(m)}
            className={`rounded-lg px-3 py-1.5 text-sm font-medium transition ${mode === m ? 'bg-fd-primary text-fd-primary-foreground' : 'border border-fd-border hover:bg-fd-accent'}`}>
            {label}
          </button>
        ))}
      </div>

      <label className="block">
        <FieldLabel>内容</FieldLabel>
        <input value={text} onChange={(e) => setText(e.target.value)} className={inputClass} />
      </label>

      {mode === 'standard' && (
        <label className="block">
          <FieldLabel>尺寸 (px)</FieldLabel>
          <input type="number" value={size} onChange={(e) => setSize(Number(e.target.value))} className={inputClass} min={64} max={512} />
        </label>
      )}

      {mode === 'template' && (
        <div className="space-y-3 rounded-lg border border-fd-border p-3">
          <div className="flex flex-wrap gap-2">
            {QR_TEMPLATE_TABS.map((tab) => (
              <button key={tab.id} type="button" onClick={() => setTemplateCategory(tab.id)}
                className={`rounded-md px-2.5 py-1 text-xs font-medium transition ${templateCategory === tab.id ? 'bg-fd-primary text-fd-primary-foreground' : 'border border-fd-border hover:bg-fd-accent'}`}>
                {tab.label}
              </button>
            ))}
          </div>
          <div className="grid grid-cols-3 gap-2 sm:grid-cols-4">
            {templates.map((tmpl) => (
              <button
                key={tmpl.id}
                type="button"
                onClick={() => setTemplateId(tmpl.id)}
                className={`rounded-md border px-2 py-2 text-left text-xs transition ${templateId === tmpl.id ? 'border-fd-primary bg-fd-accent' : 'border-fd-border hover:bg-fd-accent/50'}`}
              >
                <div className="font-medium">{tmpl.name}</div>
                <div className="mt-0.5 truncate font-mono text-[10px] text-fd-muted-foreground">{tmpl.id}</div>
              </button>
            ))}
          </div>
        </div>
      )}

      {mode === 'fancy' && (
        <div className="space-y-3 rounded-lg border border-fd-border p-3">
          <div className="grid grid-cols-2 gap-3">
            <label>
              <FieldLabel>模块形状</FieldLabel>
              <select value={moduleShape} onChange={(e) => setModuleShape(e.target.value)} className={inputClass}>
                <option value="rectangle">矩形 (Rectangle)</option>
                <option value="circle">圆形 (Circle)</option>
                <option value="rounded">圆角 (Rounded)</option>
                <option value="liquid">液态 (Liquid)</option>
                <option value="hstripe">横条纹 (HStripe)</option>
                <option value="vstripe">竖条纹 (VStripe)</option>
                <option value="diamond">菱形 (Diamond)</option>
              </select>
            </label>
            <label>
              <FieldLabel>定位点形状</FieldLabel>
              <select value={finderShape} onChange={(e) => setFinderShape(e.target.value)} className={inputClass}>
                <option value="square">方形 (Square)</option>
                <option value="rounded">圆角 (Rounded)</option>
              </select>
            </label>
            <label>
              <FieldLabel>纠错等级</FieldLabel>
              <select value={level} onChange={(e) => setLevel(e.target.value)} className={inputClass}>
                <option value="low">Low (~7%)</option>
                <option value="medium">Medium (~15%)</option>
                <option value="quartile">Quartile (~25%)</option>
                <option value="high">High (~30%)</option>
              </select>
            </label>
            <label>
              <FieldLabel>模块宽度</FieldLabel>
              <input type="number" value={moduleWidth} onChange={(e) => setModuleWidth(Number(e.target.value))} className={inputClass} min={8} max={40} />
            </label>
            <label>
              <FieldLabel>前景色</FieldLabel>
              <input type="color" value={fgColor} onChange={(e) => setFgColor(e.target.value)} className="h-10 w-full cursor-pointer rounded-lg border border-fd-border" />
            </label>
            <label>
              <FieldLabel>背景色</FieldLabel>
              <input type="color" value={bgColor} onChange={(e) => setBgColor(e.target.value)} disabled={bgTransparent} className="h-10 w-full cursor-pointer rounded-lg border border-fd-border disabled:opacity-40" />
            </label>
            <label>
              <FieldLabel>边框宽度</FieldLabel>
              <input type="number" value={borderWidth} onChange={(e) => setBorderWidth(Number(e.target.value))} className={inputClass} min={0} max={80} />
            </label>
            <label className="flex items-end gap-2 pb-2">
              <input type="checkbox" checked={bgTransparent} onChange={(e) => setBgTransparent(e.target.checked)} className="size-4" />
              <span className="text-sm">透明背景</span>
            </label>
          </div>
          <label className="flex items-center gap-2">
            <input type="checkbox" checked={useGradient} onChange={(e) => setUseGradient(e.target.checked)} className="size-4" />
            <span className="text-sm font-medium">前景渐变</span>
          </label>
          {useGradient && (
            <div className="grid grid-cols-3 gap-3">
              <label>
                <FieldLabel>角度 (°)</FieldLabel>
                <input type="number" value={gradAngle} onChange={(e) => setGradAngle(Number(e.target.value))} className={inputClass} min={0} max={360} />
              </label>
              <label>
                <FieldLabel>起始色</FieldLabel>
                <input type="color" value={gradStart} onChange={(e) => setGradStart(e.target.value)} className="h-10 w-full cursor-pointer rounded-lg border border-fd-border" />
              </label>
              <label>
                <FieldLabel>结束色</FieldLabel>
                <input type="color" value={gradEnd} onChange={(e) => setGradEnd(e.target.value)} className="h-10 w-full cursor-pointer rounded-lg border border-fd-border" />
              </label>
            </div>
          )}
        </div>
      )}

      <RunButton onClick={run} loading={loading} label="生成 QR 码" />
      {dataURL && (
        <div className="flex justify-center rounded-lg border border-fd-border p-4" style={bgTransparent && mode === 'fancy' ? { background: 'repeating-conic-gradient(#ccc 0% 25%, #fff 0% 50%) 50% / 16px 16px' } : undefined}>
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={dataURL} alt="QR Code" width={220} height={220} className="max-w-full" />
        </div>
      )}
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function BarcodeDemo() {
  const [typ, setTyp] = useState('code128');
  const [content, setContent] = useState('LING-BASE-2026');
  const [width, setWidth] = useState(320);
  const [height, setHeight] = useState(120);
  const [dataURL, setDataURL] = useState('');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const presets: Record<string, string> = {
    code128: 'LING-BASE-2026',
    code39: 'CODE39',
    ean13: '6901234567892',
    ean8: '96385074',
    upca: '036000291452',
    datamatrix: 'ling-base',
    pdf417: 'Hello PDF417',
    aztec: 'Aztec Code',
  };

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null); setDataURL('');
    try {
      const r = await callWasm('wasmBarcode', typ, content, width, height) as { dataURL?: string };
      if (r.dataURL) setDataURL(r.dataURL);
      setResult({ type: typ, content, width, height });
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [typ, content, width, height]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">条形码生成</h3>
        <p className="text-sm text-fd-muted-foreground">common/barcode — 一维 / 二维条码</p>
      </div>
      <label className="block">
        <FieldLabel>条码类型</FieldLabel>
        <select value={typ} onChange={(e) => { setTyp(e.target.value); setContent(presets[e.target.value] ?? content); }} className={inputClass}>
          <option value="code128">Code 128</option>
          <option value="code39">Code 39</option>
          <option value="code93">Code 93</option>
          <option value="codabar">Codabar</option>
          <option value="ean13">EAN-13</option>
          <option value="ean8">EAN-8</option>
          <option value="upca">UPC-A</option>
          <option value="2of5">2 of 5</option>
          <option value="pdf417">PDF417</option>
          <option value="datamatrix">Data Matrix</option>
          <option value="aztec">Aztec</option>
        </select>
      </label>
      <label className="block">
        <FieldLabel>内容</FieldLabel>
        <input value={content} onChange={(e) => setContent(e.target.value)} className={`${inputClass} font-mono`} />
      </label>
      <div className="grid grid-cols-2 gap-3">
        <label>
          <FieldLabel>宽度 (px)</FieldLabel>
          <input type="number" value={width} onChange={(e) => setWidth(Number(e.target.value))} className={inputClass} min={100} max={600} />
        </label>
        <label>
          <FieldLabel>高度 (px)</FieldLabel>
          <input type="number" value={height} onChange={(e) => setHeight(Number(e.target.value))} className={inputClass} min={40} max={400} />
        </label>
      </div>
      <RunButton onClick={run} loading={loading} label="生成条形码" />
      {dataURL && (
        <div className="flex justify-center rounded-lg border border-fd-border bg-white p-4">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={dataURL} alt="Barcode" className="max-w-full" style={{ maxHeight: 200 }} />
        </div>
      )}
      <ResultBox result={result} error={error} />
    </div>
  );
}
