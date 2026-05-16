package types

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestTextBlockRoundTrip(t *testing.T) {
	in := TextBlock{Text: "hello"}
	raw, err := EncodeBlock(in)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["type"] != "text" || wire["text"] != "hello" {
		t.Fatalf("unexpected wire form: %s", raw)
	}
	out, err := DecodeBlock(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round-trip mismatch: in=%+v out=%+v", in, out)
	}
}

func TestToolUseBlockShape(t *testing.T) {
	in := ToolUseBlock{
		ID:    "toolu_abc",
		Name:  "Read",
		Input: map[string]any{"path": "/tmp/x"},
	}
	raw, _ := EncodeBlock(in)
	out, err := DecodeBlock(raw)
	if err != nil {
		t.Fatal(err)
	}
	got := out.(ToolUseBlock)
	if got.Name != "Read" || got.Input["path"] != "/tmp/x" {
		t.Fatalf("bad decode: %+v", got)
	}
}

func TestDecodeUnknownBlock(t *testing.T) {
	_, err := DecodeBlock([]byte(`{"type":"never_heard_of"}`))
	if err == nil {
		t.Fatal("expected error for unknown block type")
	}
}

func TestMessageKindIsStable(t *testing.T) {
	cases := []struct {
		m    Message
		want string
	}{
		{UserMessage{}, "user"},
		{AssistantMessage{}, "assistant"},
		{SystemMessage{}, "system"},
		{ResultMessage{}, "result"},
	}
	for _, c := range cases {
		if got := c.m.messageKind(); got != c.want {
			t.Errorf("messageKind for %T = %q, want %q", c.m, got, c.want)
		}
	}
}

func TestPermissionModeConstants(t *testing.T) {
	// These must match upstream's Literal[...] values exactly — the CLI
	// rejects unknown modes. See types.py:25.
	want := []PermissionMode{
		PermissionDefault, PermissionAcceptEdits, PermissionPlan,
		PermissionBypass, PermissionDontAsk, PermissionAuto,
	}
	for _, m := range want {
		if string(m) == "" {
			t.Fatalf("PermissionMode constant is empty: %v", m)
		}
	}
}
