package editorui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

const formRecordingPicker = "form recording"
const cueRecordingPicker = "sound recording"
const surfaceAddPicker = "add footstep"

func (m *model) isRecordingMenu() bool {
	return m.routing.menu != nil &&
		(m.routing.menu.kind == formRecordingPicker || m.routing.menu.kind == cueRecordingPicker)
}

func (m *model) openFormRecordingPicker() tea.Cmd {
	return m.openRecordingChoices(formRecordingPicker, "")
}

func (m *model) openSoundRecordingPicker() tea.Cmd {
	cue := m.assignmentCueForSelection()
	if cue == "" {
		m.status = "Choose an ambient sound first."

		return nil
	}

	return m.openRecordingChoices(cueRecordingPicker, cue)
}

func (m *model) openRecordingChoices(kind, target string) tea.Cmd {
	menu := &routingMenu{kind: kind, title: "CHOOSE RECORDING", target: target} //nolint:exhaustruct // Fresh picker.
	if kind == formRecordingPicker {
		menu.choices = append(menu.choices, routingChoice{formEmptyRecording, "Empty slot — supply a file later", "", nil})
	}
	for _, recording := range m.doc.RecordingIDs() {
		menu.choices = append(menu.choices, routingChoice{recording, recording, target, nil})
	}
	m.routing.menu = menu

	return m.loadSelection()
}

func (m *model) openSurfaceAddition() tea.Cmd {
	menu := &routingMenu{ //nolint:exhaustruct // Fresh compatible picker.
		kind: surfaceAddPicker, title: "ADD FOOTSTEP SOUND",
	}
	for _, cue := range m.doc.CueIDs() {
		if strings.HasPrefix(cue, "sfx/") {
			menu.choices = append(menu.choices, routingChoice{cue, m.doc.Cue(cue).Name, cue, nil})
		}
	}
	m.routing.menu = menu

	return m.loadSelection()
}

func (m *model) applyFormChoice(choice routingChoice) tea.Cmd {
	menu := m.routing.menu
	switch menu.kind {
	case formRecordingPicker:
		m.form.fields[newRecordingField].value = choice.id
	case surfaceAddPicker:
		m.form.sequence = append(m.form.sequence, choice.id)
		m.form.cursor = len(m.form.sequence) - 1
	case cueRecordingPicker:
		cue := m.doc.Cue(menu.target).WithRecordings([]string{choice.id})
		if err := m.doc.SetCue(menu.target, cue); err != nil {
			m.report(err)

			return nil
		}
	}
	m.routing.menu = nil

	return m.loadSelection()
}

func (m *model) surfaceFormKey(key string) tea.Cmd {
	form := m.form
	switch key {
	case keyEscape:
		return m.closeForm()
	case keyCtrlEnter:
		return m.applyForm()
	case "a":
		return m.openSurfaceAddition()
	case "up":
		form.cursor = max(0, form.cursor-1)

		return m.loadSelection()
	case keyDown:
		form.cursor = max(0, min(len(form.sequence)-1, form.cursor+1))

		return m.loadSelection()
	case "delete":
		if len(form.sequence) > 0 {
			form.sequence = append(form.sequence[:form.cursor], form.sequence[form.cursor+1:]...)
			form.cursor = max(0, min(form.cursor, len(form.sequence)-1))
		}

		return m.loadSelection()
	case "ctrl+up", keyCtrlDown:
		if len(form.sequence) > 0 {
			other := max(0, form.cursor-1)
			if key == keyCtrlDown {
				other = min(len(form.sequence)-1, form.cursor+1)
			}
			form.sequence[form.cursor], form.sequence[other] = form.sequence[other], form.sequence[form.cursor]
			form.cursor = other
		}
	case keyEnter, "p":
		return m.previewForm(false, false)
	case " ":
		return m.previewForm(false, true)
	}

	return nil
}

func (m *model) importAssignedRecording() tea.Cmd {
	cue := m.assignmentCueForSelection()
	if cue == "" {
		return nil
	}
	recording := m.doc.Cue(cue).PrimaryRecording()
	m.jumpToAudioUsage(audio.Usage{Workspace: "recordings", Target: recording, Label: recording})

	return m.openPicker()
}
