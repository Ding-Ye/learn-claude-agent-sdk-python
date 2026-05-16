package agent

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestFullTurnAllowsAndCallsTool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	preCalled, postCalled := 0, 0
	a := &Agent{
		CLIPath: "go",
		CLIArgs: []string{"run", "./cli"},
		Policy:  Policy{Allowed: []string{"add"}},
		PreToolUse: []Hook{func(_ context.Context, name string, _ map[string]any) error {
			if name == "add" {
				preCalled++
			}
			return nil
		}},
		PostToolUse: []Hook{func(_ context.Context, _ string, _ map[string]any) error {
			postCalled++
			return nil
		}},
		MCP: map[string]MCPTool{
			"add": {
				Name: "add",
				Handler: func(_ context.Context, args map[string]any) (string, error) {
					a := args["a"].(float64)
					b := args["b"].(float64)
					return strconv.FormatFloat(a+b, 'f', -1, 64), nil
				},
			},
		},
	}

	tr, err := a.Turn(ctx, "compute 2+5")
	if err != nil {
		t.Fatalf("turn: %v", err)
	}
	if preCalled != 1 || postCalled != 1 {
		t.Fatalf("hooks not fired: pre=%d post=%d", preCalled, postCalled)
	}
	joined := strings.Join(tr, "|")
	if !strings.Contains(joined, "[text] planning:") {
		t.Fatalf("missing text in transcript: %s", joined)
	}
	if !strings.Contains(joined, "[tool-ok add → 7]") {
		t.Fatalf("tool result missing: %s", joined)
	}
}

func TestFullTurnDeniesDisallowed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := &Agent{
		CLIPath: "go",
		CLIArgs: []string{"run", "./cli"},
		Policy:  Policy{Disallowed: []string{"add"}},
		MCP: map[string]MCPTool{
			"add": {Name: "add", Handler: func(_ context.Context, _ map[string]any) (string, error) {
				t.Fatal("denied tool was called")
				return "", nil
			}},
		},
	}
	tr, err := a.Turn(ctx, "x")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(tr, "|"), "tool-deny") {
		t.Fatalf("expected deny in transcript: %v", tr)
	}
}
