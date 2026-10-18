package audio

import (
	"sync"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// WalkingSoundManager handles surface-based walking sound effects
// Cycles through sound effect enums to create realistic footstep patterns
type WalkingSoundManager struct {
	mutex        sync.Mutex
	audioManager *AudioManager
	enabled      bool
	// Shared sequence has no short reset that disrupts variant periods.
	stepCounter   uint64
	surfaceSounds map[SurfaceType][]game_state.SoundEffect // Map of surface types to their sound variants
}

// NewWalkingSoundManager creates a new walking sound manager
func NewWalkingSoundManager(audioManager *AudioManager) *WalkingSoundManager {
	// Initialize surface sound mappings
	surfaceSounds := make(map[SurfaceType][]game_state.SoundEffect)
	manager := new(WalkingSoundManager)
	manager.audioManager, manager.enabled, manager.surfaceSounds = audioManager, true, surfaceSounds
	if !audioManager.redux {
		return manager
	}
	effects := make(map[string]game_state.SoundEffect)
	for effect, name := range catalogEffectNames() {
		effects["sfx/"+name] = effect
	}
	for surface, cues := range audioManager.library.document.Routing.Surfaces {
		for _, cue := range cues {
			surfaceSounds[SurfaceType(surface)] = append(surfaceSounds[SurfaceType(surface)], effects[cue])
		}
	}

	return manager
}

// getSurfaceTypeForTile determines the surface type based on tile
func (wsm *WalkingSoundManager) getSurfaceTypeForTile(tile *references.Tile) SurfaceType {
	routing := wsm.audioManager.library.document.Routing
	if tile != nil {
		if surface := routing.Tiles[tileAudioKey(tile.Index)].Surface; surface != "" {
			return SurfaceType(surface)
		}
	}

	return SurfaceType(routing.DefaultSurface)
}

// getSoundEffectForTile determines the appropriate walking sound based on tile type
// Uses stepCounter to cycle through sound variants dynamically
func (wsm *WalkingSoundManager) getSoundEffectForTile(tile *references.Tile) game_state.SoundEffect {
	surfaceType := wsm.getSurfaceTypeForTile(tile)

	// Get the sound variants for this surface type
	soundVariants, exists := wsm.surfaceSounds[surfaceType]
	if !exists || len(soundVariants) == 0 {
		return game_state.SoundEffect(-1) // Authored silent surface.
	}

	// Use modulo to cycle through available variants
	variantIndex := wsm.stepCounter % uint64(len(soundVariants))

	return soundVariants[variantIndex]
}

// PlayWalkingSound plays appropriate walking sound based on tile surface
// Uses stepCounter to cycle through sound variants automatically
func (wsm *WalkingSoundManager) PlayWalkingSound(tile *references.Tile) error {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()
	if !wsm.enabled {
		return nil
	}
	if wsm.audioManager == nil {
		panic("audioManager is nil - programming error in WalkingSoundManager initialization")
	}

	// Get appropriate sound effect variant based on current counter
	soundEffect := wsm.getSoundEffectForTile(tile)
	if soundEffect < 0 {
		return nil
	}

	wsm.stepCounter++

	// Play the sound using the existing enum-based system
	return wsm.audioManager.PlaySoundEffectAt(soundEffect, 0, 0, 0, 0)
}
