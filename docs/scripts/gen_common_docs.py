#!/usr/bin/env python3
"""Regenerate docs/content/docs/common/*.mdx from common/*/README.md with full content."""

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
COMMON = ROOT / "common"
DOCS = ROOT / "docs" / "content" / "docs" / "common"

# Chinese titles for frontmatter
TITLES = {
    "audioutil": "音频工具",
    "barcode": "条形码",
    "bloom": "布隆过滤器",
    "cache": "Cache 缓存",
    "captcha": "验证码",
    "circuitbreaker": "Circuit Breaker 熔断器",
    "compress": "压缩",
    "config": "配置管理",
    "constants": "常量",
    "convert": "类型转换",
    "cron": "Cron 定时任务",
    "crypto": "加密",
    "eventbus": "事件总线",
    "geoip": "GeoIP",
    "hash": "哈希",
    "idgen": "ID 生成",
    "imageutil": "图像工具",
    "i18n": "国际化",
    "jwtutil": "JWT 工具",
    "limiter": "Limiter 限流",
    "lock": "Lock 分布式锁",
    "logger": "日志",
    "mathutil": "数学工具",
    "metrics": "指标",
    "middleware": "HTTP 中间件",
    "migration": "数据库迁移",
    "mq": "消息队列",
    "netutil": "网络工具",
    "nltime": "自然语言时间",
    "notification": "通知",
    "opentelemetry": "OpenTelemetry",
    "parser": "解析器",
    "passkey": "Passkey",
    "password": "密码",
    "phone": "手机号",
    "pinyin": "拼音",
    "pool": "对象池",
    "qrcode": "二维码",
    "queue": "队列",
    "random": "随机数",
    "response": "HTTP 响应",
    "retry": "Retry 重试",
    "scheduler": "调度器",
    "search": "搜索",
    "stats": "统计",
    "system": "系统信息",
    "timeutil": "时间工具",
    "totp": "TOTP",
    "tracing": "链路追踪",
    "validate": "校验",
    "videoutil": "视频工具",
}

# Safe Chinese descriptions for YAML frontmatter (avoid unquoted colons in English text)
DESCRIPTIONS = {pkg: f"ling-base common/{pkg} 模块文档" for pkg in [
    "audioutil", "barcode", "bloom", "cache", "captcha", "circuitbreaker",
    "compress", "config", "constants", "convert", "cron", "crypto",
    "eventbus", "geoip", "hash", "i18n", "idgen", "imageutil", "jwtutil",
    "limiter", "lock", "logger", "mathutil", "metrics", "middleware",
    "migration", "mq", "netutil", "nltime", "notification", "opentelemetry",
    "parser", "passkey", "password", "phone", "pinyin", "pool", "qrcode",
    "queue", "random", "response", "retry", "scheduler", "search", "stats",
    "system", "timeutil", "totp", "tracing", "validate", "videoutil",
]}
DESCRIPTIONS.update({
    "constants": "平台共享常量，环境名、数据库驱动、时区与启动参数",
    "logger": "基于 zap 的结构化日志，轮转、脱敏与 Request ID",
    "cache": "统一缓存接口，LRU、Redis、多级缓存等后端",
    "limiter": "限流与并发控制，令牌桶与分布式后端",
    "validate": "Struct tag 驱动的数据校验与自定义规则",
    "mq": "Broker 无关消息队列，RabbitMQ、Kafka 等后端",
    "videoutil": "FFmpeg 与 FFprobe 视频处理封装",
    "crypto": "AES、RSA、HMAC 等加密工具",
    "i18n": "国际化与多语言消息",
})

CROSS_REF = {
    "cache": ("/docs/infrastructure/cache", "缓存完整文档"),
    "limiter": ("/docs/infrastructure/limiter", "限流完整文档"),
    "lock": ("/docs/infrastructure/lock", "分布式锁完整文档"),
    "retry": ("/docs/infrastructure/retry", "重试完整文档"),
    "circuitbreaker": ("/docs/infrastructure/circuitbreaker", "熔断器完整文档"),
    "jwtutil": ("/docs/security/jwt", "JWT 安全文档"),
    "totp": ("/docs/security/totp", "TOTP 安全文档"),
    "password": ("/docs/security/password", "密码安全文档"),
    "passkey": ("/docs/security/passkey", "Passkey 安全文档"),
    "mq": ("/docs/providers/mq", "消息队列完整文档"),
}

