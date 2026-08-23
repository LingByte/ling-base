'use client';

import { useState, useCallback, useRef } from 'react';
import { callWasm } from '../wasm-loader';
import { ResultBox, RunButton, inputClass, FieldLabel } from '../shared';

type CaptchaType = 'image' | 'click' | 'slider' | 'math' | 'jigsaw' | 'rotate';

interface CaptchaResult {
  id: string;
  type: CaptchaType;
  data: Record<string, unknown>;
}

const TYPES: { id: CaptchaType; label: string; desc: string }[] = [
  { id: 'image', label: 'Image', desc: '扭曲文字图片' },
  { id: 'click', label: 'Click', desc: '按顺序点击文字' },
  { id: 'slider', label: 'Slider', desc: '滑块拖到最右' },
  { id: 'math', label: 'Math', desc: '算术题' },
  { id: 'jigsaw', label: 'Jigsaw', desc: '拼图滑块' },
  { id: 'rotate', label: 'Rotate', desc: '旋转图片回正' },
];

function imgSrc(v: unknown): string {
  if (typeof v !== 'string') return '';
  return v.startsWith('data:') ? v : `data:image/png;base64,${v}`;
}

export function CaptchaDemo() {
  const [captchaType, setCaptchaType] = useState<CaptchaType>('image');
  const [challenge, setChallenge] = useState<CaptchaResult | null>(null);
  const [textValue, setTextValue] = useState('');
  const [numValue, setNumValue] = useState(0);
  const [sliderValue, setSliderValue] = useState(0);
  const [rotateValue, setRotateValue] = useState(0);
  const [clickOrder, setClickOrder] = useState<{ x: number; y: number }[]>([]);
  const [result, setResult] = useState<unknown>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const clickRef = useRef<HTMLDivElement>(null);

  const generate = useCallback(async () => {
    setLoading(true); setError(null); setResult(null);
    setTextValue(''); setNumValue(0); setSliderValue(0); setRotateValue(0); setClickOrder([]);
    try {
      const r = await callWasm('wasmCaptchaGenerate', captchaType) as CaptchaResult;
      setChallenge(r);
      if (captchaType === 'slider' && r.data.trackWidth) {
        setSliderValue(0);
      }
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [captchaType]);

  const verify = useCallback(async () => {
    if (!challenge) { setError('请先生成验证码'); return; }
    setLoading(true); setError(null); setResult(null);
    let value: unknown;
    switch (challenge.type) {
      case 'image': value = textValue; break;
      case 'math': value = Number(numValue); break;
      case 'slider': value = sliderValue; break;
      case 'jigsaw': value = sliderValue; break;
      case 'rotate': value = rotateValue; break;
      case 'click': value = clickOrder; break;
      default: value = textValue;
    }
    try {
      setResult(await callWasm('wasmCaptchaVerify', JSON.stringify({
        id: challenge.id,
        type: challenge.type,
        value,
      })));
    } catch (e) { setError(String(e)); } finally { setLoading(false); }
  }, [challenge, textValue, numValue, sliderValue, rotateValue, clickOrder]);

  const handleClickChar = (x: number, y: number) => {
    setClickOrder((prev) => [...prev, { x, y }]);
  };

  const d = challenge?.data ?? {};

  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-1 font-semibold">验证码（6 种类型）</h3>
        <p className="text-sm text-fd-muted-foreground">common/captcha — Image / Click / Slider / Math / Jigsaw / Rotate</p>
      </div>

      <div className="grid grid-cols-3 gap-2 sm:grid-cols-6">
        {TYPES.map((t) => (
          <button
            key={t.id}
            type="button"
            onClick={() => setCaptchaType(t.id)}
            className={`rounded-lg border px-2 py-2 text-left text-xs transition ${
              captchaType === t.id ? 'border-fd-primary bg-fd-primary/10' : 'border-fd-border hover:bg-fd-accent'
            }`}
          >
            <div className="font-medium">{t.label}</div>
            <div className="text-fd-muted-foreground">{t.desc}</div>
          </button>
        ))}
      </div>

      <RunButton onClick={generate} loading={loading} label={`生成 ${captchaType} 验证码`} />

      {challenge && (
        <div className="space-y-3 rounded-lg border border-fd-border p-4">
          {challenge.type === 'image' && typeof d.image === 'string' && (
            <div className="flex justify-center bg-white p-2 rounded">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={imgSrc(d.image)} alt="captcha" height={60} />
            </div>
          )}

          {challenge.type === 'math' && (
            <p className="text-center text-lg font-mono font-semibold">{String(d.question)}</p>
          )}

          {challenge.type === 'slider' && (
            <div>
              <FieldLabel>拖动滑块到最右侧（trackWidth: {String(d.trackWidth)}）</FieldLabel>
              <input type="range" min={0} max={Number(d.trackWidth) || 300} value={sliderValue}
                onChange={(e) => setSliderValue(Number(e.target.value))} className="w-full" />
              <p className="text-xs text-fd-muted-foreground">当前值: {sliderValue}</p>
            </div>
          )}

          {challenge.type === 'jigsaw' && (
            <div className="space-y-2">
              <div className="relative bg-white rounded overflow-hidden" style={{ width: Number(d.width) || 320 }}>
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src={imgSrc(d.background)} alt="bg" className="w-full" />
              </div>
              <FieldLabel>拖动拼图块 X 坐标（0 ~ {Number(d.width) - Number(d.pieceWidth)}）</FieldLabel>
              <input type="range" min={0} max={Math.max(0, Number(d.width) - Number(d.pieceWidth) || 200)}
                value={sliderValue} onChange={(e) => setSliderValue(Number(e.target.value))} className="w-full" />
            </div>
          )}

          {challenge.type === 'rotate' && typeof d.image === 'string' && (
            <div className="space-y-2">
              <div className="flex justify-center">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src={imgSrc(d.image)} alt="rotate" width={Number(d.size) || 200} height={Number(d.size) || 200}
                  style={{ transform: `rotate(${rotateValue}deg)` }} className="rounded-full transition-transform" />
              </div>
              <FieldLabel>旋转角度（°）使图片回正</FieldLabel>
              <input type="range" min={0} max={359} value={rotateValue}
                onChange={(e) => setRotateValue(Number(e.target.value))} className="w-full" />
            </div>
          )}

          {challenge.type === 'click' && (
            <div>
              <p className="mb-2 text-sm">按顺序点击: <strong>{(d.targets as string[])?.join(' → ')}</strong></p>
              <div
                ref={clickRef}
                className="relative mx-auto overflow-hidden rounded border bg-white"
                style={{ width: Number(d.width) || 300, height: Number(d.height) || 200,
                  backgroundImage: d.background ? `url(${imgSrc(d.background)})` : undefined,
                  backgroundSize: 'cover' }}
              >
                {(d.chars as { char: string; x: number; y: number }[])?.map((c, i) => (
                  <button
                    key={`${c.char}-${i}`}
                    type="button"
                    onClick={() => handleClickChar(c.x, c.y)}
                    className="absolute -translate-x-1/2 -translate-y-1/2 rounded bg-white/80 px-2 py-1 text-lg font-bold shadow hover:bg-fd-primary/20"
                    style={{ left: c.x, top: c.y }}
                  >
                    {c.char}
                  </button>
                ))}
              </div>
              <p className="mt-1 text-xs text-fd-muted-foreground">已点击 {clickOrder.length} 次</p>
            </div>
          )}

          {(challenge.type === 'image') && (
            <input value={textValue} onChange={(e) => setTextValue(e.target.value)} placeholder="输入验证码" className={inputClass} />
          )}
          {challenge.type === 'math' && (
            <input type="number" value={numValue} onChange={(e) => setNumValue(Number(e.target.value))} placeholder="答案" className={inputClass} />
          )}

          <RunButton onClick={verify} loading={loading} label="校验" variant="secondary" />
        </div>
      )}

      <ResultBox result={result} error={error} />
    </div>
  );
}
