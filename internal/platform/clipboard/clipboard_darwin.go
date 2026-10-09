//go:build darwin

package clipboard

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
	"unicode/utf8"
)

const (
	clipboardTimeout         = 2 * time.Second
	clipboardWaitDelay       = 100 * time.Millisecond
	clipboardTextLimit       = 1 << 20
	clipboardDiagnosticLimit = 4 << 10
)

var (
	errInvalidText    = errors.New("clipboard text data is not valid UTF-8")
	errOutputTooLarge = errors.New("clipboard command output exceeds limit")
)

// ReadText returns plain text without trimming or normalizing it. An empty
// result is valid: pbpaste cannot distinguish empty text from no text flavor.
func ReadText() (string, error) {
	return readText(context.Background(), (*exec.Cmd).Run)
}

// Supported reports build support without probing the clipboard or user session.
// Actual availability is determined only when ReadText is requested.
func Supported() bool {
	return true
}

// The runner is local to a read so tests can exercise failures without touching
// the user's clipboard or replacing process-wide state.
func readText(parent context.Context, run func(*exec.Cmd) error) (string, error) {
	ctx, cancel := context.WithTimeout(parent, clipboardTimeout)
	defer cancel()

	stdout := clipboardOutput{data: nil, limit: clipboardTextLimit, exceeded: false, cancel: cancel}
	stderr := clipboardOutput{data: nil, limit: clipboardDiagnosticLimit, exceeded: false, cancel: cancel}
	command := exec.CommandContext(ctx, "/usr/bin/pbpaste", "-Prefer", "txt")
	// pbpaste selects its encoding from the locale. Request UTF-8 even when
	// launched from a terminal with a non-UTF-8 locale.
	command.Env = append(command.Environ(), "LC_ALL=en_US.UTF-8")
	command.Stdout = &stdout
	command.Stderr = &stderr
	// Bound pipe draining as well as process execution if a child holds a pipe.
	command.WaitDelay = clipboardWaitDelay
	err := run(command)
	if stdout.exceeded {
		return "", fmt.Errorf("read clipboard text: pbpaste stdout (limit %d bytes): %w", stdout.limit, errOutputTooLarge)
	}
	if stderr.exceeded {
		return "", fmt.Errorf("read clipboard text: pbpaste stderr (limit %d bytes): %w", stderr.limit, errOutputTooLarge)
	}
	if ctx.Err() != nil {
		return "", fmt.Errorf("read clipboard text: pbpaste: %w: %w", ErrUnavailable, ctx.Err())
	}
	if err != nil {
		return "", fmt.Errorf("read clipboard text: pbpaste: %w: %w (stderr: %q)", ErrUnavailable, err, stderr.data)
	}
	if !utf8.Valid(stdout.data) {
		return "", fmt.Errorf("read clipboard text: pbpaste: %w", errInvalidText)
	}

	return string(stdout.data), nil
}

// Each stream has its own writer; os/exec waits for its copy goroutines before
// returning. Overflow cancels the command and never returns truncated text.
type clipboardOutput struct {
	data     []byte
	limit    int
	exceeded bool
	cancel   context.CancelFunc
}

func (output *clipboardOutput) Write(data []byte) (int, error) {
	written := min(len(data), output.limit-len(output.data))
	output.data = append(output.data, data[:written]...)
	if written < len(data) {
		output.exceeded = true
		output.cancel()

		return written, errOutputTooLarge
	}

	return written, nil
}
