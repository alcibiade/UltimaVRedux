package audio

import (
	"fmt"
	"io/fs"
	"math"
	"sync"

	"github.com/bradhannah/berad-sdk/modules/berad-audio/authoring"
)

const libraryManifestPath = "assets/library.json"

// LibraryManifestRelativePath is the editable source manifest within a checkout.
const LibraryManifestRelativePath = "internal/audio/assets/library.json"

// ErrAudioLibrary identifies invalid authored library data, not an absent optional
// cue. It is Berad Audio's authoring error, so structural and Redux failures match.
var ErrAudioLibrary = authoring.ErrAudioLibrary

// The authored library is Berad Audio's schema-3 document. Berad Audio parses it
// and checks its structure; ReduxLibraryValidator adds the game's rules.
type (
	// LibraryDocument is the versioned authoring schema. Loading makes a detached
	// immutable snapshot; changing this value cannot change an existing runtime.
	LibraryDocument = authoring.LibraryDocument
	// LibraryValidator adds consumer completeness, vocabulary and routing rules to
	// structural validation. It receives a detached document and must be deterministic.
	LibraryValidator = authoring.LibraryValidator
	// Routing assigns cues to tiles, surfaces, emitters and the music, dungeon and
	// naval groups.
	Routing = authoring.Routing
	// AmbientProfile is one ambient cue's radius in tiles (0 is a background) and fades.
	AmbientProfile = authoring.AmbientProfile
	// TileAudio attaches audio to a canonical animation family or a static tile.
	TileAudio = authoring.TileAudio
	// Taxonomy groups identities under controlled, reusable classifications.
	Taxonomy = authoring.Taxonomy
	// TileTarget is a game-derived animation-family authoring descriptor.
	TileTarget = authoring.TileTarget
	// RecordingProvenance records a recording's source, licence and processing.
	RecordingProvenance = authoring.RecordingProvenance
	// PlaybackWindow is a persisted source-relative excerpt in milliseconds (EndMs exclusive).
	PlaybackWindow = authoring.PlaybackWindow
)

// SelectionCycle selects the next recording after each successful cue start.
const SelectionCycle = authoring.SelectionCycle

// The embedded manifest is immutable for the lifetime of this build. It is not
// a hot-reloaded user preference. The terminal editor opens a disk-backed copy.
//
//nolint:gochecknoglobals // One validated immutable embedded library shared by playback and catalog readers.
var productionLibrary = sync.OnceValue(func() *LibraryDocument {
	data, err := soundFiles.ReadFile(libraryManifestPath)
	if err != nil {
		panic(fmt.Sprintf("read embedded audio library: %v", err))
	}
	library, err := parseAudioLibrary(data, soundFiles)
	if err != nil {
		panic(err)
	}

	return library
})

func parseAudioLibrary(data []byte, files fs.FS) (*LibraryDocument, error) {
	library, err := parseAudioLibraryDocument(data)
	if err != nil {
		return nil, err
	}
	for recordingID, recording := range library.Recordings {
		if recording.File == nil {
			continue
		}
		info, err := fs.Stat(files, "assets/"+*recording.File)
		if err != nil {
			return nil, fmt.Errorf("%w: recording %s: %w", ErrAudioLibrary, recordingID, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("%w: recording %s points to a directory", ErrAudioLibrary, recordingID)
		}
	}

	return library, nil
}

// The editor must be able to open a structurally valid document whose files
// need repair. Production initialization and Save also validate disk references.
func parseAudioLibraryDocument(data []byte) (*LibraryDocument, error) {
	document, err := authoring.ParseDocument(data, ReduxLibraryValidator)
	if err != nil {
		return nil, fmt.Errorf("parse audio library: %w", err)
	}

	return &document, nil
}

func assetRecordingPath(library *LibraryDocument, id string) string {
	recording, exists := library.Recordings[id]
	if !exists {
		panic("unknown audio recording ID: " + id)
	}
	if recording.File == nil {
		return ""
	}

	return "assets/" + *recording.File
}

// An absent calibration is unreviewed and plays at unity. Mute is handled by
// an explicit zero-amplitude gain elsewhere, never by a finite decibel value.
func recordingCalibration(library *LibraryDocument, id string) float64 {
	if db := library.Recordings[id].CalibrationGainDB; db != nil {
		const decibelBase = 10
		const amplitudeDecibelFactor = 20

		return math.Pow(decibelBase, *db/amplitudeDecibelFactor)
	}

	return 1
}
