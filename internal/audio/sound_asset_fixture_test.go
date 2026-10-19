package audio

import "github.com/bradhannah/UltimaVRedux/internal/game_state"

func isPlaceholderSound(effect game_state.SoundEffect) bool {
	metadata, exists := GetSoundEffectMetadata(effect)
	return exists && productionLibrary().Recordings[metadata.RecordingID].File == nil
}
