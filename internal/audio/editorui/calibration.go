package editorui

import (
	"fmt"
	"math"
	"strconv"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

const fieldCalibrationTarget = "calibration target"

type calibrationDraft struct {
	audio.CalibrationSuggestion
	recordingID string
	revision    uint64
}

type calibrationCache struct {
	clip   *audio.RecordingAudio
	window audio.PlaybackWindow
	family string
	target float64
	result audio.CalibrationSuggestion
	err    error
}

func (m *model) calibrationSuggestion(recording audio.Recording) (audio.CalibrationSuggestion, error) {
	window := audio.PlaybackWindow{StartMs: 0, EndMs: m.clip.DurationMs}
	if recording.PlaybackWindow != nil {
		window = *recording.PlaybackWindow
	}
	family := m.doc.RecordingFamily(m.recordingID())
	target := audio.DefaultCalibrationTarget(family)
	if recording.CalibrationTargetDB != nil {
		target = *recording.CalibrationTargetDB
	}
	if cached := m.calibrationCache; cached != nil && cached.clip == m.clip && cached.window == window &&
		cached.family == family && cached.target == target {
		return cached.result, cached.err
	}
	result, err := m.clip.SuggestCalibration(&window, family, &target)
	if err != nil {
		err = fmt.Errorf("suggest calibration: %w", err)
	}
	m.calibrationCache = &calibrationCache{clip: m.clip, window: window, family: family,
		target: target, result: result, err: err}

	return result, err
}

func (m *model) beginAutoCalibration() {
	if !m.selectedClipReady() || m.picker != nil || m.assigning || m.duplicate != nil {
		m.status = "Select a playable recording before recalibrating."

		return
	}
	suggestion, err := m.calibrationSuggestion(m.doc.Recording(m.recordingID()))
	if err != nil {
		m.report(err)

		return
	}
	m.autoGainDraft = &calibrationDraft{CalibrationSuggestion: suggestion,
		recordingID: m.recordingID(), revision: m.doc.Revision()}
	m.status = fmt.Sprintf("Apply %+.2f dB and enable AUTO? Enter confirms; Esc keeps calibration.", suggestion.GainDB)
}

func (m *model) autoCalibrationKey(key string) {
	switch key {
	case keyEscape:
		m.autoGainDraft = nil
		m.status = "Calibration unchanged."
	case keyEnter, "y":
		if m.autoGainDraft.recordingID != m.recordingID() || m.autoGainDraft.revision != m.doc.Revision() {
			m.status, m.statusIsError = "Calibration draft is stale. Escape and recalculate before applying.", true

			return
		}
		recording := m.doc.Recording(m.recordingID())
		gain := m.autoGainDraft.GainDB
		recording.CalibrationGainDB, recording.CalibrationPolicy = &gain, audio.CalibrationAuto
		if err := m.doc.SetRecording(m.recordingID(), recording); err != nil {
			m.report(err)

			return
		}
		m.autoGainDraft = nil
		m.preview.SetGain(m.gain())
		m.refreshMixGains()
		m.status = "AUTO enabled. Trim edits follow; Recordings +/- or g locks calibration. Ctrl+s saves."
	}
}

func (m *model) followAutoCalibration(recording *audio.Recording) bool {
	if recording.CalibrationMode() != audio.CalibrationAuto {
		return true
	}
	suggestion, err := m.calibrationSuggestion(*recording)
	if err != nil {
		m.report(err)

		return false
	}
	recording.CalibrationGainDB = &suggestion.GainDB

	return true
}

func (m *model) editCalibrationTarget() {
	if m.clip == nil || m.assigning || m.picker != nil {
		return
	}
	if m.doc.Recording(m.recordingID()).CalibrationMode() == audio.CalibrationManual {
		m.status = "Manual calibration is locked. Shift+A recalculates and enables AUTO first."

		return
	}
	target := audio.DefaultCalibrationTarget(m.doc.RecordingFamily(m.recordingID()))
	if saved := m.doc.Recording(m.recordingID()).CalibrationTargetDB; saved != nil {
		target = *saved
	}
	m.field, m.input = fieldCalibrationTarget, strconv.FormatFloat(target, 'f', 1, 64)
}

func (m *model) setCalibrationTarget(value string) {
	target, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(target) || math.IsInf(target, 0) || target < -60 || target > -6 {
		m.rejectInput("Enter a target between -60 and -6 dB.")

		return
	}
	recording := m.doc.Recording(m.recordingID())
	recording.CalibrationTargetDB = &target
	if m.followAutoCalibration(&recording) {
		m.report(m.doc.SetRecording(m.recordingID(), recording))
		m.preview.SetGain(m.gain())
		m.refreshMixGains()
	}
}

func (m *model) calibrationDetails(add func(string, string)) {
	recording := m.doc.Recording(m.recordingID())
	mode := recording.CalibrationMode()
	add("Calibration mode: ", mode)
	if mode == audio.CalibrationManual {
		add("Auto-gain: ", "Paused · manual override locked")
		add("Recalculate: ", "Shift+A · confirmation replaces the override")

		return
	}
	if m.clip == nil || m.picker != nil {
		add("Auto-gain: ", "Waiting for playable source")

		return
	}
	suggestion, err := m.calibrationSuggestion(recording)
	if err != nil {
		add("Auto-gain: ", "No suggestion · silent/too quiet region")

		return
	}
	add("Method: ", suggestion.Method)
	add("Measured / target: ", fmt.Sprintf("%.1f / %.1f dB", suggestion.MeasuredDB, suggestion.TargetDB))
	add("Suggested gain: ", fmt.Sprintf("%+.2f dB · Shift+A applies", suggestion.GainDB))
	add("Predicted sample peak: ", fmt.Sprintf("%.1f dBFS", suggestion.PeakDB+suggestion.GainDB))
	if suggestion.Limited {
		add("Headroom: ", "Limited by -3 dBFS peak ceiling or calibration gain range")
	}
	add("Target: ", "Shift+T edits · defaults are starting points, not listening approval")
}
