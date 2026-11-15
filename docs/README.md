# ling-base docs

文档站基于 [shadcn/ui](https://github.com/shadcn-ui/ui) 的站点壳改造，内容为 **ling-base**。

## 开发

```bash
cd docs
pnpm install
pnpm build:packages   # 首次或 packages 变更后
# 若 styles 缺失：
pnpm --filter=v1 registry:build --style all
pnpm v1:dev
```

打开 http://localhost:4000

- 应用：`apps/v1`
- 文档 MDX：`apps/v1/content/docs`
- 站点配置：`apps/v1/lib/config.ts`

## 仓库

https://github.com/LingByte/ling-base
