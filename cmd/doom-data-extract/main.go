// Command doom-data-extract emits semantic Doom and finale evidence as JSON.
package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bradhannah/UltimaVRedux/internal/originaldata/doom"
	"github.com/spf13/cobra"
)

var errVerificationMismatch = errors.New("verification mismatch")

func newRootCommand(stdout io.Writer) *cobra.Command {
	var dataDir string
	var verifyPath string
	var compact bool
	command := &cobra.Command{ //nolint:exhaustruct // Cobra command hooks are optional.
		Use:           "doom-data-extract",
		Short:         "Extract semantic Doom and finale evidence from original data",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			files, err := readFiles(dataDir)
			if err != nil {
				return err
			}
			report, err := doom.Extract(files)
			if err != nil {
				return fmt.Errorf("extracting Doom data: %w", err)
			}
			var canonical []byte
			if compact || verifyPath != "" {
				canonical, err = doom.CanonicalEvidenceJSON(report)
			} else {
				canonical, err = doom.CanonicalJSON(report)
			}
			if err != nil {
				return fmt.Errorf("encoding extracted Doom data: %w", err)
			}
			if verifyPath != "" {
				return verifyGolden(verifyPath, canonical)
			}
			if _, err := stdout.Write(canonical); err != nil {
				return fmt.Errorf("writing Doom report: %w", err)
			}

			return nil
		},
	}
	command.Flags().StringVar(&dataDir, "data-dir", "", "Directory containing original Ultima V files")
	command.Flags().BoolVar(&compact, "compact", false, "Emit compact canonical compatibility evidence")
	command.Flags().StringVar(&verifyPath, "verify", "", "Compact canonical golden JSON path to verify without writing")
	if err := command.MarkFlagRequired("data-dir"); err != nil {
		panic(fmt.Sprintf("configuring Doom data-dir flag: %v", err))
	}

	return command
}

func readFiles(dataDir string) (doom.Files, error) {
	read := func(name string) ([]byte, error) {
		//nolint:gosec // The CLI intentionally reads named assets under its requested data directory.
		data, err := os.ReadFile(filepath.Join(dataDir, name))
		if err != nil {
			return nil, fmt.Errorf("reading required asset %s: %w", name, err)
		}

		return data, nil
	}

	var files doom.Files
	assets := []struct {
		name string
		dest *[]byte
	}{
		{"DUNGEON.DAT", &files.DungeonDAT}, {"DUNGEON.CBT", &files.DungeonCBT},
		{"MISCMAPS.DAT", &files.MiscMapsDAT}, {"END1.16", &files.End1},
		{"END2.16", &files.End2}, {"ENDSC.16", &files.EndScreen}, {"TEXT.16", &files.Text},
		{"PROPORT.PCS", &files.Proport}, {"END.DAT", &files.EndDAT}, {"ENDMSG.DAT", &files.EndMsgDAT},
	}
	for _, asset := range assets {
		data, err := read(asset.name)
		if err != nil {
			return doom.Files{}, err
		}
		*asset.dest = data
	}

	return files, nil
}

func verifyGolden(path string, actual []byte) error {
	expected, err := os.ReadFile(path) //nolint:gosec // The user explicitly supplies the read-only golden path.
	if err != nil {
		return fmt.Errorf("reading verification golden %s: %w", path, err)
	}
	if bytes.Equal(expected, actual) {
		return nil
	}
	expectedHash := sha256.Sum256(expected)
	actualHash := sha256.Sum256(actual)

	return fmt.Errorf(
		"%w for %s: golden is %d bytes (sha256 %x), extracted report is %d bytes (sha256 %x)",
		errVerificationMismatch, path, len(expected), expectedHash, len(actual), actualHash,
	)
}

func main() {
	if err := newRootCommand(os.Stdout).Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
