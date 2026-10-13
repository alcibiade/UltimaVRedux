package audio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Recording is the production recording schema, shared with the editor.
type Recording = libraryRecording

// Cue is the production cue schema, shared with the editor.
type Cue = libraryCue

const editorHistoryLimit = 100

// LibraryEditor owns a disk-backed working document. Its caller serializes edits
// and saves; detached file decoding can run in background commands.
type LibraryEditor struct {
	path     string
	files    fs.FS
	library  *audioLibrary
	baseline *audioLibrary
	saved    []byte
	undo     []*audioLibrary
	redo     []*audioLibrary
	revision uint64
}

// OpenLibraryEditor reads the real manifest, without consulting the embedded cache.
func OpenLibraryEditor(path string) (*LibraryEditor, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve audio library: %w", err)
	}
	data, err := os.ReadFile(absolute) //nolint:gosec // Explicit developer-selected authoring file.
	if err != nil {
		return nil, fmt.Errorf("open audio library: %w", err)
	}
	files := os.DirFS(filepath.Dir(filepath.Dir(absolute)))
	library, err := parseAudioLibraryDocument(data)
	if err != nil {
		return nil, err
	}

	return &LibraryEditor{path: absolute, files: files, library: library,
		baseline: library, saved: data, undo: nil, redo: nil, revision: 0}, nil
}

// Path returns the absolute manifest filename.
func (e *LibraryEditor) Path() string { return e.path }

// Revision identifies authored state, including Undo/Redo. Detached work must
// compare its starting revision before replacing the current working document.
func (e *LibraryEditor) Revision() uint64 { return e.revision }

// RecordingIDs returns sorted stable recording identities for browsing.
func (e *LibraryEditor) RecordingIDs() []string { return sortedKeys(e.library.Recordings) }

// CueIDs returns sorted stable production cue identities for browsing.
func (e *LibraryEditor) CueIDs() []string { return sortedKeys(e.library.Cues) }

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	return keys
}

// Recording returns a detached authoring record.
func (e *LibraryEditor) Recording(id string) Recording {
	return cloneAudioValue(e.library.Recordings[id])
}

// Cue returns a copy of one authored cue.
func (e *LibraryEditor) Cue(id string) Cue { return cloneAudioValue(e.library.Cues[id]) }

