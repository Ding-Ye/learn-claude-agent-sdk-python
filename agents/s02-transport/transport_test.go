package transport

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// We invoke the fake CLI via `go run ./cli`. This keeps the test hermetic.

func TestSubprocessRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tr := &SubprocessTransport{
		Path: "go",
		Args: []string{"run", "./cli"},
	}
	if err := tr.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = tr.Close(ctx) }()

	if err := tr.Send(ctx, "hello"); err != nil {
		t.Fatalf("send: %v", err)
	}

	out, errc := tr.Recv()
	var got []map[string]any
	for line := range out {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("bad line %q: %v", line, err)
		}
		got = append(got, m)
	}
	select {
	case err := <-errc:
		t.Fatalf("unexpected error: %v", err)
	default:
	}

	if len(got) != 3 {
		t.Fatalf("want 3 messages, got %d (%+v)", len(got), got)
	}
	if got[0]["type"] != "system" || got[1]["type"] != "assistant" || got[2]["type"] != "result" {
		t.Fatalf("unexpected message order: %+v", got)
	}
	asst := got[1]
	blocks := asst["content"].([]any)
	first := blocks[0].(map[string]any)
	if !strings.Contains(first["text"].(string), "hello") {
		t.Fatalf("assistant text didn't echo prompt: %+v", first)
	}
}

func TestSendBeforeStart(t *testing.T) {
	tr := &SubprocessTransport{Path: "go", Args: []string{"run", "./cli"}}
	if err := tr.Send(context.Background(), "x"); err == nil {
		t.Fatal("expected error sending on unstarted transport")
	}
}

func TestDoubleStart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tr := &SubprocessTransport{Path: "go", Args: []string{"run", "./cli"}}
	if err := tr.Start(ctx); err != nil {
		t.Fatalf("first start: %v", err)
	}
	defer func() { _ = tr.Close(ctx) }()
	if err := tr.Start(ctx); err == nil {
		t.Fatal("expected error on second Start")
	}
}
