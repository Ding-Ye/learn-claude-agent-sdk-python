# Appendix A — Subprocess vs HTTP transport

A non-code reading. Useful to understand *why* `claude-agent-sdk-python` looks the way it does, before you decide how to build your own agent SDK.

## The two models

**Subprocess model** (this SDK).
The SDK shells out to a CLI binary. The CLI does the work: model dispatch, tool execution, permissions UI, session state. The SDK is glue: spawn, IO, parse, surface results.

**HTTP model** (e.g. the Anthropic Python SDK, the TS Claude Agent SDK).
The SDK calls the model API directly over HTTPS. It manages messages, tool-use blocks, streaming events, and possibly a local tool registry.

## What you trade

| Concern | Subprocess SDK | HTTP SDK |
|---|---|---|
| Where LLM dispatch lives | inside `claude` CLI | inside the SDK process |
| Where tool execution lives | inside `claude` CLI | inside the SDK process (or out-of-process MCP) |
| Where permission UX lives | inside `claude` CLI | the SDK must build it |
| Session storage | the CLI handles `~/.claude/sessions` | the SDK has to design + ship one |
| Versioning | the SDK + CLI versions both matter | only the SDK version matters |
| Distribution | ship + invoke a binary; cross-platform headache | pure library, easier to package |
| Debuggability | stdout is a stream of JSON lines you can pipe | you see API requests, easier to trace |
| Process isolation | each query is a fresh process — easy cleanup | the SDK process keeps growing |

## When subprocess wins

- You want the same CLI-level guardrails and approval UX everywhere — every SDK gets them for free.
- You want strong process isolation; a wedged tool can't take down the SDK.
- The CLI already exists and is feature-rich (the `claude` CLI is more capable than any "agent loop you'd write from scratch").

## When HTTP wins

- You're building a server that wants to keep many agents in flight without per-agent processes.
- You need fine control over the message loop (e.g. custom mid-stream injection, deferred tool use).
- Your deployment environment doesn't let you ship + run a separate binary.

## Why this SDK picked subprocess

Reading upstream comments + the README, the design intent is: **the `claude` CLI is the source of truth for everything an agent does**. Every other SDK (Python, TS) is a thin shepherd in their host language. Behaviors stay consistent across hosts because the CLI is the only thing actually running the agent loop.

A consequence: you cannot meaningfully run this SDK without a `claude` binary on the host. The "in-process MCP" feature is the one exception — it lets the host language register tools the CLI calls back into. Everything else, the SDK delegates.

## What this means for you (porting to Go)

If you also subprocess-wrap a CLI, you get this whole architecture "for free" (the CLI does the hard parts; you write spawn + parse). If you build an HTTP-native agent loop, you'll find these eight chapters describe the wrong layering — you'd want chapters on retry, streaming-event taxonomy, tool-use blocking semantics, and not chapters on subprocess lifecycle.

Pick your model before you pick your library.
