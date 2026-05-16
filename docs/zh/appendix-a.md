# 附录 A — Subprocess vs HTTP transport

不带代码的阅读。在自己设计 agent SDK 之前，先理解 `claude-agent-sdk-python` 为什么长成这样。

## 两种模型

**Subprocess 模型**（这个 SDK）。
SDK 起一个 CLI 二进制。真活由 CLI 干：模型调度、工具执行、权限 UI、session 状态。SDK 是胶水：起进程、IO、parse、把结果展给用户。

**HTTP 模型**（比如 Anthropic Python SDK、TS Claude Agent SDK）。
SDK 直接走 HTTPS 调模型 API。它自己管 message、tool-use block、streaming event，可能还有本地工具表。

## 各自的取舍

| 关注点 | Subprocess SDK | HTTP SDK |
|---|---|---|
| LLM 调度在哪 | `claude` CLI 里 | SDK 进程里 |
| 工具执行在哪 | `claude` CLI 里 | SDK 进程里（或进程外 MCP） |
| 权限 UX 在哪 | `claude` CLI 里 | SDK 自己造 |
| Session 存储 | CLI 管 `~/.claude/sessions` | SDK 得自己设计 |
| 版本 | SDK + CLI 都得对 | 只 SDK 版本要紧 |
| 分发 | 还得带二进制；跨平台麻烦 | 纯库，打包简单 |
| 调试 | stdout 是一串 JSON 行，能 pipe | API 请求可见，trace 简单 |
| 进程隔离 | 每次 query 是新进程；清理简单 | SDK 进程会越来越大 |

## Subprocess 赢的场景

- 想让 CLI 层的护栏和审批 UX 在所有地方都一样——每个 SDK 都白拿。
- 需要强进程隔离；坏掉的工具不会撂倒 SDK。
- CLI 已经存在并且功能丰富（`claude` CLI 已经比"自己写 agent loop"强）。

## HTTP 赢的场景

- 在做一个服务端，想同时挂大量 agent，不想给每个 agent 起一个进程。
- 需要精细控制消息循环（比如 mid-stream 自定义注入、deferred tool use）。
- 部署环境不让你带 + 跑独立二进制。

## 这个 SDK 为什么选 subprocess

读上游注释和 README 的意图：**`claude` CLI 是 agent 一切行为的 source of truth**。其它 SDK（Python、TS）只是宿主语言里的薄壳。所有宿主的行为一致，是因为只有 CLI 在跑真正的 agent loop。

后果：宿主上没有 `claude` 二进制，本 SDK 就跑不起来。"in-process MCP" 是唯一例外——它让宿主语言能注册工具让 CLI 回调。其它一切，SDK 都委托给 CLI。

## 对你（用 Go 重写）意味着什么

如果你也是 subprocess 包 CLI，这八章的架构基本白送（CLI 干硬活，你写 spawn + parse）。如果你做 HTTP 原生 agent loop，会发现这八章的分层不对——你想要的是关于 retry、streaming-event 分类、tool-use blocking 语义的章节，不是 subprocess 生命周期的章节。

先选模型，再选库。
