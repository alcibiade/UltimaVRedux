package audio

// GetConfiguration returns detached game preference state.
func (manager *GameAudioManager) GetConfiguration() *AudioConfiguration {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	return manager.config.Clone()
}
