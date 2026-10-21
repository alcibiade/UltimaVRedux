package audio

import "github.com/bradhannah/UltimaVRedux/internal/references"

// GetCurrentTrack returns the name of the currently playing track
func (mm *MusicManager) GetCurrentTrack() string {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	return mm.policy.track
}

// GetCurrentLocation returns the location for which music is currently playing
func (mm *MusicManager) GetCurrentLocation() references.Location {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	return mm.currentLocation
}
