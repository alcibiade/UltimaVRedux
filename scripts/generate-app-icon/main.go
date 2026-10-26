// Command generate-app-icon writes the Windows executable icon resources that
// cmd/ultimav links for each supported Windows architecture. The objects are
// committed so go install and release builds need no generator. Regenerate them
// with make generate after changing the icon tile or the sprite atlas.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bradhannah/UltimaVRedux/internal/appicon"
)

const resourceFileMode = 0o644

func main() {
	atlasPath := flag.String("atlas", "internal/sprites/assets/transparent_u5_tiles.png", "sprite atlas PNG")
	outputDir := flag.String("output", "cmd/ultimav", "directory receiving rsrc_windows_<arch>.syso")
	flag.Parse()
	if err := run(*atlasPath, *outputDir); err != nil {
		if _, writeErr := fmt.Fprintln(os.Stderr, err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}

func run(atlasPath, outputDir string) error {
	atlasPNG, err := os.ReadFile(atlasPath) //nolint:gosec // Maintainer-supplied generator input path.
	if err != nil {
		return fmt.Errorf("reading sprite atlas: %w", err)
	}
	for _, target := range appicon.WindowsTargets() {
		object, err := appicon.WindowsResource(atlasPNG, target.Machine)
		if err != nil {
			return fmt.Errorf("building %s icon resource: %w", target.Architecture, err)
		}
		path := filepath.Join(outputDir, target.FileName())
		//nolint:gosec // Maintainer-supplied output directory with a fixed resource file name.
		if err := os.WriteFile(path, object, resourceFileMode); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	return nil
}
