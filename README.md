# learn-claude-agent-sdk-python

> Build a Go port of [`claude-agent-sdk-python`](https://github.com/anthropics/claude-agent-sdk-python) in 8 small, runnable chapters.

Each chapter is an independent Go module (`agents/sNN-*`). You can start from scratch on s01, or jump into s06 cold — every chapter copies (does not import) the types it depends on, so each one stands alone.

The upstream is a thin **subprocess-CLI shepherd**: the Python `claude_agent_sdk` package spawns the `claude` binary, sends prompts as JSON, and reads typed messages from stdout. We replicate that architecture in Go, layer by layer.

## Curriculum

| # | Chapter | What you'll build |
|---|---|---|
| s01 | `s01-types` | `Message`, `ContentBlock`, `Options` — the data model |
| s02 | `s02-transport` | Spawn a fake CLI; write & read newline-JSON over pipes |
| s03 | `s03-parser` | Stream parser: stdout JSON lines → typed `Message` values |
| s04 | `s04-query` | One-shot `Query(...)`: combine transport + parser, return `<-chan Message` |
| s05 | `s05-client` | Streaming client: keep subprocess alive, multi-turn, interruptible |
| s06 | `s06-tools-permissions` | `allowed_tools` / `disallowed_tools` / `can_use_tool` callback policy chain |
| s07 | `s07-hooks` | PreToolUse / PostToolUse hook dispatcher |
| s08 | `s08-mcp` | Register Go funcs as MCP tools, auto-derive JSON Schema |
| —   | `s_full`  | All eight layers wired into one runnable agent |

Plus two appendices in `docs/`:
- **appendix-a** — Subprocess vs HTTP transport for agent SDKs.
- **appendix-b** — Upstream source map (every concept ↔ Python file).

## Quick Start

```bash
git clone https://github.com/Ding-Ye/learn-claude-agent-sdk-python
cd learn-claude-agent-sdk-python
go work sync
go test ./agents/s01-types/...
```

Each chapter's `README.md` has its own "try it" recipe.

## Docs

- 中文：[`docs/zh/`](./docs/zh/)
- English: [`docs/en/`](./docs/en/)

Every chapter has a paired `docs/zh/sNN.md` + `docs/en/sNN.md` written in the six-section format: **Problem · Solution · How It Works · What Changed · Try It · Upstream Source Reading**.

## Upstream Source Reading

The `upstream-readings/` dir contains annotated excerpts of the Python source (pinned to commit [`c352a50`](https://github.com/anthropics/claude-agent-sdk-python/tree/c352a509929a712de65637cbafafcc3a1e3ba4f6)) referenced from the docs. Compare line-by-line with the Go you wrote.

## Acknowledgements

This repo is a **learning companion**, not a port for production use. All credit for the design goes to the [Anthropic SDK team](https://github.com/anthropics/claude-agent-sdk-python). Excerpts of the Python source are reproduced under the upstream MIT license — see [LICENSE](./LICENSE).

---

# 中文简介

用 8 章小模块把 `claude-agent-sdk-python` 的核心架构在 Go 里走一遍。每一章是独立 Go module，可以从 s01 起步，也可以直接跳到任意一章——每章自带它依赖的类型副本，不跨章 import。

上游本质上是 **subprocess CLI 调度器**：Python SDK 起一个 `claude` 子进程，用 JSON 行协议读写。每一章对应上游一个层：类型 → IO → 解析 → 编排 → 扩展。
