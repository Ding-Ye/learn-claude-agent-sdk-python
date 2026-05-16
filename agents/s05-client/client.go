// Package client is the streaming counterpart to s04's one-shot Query.
//
// Upstream:  src/claude_agent_sdk/client.py — ClaudeSDKClient
//
// Where Query() spawns + closes per call, the Client keeps the subprocess
// alive across multiple turns. Each turn ends when the CLI emits a
// ResultMessage; Send() can then be called again with a follow-up prompt.
package client

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

// ---------- types ----------

type Message interface{ messageKind() string }
type AssistantMessage struct {
	Content []ContentBlock
}
type SystemMessage struct {
	Subtype string
	Data    map[string]any
}
type ResultMessage struct {
	Subtype    string
	DurationMs int
	IsError    bool
}

func (AssistantMessage) messageKind() string { return "assistant" }
func (SystemMessage) messageKind() string    { return "system" }
func (ResultMessage) messageKind() string    { return "result" }

type ContentBlock interface{ blockKind() string }
type TextBlock struct{ Text string }

func (TextBlock) blockKind() string { return "text" }

// ---------- client ----------

type Options struct {
	CLIPath string
	CLIArgs []string
}

type Client struct {
	opts Options

	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	out    chan Message
	errc   chan error
	open   bool
}

// New returns a Client whose subprocess has not been started yet.
func New(opts Options) *Client { return &Client{opts: opts} }

// Connect spawns the subprocess. Call once; subsequent Send/Recv use it.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.open {
		return errors.New("client: already connected")
	}
	cmd := exec.CommandContext(ctx, c.opts.CLIPath, c.opts.CLIArgs...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	c.cmd, c.stdin, c.stdout = cmd, stdin, stdout
	c.out = make(chan Message, 32)
	c.errc = make(chan error, 1)
	c.open = true
	go c.pump()
	return nil
}

func (c *Client) pump() {
	defer close(c.out)
	scanner := bufio.NewScanner(c.stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		m, err := parseLine(scanner.Bytes())
		if err != nil {
			c.errc <- err
			return
		}
		if m == nil {
			continue
		}
		c.out <- m
	}
	if err := scanner.Err(); err != nil {
		c.errc <- err
	}
}

// Send writes a prompt to the subprocess. The Client does NOT block on
// the response; callers consume via Recv() until a ResultMessage closes
// the turn.
func (c *Client) Send(prompt string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.open {
		return errors.New("client: not connected")
	}
	envelope, _ := json.Marshal(map[string]any{"prompt": prompt})
	_, err := io.WriteString(c.stdin, string(envelope)+"\n")
	return err
}

// Recv returns the message channel. The same channel is used across turns.
func (c *Client) Recv() <-chan Message { return c.out }

// Err returns the error channel (buffered, set once at fatal exit).
func (c *Client) Err() <-chan error { return c.errc }

// Close drops the subprocess. Once closed, the client is dead.
func (c *Client) Close() error {
	c.mu.Lock()
	if !c.open {
		c.mu.Unlock()
		return nil
	}
	c.open = false
	_ = c.stdin.Close()
	cmd := c.cmd
	c.mu.Unlock()
	return cmd.Wait()
}

// ---------- helpers ----------

func parseLine(line []byte) (Message, error) {
	var peek struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
	}
	if err := json.Unmarshal(line, &peek); err != nil {
		return nil, fmt.Errorf("client: parse: %w (line=%q)", err, line)
	}
	switch peek.Type {
	case "system":
		var d map[string]any
		_ = json.Unmarshal(line, &d)
		return SystemMessage{Subtype: peek.Subtype, Data: d}, nil
	case "assistant":
		var raw struct {
			Content []json.RawMessage `json:"content"`
		}
		_ = json.Unmarshal(line, &raw)
		blocks := make([]ContentBlock, 0, len(raw.Content))
		for _, rb := range raw.Content {
			var p struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(rb, &p)
			if p.Type == "text" {
				var b struct {
					Text string `json:"text"`
				}
				_ = json.Unmarshal(rb, &b)
				blocks = append(blocks, TextBlock{Text: b.Text})
			}
		}
		return AssistantMessage{Content: blocks}, nil
	case "result":
		var r struct {
			Subtype    string `json:"subtype"`
			DurationMs int    `json:"duration_ms"`
			IsError    bool   `json:"is_error"`
		}
		_ = json.Unmarshal(line, &r)
		return ResultMessage(r), nil
	}
	return nil, nil
}

// IsTerminal reports whether a message ends one turn.
func IsTerminal(m Message) bool {
	_, ok := m.(ResultMessage)
	return ok
}
