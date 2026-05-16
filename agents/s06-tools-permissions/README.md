# s06-tools-permissions — the policy chain

The CLI asks the SDK "may Claude call tool X with these args?" The SDK applies a six-step chain and returns allow/deny. This chapter implements just that chain, no IO.

## Decision order

1. `disallowed_tools` matches → **deny** (this is the strongest rule)
2. `allowed_tools` matches → allow
3. mode = `bypassPermissions` → allow
4. mode = `dontAsk` → deny
5. `CanUseTool` callback set → run it
6. fallback → deny

## What you build

- `Policy` struct (lists + mode + callback).
- `Decision` struct (allow + reason + optional rewritten input).
- `Policy.Evaluate(ctx, toolName, input)` → `(Decision, error)`.

## Try it

```
cd agents/s06-tools-permissions
go test ./...
```

Nine tests cover: disallowed-wins-over-allowed, bypass, dontAsk, callback path, callback errors, default-deny, wildcards.

## Upstream source reading

- `src/claude_agent_sdk/types.py` — `PermissionMode`, `CanUseTool`, `PermissionResultAllow`, `PermissionResultDeny`. Note `PermissionResultAllow.updated_input` lets the callback **rewrite** the args before the tool runs — we expose the same field via `Decision.UpdatedInput`.
- `src/claude_agent_sdk/_internal/query.py` — search for `can_use_tool`; the routing into the callback lives there.

## Why deny wins

The upstream comment on `disallowed_tools`: "you can't accidentally re-allow a tool via overlap with the allow list." Treating the deny list as the highest-priority rule means a guardrails team can set it once and trust it.
