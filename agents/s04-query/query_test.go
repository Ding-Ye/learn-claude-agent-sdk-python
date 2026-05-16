package query

import (
	"context"
	"testing"
	"time"
)

func TestQueryEchoes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, errc := Query(ctx, "hi there", Options{
		CLIPath: "go",
		CLIArgs: []string{"run", "./cli"},
	})

	var msgs []Message
	for m := range out {
		msgs = append(msgs, m)
	}
	if err, ok := <-errc; ok && err != nil {
		t.Fatalf("query error: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("want 3 messages, got %d", len(msgs))
	}
	am, ok := msgs[1].(AssistantMessage)
	if !ok {
		t.Fatalf("second msg should be assistant, got %T", msgs[1])
	}
	if len(am.Content) != 1 {
		t.Fatalf("want 1 block, got %d", len(am.Content))
	}
	tb := am.Content[0].(TextBlock)
	if tb.Text != "echo:hi there" {
		t.Fatalf("bad echo: %q", tb.Text)
	}
	rm := msgs[2].(ResultMessage)
	if rm.Subtype != "end" || rm.IsError {
		t.Fatalf("bad result: %+v", rm)
	}
}

func TestQueryRequiresCLIPath(t *testing.T) {
	ctx := context.Background()
	_, errc := Query(ctx, "x", Options{})
	err := <-errc
	if err == nil {
		t.Fatal("expected error when CLIPath is empty")
	}
}

func TestQueryStopsAfterResult(t *testing.T) {
	// If the CLI emitted extra lines after result, we should still terminate.
	// (Our fake CLI emits exactly one result; this test verifies the channel
	// closes promptly rather than hanging.)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, _ := Query(ctx, "go", Options{CLIPath: "go", CLIArgs: []string{"run", "./cli"}})
	count := 0
	for range out {
		count++
	}
	if count != 3 {
		t.Fatalf("expected 3 messages, got %d", count)
	}
}