COMMON_PKGS = {p.name for p in COMMON.iterdir() if p.is_dir() and (p / "README.md").exists()}


def fix_import_paths(text: str) -> str:
    """READMEs often omit /common/ in import paths."""
    for pkg in sorted(COMMON_PKGS, key=len, reverse=True):
        text = text.replace(
            f"github.com/LingByte/ling-base/{pkg}",
            f"github.com/LingByte/ling-base/common/{pkg}",
        )
    return text


def yaml_quote(s: str) -> str:
    s = " ".join(s.split())[:120]
    # Strip YAML-special chars; use plain string (no colons at word boundaries)
    s = s.replace(":", " -")
    return s


def package_description(pkg: str, body: str) -> str:
    if pkg in DESCRIPTIONS:
        return DESCRIPTIONS[pkg]
    title = TITLES.get(pkg, pkg)
    return f"{title} - ling-base common/{pkg}"


def sanitize_mdx_body(body: str) -> str:
    """Escape patterns that break MDX/JSX parsing outside code fences."""
    lines = body.splitlines()
    out: list[str] = []
    in_fence = False
    for line in lines:
        stripped = line.strip()
        if stripped.startswith("```"):
            in_fence = not in_fence
            out.append(line)
            continue
        if in_fence:
            out.append(line)
            continue
        # <0.1% style numeric comparisons in tables
        line = re.sub(r"<(\d)", r"&lt;\1", line)
        line = line.replace("<=", "≤").replace(">=", "≥")
        # angle-bracket placeholders outside code (e.g. <date> in table cells)
        line = re.sub(r"`([^`]*?)<([a-zA-Z_][a-zA-Z0-9_]*)>([^`]*?)`", r"`\1{\<\2\>}\3`", line)
        out.append(line)
    return "\n".join(out)


def first_paragraph_desc(body: str) -> str:
    lines = body.splitlines()
    for line in lines:
        line = line.strip()
        if line and not line.startswith("#") and not line.startswith("|") and not line.startswith("```"):
            return line[:120]
    return ""


def extract_title(body: str, pkg: str) -> str:
    m = re.match(r"^#\s+(.+)$", body, re.MULTILINE)
    if m:
        return m.group(1).strip()
    return TITLES.get(pkg, pkg)


def readme_to_mdx(pkg: str, readme_path: Path) -> str:
    body = readme_path.read_text(encoding="utf-8")
    body = fix_import_paths(body)
    body = sanitize_mdx_body(body)

    title = TITLES.get(pkg, extract_title(body, pkg))
    desc = yaml_quote(package_description(pkg, body))

    lines = [
        "---",
        f"title: {title}",
        f"description: {desc}",
        "---",
        "",
    ]

    if pkg in CROSS_REF:
        href, label = CROSS_REF[pkg]
        lines.extend([
            f"> 另见专题文档：[{label}]({href})。以下为 `common/{pkg}` 包 README 全文。",
            "",
        ])

    lines.append(body.rstrip())
    lines.append("")
    return "\n".join(lines)


def main() -> None:
    DOCS.mkdir(parents=True, exist_ok=True)
    count = 0
    for pkg_dir in sorted(COMMON.iterdir()):
        if not pkg_dir.is_dir():
            continue
        readme = pkg_dir / "README.md"
        if not readme.exists():
            continue
        out = DOCS / f"{pkg_dir.name}.mdx"
        content = readme_to_mdx(pkg_dir.name, readme)
        out.write_text(content, encoding="utf-8")
        count += 1
        print(f"  wrote {out.relative_to(ROOT)}")
    print(f"Done: {count} common module pages")


if __name__ == "__main__":
    main()
