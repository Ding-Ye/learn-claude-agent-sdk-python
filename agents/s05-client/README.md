# s05-client — streaming, multi-turn

`Query()` from s04 spawns a fresh subprocess per call. That's fine for "what is 2+2?" but wasteful when you want a chat: every turn pays cold-start cost.

`Client` keeps the subprocess alive across turns. You `Connect` once, `Send` repeatedly, and consume from one continuous `Recv()` channel.

## What you build

- `New(opts)` / `Connect(ctx)` / `Send(prompt)` / `Recv()` / `Close()`.
- A pump goroutine that drains stdout into the message channel forever.
- A fake CLI that handles **multiple** prompts (vs s04's one-shot fake).

## Try it

```
cd agents/s05-client
go test ./...
```

`TestClientMultipleTurns` sends "alpha" then "beta" and asserts both responses come back labeled `turn-1:` / `turn-2:`.

## Upstream source reading

- `src/claude_agent_sdk/client.py` — `ClaudeSDKClient`. Compare its `connect` / `query` / `disconnect` lifecycle with ours.
- The Python version supports `interrupt()`, which sends SIGINT to the child. Easy to add — we leave it as an exercise.

## When to use Query vs Client (upstream's tradeoff)

> Use `query()` for fire-and-forget. Use `ClaudeSDKClient` for chat / REPL / anything where you want to react to a turn before sending the next.

That tradeoff applies verbatim to our Go port.
