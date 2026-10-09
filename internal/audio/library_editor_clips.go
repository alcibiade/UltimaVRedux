package audio

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// NextRecordingID preserves suffix padding and skips every existing identity.
func (e *LibraryEditor) NextRecordingID(sourceID string) (string, error) {
	if _, exists := e.library.Recordings[sourceID]; !exists {
		return "", fmt.Errorf("%w: unknown recording %s", ErrAudioLibrary, sourceID)
	}
	const defaultDigits = 2
	const firstDuplicate = 2
	base, digits, next := sourceID, defaultDigits, uint64(firstDuplicate)
	if split := strings.LastIndex(sourceID, "_"); split >= 0 {
		suffix := sourceID[split+1:]
		if number, err := strconv.ParseUint(suffix, 10, 64); err == nil {
			if number == ^uint64(0) {
				return "", fmt.Errorf("%w: recording suffix is too large", ErrAudioLibrary)
			}
			base, digits, next = sourceID[:split], len(suffix), number+1
		}
	}
	for {
		candidate := fmt.Sprintf("%s_%0*d", base, digits, next)
		if _, exists := e.library.Recordings[candidate]; !exists {
			return candidate, nil
		}
		if next == ^uint64(0) {
			return "", fmt.Errorf("%w: recording suffix is too large", ErrAudioLibrary)
		}
		next++
	}
}

// DuplicateClip creates metadata only. Bounds and optional cycle membership are
// validated and published together as one undoable edit; source bytes never move.
func (e *LibraryEditor) DuplicateClip(sourceID, newID string, window PlaybackWindow, appendCue string) error {
	recording, exists := e.library.Recordings[sourceID]
	if !exists || recording.File == nil {
		return fmt.Errorf("%w: source clip needs an audio file", ErrAudioLibrary)
	}
	if _, exists := e.library.Recordings[newID]; exists {
		return fmt.Errorf("%w: recording %s already exists", ErrAudioLibrary, newID)
	}
	recording = cloneAudioValue(recording)
	recording.PlaybackWindow = &window
	recording.Note = "" // Source provenance carries over, but clip-specific observations do not.
	if err := e.validateWindow(recording); err != nil {
		return err
	}
	if recording.CalibrationMode() == CalibrationAuto {
		path, err := e.RecordingPath(sourceID)
		if err != nil {
			return err
		}
		clip, err := LoadRecording(path)
		if err != nil {
			return err
		}
		suggestion, err := clip.SuggestCalibration(&window, e.RecordingFamily(sourceID), recording.CalibrationTargetDB)
		if err != nil {
			return err
		}
		recording.CalibrationGainDB = &suggestion.GainDB
	}
	next := cloneAudioValue(e.library)
	next.Recordings[newID] = recording
	family := next.Taxonomy.recordingFamily(sourceID)
	next.Taxonomy.RecordingFamilies[family] = append(next.Taxonomy.RecordingFamilies[family], newID)
	slices.Sort(next.Taxonomy.RecordingFamilies[family])
	if appendCue != "" {
		cue, exists := next.Cues[appendCue]
		if !exists || !strings.HasPrefix(appendCue, "sfx/") {
			return fmt.Errorf("%w: choose an SFX moment for clip cycling", ErrAudioLibrary)
		}
		next.Cues[appendCue] = cue.WithRecordings(append(cue.Recordings(), newID))
	}

	return e.replace(next)
}

// AppendCueRecording adds one explicit cycle member, retaining the existing order.
func (e *LibraryEditor) AppendCueRecording(cueID, recordingID string) error {
	cue, exists := e.library.Cues[cueID]
	if !exists {
		return fmt.Errorf("%w: unknown cue %s", ErrAudioLibrary, cueID)
	}
	ids := cue.Recordings()
	if slices.Contains(ids, recordingID) {
		return fmt.Errorf("%w: %s is already assigned", ErrAudioLibrary, recordingID)
	}

	return e.SetCue(cueID, cue.WithRecordings(append(ids, recordingID)))
}

// SetCueRecordings changes ordering or membership through production validation.
func (e *LibraryEditor) SetCueRecordings(cueID string, ids []string) error {
	return e.SetCue(cueID, e.Cue(cueID).WithRecordings(ids))
}
