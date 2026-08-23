export const appName = 'ling-base';
export const docsRoute = '/docs';
export const docsImageRoute = '/og/docs';
export const docsContentRoute = '/llms.mdx/docs';

export const gitConfig = {
  user: 'LingByte',
  repo: 'ling-base',
  branch: 'main',
};

// GitHub Pages 部署在子路径 /ling-base 下，next/image (unoptimized) 不会自动加 basePath，
// 需要手动拼接。本地开发时 NEXT_PUBLIC_BASE_PATH 为空，assetPath 返回原路径。
export const basePath = process.env.NEXT_PUBLIC_BASE_PATH || '';

/** 静态导出部署（GitHub Pages）：无 API 路由与 WebSocket 代理 */
export const isStaticDeploy =
  process.env.NEXT_PUBLIC_STATIC_DEPLOY === '1' ||
  process.env.NEXT_PUBLIC_STATIC_DEPLOY === 'true';

/** 拼接静态资源路径（用于 <img src> / <script src> / fetch 等） */
export function asset(path: string): string {
  return `${basePath}${path}`;
}
