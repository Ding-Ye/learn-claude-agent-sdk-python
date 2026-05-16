// Command fake-cli is a stand-in for the real `claude` binary so transport
// tests don't need network or model access. Behaviour:
//
//   1. Read one prompt line from stdin.
//   2. Emit 3 JSON messages: system init, assistant text, result.
//
// Each test build of s02 invokes `go run ./cli` to spawn this.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "fake-cli: no prompt received")
		os.Exit(2)
	}
	prompt := scanner.Text()

	emit := func(m map[string]any) {
		b, _ := json.Marshal(m)
		fmt.Println(string(b))
	}

	emit(map[string]any{"type": "system", "subtype": "init", "session_id": "sess-1"})
	emit(map[string]any{
		"type":    "assistant",
		"model":   "fake-1",
		"content": []map[string]any{{"type": "text", "text": "echo: " + prompt}},
	})
	// small delay so the channel ordering is exercised
	time.Sleep(5 * time.Millisecond)
	emit(map[string]any{
		"type":        "result",
		"subtype":     "end",
		"duration_ms": 12,
		"num_turns":   1,
		"session_id":  "sess-1",
		"is_error":    false,
	})
}
