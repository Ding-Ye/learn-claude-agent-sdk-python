// Package perm decides whether a tool call is allowed.
//
// Upstream:
//
//	src/claude_agent_sdk/types.py — PermissionResult, CanUseTool
//	src/claude_agent_sdk/_internal/query.py — the routing that calls
//	    can_use_tool and applies allow/disallow lists.
//
// The decision chain (in upstream's evaluation order):
//
//	1. disallowed_tools matches name? → DENY
//	2. allowed_tools matches name? → ALLOW
//	3. PermissionMode is "bypassPermissions"? → ALLOW
//	4. PermissionMode is "dontAsk"? → DENY
//	5. CanUseTool callback set? → run it
//	6. Default: DENY
//
// Step 1 winning over step 2 mirrors upstream — disallow is the harder rule.
package perm

import (
	"context"
	"errors"
)

type PermissionMode string

const (
	ModeDefault     PermissionMode = "default"
	ModeAcceptEdits PermissionMode = "acceptEdits"
	ModePlan        PermissionMode = "plan"
	ModeBypass      PermissionMode = "bypassPermissions"
	ModeDontAsk     PermissionMode = "dontAsk"
)

// Decision is the result of the policy chain.
type Decision struct {
	Allow         bool
	Reason        string         // human-readable, for logs
	UpdatedInput  map[string]any // if the callback rewrote the input
	Interruptible bool           // mirrors PermissionResultAllow.interrupt
}

// CanUseTool is the user-supplied callback shape.
type CanUseTool func(ctx context.Context, toolName string, input map[string]any) (Decision, error)

// Policy is the policy bundle a Query/Client would pass into the SDK.
type Policy struct {
	AllowedTools    []string
	DisallowedTools []string
	Mode            PermissionMode
	CanUseTool      CanUseTool
}

// Evaluate applies the chain documented above and returns the decision.
// Errors from the callback bubble back out so callers can fail the turn.
func (p Policy) Evaluate(ctx context.Context, toolName string, input map[string]any) (Decision, error) {
	if matches(p.DisallowedTools, toolName) {
		return Decision{Allow: false, Reason: "matched disallowed_tools"}, nil
	}
	if matches(p.AllowedTools, toolName) {
		return Decision{Allow: true, Reason: "matched allowed_tools"}, nil
	}
	switch p.Mode {
	case ModeBypass:
		return Decision{Allow: true, Reason: "bypassPermissions"}, nil
	case ModeDontAsk:
		return Decision{Allow: false, Reason: "dontAsk and not pre-approved"}, nil
	}
	if p.CanUseTool != nil {
		return p.CanUseTool(ctx, toolName, input)
	}
	return Decision{Allow: false, Reason: "default deny"}, nil
}

// matches does the same wildcard handling as upstream: exact name or "*".
func matches(list []string, name string) bool {
	for _, item := range list {
		if item == "*" || item == name {
			return true
		}
	}
	return false
}

// ErrUserDenied is the sentinel a CanUseTool can return to signal the user
// rejected the prompt. The caller can render it specially.
var ErrUserDenied = errors.New("perm: user denied tool call")
