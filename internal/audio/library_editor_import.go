package audio

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ImportRecording validates and copies a file into the library before assigning
// it. Existing repository files are referenced in place. Outside files receive
// content-qualified names, so replacing a source never overwrites shared audio.
// Imported files remain on disk after Undo; Git exposes them for review.
func (e *LibraryEditor) ImportRecording(recordingID, path string) error {
	previous, exists := e.library.Recordings[recordingID]
	if !exists {
		return fmt.Errorf("%w: select or create a recording first", ErrAudioLibrary)
	}
	clip, err := LoadRecording(path)
	if err != nil {
		return err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve recording path: %w", err)
	}
	relative, err := filepath.Rel(filepath.Dir(e.path), absolute)
	if err != nil {
		return fmt.Errorf("resolve relative recording path: %w", err)
	}
	relative = filepath.ToSlash(relative)
	if !validRecordingPath(relative) {
		relative, err = e.copyRecording(recordingID, previous, clip)
		if err != nil {
			return err
		}
	}
	if previous.File != nil && *previous.File == relative {
		return nil
	}
	provenance := unknownRecordingProvenance()
	provenance.OriginalFilename = filepath.Base(path)
	for _, known := range e.library.Recordings {
		if known.File != nil && *known.File == relative {
			provenance = cloneAudioValue(known.Provenance)

			break
		}
	}

	replacement := previous
	replacement.File, replacement.PlaybackWindow = &relative, nil
	replacement.Provenance, replacement.Note = provenance, ""
	// Manual overrides belong to the recording identity and survive replacement.
	// Auto-managed replacements are measured before publishing the single edit.
	if replacement.CalibrationMode() == CalibrationAuto {
		suggestion, err := clip.SuggestCalibration(nil, e.RecordingFamily(recordingID), replacement.CalibrationTargetDB)
		if err != nil {
			return err
		}
		replacement.CalibrationGainDB = &suggestion.GainDB
	}

	return e.SetRecording(recordingID, replacement)
}

func (e *LibraryEditor) copyRecording(recordingID string, previous Recording, clip *RecordingAudio) (string, error) {
	directory := "sounds/imported"
	if previous.File != nil && strings.HasPrefix(*previous.File, "music/") {
		directory = "music/imported"
	}
	hash := sha256.Sum256(clip.data)
	name := fmt.Sprintf("%s_%x%s", recordingID, hash[:6], strings.ToLower(filepath.Ext(clip.Path)))
	relative := directory + "/" + name
	path := filepath.Join(filepath.Dir(e.path), filepath.FromSlash(relative))
	const directoryMode = 0o750
	const recordingMode = 0o644
	if err := os.MkdirAll(filepath.Dir(path), directoryMode); err != nil {
		return "", fmt.Errorf("create recording directory: %w", err)
	}
	//nolint:gosec // Destination is built from a validated recording ID and content hash under the library root.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, recordingMode)
	if os.IsExist(err) {
		return matchImportedFile(path, relative, hash)
	}
	if err != nil {
		return "", fmt.Errorf("create imported recording: %w", err)
	}
	_, writeErr := file.Write(clip.data)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		removeErr := os.Remove(path)

		return "", fmt.Errorf("write imported recording: %w", errors.Join(writeErr, closeErr, removeErr))
	}

	return relative, nil
}

func matchImportedFile(path, relative string, expected [sha256.Size]byte) (string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // Validated library import destination.
	if err != nil {
		return "", fmt.Errorf("read existing import: %w", err)
	}
	if sha256.Sum256(data) != expected {
		return "", fmt.Errorf("%w: import destination already has different contents", ErrAudioLibrary)
	}

	return relative, nil
}
