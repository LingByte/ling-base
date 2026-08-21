'use client';

import { useState, useCallback } from 'react';
import { useWasm, callWasm } from './wasm-loader';
import {
  ShieldCheck,
  KeyRound,
  FileArchive,
  Loader2,
  Play,
  CheckCircle2,
  XCircle,
  AlertTriangle,
} from 'lucide-react';

type TabId = 'totp' | 'password' | 'compress';

const tabs: { id: TabId; label: string; icon: typeof ShieldCheck }[] = [
  { id: 'totp', label: 'TOTP 两步验证', icon: ShieldCheck },
  { id: 'password', label: '密码哈希', icon: KeyRound },
  { id: 'compress', label: '压缩', icon: FileArchive },
];

export function Playground() {
  const { loaded, error: wasmError } = useWasm();
  const [activeTab, setActiveTab] = useState<TabId>('totp');

  return (
    <div className="rounded-xl border border-fd-border bg-fd-card overflow-hidden">
      {/* Tab bar */}
      <div className="flex border-b border-fd-border">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex items-center gap-2 px-5 py-3 text-sm font-medium transition border-b-2 ${
              activeTab === tab.id
                ? 'border-fd-primary text-fd-primary'
                : 'border-transparent text-fd-muted-foreground hover:text-fd-foreground'
            }`}
          >
            <tab.icon className="size-4" />
            {tab.label}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="p-6">
        {wasmError ? (
          <div className="flex items-center gap-2 rounded-lg bg-red-500/10 p-4 text-sm text-red-500">
            <AlertTriangle className="size-4 shrink-0" />
            WASM 加载失败: {wasmError}
          </div>
        ) : !loaded ? (
          <div className="flex items-center justify-center gap-2 py-12 text-sm text-fd-muted-foreground">
            <Loader2 className="size-5 animate-spin" />
            正在加载 WASM 模块...
          </div>
        ) : (
          <>
            {activeTab === 'totp' && <TOTPDemo />}
            {activeTab === 'password' && <PasswordDemo />}
            {activeTab === 'compress' && <CompressDemo />}
          </>
        )}
      </div>
    </div>
  );
}

// ─── Result display ─────────────────────────────────────

function ResultBox({ result, error }: { result: any; error: string | null }) {
  if (error) {
    return (
      <div className="mt-4 flex items-start gap-2 rounded-lg bg-red-500/10 p-4 text-sm text-red-500">
        <XCircle className="size-4 shrink-0 mt-0.5" />
        <pre className="whitespace-pre-wrap break-all">{error}</pre>
      </div>
    );
  }
  if (!result) return null;
  return (
    <div className="mt-4 rounded-lg bg-fd-secondary/50 p-4">
      <div className="mb-2 flex items-center gap-2 text-sm font-medium text-emerald-500">
        <CheckCircle2 className="size-4" />
        执行成功
      </div>
      <pre className="overflow-x-auto text-sm text-fd-foreground">
        {JSON.stringify(result, null, 2)}
      </pre>
    </div>
  );
}

function RunButton({ onClick, loading }: { onClick: () => void; loading: boolean }) {
  return (
    <button
      onClick={onClick}
      disabled={loading}
      className="inline-flex items-center gap-2 rounded-lg bg-fd-primary px-5 py-2 text-sm font-medium text-fd-primary-foreground transition hover:bg-fd-primary/90 disabled:opacity-50"
    >
      {loading ? <Loader2 className="size-4 animate-spin" /> : <Play className="size-4" />}
      运行
    </button>
  );
}

// ─── TOTP Demo ──────────────────────────────────────────

function TOTPDemo() {
  const [issuer, setIssuer] = useState('ling-base');
  const [account, setAccount] = useState('user@example.com');
  const [code, setCode] = useState('');
  const [secret, setSecret] = useState('');
  const [result, setResult] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState<'generate' | 'validate' | 'current' | 'backup' | null>(null);

  const generate = useCallback(async () => {
    setLoading('generate');
    setError(null);
    setResult(null);
    try {
      const r = await callWasm('wasmTOTPGenerate', issuer, account);
      setResult(r);
      setSecret(r.secret);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(null);
    }
  }, [issuer, account]);

  const validate = useCallback(async () => {
    if (!code || !secret) {
      setError('请先输入 code 和 secret');
      return;
    }
    setLoading('validate');
    setError(null);
    setResult(null);
    try {
      const r = await callWasm('wasmTOTPValidate', code, secret);
      setResult(r);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(null);
    }
  }, [code, secret]);

  const currentCode = useCallback(async () => {
    if (!secret) {
      setError('请先生成或输入 secret');
      return;
    }
    setLoading('current');
    setError(null);
    setResult(null);
    try {
      const r = await callWasm('wasmTOTPCurrentCode', secret);
      setResult(r);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(null);
    }
  }, [secret]);

  const backupCodes = useCallback(async () => {
    setLoading('backup');
    setError(null);
    setResult(null);
    try {
      const r = await callWasm('wasmTOTPBackupCodes', 10);
      setResult(r);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(null);
    }
  }, []);

  return (
    <div className="space-y-4">
      <p className="text-sm text-fd-muted-foreground">
        TOTP（基于时间的一次性密码）两步验证。生成密钥、验证码、QR 码和备份码。
      </p>

      <div className="grid grid-cols-2 gap-3">
        <label className="block">
          <span className="mb-1 block text-xs font-medium text-fd-muted-foreground">Issuer</span>
          <input
            value={issuer}
            onChange={(e) => setIssuer(e.target.value)}
            className="w-full rounded-lg border border-fd-border bg-fd-background px-3 py-2 text-sm"
          />
        </label>
        <label className="block">
          <span className="mb-1 block text-xs font-medium text-fd-muted-foreground">Account</span>
          <input
            value={account}
            onChange={(e) => setAccount(e.target.value)}
            className="w-full rounded-lg border border-fd-border bg-fd-background px-3 py-2 text-sm"
          />
        </label>
      </div>

      <div className="flex flex-wrap gap-2">
        <RunButton onClick={generate} loading={loading === 'generate'} />
        <button
          onClick={currentCode}
          disabled={loading === 'current' || !secret}
          className="inline-flex items-center gap-2 rounded-lg border border-fd-border px-4 py-2 text-sm font-medium transition hover:bg-fd-accent disabled:opacity-50"
        >
          {loading === 'current' ? <Loader2 className="size-4 animate-spin" /> : <KeyRound className="size-4" />}
          当前验证码
        </button>
        <button
          onClick={backupCodes}
          disabled={loading === 'backup'}
          className="inline-flex items-center gap-2 rounded-lg border border-fd-border px-4 py-2 text-sm font-medium transition hover:bg-fd-accent disabled:opacity-50"
        >
          {loading === 'backup' ? <Loader2 className="size-4 animate-spin" /> : <ShieldCheck className="size-4" />}
          生成备份码
        </button>
      </div>

      {result?.qrDataUrl && (
        <div className="flex items-center gap-4 rounded-lg border border-fd-border p-4">
          <img src={result.qrDataUrl} alt="TOTP QR" className="size-32 rounded-lg" />
          <div className="text-sm">
            <p className="mb-1 font-medium">扫码添加到 Authenticator</p>
            <p className="text-xs text-fd-muted-foreground break-all">Secret: {result.secret}</p>
          </div>
        </div>
      )}

      {/* Validate section */}
      <div className="border-t border-fd-border pt-4">
        <h4 className="mb-3 text-sm font-semibold">验证码校验</h4>
        <div className="grid grid-cols-2 gap-3">
          <input
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder="6 位验证码"
            className="rounded-lg border border-fd-border bg-fd-background px-3 py-2 text-sm"
          />
          <input
            value={secret}
            onChange={(e) => setSecret(e.target.value)}
            placeholder="Secret"
            className="rounded-lg border border-fd-border bg-fd-background px-3 py-2 text-sm"
          />
        </div>
        <button
          onClick={validate}
          disabled={loading === 'validate'}
          className="mt-2 inline-flex items-center gap-2 rounded-lg border border-fd-border px-4 py-2 text-sm font-medium transition hover:bg-fd-accent disabled:opacity-50"
        >
          {loading === 'validate' ? <Loader2 className="size-4 animate-spin" /> : <CheckCircle2 className="size-4" />}
          校验
        </button>
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
  const [loading, setLoading] = useState<'hash' | 'verify' | null>(null);

  const doHash = useCallback(async () => {
    if (!plain) {
      setError('请输入密码');
      return;
    }
    setLoading('hash');
    setError(null);
    setResult(null);
    try {
      const r = await callWasm('wasmPasswordHash', plain, algorithm);
      setResult(r);
      setHash(r.hash);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(null);
    }
  }, [plain, algorithm]);

  const doVerify = useCallback(async () => {
    if (!verifyPlain || !hash) {
      setError('请输入密码和 hash');
      return;
    }
    setLoading('verify');
    setError(null);
    setResult(null);
    try {
      const r = await callWasm('wasmPasswordVerify', verifyPlain, hash);
      setResult(r);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(null);
    }
  }, [verifyPlain, hash]);

  return (
    <div className="space-y-4">
      <p className="text-sm text-fd-muted-foreground">
        密码哈希（argon2id / bcrypt）。支持哈希生成和验证。
      </p>

      <div className="grid grid-cols-2 gap-3">
        <label className="block">
          <span className="mb-1 block text-xs font-medium text-fd-muted-foreground">密码</span>
          <input
            value={plain}
            onChange={(e) => setPlain(e.target.value)}
            className="w-full rounded-lg border border-fd-border bg-fd-background px-3 py-2 text-sm font-mono"
          />
        </label>
        <label className="block">
          <span className="mb-1 block text-xs font-medium text-fd-muted-foreground">算法</span>
          <select
            value={algorithm}
            onChange={(e) => setAlgorithm(e.target.value)}
            className="w-full rounded-lg border border-fd-border bg-fd-background px-3 py-2 text-sm"
          >
            <option value="argon2id">argon2id</option>
            <option value="bcrypt">bcrypt</option>
          </select>
        </label>
      </div>

      <RunButton onClick={doHash} loading={loading === 'hash'} />

      {/* Verify section */}
      <div className="border-t border-fd-border pt-4">
        <h4 className="mb-3 text-sm font-semibold">密码验证</h4>
        <div className="grid grid-cols-2 gap-3">
          <input
            value={verifyPlain}
            onChange={(e) => setVerifyPlain(e.target.value)}
            placeholder="输入密码验证"
            className="rounded-lg border border-fd-border bg-fd-background px-3 py-2 text-sm font-mono"
          />
          <input
            value={hash}
            onChange={(e) => setHash(e.target.value)}
            placeholder="Hash 值"
            className="rounded-lg border border-fd-border bg-fd-background px-3 py-2 text-xs font-mono"
          />
        </div>
        <button
          onClick={doVerify}
          disabled={loading === 'verify'}
          className="mt-2 inline-flex items-center gap-2 rounded-lg border border-fd-border px-4 py-2 text-sm font-medium transition hover:bg-fd-accent disabled:opacity-50"
        >
          {loading === 'verify' ? <Loader2 className="size-4 animate-spin" /> : <CheckCircle2 className="size-4" />}
          验证
        </button>
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
    setLoading(true);
    setError(null);
    setResult(null);
    try {
      const fnMap: Record<string, [string, string]> = {
        zstd: ['wasmZstdCompress', 'wasmZstdDecompress'],
        snappy: ['wasmSnappyCompress', 'wasmSnappyDecompress'],
        lz4: ['wasmLZ4Compress', 'wasmLZ4Decompress'],
        gzip: ['wasmGzipCompress', 'wasmGzipDecompress'],
      };
      const [compressFn, decompressFn] = fnMap[algo];
      const fn = mode === 'compress' ? compressFn : decompressFn;

      // 将字符串转为 Uint8Array 传给 WASM
      const encoder = new TextEncoder();
      const data = encoder.encode(input);
      const jsArray = Array.from(data);

      // callWasm 接受 JS 参数，内部处理
      const r = await callWasm(fn, jsArray);
      setResult(r);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [input, algo, mode]);

  return (
    <div className="space-y-4">
      <p className="text-sm text-fd-muted-foreground">
        数据压缩（zstd / snappy / lz4 / gzip）。完全在浏览器中运行，无需后端。
      </p>

      <div className="flex gap-3">
        <label className="flex-1">
          <span className="mb-1 block text-xs font-medium text-fd-muted-foreground">算法</span>
          <select
            value={algo}
            onChange={(e) => setAlgo(e.target.value)}
            className="w-full rounded-lg border border-fd-border bg-fd-background px-3 py-2 text-sm"
          >
            <option value="zstd">Zstd</option>
            <option value="snappy">Snappy</option>
            <option value="lz4">LZ4</option>
            <option value="gzip">Gzip</option>
          </select>
        </label>
        <label className="flex-1">
          <span className="mb-1 block text-xs font-medium text-fd-muted-foreground">模式</span>
          <select
            value={mode}
            onChange={(e) => setMode(e.target.value as 'compress' | 'decompress')}
            className="w-full rounded-lg border border-fd-border bg-fd-background px-3 py-2 text-sm"
          >
            <option value="compress">压缩</option>
            <option value="decompress">解压</option>
          </select>
        </label>
      </div>

      <label className="block">
        <span className="mb-1 block text-xs font-medium text-fd-muted-foreground">输入数据</span>
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          rows={5}
          className="w-full rounded-lg border border-fd-border bg-fd-background px-3 py-2 text-sm font-mono"
        />
      </label>

      <RunButton onClick={run} loading={loading} />

      <ResultBox result={result} error={error} />
    </div>
  );
}
