// Package hooks dispatches hook events to user-registered handlers.
//
// Upstream:
//
//	src/claude_agent_sdk/types.py — HookCallback, HookMatcher, HookEventName
//	src/claude_agent_sdk/_internal/query.py — the dispatcher in handle_hook
//
// A hook is a Go function registered against (event_name, tool_filter). When
// the CLI emits an event matching one of those, every registered callback is
// invoked. Callbacks can:
//
//   - return a continue=true output (the default), or
//   - return continue=false to ask the SDK to interrupt the turn, or
//   - return a deny/allow PermissionRequest decision for the PreToolUse case.
package hooks

import (
	"context"
	"sync"
)

// Event is the inbound hook event from the CLI.
type Event struct {
	Name     string // e.g. "PreToolUse", "PostToolUse", "Stop"
	ToolName string // empty for non-tool events
	Input    map[string]any
}

// Output is what a callback returns. Continue=false signals the SDK to
// interrupt the turn; Decision lets PreToolUse callbacks short-circuit a
// permission check.
type Output struct {
	Continue    bool
	StopReason  string // optional, surfaced to the model
	Decision    string // "" | "allow" | "deny"
	DecisionMsg string
	Replace     map[string]any // optional rewritten input (PreToolUse)
}

// Callback runs in the SDK loop; keep it short — the CLI is blocked.
type Callback func(ctx context.Context, ev Event) (Output, error)

// Matcher binds (event name, optional tool filter) to a list of callbacks.
type Matcher struct {
	Event     string   // required
	Tools     []string // empty = match all tools
	Callbacks []Callback
}

// Registry holds all matchers. Safe for concurrent registration + dispatch.
type Registry struct {
	mu       sync.RWMutex
	matchers []Matcher
}

func (r *Registry) Add(m Matcher) {
	r.mu.Lock()
	r.matchers = append(r.matchers, m)
	r.mu.Unlock()
}

// Dispatch runs every callback whose matcher matches `ev`. It collects all
// outputs into a slice. The first callback that returns Continue=false stops
// the chain (matches upstream's short-circuit on stop).
func (r *Registry) Dispatch(ctx context.Context, ev Event) ([]Output, error) {
	r.mu.RLock()
	matched := make([]Callback, 0)
	for _, m := range r.matchers {
		if m.Event != ev.Name {
			continue
		}
		if len(m.Tools) > 0 && !contains(m.Tools, ev.ToolName) {
			continue
		}
		matched = append(matched, m.Callbacks...)
	}
	r.mu.RUnlock()

	outputs := make([]Output, 0, len(matched))
	for _, cb := range matched {
		out, err := cb(ctx, ev)
		if err != nil {
			return outputs, err
		}
		outputs = append(outputs, out)
		if !out.Continue {
			return outputs, nil
		}
	}
	return outputs, nil
}

func contains(list []string, x string) bool {
	for _, item := range list {
		if item == x {
			return true
		}
	}
	return false
}
