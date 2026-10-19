package audio

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// SurfaceType names a game-authored footstep surface.
type SurfaceType string

// Built-in names coexist with additional surfaces in Redux's JSON routing.
const (
	SurfaceGrass SurfaceType = "grass"
	SurfaceStone SurfaceType = "stone"
	SurfaceWood  SurfaceType = "wood"
	SurfaceMetal SurfaceType = "metal"
)

// PlayWalkingSound retains Redux's shared surface-step sequence and committed-step timing.
func (manager *GameAudioManager) PlayWalkingSound(tile *references.Tile) error {
	manager.mutex.Lock()
	if manager.closed {
		manager.mutex.Unlock()

		return ErrAudioClosed
	}
	if !manager.enabled {
		manager.mutex.Unlock()

		return nil
	}
	routing := manager.library.document.Routing
	surface := routing.DefaultSurface
	if tile != nil && routing.Tiles[tileAudioKey(tile.Index)].Surface != "" {
		surface = routing.Tiles[tileAudioKey(tile.Index)].Surface
	}
	cues := routing.Surfaces[surface]
	if len(cues) == 0 {
		manager.mutex.Unlock()

		return nil
	}
	cue := cues[manager.steps%uint64(len(cues))]
	manager.steps++
	effect, known := manager.effects[cue]
	manager.mutex.Unlock()
	if !known {
		return fmt.Errorf("%w: unknown footstep cue %s", ErrAudioLibrary, cue)
	}

	return manager.PlaySoundEffect(effect)
}
