package audio

// Legacy manager support remains for the authoring/runtime migration tests.
// Live game scenes construct GameAudioManager, which supplies the same game policy.
func (am *AudioManager) musicLibrary() *Library { return am.library }
func (am *AudioManager) musicReadLock()         { am.mutex.RLock() }
func (am *AudioManager) musicReadUnlock()       { am.mutex.RUnlock() }

func NewMusicManager(manager *AudioManager) *MusicManager {
	if manager == nil {
		panic("MusicManager requires AudioManager")
	}

	return newMusicManager(manager, manager.backgroundMusic, manager.clock.Now)
}
