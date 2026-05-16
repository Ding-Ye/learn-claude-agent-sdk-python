# s01-types — the data model

What the SDK reads off the CLI's stdout, and what it lets users pass in.

## Mental model

The SDK is a **typed pipe**: bytes go down one direction (the prompt), structs come back up (the messages). This chapter is just the structs. No IO, no goroutines, no policy.

## What you build

- `Message` — sealed interface with 4 implementations.
- `ContentBlock` — sealed interface with 4 implementations.
- `Options` — what the user passes when they kick off a query.
- `EncodeBlock` / `DecodeBlock` — JSON ⇄ struct, gated on the `type` discriminator.

## Try it

```
cd agents/s01-types
go test ./...
```

You should see five tests pass. They round-trip text & tool-use blocks through JSON, check that an unknown block type errors cleanly, and assert the tag methods stay stable.

## Upstream source reading

- `src/claude_agent_sdk/types.py` — the master list of dataclasses.
- `src/claude_agent_sdk/__init__.py:130-280` — the public `__all__` re-export list.
- `src/claude_agent_sdk/_internal/message_parser.py:parse_message` — what consumes these structs in the next chapter.

The Python file is ~2000 lines because every TypedDict + hook variant lives there. We pick a subset that's enough to plug into s02-s08.
