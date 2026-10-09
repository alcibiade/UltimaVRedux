package editorui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

type palette struct {
	title, focus, ready, warning, problem, dim, selected, key lipgloss.Style
	detailLabel, detailValue                                  lipgloss.Style
}

const (
	hintStopAll  = "Stop all"
	hintCancel   = "Cancel"
	hintPause    = "Pause"
	hintGainKeys = "+/-"
)

func newPalette() palette {
	return palette{
		title: lipgloss.NewStyle().Bold(true), focus: lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true),
		ready:       lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		warning:     lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		problem:     lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true),
		dim:         lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		selected:    lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("4")).Bold(true),
		key:         lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("6")).Bold(true),
		detailLabel: lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true),
		detailValue: lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
	}
}

func (m *model) badge(status audio.RecordingAvailability) string {
	symbol, style := "…", m.colors.dim
	switch status.State {
	case audio.RecordingReady:
		symbol, style = "●", m.colors.ready
	case audio.RecordingNoFile:
		symbol, style = "○", m.colors.warning
	case audio.RecordingMissing, audio.RecordingInvalid:
		symbol, style = "!", m.colors.problem
	case audio.RecordingChecking:
	}

	return style.Render(symbol + " " + string(status.State))
}

type keyHint struct{ key, action string }

// hintRows packs complete shortcuts across the real terminal width. A long
// terminal gets fewer footer rows rather than retaining narrow fixed groups.
func (m *model) hintRows(hints []keyHint) []string {
	var rows []string
	line := ""
	for _, hint := range hints {
		item := hint.action
		if hint.key != "" {
			item = m.colors.key.Render("["+hint.key+"]") + " " + item
		}
		separator := ""
		if line != "" {
			separator = "  "
		}
		if line != "" && ansi.StringWidth(line+separator+item) > m.width {
			rows = append(rows, fit(line, m.width))
			line = ""
			separator = ""
		}
		line += separator + item
	}
	if line != "" {
		rows = append(rows, fit(line, m.width))
	}

	return rows
}

