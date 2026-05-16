# s_full — integration

Every prior chapter, glued together. One `Agent` struct holds the policy, hook lists, MCP tool map, and CLI invocation. One `Turn(ctx, prompt)` runs the full loop.

## Architecture

```
              ┌──────────────────────────────────────────────────────┐
              │                       Agent                          │
              │                                                      │
prompt ──────►│  ┌─────────┐  ┌────────┐  ┌────────┐  ┌─────────┐    │──► transcript
              │  │transport│──│parser  │──│policy  │──│hooks +  │    │
              │  │(s02)    │  │(s03)   │  │(s06)   │  │MCP (s07,│    │
              │  └─────────┘  └────────┘  └────────┘  │     s08)│    │
              │                                       └─────────┘    │
              └──────────────────────────────────────────────────────┘
```

## Execution trace (one turn, this test)

1. Caller: `a.Turn(ctx, "compute 2+5")`.
2. Agent spawns `go run ./cli`.
3. Agent writes `{"prompt":"compute 2+5"}\n` to its stdin, closes stdin.
4. Child emits a JSON line: `assistant` with a text block + a tool_use block (`add(2,5)`).
5. Parser yields `AssistantMessage{Content:[TextBlock, ToolUseBlock]}`.
6. Agent appends `[text] planning: compute 2+5` to transcript.
7. For the `ToolUseBlock("add", {a:2,b:5})`:
   - **Policy.Decide("add", ...)** → matches `Allowed:["add"]` → allow.
   - **PreToolUse hooks** → `preCalled=1`.
   - **MCP.Call("add", {a:2,b:5})** → handler returns `"7"`.
   - **PostToolUse hooks** → `postCalled=1`.
   - Transcript line: `[tool-ok add → 7]`.
8. Child emits `result` line; parser yields `ResultMessage{IsError:false}`.
9. Agent returns transcript.

## Deliberate omissions

| Feature | Why skipped |
|---|---|
| Sending the tool result *back* to the CLI to feed Claude | Real protocol needs the CLI's ToolResult message format; the fake CLI doesn't loop. |
| Streaming `ClaudeSDKClient` style across multiple turns | s05 covers that; here we focus on integration breadth. |
| Sessions / save / resume | Punted to appendix-b. |
| Real JSON Schema generation for tool args | s08 has a working `SchemaFromStruct` reflection helper; the integration just inlines a tiny handler. |

## Try it

```
cd agents/s_full
go test ./...
```
