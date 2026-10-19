package audiomodulecheck_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGameBuildCannotContainLegacyPlaybackEngines(t *testing.T) {
	root := filepath.Join("..", "..", "internal", "audio")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	retired := map[string]bool{"AudioManager": true, "SoundEffectPlayer": true, "BackgroundMusicSystem": true,
		"AmbientSoundManager": true, "WalkingSoundManager": true, "AudioSourceSet": true, "gainPlayer": true,
		"runtimeStream": true, "LibraryEditor": true, "RecordingAudio": true, "Audition": true, "auditionPlayer": true}
	current := false
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasPrefix(name, "legacy_") {
			t.Errorf("retired audio implementation fixture returned: %s", name)
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if declaration, ok := node.(*ast.TypeSpec); ok {
				if retired[declaration.Name.Name] {
					t.Errorf("retired audio implementation %s returned in %s", declaration.Name.Name, name)
				}
				current = current || declaration.Name.Name == "GameAudioManager"
			}
			return true
		})
	}
	if !current {
		t.Fatal("live game adapter is missing")
	}
}
