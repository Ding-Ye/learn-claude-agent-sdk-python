# s08-mcp — in-process MCP server

The SDK can register Go funcs as tools the CLI knows how to call. "In-process" means the tool runs in the same process — no subprocess, no IPC. Cheap and fast.

## What you build

- `Tool{Name, Description, Schema, Handler}`.
- `Server.New(name, version, tools...)` + `Call(ctx, name, args)`.
- `SchemaFromStruct(zeroValue)` — a reflection helper that builds a JSON Schema object from struct tags.
- `Text(s)` convenience for the common "return a string" tool.

## Try it

```
cd agents/s08-mcp
go test ./...
```

The tests build a calculator MCP server with one `add` tool, call it, and check the schema reflection.

## Upstream source reading

- `src/claude_agent_sdk/__init__.py:create_sdk_mcp_server` — the constructor.
- `src/claude_agent_sdk/__init__.py:tool` — the `@tool` decorator (Python's way of stamping name/desc/schema onto a function).
- `src/claude_agent_sdk/__init__.py:_python_type_to_json_schema` — the reflection-equivalent in Python. Our `SchemaFromStruct` is the Go counterpart.

## Python `@tool` ↔ Go pattern

Python:
```python
@tool("add", "Add two numbers", {"a": float, "b": float})
async def add(args):
    return {"content": [{"type": "text", "text": f"sum={args['a']+args['b']}"}]}
```

Go:
```go
type AddArgs struct {
    A float64 `json:"a" mcp:"first addend"`
    B float64 `json:"b" mcp:"second addend"`
}

mcp.New("calc", "1.0",
    mcp.Tool{
        Name:        "add",
        Description: "Add two numbers",
        Schema:      mcp.SchemaFromStruct(AddArgs{}),
        Handler:     addHandler,
    },
)
```

Same data, different idiom — Python uses runtime decorators + typed dict reflection; Go uses struct tags + reflect.
