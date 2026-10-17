package journal

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const notebookTestID = "0123456789abcdef0123456789abcdef"

func testExchange(id, value string) Exchange {
	return Exchange{ID: id, Speaker: "Speaker", Location: "Britain", Input: "WORD",
		Passages: []Passage{{Text: value, Runic: false}}}
}

func TestLegacyNotebookReadOnlyCompatibility(t *testing.T) {
	r := NotebookRepository{Root: t.TempDir()}
	path := filepath.Join(r.Root, notebookTestID+".json")
	for _, data := range []string{
		`{"version":1,"excerpts":[{"id":"old","speaker":"Speaker","location":"Britain","input":"WORD","passages":[{"text":"Old notebook","runic":false}]}]}`,
		`{"version":1,"excerpts":[{"id":"old","speaker":"Iolo","location":"Britain","input":"WORD","passages":[{"text":"SUMM","runic":true}],"archived":true}]}`,
	} {
		require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
		loaded, err := r.Load(notebookTestID)
		require.NoError(t, err)
		require.Len(t, loaded.Excerpts, 1)
		require.Equal(t, loaded.Excerpts[0].Speaker == "Iolo", loaded.Excerpts[0].Archived)
		after, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, data, string(after))
	}
	for _, data := range []string{"broken", `{"version":2,"excerpts":[]}`, `{"version":1,"excerpts":null}`} {
		require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
		_, err := r.Load(notebookTestID)
		require.Error(t, err)
		after, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		require.Equal(t, data, string(after))
	}
	require.NoError(t, os.Remove(path))
	loaded, err := r.Load(notebookTestID)
	require.NoError(t, err)
	require.Empty(t, loaded.Excerpts)
	_, err = r.Load("../invalid")
	require.ErrorIs(t, err, ErrInvalidNotebook)
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
