// Package agent wires every prior chapter into one end-to-end agent so a
// learner can see how the pieces compose. The dependencies are inlined
// rather than imported across sessions, keeping the "drop into any chapter
// cold" rule.
//
// Flow per turn:
//
//   user.Send(prompt)
//     ↓
//   transport: spawn CLI, write prompt
//     ↓
//   parser: read stdout JSON, emit typed Message
//     ↓
//   for each ToolUseBlock: permissions.Evaluate → hooks.PreToolUse →
//                           mcp.Call (if registered) → result back to CLI
//     ↓
//   ResultMessage closes the turn
package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

// Inlined types --------------------------------------------------

type Message interface{ messageKind() string }
type AssistantMessage struct {
	Content []ContentBlock
}
type ResultMessage struct {
	IsError    bool
	DurationMs int
}

func (AssistantMessage) messageKind() string { return "assistant" }
func (ResultMessage) messageKind() string    { return "result" }

type ContentBlock interface{ blockKind() string }
type TextBlock struct{ Text string }
type ToolUseBlock struct {
	ID, Name string
	Input    map[string]any
}

func (TextBlock) blockKind() string    { return "text" }
func (ToolUseBlock) blockKind() string { return "tool_use" }

// Policy chain (inlined) -----------------------------------------

type CanUseTool func(ctx context.Context, name string, in map[string]any) (bool, string)

type Policy struct {
	Allowed    []string
	Disallowed []string
	Callback   CanUseTool
}

func (p Policy) Decide(ctx context.Context, name string, in map[string]any) (bool, string) {
	for _, x := range p.Disallowed {
		if x == name || x == "*" {
			return false, "disallowed"
		}
	}
	for _, x := range p.Allowed {
		if x == name || x == "*" {
			return true, "allowed"
		}
	}
	if p.Callback != nil {
		return p.Callback(ctx, name, in)
	}
	return false, "default-deny"
}

// Hook registry --------------------------------------------------

type Hook func(ctx context.Context, toolName string, in map[string]any) error

// MCP registry ---------------------------------------------------

type MCPTool struct {
	Name    string
	Handler func(ctx context.Context, args map[string]any) (string, error)
}

// Agent ----------------------------------------------------------

type Agent struct {
	CLIPath string
	CLIArgs []string

	Policy      Policy
	PreToolUse  []Hook
	PostToolUse []Hook
	MCP         map[string]MCPTool
}

// Turn runs one prompt → result cycle. Returns the collected assistant text
// + any tool-call summaries.
func (a *Agent) Turn(ctx context.Context, prompt string) (transcript []string, err error) {
	if a.CLIPath == "" {
		return nil, errors.New("agent: CLIPath required")
	}

	cmd := exec.CommandContext(ctx, a.CLIPath, a.CLIArgs...)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}

	envelope, _ := json.Marshal(map[string]any{"prompt": prompt})
	_, _ = io.WriteString(stdin, string(envelope)+"\n")
	_ = stdin.Close()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		m, perr := parseLine(scanner.Bytes())
		if perr != nil {
			return transcript, perr
		}
		if m == nil {
			continue
		}
		switch v := m.(type) {
		case AssistantMessage:
			for _, b := range v.Content {
				switch bb := b.(type) {
				case TextBlock:
					transcript = append(transcript, "[text] "+bb.Text)
				case ToolUseBlock:
					line, terr := a.handleToolUse(ctx, bb)
					if terr != nil {
						return transcript, terr
					}
					transcript = append(transcript, line)
				}
			}
		case ResultMessage:
			_ = cmd.Wait()
			return transcript, nil
		}
	}
	_ = cmd.Wait()
	if err := scanner.Err(); err != nil {
		return transcript, err
	}
	return transcript, nil
}

func (a *Agent) handleToolUse(ctx context.Context, tu ToolUseBlock) (string, error) {
	allow, reason := a.Policy.Decide(ctx, tu.Name, tu.Input)
	if !allow {
		return fmt.Sprintf("[tool-deny %s: %s]", tu.Name, reason), nil
	}
	for _, h := range a.PreToolUse {
		if err := h(ctx, tu.Name, tu.Input); err != nil {
			return "", err
		}
	}
	if a.MCP != nil {
		if tool, ok := a.MCP[tu.Name]; ok {
			result, err := tool.Handler(ctx, tu.Input)
			if err != nil {
				return "", err
			}
			for _, h := range a.PostToolUse {
				_ = h(ctx, tu.Name, tu.Input)
			}
			return fmt.Sprintf("[tool-ok %s → %s]", tu.Name, result), nil
		}
	}
	return fmt.Sprintf("[tool-ok %s (external)]", tu.Name), nil
}

func parseLine(line []byte) (Message, error) {
	var peek struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &peek); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	switch peek.Type {
	case "assistant":
		var raw struct {
			Content []json.RawMessage `json:"content"`
		}
		_ = json.Unmarshal(line, &raw)
		var blocks []ContentBlock
		for _, rb := range raw.Content {
			var p struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(rb, &p)
			switch p.Type {
			case "text":
				var t struct {
					Text string `json:"text"`
				}
				_ = json.Unmarshal(rb, &t)
				blocks = append(blocks, TextBlock{Text: t.Text})
			case "tool_use":
				var t struct {
					ID    string         `json:"id"`
					Name  string         `json:"name"`
					Input map[string]any `json:"input"`
				}
				_ = json.Unmarshal(rb, &t)
				blocks = append(blocks, ToolUseBlock(t))
			}
		}
		return AssistantMessage{Content: blocks}, nil
	case "result":
		var r struct {
			IsError    bool `json:"is_error"`
			DurationMs int  `json:"duration_ms"`
		}
		_ = json.Unmarshal(line, &r)
		return ResultMessage(r), nil
	}
	return nil, nil
}
