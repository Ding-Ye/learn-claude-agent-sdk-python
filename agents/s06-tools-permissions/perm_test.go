package perm

import (
	"context"
	"errors"
	"testing"
)

func mustDecide(t *testing.T, p Policy, tool string) Decision {
	t.Helper()
	d, err := p.Evaluate(context.Background(), tool, nil)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDisallowedBeatsAllowed(t *testing.T) {
	p := Policy{
		AllowedTools:    []string{"Bash", "Read"},
		DisallowedTools: []string{"Bash"},
	}
	d := mustDecide(t, p, "Bash")
	if d.Allow {
		t.Fatalf("Bash should be denied, got %+v", d)
	}
}

func TestAllowlistGrants(t *testing.T) {
	p := Policy{AllowedTools: []string{"Read"}}
	d := mustDecide(t, p, "Read")
	if !d.Allow {
		t.Fatalf("Read should be allowed, got %+v", d)
	}
}

func TestBypassMode(t *testing.T) {
	p := Policy{Mode: ModeBypass}
	d := mustDecide(t, p, "AnythingHere")
	if !d.Allow {
		t.Fatal("bypass should allow")
	}
}

func TestDontAskMode(t *testing.T) {
	p := Policy{Mode: ModeDontAsk}
	d := mustDecide(t, p, "AnythingHere")
	if d.Allow {
		t.Fatal("dontAsk should deny when not pre-approved")
	}
}

func TestCallbackRuns(t *testing.T) {
	called := false
	p := Policy{
		CanUseTool: func(_ context.Context, name string, _ map[string]any) (Decision, error) {
			called = true
			return Decision{Allow: name == "Read"}, nil
		},
	}
	d, err := p.Evaluate(context.Background(), "Read", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !called || !d.Allow {
		t.Fatalf("callback wasn't honored: called=%v allow=%v", called, d.Allow)
	}
	d, _ = p.Evaluate(context.Background(), "Bash", nil)
	if d.Allow {
		t.Fatal("callback denied Bash but result said allowed")
	}
}

func TestCallbackErrorBubbles(t *testing.T) {
	want := errors.New("boom")
	p := Policy{
		CanUseTool: func(_ context.Context, _ string, _ map[string]any) (Decision, error) {
			return Decision{}, want
		},
	}
	_, err := p.Evaluate(context.Background(), "X", nil)
	if !errors.Is(err, want) {
		t.Fatalf("error didn't bubble: %v", err)
	}
}

func TestDefaultDeny(t *testing.T) {
	p := Policy{}
	d := mustDecide(t, p, "Unknown")
	if d.Allow {
		t.Fatal("empty policy should default-deny")
	}
}

func TestWildcardInAllow(t *testing.T) {
	p := Policy{AllowedTools: []string{"*"}}
	d := mustDecide(t, p, "Anything")
	if !d.Allow {
		t.Fatal("wildcard allow should match")
	}
}
