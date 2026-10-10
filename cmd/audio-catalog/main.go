// Command audio-catalog exports the production inventory and draft recording briefs.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		logging.Error(logging.CategoryAudio, "Audio catalog export: %v", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("audio-catalog", flag.ContinueOnError)
	root := flags.String("source", ".", "Checkout root for static trigger evidence")
	output := flags.String("output", "bin/audio-catalog.json", "Derived inventory JSON output")
	guide := flags.String("guide", "bin/audio-generation-guide.md", "Derived recording brief and inventory guide")
	measure := flags.Bool("measure", false, "Decode full recordings for sample peak/RMS (not LUFS or true peak)")
	if err := flags.Parse(arguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return fmt.Errorf("parse catalog options: %w", err)
	}
	if flags.NArg() != 0 || filepath.Clean(*output) == filepath.Clean(*guide) {
		return fmt.Errorf("%w: require distinct output paths and no positional arguments", audio.ErrAudioCatalog)
	}
	if err := validateExportPaths(*root, *output, *guide); err != nil {
		return err
	}
	catalog, err := audio.BuildCatalog(*measure)
	if err != nil {
		return fmt.Errorf("build audio catalog: %w", err)
	}
	if err := catalog.AttachSourceEvidence(*root); err != nil {
		return fmt.Errorf("attach audio evidence: %w", err)
	}
	encoded, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("encode audio catalog: %w", err)
	}
	const outputMode = 0o600
	if err := os.WriteFile(*output, append(encoded, '\n'), outputMode); err != nil {
		return fmt.Errorf("write audio catalog: %w", err)
	}
	if err := os.WriteFile(*guide, []byte(catalog.GenerationGuide()), outputMode); err != nil {
		return fmt.Errorf("write recording guide: %w", err)
	}
	logging.Info(logging.CategoryAudio, "Exported %d recordings and %d moments to %s and %s",
		len(catalog.Assets), len(catalog.Moments), *output, *guide)

	return nil
}

func validateExportPaths(root, output, guide string) error {
	manifest, err := filepath.Abs(filepath.Join(root, audio.LibraryManifestRelativePath))
	if err != nil {
		return fmt.Errorf("resolve manifest path: %w", err)
	}
	paths := make(map[string]bool)
	for _, destination := range []string{output, guide} {
		absolute, err := filepath.Abs(destination)
		if err != nil {
			return fmt.Errorf("resolve export path: %w", err)
		}
		if absolute == manifest || paths[absolute] {
			return fmt.Errorf("%w: exports must differ from each other and the authored manifest", audio.ErrAudioCatalog)
		}
		paths[absolute] = true
	}

	return nil
}
