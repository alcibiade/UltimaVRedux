package audio

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
	for _, sound := range asm.activeSounds {
		if sound.emitter == "bard" && sound.Player.IsPlaying() && asm.effectiveVolume(sound.Volume) > 0 {
			return true
		}
	}
	return false
}
