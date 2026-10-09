//go:build darwin

package clipboard

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestReadTextPreservesBytes(t *testing.T) {
	for _, value := range []string{"", "  /Users/Avatar/日本語/é  \r\n", "first\nsecond", "\x00\t\ufeffe\u0301"} {
		t.Run(value, func(t *testing.T) {
			text, err := readText(context.Background(), func(command *exec.Cmd) error {
				if command.Path != "/usr/bin/pbpaste" || !reflect.DeepEqual(command.Args, []string{"/usr/bin/pbpaste", "-Prefer", "txt"}) {
					t.Fatalf("unexpected command: %q %q", command.Path, command.Args)
				}
				if command.Cancel == nil || command.WaitDelay != clipboardWaitDelay {
					t.Fatal("command must bound execution and pipe draining")
				}
				if command.Env[len(command.Env)-1] != "LC_ALL=en_US.UTF-8" {
					t.Fatal("pbpaste must request UTF-8")
				}
				_, err := io.WriteString(command.Stdout, value)
				return err
			})
			if err != nil || text != value {
				t.Fatalf("readText = %q, %v; want %q, nil", text, err, value)
			}
		})
	}
}

func TestReadTextRejectsInvalidUTF8(t *testing.T) {
	text, err := readText(context.Background(), func(command *exec.Cmd) error {
		_, err := command.Stdout.Write([]byte{0xff})
		return err
	})
	if text != "" || !errors.Is(err, errInvalidText) {
		t.Fatalf("readText = %q, %v; want invalid UTF-8 error", text, err)
	}
}

func TestReadTextOutputLimits(t *testing.T) {
	for _, stream := range []string{"stdout", "stderr"} {
		for _, overflow := range []bool{false, true} {
			t.Run(stream+"/overflow="+strconv.FormatBool(overflow), func(t *testing.T) {
				text, err := readText(context.Background(), func(command *exec.Cmd) error {
					writer, limit := command.Stdout, clipboardTextLimit
					if stream == "stderr" {
						writer, limit = command.Stderr, clipboardDiagnosticLimit
					}
					if _, err := io.WriteString(writer, strings.Repeat("x", limit)); err != nil {
						t.Fatalf("exact limit: %v", err)
					}
					if overflow {
						if n, err := io.WriteString(writer, "extra"); n != 0 || !errors.Is(err, errOutputTooLarge) {
							t.Fatalf("overflow write = %d, %v", n, err)
						}
					}
					if len(writer.(*clipboardOutput).data) != limit {
						t.Fatal("output storage must stay bounded")
					}
					return nil
				})
				if overflow {
					if text != "" || !errors.Is(err, errOutputTooLarge) || !strings.Contains(err.Error(), stream) {
						t.Fatalf("readText = %q, %v; want %s overflow error", text, err, stream)
					}
				} else if err != nil {
					t.Fatalf("exact limit: %v", err)
				}
			})
		}
	}
}

func TestReadTextWrapsSessionAndLaunchErrors(t *testing.T) {
	for _, failure := range []error{os.ErrNotExist, os.ErrPermission, &exec.ExitError{}} {
		text, err := readText(context.Background(), func(command *exec.Cmd) error {
			_, writeErr := io.WriteString(command.Stderr, "pasteboard session unavailable\n")
			if writeErr != nil {
				t.Fatal(writeErr)
			}
			return failure
		})
		if text != "" || !errors.Is(err, ErrUnavailable) || !errors.Is(err, failure) ||
			!strings.Contains(err.Error(), "pasteboard session unavailable") {
			t.Fatalf("readText = %q, %v; want wrapped %v", text, err, failure)
		}
	}
}

func TestReadTextTimeout(t *testing.T) {
	// Run only this test binary's inert helper. Never execute pbpaste in tests.
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	text, err := readText(context.Background(), func(command *exec.Cmd) error {
		command.Path = executable
		command.Args = []string{executable, "-test.run=^TestClipboardTimeoutHelper$"}
		command.Env = append(command.Env, "CLIPBOARD_TIMEOUT_HELPER=1")
		return command.Run()
	})
	if text != "" || !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("readText = %q, %v; want wrapped deadline error", text, err)
	}
	if elapsed := time.Since(started); elapsed > clipboardTimeout+5*time.Second {
		t.Fatalf("timeout took %s", elapsed)
	}
}

func TestClipboardTimeoutHelper(t *testing.T) {
	if os.Getenv("CLIPBOARD_TIMEOUT_HELPER") != "1" {
		return
	}
	time.Sleep(time.Minute)
}
