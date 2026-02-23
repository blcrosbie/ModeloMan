package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	errPTYUnsupported = errors.New("pty execution is not supported on this platform")
)

type Event struct {
	Type    string         `json:"type"`
	At      string         `json:"at"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

type Result struct {
	ExitCode            int
	StartedAt           time.Time
	EndedAt             time.Time
	Duration            time.Duration
	Err                 error
	Events              []Event
	Transcript          string
	TranscriptTruncated bool
}

type Options struct {
	Backend            string
	RepoDir            string
	Prompt             string
	UsePTY             bool
	CaptureTranscript  bool
	MaxTranscriptBytes int
	ForwardInput       bool
	InputReader        io.Reader
	OutputWriter       io.Writer
	OnOutput           func(string)
	OnEvent            func(Event)
}

func Run(ctx context.Context, opts Options) Result {
	start := time.Now().UTC()
	result := Result{
		ExitCode:  -1,
		StartedAt: start,
	}

	backend := strings.TrimSpace(opts.Backend)
	if backend == "" {
		result.Err = fmt.Errorf("backend is required")
		event := newEvent("backend_error", "backend command is empty", nil)
		result.Events = append(result.Events, event)
		dispatchEvent(opts.OnEvent, event)
		result.EndedAt = time.Now().UTC()
		result.Duration = result.EndedAt.Sub(result.StartedAt)
		return result
	}
	if opts.MaxTranscriptBytes <= 0 {
		opts.MaxTranscriptBytes = 200000
	}
	if opts.OutputWriter == nil {
		opts.OutputWriter = os.Stdout
	}

	transcript := newCappedBuffer(opts.MaxTranscriptBytes)
	var code int
	var err error
	stream := newChunkWriter(opts.OutputWriter, opts.OnOutput)
	if opts.UsePTY {
		var ptyEvents []Event
		code, ptyEvents, err = runWithPTY(ctx, opts, transcript, stream)
		result.Events = append(result.Events, ptyEvents...)
		dispatchEvents(opts.OnEvent, ptyEvents)
		if err == nil || !errors.Is(err, errPTYUnsupported) {
			result.ExitCode = code
			result.Err = err
			result.Transcript = transcript.String()
			result.TranscriptTruncated = transcript.Truncated()
			result.EndedAt = time.Now().UTC()
			result.Duration = result.EndedAt.Sub(result.StartedAt)
			return result
		}
		fallbackEvent := newEvent("backend_warn", "pty unavailable, falling back to stdin injection", map[string]any{
			"error": err.Error(),
		})
		result.Events = append(result.Events, fallbackEvent)
		dispatchEvent(opts.OnEvent, fallbackEvent)
		if opts.ForwardInput {
			attachEvent := newEvent("backend_warn", "switching to attached terminal mode due to missing pty support", nil)
			result.Events = append(result.Events, attachEvent)
			dispatchEvent(opts.OnEvent, attachEvent)

			var attachedEvents []Event
			code, attachedEvents, err = runAttached(ctx, opts, stream)
			result.Events = append(result.Events, attachedEvents...)
			dispatchEvents(opts.OnEvent, attachedEvents)
			result.ExitCode = code
			result.Err = err
			result.Transcript = transcript.String()
			result.TranscriptTruncated = transcript.Truncated()
			result.EndedAt = time.Now().UTC()
			result.Duration = result.EndedAt.Sub(result.StartedAt)
			return result
		}
	}

	var injectedEvents []Event
	code, injectedEvents, err = runInjected(ctx, opts, transcript, stream)
	result.Events = append(result.Events, injectedEvents...)
	dispatchEvents(opts.OnEvent, injectedEvents)

	// Fallback for interactive-only tools that still require direct terminal attachment.
	if isLikelyTTYError(err) || transcriptSuggestsTTYIssue(transcript.String()) {
		var attachedEvents []Event
		code, attachedEvents, err = runAttached(ctx, opts, stream)
		result.Events = append(result.Events, attachedEvents...)
		dispatchEvents(opts.OnEvent, attachedEvents)
		result.ExitCode = code
		result.Err = err
		result.Transcript = transcript.String()
		result.TranscriptTruncated = transcript.Truncated()
		result.EndedAt = time.Now().UTC()
		result.Duration = result.EndedAt.Sub(result.StartedAt)
		return result
	}

	result.ExitCode = code
	result.Err = err
	result.Transcript = transcript.String()
	result.TranscriptTruncated = transcript.Truncated()
	result.EndedAt = time.Now().UTC()
	result.Duration = result.EndedAt.Sub(result.StartedAt)
	return result
}

func runInjected(ctx context.Context, opts Options, transcript io.Writer, stream io.Writer) (int, []Event, error) {
	cmd := exec.CommandContext(ctx, opts.Backend)
	cmd.Dir = opts.RepoDir
	cmd.Stdout = io.MultiWriter(stream, transcript)
	cmd.Stderr = io.MultiWriter(stream, transcript)

	events := []Event{
		newEvent("backend_started", "backend started via stdin-injection mode", map[string]any{
			"backend": opts.Backend,
			"mode":    "stdin",
		}),
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		events = append(events, newEvent("backend_error", "failed to open stdin pipe", map[string]any{"error": err.Error()}))
		return -1, events, err
	}
	if err := cmd.Start(); err != nil {
		events = append(events, newEvent("backend_error", "backend failed to start", map[string]any{"error": err.Error()}))
		return -1, events, err
	}

	if prompt := strings.TrimSpace(opts.Prompt); prompt != "" {
		if _, writeErr := io.WriteString(stdin, prompt+"\n"); writeErr == nil {
			events = append(events, newEvent("prompt_injected", "prompt sent to backend stdin", map[string]any{
				"bytes": len(prompt),
			}))
		}
	}
	if opts.ForwardInput {
		in := opts.InputReader
		if in == nil {
			in = os.Stdin
		}
		go func() {
			defer func() { _ = stdin.Close() }()
			_, _ = io.Copy(stdin, in)
		}()
	} else {
		_ = stdin.Close()
	}

	err = cmd.Wait()
	code := exitCode(cmd, err)
	events = append(events, newEvent("backend_ended", "backend process exited", map[string]any{
		"exit_code": code,
	}))
	return code, events, err
}

func runAttached(ctx context.Context, opts Options, stream io.Writer) (int, []Event, error) {
	cmd := exec.CommandContext(ctx, opts.Backend)
	cmd.Dir = opts.RepoDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = stream
	cmd.Stderr = stream

	events := []Event{
		newEvent("backend_started", "backend started in attached mode", map[string]any{
			"backend": opts.Backend,
			"mode":    "attached",
		}),
	}
	err := cmd.Run()
	code := exitCode(cmd, err)
	events = append(events, newEvent("backend_ended", "backend process exited", map[string]any{
		"exit_code": code,
	}))
	return code, events, err
}

func exitCode(cmd *exec.Cmd, err error) int {
	if cmd != nil && cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	var exitErr *exec.ExitError
	if err != nil && strings.Contains(err.Error(), "executable file not found") {
		return 127
	}
	if err != nil && errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func isLikelyTTYError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "tty") || strings.Contains(msg, "inappropriate ioctl") || strings.Contains(msg, "not a terminal")
}

func transcriptSuggestsTTYIssue(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "stdin is not a terminal") ||
		strings.Contains(lower, "not a tty") ||
		strings.Contains(lower, "requires a terminal")
}

func newEvent(eventType, message string, data map[string]any) Event {
	cloned := map[string]any(nil)
	if data != nil {
		cloned = maps.Clone(data)
	}
	return Event{
		Type:    eventType,
		At:      time.Now().UTC().Format(time.RFC3339Nano),
		Message: message,
		Data:    cloned,
	}
}

func dispatchEvents(handler func(Event), events []Event) {
	for _, event := range events {
		dispatchEvent(handler, event)
	}
}

func dispatchEvent(handler func(Event), event Event) {
	if handler != nil {
		handler(event)
	}
}

func newChunkWriter(base io.Writer, handler func(string)) io.Writer {
	if base == nil && handler == nil {
		return io.Discard
	}
	if base == nil {
		base = io.Discard
	}
	if handler == nil {
		return base
	}
	return io.MultiWriter(base, chunkCallbackWriter(handler))
}

type chunkCallbackWriter func(string)

func (c chunkCallbackWriter) Write(p []byte) (int, error) {
	if c != nil && len(p) > 0 {
		c(string(p))
	}
	return len(p), nil
}

type cappedBuffer struct {
	max       int
	buf       bytes.Buffer
	truncated bool
	mu        sync.Mutex
}

func newCappedBuffer(max int) *cappedBuffer {
	if max <= 0 {
		max = 200000
	}
	return &cappedBuffer{max: max}
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.max <= 0 {
		return len(p), nil
	}
	remaining := c.max - c.buf.Len()
	if remaining <= 0 {
		c.truncated = true
		return len(p), nil
	}
	if len(p) <= remaining {
		_, _ = c.buf.Write(p)
		return len(p), nil
	}
	_, _ = c.buf.Write(p[:remaining])
	c.truncated = true
	return len(p), nil
}

func (c *cappedBuffer) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

func (c *cappedBuffer) Truncated() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.truncated
}
