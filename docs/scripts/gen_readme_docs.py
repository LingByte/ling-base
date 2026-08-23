#!/usr/bin/env python3
"""Convert package README.md files to docs/content/docs/*.mdx with safe frontmatter."""

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
DOCS = ROOT / "docs" / "content" / "docs"

# (source_readme, output_mdx, title, description)
PACKAGES = [
    ("bootstrap/README.md", "bootstrap/index.mdx", "应用启动框架", "Spring Boot 风格启动器，生命周期、Profile、优雅关闭"),
    ("version/README.md", "version/index.mdx", "版本信息", "构建时与运行时版本、ldflags 注入"),
    ("apidocs/README.md", "apidocs/index.mdx", "API 文档", "Huma OpenAPI 3.1，Scalar/Swagger/Redoc 一行挂载"),
    ("pentest/README.md", "pentest/index.mdx", "渗透测试工具", "27 个 Web 安全测试工具，仅标准库"),
    ("lingcli/README.md", "lingcli/index.mdx", "lingcli 脚手架", "一键生成集成 ling-base 的 Go 项目"),
    ("common/README.md", "common/root.mdx", "根工具包 (common)", "BaseModel、GORM 初始化、环境变量与文件工具"),
]


def sanitize_mdx_body(body: str) -> str:
    lines = body.splitlines()
    out: list[str] = []
    in_fence = False
    for line in lines:
        if line.strip().startswith("```"):
            in_fence = not in_fence
            out.append(line)
            continue
        if in_fence:
            out.append(line)
            continue
        line = re.sub(r"<(\d)", r"&lt;\1", line)
        line = line.replace("<=", "≤").replace(">=", "≥")
        line = re.sub(r"`([^`]*?)<([a-zA-Z_][a-zA-Z0-9_]*)>([^`]*?)`", r"`\1{\<\2\>}\3`", line)
        out.append(line)
    return "\n".join(out)


def fix_paths(body: str) -> str:
    replacements = [
        ("github.com/LingByte/ling-base/cache", "github.com/LingByte/ling-base/common/cache"),
        ("github.com/LingByte/ling-base/limiter", "github.com/LingByte/ling-base/common/limiter"),
    ]
    for old, new in replacements:
        body = body.replace(old, new)
    # common subpackages without /common/
    common_pkgs = {p.name for p in (ROOT / "common").iterdir() if p.is_dir()}
    for pkg in sorted(common_pkgs, key=len, reverse=True):
        body = body.replace(
            f"github.com/LingByte/ling-base/{pkg}",
            f"github.com/LingByte/ling-base/common/{pkg}",
        )
    return body


def to_mdx(title: str, description: str, body: str, note: str = "") -> str:
    desc = description.replace(":", " -")[:120]
    lines = ["---", f"title: {title}", f"description: {desc}", "---", ""]
    if note:
        lines.extend([f"> {note}", ""])
    lines.append(body.rstrip())
    lines.append("")
    return "\n".join(lines)


def main() -> None:
    for src_rel, out_rel, title, desc in PACKAGES:
        src = ROOT / src_rel
        out = DOCS / out_rel
        if not src.exists():
            print(f"  skip (no README): {src_rel}")
            continue
        body = fix_paths(sanitize_mdx_body(src.read_text(encoding="utf-8")))
        note = ""
        if out_rel == "common/root.mdx":
            note = "以下为 `common` 根包 README；子包文档见 [通用工具索引](/docs/common)。"
        content = to_mdx(title, desc, body, note)
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(content, encoding="utf-8")
        print(f"  wrote {out.relative_to(ROOT)}")
    print("Done")


if __name__ == "__main__":
    main()
