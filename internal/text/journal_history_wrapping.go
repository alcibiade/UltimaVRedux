package text

import (
	"fmt"
	"strings"

	beradtext "github.com/bradhannah/berad-sdk/modules/berad-text"
)

// JournalHistoryRow maps a display row to its logical message and UTF-8 offset.
// Blank separators retain Message -1; journal ownership stays with the game.
type JournalHistoryRow struct {
	Message      int
	SourceOffset int
	Text         string
}

// WrapJournalHistory delegates measured source-preserving reflow to the SDK.
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
		content, err := beradtext.NewContent([]beradtext.Span{{Text: line, Font: font.sdkForScale(1), Paint: "", Tag: ""}})
		if err != nil {
			panic(fmt.Sprintf("composing retained game text: %v", err))
		}
		var options beradtext.LayoutOptions
		options.Width = float64(width)
		layout, err := beradtext.NewLayout(content, options)
		if err != nil {
			panic(fmt.Sprintf("wrapping retained game text: %v", err))
		}
		for _, row := range layout.Lines() {
			rows = append(rows, JournalHistoryRow{Message: message, SourceOffset: row.Start, Text: line[row.Start:row.End]})
		}
		message++
	}

	return rows
}
