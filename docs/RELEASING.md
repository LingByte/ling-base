# 文档站发布

文档站**不单独发 npm 包**。发布路径是静态站点部署到 GitHub Pages。

## 自动发布

1. 合并改动到 `main`（且触及 `docs/**`）
2. `.github/workflows/docs-deploy.yml` 执行 `pnpm build:static`
3. 产物上传并部署到 `https://lingbyte.github.io/ling-base/`

环境变量：

- `NEXT_PUBLIC_BASE_PATH=/ling-base`
- `NEXT_PUBLIC_STATIC_DEPLOY=1`

## 手动触发

在 GitHub Actions 中对 `Deploy Docs` workflow 使用 **Run workflow**。

## Go 模块发版

库模块的 tag / GitHub Release 仍由仓库根目录的 `.github/workflows/release.yml` 处理，与文档站无关。详见根目录 `CONTRIBUTING.md`。
