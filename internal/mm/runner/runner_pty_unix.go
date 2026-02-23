//go:build linux || darwin

package runner

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/creack/pty"
)

func runWithPTY(ctx context.Context, opts Options, transcript io.Writer, stream io.Writer) (int, []Event, error) {
	cmd := exec.CommandContext(ctx, opts.Backend)
	cmd.Dir = opts.RepoDir

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return -1, []Event{
			newEvent("backend_error", "failed to start backend with pty", map[string]any{
				"error": err.Error(),
			}),
		}, err
	}
	defer func() {
		_ = ptmx.Close()
	}()

	events := []Event{
		newEvent("backend_started", "backend started via pty mode", map[string]any{
			"backend": opts.Backend,
			"mode":    "pty",
		}),
	}

	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		writer := stream
		if writer == nil {
			writer = os.Stdout
		}
		if opts.CaptureTranscript {
			writer = io.MultiWriter(writer, transcript)
		}
		_, _ = io.Copy(writer, ptmx)
	}()

	ioCtx, cancelIO := context.WithCancel(ctx)
	defer cancelIO()
	inputDone := make(chan struct{}, 1)
	if opts.ForwardInput {
		in := opts.InputReader
		if in == nil {
			in = os.Stdin
		}
		go forwardInput(ioCtx, in, ptmx, inputDone)
	} else {
		inputDone <- struct{}{}
	}

	if prompt := strings.TrimSpace(opts.Prompt); prompt != "" {
		if _, writeErr := io.WriteString(ptmx, prompt+"\n"); writeErr == nil {
			events = append(events, newEvent("prompt_injected", "prompt sent to backend pty", map[string]any{
				"bytes": len(prompt),
			}))
		}
	}

	waitErr := cmd.Wait()
	code := exitCode(cmd, waitErr)
	cancelIO()
	_ = ptmx.Close()

	select {
	case <-inputDone:
	case <-time.After(400 * time.Millisecond):
	}
	select {
	case <-readerDone:
	case <-time.After(400 * time.Millisecond):
	}

	events = append(events, newEvent("backend_ended", "backend process exited", map[string]any{
		"exit_code": code,
	}))
	return code, events, waitErr
}

func forwardInput(_ context.Context, in io.Reader, out io.Writer, done chan<- struct{}) {
	defer func() {
		select {
		case done <- struct{}{}:
		default:
		}
	}()
	_, _ = io.Copy(out, in)
}
