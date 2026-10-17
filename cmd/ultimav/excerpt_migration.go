package main

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

// TEMPORARY COMPATIBILITY: Import external notebooks only for native saves
// predating embedded excerpts. Resaving makes the snapshot authoritative;
// current-format saves, including empty collections, never consult the notebook.
// Remove this importer and its legacy path wiring in a future iteration once
// support for external-notebook saves is retired. Never consume the old file:
// other old checkpoints with the same identity still need independent imports.
func (g *Game) prepareExcerptMigration(envelope save_snapshot.Envelope) (save_snapshot.SaveSnapshot, error) {
	snapshot := envelope.Snapshot
	if !envelope.NeedsExcerptMigration() {
		return snapshot, nil
	}
	if g.notebookRoot == "" || !save_snapshot.ValidPlaythroughID(envelope.Metadata.PlaythroughID) {
		logging.Warn(logging.CategoryConversation,
			"Old save has no excerpt migration directory or identity; starting with no excerpts")

		return snapshot, nil
	}
	notebook, err := (journal.NotebookRepository{Root: g.notebookRoot}).Load(envelope.Metadata.PlaythroughID)
	if err != nil {
		return save_snapshot.SaveSnapshot{}, fmt.Errorf("import legacy excerpts (original notebook preserved): %w", err)
	}
	snapshot.Excerpts = journal.Excerpts(notebook.Excerpts).Clone()

	return snapshot, nil
}
