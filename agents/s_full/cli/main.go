// Fake CLI for s_full: reads one prompt envelope, emits assistant message
// that contains BOTH a text block and a tool_use block, then a result.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "no prompt")
		os.Exit(2)
	}
	var env map[string]any
	_ = json.Unmarshal(scanner.Bytes(), &env)
	prompt, _ := env["prompt"].(string)

	emit := func(m map[string]any) {
		b, _ := json.Marshal(m)
		fmt.Println(string(b))
	}
	emit(map[string]any{
		"type": "assistant",
		"content": []map[string]any{
			{"type": "text", "text": "planning: " + prompt},
			{"type": "tool_use", "id": "t1", "name": "add", "input": map[string]any{"a": 2.0, "b": 5.0}},
		},
	})
	emit(map[string]any{"type": "result", "subtype": "end", "duration_ms": 9, "is_error": false})
}
