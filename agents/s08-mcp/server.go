// Package mcp implements an in-process MCP server for registering Go funcs
// as tools the CLI can call. The "MCP" framing matters: the CLI hands tool
// calls to whichever server claims the name; in-process means the server
// lives in the same Go program as the SDK, no IPC.
//
// Upstream:
//
//	src/claude_agent_sdk/__init__.py  — create_sdk_mcp_server, @tool, _python_type_to_json_schema
//	src/claude_agent_sdk/types.py     — McpSdkServerConfig
//
// Python uses the @tool decorator and reflection on the input dataclass to
// build a JSON Schema. Go doesn't have decorators or named structural
// reflection on function args, so the public API explicitly takes a
// schema map alongside the handler. A reflection-based helper
// SchemaFromStruct[T] is provided for the common case.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// CallResult mirrors upstream's MCP tool-result shape.
type CallResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"is_error,omitempty"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// Handler is the function the user registers.
type Handler func(ctx context.Context, args map[string]any) (CallResult, error)

// Tool is one registered function.
type Tool struct {
	Name        string
	Description string
	Schema      map[string]any // JSON Schema object
	Handler     Handler
}

// Server holds a map of registered tools, indexed by name.
type Server struct {
	Name    string
	Version string
	tools   map[string]Tool
}

func New(name, version string, tools ...Tool) *Server {
	s := &Server{Name: name, Version: version, tools: make(map[string]Tool, len(tools))}
	for _, t := range tools {
		s.tools[t.Name] = t
	}
	return s
}

// ListTools returns tool descriptors sorted by name (stable for tests).
func (s *Server) ListTools() []Tool {
	out := make([]Tool, 0, len(s.tools))
	for _, t := range s.tools {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Call dispatches by name. Returns ErrUnknownTool if not registered.
func (s *Server) Call(ctx context.Context, name string, args map[string]any) (CallResult, error) {
	t, ok := s.tools[name]
	if !ok {
		return CallResult{}, fmt.Errorf("%w: %s", ErrUnknownTool, name)
	}
	return t.Handler(ctx, args)
}

// ErrUnknownTool is returned by Call when no tool matches the requested name.
var ErrUnknownTool = errors.New("mcp: unknown tool")

// ---------- helpers for schema derivation ----------

// SchemaFromStruct uses reflection to build a JSON Schema object from a Go
// struct type. Tags supported:
//
//	`json:"name,omitempty"`         — field name + optional marker
//	`mcp:"description here"`        — field description
//
// Unsupported fields fall back to {"type":"string"} so the call still works.
func SchemaFromStruct(zero any) map[string]any {
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return map[string]any{"type": "object"}
	}

	props := map[string]any{}
	required := []string{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		jsonTag := f.Tag.Get("json")
		name, omitempty := parseJSONTag(jsonTag, f.Name)
		if name == "-" {
			continue
		}
		field := map[string]any{"type": goKindToJSONSchema(f.Type.Kind())}
		if desc := f.Tag.Get("mcp"); desc != "" {
			field["description"] = desc
		}
		props[name] = field
		if !omitempty {
			required = append(required, name)
		}
	}
	sort.Strings(required)
	out := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}

func parseJSONTag(tag, fallback string) (name string, omitempty bool) {
	if tag == "" {
		return fallback, false
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	if name == "" {
		name = fallback
	}
	for _, p := range parts[1:] {
		if p == "omitempty" {
			omitempty = true
		}
	}
	return
}

func goKindToJSONSchema(k reflect.Kind) string {
	switch k {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Map, reflect.Struct:
		return "object"
	case reflect.Slice, reflect.Array:
		return "array"
	default:
		return "string"
	}
}

// ---------- text-result convenience ----------

// Text builds a CallResult with a single text content item. Use for the
// common case of "return a string and you're done."
func Text(s string) CallResult {
	return CallResult{Content: []ContentItem{{Type: "text", Text: s}}}
}

// MustEncode is a tiny helper for tests/examples that need to serialize a
// CallResult to inspect it.
func MustEncode(r CallResult) string {
	b, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}
	return string(b)
}
