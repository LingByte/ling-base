'use client';

import { useState, useCallback } from 'react';
import { callWasm } from '../wasm-loader';
import { ResultBox, RunButton, inputClass, FieldLabel } from '../shared';

export function IdGenDemo() {
  const [kind, setKind] = useState('uuidv4');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try { setResult(await callWasm('wasmIDGen', kind)); }
    catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [kind]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">ID 生成</h3>
        <p className="text-sm text-fd-muted-foreground">common/idgen</p>
      </div>
      <select value={kind} onChange={(e) => setKind(e.target.value)} className={inputClass}>
        <option value="uuidv4">UUIDv4</option>
        <option value="uuidv7">UUIDv7</option>
        <option value="ordered">Ordered UUID</option>
        <option value="snowflake">Snowflake</option>
        <option value="short">ShortID</option>
      </select>
      <RunButton onClick={run} loading={loading} label="生成 ID" />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function RandomDemo() {
  const [kind, setKind] = useState('password');
  const [n, setN] = useState(16);
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try { setResult(await callWasm('wasmRandom', kind, n)); }
    catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [kind, n]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">随机数 / 字符串</h3>
        <p className="text-sm text-fd-muted-foreground">common/random</p>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <select value={kind} onChange={(e) => setKind(e.target.value)} className={inputClass}>
          <option value="string">随机字符串</option>
          <option value="numeric">数字串</option>
          <option value="hex">Hex</option>
          <option value="password">密码</option>
          <option value="uuid">UUID</option>
          <option value="color">颜色</option>
          <option value="int">整数范围</option>
        </select>
        <input type="number" value={n} onChange={(e) => setN(Number(e.target.value))} className={inputClass} min={1} max={128} />
      </div>
      <RunButton onClick={run} loading={loading} label="生成" />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function PinyinDemo() {
  const [text, setText] = useState('零字节科技');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try { setResult(await callWasm('wasmPinyin', text, ' ')); }
    catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [text]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">拼音转换</h3>
        <p className="text-sm text-fd-muted-foreground">common/pinyin</p>
      </div>
      <input value={text} onChange={(e) => setText(e.target.value)} className={inputClass} />
      <RunButton onClick={run} loading={loading} label="转拼音" />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function PhoneDemo() {
  const [num, setNum] = useState('13800138000');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try { setResult(await callWasm('wasmPhoneLookup', num)); }
    catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [num]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">手机号归属地</h3>
        <p className="text-sm text-fd-muted-foreground">common/phone</p>
      </div>
      <input value={num} onChange={(e) => setNum(e.target.value)} className={`${inputClass} font-mono`} />
      <RunButton onClick={run} loading={loading} label="查询" />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function ConvertDemo() {
  const [from, setFrom] = useState('json');
  const [to, setTo] = useState('yaml');
  const [data, setData] = useState('{"name":"ling-base","modules":51}');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try { setResult(await callWasm('wasmConvert', from, to, data)); }
    catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [from, to, data]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">格式转换</h3>
        <p className="text-sm text-fd-muted-foreground">common/convert — JSON / YAML / TOML</p>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <select value={from} onChange={(e) => setFrom(e.target.value)} className={inputClass}>
          <option value="json">JSON</option>
          <option value="yaml">YAML</option>
          <option value="toml">TOML</option>
        </select>
        <select value={to} onChange={(e) => setTo(e.target.value)} className={inputClass}>
          <option value="json">JSON</option>
          <option value="yaml">YAML</option>
          <option value="toml">TOML</option>
        </select>
      </div>
      <textarea value={data} onChange={(e) => setData(e.target.value)} rows={5} className={`${inputClass} font-mono text-xs`} />
      <RunButton onClick={run} loading={loading} label="转换" />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function CryptoDemo() {
  const [mode, setMode] = useState<'encrypt' | 'decrypt'>('encrypt');
  const [text, setText] = useState('hello ling-base');
  const [key, setKey] = useState('0123456789abcdef0123456789abcdef');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try {
      const r = await callWasm('wasmCryptoAES', mode, text, key) as { ciphertext?: string; plaintext?: string };
      if (mode === 'encrypt' && r.ciphertext) setText(r.ciphertext);
      if (mode === 'decrypt' && r.plaintext) setText(r.plaintext);
      setResult(r);
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [mode, text, key]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">AES-GCM 加解密</h3>
        <p className="text-sm text-fd-muted-foreground">common/crypto</p>
      </div>
      <select value={mode} onChange={(e) => setMode(e.target.value as 'encrypt' | 'decrypt')} className={inputClass}>
        <option value="encrypt">加密</option>
        <option value="decrypt">解密</option>
      </select>
      <label className="block">
        <FieldLabel>Key（16/24/32 字节）</FieldLabel>
        <input value={key} onChange={(e) => setKey(e.target.value)} className={`${inputClass} font-mono text-xs`} />
      </label>
      <textarea value={text} onChange={(e) => setText(e.target.value)} rows={3} className={`${inputClass} font-mono text-xs`} />
      <RunButton onClick={run} loading={loading} label={mode === 'encrypt' ? '加密' : '解密'} />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function NLTimeDemo() {
  const [expr, setExpr] = useState('tomorrow');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const presets = ['now', 'today', 'tomorrow', 'yesterday', '3 days ago', 'in 2 hours', 'next Monday'];

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try { setResult(await callWasm('wasmNLTime', expr)); }
    catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [expr]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">自然语言时间</h3>
        <p className="text-sm text-fd-muted-foreground">common/nltime</p>
      </div>
      <div className="flex flex-wrap gap-2">
        {presets.map((p) => (
          <button key={p} type="button" onClick={() => setExpr(p)} className="rounded-md border border-fd-border px-2 py-1 text-xs hover:bg-fd-accent">{p}</button>
        ))}
      </div>
      <input value={expr} onChange={(e) => setExpr(e.target.value)} className={inputClass} />
      <RunButton onClick={run} loading={loading} label="解析" />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function MathUtilDemo() {
  const [nums, setNums] = useState('1,2,3,4,5,6,7,8,9,10');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try {
      const arr = nums.split(/[,，\s]+/).map(Number).filter((n) => !Number.isNaN(n));
      setResult(await callWasm('wasmMathStats', JSON.stringify(arr)));
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [nums]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">统计计算</h3>
        <p className="text-sm text-fd-muted-foreground">common/mathutil — mean / median / stdDev / p95</p>
      </div>
      <input value={nums} onChange={(e) => setNums(e.target.value)} className={inputClass} />
      <RunButton onClick={run} loading={loading} label="计算统计量" />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function NetUtilDemo() {
  const [ip, setIp] = useState('192.168.1.1');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try { setResult(await callWasm('wasmNetIP', ip)); }
    catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [ip]);

  const presets = ['127.0.0.1', '192.168.1.1', '10.0.0.1', '8.8.8.8'];

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">IP 地址判断</h3>
        <p className="text-sm text-fd-muted-foreground">common/netutil — private / loopback / public</p>
      </div>
      <div className="flex flex-wrap gap-2">
        {presets.map((p) => (
          <button key={p} type="button" onClick={() => setIp(p)} className="rounded-md border border-fd-border px-2 py-1 text-xs hover:bg-fd-accent">{p}</button>
        ))}
      </div>
      <input value={ip} onChange={(e) => setIp(e.target.value)} className={`${inputClass} font-mono`} />
      <RunButton onClick={run} loading={loading} label="检测" />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function I18nDemo() {
  const [locale, setLocale] = useState('zh');
  const [key, setKey] = useState('welcome');
  const [name, setName] = useState('ling-base');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try {
      setResult(key === 'hello'
        ? await callWasm('wasmI18n', locale, key, name)
        : await callWasm('wasmI18n', locale, key));
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [locale, key, name]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">国际化翻译</h3>
        <p className="text-sm text-fd-muted-foreground">common/i18n — 多语言 key 查找</p>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <select value={locale} onChange={(e) => setLocale(e.target.value)} className={inputClass}>
          <option value="zh">中文</option>
          <option value="en">English</option>
        </select>
        <select value={key} onChange={(e) => setKey(e.target.value)} className={inputClass}>
          <option value="welcome">welcome</option>
          <option value="hello">hello (带参数)</option>
        </select>
      </div>
      {key === 'hello' && <input value={name} onChange={(e) => setName(e.target.value)} className={inputClass} />}
      <RunButton onClick={run} loading={loading} label="翻译" />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function TimeUtilDemo() {
  const [action, setAction] = useState('format');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try { setResult(await callWasm('wasmTimeUtil', action, 'Asia/Shanghai')); }
    catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [action]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">时间工具</h3>
        <p className="text-sm text-fd-muted-foreground">common/timeutil — 格式化 / 日界</p>
      </div>
      <select value={action} onChange={(e) => setAction(e.target.value)} className={inputClass}>
        <option value="format">时区格式化</option>
        <option value="startOfDay">当天起止</option>
      </select>
      <RunButton onClick={run} loading={loading} label="执行" />
      <ResultBox result={result} error={error} />
    </div>
  );
}

export function BloomDemo() {
  const [add, setAdd] = useState('alice,bob,carol');
  const [test, setTest] = useState('alice,dave');
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const run = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try {
      const payload = JSON.stringify({
        add: add.split(/[,，\s]+/).filter(Boolean),
        test: test.split(/[,，\s]+/).filter(Boolean),
      });
      setResult(await callWasm('wasmBloomDemo', payload));
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [add, test]);

  const estimate = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    try { setResult(await callWasm('wasmBloomEstimate', 10000, 0.01)); }
    catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, []);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">布隆过滤器</h3>
        <p className="text-sm text-fd-muted-foreground">common/bloom</p>
      </div>
      <label className="block">
        <FieldLabel>Add（逗号分隔）</FieldLabel>
        <input value={add} onChange={(e) => setAdd(e.target.value)} className={inputClass} />
      </label>
      <label className="block">
        <FieldLabel>Test（逗号分隔）</FieldLabel>
        <input value={test} onChange={(e) => setTest(e.target.value)} className={inputClass} />
      </label>
      <div className="flex gap-2">
        <RunButton onClick={run} loading={loading} label="Add + Test" />
        <RunButton onClick={estimate} loading={loading} label="估算参数" variant="secondary" />
      </div>
      <ResultBox result={result} error={error} />
    </div>
  );
}
