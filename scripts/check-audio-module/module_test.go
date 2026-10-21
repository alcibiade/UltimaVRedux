package audiomodulecheck_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBeradAudioModule(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	python := filepath.Join(root, ".cache", "python-quality", "bin", "python")
	if runtime.GOOS == "windows" {
		python = filepath.Join(root, ".cache", "python-quality", "Scripts", "python.exe")
	}
	if info, statErr := os.Stat(python); statErr != nil || !info.Mode().IsRegular() {
		t.Fatalf("Python quality interpreter unavailable at %s; run make python-tools PYTHON_BOOTSTRAP=python3.12 (or another Python 3.12+ interpreter)", python)
	}
	ready := exec.CommandContext(t.Context(), python, "-I", "-B", filepath.Join(root, "scripts", "python_quality.py"), "ready")
	ready.Dir = root
	if output, readyErr := ready.CombinedOutput(); readyErr != nil {
		t.Fatalf("Python quality environment is not ready: %v\n%s", readyErr, output)
	}
	command := exec.CommandContext(t.Context(), python, "-B", filepath.Join(root, "scripts", "audio_module_gate.py"))
	if moduleRace {
		command.Args = append(command.Args, "--race")
	}
	command.Dir = root
	output, err := command.CombinedOutput()
	t.Logf("%s", output)
	if err != nil {
		t.Fatal(err)
	}
}