// Usages lists every production cue affected by editing a shared recording.
func (e *LibraryEditor) Usages(recordingID string) []string {
	var ids []string
	for id, cue := range e.library.Cues {
		if slices.Contains(cue.Recordings(), recordingID) {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)

	return ids
}

// RecordingPath returns a source file path, or an explicit missing-slot error.
func (e *LibraryEditor) RecordingPath(id string) (string, error) {
	recording, exists := e.library.Recordings[id]
	if !exists || recording.File == nil {
		return "", fmt.Errorf("%w: %s has no audio file", ErrAudioLibrary, id)
	}

	return filepath.Join(filepath.Dir(e.path), filepath.FromSlash(*recording.File)), nil
}

// SetRecording validates a replacement before changing the document or history.
func (e *LibraryEditor) SetRecording(recordingID string, recording Recording) error {
	if recording.File != nil && recording.PlaybackWindow != nil {
		if err := e.validateWindow(recording); err != nil {
			return err
		}
	}
	next := cloneAudioValue(e.library)
	next.Recordings[recordingID] = recording

	return e.replace(next)
}

// SetRecordingLicense changes provenance only, including for an empty or missing
// audio file. Metadata authoring does not decode, trim or recalibrate the source.
func (e *LibraryEditor) SetRecordingLicense(recordingID, license string) error {
	if _, exists := e.library.Recordings[recordingID]; !exists {
		return fmt.Errorf("%w: unknown recording %s", ErrAudioLibrary, recordingID)
	}
	next := cloneAudioValue(e.library)
	recording := next.Recordings[recordingID]
	recording.Provenance.License = strings.TrimSpace(license)
	next.Recordings[recordingID] = recording

	return e.replace(next)
}

// AddRecording creates a named, intentionally empty slot for a later import.
func (e *LibraryEditor) AddRecording(recordingID string) error {
	if _, exists := e.library.Recordings[recordingID]; exists {
		return fmt.Errorf("%w: recording %s already exists", ErrAudioLibrary, recordingID)
	}

	next := cloneAudioValue(e.library)
	next.Recordings[recordingID] = Recording{File: nil, CalibrationGainDB: nil, PlaybackWindow: nil,
		CalibrationPolicy: "", CalibrationTargetDB: nil,
		Provenance: unknownRecordingProvenance(), Note: "", SearchTerms: nil}
	next.Taxonomy.RecordingFamilies["unreviewed"] = append(next.Taxonomy.RecordingFamilies["unreviewed"], recordingID)
	slices.Sort(next.Taxonomy.RecordingFamilies["unreviewed"])

	return e.replace(next)
}

// Clone creates a detached working copy for a blocking save/import command.
func (e *LibraryEditor) Clone() *LibraryEditor {
	cloned := *e
	cloned.library = cloneAudioValue(e.library)
	cloned.saved = bytes.Clone(e.saved)
	cloned.undo = slices.Clone(e.undo)
	cloned.redo = slices.Clone(e.redo)

	return &cloned
}

// SetCue validates an assignment or gain before changing the working document.
func (e *LibraryEditor) SetCue(id string, cue Cue) error {
	next := cloneAudioValue(e.library)
	next.Cues[id] = cue

	return e.replace(next)
}

// RecordingFamily describes the actual acoustic content independently of cue uses.
func (e *LibraryEditor) RecordingFamily(id string) string {
	return e.library.Taxonomy.recordingFamily(id)
}

// CueProfile names the reusable semantic classification assigned to a moment.
func (e *LibraryEditor) CueProfile(id string) string { return e.library.Taxonomy.cueProfile(id) }

// CueClassification returns a detached description of the intended moment.
func (e *LibraryEditor) CueClassification(id string) CueClassification {
	return cloneAudioValue(e.library.Taxonomy.Profiles[e.CueProfile(id)])
}

// ClassificationProfiles lists the valid authored profile names.
func (e *LibraryEditor) ClassificationProfiles() []string {
	return sortedKeys(e.library.Taxonomy.Profiles)
}

// AcousticFamilies lists the valid recording family names.
func AcousticFamilies() []string { return slices.Clone(acousticFamilies) }

// SetRecordingFamily changes one recording's acoustic description atomically.
func (e *LibraryEditor) SetRecordingFamily(recordingID, family string) error {
	if _, exists := e.library.Recordings[recordingID]; !exists || !slices.Contains(acousticFamilies, family) {
		return fmt.Errorf("%w: unknown recording or acoustic family", ErrAudioLibrary)
	}
	if e.RecordingFamily(recordingID) == family {
		return nil
	}
	next := cloneAudioValue(e.library)
	moveTaxonomyID(next.Taxonomy.RecordingFamilies, recordingID, family)
	recording := next.Recordings[recordingID]
	if recording.File != nil && recording.CalibrationMode() == CalibrationAuto {
		path, err := e.RecordingPath(recordingID)
		if err != nil {
			return err
		}
		clip, err := LoadRecording(path)
		if err != nil {
			return err
		}
		suggestion, err := clip.SuggestCalibration(recording.PlaybackWindow, family, recording.CalibrationTargetDB)
		if err != nil {
			return fmt.Errorf("calibrate changed recording family: %w", err)
		}
		recording.CalibrationGainDB = &suggestion.GainDB
		next.Recordings[recordingID] = recording
	}

	return e.replace(next)
}

// SetCueProfile changes one moment's semantic intent atomically.
func (e *LibraryEditor) SetCueProfile(cueID, profile string) error {
	if _, exists := e.library.Cues[cueID]; !exists {
		return fmt.Errorf("%w: unknown cue %s", ErrAudioLibrary, cueID)
	}
	if _, exists := e.library.Taxonomy.Profiles[profile]; !exists {
		return fmt.Errorf("%w: unknown classification profile %s", ErrAudioLibrary, profile)
	}
	if e.CueProfile(cueID) == profile {
		return nil
	}
	next := cloneAudioValue(e.library)
	moveTaxonomyID(next.Taxonomy.CueProfiles, cueID, profile)

	return e.replace(next)
}

func moveTaxonomyID(groups map[string][]string, identity, target string) {
	for group, ids := range groups {
		groups[group] = slices.DeleteFunc(ids, func(candidate string) bool { return candidate == identity })
		if len(groups[group]) == 0 {
			delete(groups, group)
		}
	}
	groups[target] = append(groups[target], identity)
	slices.Sort(groups[target])
}

func (e *LibraryEditor) replace(next *audioLibrary) error {
	encoded, err := json.Marshal(next)
	if err != nil {
		return fmt.Errorf("invalid audio edit: %w", err)
	}
	if _, err := parseAudioLibraryDocument(encoded); err != nil {
		return err
	}
	if bytes.Equal(encoded, encodeAudioValue(e.library)) {
		return nil
	}
	e.undo = append(e.undo, e.library)
	if len(e.undo) > editorHistoryLimit {
		e.undo = e.undo[1:]
	}
	e.library, e.redo = cloneAudioValue(next), nil
	e.revision++

	return nil
}

// Undo restores the preceding document, without deleting imported source files.
func (e *LibraryEditor) Undo() bool {
	if len(e.undo) == 0 {
		return false
	}
	e.redo = append(e.redo, e.library)
	e.library = e.undo[len(e.undo)-1]
	e.undo = e.undo[:len(e.undo)-1]
	e.revision++

	return true
}

// Redo reapplies a document edit undone in this session.
func (e *LibraryEditor) Redo() bool {
	if len(e.redo) == 0 {
		return false
	}
	e.undo = append(e.undo, e.library)
	e.library = e.redo[len(e.redo)-1]
	e.redo = e.redo[:len(e.redo)-1]
	e.revision++

	return true
}

// Dirty compares authored values with the most recent saved document.
func (e *LibraryEditor) Dirty() bool {
	return !bytes.Equal(encodeAudioValue(e.baseline), encodeAudioValue(e.library))
}

// RecordingChanged reports edits to this recording since the last Save.
func (e *LibraryEditor) RecordingChanged(id string) bool {
	return e.baseline.Taxonomy.recordingFamily(id) != e.RecordingFamily(id) ||
		!bytes.Equal(encodeAudioValue(e.baseline.Recordings[id]), encodeAudioValue(e.library.Recordings[id]))
}

// CueChanged reports authored cue edits, independently of its recording's state.
func (e *LibraryEditor) CueChanged(id string) bool {
	return e.baseline.Taxonomy.cueProfile(id) != e.CueProfile(id) ||
		!bytes.Equal(encodeAudioValue(e.baseline.Cues[id]), encodeAudioValue(e.library.Cues[id]))
}

func encodeAudioValue(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err) // Internal snapshots contain only validated JSON schema values.
	}

	return data
}

//nolint:ireturn // The helper preserves the concrete input schema type while detaching JSON-backed pointers.
func cloneAudioValue[T any](value T) T {
	var clone T
	if err := json.Unmarshal(encodeAudioValue(value), &clone); err != nil {
		panic(err)
	}

	return clone
}
