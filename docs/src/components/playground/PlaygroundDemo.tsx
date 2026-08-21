'use client';

import { useState, useCallback } from 'react';
import { callWasm } from './wasm-loader';
import {
  Loader2,
  Play,
  CheckCircle2,
  XCircle,
  KeyRound,
  ShieldCheck,
  FileArchive,
} from 'lucide-react';

type DemoId = 'totp' | 'password' | 'compress';

export function PlaygroundDemo({ demoId }: { demoId: DemoId }) {
  switch (demoId) {
    case 'totp':
      return <TOTPDemo />;
    case 'password':
      return <PasswordDemo />;
    case 'compress':
      return <CompressDemo />;
  }
}

// ─── Shared UI ──────────────────────────────────────────

function ResultBox({ result, error }: { result: any; error: string | null }) {
  if (error) {
    return (
      <div className="mt-4 flex items-start gap-2 rounded-lg bg-red-500/10 p-3 text-sm text-red-500">
        <XCircle className="size-4 shrink-0 mt-0.5" />
        <pre className="whitespace-pre-wrap break-all">{error}</pre>
      </div>
    );
  }
  if (!result) return null;
  return (
    <div className="mt-4 rounded-lg bg-fd-secondary/50 p-3">
      <div className="mb-2 flex items-center gap-2 text-sm font-medium text-emerald-500">
        <CheckCircle2 className="size-4" />
        执行成功
      </div>
      <pre className="overflow-x-auto text-xs text-fd-foreground">
        {JSON.stringify(result, null, 2)}
      </pre>
    </div>
  );
}

function RunButton({ onClick, loading, label }: { onClick: () => void; loading: boolean; label?: string }) {
  return (
    <button
      onClick={onClick}
      disabled={loading}
      className="inline-flex items-center gap-2 rounded-lg bg-fd-primary px-4 py-2 text-sm font-medium text-fd-primary-foreground transition hover:bg-fd-primary/90 disabled:opacity-50"
    >
      {loading ? <Loader2 className="size-4 animate-spin" /> : <Play className="size-4" />}
      {label || '运行'}
    </button>
  );
}

function SecondaryButton({ onClick, loading, icon: Icon, label, disabled }: {
  onClick: () => void;
  loading: boolean;
  icon: typeof KeyRound;
  label: string;
  disabled?: boolean;
}) {
  return (
    <button
      onClick={onClick}
      disabled={loading || disabled}
      className="inline-flex items-center gap-2 rounded-lg border border-fd-border px-3 py-2 text-sm font-medium transition hover:bg-fd-accent disabled:opacity-50"
    >
      {loading ? <Loader2 className="size-4 animate-spin" /> : <Icon className="size-4" />}
      {label}
    </button>
  );
}

const inputClass = 'w-full rounded-lg border border-fd-border bg-fd-background px-3 py-2 text-sm';

// ─── TOTP Demo ──────────────────────────────────────────

function TOTPDemo() {
  const [issuer, setIssuer] = useState('ling-base');
  const [account, setAccount] = useState('user@example.com');
  const [code, setCode] = useState('');
  const [secret, setSecret] = useState('');
  const [result, setResult] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState<string | null>(null);

  const generate = useCallback(async () => {
    setLoading('generate'); setError(null); setResult(null);
    try {
      const r = await callWasm('wasmTOTPGenerate', issuer, account);
      setResult(r); setSecret(r.secret);
    } catch (e) { setError(String(e)); } finally { setLoading(null); }
  }, [issuer, account]);

  const validate = useCallback(async () => {
    if (!code || !secret) { setError('请先输入 code 和 secret'); return; }
    setLoading('validate'); setError(null); setResult(null);
    try {
      const r = await callWasm('wasmTOTPValidate', code, secret);
      setResult(r);
    } catch (e) { setError(String(e)); } finally { setLoading(null); }
  }, [code, secret]);

  const currentCode = useCallback(async () => {
    if (!secret) { setError('请先生成或输入 secret'); return; }
    setLoading('current'); setError(null); setResult(null);
    try {
      const r = await callWasm('wasmTOTPCurrentCode', secret);
      setResult(r);
    } catch (e) { setError(String(e)); } finally { setLoading(null); }
  }, [secret]);

  const backupCodes = useCallback(async () => {
    setLoading('backup'); setError(null); setResult(null);
    try {
      const r = await callWasm('wasmTOTPBackupCodes', 10);
      setResult(r);
    } catch (e) { setError(String(e)); } finally { setLoading(null); }
  }, []);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">TOTP 两步验证</h3>
        <p className="text-sm text-fd-muted-foreground">生成密钥、QR 码、验证码和备份码</p>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <label className="block">
          <span className="mb-1 block text-xs font-medium text-fd-muted-foreground">Issuer</span>
          <input value={issuer} onChange={(e) => setIssuer(e.target.value)} className={inputClass} />
        </label>
        <label className="block">
          <span className="mb-1 block text-xs font-medium text-fd-muted-foreground">Account</span>
          <input value={account} onChange={(e) => setAccount(e.target.value)} className={inputClass} />
        </label>
      </div>

      <div className="flex flex-wrap gap-2">
        <RunButton onClick={generate} loading={loading === 'generate'} label="生成密钥" />
        <SecondaryButton onClick={currentCode} loading={loading === 'current'} icon={KeyRound} label="当前验证码" disabled={!secret} />
        <SecondaryButton onClick={backupCodes} loading={loading === 'backup'} icon={ShieldCheck} label="生成备份码" />
      </div>

      {result?.qrDataUrl && (
        <div className="flex items-center gap-4 rounded-lg border border-fd-border p-3">
          <img src={result.qrDataUrl} alt="QR" className="size-28 rounded-lg" />
          <div className="text-sm">
            <p className="mb-1 font-medium">扫码添加到 Authenticator</p>
            <p className="text-xs text-fd-muted-foreground break-all">Secret: {result.secret}</p>
          </div>
        </div>
      )}

      <div className="border-t border-fd-border pt-3">
        <h4 className="mb-2 text-sm font-semibold">验证码校验</h4>
        <div className="grid grid-cols-2 gap-3">
          <input value={code} onChange={(e) => setCode(e.target.value)} placeholder="6 位验证码" className={inputClass} />
          <input value={secret} onChange={(e) => setSecret(e.target.value)} placeholder="Secret" className={inputClass} />
        </div>
        <div className="mt-2">
          <SecondaryButton onClick={validate} loading={loading === 'validate'} icon={CheckCircle2} label="校验" />
        </div>
      </div>

      <ResultBox result={result} error={error} />
    </div>
  );
}

