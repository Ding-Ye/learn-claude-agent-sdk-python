# s_full — 集成

## Problem

八个独立章节。读完 s08 还看不到"agent"长什么样。这章是把零件拼起来的演示。

## Solution

一个 `Agent` struct 持有 policy（s06）、pre/post hooks（s07）、MCP 工具表（s08）和 CLIPath + CLIArgs（s02 风格的进程起）。`Turn(ctx, prompt)` 跑完整一轮。

## How It Works

每轮：起进程 → 写 envelope 到 stdin → 关 stdin → 抽 stdout → parse → 对每个 `ToolUseBlock`：policy 判定 → pre-hook → MCP 调用 → post-hook → 记录。见 `ResultMessage` 就停。transcript 是 `[]string`，方便测试断言顺序。

## What Changed

- 新模块 `agents/s_full`，前面所有依赖内联。
- `agent_test.go` 覆盖"允许并调用"和"拒绝"两条路径。

## Try It

```
cd agents/s_full
go test ./...
```

两个测试都通过——一个把 `add` 工具路由到 MCP handler 拿到 `7`；另一个用 `Disallowed:["add"]` 拒绝，验证 handler 没被调用。

## Deliberate Omissions

| 特性 | 原因 |
|---|---|
| 把 tool_result 写回 CLI | 真协议需要 CLI 再来一轮；假 CLI 不 loop。 |
| 多轮流式 | s05 已有；这里聚焦集成广度。 |
| Sessions / save / resume | 见 appendix-b。 |

## Upstream Source Reading

- `src/claude_agent_sdk/_internal/query.py`——上游真做这件事那个 900 行函数。把它的主循环跟我们的 `Turn` 对一下。
- `examples/streaming_mode.py`——上游最"像 agent"的例子。当作扩展本 Agent struct 的目标状态。
