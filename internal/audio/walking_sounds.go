package audio

import (
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// SurfaceType represents different walking surface types
type SurfaceType string

// Built-in surface names; new surfaces are authored in the routing manifest.
const (
	SurfaceGrass SurfaceType = "grass"
	SurfaceStone SurfaceType = "stone"
	SurfaceWood  SurfaceType = "wood"
	SurfaceMetal SurfaceType = "metal"
)

// WalkingSoundManager handles surface-based walking sound effects
// Cycles through sound effect enums to create realistic footstep patterns
type WalkingSoundManager struct {
	audioManager  *AudioManager
	enabled       bool
	stepCounter   int                                      // Single counter for all walking sound cycling
	surfaceSounds map[SurfaceType][]game_state.SoundEffect // Map of surface types to their sound variants
}

// NewWalkingSoundManager creates a new walking sound manager
func NewWalkingSoundManager(audioManager *AudioManager) *WalkingSoundManager {
	// Initialize surface sound mappings
	surfaceSounds := make(map[SurfaceType][]game_state.SoundEffect)
	effects := make(map[string]game_state.SoundEffect)
	for effect, name := range catalogEffectNames() {
		effects["sfx/"+name] = effect
	}
	for surface, cues := range productionLibrary().Routing.Surfaces {
		for _, cue := range cues {
			surfaceSounds[SurfaceType(surface)] = append(surfaceSounds[SurfaceType(surface)], effects[cue])
		}
	}

	return &WalkingSoundManager{
		audioManager:  audioManager,
		enabled:       true,
		stepCounter:   0,
		surfaceSounds: surfaceSounds,
	}
}

// getSurfaceTypeForTile determines the surface type based on tile
func (wsm *WalkingSoundManager) getSurfaceTypeForTile(tile *references.Tile) SurfaceType {
	routing := productionLibrary().Routing
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
	variantIndex := wsm.stepCounter % len(soundVariants)

	return soundVariants[variantIndex]
}

// PlayWalkingSound plays appropriate walking sound based on tile surface
// Uses stepCounter to cycle through sound variants automatically
func (wsm *WalkingSoundManager) PlayWalkingSound(tile *references.Tile) error {
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

	// Increment counter for next call (prevents overflow with modulo)
	wsm.stepCounter = (wsm.stepCounter + 1) % 1000

	// Play the sound using the existing enum-based system
	return wsm.audioManager.PlaySoundEffectAt(soundEffect, 0, 0, 0, 0)
}
