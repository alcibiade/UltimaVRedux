// Command graphics-export decodes original Ultima V graphics into local PNG review files.
package main

import (
	"fmt"
	"os"

	"github.com/bradhannah/UltimaVRedux/internal/graphics/exporter"
	"github.com/spf13/cobra"
)

func newRootCommand() *cobra.Command {
	var options exporter.Options
	command := &cobra.Command{
		Use:           "graphics-export",
		Short:         "Export original Ultima V non-tile graphics to PNG",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			manifest, err := exporter.Export(options)
			if err != nil {
				return fmt.Errorf("exporting graphics: %w", err)
			}
			_, err = fmt.Fprintf(
				os.Stdout,
				"Exported %d graphics archives to %s\n",
				len(manifest.Archives),
				options.OutputDir,
			)
			if err != nil {
				return fmt.Errorf("writing export summary: %w", err)
			}

			return nil
		},
	}
	command.Flags().StringVar(&options.DataDir, "data-dir", "", "Directory containing original Ultima V files")
	command.Flags().StringVar(&options.OutputDir, "output", "", "Directory for generated PNGs and manifest")
	if err := command.MarkFlagRequired("data-dir"); err != nil {
		panic(fmt.Sprintf("configuring graphics data-dir flag: %v", err))
	}
	if err := command.MarkFlagRequired("output"); err != nil {
		panic(fmt.Sprintf("configuring graphics output flag: %v", err))
	}

	return command
}

func main() {
	if err := newRootCommand().Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
