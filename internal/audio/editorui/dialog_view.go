package editorui

import (
	"fmt"
	"path"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

type editorDialog struct {
	title string
	rows  []string
	focus int
	help  []string
}

const (
	editorDialogMargin = 4
	dialogHeaderRows   = 2
	inputTailRows      = 3
	inputPadding       = 4
)

func (m *model) inlineInput() bool {
	switch m.field {
	case fieldSearch, fieldGain, fieldStart, fieldEnd, fieldCalibrationTarget:
		return true
	default:
		return false
	}
}

func (m *model) inputLines(width int) []string {
	label := "EDITING " + strings.ToUpper(m.fieldLabel())
	value := m.input + "▏"
	var rows []string
	for _, line := range strings.Split(value, "\n") {
		rows = append(rows, strings.Split(ansi.Hardwrap(safe(line), width-inputPadding, true), "\n")...)
	}
	for index := range rows {
		rows[index] = m.colors.key.Render(fit("  "+rows[index], width))
	}
	rows = append([]string{m.colors.focus.Render(fit(label, width)), ""}, rows...)
	if m.field == fieldLicense {
		const subjectRows = 3
		subject := make([]string, 0, subjectRows+len(rows))
		subject = append(subject, m.colors.detailLabel.Render("Recording: ")+
			m.colors.detailValue.Render(safe(m.licenseRecordingID)),
			fmt.Sprintf("Shared provenance · used by %d Moment(s).", len(m.doc.Usages(m.licenseRecordingID))), "")
		rows = append(subject, rows...)
	}
	if m.inputError != "" {
		for _, line := range strings.Split(ansi.Hardwrap("! "+safe(m.inputError), width, true), "\n") {
			rows = append(rows, m.colors.problem.Render(line))
		}
	}
	rows = append(rows, "", m.colors.dim.Render(fit(m.inputGuidance(), width)))

	return rows
}

// Modal priority mirrors input ownership, including nested picker/volume dialogs.
func (m *model) dialogContent(width int) (editorDialog, bool) {
	if m.confirmQuit {
		return editorDialog{title: "UNSAVED CHANGES", rows: []string{
			m.colors.warning.Render("The library or editing dialog contains unsaved changes."),
			"Y discards them and quits. Escape returns to editing.",
		}, focus: 0, help: []string{"Y: discard and quit   Escape: keep editing"}}, true
	}
	if m.field != "" && !m.inlineInput() {
		help := "Enter: apply   Escape: cancel   Ctrl+U: clear"
		if m.field == fieldNote {
			help = "Ctrl+Enter: apply   Enter: new line   Escape: cancel"
		}
		rows := m.inputLines(width)

		return editorDialog{title: "EDIT " + strings.ToUpper(m.fieldLabel()), rows: rows,
			focus: max(0, len(rows)-inputTailRows), help: []string{help}}, true
	}
	if m.infoDialog != nil {
		return editorDialog{title: m.infoDialog.title, rows: m.infoDialog.lines, focus: m.infoDialog.scroll,
			help: []string{"Up/Down: scroll   Enter/Escape: close"}}, true
	}
	if m.volumeControls && m.hasDraft() {
		return editorDialog{title: "PREVIEW VOLUME — temporary", rows: m.previewVolumeLines(width),
			focus: m.volumeCursor, help: []string{"Up/Down: control   Left/Right: adjust   Escape: return"}}, true
	}
	if m.routing.menu != nil {
		return m.routingPickerDialog(width), true
	}
	if m.form != nil {
		return m.formDialog(width), true
	}
	if m.picker != nil {
		return m.filePickerDialog(width), true
	}
	if m.assigning {
		return m.recordingAssignmentDialog(width), true
	}
	if m.duplicate != nil {
		return editorDialog{title: "DUPLICATE CLIP — draft", rows: m.duplicateLines(width), focus: m.duplicate.field + 1,
			help: []string{"Enter: edit field   P: preview   C: create   Escape: cancel"}}, true
	}
	if m.autoGainDraft != nil {
		return editorDialog{title: "AUTO-GAIN", rows: []string{
			fmt.Sprintf("Suggested recording calibration: %+.2f dB", m.autoGainDraft.GainDB),
			"Apply this calibration and enable automatic gain updates?",
		}, focus: 0, help: []string{"Enter: apply   Escape: keep existing calibration"}}, true
	}
	if m.pendingAmbient != nil {
		return editorDialog{title: "REPLACE AMBIENT LAYER", rows: []string{
			"Both ambient slots are occupied. Choose which layer to replace.",
			fmt.Sprintf("Selected slot: %d", m.mixCursor+1),
		}, focus: 1, help: []string{"1/2: select slot   Enter: replace   Escape: cancel"}}, true
	}

	return editorDialog{title: "", rows: nil, focus: 0, help: nil}, false
}

//nolint:cyclop,nestif // Fields, inline errors and actions render in order so focus offsets stay aligned.
func (m *model) formDialog(width int) editorDialog {
	form := m.form
	rows := []string{m.colors.warning.Render("DRAFT · Apply updates the library; Ctrl+S saves the file."), ""}
	focus := 0
	if form.kind == formSurface {
		rows = append(rows, "Ordered footsteps · shared by every tile using "+form.surface)
		for index, cue := range form.sequence {
			line := fmt.Sprintf("%d. %s", index+1, m.doc.Cue(cue).Name)
			if index == form.cursor {
				focus = len(rows)
				line = m.colors.key.Render(fit("> "+line, width))
			}
			rows = append(rows, line)
		}
		if len(form.sequence) == 0 {
			rows = append(rows, "Silent surface. A adds a footstep sound.")
		}
	} else {
		if form.kind == formTileAmbient {
			rows = append(rows, "Applies only to this tile family, including all its animation frames.",
				"Other tile types can reuse the same sound with different distance/level.", "")
		}
		if form.kind == formAmbient {
			rows = append(rows, "Shared sound settings: every source using this sound is affected.", "")
		}
		for index, field := range form.fields {
			if index == form.cursor {
				focus = len(rows)
			}
			rows = append(rows, m.formFieldLine(field, index == form.cursor, form.editing, width))
			if index == form.cursor {
				rows = append(rows, m.colors.dim.Render(fit(field.help, width)))
			}
			if index == form.errorField && form.errorText != "" {
				rows = append(rows, m.colors.problem.Render("! "+safe(form.errorText)))
			}
		}
		for index, action := range []string{"Apply changes", hintCancel} {
			line := "[ " + action + " ]"
			if form.cursor == len(form.fields)+index {
				focus = len(rows)
				line = m.colors.key.Render(fit("> "+line, width))
			}
			rows = append(rows, line)
		}
	}
	if form.errorField < 0 && form.errorText != "" {
		rows = append(rows, m.colors.problem.Render("! "+safe(form.errorText)))
		focus = len(rows) - 1
	}
	help := []string{"Tab/Up/Down: focus   Enter: edit/activate   Left/Right: adjust",
		"Ctrl+Enter: apply   Escape: cancel   P: preview   Space: pause   L: departure"}
	if form.editing {
		help = []string{"TYPING IN HIGHLIGHTED FIELD · Ctrl+U clears · Enter finishes field",
			"Ctrl+Enter: apply draft   Escape: leave field"}
	} else if form.kind == formSurface {
		help = []string{"A: add   Delete: remove   Ctrl+Up/Down: reorder   Space: preview/pause",
			"Ctrl+Enter: apply sequence   Escape: cancel"}
	}

	return editorDialog{title: form.title, rows: rows, focus: focus, help: help}
}

func (m *model) formFieldLine(field formField, active, editing bool, width int) string {
	value := safe(field.value)
	if active && editing {
		value += "▏"
		available := max(1, width-ansi.StringWidth(field.label+field.unit)-inputPadding-dialogHeaderRows)
		if length := ansi.StringWidth(value); length > available {
			value = ansi.Cut(value, length-available, length)
		}
	}
	if active {
		return m.colors.key.Render(fit("> "+field.label+": [ "+value+" ] "+field.unit, width))
	}

	return m.colors.detailLabel.Render(field.label+": ") + m.colors.detailValue.Render("[ "+value+" ] "+field.unit)
}

//nolint:funlen // Keep picker rows, semantic status colours and the focused row in one rendering pass.
func (m *model) routingPickerDialog(width int) editorDialog {
	menu := m.routing.menu
	choices := m.routingMenuChoices()
	const pickerGaps = 2
	remaining := width - rowPrefixWidth - routingStatusWidth - pickerGaps
	nameWidth, recordingWidth := remaining/columnSplit, remaining-remaining/columnSplit
	search := "Search: " + safe(menu.search)
	if menu.editingSearch {
		search = m.colors.key.Render(fit("EDIT SEARCH: "+safe(menu.search)+"▏", width))
	}
	rows := []string{search, m.colors.detailLabel.Render(strings.Repeat(" ", rowPrefixWidth) +
		fit("Name", nameWidth) + " " + fit("Recording / file", recordingWidth) + " " + fit("State", routingStatusWidth))}
	for index, choice := range choices {
		status, recording, style := "— None", "—", m.colors.dim
		if choice.cue != "" {
			status, style = m.routingAssignmentStyle(choice.cue)
			recording = m.doc.Cue(choice.cue).PrimaryRecording()
		}
		if m.isRecordingMenu() && choice.id != formEmptyRecording {
			availability := m.doc.FileAvailability(choice.id)
			if cached, exists := m.availability[choice.id]; exists {
				availability = cached
			}
			status = string(availability.State)
			style = m.colors.warning
			switch availability.State { //nolint:exhaustive // Empty/checking retain the amber default.
			case audio.RecordingReady:
				style = m.colors.ready
			case audio.RecordingInvalid, audio.RecordingMissing:
				style = m.colors.problem
			}
			if file := m.doc.Recording(choice.id).File; file != nil {
				recording = path.Base(*file)
			}
		}
		prefix, nameStyle := "  ", m.colors.detailValue
		if index == menu.cursor {
			prefix, nameStyle = "> ", m.colors.selected
		}
		line := nameStyle.Render(prefix+fit(safe(choice.label), nameWidth)) + " " +
			style.Render(fit(safe(recording), recordingWidth)) + " " + style.Render(fit(status, routingStatusWidth))
		rows = append(rows, line)
	}
	focus := menu.cursor + dialogHeaderRows
	if menu.editingSearch {
		focus = 0
	}
	if len(choices) == 0 {
		rows = append(rows, "No matching sounds. / edits the search.")
	}

	help := []string{m.transportState() + " · " + safe(m.previewDescription()),
		"Enter: choose   Space: preview/pause   /: search   Escape: cancel"}
	if menu.kind == "cue" {
		help = append(help, "B: sound settings   r: recording   Ctrl+A: new ambient sound")
	}

	return editorDialog{title: menu.title, rows: rows, focus: focus, help: help}
}

func (m *model) recordingAssignmentDialog(width int) editorDialog {
	rows := append(m.browserSearchLines(width), m.columnHeader(width))
	for index, identity := range m.keys {
		rows = append(rows, m.itemLine(identity, index == m.cursor, width))
	}

	return editorDialog{title: "CHOOSE RECORDING — " + m.doc.Cue(m.assignmentCue).Name,
		rows: rows, focus: m.cursor + len(m.browserSearchLines(width)) + 1,
		help: []string{m.transportState() + " · " + safe(m.previewDescription()),
			"Enter: assign   Space: preview/pause   /: search   Escape: cancel"}}
}

func (m *model) filePickerDialog(width int) editorDialog {
	rows := make([]string, 0, dialogHeaderRows+len(m.picker.entries))
	rows = append(rows, m.colors.detailLabel.Render(fit(safe(m.picker.directory), width)), "")
	for index, entry := range m.picker.entries {
		line := safe(entry.name)
		if entry.directory {
			line += "/"
		}
		if index == m.picker.cursor {
			line = m.colors.key.Render(fit("> "+line, width))
		}
		rows = append(rows, line)
	}

	return editorDialog{title: "CHOOSE FILE — " + m.recordingID(), rows: rows, focus: m.picker.cursor + dialogHeaderRows,
		help: []string{"Shared recording: all its uses follow this file assignment.",
			"Enter: open/import   Space: preview/pause   /: path   Escape: cancel"}}
}

func (m *model) overlayDialog(background string, dialog editorDialog, width int) string {
	maxHeight := m.height - editorDialogMargin
	help := make([]string, 0, len(dialog.help))
	for _, line := range dialog.help {
		help = append(help, strings.Split(ansi.Wrap(line, width-panelEdges, ""), "\n")...)
	}
	wrapped := make([]string, 0, len(dialog.rows))
	focus := 0
	for index, line := range dialog.rows {
		if index == dialog.focus {
			focus = len(wrapped)
		}
		wrapped = append(wrapped, strings.Split(ansi.Wrap(line, width-panelEdges, ""), "\n")...)
	}
	capacity := max(1, maxHeight-panelEdges-len(help)-1)
	height := min(maxHeight, len(wrapped)+panelEdges+len(help)+1)
	start := listViewportStart(0, focus, len(wrapped), capacity)
	rows := append([]string(nil), wrapped[start:min(len(wrapped), start+capacity)]...)
	rows = append(rows, "")
	rows = append(rows, help...)
	box := strings.Split(m.panel(dialog.title, rows, width, height, true), "\n")
	base := strings.Split(background, "\n")
	left, top := (m.width-width)/columnSplit, max(dialogHeaderRows, (m.height-height)/columnSplit)
	for index, line := range base {
		if index >= dialogHeaderRows {
			base[index] = m.colors.dim.Render(fit(ansi.Strip(line), m.width))
		}
	}
	for index, line := range box {
		row := top + index
		if row >= len(base) {
			break
		}
		base[row] = ansi.Cut(base[row], 0, left) + line + ansi.Cut(base[row], left+width, m.width)
	}

	return strings.Join(base, "\n")
}
