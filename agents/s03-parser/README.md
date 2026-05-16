# s03-parser — JSON line → typed Message

The Python `message_parser.py` is a 300-line match statement. Our Go version is the same shape: a switch on the `type` field that routes into a per-variant decoder.

## What you build

- `Parse(line []byte) (Message, error)` for the four top-level message variants.
- A private `decodeBlock` that handles content-block polymorphism inside `assistant`/`user` messages.
- `IsTerminal(Message) bool` — a one-liner s04 will lean on.
- A typed `ParseError` so the caller can surface the offending line.

## Try it

```
cd agents/s03-parser
go test ./...
```

Seven tests cover the happy path, unknown types (error), missing type (error), and an unknown content-block-inside-known-message case (drop, don't fail).

## Upstream source reading

- `src/claude_agent_sdk/_internal/message_parser.py:parse_message` — the big match block we mirror.
- Note the **hook event short-circuit** at the top of `parse_message`: system messages with `subtype=hook_started|hook_response` are routed to a different variant. We skip that in this chapter; s07 picks it up.

## Why we drop unknown blocks

Upstream Python's `_parse_content_blocks` logs a warning and skips unknown block types. This matters because the CLI evolves faster than the SDK — new block types appear, and old SDKs shouldn't crash on them. We replicate that.
