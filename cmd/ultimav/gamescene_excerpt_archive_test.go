package main

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/stretchr/testify/require"
)

func TestExcerptArchiveSurvivesOlderCheckpointWithoutChangingGameplay(t *testing.T) {
	source := newJournalSnapshotOriginalScene(t)
	source.setPlaythroughMetadata(playthroughMetadata{id: testPlaythroughID})
	source.notebookRoot = t.TempDir()
	exchange := journal.Exchange{ID: "archive-checkpoint-1", Speaker: "Iolo", Location: "Britain", Input: "JOB",
		Passages: []journal.Passage{{Text: "Captured speech", Runic: false}}}
	require.NoError(t, source.lastConversation.Present(exchange))
	_, err := source.keepExcerpt(exchange)
	require.NoError(t, err)
	checkpoint, err := source.CaptureSaveSnapshot()
	require.NoError(t, err)

	for _, operation := range []string{"archive", "restore", "delete"} {
		t.Run(operation, func(t *testing.T) {
			if operation == "delete" {
				_, err = source.deleteExcerpt(exchange.ID)
			} else {
				_, err = source.archiveExcerpt(exchange.ID, operation == "archive")
			}
			require.NoError(t, err)
			after, captureErr := source.CaptureSaveSnapshot()
			require.NoError(t, captureErr)
			require.Equal(t, checkpoint, after, "notebook operations cannot change conversation, learned facts or any checkpoint state")

			restored, loadErr := NewGameSceneFromSaveSnapshot(source.gameConfig, checkpoint)
			require.NoError(t, loadErr)
			t.Cleanup(restored.Dispose)
			restored.notebookRoot = source.notebookRoot
			restored.setPlaythroughMetadata(playthroughMetadata{id: testPlaythroughID})
			excerpts, loadErr := restored.loadExcerpts()
			require.NoError(t, loadErr)
			if operation == "delete" {
				require.Empty(t, excerpts)
			} else {
				require.Len(t, excerpts, 1)
				require.Equal(t, operation == "archive", excerpts[0].Archived)
				require.Equal(t, exchange.Passages, excerpts[0].Passages)
			}
			restored.setPlaythroughMetadata(playthroughMetadata{id: otherTestPlaythroughID})
			excerpts, loadErr = restored.loadExcerpts()
			require.NoError(t, loadErr)
			require.Empty(t, excerpts)
		})
	}
}
