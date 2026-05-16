# s02-transport — spawn a child, speak newline-JSON

The upstream `SubprocessCLITransport` is ~700 lines (env munging, beta headers, atexit cleanup, two flavors of buffer limit). The kernel is much smaller: **bidirectional pipes wrapped in channels, with a graceful close**.

## What you build

- `SubprocessTransport{Path, Args, Env}` — `Start` boots the child, `Send` writes a line of JSON to stdin, `Recv()` returns a channel of stdout lines.
- A `cli/main.go` fake that pretends to be `claude`. It reads one prompt and emits 3 JSON messages.

## Try it

```
cd agents/s02-transport
go test ./...
```

This builds and runs the fake child via `go run ./cli`. No model access, no network.

## Upstream source reading

- `src/claude_agent_sdk/_internal/transport/__init__.py` — the abstract `Transport`.
- `src/claude_agent_sdk/_internal/transport/subprocess_cli.py` — concrete impl. Read `_kill_active_children`, `connect`, and `close` first — the rest is config plumbing.

## What we omitted (on purpose)

- `_ACTIVE_CHILDREN` atexit registry. Easy to add later.
- Buffer size sentinels with a custom JSON-decode-error wrapper.
- The Windows-only signal mapping. Go's `os/exec` already does the right thing.
