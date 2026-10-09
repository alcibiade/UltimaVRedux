package player_save

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
)

// Corrupt-file fixtures need a directory before they can install raw bytes.
// Normal writes create the directory while acquiring their transaction lock.
func (repository *Repository) ensureRoot() error {
	policy := repositoryfs.Root{Path: repository.root, DirMode: rootMode, FileMode: fileMode}
	if err := repositoryfs.EnsureRoot(policy); err != nil {
		return fmt.Errorf("create player save root: %w", err)
	}

	return nil
}
