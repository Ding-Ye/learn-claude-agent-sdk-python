# Appendix B — Upstream source map

Every concept in this repo, mapped back to its home in `claude-agent-sdk-python`. Use this when you finish a chapter and want to know "what did the upstream do beyond what I built?"

Upstream pinned to commit [`c352a50`](https://github.com/anthropics/claude-agent-sdk-python/tree/c352a509929a712de65637cbafafcc3a1e3ba4f6).

## By chapter

### s01 — Types

| Concept | Upstream file:lines |
|---|---|
| `Message` interface ↔ Python `Message` union | `types.py:1-100` (search "Message:") |
| `ContentBlock` interface ↔ Python dataclasses | `types.py:300-600` (TextBlock, ToolUseBlock, etc.) |
| `Options` ↔ `ClaudeAgentOptions` | `types.py:700-900` |
| `PermissionMode` ↔ `PermissionMode` Literal | `types.py:25` |
| Hook input variants (we omit) | `types.py:1000-2000` (PreToolUseHookInput, etc.) |
| MCP server config | `types.py:McpSdkServerConfig` |

### s02 — Transport

| Concept | Upstream file:lines |
|---|---|
| `Transport` interface | `_internal/transport/__init__.py:1-50` |
| `SubprocessTransport` | `_internal/transport/subprocess_cli.py:1-400` |
| Stdout pump (`pumpStdout`) | `subprocess_cli.py:_read_stdout` |
| Graceful Close + kill timeout | `subprocess_cli.py:close` |
| `_kill_active_children` atexit (we skip) | `subprocess_cli.py:34-46` |

### s03 — Parser

| Concept | Upstream file:lines |
|---|---|
| `parse_message` | `_internal/message_parser.py:parse_message` |
| `_parse_content_blocks` | `_internal/message_parser.py` (same file, search "content") |
| Hook event short-circuit (skipped; see s07) | `message_parser.py:50-90` |
| Unknown-block-skip policy | `message_parser.py` (look for `warning` + `continue`) |

### s04 — Query

| Concept | Upstream file:lines |
|---|---|
| Public `query()` | `query.py` (whole file) |
| Internal iterator | `_internal/query.py:1-200` |
| Options envelope serialization | `_internal/query.py:_prompt_for_session` |

### s05 — Client

| Concept | Upstream file:lines |
|---|---|
| `ClaudeSDKClient` | `client.py` (whole file) |
| `interrupt()` (we skipped) | `client.py:interrupt` |
| Cross-runtime caveat | `client.py:50-90` (long docstring) |

### s06 — Tools & Permissions

| Concept | Upstream file:lines |
|---|---|
| Decision chain (lists → mode → callback) | `_internal/query.py:_handle_permission_request` |
| `PermissionResultAllow.updated_input` | `types.py:PermissionResultAllow` |
| `CanUseTool` callback signature | `types.py:CanUseTool` |

### s07 — Hooks

| Concept | Upstream file:lines |
|---|---|
| Registry + matchers | `_internal/query.py:_handle_hook_event` |
| `HookMatcher` / `HookCallback` | `types.py:HookMatcher`, `HookCallback` |
| Per-event typed payloads | `types.py:PreToolUseHookInput` etc. |

### s08 — In-Process MCP

| Concept | Upstream file:lines |
|---|---|
| `create_sdk_mcp_server` | `__init__.py:create_sdk_mcp_server` (~150 lines) |
| `@tool` decorator | `__init__.py:tool` |
| `_python_type_to_json_schema` | `__init__.py` (above `create_sdk_mcp_server`) |

## What we punted to the appendix

These exist in upstream but get no chapter here. Pointer for the curious:

### Sessions

| File | Why interesting |
|---|---|
| `_internal/session_store.py` (194 LOC) | The `SessionStore` interface + `InMemorySessionStore`. |
| `_internal/sessions.py` (1918 LOC) | `list_sessions`, `get_session_messages`, the file-system store. The bulk of the SDK. |
| `_internal/session_summary.py` (232 LOC) | Auto-summary of long sessions (calls Claude). |
| `_internal/session_resume.py` (534 LOC) | Resume + materialize a saved session. |
| `_internal/session_mutations.py` (962 LOC) | Rename / tag / delete / fork. |

If you wanted a chapter 9, this is where it'd start. The minimum useful build is: a `SessionStore` interface (Save / Load / List), an `InMemorySessionStore`, and a JSON-lines `FsSessionStore`. The mutations and summaries are large but each is self-contained.

### Sandbox + plugins

| File | Why interesting |
|---|---|
| `types.py:SandboxSettings` etc. | Sandbox knobs (network allow-list, ignored violations). The CLI does the actual sandboxing; the SDK just passes config through. |
| `types.py:SdkPluginConfig` | Plugin discovery format. Minimal SDK-side surface; the CLI plugin loader has all the logic. |

### Beta features

| File | Why interesting |
|---|---|
| `types.py:SdkBeta` | The Literal of opt-in beta flags. New flags appear here as the API evolves. |
| `types.py:TaskBudget` | Token-budget feature gated behind a beta flag. |

## How to use this map

When you finish a chapter, scroll to its block here and `Read .learn/upstream/<file>` from the lines listed. Compare your Go to the Python — note what's missing, decide if it matters for your port.

`.learn/upstream/` is preserved by default after the generator runs; if you removed it, re-clone with `git clone --depth 1 https://github.com/anthropics/claude-agent-sdk-python .learn/upstream`.
