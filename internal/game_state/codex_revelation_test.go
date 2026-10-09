package game_state

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodexJournalTextPreservesEverySourceWord(t *testing.T) {
	var source []string
	for _, paragraph := range CodexRevelationMessages[1:] {
		source = append(source, strings.Join(strings.Fields(paragraph), " "))
	}
	require.Equal(t, strings.Join(source, "\n\n"), strings.ToUpper(codexRevelationText))
}
