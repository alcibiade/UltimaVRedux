package audiomodulecheck_test

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBeradAudioModule(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(t.Context(), "python3", filepath.Join(root, "scripts", "audio_module_gate.py"))
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
