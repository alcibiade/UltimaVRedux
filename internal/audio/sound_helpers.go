package audio

import "github.com/bradhannah/UltimaVRedux/internal/game_state"

// SoundEffectMetadata is derived from the authored library. Playback request
// mode remains authoritative; filenames are resolved only by recording ID.
type SoundEffectMetadata struct {
	Name           string
	RecordingID    string
	VolumeModifier float64
}

//nolint:gochecknoglobals // Immutable metadata derived from the validated embedded library.
var soundEffectMetadataMap = librarySoundMetadata()

func librarySoundMetadata() map[game_state.SoundEffect]SoundEffectMetadata {
	library := productionLibrary()
	metadata := make(map[game_state.SoundEffect]SoundEffectMetadata)
	for effect, symbol := range catalogEffectNames() {
		cue := library.Cues["sfx/"+symbol]
		metadata[effect] = SoundEffectMetadata{
			Name: cue.Name, RecordingID: cue.PrimaryRecording(), VolumeModifier: cue.VolumeModifier,
		}
	}

	return metadata
}

// GetSoundEffectMetadata returns the code-backed cue's library metadata.
func GetSoundEffectMetadata(effect game_state.SoundEffect) (SoundEffectMetadata, bool) {
	metadata, exists := soundEffectMetadataMap[effect]

	return metadata, exists
}
