package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/legacy_save"
	"github.com/bradhannah/UltimaVRedux/internal/player_save"
)

var errDestination = errors.New("unsafe save destination")

func prepareDestination(ctx context.Context, parsed options) (destination, string, error) {
	var empty destination
	source, err := legacy_save.ResolveSavedGamePathContext(ctx, parsed.source, "")
	if err != nil {
		return empty, "", fmt.Errorf("resolve --source: %w", err)
	}
	sourceDirectory, err := existingDirectory(filepath.Dir(source))
	if err != nil {
		return empty, "", err
	}
	dataDirectory, err := existingDirectory(parsed.dataDirectory)
	if err != nil {
		return empty, "", err
	}
	target, err := destinationPath(parsed)
	if err != nil {
		return empty, "", err
	}
	if err := checkInputOverlap(target, parsed.slot != "", sourceDirectory, dataDirectory); err != nil {
		return empty, "", err
	}
	inputs, err := selectedInputIdentities(ctx, source)
	if err != nil {
		return empty, "", err
	}
	if err := rejectInputAlias(target, inputs); err != nil {
		return empty, "", err
	}
	reviewed, err := inspectDestination(target, player_save.SlotID(parsed.slot), parsed.overwrite)
	if err != nil {
		return empty, "", err
	}
	reviewed.inputs = inputs

	return reviewed, source, nil
}

func destinationPath(parsed options) (string, error) {
	if parsed.output != "" {
		if err := validateOutputName(filepath.Base(parsed.output)); err != nil {
			return "", err
		}
		parent, err := existingDirectory(filepath.Dir(parsed.output))
		if err != nil {
			return "", err
		}

		return filepath.Join(parent, filepath.Base(parsed.output)), nil
	}
	root, err := canonicalSlotRoot(parsed.saveDirectory)
	if err != nil {
		return "", err
	}

	return filepath.Join(root, parsed.slot+".json"), nil
}

func validateOutputName(name string) error {
	lower := strings.ToLower(name)
	if filepath.Ext(name) != ".json" || lower == "config.json" {
		return fmt.Errorf("%w: --output requires lowercase .json and cannot name a configuration file", errDestination)
	}
	if strings.HasPrefix(lower, "slot-") {
		return fmt.Errorf("%w: reserved slot filenames require --save-dir and --slot", errDestination)
	}

	return nil
}

func existingDirectory(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve directory: %w", err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve directory %q: %w", path, err)
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", fmt.Errorf("inspect directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w: %q is not a directory", errDestination, path)
	}

	return canonical, nil
}

// Only a missing final slot-root component is accepted. Its parent must already
// exist, making path policy independent of future directory creation.
func canonicalSlotRoot(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve --save-dir: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err == nil && !info.IsDir() {
		return "", fmt.Errorf("%w: --save-dir must be a directory, not a symlink or file", errDestination)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect --save-dir: %w", err)
	}
	parent, err := existingDirectory(filepath.Dir(absolute))
	if err != nil {
		return "", err
	}

	return filepath.Join(parent, filepath.Base(absolute)), nil
}

func checkInputOverlap(target string, slot bool, inputDirectories ...string) error {
	for _, input := range inputDirectories {
		contained, err := withinDirectory(input, target)
		if err != nil {
			return err
		}
		if contained {
			return fmt.Errorf("%w: output must be outside the source-save and original-data directories", errDestination)
		}
		if !slot {
			continue
		}
		ancestor, err := withinDirectory(filepath.Dir(target), input)
		if err != nil {
			return err
		}
		if ancestor {
			return fmt.Errorf("%w: --save-dir must not be an ancestor of an input directory", errDestination)
		}
	}

	return nil
}

func withinDirectory(directory, path string) (bool, error) {
	lexicalChild, err := lexicalWithinDirectory(directory, path)
	if err != nil {
		return false, err
	}
	if lexicalChild {
		return true, nil
	}
	ancestor, err := os.Stat(directory)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect protected directory: %w", err)
	}

	return sameDirectoryAncestor(ancestor, path)
}

func lexicalWithinDirectory(directory, path string) (bool, error) {
	relative, err := filepath.Rel(directory, path)
	if err != nil {
		// Rel cannot represent paths on different Windows drives or UNC shares.
		// They are not lexical descendants; filesystem identity must still be checked.
		if filepath.IsAbs(directory) && filepath.IsAbs(path) &&
			!strings.EqualFold(filepath.VolumeName(directory), filepath.VolumeName(path)) {
			return false, nil
		}

		return false, fmt.Errorf("compare destination paths: %w", err)
	}

	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)), nil
}

// EvalSymlinks does not promise canonical casing. Compare directory identities
// along the existing ancestry as well, including when the output does not exist.
func sameDirectoryAncestor(ancestor os.FileInfo, path string) (bool, error) {
	for {
		info, err := os.Stat(path)
		if err == nil && info.IsDir() && ancestor.IsDir() && os.SameFile(ancestor, info) {
			return true, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("inspect destination ancestry: %w", err)
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false, nil
		}
		path = parent
	}
}
