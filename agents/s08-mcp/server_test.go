package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// A small calculator server mirroring upstream's examples/mcp_calculator.py.

type AddArgs struct {
	A float64 `json:"a" mcp:"first addend"`
	B float64 `json:"b" mcp:"second addend"`
}

func addHandler(_ context.Context, args map[string]any) (CallResult, error) {
	a := args["a"].(float64)
	b := args["b"].(float64)
	return Text("sum=" + numToStr(a+b)), nil
}

func numToStr(f float64) string {
	// quick fmt without importing strconv twice
	s := []byte{}
	if f == float64(int64(f)) {
		// integer-valued
		n := int64(f)
		if n == 0 {
			return "0"
		}
		neg := n < 0
		if neg {
			n = -n
		}
		for n > 0 {
			s = append([]byte{byte('0' + n%10)}, s...)
			n /= 10
		}
		if neg {
			s = append([]byte{'-'}, s...)
		}
		return string(s)
	}
	// fallback to a fixed format
	return "f"
}

func TestRegisterAndCall(t *testing.T) {
	srv := New("calc", "1.0",
		Tool{
			Name:        "add",
			Description: "Add two numbers",
			Schema:      SchemaFromStruct(AddArgs{}),
			Handler:     addHandler,
		},
	)

	r, err := srv.Call(context.Background(), "add", map[string]any{"a": 2.0, "b": 3.0})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Content) != 1 || r.Content[0].Text != "sum=5" {
		t.Fatalf("bad result: %+v", r)
	}
}

func TestUnknownTool(t *testing.T) {
	srv := New("empty", "1.0")
	_, err := srv.Call(context.Background(), "ghost", nil)
	if !errors.Is(err, ErrUnknownTool) {
		t.Fatalf("want ErrUnknownTool, got %v", err)
	}
}

func TestListToolsSorted(t *testing.T) {
	srv := New("multi", "1.0",
		Tool{Name: "zap", Handler: addHandler},
		Tool{Name: "alpha", Handler: addHandler},
		Tool{Name: "mid", Handler: addHandler},
	)
	tools := srv.ListTools()
	names := []string{tools[0].Name, tools[1].Name, tools[2].Name}
	want := []string{"alpha", "mid", "zap"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("want %v got %v", want, names)
	}
}

func TestSchemaFromStruct(t *testing.T) {
	s := SchemaFromStruct(AddArgs{})
	if s["type"] != "object" {
		t.Fatalf("want type=object got %v", s["type"])
	}
	props := s["properties"].(map[string]any)
	a := props["a"].(map[string]any)
	if a["type"] != "number" || a["description"] != "first addend" {
		t.Fatalf("bad schema for a: %+v", a)
	}
	required := s["required"].([]string)
	if len(required) != 2 {
		t.Fatalf("want 2 required fields, got %v", required)
	}
}

func TestTextHelper(t *testing.T) {
	r := Text("hi")
	if len(r.Content) != 1 || r.Content[0].Type != "text" || r.Content[0].Text != "hi" {
		t.Fatalf("bad Text result: %+v", r)
	}
	if r.IsError {
		t.Fatal("Text should not mark IsError by default")
	}
}
