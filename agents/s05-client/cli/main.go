// Streaming fake CLI: reads one prompt per line, emits assistant + result
// for each, keeps going until stdin closes.
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

	turn := 0
	emit := func(m map[string]any) {
		b, _ := json.Marshal(m)
		fmt.Println(string(b))
	}

	for scanner.Scan() {
		turn++
		var env map[string]any
		_ = json.Unmarshal(scanner.Bytes(), &env)
		prompt, _ := env["prompt"].(string)

		emit(map[string]any{
			"type":    "assistant",
			"content": []map[string]any{{"type": "text", "text": fmt.Sprintf("turn-%d:%s", turn, prompt)}},
		})
		emit(map[string]any{
			"type":        "result",
			"subtype":     "end",
			"duration_ms": 3,
			"is_error":    false,
		})
	}
}
