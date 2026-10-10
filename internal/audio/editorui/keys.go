package editorui

import (
	"math"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

const (
	calibrationStep    = 0.5
	cueGainStep        = 0.05
	minimumCalibration = -48.0
	maximumCalibration = 12.0
	fineSeekMs         = 10
	coarseSeekMs       = 100
)

func (m *model) key(key tea.KeyMsg) tea.Cmd {
	if key.String() == "x" && m.field == "" && (m.form == nil || !m.form.editing) &&
		(m.routing.menu == nil || !m.routing.menu.editingSearch) {
		m.stopAll()
		m.status = "All playback stopped."

		return nil
	}
	if m.busy {
		m.status = "Finishing file operation…"

		return nil
	}
	if key.String() == "ctrl+c" {
		m.confirmQuit = true

		return m.quit()
	}
	if m.confirmQuit {
		if key.String() == "y" {
			return tea.Quit
		}
		m.confirmQuit = false

		return nil
	}
	if command, handled := m.modalKey(key); handled {
		return command
	}
	if m.routing.menu != nil {
		return m.routingMenuKey(key)
	}
	if m.routingWorkspace() {
		return m.routingKey(key.String())
	}
	if m.focus == focusMix {
		return m.mixKey(key)
	}
	if m.focus == focusDetails && m.scrollDetails(key.String()) {
		return nil
	}
	if m.focus == focusWave && (key.String() == "up" || key.String() == keyDown) {
		return nil
	}

	return m.browserKey(key.String())
}

func (m *model) modalKey(key tea.KeyMsg) (tea.Cmd, bool) {
	if m.field != "" {
		return m.editInput(key), true
	}
	if m.infoDialog != nil {
		m.infoKey(key.String())

		return nil, true
	}
	if m.volumeControls {
		m.previewVolumeKey(key.String())

		return nil, true
	}
	if m.routing.menu != nil && m.routing.menu.editingSearch {
		return m.routingMenuKey(key), true
	}
	if m.form != nil && m.routing.menu == nil {
		return m.formKey(key), true
	}
	if m.pendingAmbient != nil {
		m.ambientReplacementKey(key.String())

		return nil, true
	}
	if m.autoGainDraft != nil {
		m.autoCalibrationKey(key.String())

		return nil, true
	}
	if key.String() == "v" {
		m.openPreviewVolumes()

		return nil, true
	}
	if m.duplicate != nil {
		return m.duplicateKey(key.String()), true
	}
	if m.picker != nil {
		return m.pickerKey(key), true
	}
	if m.variantCue != "" && !m.assigning && m.focus != focusMix {
		return m.variantKey(key.String()), true
	}

	return nil, false
}

//nolint:cyclop,funlen,gocyclo,maintidx // One explicit keyboard dispatch table keeps bindings auditable.
func (m *model) browserKey(key string) tea.Cmd {
	switch key {
	case "q":
		return m.quit()
	case "ctrl+s":
		return m.save()
	case "up", "k":
		return m.move(-1)
	case keyDown, "j":
		return m.move(1)
	case keyPageUp, keyShiftUp, keyPreviousPage:
		m.browserStart = m.browserWindowStart() - m.listHeight()

		return m.move(-m.listHeight())
	case keyPageDown, keyShiftDown, keyNextPage:
		m.browserStart = m.browserWindowStart() + m.listHeight()

		return m.move(m.listHeight())
	case keyTab:
		return m.selectWorkspace((m.workspace + 1) % workspaceCount)
	case keyShiftTab:
		return m.selectWorkspace((m.workspace + workspaceCount - 1) % workspaceCount)
	case "1":
		return m.switchWorkspace(false)
	case "2":
		return m.switchWorkspace(true)
	case "3":
		return m.selectWorkspace(workspaceTiles)
	case "4":
		return m.selectWorkspace(workspaceLocations)
	case "U":
		return m.openRoutingUsages()
	case "B":
		return m.editAmbientProfile()
	case keyCtrlA:
		m.openNewAmbientForm()
	case "ctrl+f":
		selected := m.selectedID()
		m.filter = (m.filter + 1) % filterCount
		m.refreshKeys()
		if selected != m.selectedID() {
			return m.loadSelection()
		}
	case "f5":
		return m.recheckFiles()
	case "w":
		m.focus = focusWave
	case "d":
		m.soundscapeVisible = false
		m.focus = focusDetails
	case "/":
		m.beginSearch()
	case "K":
		m.beginMetadataEdit(false)
	case "H":
		m.beginMetadataEdit(true)
	case keyEnter, "p":
		if m.assigning && key == keyEnter {
			return m.acceptAssignment()
		}
		m.play()
	case "V":
		return m.openVariants()
	case "D":
		m.beginDuplicate()
	case " ":
		m.togglePreview()
	case "x":
		m.stopAll()
	case "S":
		m.fullSource = !m.fullSource
		m.play()
	case "l":
		m.looping = !m.looping
		m.play()
	case "+", "=":
		m.adjustGain(1)
	case "-":
		m.adjustGain(-1)
	case "g":
		m.field, m.input = fieldGain, ""
	case "ctrl+l":
		m.beginLicenseEdit()
	case "v":
		m.openPreviewVolumes()
	case "i":
		m.focus = focusWave
		m.markPosition(true)
	case "o":
		m.focus = focusWave
		m.markPosition(false)
	case "[":
		m.marker = markerStart
		m.focus = focusWave
	case "]":
		m.marker = markerEnd
		m.focus = focusWave
	case "s":
		m.focus = focusWave
		m.field, m.input = fieldStart, strconv.FormatInt(m.window().StartMs, 10)
	case "e":
		m.focus = focusWave
		m.field, m.input = fieldEnd, strconv.FormatInt(m.window().EndMs, 10)
	case keyLeft:
		m.nudge(-fineSeekMs)
	case "right":
		m.nudge(fineSeekMs)
	case "shift+left":
		m.nudge(-coarseSeekMs)
	case "shift+right":
		m.nudge(coarseSeekMs)
	case "z":
		m.zoom = !m.zoom
	case "t":
		m.suggestTrim()
	case "r":
		m.setWindow(nil)
	case "u":
		return m.history(false)
	case "ctrl+r":
		return m.history(true)
	case "f":
		return m.chooseFileOrAssignment()
	case "a":
		m.addMix()
	case "m":
		m.soundscapeVisible, m.focus = true, focusMix
	case "n":
		if m.browsingCues() {
			return m.previewNextClip()
		}
		if m.workspace == workspaceRecordings {
			m.field, m.input = "new", ""
		}
	case "N":
		m.field, m.input = fieldNote, m.doc.Recording(m.recordingID()).Note
	case "C":
		m.beginClassificationEdit()
	case "A":
		m.beginAutoCalibration()
	case "T":
		m.editCalibrationTarget()
	case "y":
		return m.acceptAssignment()
	case keyEscape:
		if m.assigning {
			return m.cancelAssignment()
		}
		if m.routing.hasReturn {
			return m.returnFromRoutingJump()
		}
		m.marker = 0
		m.focus = focusBrowser
	case "?":
		m.openHelp()
	}

	return nil
}

func (m *model) move(delta int) tea.Cmd {
	if m.routingWorkspace() {
		m.routing.footsteps = false
	}
	m.cursor = max(0, min(m.cursor+delta, len(m.keys)-1))
	m.browserStart = m.browserWindowStart()
	m.marker = 0
	m.detailsScroll = 0

	return m.loadSelection()
}

func (m *model) quit() tea.Cmd {
	if m.doc.Dirty() || m.field != "" || m.form != nil {
		m.confirmQuit = true
		m.status = "Unsaved changes: y discards and quits; any other key cancels. Ctrl+s saves."

		return nil
	}

	return tea.Quit
}

func (m *model) history(redo bool) tea.Cmd {
	if redo {
		m.doc.Redo()
	} else {
		m.doc.Undo()
	}
	m.refreshKeys()
	m.clearSoundscape()

	return m.recheckFiles()
}

func (m *model) adjustGain(direction float64) {
	if m.browsingCues() {
		cue := m.doc.Cue(m.selectedID())
		m.setGain(math.Max(0, math.Min(1, cue.VolumeModifier+direction*cueGainStep)))

		return
	}
	recording := m.doc.Recording(m.recordingID())
	value := 0.0
	if recording.CalibrationGainDB != nil {
		value = *recording.CalibrationGainDB
	}
	m.setGain(math.Max(minimumCalibration, math.Min(maximumCalibration, value+direction*calibrationStep)))
}

func (m *model) setGain(value float64) {
	if m.assigning {
		m.status = "Confirm the assignment before editing."

		return
	}
	if m.browsingCues() {
		cue := m.doc.Cue(m.selectedID())
		cue.VolumeModifier = value
		m.report(m.doc.SetCue(m.selectedID(), cue))
	} else {
		recording := m.doc.Recording(m.recordingID())
		recording.CalibrationGainDB = &value
		recording.CalibrationPolicy = audio.CalibrationManual
		m.report(m.doc.SetRecording(m.recordingID(), recording))
	}
	m.preview.SetGain(m.gain())
	m.refreshMixGains()
}

func (m *model) window() audio.PlaybackWindow {
	if m.duplicate != nil {
		return m.duplicate.window
	}
	window := audio.PlaybackWindow{StartMs: 0, EndMs: 0}
	if m.clip != nil {
		window.EndMs = m.clip.DurationMs
	}
	if saved := m.doc.Recording(m.recordingID()).PlaybackWindow; saved != nil {
		window = *saved
	}

	return window
}

func (m *model) setWindow(window *audio.PlaybackWindow) {
	if m.clip == nil || m.assigning {
		m.status = "Select a playable recording before editing its region."

		return
	}
	recording := m.doc.Recording(m.recordingID())
	recording.PlaybackWindow = window
	if !m.followAutoCalibration(&recording) {
		return
	}
	if err := m.doc.SetRecording(m.recordingID(), recording); err != nil {
		m.report(err)

		return
	}
	wasPlaying := m.preview.Playing()
	if !m.sceneFXPreview && m.preview.Active() {
		m.play()
		m.preview.Pause(!wasPlaying)
	}
	m.restartMixRecording(m.recordingID(), m.clip)
	m.preview.SetGain(m.gain())
	m.updateSelectedAvailability(loadedMsg{generation: m.generation, clip: m.clip, err: nil, autoplay: false})
	m.status = "Updated shared recording region; Ctrl+s saves, u undoes."
}

func (m *model) markPosition(start bool) {
	window := m.window()
	if start {
		window.StartMs = m.position
	} else {
		window.EndMs = m.position
	}
	m.setWindow(&window)
}

func (m *model) nudge(delta int64) {
	if m.clip == nil {
		return
	}
	if m.marker != 0 {
		window := m.window()
		if m.marker == markerStart {
			window.StartMs += delta
		} else {
			window.EndMs += delta
		}
		m.setWindow(&window)

		return
	}
	m.position = max(0, min(m.clip.DurationMs, m.position+delta))
	m.report(m.previewVoice().SeekMs(m.position))
}

func (m *model) suggestTrim() {
	if m.clip == nil {
		return
	}
	window, ok := m.clip.SuggestedWindow()
	if !ok {
		m.status = "No sound above −40 dB found; set markers manually."

		return
	}
	m.setWindow(window)
	m.status = "Suggested trim applied to working copy (−40 dB, 10 ms padding). Audition it; u undoes."
}
