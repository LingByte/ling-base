#!/usr/bin/env python3
"""Merge common/*/README into infrastructure/*.mdx with Chinese intro."""

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
DOCS = ROOT / "docs" / "content" / "docs"

MAP = {
    "cache": ("缓存", "通用 cache.Cache 接口与多后端"),
    "limiter": ("限流器", "令牌桶、滑动窗口与分布式限流"),
    "lock": ("分布式锁", "进程内与 Redis/etcd 分布式锁"),
    "retry": ("重试", "指数退避与可配置重试策略"),
    "circuitbreaker": ("熔断器", "Closed/Open/Half-Open 熔断状态机"),
}

INTROS = {
    "cache": """# 缓存

通用 `cache.Cache` 接口，支持 LRU、内存、多级缓存及 Redis/Memcached/BigCache 等后端。

```bash
go get github.com/LingByte/ling-base/common/cache
go get github.com/LingByte/ling-base/common/cache/lru
go get github.com/LingByte/ling-base/common/cache/redis
```

> 以下为 `common/cache` 包 README 全文。

""",
    "limiter": """# 限流器

统一限流与并发控制，支持令牌桶、按 Key 限流及 Redis/MongoDB/etcd 分布式后端。

```bash
go get github.com/LingByte/ling-base/common/limiter
go get github.com/LingByte/ling-base/common/limiter/tokenbucket
```

> 以下为 `common/limiter` 包 README 全文。

""",
    "lock": """# 分布式锁

统一 Lock 接口，支持 mutex、Redis、etcd、PostgreSQL 等后端。

```bash
go get github.com/LingByte/ling-base/common/lock
go get github.com/LingByte/ling-base/common/lock/redis
```

> 以下为 `common/lock` 包 README 全文。

""",
    "retry": """# 重试

可配置重试策略，支持固定/指数/线性退避与 jitter。

```bash
go get github.com/LingByte/ling-base/common/retry
```

> 以下为 `common/retry` 包 README 全文。

""",
    "circuitbreaker": """# 熔断器

线程安全熔断器，Closed → Open → Half-Open 状态转换。

```bash
go get github.com/LingByte/ling-base/common/circuitbreaker
```

> 以下为 `common/circuitbreaker` 包 README 全文。

""",
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
        # skip duplicate h1
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
        out = DOCS / "infrastructure" / f"{pkg}.mdx"
        out.write_text(content, encoding="utf-8")
        print(f"  wrote {out.relative_to(ROOT)}")
    print("Done")


if __name__ == "__main__":
    main()
