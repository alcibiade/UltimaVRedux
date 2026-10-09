package text

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	ultimacolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

type journalMessage struct {
	text      string
	separator bool
}

// The unfinished target is separate from retained messages: appending to an
// explicit blank fills that row, whereas a new Add leaves a separator behind.
type journalHistory struct {
	entries      []journalMessage
	target       []byte
	pendingBlank bool
	explicit     bool
	nonblank     bool
	saturated    bool
}

// EnableJournalHistory opts this output into plain main-rail retention.
// Existing wrapped rows are deliberately not imported as logical messages.
func (o *Output) EnableJournalHistory() {
	if o.journalHistory == nil {
		o.journalHistory = new(journalHistory)
	}
}

// JournalHistory returns a detached canonical snapshot, without unfinished blank padding.
func (o *Output) JournalHistory() string {
	if o.journalHistory == nil {
		return ""
	}
	history := o.journalHistory
	var result strings.Builder
	for _, entry := range history.entries {
		writeJournalMessage(&result, entry.text, entry.separator)
	}
	if history.nonblank {
		writeJournalMessage(&result, string(history.target), history.pendingBlank)
	}

	return result.String()
}

func writeJournalMessage(result *strings.Builder, message string, separator bool) {
	if result.Len() > 0 {
		result.WriteByte('\n')
		if separator {
			result.WriteByte('\n')
		}
	}
	result.WriteString(message)
}

// RestoreJournalHistory validates before replacing either projection. Disabled
// outputs remain disabled; restoration never enables modal transcript capture.
func (o *Output) RestoreJournalHistory(history string) error {
	if err := journal.ValidateHistory(history); err != nil {
		return fmt.Errorf("restore journal history: %w", err)
	}
	restored := new(journalHistory)
	if history != "" {
		restored.retain(outputInsertion{
			raw: history, plain: true, segments: nil, color: nil,
			alignment: 0, indent: 0, append: false, trim: false,
		})
		restored.saturated = len(restored.target) >= journal.MaxHistoryLineBytes-utf8.UTFMax+1 &&
			strings.HasSuffix(string(restored.target), journal.HistoryShortened)
	}
	savedHistory, savedJournal := o.history, o.journalHistory
	o.history, o.journalHistory = nil, nil
	o.Reset()
	o.restoreJournalRail(history)
	o.history = savedHistory
	if savedJournal != nil {
		o.journalHistory = restored
	}

	return nil
}

func (o *Output) restoreJournalRail(history string) {
	if o.maxLines == 0 || history == "" {
		return
	}
	for line := range strings.SplitSeq(history, "\n") {
		segments := []ColorSegment{{
			Text: line, Color: ultimacolor.EGAWhite, FontType: FontTypeNormal, FontScale: FontScaleNormal,
		}}
		// The rune-aware wrapper avoids introducing broken UTF-8 into a restored
		// rail, while the collector keeps the exact unwrapped checkpoint source.
		for _, wrapped := range o.wrapSingleLogicalLine(segments) {
			var plain strings.Builder
			for _, segment := range wrapped.Segments {
				plain.WriteString(segment.Text)
			}
			o.lines[o.nextLineToIndex] = StyledLine{
				Text: plain.String(), Color: ultimacolor.EGAWhite, Segments: nil, Alignment: 0,
			}
			o.nextLineToIndex = (o.nextLineToIndex + 1) % o.maxLines
		}
	}
}

func (h *journalHistory) startTarget(explicit bool) {
	if h.nonblank {
		h.entries = append(h.entries, journalMessage{text: string(h.target), separator: h.pendingBlank})
		h.pendingBlank = false
	} else if h.explicit {
		h.pendingBlank = true
	}
	h.target = h.target[:0]
	h.explicit = explicit
	h.nonblank = false
	h.saturated = false
	h.enforceLimits()
}