// ─── Password Demo ──────────────────────────────────────

function PasswordDemo() {
  const [plain, setPlain] = useState('mypassword123');
  const [algorithm, setAlgorithm] = useState('argon2id');
  const [hash, setHash] = useState('');
  const [verifyPlain, setVerifyPlain] = useState('');
  const [result, setResult] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState<string | null>(null);

  const doHash = useCallback(async () => {
    if (!plain) { setError('请输入密码'); return; }
    setLoading('hash'); setError(null); setResult(null);
    try {
      const r = await callWasm('wasmPasswordHash', plain, algorithm);
      setResult(r); setHash(r.hash);
    } catch (e) { setError(String(e)); } finally { setLoading(null); }
  }, [plain, algorithm]);

  const doVerify = useCallback(async () => {
    if (!verifyPlain || !hash) { setError('请输入密码和 hash'); return; }
    setLoading('verify'); setError(null); setResult(null);
    try {
      const r = await callWasm('wasmPasswordVerify', verifyPlain, hash);
      setResult(r);
    } catch (e) { setError(String(e)); } finally { setLoading(null); }
  }, [verifyPlain, hash]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">密码哈希</h3>
        <p className="text-sm text-fd-muted-foreground">argon2id / bcrypt 哈希生成与验证</p>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <label className="block">
          <span className="mb-1 block text-xs font-medium text-fd-muted-foreground">密码</span>
          <input value={plain} onChange={(e) => setPlain(e.target.value)} className={`${inputClass} font-mono`} />
        </label>
        <label className="block">
          <span className="mb-1 block text-xs font-medium text-fd-muted-foreground">算法</span>
          <select value={algorithm} onChange={(e) => setAlgorithm(e.target.value)} className={inputClass}>
            <option value="argon2id">argon2id</option>
            <option value="bcrypt">bcrypt</option>
          </select>
        </label>
      </div>

      <RunButton onClick={doHash} loading={loading === 'hash'} label="生成哈希" />

      <div className="border-t border-fd-border pt-3">
        <h4 className="mb-2 text-sm font-semibold">密码验证</h4>
        <div className="space-y-2">
          <input value={verifyPlain} onChange={(e) => setVerifyPlain(e.target.value)} placeholder="输入密码验证" className={`${inputClass} font-mono`} />
          <textarea value={hash} onChange={(e) => setHash(e.target.value)} placeholder="Hash 值" rows={2} className={`${inputClass} font-mono text-xs`} />
        </div>
        <div className="mt-2">
          <SecondaryButton onClick={doVerify} loading={loading === 'verify'} icon={CheckCircle2} label="验证" />
        </div>
      </div>

      <ResultBox result={result} error={error} />
    </div>
  );
}

// ─── Compress Demo ──────────────────────────────────────

function CompressDemo() {
  const [input, setInput] = useState('Hello, ling-base! This is a test string for compression. '.repeat(10));
  const [algo, setAlgo] = useState('zstd');
  const [mode, setMode] = useState<'compress' | 'decompress'>('compress');
  const [result, setResult] = useState<any>(null);
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
      const encoder = new TextEncoder();
      const data = encoder.encode(input);
      const jsArray = Array.from(data);
      const r = await callWasm(fn, jsArray);
      setResult(r);
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [input, algo, mode]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">数据压缩</h3>
        <p className="text-sm text-fd-muted-foreground">zstd / snappy / lz4 / gzip — 浏览器中运行</p>
      </div>

      <div className="flex gap-3">
        <label className="flex-1">
          <span className="mb-1 block text-xs font-medium text-fd-muted-foreground">算法</span>
          <select value={algo} onChange={(e) => setAlgo(e.target.value)} className={inputClass}>
            <option value="zstd">Zstd</option>
            <option value="snappy">Snappy</option>
            <option value="lz4">LZ4</option>
            <option value="gzip">Gzip</option>
          </select>
        </label>
        <label className="flex-1">
          <span className="mb-1 block text-xs font-medium text-fd-muted-foreground">模式</span>
          <select value={mode} onChange={(e) => setMode(e.target.value as 'compress' | 'decompress')} className={inputClass}>
            <option value="compress">压缩</option>
            <option value="decompress">解压</option>
          </select>
        </label>
      </div>

      <label className="block">
        <span className="mb-1 block text-xs font-medium text-fd-muted-foreground">输入数据</span>
        <textarea value={input} onChange={(e) => setInput(e.target.value)} rows={4} className={`${inputClass} font-mono`} />
      </label>

      <RunButton onClick={run} loading={loading} />

      <ResultBox result={result} error={error} />
    </div>
  );
}
