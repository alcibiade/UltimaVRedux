// Package texteditor requests the host text editor without invoking a shell.
package texteditor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

var (
	errPathNotAbsolute = errors.New("text editor path must be absolute")
	errUnsupportedOS   = errors.New("text editor is unsupported on this operating system")
)

// Launcher opens one exact settings file in the host text editor.
type Launcher interface {
	Open(path string) (<-chan error, error)
}

// SystemLauncher launches the host editor process and reaps it asynchronously.
type SystemLauncher struct{}

// Open starts the platform editor. Start failures are returned immediately;
// process completion is delivered on a buffered channel without blocking the game.
func (SystemLauncher) Open(path string) (<-chan error, error) {
	if !filepath.IsAbs(path) {
		return nil, errPathNotAbsolute
	}
	name, arguments, err := Command(runtime.GOOS, path)
	if err != nil {
		return nil, err
	}
	//nolint:gosec // Fixed executable and argument array; no shell is involved.
	command := exec.CommandContext(context.Background(), name, arguments...)
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("starting %s: %w", name, err)
	}
	done := make(chan error, 1)
	go func() {
		done <- command.Wait()
		close(done)
	}()

	return done, nil
}

// Command returns the exact executable and argument array for a supported host.
func Command(goos, path string) (string, []string, error) {
	switch goos {
	case "windows":
		return "notepad.exe", []string{path}, nil
	case "darwin":
		return "open", []string{"-t", path}, nil
	case "linux":
		return "xdg-open", []string{path}, nil
	default:
		return "", nil, fmt.Errorf("%w: %s", errUnsupportedOS, goos)
	}
}
