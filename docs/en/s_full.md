# s_full — Integration

## Problem

We have eight isolated chapters. A reader who finishes s08 still doesn't see how they snap together into "an agent." This chapter is the demo that proves the parts fit.

## Solution

An `Agent` struct holding the policy (s06), pre/post hooks (s07), an MCP tool map (s08), and a CLIPath + CLIArgs (s02-style spawn). One `Turn(ctx, prompt)` method runs the full loop.

## How It Works

Per turn: spawn → write envelope to stdin → close stdin → drain stdout → parse → for each `ToolUseBlock`, run policy → pre-hook → MCP call → post-hook → record. Stop on `ResultMessage`. The transcript is a `[]string` so tests can assert the order.

## What Changed

- New module `agents/s_full` with all prior concerns inlined.
- `agent_test.go` covers both allow-and-call and deny paths.

## Try It

```
cd agents/s_full
go test ./...
```

Two tests pass — one routes the `add` tool to an MCP handler that returns `7`; the other denies via `Disallowed:["add"]` and verifies the handler was *not* called.

## Deliberate Omissions

| Feature | Why |
|---|---|
| Sending tool_result back to CLI | Real protocol requires a follow-up CLI roundtrip; the fake doesn't loop. |
| Multi-turn streaming | s05 already has it; here we focus on integration breadth. |
| Sessions / save / resume | See appendix-b. |

## Upstream Source Reading

- `src/claude_agent_sdk/_internal/query.py` — the 900-line function that does all of this for real. Compare its main loop to our `Turn`.
- `examples/streaming_mode.py` — the most "agent-like" upstream example. Use it as a goal state when you extend this Agent struct.
