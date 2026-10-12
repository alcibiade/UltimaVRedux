package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// selectedInputIdentities follows individual input symlinks just as ReadBundle
// does. Directory overlap alone cannot protect backing files outside that tree.
func selectedInputIdentities(ctx context.Context, source string) ([]os.FileInfo, error) {
	objects, err := selectedObjectsSibling(ctx, filepath.Dir(source))
	if err != nil {
		return nil, err
	}
	paths := []string{source, objects}
	identities := make([]os.FileInfo, 0, len(paths))
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("inspect selected input %q: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%w: selected input %q must resolve to a regular file", errDestination, path)
		}
		identities = append(identities, info)
	}

	return identities, nil
}

// ResolveSavedGamePathContext has already validated the pair. Enumerate again
// to recover the actual OOL spelling, rejecting ambiguity rather than assuming
// uppercase names on case-sensitive hosts. Keep the same bounded directory size.
func selectedObjectsSibling(ctx context.Context, directory string) (string, error) {
	const maximumSaveEntries = 1024
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("resolve selected object list: %w", err)
	}
	//nolint:gosec // Explicit source directory already resolved by the legacy reader.
	handle, err := os.Open(directory)
	if err != nil {
		return "", fmt.Errorf("open selected save directory: %w", err)
	}
	entries, readErr := handle.ReadDir(maximumSaveEntries + 1)
	closeErr := handle.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return "", fmt.Errorf("read selected save directory: %w", err)
	}
	if len(entries) > maximumSaveEntries {
		return "", fmt.Errorf("%w: selected save directory exceeds %d entries", errDestination, maximumSaveEntries)
	}
	var sibling string
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(entry.Name(), "SAVED.OOL") {
			continue
		}
		if sibling != "" {
			return "", fmt.Errorf("%w: ambiguous selected SAVED.OOL sibling", errDestination)
		}
		sibling = filepath.Join(directory, entry.Name())
	}
	if sibling == "" {
		return "", fmt.Errorf("%w: selected SAVED.OOL sibling is missing", errDestination)
	}

	return sibling, nil
}

func rejectInputAlias(path string, inputs []os.FileInfo) error {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect output identity: %w", err)
	}
	for _, input := range inputs {
		if os.SameFile(input, info) {
			return fmt.Errorf("%w: output aliases a selected original save file", errDestination)
		}
	}

	return nil
}
