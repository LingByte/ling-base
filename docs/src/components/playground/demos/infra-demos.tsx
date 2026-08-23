'use client';

import { useState, useCallback } from 'react';
import { ResultBox, RunButton, inputClass, FieldLabel } from '../shared';

export function CircuitBreakerDemo() {
  const [threshold, setThreshold] = useState(3);
  const [result, setResult] = useState<unknown>(null);
  const [loading, setLoading] = useState(false);

  const simulate = useCallback(() => {
    setLoading(true);
    let state: 'closed' | 'open' | 'half-open' = 'closed';
    let failures = 0;
    const log: string[] = [];
    const requests = 12;

    for (let i = 1; i <= requests; i++) {
      const fail = i % 4 === 0; // 每 4 次失败一次
      if (state === 'open') {
        if (i === 8) { state = 'half-open'; log.push(`#${i} half-open 试探`); }
        else { log.push(`#${i} ❌ 熔断拒绝`); continue; }
      }
      if (fail) {
        failures++;
        log.push(`#${i} ❌ 失败 (${failures}/${threshold})`);
        if (failures >= threshold) { state = 'open'; log.push('   → 熔断器 OPEN'); }
      } else {
        failures = 0;
        if (state === 'half-open') state = 'closed';
        log.push(`#${i} ✅ 成功`);
      }
    }
    setResult({ threshold, finalState: state, log });
    setLoading(false);
  }, [threshold]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">熔断器模拟</h3>
        <p className="text-sm text-fd-muted-foreground">common/circuitbreaker — 失败阈值触发熔断</p>
      </div>
      <label>
        <FieldLabel>失败阈值</FieldLabel>
        <input type="number" value={threshold} onChange={(e) => setThreshold(Number(e.target.value))} className={inputClass} min={1} max={10} />
      </label>
      <RunButton onClick={simulate} loading={loading} label="模拟请求" />
      <ResultBox result={result} error={null} />
    </div>
  );
}

export function RetryDemo() {
  const [maxAttempts, setMaxAttempts] = useState(3);
  const [result, setResult] = useState<unknown>(null);
  const [loading, setLoading] = useState(false);

  const simulate = useCallback(() => {
    setLoading(true);
    const failUntil = 2; // 第 3 次才成功
    const log: string[] = [];
    let attempt = 0;
    let ok = false;
    while (attempt < maxAttempts && !ok) {
      attempt++;
      if (attempt > failUntil) {
        ok = true;
        log.push(`第 ${attempt} 次: ✅ 成功`);
      } else {
        log.push(`第 ${attempt} 次: ❌ 失败，${attempt < maxAttempts ? '重试...' : '放弃'}`);
      }
    }
    setResult({ maxAttempts, success: ok, attempts: attempt, log });
    setLoading(false);
  }, [maxAttempts]);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">重试策略模拟</h3>
        <p className="text-sm text-fd-muted-foreground">common/retry — 指数退避重试</p>
      </div>
      <label>
        <FieldLabel>最大重试次数</FieldLabel>
        <input type="number" value={maxAttempts} onChange={(e) => setMaxAttempts(Number(e.target.value))} className={inputClass} min={1} max={10} />
      </label>
      <RunButton onClick={simulate} loading={loading} label="模拟重试" />
      <ResultBox result={result} error={null} />
    </div>
  );
}
