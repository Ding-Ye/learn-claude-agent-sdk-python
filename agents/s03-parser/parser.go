// Package parser turns the line-delimited JSON the CLI emits into typed
// Message values. Upstream's counterpart is `parse_message` in
//
//	src/claude_agent_sdk/_internal/message_parser.py
//
// That function is a big match statement on the `type` field. We do the same
// with a Go switch — the only twist is content blocks (assistant/user
// messages embed a list of polymorphic blocks).
package parser

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Types are inlined per the "no cross-session imports" rule — a learner
// can drop into s03 cold without dragging in s01.

type Message interface{ messageKind() string }
type UserMessage struct{ Content []ContentBlock }
type AssistantMessage struct {
	Content []ContentBlock
	Model   string
}
type SystemMessage struct {
	Subtype string
	Data    map[string]any
}
type ResultMessage struct {
	Subtype    string
	DurationMs int
	NumTurns   int
	SessionID  string
	IsError    bool
}

func (UserMessage) messageKind() string      { return "user" }
func (AssistantMessage) messageKind() string { return "assistant" }
func (SystemMessage) messageKind() string    { return "system" }
func (ResultMessage) messageKind() string    { return "result" }

type ContentBlock interface{ blockKind() string }
type TextBlock struct{ Text string }
type ThinkingBlock struct{ Thinking string }
type ToolUseBlock struct {
	ID, Name string
	Input    map[string]any
}
type ToolResultBlock struct {
	ToolUseID, Content string
	IsError            bool
}

func (TextBlock) blockKind() string       { return "text" }
func (ThinkingBlock) blockKind() string   { return "thinking" }
func (ToolUseBlock) blockKind() string    { return "tool_use" }
func (ToolResultBlock) blockKind() string { return "tool_result" }

// ParseError wraps a parse failure with the raw JSON for debugging.
type ParseError struct {
	Reason string
	Raw    string
}

func (e *ParseError) Error() string { return fmt.Sprintf("parser: %s: %s", e.Reason, e.Raw) }

// Parse takes one JSON line (no trailing newline) and returns a typed Message.
// It returns (nil, nil) for messages we deliberately don't handle yet, mirroring
// upstream which returns None for some hook subtypes.
func Parse(line []byte) (Message, error) {
	var peek struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
	}
	if err := json.Unmarshal(line, &peek); err != nil {
		return nil, &ParseError{"invalid JSON", string(line)}
	}
	if peek.Type == "" {
		return nil, &ParseError{"missing type field", string(line)}
	}

	switch peek.Type {
	case "user":
		var raw struct {
			Content []json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(line, &raw); err != nil {
			return nil, &ParseError{"bad user payload", string(line)}
		}
		blocks, err := decodeBlocks(raw.Content)
		if err != nil {
			return nil, err
		}
		return UserMessage{Content: blocks}, nil

	case "assistant":
		var raw struct {
			Model   string            `json:"model"`
			Content []json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(line, &raw); err != nil {
			return nil, &ParseError{"bad assistant payload", string(line)}
		}
		blocks, err := decodeBlocks(raw.Content)
		if err != nil {
			return nil, err
		}
		return AssistantMessage{Model: raw.Model, Content: blocks}, nil

	case "system":
		var data map[string]any
		_ = json.Unmarshal(line, &data)
		return SystemMessage{Subtype: peek.Subtype, Data: data}, nil

	case "result":
		var rm ResultMessage
		// Custom field names — manual mapping keeps Go tags simple.
		var raw struct {
			Subtype    string `json:"subtype"`
			DurationMs int    `json:"duration_ms"`
			NumTurns   int    `json:"num_turns"`
			SessionID  string `json:"session_id"`
			IsError    bool   `json:"is_error"`
		}
		if err := json.Unmarshal(line, &raw); err != nil {
			return nil, &ParseError{"bad result payload", string(line)}
		}
		rm = ResultMessage(raw)
		return rm, nil
	}

	return nil, &ParseError{"unknown message type " + peek.Type, string(line)}
}

func decodeBlocks(raws []json.RawMessage) ([]ContentBlock, error) {
	out := make([]ContentBlock, 0, len(raws))
	for _, raw := range raws {
		b, err := decodeBlock(raw)
		if err != nil {
			return nil, err
		}
		if b != nil {
			out = append(out, b)
		}
	}
	return out, nil
}

func decodeBlock(raw json.RawMessage) (ContentBlock, error) {
	var peek struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &peek); err != nil {
		return nil, &ParseError{"bad block", string(raw)}
	}
	switch peek.Type {
	case "text":
		var b struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, &ParseError{"bad text block", string(raw)}
		}
		return TextBlock{Text: b.Text}, nil
	case "thinking":
		var b struct {
			Thinking string `json:"thinking"`
		}
		_ = json.Unmarshal(raw, &b)
		return ThinkingBlock{Thinking: b.Thinking}, nil
	case "tool_use":
		var b struct {
			ID    string         `json:"id"`
			Name  string         `json:"name"`
			Input map[string]any `json:"input"`
		}
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, &ParseError{"bad tool_use block", string(raw)}
		}
		return ToolUseBlock{ID: b.ID, Name: b.Name, Input: b.Input}, nil
	case "tool_result":
		var b struct {
			ToolUseID string `json:"tool_use_id"`
			Content   string `json:"content"`
			IsError   bool   `json:"is_error"`
		}
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, &ParseError{"bad tool_result block", string(raw)}
		}
		return ToolResultBlock{ToolUseID: b.ToolUseID, Content: b.Content, IsError: b.IsError}, nil
	}
	// Mirror upstream: skip unknown block types rather than fail the whole message.
	return nil, nil
}

// IsTerminal returns true for messages that mean "the query is done".
// Useful for the s04 query iterator to know when to close the loop.
func IsTerminal(m Message) bool {
	_, ok := m.(ResultMessage)
	return ok
}

var _ = errors.New // keep errors imported even if compile-time unused
