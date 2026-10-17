package journal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExcerptsBranchesArchiveDeleteAndAttribution(t *testing.T) {
	conversationID := strings.Repeat("b", 32)
	original := testExchange(conversationID+"-1", "All presented speech")
	original.Passages = append(original.Passages, Passage{Text: "SUMM", Runic: true})
	var first Excerpts
	require.NoError(t, first.Keep(original))
	second := first.Clone()
	second.SetArchived(original.ID, true)
	require.NoError(t, second.AttributeConversation(conversationID, "Iolo"))
	require.True(t, second[0].Archived)
	require.Equal(t, "Iolo", second[0].Speaker)
	require.Equal(t, Excerpts{original}, first)
	second[0].Passages[0].Text = "Changed branch"
	require.Equal(t, original.Passages, first[0].Passages)
	second.Delete(original.ID)
	second.SetArchived(original.ID, false)
	require.Empty(t, second, "stale restore must not resurrect a deletion")
	require.NoError(t, second.AttributeConversation(conversationID, "Iolo"))
	require.Empty(t, second, "attribution cannot capture an unkept exchange")
}

func TestExcerptsKeepRestoresWithoutShorteningSpeechOrLosingName(t *testing.T) {
	older := testExchange("exchange", "Greeting")
	newer := older.Clone()
	newer.Speaker = "Iolo"
	newer.Passages = append(newer.Passages, Passage{Text: " and further speech", Runic: true})
	var excerpts Excerpts
	require.NoError(t, excerpts.Keep(newer))
	excerpts.SetArchived(newer.ID, true)
	require.NoError(t, excerpts.Keep(older))
	require.Equal(t, Excerpts{newer}, excerpts)
	newer.ID, newer.Archived = "another", true
	require.NoError(t, excerpts.Keep(newer))
	require.False(t, excerpts[1].Archived)
	newer.Passages[0].Text = "Mutated caller"
	require.Equal(t, "Greeting", excerpts[1].Passages[0].Text)
}

func TestExcerptsLimitsAreTransactionalAndArchiveRetainsCapacity(t *testing.T) {
	excerpts := make(Excerpts, MaxExcerpts)
	for i := range excerpts {
		excerpts[i] = testExchange(fmt.Sprint(i), "Kept speech")
	}
	excerpts.SetArchived("0", true)
	before := excerpts.Clone()
	require.ErrorIs(t, excerpts.Keep(testExchange("overflow", "Extra")), ErrNotebookFull)
	require.Equal(t, before, excerpts)
	excerpts.SetArchived("0", false)
	require.False(t, excerpts[0].Archived)
	require.Error(t, excerpts.Keep(testExchange("bad", "\x00")))
	require.Len(t, excerpts, MaxExcerpts)
	excerpts.Delete("0")
	require.NoError(t, excerpts.Keep(testExchange("new", "Fits now")))
}
