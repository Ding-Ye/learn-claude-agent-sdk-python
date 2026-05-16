// Package query is the one-shot iterator: spawn the CLI, write a prompt,
// stream typed messages back, close when we see ResultMessage.
//
// Upstream:
//
//	src/claude_agent_sdk/query.py        — the public async function
//	src/claude_agent_sdk/_internal/query.py — the internal iterator
//
// Our function returns two channels (messages + error) instead of returning
// an AsyncIterator[Message]. Same semantics, different idiom.
package query

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// ---------- types (inlined from s01) ----------

type Message interface{ messageKind() string }
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

func (AssistantMessage) messageKind() string { return "assistant" }
func (SystemMessage) messageKind() string    { return "system" }
func (ResultMessage) messageKind() string    { return "result" }

type ContentBlock interface{ blockKind() string }
type TextBlock struct{ Text string }
type ToolUseBlock struct {
	ID, Name string
	Input    map[string]any
}

func (TextBlock) blockKind() string    { return "text" }
func (ToolUseBlock) blockKind() string { return "tool_use" }

// ---------- options ----------

type Options struct {
	CLIPath        string   // executable to spawn
	CLIArgs        []string // extra args
	SystemPrompt   string
	AllowedTools   []string
	PermissionMode string
	MaxTurns       int
}

// ---------- public Query ----------

// Query spawns the CLI, sends `prompt`, and streams Message values until a
// ResultMessage closes the loop. The returned channels are closed when the
// stream ends; check `errc` for any IO/parse error.
func Query(ctx context.Context, prompt string, opts Options) (<-chan Message, <-chan error) {
	out := make(chan Message, 16)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		defer close(errc)
		if err := run(ctx, prompt, opts, out); err != nil {
			errc <- err
		}
	}()
	return out, errc
}

func run(ctx context.Context, prompt string, opts Options, out chan<- Message) error {
	if opts.CLIPath == "" {
		return errors.New("query: Options.CLIPath is required")
	}
	cmd := exec.CommandContext(ctx, opts.CLIPath, opts.CLIArgs...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	// Build the "init prompt" envelope and write it.
	header, _ := json.Marshal(map[string]any{
		"prompt":          prompt,
		"system_prompt":   opts.SystemPrompt,
		"allowed_tools":   opts.AllowedTools,
		"permission_mode": opts.PermissionMode,
		"max_turns":       opts.MaxTurns,
	})
	if _, err := io.WriteString(stdin, string(header)+"\n"); err != nil {
		return fmt.Errorf("write prompt: %w", err)
	}
	_ = stdin.Close() // CLI will read till EOF and exit

	var wg sync.WaitGroup
	wg.Add(1)
	parseErr := make(chan error, 1)
	go func() {
		defer wg.Done()
		parseErr <- pump(stdout, out)
	}()

	wg.Wait()
	close(parseErr)
	if err := <-parseErr; err != nil {
		_ = cmd.Wait()
		return err
	}
	return cmd.Wait()
}

func pump(r io.Reader, out chan<- Message) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		m, err := parseLine(scanner.Bytes())
		if err != nil {
			return err
		}
		if m == nil {
			continue
		}
		out <- m
		if _, terminal := m.(ResultMessage); terminal {
			return nil
		}
	}
	return scanner.Err()
}

// ---------- minimal parser (inlined from s03) ----------

func parseLine(line []byte) (Message, error) {
	var peek struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
	}
	if err := json.Unmarshal(line, &peek); err != nil {
		return nil, fmt.Errorf("parse: %w (line=%q)", err, line)
	}
	switch peek.Type {
	case "system":
		var data map[string]any
		_ = json.Unmarshal(line, &data)
		return SystemMessage{Subtype: peek.Subtype, Data: data}, nil
	case "assistant":
		var raw struct {
			Model   string            `json:"model"`
			Content []json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(line, &raw); err != nil {
			return nil, err
		}
		blocks := make([]ContentBlock, 0, len(raw.Content))
		for _, rb := range raw.Content {
			b, err := parseBlock(rb)
			if err != nil {
				return nil, err
			}
			if b != nil {
				blocks = append(blocks, b)
			}
		}
		return AssistantMessage{Model: raw.Model, Content: blocks}, nil
	case "result":
		var raw struct {
			Subtype    string `json:"subtype"`
			DurationMs int    `json:"duration_ms"`
			NumTurns   int    `json:"num_turns"`
			SessionID  string `json:"session_id"`
			IsError    bool   `json:"is_error"`
		}
		if err := json.Unmarshal(line, &raw); err != nil {
			return nil, err
		}
		return ResultMessage(raw), nil
	}
	// Quietly drop unknown top-level message types.
	return nil, nil
}

func parseBlock(raw json.RawMessage) (ContentBlock, error) {
	var peek struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &peek); err != nil {
		return nil, err
	}
	switch peek.Type {
	case "text":
		var b struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, err
		}
		return TextBlock{Text: b.Text}, nil
	case "tool_use":
		var b struct {
			ID    string         `json:"id"`
			Name  string         `json:"name"`
			Input map[string]any `json:"input"`
		}
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, err
		}
		return ToolUseBlock(b), nil
	}
	return nil, nil
}
