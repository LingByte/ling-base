# 贡献文档站

本目录是 ling-base 的文档站点（`apps/v1`），基于 Next.js。Go 库本身的贡献流程见仓库根目录 [`CONTRIBUTING.md`](../CONTRIBUTING.md)。

## 本地开发

```bash
cd docs
pnpm install
pnpm v1:dev
# http://localhost:4000
```

首次或样式缺失时需要生成 registry / styles：

```bash
pnpm --filter=v1 registry:build
```

## 质量检查

```bash
pnpm --filter=v1 lint
pnpm build:static          # 默认：秒级 MDX→HTML（GitHub Pages）
# DOCS_STATIC_ENGINE=next pnpm build:static   # 可选：完整 Next 导出（很慢）
```

## CI

- PR：`.github/workflows/docs-ci.yml`（typecheck / lint / static build）
- 合并到 `main`：`.github/workflows/docs-deploy.yml` 部署到 GitHub Pages

文档内容主要在 `apps/v1/content/docs/`。站点品牌与导航在 `apps/v1/lib/config.ts`。
