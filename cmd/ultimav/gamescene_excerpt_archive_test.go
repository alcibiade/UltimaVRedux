package main

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/stretchr/testify/require"
)

func TestExcerptArchiveRestoresWithCheckpointWithoutChangingGameplay(t *testing.T) {
	source := newJournalSnapshotOriginalScene(t)
	source.setPlaythroughMetadata(playthroughMetadata{id: testPlaythroughID})
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
			if operation == "delete" {
				require.Empty(t, after.Excerpts)
			} else {
				require.Equal(t, operation == "archive", after.Excerpts[0].Archived)
			}
			after.Excerpts = checkpoint.Excerpts
			require.Equal(t, checkpoint, after, "excerpt edits cannot change conversation, knowledge or gameplay")
			restored, loadErr := NewGameSceneFromSaveSnapshot(source.gameConfig, checkpoint)
			require.NoError(t, loadErr)
			t.Cleanup(restored.Dispose)
			restored.setPlaythroughMetadata(playthroughMetadata{id: testPlaythroughID})
			require.Equal(t, checkpoint.Excerpts, restored.excerpts)
			restored.excerpts[0].Passages[0].Text = "Detached branch"
			require.Equal(t, exchange.Passages, checkpoint.Excerpts[0].Passages)
		})
	}
}
