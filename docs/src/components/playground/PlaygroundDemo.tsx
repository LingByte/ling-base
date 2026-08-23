'use client';

import type { DemoId } from '@/lib/playground/types';
import {
  TOTPDemo,
  CompressDemo,
  HashDemo,
  PasswordDemo,
  ValidateDemo,
  JwtDemo,
  QRCodeDemo,
  BarcodeDemo,
} from './demos/wasm-demos';
import {
  IdGenDemo,
  RandomDemo,
  PinyinDemo,
  PhoneDemo,
  ConvertDemo,
  CryptoDemo,
  NLTimeDemo,
  BloomDemo,
  MathUtilDemo,
  NetUtilDemo,
  I18nDemo,
  TimeUtilDemo,
} from './demos/extra-demos';
import { CaptchaDemo } from './demos/captcha-demo';
import { RealtimeDemo } from './demos/realtime-demos';
import { CircuitBreakerDemo, RetryDemo } from './demos/infra-demos';
import {
  ChatDemo,
  ChatStreamDemo,
  EmbedDemo,
  LimiterDemo,
} from './demos/api-demos';

export function PlaygroundDemo({ demoId }: { demoId: DemoId }) {
  switch (demoId) {
    case 'totp': return <TOTPDemo />;
    case 'compress': return <CompressDemo />;
    case 'hash': return <HashDemo />;
    case 'password': return <PasswordDemo />;
    case 'validate': return <ValidateDemo />;
    case 'jwt': return <JwtDemo />;
    case 'qrcode': return <QRCodeDemo />;
    case 'barcode': return <BarcodeDemo />;
    case 'idgen': return <IdGenDemo />;
    case 'random': return <RandomDemo />;
    case 'pinyin': return <PinyinDemo />;
    case 'phone': return <PhoneDemo />;
    case 'convert': return <ConvertDemo />;
    case 'crypto': return <CryptoDemo />;
    case 'captcha': return <CaptchaDemo />;
    case 'nltime': return <NLTimeDemo />;
    case 'bloom': return <BloomDemo />;
    case 'mathutil': return <MathUtilDemo />;
    case 'netutil': return <NetUtilDemo />;
    case 'i18n': return <I18nDemo />;
    case 'timeutil': return <TimeUtilDemo />;
    case 'circuitbreaker': return <CircuitBreakerDemo />;
    case 'retry': return <RetryDemo />;
    case 'chat': return <ChatDemo />;
    case 'chat-stream': return <ChatStreamDemo />;
    case 'realtime': return <RealtimeDemo />;
    case 'embed': return <EmbedDemo />;
    case 'limiter': return <LimiterDemo />;
  }
}
