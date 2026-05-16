# s07-hooks — fire callbacks on lifecycle events

Hooks let user code interpose on the agent loop: log a tool call, deny it, rewrite its input, decide to stop the turn early.

## Mental model

`Registry` is a list of `Matcher`s. Each Matcher pairs **(event name + optional tool filter)** with one or more `Callback` Go functions. When the CLI emits a `hook_event`, the SDK runs every callback whose matcher matches.

A callback returns an `Output`. The kicker: if `Continue=false`, the chain short-circuits — no later callbacks run, and the SDK sends a "stop the turn" signal back to the CLI.

## What you build

- `Registry.Add(Matcher)` and `Registry.Dispatch(ctx, Event)`.
- Matcher filtering by event name and optional tool name list.
- Short-circuit semantics on `Output.Continue == false`.

## Try it

```
cd agents/s07-hooks
go test ./...
```

Four tests cover routing by event, tool-name scoping, short-circuit semantics, and error propagation.

## Upstream source reading

- `src/claude_agent_sdk/types.py` — search for `HookMatcher`, `HookCallback`, `BaseHookInput`. Each event has its own input dataclass (`PreToolUseHookInput`, `PostToolUseHookInput`, etc.) so the callback gets typed access to the relevant fields.
- `src/claude_agent_sdk/_internal/query.py` — the actual dispatch site. Watch for `if matcher.hook_event_name == ...` blocks.

## What we left out

Upstream has ~12 hook event types and each carries a typed payload struct. Our `Event.Input map[string]any` is the bag-of-bytes shortcut; in a real port you'd swap that for one struct per event.
