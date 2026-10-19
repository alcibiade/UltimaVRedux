package audio

import "sync"

// MusicTrackMetadata is game-owned authoring/display data, not a playback engine.
type MusicTrackMetadata struct {
	Name           string
	RecordingID    string
	VolumeModifier float64
	Description    string
}

//nolint:gochecknoglobals // Immutable projection of the game's embedded manifest.
var musicTrackMetadataMap = sync.OnceValue(libraryMusicMetadata)

func libraryMusicMetadata() map[string]MusicTrackMetadata {
	metadata := make(map[string]MusicTrackMetadata)
	for _, track := range musicTrackIDs() {
		cue := productionLibrary().Cues["music/track/"+track]
		metadata[track] = MusicTrackMetadata{Name: cue.Name, RecordingID: cue.RecordingID,
			VolumeModifier: cue.VolumeModifier, Description: cue.HowToHear}
	}

	return metadata
}

// GetMusicTrackMetadata returns the game's authored track label/assignment.
func GetMusicTrackMetadata(track string) (MusicTrackMetadata, bool) {
	metadata, exists := musicTrackMetadataMap()[track]

	return metadata, exists
}
