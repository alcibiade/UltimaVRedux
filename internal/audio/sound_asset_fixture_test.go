package audio

import "github.com/bradhannah/UltimaVRedux/internal/game_state"

// loadSoundAsset installs one fixture asset using the production decoder;
// normal loading resolves all recordings through loadConfiguredSoundAsset.
func (sep *SoundEffectPlayer) loadSoundAsset(effect game_state.SoundEffect, filePath string) error {
	metadata, _ := GetSoundEffectMetadata(effect)
	asset, err := sep.prepareSoundAsset(effect, metadata.RecordingID, filePath, soundPlaybackWindow(effect))
	if err != nil {
		return err
	}
	sep.soundAssets[effect] = []*SoundAsset{asset}

	return nil
}
