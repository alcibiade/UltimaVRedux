package audio

import (
	"fmt"
	"math"
	"time"

	berad "github.com/bradhannah/UltimaVRedux/modules/berad-audio"
)

// PlaybackDefinition projects Redux-owned JSON playback properties only. Routing,
// taxonomy, provenance, credits and game identities remain in the application.
func PlaybackDefinition(document LibraryDocument) (berad.Definition, error) {
	definition := berad.Definition{Recordings: make(map[berad.RecordingID]berad.Recording),
		Cues: make(map[berad.CueID]berad.Cue)}
	for identity, source := range document.Recordings {
		var recording berad.Recording
		if source.File != nil {
			recording.File = *source.File
		}
		if source.CalibrationGainDB != nil {
			recording.CalibrationDB = *source.CalibrationGainDB
		}
		if source.PlaybackWindow != nil {
			const maximumMilliseconds = math.MaxInt64 / int64(time.Millisecond)
			window := source.PlaybackWindow
			if window.StartMs < 0 || window.EndMs <= window.StartMs || window.EndMs > maximumMilliseconds {
				return berad.Definition{}, fmt.Errorf("%w: recording %s window exceeds duration range", ErrAudioLibrary, identity)
			}
			recording.Window = &berad.Window{Start: time.Duration(window.StartMs) * time.Millisecond,
				End: time.Duration(window.EndMs) * time.Millisecond}
		}
		definition.Recordings[berad.RecordingID(identity)] = recording
	}
	for identity, source := range document.Cues {
		cue := berad.Cue{Recordings: make([]berad.RecordingID, 0, len(source.Recordings())), Gain: source.VolumeModifier}
		for _, recording := range source.Recordings() {
			cue.Recordings = append(cue.Recordings, berad.RecordingID(recording))
		}
		definition.Cues[berad.CueID(identity)] = cue
	}

	return definition, nil
}

// PlaybackLibrary lazily freezes one Berad Audio projection shared by this
// project's managers/lanes. It never opens production defaults for another project.
func (library *Library) PlaybackLibrary() (*berad.Library, error) {
	library.playbackOnce.Do(func() {
		definition, err := PlaybackDefinition(library.Document())
		if err != nil {
			library.playbackError = err

			return
		}
		library.playback, library.playbackError = berad.NewLibrary(definition,
			encodedLibraryFS{library: library}, max(1, library.encodedBytes))
	})
	if library.playbackError != nil {
		return nil, fmt.Errorf("freeze module playback library: %w", library.playbackError)
	}

	return library.playback, nil
}
