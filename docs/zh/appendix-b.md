# 附录 B — 上游源码地图

本仓库每个概念，都映回上游 `claude-agent-sdk-python` 的对应位置。读完一章想"上游做了啥我没做"时翻这里。

上游 pin 在 [`c352a50`](https://github.com/anthropics/claude-agent-sdk-python/tree/c352a509929a712de65637cbafafcc3a1e3ba4f6)。

## 按章节

### s01 — 类型

| 概念 | 上游 file:lines |
|---|---|
| `Message` 接口 ↔ Python `Message` union | `types.py:1-100`（搜 "Message:") |
| `ContentBlock` 接口 ↔ Python dataclass | `types.py:300-600`（TextBlock、ToolUseBlock 等） |
| `Options` ↔ `ClaudeAgentOptions` | `types.py:700-900` |
| `PermissionMode` ↔ Python Literal | `types.py:25` |
| Hook 输入 variants（本仓库省略） | `types.py:1000-2000` |
| MCP server config | `types.py:McpSdkServerConfig` |

### s02 — Transport

| 概念 | 上游 file:lines |
|---|---|
| `Transport` 接口 | `_internal/transport/__init__.py:1-50` |
| `SubprocessTransport` | `_internal/transport/subprocess_cli.py:1-400` |
| Stdout pump | `subprocess_cli.py:_read_stdout` |
| 优雅 Close + kill timeout | `subprocess_cli.py:close` |
| `_kill_active_children` atexit（跳过） | `subprocess_cli.py:34-46` |

### s03 — Parser

| 概念 | 上游 file:lines |
|---|---|
| `parse_message` | `_internal/message_parser.py:parse_message` |
| `_parse_content_blocks` | 同文件，搜 "content" |
| Hook event 短路（s07 接手） | `message_parser.py:50-90` |
| 未知 block 跳过策略 | `message_parser.py`（搜 `warning` + `continue`） |

### s04 — Query

| 概念 | 上游 file:lines |
|---|---|
| 公开 `query()` | `query.py` 整文件 |
| 内部迭代器 | `_internal/query.py:1-200` |
| Options envelope 序列化 | `_internal/query.py:_prompt_for_session` |

### s05 — Client

| 概念 | 上游 file:lines |
|---|---|
| `ClaudeSDKClient` | `client.py` 整文件 |
| `interrupt()`（本仓库跳过） | `client.py:interrupt` |
| 跨 runtime 警告 | `client.py:50-90`（长 docstring） |

### s06 — 工具权限

| 概念 | 上游 file:lines |
|---|---|
| 决策链（list → mode → callback） | `_internal/query.py:_handle_permission_request` |
| `PermissionResultAllow.updated_input` | `types.py:PermissionResultAllow` |
| `CanUseTool` 回调签名 | `types.py:CanUseTool` |

### s07 — Hooks

| 概念 | 上游 file:lines |
|---|---|
| Registry + matchers | `_internal/query.py:_handle_hook_event` |
| `HookMatcher` / `HookCallback` | `types.py:HookMatcher`, `HookCallback` |
| 每事件 typed payload | `types.py:PreToolUseHookInput` 等 |

### s08 — 进程内 MCP

| 概念 | 上游 file:lines |
|---|---|
| `create_sdk_mcp_server` | `__init__.py:create_sdk_mcp_server`（~150 行） |
| `@tool` 装饰器 | `__init__.py:tool` |
| `_python_type_to_json_schema` | `__init__.py`（`create_sdk_mcp_server` 之上） |

## 我们扔到附录的内容

上游有但本仓库没设章节，留指针给好奇的人。

### Sessions

| 文件 | 为什么有趣 |
|---|---|
| `_internal/session_store.py` (194 LOC) | `SessionStore` 接口 + `InMemorySessionStore`。 |
| `_internal/sessions.py` (1918 LOC) | `list_sessions`、`get_session_messages`、文件系统 store。SDK 体量主要在这里。 |
| `_internal/session_summary.py` (232 LOC) | 长 session 自动摘要（调 Claude）。 |
| `_internal/session_resume.py` (534 LOC) | resume + materialize 已存的 session。 |
| `_internal/session_mutations.py` (962 LOC) | rename / tag / delete / fork。 |

想做 chapter 9，从这里入手。最小可用版本：一个 `SessionStore` 接口（Save / Load / List）、一个 `InMemorySessionStore`、一个 JSON-lines 的 `FsSessionStore`。mutation 和 summary 大但各自自洽。

### Sandbox + plugins

| 文件 | 为什么有趣 |
|---|---|
| `types.py:SandboxSettings` 等 | 沙箱旋钮（网络白名单、忽略违规）。真沙箱是 CLI 做的；SDK 只透传。 |
| `types.py:SdkPluginConfig` | 插件发现格式。SDK 侧面很小；CLI 的插件加载器才是逻辑大头。 |

### Beta 特性

| 文件 | 为什么有趣 |
|---|---|
| `types.py:SdkBeta` | opt-in 的 Literal 列表。API 演进时新 flag 加这里。 |
| `types.py:TaskBudget` | beta flag 后面的 token 预算特性。 |

## 怎么用这张地图

读完一章，翻到这里的对应块，把 `.learn/upstream/<file>` 跳到列出的行数 Read 一下。把你的 Go 跟 Python 对一下，看缺了什么，决定对你的 port 重不重要。

`.learn/upstream/` 在生成器跑完后默认保留；删了的话，重新 clone：`git clone --depth 1 https://github.com/anthropics/claude-agent-sdk-python .learn/upstream`。
