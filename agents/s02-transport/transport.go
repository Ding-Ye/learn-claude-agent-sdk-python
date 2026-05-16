// Package transport spawns a child process and speaks newline-delimited JSON
// over its stdin and stdout. It owns the lifecycle: start, send lines, read
// lines, and graceful close (with a SIGKILL escape hatch).
//
// Upstream reference:
//
//	src/claude_agent_sdk/_internal/transport/subprocess_cli.py
//
// We drop ~700 lines of upstream concerns (env munging, beta headers, max
// buffer policing, atexit child cleanup) and keep the kernel: stdin/stdout
// pipes glued to channels, ctx-cancellable.
package transport

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// Transport is the read/write half-duplex view the rest of the SDK uses.
// In upstream this is the `Transport` abstract base class in
// `_internal/transport/__init__.py`.
type Transport interface {
	Send(ctx context.Context, line string) error
	Recv() (<-chan string, <-chan error)
	Close(ctx context.Context) error
}

// SubprocessTransport boots the binary at Path with the given args. Stdout
// lines are surfaced via Recv(); Send writes a line to stdin.
type SubprocessTransport struct {
	Path string
	Args []string
	Env  []string

	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	out     chan string
	errc    chan error
	started bool
	closed  bool
}

// Start spawns the child. Must be called before Send/Recv.
func (t *SubprocessTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.started {
		return errors.New("transport: already started")
	}
	cmd := exec.CommandContext(ctx, t.Path, t.Args...)
	cmd.Env = t.Env

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("transport: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("transport: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("transport: stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("transport: start: %w", err)
	}
	t.cmd, t.stdin, t.stdout, t.stderr = cmd, stdin, stdout, stderr
	t.out = make(chan string, 64)
	t.errc = make(chan error, 2)
	t.started = true

	go t.pumpStdout()
	go t.drainStderr()
	return nil
}

func (t *SubprocessTransport) pumpStdout() {
	defer close(t.out)
	scanner := bufio.NewScanner(t.stdout)
	// Upstream caps the buffer at 1MB (subprocess_cli.py:_DEFAULT_MAX_BUFFER_SIZE);
	// we do the same.
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		t.out <- scanner.Text()
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		t.errc <- err
	}
}

func (t *SubprocessTransport) drainStderr() {
	_, _ = io.Copy(io.Discard, t.stderr) // tests stream; production code might log
}

// Send writes one newline-terminated line of JSON to the child's stdin.
func (t *SubprocessTransport) Send(_ context.Context, line string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.started || t.closed {
		return errors.New("transport: not running")
	}
	if _, err := io.WriteString(t.stdin, line+"\n"); err != nil {
		return fmt.Errorf("transport: send: %w", err)
	}
	return nil
}

// Recv returns the output channel and an error channel. The output channel is
// closed when the child exits or stdout EOFs.
func (t *SubprocessTransport) Recv() (<-chan string, <-chan error) {
	return t.out, t.errc
}

// Close closes stdin (signaling the child no more prompts are coming), waits
// up to 2s for graceful exit, then kills if necessary. This mirrors
// upstream's two-step shutdown in subprocess_cli.py:close().
func (t *SubprocessTransport) Close(_ context.Context) error {
	t.mu.Lock()
	if t.closed || !t.started {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	_ = t.stdin.Close()
	cmd := t.cmd
	t.mu.Unlock()

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil && !isExpectedExitErr(err) {
			return fmt.Errorf("transport: wait: %w", err)
		}
		return nil
	case <-time.After(2 * time.Second):
		_ = cmd.Process.Kill()
		<-done // drain
		return errors.New("transport: child did not exit in 2s, killed")
	}
}

func isExpectedExitErr(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}
	// SIGPIPE / EOF after we close stdin is normal.
	return exitErr.ExitCode() != 0
}
