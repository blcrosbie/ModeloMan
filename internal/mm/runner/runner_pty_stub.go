//go:build !linux && !darwin

package runner

import (
	"context"
	"io"
)

func runWithPTY(_ context.Context, _ Options, _ io.Writer, _ io.Writer) (int, []Event, error) {
	return -1, []Event{
		newEvent("backend_warn", "pty is not available on this platform build", nil),
	}, errPTYUnsupported
}
