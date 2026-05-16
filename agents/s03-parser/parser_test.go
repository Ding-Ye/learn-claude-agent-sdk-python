package parser

import (
	"testing"
)

func TestParseSystemInit(t *testing.T) {
	m, err := Parse([]byte(`{"type":"system","subtype":"init","session_id":"abc"}`))
	if err != nil {
		t.Fatal(err)
	}
	sm, ok := m.(SystemMessage)
	if !ok {
		t.Fatalf("want SystemMessage, got %T", m)
	}
	if sm.Subtype != "init" {
		t.Fatalf("subtype = %q", sm.Subtype)
	}
}

func TestParseAssistantWithText(t *testing.T) {
	line := []byte(`{"type":"assistant","model":"opus","content":[{"type":"text","text":"hello"}]}`)
	m, err := Parse(line)
	if err != nil {
		t.Fatal(err)
	}
	am, ok := m.(AssistantMessage)
	if !ok {
		t.Fatalf("want AssistantMessage, got %T", m)
	}
	if am.Model != "opus" || len(am.Content) != 1 {
		t.Fatalf("bad message: %+v", am)
	}
	tb, ok := am.Content[0].(TextBlock)
	if !ok || tb.Text != "hello" {
		t.Fatalf("bad first block: %+v", am.Content[0])
	}
}

func TestParseAssistantWithToolUse(t *testing.T) {
	line := []byte(`{"type":"assistant","content":[{"type":"tool_use","id":"t1","name":"Read","input":{"path":"/x"}}]}`)
	m, err := Parse(line)
	if err != nil {
		t.Fatal(err)
	}
	am := m.(AssistantMessage)
	tu := am.Content[0].(ToolUseBlock)
	if tu.Name != "Read" || tu.Input["path"] != "/x" {
		t.Fatalf("bad tool_use: %+v", tu)
	}
}

func TestParseResultIsTerminal(t *testing.T) {
	m, err := Parse([]byte(`{"type":"result","subtype":"end","duration_ms":12,"num_turns":1,"session_id":"s1","is_error":false}`))
	if err != nil {
		t.Fatal(err)
	}
	if !IsTerminal(m) {
		t.Fatal("expected terminal")
	}
	rm := m.(ResultMessage)
	if rm.DurationMs != 12 || rm.NumTurns != 1 {
		t.Fatalf("bad result: %+v", rm)
	}
}

func TestParseUnknownTypeErrors(t *testing.T) {
	_, err := Parse([]byte(`{"type":"never_heard_of_it"}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*ParseError); !ok {
		t.Fatalf("want ParseError, got %T", err)
	}
}

func TestParseMissingType(t *testing.T) {
	_, err := Parse([]byte(`{"foo":"bar"}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseUnknownBlockIsSkipped(t *testing.T) {
	// Mirrors upstream behaviour: unknown content blocks are dropped, not fatal.
	line := []byte(`{"type":"assistant","content":[{"type":"text","text":"a"},{"type":"future_block","x":1},{"type":"text","text":"b"}]}`)
	m, err := Parse(line)
	if err != nil {
		t.Fatal(err)
	}
	am := m.(AssistantMessage)
	if len(am.Content) != 2 {
		t.Fatalf("want 2 blocks after dropping unknown, got %d", len(am.Content))
	}
}
