#!/usr/bin/env python3
"""Merge common security READMEs into security/*.mdx."""

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
DOCS = ROOT / "docs" / "content" / "docs"

MAP = {
    "jwtutil": ("JWT", "JWT 签发、刷新与吊销"),
    "totp": ("TOTP 两步验证", "TOTP 生成、校验与 QR 码"),
    "password": ("密码哈希", "Argon2id、bcrypt 密码哈希与校验"),
    "passkey": ("Passkey", "WebAuthn 无密码认证"),
}

INTROS = {
    "jwtutil": "# JWT\n\n> 以下为 `common/jwtutil` README 全文。\n\n",
    "totp": "# TOTP 两步验证\n\n> 以下为 `common/totp` README 全文。\n\n",
    "password": "# 密码哈希\n\n> 以下为 `common/password` README 全文。\n\n",
    "passkey": "# Passkey\n\n> 以下为 `common/passkey` README 全文。\n\n",
}

OUT = {
    "jwtutil": "jwt",
    "totp": "totp",
    "password": "password",
    "passkey": "passkey",
}


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
        out.append(line)
    return "\n".join(out)


def fix_imports(text: str, pkg: str) -> str:
    return text.replace(
        f"github.com/LingByte/ling-base/{pkg}",
        f"github.com/LingByte/ling-base/common/{pkg}",
    )


def main() -> None:
    for pkg, (title, desc) in MAP.items():
        readme = ROOT / "common" / pkg / "README.md"
        if not readme.exists():
            continue
        body = fix_imports(readme.read_text(encoding="utf-8"), pkg)
        body = sanitize_mdx_body(body)
        lines = body.splitlines()
        if lines and lines[0].startswith("# "):
            body = "\n".join(lines[1:]).lstrip()
        content = "\n".join([
            "---",
            f"title: {title}",
            f"description: {desc}",
            "---",
            "",
            INTROS[pkg],
            body.rstrip(),
            "",
        ])
        out = DOCS / "security" / f"{OUT[pkg]}.mdx"
        out.write_text(content, encoding="utf-8")
        print(f"  wrote {out.relative_to(ROOT)}")
    print("Done")


if __name__ == "__main__":
    main()
