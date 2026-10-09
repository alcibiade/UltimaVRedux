package originaldata

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

// TestProcess runs a negative fixture in this test binary, so Fatal and Skip can
// be distinguished without failing or skipping the parent regression test.
// Callers supply a private working directory and explicit environment overrides.
func TestProcess(t *testing.T, name, directory string, environment ...string) ([]byte, error) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	//nolint:gosec // Runs only this test binary with a literal test name, never a shell.
	command := exec.CommandContext(t.Context(), executable, "-test.run=^"+name+"$", "-test.v")
	command.Dir = directory
	command.Env = append(os.Environ(), environment...)

	output, err := command.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("running test fixture %s: %w", name, err)
	}

	return output, nil
}
