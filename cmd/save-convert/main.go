// Command save-convert imports an original save into a canonical native save
// without starting the game or loading a user's configuration.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

var errArguments = errors.New("invalid save-convert arguments")

type options struct {
	dataDirectory string
	source        string
	output        string
	saveDirectory string
	slot          string
	description   string
	overwrite     bool
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		// The process is already failing; failure to report to stderr cannot be recovered.
		_, writeErr := fmt.Fprintf(os.Stderr, "save-convert: %v\n", err)
		if writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}

func parseOptions(arguments []string, diagnostics io.Writer) (options, error) {
	var parsed options
	flags := flag.NewFlagSet("save-convert", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	flags.StringVar(&parsed.dataDirectory, "data-dir", "", "complete original Ultima V installation (required)")
	flags.StringVar(&parsed.source, "source", "", "save folder or SAVED.GAM with exact SAVED.OOL sibling (required)")
	flags.StringVar(&parsed.output, "output", "",
		"standalone file with lowercase .json extension; parent must exist.\n"+
			"Workbench named-library loading requires a lowercase alphanumeric slug with optional internal hyphens, "+
			"e.g. original-save.json")
	flags.StringVar(&parsed.saveDirectory, "save-dir", "", "private player save directory; requires --slot")
	flags.StringVar(&parsed.slot, "slot", "", "player slot: slot-01 through slot-08; requires --save-dir")
	flags.StringVar(&parsed.description, "description", "", "optional single-line player note, at most 80 characters")
	flags.BoolVar(&parsed.overwrite, "overwrite", false, "replace the destination revision inspected before conversion")
	if err := flags.Parse(arguments); err != nil {
		return parsed, fmt.Errorf("parse flags: %w", err)
	}
	if flags.NArg() != 0 {
		return parsed, fmt.Errorf("%w: positional arguments are not accepted", errArguments)
	}
	if err := parsed.validate(); err != nil {
		return parsed, err
	}

	return parsed, nil
}

func (parsed options) validate() error {
	if strings.TrimSpace(parsed.dataDirectory) == "" || strings.TrimSpace(parsed.source) == "" {
		return fmt.Errorf("%w: --data-dir and --source are required", errArguments)
	}
	if strings.HasPrefix(strings.TrimSpace(parsed.source), "~") {
		return fmt.Errorf("%w: expand --source to an explicit path before invoking save-convert", errArguments)
	}
	if parsed.output != "" {
		if parsed.saveDirectory != "" || parsed.slot != "" {
			return fmt.Errorf("%w: --output is mutually exclusive with --save-dir and --slot", errArguments)
		}
	} else {
		if strings.TrimSpace(parsed.saveDirectory) == "" || parsed.slot == "" {
			return fmt.Errorf("%w: use --output or both --save-dir and --slot", errArguments)
		}
		if _, err := player_save.ParseSlotID(parsed.slot); err != nil {
			return fmt.Errorf("validate --slot: %w", err)
		}
	}
	if err := save_snapshot.ValidateUserDescription(parsed.description); err != nil {
		return fmt.Errorf("validate --description: %w", err)
	}

	return nil
}

func run(ctx context.Context, arguments []string, output, diagnostics io.Writer) error {
	parsed, err := parseOptions(arguments, diagnostics)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	destination, source, err := prepareDestination(ctx, parsed)
	if err != nil {
		return err
	}
	envelope, warnings, err := convert(ctx, source, parsed, time.Now, newPlaythroughID)
	if err != nil {
		return err
	}
	for _, warning := range warnings {
		_, err := fmt.Fprintf(diagnostics, "warning: %s [%s] %s\n", warning.Path, warning.Code, warning.Message)
		if err != nil {
			return fmt.Errorf("report conversion warning: %w", err)
		}
	}
	if err := destination.publish(ctx, envelope); err != nil {
		return err
	}

	return writeReceipt(output, destination.path, envelope, len(warnings))
}

func writeReceipt(output io.Writer, path string, envelope save_snapshot.Envelope, warningCount int) error {
	_, err := fmt.Fprintf(output,
		"Converted save: %q\nPlaythrough ID: %s\nFormat: %s (envelope %d, snapshot %d)\nWarnings: %d\n",
		path, envelope.Metadata.PlaythroughID, envelope.Format,
		envelope.EnvelopeVersion, envelope.Snapshot.Version, warningCount,
	)
	if err != nil {
		return fmt.Errorf("output was published, but reporting success failed: %w", err)
	}

	return nil
}

func buildVersion() string {
	info, available := debug.ReadBuildInfo()
	if available && info.Main.Version != "" {
		return info.Main.Version
	}

	return "(devel)"
}
