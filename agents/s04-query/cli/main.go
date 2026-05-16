// Fake CLI for s04 tests: reads a JSON prompt envelope, echoes it back as an
// assistant text block, then emits a result.
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
	var hdr map[string]any
	_ = json.Unmarshal(scanner.Bytes(), &hdr)
	prompt, _ := hdr["prompt"].(string)

	emit := func(m map[string]any) {
		b, _ := json.Marshal(m)
		fmt.Println(string(b))
	}
	emit(map[string]any{"type": "system", "subtype": "init", "session_id": "s4"})
	emit(map[string]any{
		"type":    "assistant",
		"model":   "fake-4",
		"content": []map[string]any{{"type": "text", "text": "echo:" + prompt}},
	})
	emit(map[string]any{
		"type":        "result",
		"subtype":     "end",
		"duration_ms": 5,
		"num_turns":   1,
		"session_id":  "s4",
		"is_error":    false,
	})
}
