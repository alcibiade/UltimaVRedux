package widgets

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/text"
)

const questJournalHistoryScreens = 3

type questJournalMessageSpan struct {
	message int
	start   int
	end     int
}

// Row positions are disposable. Message identity and the source offset survive reflow.
type questJournalHistory struct {
	source   string
	rows     []text.JournalHistoryRow
	spans    []questJournalMessageSpan
	capacity int
	window   int
	selected int
	top      int
	anchor   int
}

func (h *questJournalHistory) reflow(font *text.UltimaFont, width, capacity int) {
	h.rows = text.WrapJournalHistory(h.source, font, max(1, width))
	h.capacity = max(1, capacity)
	h.window = max(0, len(h.rows)-questJournalHistoryScreens*h.capacity)
	h.spans = h.spans[:0]
	for index := h.window; index < len(h.rows); index++ {
		message := h.rows[index].Message
		if message < 0 {
			continue
		}
		if len(h.spans) == 0 || h.spans[len(h.spans)-1].message != message {
			h.spans = append(h.spans, questJournalMessageSpan{message: message, start: index, end: index + 1})
		} else {
			h.spans[len(h.spans)-1].end = index + 1
		}
	}
	if len(h.spans) == 0 {
		h.selected, h.top, h.anchor = -1, 0, 0

		return
	}
	if h.selected < 0 {
		h.edge(true)

		return
	}
	if h.selected < h.spans[0].message {
		h.edge(false)

		return
	}
	if h.selected > h.spans[len(h.spans)-1].message {
		h.edge(true)

		return
	}
	span := h.spans[h.selectedSpan()]
	row := span.start
	for index := span.start; index < span.end && h.rows[index].SourceOffset <= h.anchor; index++ {
		row = index
	}
	if span.end-span.start <= h.capacity {
		h.center(span)
	} else {
		h.top = h.clampTop(row)
	}
	// Keep the exact source anchor across successive resizes, unless it was evicted.
	h.anchor = max(h.anchor, h.rows[span.start].SourceOffset)
}

func (h *questJournalHistory) selectedSpan() int {
	for index, span := range h.spans {
		if span.message == h.selected {
			return index
		}
	}

	return 0
}

func (h *questJournalHistory) clampTop(top int) int {
	return max(h.window, min(top, max(h.window, len(h.rows)-h.capacity)))
}

func (h *questJournalHistory) center(span questJournalMessageSpan) {
	h.top = h.clampTop(span.start - max(0, h.capacity-(span.end-span.start))/questJournalCenterDivisor)
}

func (h *questJournalHistory) rememberAnchor() {
	if h.selected < 0 {
		return
	}
	span := h.spans[h.selectedSpan()]
	h.anchor = h.rows[max(span.start, min(h.top, span.end-1))].SourceOffset
}

func (h *questJournalHistory) edge(newest bool) {
	if len(h.spans) == 0 {
		return
	}
	span := h.spans[0]
	h.top = h.clampTop(span.start)
	if newest {
		span = h.spans[len(h.spans)-1]
		h.top = h.clampTop(len(h.rows) - h.capacity)
	}
	h.selected = span.message
	h.rememberAnchor()
}

func (h *questJournalHistory) move(delta int) {
	if len(h.spans) == 0 {
		return
	}
	index := h.selectedSpan()
	next := max(0, min(index+delta, len(h.spans)-1))
	if next == index {
		return
	}
	span := h.spans[next]
	h.selected = span.message
	h.center(span)
	if delta < 0 && span.end-span.start > h.capacity {
		h.top = h.clampTop(span.end - h.capacity)
	}
	h.rememberAnchor()
}

func (h *questJournalHistory) page(direction int) {
	if len(h.spans) == 0 {
		return
	}
	index := h.selectedSpan()
	span := h.spans[index]
	if span.end-span.start > h.capacity {
		if direction > 0 && h.top+h.capacity < span.end {
			h.top = h.clampTop(min(h.top+h.capacity, span.end-h.capacity))
			h.rememberAnchor()

			return
		}
		if direction < 0 && h.top > span.start {
			h.top = h.clampTop(max(span.start, h.top-h.capacity))
			h.rememberAnchor()

			return
		}
		// Finish an oversized message before entering its neighbour.
		h.move(direction)

		return
	}
	target := max(h.window, min(span.start+direction*h.capacity, len(h.rows)-1))
	next := max(0, min(index+direction, len(h.spans)-1))
	for candidate, other := range h.spans {
		if other.start <= target && target < other.end {
			next = candidate

			break
		}
		if direction > 0 && other.start >= target {
			next = candidate

			break
		}
		if direction < 0 && other.end <= target {
			next = candidate
		}
	}
	h.move(next - index)
}

func (d *QuestJournalDialog) historyContentRect() image.Rectangle {
	content := insetRectangle(d.layout.bodyRect, d.layout.padding*questJournalPaneInsetMultiplier)
	content.Min.Y += d.rowHeight()
	content.Max.Y = max(content.Min.Y+d.rowHeight(), content.Max.Y-d.rowHeight())

	return content
}

func (d *QuestJournalDialog) drawHistory(screen *ebiten.Image) {
	content := d.historyContentRect()
	heading := image.Rect(content.Min.X, content.Min.Y-d.rowHeight(), content.Max.X, content.Min.Y)
	d.drawSingleLine(screen, heading, "Recent history", DialogHeaderColor)
	if len(d.history.spans) == 0 {
		d.drawSingleLine(screen, content, "No recent messages.", DialogEmptyColor)

		return
	}
	end := min(len(d.history.rows), d.history.top+d.history.capacity)
	for index := d.history.top; index < end; index++ {
		row := d.history.rows[index]
		top := content.Min.Y + (index-d.history.top)*d.rowHeight()
		rect := image.Rect(content.Min.X, top, content.Max.X, top+d.rowHeight())
		rowColor := DialogValueColor
		if row.Message == d.history.selected {
			vector.FillRect(screen, float32(rect.Min.X), float32(rect.Min.Y),
				float32(rect.Dx()), float32(rect.Dy()), DialogSelectedBackground, false)
			rowColor = DialogSelectedText
		}
		query := ""
		if row.Message == d.history.selected {
			query = d.readingQuery
		}
		d.drawJournalMatchLine(screen, rect, row.Text, query, rowColor, row.Message == d.history.selected)
	}
	position := "Newer text below"
	if end == len(d.history.rows) {
		position = "Newest"
	}
	if d.history.top > d.history.window {
		position = "Older text above | " + position
	}
	d.drawSingleLine(screen,
		image.Rect(content.Min.X, content.Max.Y, content.Max.X, content.Max.Y+d.rowHeight()), position, DialogEmptyColor)
}
