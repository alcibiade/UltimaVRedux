package journal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

const notebookTestID = "0123456789abcdef0123456789abcdef"

func testExchange(id, value string) Exchange {
	return Exchange{ID: id, Speaker: "Speaker", Location: "Britain", Input: "WORD",
		Passages: []Passage{{Text: value, Runic: false}}}
}

func TestNotebookSurvivesOlderCopiesAndSeparatesPlaythroughs(t *testing.T) {
	r := NotebookRepository{Root: t.TempDir()}
	first := testExchange("conversation-1", "First clause")
	_, err := r.Keep(notebookTestID, first)
	require.NoError(t, err)
	extended := first.Clone()
	extended.Passages = append(extended.Passages, Passage{Text: " and revealed continuation", Runic: true})
	_, err = r.Keep(notebookTestID, extended)
	require.NoError(t, err)
	_, err = r.Keep(notebookTestID, first)
	require.NoError(t, err)
	loaded, err := r.Load(notebookTestID)
	require.NoError(t, err)
	require.Equal(t, []Exchange{extended}, loaded.Excerpts)
	other, err := r.Load(strings.Repeat("a", 32))
	require.NoError(t, err)
	require.Empty(t, other.Excerpts)
	_, err = r.Delete(notebookTestID, first.ID)
	require.NoError(t, err)
	loaded, err = r.Load(notebookTestID)
	require.NoError(t, err)
	require.Empty(t, loaded.Excerpts)
}

func TestNotebookKeepsRevealedNameEvenWhenShorterThanSpeakerPlaceholder(t *testing.T) {
	repository := NotebookRepository{Root: t.TempDir()}
	exchange := testExchange("same-exchange", "Some presented speech")
	_, err := repository.Keep(notebookTestID, exchange)
	require.NoError(t, err)
	named := exchange.Clone()
	named.Speaker = "Iolo"
	kept, err := repository.Keep(notebookTestID, named)
	require.NoError(t, err)
	require.Equal(t, "Iolo", kept.Excerpts[0].Speaker)
	kept, err = repository.Keep(notebookTestID, exchange)
	require.NoError(t, err)
	require.Equal(t, "Iolo", kept.Excerpts[0].Speaker)
}

func TestNotebookAttributesOnlyExistingConversationExcerpts(t *testing.T) {
	repository := NotebookRepository{Root: t.TempDir()}
	conversationID := strings.Repeat("b", 32)
	kept := testExchange(conversationID+"-1", "Some presented speech")
	_, err := repository.Keep(notebookTestID, kept)
	require.NoError(t, err)

	notebook, err := repository.AttributeConversation(notebookTestID, conversationID, "Iolo")
	require.NoError(t, err)
	require.Equal(t, "Iolo", notebook.Excerpts[0].Speaker)
	require.Len(t, notebook.Excerpts, 1)

	notebook, err = repository.AttributeConversation(notebookTestID, strings.Repeat("c", 32), "Shamino")
	require.NoError(t, err)
	require.Len(t, notebook.Excerpts, 1, "attribution must not capture an unkept exchange")
}

func TestNotebookAttributionWithoutCaptureDoesNotCreateStorage(t *testing.T) {
	root := t.TempDir()
	repository := NotebookRepository{Root: root}
	notebook, err := repository.AttributeConversation(notebookTestID, strings.Repeat("b", 32), "Iolo")
	require.NoError(t, err)
	require.Empty(t, notebook.Excerpts)
	_, err = os.Stat(filepath.Join(root, notebookTestID+".json"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestNotebookConcurrentCapturesAndCorruption(t *testing.T) {
	r := NotebookRepository{Root: t.TempDir()}
	var group sync.WaitGroup
	errors := make(chan error, 8)
	for i := range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := r.Keep(notebookTestID, testExchange(fmt.Sprint(i), "Presented"))
			errors <- err
		}()
	}
	group.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	loaded, err := r.Load(notebookTestID)
	require.NoError(t, err)
	require.Len(t, loaded.Excerpts, 8)
	path := filepath.Join(r.Root, notebookTestID+".json")
	require.NoError(t, os.WriteFile(path, []byte("broken"), 0o600))
	_, err = r.Keep(notebookTestID, testExchange("new", "No overwrite"))
	require.Error(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "broken", string(data))
}

func TestConversationRetentionAndRunicSearch(t *testing.T) {
	c := Conversation{Exchanges: nil, Truncated: false}
	for i := range MaxExchanges + 1 {
		require.NoError(t, c.Present(testExchange(fmt.Sprint(i), "He said something mysterious.")))
	}
	require.True(t, c.Truncated)
	require.Len(t, c.Exchanges, MaxExchanges)
	require.Equal(t, "1", c.Exchanges[0].ID)
	exchange := testExchange("runes", "Visible speech")
	exchange.Passages = append(exchange.Passages, Passage{Text: "AIPONI", Runic: true})
	require.Contains(t, exchange.SearchText(), "Visible speech")
	require.NotContains(t, exchange.SearchText(), "AIPONI")
	cloned := exchange.Clone()
	cloned.Passages[0].Text = "Changed"
	require.Equal(t, "Visible speech", exchange.Passages[0].Text)
	require.Error(t, c.Present(testExchange("bad", "\x00")))
}
