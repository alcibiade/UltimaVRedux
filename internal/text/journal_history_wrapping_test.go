package text

import (
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestWrapJournalHistoryMessageSpansAndSourceOffsets(t *testing.T) {
	font := NewUltimaFont(16)
	history := "  One complete message with authored spaces. \n\nrepeat\nrepeat\n" + strings.Repeat("\u00e9", 40)
	for _, width := range []int{1, 20, 80, 200, 1000} {
		rows := WrapJournalHistory(history, font, width)
		messages := make([]string, 4)
		blanks := 0
		for _, row := range rows {
			if row.Message == -1 {
				blanks++
				require.Empty(t, row.Text)
				continue
			}
			require.Equal(t, len(messages[row.Message]), row.SourceOffset)
			require.True(t, utf8.ValidString(row.Text))
			measured, _ := font.MeasureString(row.Text)
			if utf8.RuneCountInString(row.Text) > 1 {
				require.LessOrEqual(t, measured, float64(width))
			}
			messages[row.Message] += row.Text
		}
		require.Equal(t, 1, blanks)
		require.Equal(t, []string{"  One complete message with authored spaces. ", "repeat", "repeat", strings.Repeat("\u00e9", 40)}, messages)
	}
	require.Empty(t, WrapJournalHistory("", font, 100))
	require.Panics(t, func() { WrapJournalHistory("A", nil, 100) })
	require.Panics(t, func() { WrapJournalHistory("A", font, 0) })
}

func TestWrapJournalHistoryUsesActualWidthWithoutHyphens(t *testing.T) {
	font := NewUltimaFont(16)
	message := "abcdeabcdeabcdeabcde"
	measured, _ := font.MeasureString(message)
	rows := WrapJournalHistory(message, font, int(math.Ceil(measured)))
	require.Equal(t, []JournalHistoryRow{{Message: 0, SourceOffset: 0, Text: message}}, rows)
	rows = WrapJournalHistory(message, font, int(math.Ceil(measured/3)))
	var joined strings.Builder
	for _, row := range rows {
		require.Equal(t, 0, row.Message)
		joined.WriteString(row.Text)
	}
	require.Equal(t, message, joined.String())
	require.GreaterOrEqual(t, len(rows), 3)
}
