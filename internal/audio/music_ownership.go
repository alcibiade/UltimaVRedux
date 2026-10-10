package audio

import (
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

func isInstrumentEffect(effect game_state.SoundEffect) bool {
	return effect == game_state.SoundBardLute || effect == game_state.SoundTavernBardMusic ||
		(effect >= game_state.SoundHarpsichord0 && effect <= game_state.SoundHarpsichord9)
}

func (sep *SoundEffectPlayer) hasInstrumentMusic() bool {
	sep.mutex.RLock()
	defer sep.mutex.RUnlock()
	for _, sound := range sep.activeSounds {
		if isInstrumentEffect(sound.Effect) && sound.Player.IsPlaying() && sound.Player.Volume() > 0 {
			return true
		}
	}

	return false
}

func (asm *AmbientSoundManager) hasBardMusic() bool {
	asm.mu.Lock()
	defer asm.mu.Unlock()
	for key, sound := range asm.activeSounds {
		// Include the initial zero-envelope frame and the entire release tail.
		if strings.HasPrefix(string(key), "source/bard/") && sound.Player.IsPlaying() &&
			asm.effectiveVolume(sound.Volume) > 0 {
			return true
		}
	}

	return false
}
