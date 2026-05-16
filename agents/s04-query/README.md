# s04-query — one-shot stream

Glue s02 (transport) + s03 (parser) into a single function. The user passes a prompt and options; gets back a channel of typed messages.

## Mental model

```
Query(ctx, prompt, opts)
  │
  ▼
spawn CLI ──► write JSON envelope to stdin
              ▲
              │  close stdin (no more prompts)
              ▼
read stdout ──► parse each line ──► send to out channel
              ▲
              │  hit ResultMessage? stop, close channels
              ▼
wait, return
```

## What you build

- `Query(ctx, prompt string, opts Options) (<-chan Message, <-chan error)`
- A small JSON envelope passed in via stdin (mirrors how the real CLI takes options).
- Termination on `ResultMessage` — we don't keep reading.

## Try it

```
cd agents/s04-query
go test ./...
```

The fake `cli/main.go` reads the envelope and emits init → assistant text → result.

## Upstream source reading

- `src/claude_agent_sdk/query.py` — the public function. Note the long docstring that explains when to choose `query()` vs `ClaudeSDKClient` (s05's topic).
- `src/claude_agent_sdk/_internal/query.py` — the actual iterator. ~900 lines because it also handles can_use_tool, hooks, mcp routing. We strip those out for this chapter; they reappear as separate chapters s06/s07/s08.

## Why two channels, not one

Go's idiom for "stream + error" is two channels. Upstream's Python uses an async iterator that *raises* errors as exceptions. Same data, different idiom; the Go form is easier for the caller to compose with `select`.
