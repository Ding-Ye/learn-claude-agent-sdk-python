package hooks

import (
	"context"
	"errors"
	"testing"
)

func TestRegistryRoutesByEventName(t *testing.T) {
	var r Registry
	called := 0
	r.Add(Matcher{
		Event: "PreToolUse",
		Callbacks: []Callback{
			func(_ context.Context, _ Event) (Output, error) {
				called++
				return Output{Continue: true}, nil
			},
		},
	})
	_, err := r.Dispatch(context.Background(), Event{Name: "PreToolUse"})
	if err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Fatalf("want 1 call, got %d", called)
	}

	// Unrelated event should not fire it.
	_, _ = r.Dispatch(context.Background(), Event{Name: "Stop"})
	if called != 1 {
		t.Fatalf("Stop event shouldn't have fired PreToolUse hook (got %d)", called)
	}
}

func TestToolFilterScopesMatch(t *testing.T) {
	var r Registry
	bashCalled, readCalled := 0, 0
	r.Add(Matcher{
		Event: "PreToolUse",
		Tools: []string{"Bash"},
		Callbacks: []Callback{
			func(_ context.Context, _ Event) (Output, error) {
				bashCalled++
				return Output{Continue: true}, nil
			},
		},
	})
	r.Add(Matcher{
		Event: "PreToolUse",
		Tools: []string{"Read"},
		Callbacks: []Callback{
			func(_ context.Context, _ Event) (Output, error) {
				readCalled++
				return Output{Continue: true}, nil
			},
		},
	})

	_, _ = r.Dispatch(context.Background(), Event{Name: "PreToolUse", ToolName: "Bash"})
	_, _ = r.Dispatch(context.Background(), Event{Name: "PreToolUse", ToolName: "Read"})
	_, _ = r.Dispatch(context.Background(), Event{Name: "PreToolUse", ToolName: "Edit"})

	if bashCalled != 1 || readCalled != 1 {
		t.Fatalf("bash=%d read=%d", bashCalled, readCalled)
	}
}

func TestDispatchShortCircuitsOnContinueFalse(t *testing.T) {
	var r Registry
	first, second := 0, 0
	r.Add(Matcher{
		Event: "Stop",
		Callbacks: []Callback{
			func(_ context.Context, _ Event) (Output, error) {
				first++
				return Output{Continue: false, StopReason: "wrap up"}, nil
			},
			func(_ context.Context, _ Event) (Output, error) {
				second++
				return Output{Continue: true}, nil
			},
		},
	})
	outs, err := r.Dispatch(context.Background(), Event{Name: "Stop"})
	if err != nil {
		t.Fatal(err)
	}
	if first != 1 || second != 0 {
		t.Fatalf("expected only first to fire; first=%d second=%d", first, second)
	}
	if len(outs) != 1 || outs[0].StopReason != "wrap up" {
		t.Fatalf("unexpected outputs: %+v", outs)
	}
}

func TestErrorAborts(t *testing.T) {
	var r Registry
	want := errors.New("bad")
	r.Add(Matcher{
		Event: "Stop",
		Callbacks: []Callback{
			func(_ context.Context, _ Event) (Output, error) {
				return Output{}, want
			},
		},
	})
	_, err := r.Dispatch(context.Background(), Event{Name: "Stop"})
	if !errors.Is(err, want) {
		t.Fatalf("error didn't bubble: %v", err)
	}
}
