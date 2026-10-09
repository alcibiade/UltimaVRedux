package text

import (
	"strings"
	"unicode/utf8"
)

// JournalHistoryRow maps one measured display row to a logical message and its
// UTF-8 byte offset. Explicit blank separators have Message -1.
type JournalHistoryRow struct {
	Message      int
	SourceOffset int
	Text         string
}

// WrapJournalHistory reflows the detached source, never the rail's hyphenated
// rows. A single glyph wider than width is retained alone so reading progresses.
func WrapJournalHistory(history string, font *UltimaFont, width int) []JournalHistoryRow {
	if width <= 0 || font == nil || font.TextFace == nil || font.TextFace.Source == nil {
		panic("journal history wrapping requires positive width and an initialized font")
	}
	var rows []JournalHistoryRow
	if history == "" {
		return rows
	}
	message := 0
	for line := range strings.SplitSeq(history, "\n") {
		if line == "" {
			rows = append(rows, JournalHistoryRow{Message: -1, SourceOffset: 0, Text: ""})

			continue
		}
		for offset := 0; offset < len(line); {
			end := journalWrapEnd(line[offset:], font, width) + offset
			rows = append(rows, JournalHistoryRow{Message: message, SourceOffset: offset, Text: line[offset:end]})
			offset = end
		}
		message++
	}

	return rows
}

func journalWrapEnd(line string, font *UltimaFont, width int) int {
	// Binary search rune boundaries using the actual shaped string width, not
	// an average character budget. Preserve authored spaces and add no hyphens.
	boundaries := make([]int, 0, utf8.RuneCountInString(line))
	for index, char := range line {
		boundaries = append(boundaries, index+utf8.RuneLen(char))
	}
	low, high := 0, len(boundaries)
	for low < high {
		middle := low + (high-low)/2 //nolint:mnd // Binary search bisects the candidate range.
		measured, _ := font.MeasureString(line[:boundaries[middle]])
		if measured <= float64(width) {
			low = middle + 1
		} else {
			high = middle
		}
	}
	end := boundaries[max(0, low-1)]
	if end < len(line) {
		if space := strings.LastIndexByte(line[:end], ' '); space > 0 {
			// Keep the delimiter on its source row; offsets and every authored
			// byte then remain exact even when the viewport starts mid-message.
			end = space + 1
		}
	}

	return end
}
