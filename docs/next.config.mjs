import { createMDX } from 'fumadocs-mdx/next';

const withMDX = createMDX();

// GitHub Pages 需要静态导出；本地 pnpm dev 必须关闭 export，否则 API 路由（Chat 代理）会 404。
const staticExport = process.env.STATIC_EXPORT === '1' || process.env.STATIC_EXPORT === 'true';

/** @type {import('next').NextConfig} */
const config = {
  reactStrictMode: true,
  ...(staticExport ? { output: 'export' } : {}),
  images: {
    unoptimized: true,
  },
  basePath: process.env.NEXT_PUBLIC_BASE_PATH || '',
};

export default withMDX(config);
