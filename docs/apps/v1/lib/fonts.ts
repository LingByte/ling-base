import { cn } from "cn"

/**
 * Avoid next/font/google in this app: Turbopack/webpack + CI/static export
 * frequently fail while fetching Google Fonts. Rely on system stacks via CSS vars.
 */
export const fontVariables = cn(
  "[--font-sans:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif]",
  "[--font-heading:ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif]",
  "[--font-mono:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace]"
)
