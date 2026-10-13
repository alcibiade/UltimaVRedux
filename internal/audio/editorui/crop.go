package editorui

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

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

// selectedClipReady binds editing and audition to the source actually decoded.
// A file-picker candidate is preview-only until its assignment is published.
func (m *model) selectedClipReady() bool {
	if m.clip == nil || m.clipRecordingID != m.recordingID() {
		return false
	}
	path, err := m.doc.RecordingPath(m.recordingID())

	return err == nil && path == m.clip.Path
}

func (m *model) validWindow(window audio.PlaybackWindow) bool {
	if window.StartMs < 0 || window.EndMs > m.clip.DurationMs || window.EndMs <= window.StartMs {
		m.rejectInput(fmt.Sprintf("Use 0 ≤ Start < End ≤ %d ms; the region was not changed.", m.clip.DurationMs))

		return false
	}

	return true
}

// setWindow shares bounds and transport behavior between authored and duplicate
// crops. Duplicate changes remain detached until Create publishes one edit.
func (m *model) setWindow(window *audio.PlaybackWindow) bool {
	if !m.selectedClipReady() || m.assigning || m.picker != nil {
		m.rejectInput("Wait for the selected recording to load before editing its region.")

		return false
	}
	region := audio.PlaybackWindow{StartMs: 0, EndMs: m.clip.DurationMs}
	if window != nil {
		region = *window
	}
	if !m.validWindow(region) {
		return false
	}
	m.statusIsError, m.inputError = false, ""
	if m.duplicate != nil {
		m.duplicate.window = region
	} else if !m.publishWindow(window) {
		return false
	}
	if m.statusIsError || !m.restartCroppedPreview() {
		return true // The edit succeeded; retain the distinct playback error.
	}
	m.status = "Updated shared recording region; Ctrl+s saves, u undoes."
	if m.duplicate != nil {
		m.status = "Updated duplicate draft region; c creates, Escape cancels."
	}

	return true
}

func (m *model) publishWindow(window *audio.PlaybackWindow) bool {
	recording := m.doc.Recording(m.recordingID())
	recording.PlaybackWindow = window
	if !m.followAutoCalibration(&recording) {
		return false
	}
	if err := m.doc.SetRecording(m.recordingID(), recording); err != nil {
		m.report(err)

		return false
	}
	m.restartMixRecording(m.recordingID(), m.clip)
	m.updateSelectedAvailability(loadedMsg{generation: m.generation, recordingID: m.recordingID(),
		clip: m.clip, err: nil, autoplay: false})

	return true
}

func (m *model) restartCroppedPreview() bool {
	if m.sceneFXPreview || !m.preview.Active() {
		return true
	}
	position, wasPlaying := m.preview.PositionMs(), m.preview.Playing()
	if !m.playStandalone() {
		return false
	}
	if err := m.preview.SeekMs(position); err != nil {
		m.report(err)
		m.pausePreview(true)

		return false
	}
	m.position = m.preview.PositionMs()
	m.pausePreview(!wasPlaying)

	return true
}

func (m *model) markPosition(start bool) {
	window, marker := m.window(), markerEnd
	if start {
		window.StartMs, marker = m.position, markerStart
	} else {
		window.EndMs = m.position
	}
	if m.setWindow(&window) {
		m.marker = marker
	}
}

func (m *model) nudge(delta int64) {
	if !m.selectedClipReady() {
		m.status = "Wait for the selected recording to load before seeking or moving a marker."

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
	position := max(0, min(m.clip.DurationMs, m.position+delta))
	voice := m.previewVoice()
	if err := voice.SeekMs(position); err != nil {
		m.report(err)

		return
	}
	m.position, m.statusIsError = position, false
	m.status = fmt.Sprintf("Source cursor: %d ms. [/]: select a trim marker.", position)
	if voice.Active() {
		m.position = voice.PositionMs()
		m.status = fmt.Sprintf("Preview cursor: %d ms (within the playing selection). Ctrl+x stops; [/]: marker.", m.position)
	}
}

func (m *model) suggestTrim() {
	if !m.selectedClipReady() {
		m.status = "Wait for the selected recording to load before suggesting a region."

		return
	}
	window, ok := m.clip.SuggestedWindow()
	if !ok {
		m.status = "No sound above −40 dB found; set markers manually."

		return
	}
	if m.setWindow(window) && !m.statusIsError {
		m.status = "Suggested trim applied to working copy (−40 dB, 10 ms padding). Audition it; u undoes."
	}
}
