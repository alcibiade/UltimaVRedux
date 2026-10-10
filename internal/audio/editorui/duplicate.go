package editorui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

const fieldClipID = "clip ID"
const fieldClipStart = "clip start ms"
const fieldClipEnd = "clip end ms"

type clipDraft struct {
	sourceID, id, eligibleCue string
	window                    audio.PlaybackWindow
	appendToCue               bool
	field                     int
}

func (m *model) beginDuplicate() {
	if m.clip == nil || m.assigning {
		m.status = "Load a source recording before duplicating its metadata."

		return
	}
	newID, err := m.doc.NextRecordingID(m.recordingID())
	if err != nil {
		m.report(err)

		return
	}
	window := m.window()
	if window.EndMs < m.clip.DurationMs {
		window.StartMs = window.EndMs
		window.EndMs = m.clip.DurationMs
	}
	cueID := m.activeCue()
	if cueID == "" {
		uses := m.doc.Usages(m.recordingID())
		if len(uses) == 1 {
			cueID = uses[0]
		}
	}
	if !strings.HasPrefix(cueID, "sfx/") {
		cueID = ""
	}
	m.duplicate = &clipDraft{sourceID: m.recordingID(), id: newID, window: window,
		eligibleCue: cueID, appendToCue: false, field: 0}
	m.fullSource, m.marker = false, 0
	m.report(m.preview.Close())
	m.position = window.StartMs
	m.status = "New metadata-only clip: edit ID/start/end; p previews; c creates; Esc cancels."
}

func (m *model) duplicateKey(key string) tea.Cmd {
	draft := m.duplicate
	const finalClipField = 2
	switch key {
	case keyEscape:
		m.duplicate = nil

		return m.loadSelection()
	case "up":
		draft.field = max(0, draft.field-1)
	case keyDown:
		draft.field = min(finalClipField, draft.field+1)
	case keyEnter:
		fields := []string{fieldClipID, fieldClipStart, fieldClipEnd}
		values := []string{draft.id, strconv.FormatInt(draft.window.StartMs, 10), strconv.FormatInt(draft.window.EndMs, 10)}
		m.field, m.input = fields[draft.field], values[draft.field]
	case "s":
		m.field, m.input = fieldClipStart, strconv.FormatInt(draft.window.StartMs, 10)
	case "e":
		m.field, m.input = fieldClipEnd, strconv.FormatInt(draft.window.EndMs, 10)
	case "a":
		if draft.eligibleCue != "" {
			draft.appendToCue = !draft.appendToCue
		}
	case "p":
		m.report(m.preview.Play(m.clip, &draft.window, false, m.gain()))
	case "i":
		draft.window.StartMs = m.position
	case "o":
		draft.window.EndMs = m.position
	case "c":
		return m.createDuplicate()
	}

	return nil
}

func (m *model) commitDuplicateField(field, value string) {
	if field == fieldClipID {
		m.duplicate.id = value

		return
	}
	milliseconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || milliseconds < 0 || milliseconds > m.clip.DurationMs {
		m.rejectInput("Enter whole milliseconds within the source. Both bounds are checked together on Create.")

		return
	}
	if field == fieldClipStart {
		m.duplicate.window.StartMs = milliseconds
	} else {
		m.duplicate.window.EndMs = milliseconds
	}
}

func (m *model) createDuplicate() tea.Cmd {
	draft := m.duplicate
	appendCue := ""
	if draft.appendToCue {
		appendCue = draft.eligibleCue
	}
	if err := m.doc.DuplicateClip(draft.sourceID, draft.id, draft.window, appendCue); err != nil {
		m.report(err)

		return nil
	}
	m.duplicate = nil
	m.clearSoundscape()
	if m.variantCue == "" || appendCue != m.variantCue {
		m.variantCue = ""
		m.workspace = workspaceRecordings
	}
	m.search, m.filter = "", filterAll
	m.prepareAvailability()
	m.refreshKeys()
	m.selectID(draft.id)
	m.status = "Created " + draft.id + " using the same file; u undoes the entire edit; Ctrl+s saves."

	return tea.Batch(m.scanNext(), m.loadSelection())
}

func (m *model) duplicateLines(width int) []string {
	draft := m.duplicate
	values := []string{draft.id, fmt.Sprintf("%d ms", draft.window.StartMs), fmt.Sprintf("%d ms", draft.window.EndMs)}
	labels := []string{"ID: ", "Start: ", "End: "}
	lines := []string{fit("Same physical file; independent metadata.", width)}
	for index, value := range values {
		prefix := "  "
		if draft.field == index {
			prefix = "> "
		}
		lines = append(lines, fit(prefix+labels[index]+value, width))
	}
	if draft.eligibleCue != "" {
		lines = append(lines, fit(fmt.Sprintf("Append: %t (a toggles)", draft.appendToCue), width),
			fit("Moment: "+draft.eligibleCue, width))
	}

	return lines
}