//nolint:goconst // Help rows repeat literal keys and labels so each context is readable as a binding table.
func (m *model) footerLines() []string {
	if (m.routing.menu != nil || m.routingWorkspace()) && m.field == "" && !m.confirmQuit && !m.volumeControls {
		return m.routingFooter()
	}
	if m.pendingAmbient != nil {
		return m.hintRows([]keyHint{{"1/2 or ↑↓", "Ambient slot"}, {"Enter", "Replace"},
			{"Esc", "Cancel"}, {"x", hintStopAll}})
	}
	if m.volumeControls {
		return m.hintRows([]keyHint{{"↑↓", "Control"}, {"←→ or +/-", "Change"},
			{"r", "Game Mix/Reference"}, {"Enter/Esc", "Close"}, {"x", hintStopAll}})
	}
	if m.autoGainDraft != nil {
		return m.hintRows([]keyHint{{"Enter", "Apply auto-gain"}, {"Esc", "Keep calibration"}, {"x", hintStopAll}})
	}
	if m.busy {
		return m.hintRows([]keyHint{{"", "Finishing file operation…"}, {"x", hintStopAll}})
	}
	if m.field != "" {
		if m.field == fieldSearch {
			backspace := "Delete character"
			if m.filterClearPending {
				backspace = "Clear existing filter once"
			}

			return m.hintRows([]keyHint{{"", "EDIT FILTER · keywords or category:/family:/context:"},
				{"Backspace", backspace}, {"Enter", "Apply filter"}, {"Esc", hintCancel}, {"Ctrl+u", "Clear"}})
		}

		return m.hintRows([]keyHint{{"", m.inputGuidance()},
			{"Enter", "Confirm"}, {"Esc", hintCancel}, {"Ctrl+u", "Clear"}})
	}
	if m.confirmQuit {
		return m.hintRows([]keyHint{{"y", "Discard and quit"}, {"Esc", "Keep editing"}})
	}
	if m.duplicate != nil {
		return m.hintRows([]keyHint{{"↑↓", "Field"}, {"Enter", "Edit"}, {"s/e", "Bounds"}, {"i/o", "Mark"},
			{"p", "Preview"}, {"a", "Append toggle"}, {"c", "Create clip"}, {"Esc", "Cancel"}})
	}
	if m.picker != nil {
		return m.hintRows([]keyHint{{"↑↓", "Choose"}, {"Enter", "Open/import"}, {"p", "Preview"},
			{"Ctrl+p/n", "Prev/next page"}, {"/", "Path"}, {"Backspace", "Parent"}, {"Esc", hintCancel}, {"x", "Stop"}})
	}
	if m.assigning {
		return m.hintRows([]keyHint{{"↑↓", "Choose"}, {"Enter", "Preview"}, {"y", "Assign"}, {"Esc", hintCancel},
			{"/", "Search"}, {"Ctrl+f", "Filter"}, {"x", hintStopAll}})
	}
	if m.variantCue != "" {
		return m.hintRows([]keyHint{{"↑↓", "Clip"}, {"Enter", "Preview"}, {"a", "Append"}, {"d", "Remove"},
			{"Ctrl+↑↓", "Reorder"}, {"Ctrl+p/n", "Prev/next page"}, {"Shift+d", "Duplicate"},
			{"Ctrl+s", "Save"}, {"Esc", "Back"}})
	}
	const primaryHintCapacity = 9
	rows := make([]keyHint, 0, primaryHintCapacity)
	rows = append(rows, keyHint{"Tab/1–4", "Workspace"}, keyHint{"/", "Search"},
		keyHint{"f", "Choose sound/file"}, keyHint{"?", "All shortcuts"})
	if m.focus == focusWave {
		rows = []keyHint{{"i/o", "Mark trim"}, {"s/e", "Exact trim"}, {"←→", "Seek/nudge"}, {"Esc", "Browse"}, {"?", "Help"}}
	}
	if m.focus == focusDetails {
		rows = []keyHint{{"↑↓", "Scroll"}, {"f", "Choose sound/file"}, {"Ctrl+L", "License"},
			{"Esc", "Browse"}, {"?", "Help"}}
	}
	global := []keyHint{{"Ctrl+s", "Save"}, {"u", "Undo"}, {"q", "Quit"}}
	if m.focus == focusMix {
		rows = []keyHint{{"↑↓", "Layer"}, {"Enter", "Restart"}, {"Space", "Pause"}, {"u", "Mute"}, {"Esc", "Browse"}}
		global = []keyHint{{"v", "Preview volumes"}, {"x", hintStopAll}, {"Ctrl+s", "Save"}, {"q", "Quit"}}
	}

	return m.hintRows(append(rows, global...))
}

func (m *model) fieldLabel() string {
	if m.field == fieldStart {
		return "Start time (ms)"
	}
	if m.field == fieldEnd {
		return "End time (ms)"
	}
	if m.field == fieldGain && !m.browsingCues() {
		return "Recording calibration (saved dB)"
	}
	if m.field == fieldGain {
		return "Moment gain (saved amplitude)"
	}

	return m.field
}

func (m *model) trimExamples() (int64, int64) {
	const exampleStartMs = 420
	const exampleEndMs = 2350
	const startFraction = 4
	end := int64(exampleEndMs)
	if m.clip != nil {
		end = min(end, m.clip.DurationMs)
		if m.window().StartMs >= end {
			end = m.clip.DurationMs
		}
	}
	startLimit := end
	if m.clip != nil {
		startLimit = min(startLimit, m.window().EndMs)
	}
	start := min(int64(exampleStartMs), max(0, startLimit)/startFraction)

	return start, end
}

func timeExample(label string, milliseconds int64) string {
	return fmt.Sprintf("%s ms e.g. %d=%.3fs", label, milliseconds, float64(milliseconds)/msPerSecond)
}

func (m *model) inputGuidance() string {
	start, end := m.trimExamples()
	switch m.field {
	case fieldLicense:
		return "Enter documented license terms (e.g. CC BY 4.0), or unknown. Blank is not allowed."
	case fieldKeywords:
		return "Comma-separated keywords or phrases · replaces this item's custom terms"
	case fieldTrigger:
		return "One sentence describing this moment's usual gameplay trigger"
	case fieldStart, fieldClipStart:
		return timeExample("Start", start) + "; before end."
	case fieldEnd, fieldClipEnd:
		return timeExample("End", end) + "; after start."
	default:
		return "Text input · Enter applies"
	}
}
