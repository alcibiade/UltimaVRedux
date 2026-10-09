package journal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotebookArchiveReloadRestoreAndPlaythroughIsolation(t *testing.T) {
	repository := NotebookRepository{Root: t.TempDir()}
	original := testExchange("exchange", "All presented speech")
	original.Passages = append(original.Passages, Passage{Text: "SUMM", Runic: true})
	otherID := strings.Repeat("a", 32)
	for _, id := range []string{notebookTestID, otherID} {
		_, err := repository.Keep(id, original)
		require.NoError(t, err)
	}
	_, err := repository.SetArchived(notebookTestID, original.ID, true)
	require.NoError(t, err)
	reopened := NotebookRepository{Root: repository.Root}
	loaded, err := reopened.Load(notebookTestID)
	require.NoError(t, err)
	expected := original.Clone()
	expected.Archived = true
	require.Equal(t, []Exchange{expected}, loaded.Excerpts)
	other, err := reopened.Load(otherID)
	require.NoError(t, err)
	require.Equal(t, []Exchange{original}, other.Excerpts)

	_, err = reopened.SetArchived(notebookTestID, original.ID, false)
	require.NoError(t, err)
	loaded, err = reopened.Load(notebookTestID)
	require.NoError(t, err)
	require.Equal(t, []Exchange{original}, loaded.Excerpts)
	_, err = reopened.SetArchived(notebookTestID, original.ID, true)
	require.NoError(t, err)
	_, err = reopened.Delete(notebookTestID, original.ID)
	require.NoError(t, err)
	loaded, err = reopened.Load(notebookTestID)
	require.NoError(t, err)
	require.Empty(t, loaded.Excerpts)
	// A stale restore action must never recreate a permanently deleted exchange.
	loaded, err = reopened.SetArchived(notebookTestID, original.ID, false)
	require.NoError(t, err)
	require.Empty(t, loaded.Excerpts)
}

func TestNotebookKeepRestoresWithoutShorteningOlderExchange(t *testing.T) {
	repository := NotebookRepository{Root: t.TempDir()}
	older := testExchange("exchange", "Greeting")
	newer := older.Clone()
	newer.Speaker = "Iolo"
	newer.Passages = append(newer.Passages, Passage{Text: " and further speech", Runic: true})
	_, err := repository.Keep(notebookTestID, newer)
	require.NoError(t, err)
	_, err = repository.SetArchived(notebookTestID, newer.ID, true)
	require.NoError(t, err)
	_, err = repository.Keep(notebookTestID, older)
	require.NoError(t, err)
	loaded, err := repository.Load(notebookTestID)
	require.NoError(t, err)
	require.Equal(t, []Exchange{newer}, loaded.Excerpts, "restore must retain deduplication, speech and revealed attribution")
	newer.ID = "new-exchange"
	newer.Archived = true
	loaded, err = repository.Keep(notebookTestID, newer)
	require.NoError(t, err)
	require.False(t, loaded.Excerpts[1].Archived, "deliberately kept exchanges always become active")
}

func TestNotebookArchiveLoadsVersionOneWithoutArchiveField(t *testing.T) {
	repository := NotebookRepository{Root: t.TempDir()}
	path := filepath.Join(repository.Root, notebookTestID+".json")
	require.NoError(t, os.WriteFile(path, []byte(`{"version":1,"excerpts":[{"id":"old","speaker":"Speaker","location":"Britain","input":"WORD","passages":[{"text":"Old notebook","runic":false}]}]}`), 0o600))
	loaded, err := repository.Load(notebookTestID)
	require.NoError(t, err)
	require.False(t, loaded.Excerpts[0].Archived)
	_, err = repository.SetArchived(notebookTestID, "old", true)
	require.NoError(t, err)
	loaded, err = repository.Load(notebookTestID)
	require.NoError(t, err)
	require.True(t, loaded.Excerpts[0].Archived)
	require.Equal(t, "Old notebook", loaded.Excerpts[0].Passages[0].Text)
}

func TestNotebookAttributionPreservesArchiveState(t *testing.T) {
	repository := NotebookRepository{Root: t.TempDir()}
	conversationID := strings.Repeat("b", 32)
	exchange := testExchange(conversationID+"-1", "Presented before the name")
	_, err := repository.Keep(notebookTestID, exchange)
	require.NoError(t, err)
	_, err = repository.SetArchived(notebookTestID, exchange.ID, true)
	require.NoError(t, err)
	_, err = repository.AttributeConversation(notebookTestID, conversationID, "Iolo")
	require.NoError(t, err)
	loaded, err := repository.Load(notebookTestID)
	require.NoError(t, err)
	require.True(t, loaded.Excerpts[0].Archived)
	require.Equal(t, "Iolo", loaded.Excerpts[0].Speaker)
	require.Equal(t, exchange.Passages, loaded.Excerpts[0].Passages)
}

func TestNotebookArchiveRetainsCapacityAndFailedWritesPreserveStorage(t *testing.T) {
	repository := NotebookRepository{Root: t.TempDir()}
	notebook := Notebook{Version: notebookVersion, Excerpts: make([]Exchange, MaxExcerpts)}
	for index := range notebook.Excerpts {
		notebook.Excerpts[index] = testExchange(fmt.Sprint(index), "Kept speech")
	}
	data, err := json.Marshal(notebook)
	require.NoError(t, err)
	path := filepath.Join(repository.Root, notebookTestID+".json")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	_, err = repository.SetArchived(notebookTestID, "0", true)
	require.NoError(t, err)
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	_, err = repository.Keep(notebookTestID, testExchange("overflow", "Extra"))
	require.ErrorIs(t, err, ErrNotebookFull)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after, "archive must not free capacity or allow evicting retained excerpts")
	_, err = repository.SetArchived(notebookTestID, "0", false)
	require.NoError(t, err, "restore must work at capacity")

	require.NoError(t, os.WriteFile(path, []byte("corrupt notebook"), 0o600))
	_, err = repository.SetArchived(notebookTestID, "0", true)
	require.Error(t, err)
	after, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "corrupt notebook", string(after))
}
