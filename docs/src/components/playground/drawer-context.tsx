'use client';

import { createContext, useContext, useState, useCallback, type ReactNode } from 'react';

type DemoId = 'totp' | 'compress';

interface PlaygroundContextValue {
  open: boolean;
  demoId: DemoId | null;
  openDemo: (id: DemoId) => void;
  close: () => void;
}

const PlaygroundContext = createContext<PlaygroundContextValue | null>(null);

export function usePlayground() {
  const ctx = useContext(PlaygroundContext);
  if (!ctx) throw new Error('usePlayground must be used within PlaygroundProvider');
  return ctx;
}

export function PlaygroundProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const [demoId, setDemoId] = useState<DemoId | null>(null);

  const openDemo = useCallback((id: DemoId) => {
    setDemoId(id);
    setOpen(true);
  }, []);

  const close = useCallback(() => {
    setOpen(false);
  }, []);

  return (
    <PlaygroundContext.Provider value={{ open, demoId, openDemo, close }}>
      {children}
    </PlaygroundContext.Provider>
  );
}
