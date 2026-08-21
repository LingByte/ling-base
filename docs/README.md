# ling-base Documentation

This directory contains architecture docs and design notes for ling-base modules.

## Module Index

### Core Libraries

| Module | Path | Description |
|--------|------|-------------|
| common | [`common/`](../common/) | 50+ standalone utility modules (cache, lock, limiter, crypto, i18n, ...) |
| relay | [`relay/`](../relay/) | Unified LLM relay layer with 40+ provider adaptors |
| voice | [`voice/`](../voice/) | Speech recognition (ASR), synthesis (TTS), and realtime conversation |
| stores | [`stores/`](../stores/) | Object storage abstraction across 9 backends |
| mq | [`mq/`](../mq/) | Message queue interface (Kafka, RabbitMQ, ActiveMQ, RocketMQ, Redis) |
| censor | [`censor/`](../censor/) | Content moderation (Aliyun, Tencent, Qiniu) |
| ocr | [`ocr/`](../ocr/) | OCR optical character recognition (6 providers) |
| middleware | [`middleware/`](../middleware/) | HTTP middleware for Gin (CORS, rate limit, maintenance, ...) |
| bootstrap | [`bootstrap/`](../bootstrap/) | Application bootstrap framework (lifecycle + banner) |
| version | [`version/`](../version/) | Version info |
| lingcli | [`lingcli/`](../lingcli/) | Project scaffolding CLI |
| agent | [`agent/`](../agent/) | Coding agent with TUI, tools, sessions, and relay-powered LLM access |

### Agent Docs

| Doc | Description |
|-----|-------------|
| [compaction.md](compaction.md) | Context compaction design (micro + auto) |
| [memory-architecture.md](memory-architecture.md) | Auto-memory store architecture |
| [parity.md](parity.md) | Feature parity reference |
| [server-side-tools.md](server-side-tools.md) | Server-side tool protocol |

### Architecture & Overview

| Doc | Description |
|-----|-------------|
| [architecture.md](architecture.md) | High-level architecture: relay/voice/common deep dive, module relationships, design principles |
| [../README.md](../README.md) | Full module tree, install guide, and usage examples |
| [../ARCHITECTURE.md](../ARCHITECTURE.md) | Original architecture doc (mirror of architecture.md) |
