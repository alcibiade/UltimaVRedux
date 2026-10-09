package text

import (
	"image/color"
	"strings"
	"unicode/utf8"

	ultimacolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
)

const historyTruncationNotice = "[Earlier conversation text truncated]"

type outputInsertion struct {
	segments  []ColorSegment
	raw       string
	color     color.Color
	alignment etext.Align
	indent    int
	append    bool
	plain     bool
	trim      bool
}

// OutputHistory retains only already-inserted logical text, not wrapped rows or
// gameplay callbacks. It is opt-in and owned by one transient presentation.
type OutputHistory struct {
	entries    []outputInsertion
	bytes      int
	maxEntries int
	maxBytes   int
	truncated  bool
}

// RetainLogicalHistory starts bounded retention for subsequent insertions.
func (o *Output) RetainLogicalHistory(maxEntries, maxBytes int) {
	if maxEntries <= 0 || maxBytes <= 0 {
		panic("logical history limits must be positive")
	}
	o.history = &OutputHistory{entries: nil, bytes: 0, maxEntries: maxEntries, maxBytes: maxBytes, truncated: false}
}

func (o *Output) retainInsertion(entry outputInsertion) {
	if o.journalHistory != nil {
		o.journalHistory.retain(entry)
	}
	history := o.history
	if history == nil || (entry.raw == "" && len(entry.segments) == 0) {
		return
	}
	size := entry.byteLen()
	// Retain the newest text even for an oversized emission. The notice makes
	// the missing prefix explicit, and slicing never splits a UTF-8 character.
	if size > history.maxBytes {
		entry = trimHistoryInsertion(entry, size-history.maxBytes)
		size = entry.byteLen()
		history.entries = nil
		history.bytes = 0
		history.truncated = true
	}
	for len(history.entries) > 0 &&
		(len(history.entries) >= history.maxEntries || history.bytes+size > history.maxBytes) {
		history.bytes -= history.entries[0].byteLen()
		clear(history.entries[:1])
		history.entries = history.entries[1:]
		history.truncated = true
	}
	entry.raw = strings.Clone(entry.raw)
	entry.segments = append([]ColorSegment(nil), entry.segments...)
	for index := range entry.segments {
		entry.segments[index].Text = strings.Clone(entry.segments[index].Text)
	}
	history.entries = append(history.entries, entry)
	history.bytes += size
}

func (entry outputInsertion) byteLen() int {
	size := len(entry.raw)
	for _, segment := range entry.segments {
		size += len(segment.Text)
	}

	return size
}

func trimHistoryInsertion(entry outputInsertion, discard int) outputInsertion {
	if entry.plain {
		for discard < len(entry.raw) && !utf8.RuneStart(entry.raw[discard]) {
			discard++
		}
		entry.raw = entry.raw[discard:]

		return entry
	}
	entry.segments = append([]ColorSegment(nil), entry.segments...)
	for len(entry.segments) > 0 && discard >= len(entry.segments[0].Text) {
		discard -= len(entry.segments[0].Text)
		entry.segments = entry.segments[1:]
	}
	if len(entry.segments) > 0 {
		content := entry.segments[0].Text
		for discard < len(content) && !utf8.RuneStart(content[discard]) {
			discard++
		}
		entry.segments[0].Text = content[discard:]
	}

	return entry
}

// ReflowRetained rebuilds the live buffer with new typography, preserving source
// boundaries, colours and Runic segments. No engine or action is invoked.
func (o *Output) ReflowRetained(font *UltimaFont, spacing float64, chars int) {
	previousChars := o.maxCharsPerLine
	o.SetFont(font, spacing)
	o.SetMaxCharsPerLine(chars)
	// A font-only resize needs no text replay. In particular, bounded history
	// must not erase text that is still present in the live scrolling buffer.
	if o.history == nil || previousChars == chars {
		return
	}
	savedJournal := o.journalHistory
	o.journalHistory = nil
	defer func() { o.journalHistory = savedJournal }()
	o.Reset()
	o.replayRetained(o)
}

func (o *Output) replayRetained(target *Output) {
	history := o.history
	saved := target.history
	savedJournal := target.journalHistory
	target.history = nil
	target.journalHistory = nil
	defer func() {
		target.history = saved
		target.journalHistory = savedJournal
	}()
	if history.truncated {
		target.AddTextWithColor(historyTruncationNotice, ultimacolor.EGAYellow)
	}
	for index, entry := range history.entries {
		switch {
		case entry.plain:
			target.addTextInternal(entry.raw, entry.trim, entry.color)
		case entry.append:
			if index == 0 && history.truncated {
				target.AddStyledText(entry.segments, etext.AlignStart)
			} else {
				target.AppendStyledText(entry.segments)
			}
		default:
			target.addStyledText(entry.segments, entry.alignment, entry.indent)
		}
	}
}

// RetainedHistoryPanel produces chronological physical rows for keyboard paging.
// The conservative row bound includes explicit newlines and one-glyph wrapping.
func (o *Output) RetainedHistoryPanel() *Output {
	if o.history == nil {
		return o.WrappedCopy(o.maxCharsPerLine)
	}
	capacity := max(1, o.history.bytes+2*len(o.history.entries)+len(historyTruncationNotice)+1)
	panel := NewOutput(o.Font, o.lineSpacing, capacity, o.maxCharsPerLine)
	o.replayRetained(panel)

	return panel.WrappedCopy(o.maxCharsPerLine)
}
