package client

import (
	"context"
	"testing"
	"time"
)

func collectOneTurn(t *testing.T, c *Client) (assistant TextBlock, result ResultMessage) {
	t.Helper()
	deadline := time.NewTimer(15 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case m, ok := <-c.Recv():
			if !ok {
				t.Fatal("recv channel closed early")
			}
			switch v := m.(type) {
			case AssistantMessage:
				if len(v.Content) > 0 {
					assistant = v.Content[0].(TextBlock)
				}
			case ResultMessage:
				result = v
				return
			}
		case <-deadline.C:
			t.Fatal("turn timeout")
		}
	}
}

func TestClientMultipleTurns(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c := New(Options{CLIPath: "go", CLIArgs: []string{"run", "./cli"}})
	if err := c.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer c.Close()

	if err := c.Send("alpha"); err != nil {
		t.Fatal(err)
	}
	tb, rm := collectOneTurn(t, c)
	if tb.Text != "turn-1:alpha" || rm.Subtype != "end" {
		t.Fatalf("bad turn 1: %q %+v", tb.Text, rm)
	}

	if err := c.Send("beta"); err != nil {
		t.Fatal(err)
	}
	tb, rm = collectOneTurn(t, c)
	if tb.Text != "turn-2:beta" {
		t.Fatalf("bad turn 2: %q", tb.Text)
	}
}

func TestSendBeforeConnect(t *testing.T) {
	c := New(Options{CLIPath: "go", CLIArgs: []string{"run", "./cli"}})
	if err := c.Send("x"); err == nil {
		t.Fatal("expected error before Connect")
	}
}
