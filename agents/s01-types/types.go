// Package types is s01's data model: the structs the SDK reads off the CLI's
// stdout and the options it writes back in.
//
// The upstream Python equivalents live in:
//
//	src/claude_agent_sdk/types.py        — all public dataclasses
//	src/claude_agent_sdk/__init__.py     — the re-export surface
//
// In Python the variants are dataclasses unified under typing.Union. In Go we
// use a sealed interface (lowercase tag method) plus a discriminator field
// when the runtime needs to dispatch — same shape, different language.
package types

import (
	"encoding/json"
	"fmt"
)

// ---------- Message variants ----------

// Message is the sum type. UserMessage / AssistantMessage / SystemMessage /
// ResultMessage all implement it. The interface is sealed (the tag method is
// lowercase) so external packages can't sneak in extra variants.
type Message interface {
	messageKind() string
}

type UserMessage struct {
	Content []ContentBlock `json:"content"`
}

func (UserMessage) messageKind() string { return "user" }

type AssistantMessage struct {
	Content []ContentBlock `json:"content"`
	Model   string         `json:"model,omitempty"`
}

func (AssistantMessage) messageKind() string { return "assistant" }

// SystemMessage carries init / hook / mcp_status events. We keep the payload
// generic (the upstream has 30+ subtypes; we don't enumerate them here).
type SystemMessage struct {
	Subtype string         `json:"subtype"`
	Data    map[string]any `json:"data,omitempty"`
}

func (SystemMessage) messageKind() string { return "system" }

// ResultMessage is terminal — emitted once per query when the CLI is done.
type ResultMessage struct {
	Subtype    string `json:"subtype"`
	DurationMs int    `json:"duration_ms"`
	NumTurns   int    `json:"num_turns"`
	SessionID  string `json:"session_id"`
	IsError    bool   `json:"is_error"`
}

func (ResultMessage) messageKind() string { return "result" }

// ---------- Content blocks (inside Assistant/UserMessage.Content) ----------

type ContentBlock interface {
	blockKind() string
}

type TextBlock struct {
	Text string `json:"text"`
}

func (TextBlock) blockKind() string { return "text" }

type ThinkingBlock struct {
	Thinking string `json:"thinking"`
}

func (ThinkingBlock) blockKind() string { return "thinking" }

type ToolUseBlock struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

func (ToolUseBlock) blockKind() string { return "tool_use" }

type ToolResultBlock struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

func (ToolResultBlock) blockKind() string { return "tool_result" }

// ---------- Options (what users pass to query / client) ----------

// PermissionMode mirrors upstream's
// Literal["default","acceptEdits","plan","bypassPermissions","dontAsk","auto"].
type PermissionMode string

const (
	PermissionDefault     PermissionMode = "default"
	PermissionAcceptEdits PermissionMode = "acceptEdits"
	PermissionPlan        PermissionMode = "plan"
	PermissionBypass      PermissionMode = "bypassPermissions"
	PermissionDontAsk     PermissionMode = "dontAsk"
	PermissionAuto        PermissionMode = "auto"
)

type Options struct {
	SystemPrompt    string
	AllowedTools    []string
	DisallowedTools []string
	MaxTurns        int
	CWD             string
	PermissionMode  PermissionMode
}

// ---------- JSON helpers ----------

// EncodeBlock returns the wire form: {"type":"text","text":"hi"} etc.
// Upstream's parse_message expects exactly this shape.
func EncodeBlock(b ContentBlock) ([]byte, error) {
	wrap := map[string]any{"type": b.blockKind()}
	switch v := b.(type) {
	case TextBlock:
		wrap["text"] = v.Text
	case ThinkingBlock:
		wrap["thinking"] = v.Thinking
	case ToolUseBlock:
		wrap["id"] = v.ID
		wrap["name"] = v.Name
		wrap["input"] = v.Input
	case ToolResultBlock:
		wrap["tool_use_id"] = v.ToolUseID
		wrap["content"] = v.Content
		if v.IsError {
			wrap["is_error"] = true
		}
	default:
		return nil, fmt.Errorf("types: unknown ContentBlock %T", b)
	}
	return json.Marshal(wrap)
}

// DecodeBlock is the inverse of EncodeBlock. It's the kernel of s03's parser.
func DecodeBlock(raw []byte) (ContentBlock, error) {
	var peek struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &peek); err != nil {
		return nil, fmt.Errorf("types: peek block type: %w", err)
	}
	switch peek.Type {
	case "text":
		var b TextBlock
		return b, json.Unmarshal(raw, &b)
	case "thinking":
		var b ThinkingBlock
		return b, json.Unmarshal(raw, &b)
	case "tool_use":
		var b ToolUseBlock
		return b, json.Unmarshal(raw, &b)
	case "tool_result":
		var b ToolResultBlock
		return b, json.Unmarshal(raw, &b)
	default:
		return nil, fmt.Errorf("types: unknown block type %q", peek.Type)
	}
}