func (h *journalHistory) retain(entry outputInsertion) {
	if entry.plain && entry.raw == "" || !entry.plain && len(entry.segments) == 0 {
		return
	}
	if !entry.append {
		marker := slices.ContainsFunc(entry.segments, func(segment ColorSegment) bool {
			return segment.FontType == FontTypeCommandPrompt
		})
		h.startTarget(!marker)
	}
	// Delay each LF until the next character so plain Add and Append discard
	// exactly one terminal empty component, including across styled segments.
	pendingLF := false
	consume := func(char rune) {
		if pendingLF {
			h.startTarget(true)
			pendingLF = false
		}
		if char == '\n' {
			pendingLF = true
			h.explicit = true

			return
		}
		h.explicit = true
		h.addRune(char)
	}
	var pendingUTF8 []byte
	if entry.plain {
		consumeJournalUTF8(entry.raw, &pendingUTF8, consume)
	} else {
		for _, segment := range entry.segments {
			if segment.FontType != FontTypeCommandPrompt {
				consumeJournalUTF8(segment.Text, &pendingUTF8, consume)
			}
		}
	}
	for range pendingUTF8 {
		consume(utf8.RuneError)
	}
	if pendingLF && !entry.plain && !entry.append {
		h.startTarget(true)
	}
	h.enforceLimits()
}

// Carry at most one incomplete rune between segments so concatenation precedes
// normalization, without allocating a copy of an arbitrarily large emission.
func consumeJournalUTF8(source string, pending *[]byte, consume func(rune)) {
	for len(source) > 0 {
		if len(*pending) == 0 && utf8.FullRuneInString(source) {
			char, size := utf8.DecodeRuneInString(source)
			consume(char)
			source = source[size:]

			continue
		}
		*pending = append(*pending, source[0])
		source = source[1:]
		for utf8.FullRune(*pending) {
			char, size := utf8.DecodeRune(*pending)
			consume(char)
			*pending = (*pending)[size:]
		}
	}
}

func (h *journalHistory) addRune(char rune) {
	if unicode.IsControl(char) || char == '\u2028' || char == '\u2029' {
		char = ' '
	}
	h.nonblank = h.nonblank || !unicode.IsSpace(char)
	if h.saturated {
		return
	}
	if len(h.target)+utf8.RuneLen(char) <= journal.MaxHistoryLineBytes {
		h.target = utf8.AppendRune(h.target, char)

		return
	}
	prefix := journal.MaxHistoryLineBytes - len(journal.HistoryShortened)
	for prefix > 0 && !utf8.RuneStart(h.target[prefix]) {
		prefix--
	}
	h.target = append(h.target[:prefix], journal.HistoryShortened...)
	h.saturated = true
}

func (h *journalHistory) enforceLimits() {
	count, size := 0, 0
	for _, entry := range h.entries {
		if count > 0 {
			size++
			if entry.separator {
				size++
			}
		}
		size += len(entry.text)
		count++
	}
	if h.nonblank {
		if count > 0 {
			size++
			if h.pendingBlank {
				size++
			}
		}
		size += len(h.target)
		count++
	}
	for len(h.entries) > 0 && (count > journal.MaxHistoryEntries || size > journal.MaxHistoryBytes) {
		size -= len(h.entries[0].text)
		clear(h.entries[:1])
		h.entries = h.entries[1:]
		count--
		separator := h.pendingBlank
		if len(h.entries) > 0 {
			separator = h.entries[0].separator
			h.entries[0].separator = false
		} else {
			h.pendingBlank = false
		}
		if count > 0 {
			size--
			if separator {
				size--
			}
		}
	}
}

func (h *journalHistory) removeCharacter() {
	if h.saturated || len(h.target) == 0 {
		return
	}
	_, size := utf8.DecodeLastRune(h.target)
	h.target = h.target[:len(h.target)-size]
	h.nonblank = strings.TrimSpace(string(h.target)) != ""
}
