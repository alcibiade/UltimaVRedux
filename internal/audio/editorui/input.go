package editorui

import (
	"math"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

func (m *model) editInput(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case keyEscape:
		m.field, m.input = "", ""
		m.inputError = ""
		m.filterClearPending = false

		return nil
	case keyCtrlEnter, keySave:
		return m.commitInput()
	case keyEnter:
		if m.field == fieldNote {
			m.input += "\n"

			return nil
		}

		return m.commitInput()
	case keyBackspace:
		if m.field == fieldSearch && m.filterClearPending {
			m.input, m.filterClearPending = "", false

			return nil
		}
		runes := []rune(m.input)
		if len(runes) > 0 {
			m.input = string(runes[:len(runes)-1])
		}
	case "ctrl+u":
		m.input = ""
		m.filterClearPending = false
	case keyLeft, keyRight:
		m.adjustInlineInput(key.String() == keyRight)
	case " ":
		m.input += " "
		m.filterClearPending = false
	default:
		if key.Type == tea.KeyRunes {
			m.input += string(key.Runes)
			m.filterClearPending = false
		}
	}

	return nil
}

func (m *model) adjustInlineInput(increase bool) {
	step := calibrationStep
	switch m.field {
	case fieldGain:
		if m.browsingCues() {
			step = cueGainStep
		}
	case fieldStart, fieldEnd:
		step = float64(fineSeekMs)
	case fieldCalibrationTarget:
	default:
		return
	}
	value, err := strconv.ParseFloat(m.input, 64)
	if err != nil || math.IsInf(value, 0) || math.IsNaN(value) {
		return
	}
	if !increase {
		step = -step
	}
	m.input = strconv.FormatFloat(value+step, 'f', -1, 64)
}

func (m *model) beginClassificationEdit() {
	if m.selectedID() == "" || m.assigning {
		return
	}
	value, options, cue := m.doc.RecordingFamily(m.recordingID()), audio.AcousticFamilies(), ""
	if m.browsingCues() {
		cue = m.activeCue()
		value, options = m.doc.CueProfile(cue), m.doc.ClassificationProfiles()
	}
	m.form = &editorForm{ //nolint:exhaustruct // Choice draft applies explicitly.
		kind: formClassification, title: "CLASSIFICATION", cue: cue, errorField: -1,
		fields: []formField{{label: "Classification", value: value,
			help: "Left/Right chooses an authored classification.", unit: "", step: 0, options: options}},
	}
}

func (m *model) commitInput() tea.Cmd {
	m.inputError, m.statusIsError = "", false
	m.committingInput = true
	defer func() { m.committingInput = false }()
	command := m.commitInputValue()
	if m.inputError == "" {
		m.field, m.input = "", ""
		m.filterClearPending = false
	}

	return command
}

func (m *model) commitInputValue() tea.Cmd {
	selected := m.selectedID()
	field, value := m.field, strings.TrimSpace(m.input)
	if m.duplicate != nil {
		m.commitDuplicateField(field, value)

		return nil
	}
	switch field {
	case fieldLicense:
		m.report(m.doc.SetRecordingLicense(m.licenseRecordingID, value))
	case fieldCalibrationTarget:
		m.setCalibrationTarget(value)
	case fieldSearch:
		m.search = value
		m.refreshKeys()
		if selected != m.selectedID() {
			return m.loadSelection()
		}
	case "new":
		return m.createEmptyRecording(value)
	case fieldNote:
		recording := m.doc.Recording(m.recordingID())
		recording.Note = value
		m.report(m.doc.SetRecording(m.recordingID(), recording))
		m.refreshKeys()
	case "classification":
		m.commitClassification(value)
		m.refreshKeys()
	case "path":
		return m.enterPickerPath(value)
	case fieldKeywords:
		m.commitKeywords(value)
		m.refreshKeys()
	case fieldTrigger:
		cue := m.doc.Cue(m.activeCue())
		cue.HowToHear = value
		m.report(m.doc.SetCue(m.activeCue(), cue))
		m.refreshKeys()
	default:
		m.commitNumber(field, value)
	}
	if selected != m.selectedID() {
		return m.loadSelection()
	}

	return nil
}

func (m *model) commitClassification(value string) {
	if m.browsingCues() {
		m.report(m.doc.SetCueProfile(m.activeCue(), value))

		return
	}
	m.report(m.doc.SetRecordingFamily(m.recordingID(), value))
	if m.clip != nil {
		recording := m.doc.Recording(m.recordingID())
		if m.followAutoCalibration(&recording) {
			m.report(m.doc.SetRecording(m.recordingID(), recording))
			m.preview.SetGain(m.gain())
			m.refreshMixGains()
		}
	}
}

func (m *model) createEmptyRecording(recordingID string) tea.Cmd {
	if err := m.doc.AddRecording(recordingID); err != nil {
		m.report(err)

		return nil
	}
	m.search, m.filter = "", filterAll
	m.refreshKeys()
	m.selectID(recordingID)

	return m.loadSelection()
}

func (m *model) commitNumber(field, value string) {
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		m.rejectInput("Enter a finite numeric value.")

		return
	}
	switch field {
	case fieldGain:
		m.setGain(number)
	case fieldStart, fieldEnd:
		if m.clip == nil || number < 0 || number > float64(m.clip.DurationMs) || math.Trunc(number) != number {
			m.rejectInput("Enter whole milliseconds within the recording.")

			return
		}
		window := m.window()
		if field == fieldStart {
			window.StartMs = int64(number)
		} else {
			window.EndMs = int64(number)
		}
		m.setWindow(&window)
	}
}

func (m *model) selectID(id string) {
	for index, key := range m.keys {
		if key == id {
			m.cursor = index

			return
		}
	}
}

func (m *model) chooseFileOrAssignment() tea.Cmd {
	if m.selectedID() == "" {
		return nil
	}
	m.focus = focusBrowser
	if m.workspace == workspaceMoments && !m.assigning {
		if m.variantCue != "" {
			return m.beginAppendVariant()
		}
		m.assignmentState = m.currentBrowseState()
		m.assigning, m.assignmentCue = true, m.selectedID()
		m.search = ""
		m.filter = filterAll
		m.refreshKeys()
		m.selectID(m.doc.Cue(m.assignmentCue).PrimaryRecording())
		m.status = "Choose a recording: Enter assigns; Space previews/pauses; Escape cancels."

		return m.loadSelection()
	}

	return m.openPicker()
}

func (m *model) acceptAssignment() tea.Cmd {
	if !m.assigning {
		return nil
	}
	if m.itemStatus(m.selectedID()).State != audio.RecordingReady {
		m.status = "Choose a READY recording; wait for checking or repair its file first."

		return nil
	}
	var err error
	if m.assignmentAppend {
		err = m.doc.AppendCueRecording(m.assignmentCue, m.recordingID())
	} else {
		err = m.doc.SetCueRecordings(m.assignmentCue, []string{m.recordingID()})
	}
	if err != nil {
		m.report(err)

		return nil
	}
	m.status = "Recording assigned; Ctrl+s saves."

	return m.cancelAssignment()
}

func (m *model) cancelAssignment() tea.Cmd {
	m.assigning = false
	m.assignmentAppend = false
	m.assignmentCue = ""
	m.restoreBrowseState(m.assignmentState)

	return m.loadSelection()
}
