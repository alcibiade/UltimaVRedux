package audio

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

func TestWalkingSoundCycling(t *testing.T) {
	// Create a mock audio manager for testing
	audioManager, err := NewAudioManager()
	assert.NoError(t, err)

	wsm := NewWalkingSoundManager(audioManager)

	// Create a grass tile using a forest tile index that maps to grass sounds
	grassTile := &references.Tile{
		Index: indexes.Forest1, // Use an actual forest tile index
	}

	// Test surface type detection
	surfaceType := wsm.getSurfaceTypeForTile(grassTile)
	assert.Equal(t, SurfaceGrass, surfaceType, "Forest tile should map to grass surface")

	// Test that grass surface cycling works correctly
	// We'll test by checking the actual sound effects returned

	// Test cycling behavior - should alternate left/right for grass
	wsm.stepCounter = 0 // Even counter
	soundEffect1 := wsm.getSoundEffectForTile(grassTile)
	assert.Equal(t, game_state.SoundWalkGrassLeft, soundEffect1, "Even counter should give left foot")

	wsm.stepCounter = 1 // Odd counter
	soundEffect2 := wsm.getSoundEffectForTile(grassTile)
	assert.Equal(t, game_state.SoundWalkGrassRight, soundEffect2, "Odd counter should give right foot")

	wsm.stepCounter = 2 // Even again
	soundEffect3 := wsm.getSoundEffectForTile(grassTile)
	assert.Equal(t, game_state.SoundWalkGrassLeft, soundEffect3, "Should cycle back to left foot")
}

func TestStoneWalkingSoundVariants(t *testing.T) {
	// Create audio manager and walking sound manager
	audioManager, err := NewAudioManager()
	assert.NoError(t, err)

	wsm := NewWalkingSoundManager(audioManager)

	// Test that stone surface type detection works
	stoneTile := &references.Tile{Index: indexes.BrickFloor}
	surfaceType := wsm.getSurfaceTypeForTile(stoneTile)
	assert.Equal(t, SurfaceStone, surfaceType, "Brick floor should map to stone surface")

	// Test cycling through all 4 variants

	wsm.stepCounter = 0
	assert.Equal(t, game_state.SoundWalkStone1, wsm.getSoundEffectForTile(stoneTile))

	wsm.stepCounter = 1
	assert.Equal(t, game_state.SoundWalkStone2, wsm.getSoundEffectForTile(stoneTile))

	wsm.stepCounter = 2
	assert.Equal(t, game_state.SoundWalkStone3, wsm.getSoundEffectForTile(stoneTile))

	wsm.stepCounter = 3
	assert.Equal(t, game_state.SoundWalkStone4, wsm.getSoundEffectForTile(stoneTile))

	wsm.stepCounter = 4 // Should cycle back
	assert.Equal(t, game_state.SoundWalkStone1, wsm.getSoundEffectForTile(stoneTile))
}

func TestWoodWalkingSoundVariants(t *testing.T) {
	// Create audio manager and walking sound manager
	audioManager, err := NewAudioManager()
	assert.NoError(t, err)

	wsm := NewWalkingSoundManager(audioManager)

	// Test that wood surface type detection works
	woodTile := &references.Tile{Index: indexes.WoodFloor1}
	surfaceType := wsm.getSurfaceTypeForTile(woodTile)
	assert.Equal(t, SurfaceWood, surfaceType, "Wood floor should map to wood surface")

	// Test cycling through all 3 variants

	wsm.stepCounter = 0
	assert.Equal(t, game_state.SoundWalkWood1, wsm.getSoundEffectForTile(woodTile))

	wsm.stepCounter = 1
	assert.Equal(t, game_state.SoundWalkWood2, wsm.getSoundEffectForTile(woodTile))

	wsm.stepCounter = 2
	assert.Equal(t, game_state.SoundWalkWood3, wsm.getSoundEffectForTile(woodTile))

	wsm.stepCounter = 3 // Should cycle back
	assert.Equal(t, game_state.SoundWalkWood1, wsm.getSoundEffectForTile(woodTile))
}

func TestMetalWalkingSoundFixed(t *testing.T) {
	// Create audio manager and walking sound manager
	audioManager, err := NewAudioManager()
	assert.NoError(t, err)

	wsm := NewWalkingSoundManager(audioManager)

	// Test that metal surface type detection works
	metalTile := &references.Tile{Index: indexes.MetalFloor}
	surfaceType := wsm.getSurfaceTypeForTile(metalTile)
	assert.Equal(t, SurfaceMetal, surfaceType, "Metal floor should map to metal surface")

	// Test that metal returns its own sound effect
	wsm.stepCounter = 0
	metalSound := wsm.getSoundEffectForTile(metalTile)
	assert.Equal(t, game_state.SoundWalkMetal1, metalSound, "Metal should use its own sound effect")

	// Test that individual enum values have correct metadata
	metalMetadata, exists := GetSoundEffectMetadata(game_state.SoundWalkMetal1)
	assert.True(t, exists, "Metal walking sound metadata should exist")
	_, registered := productionLibrary().Recordings[metalMetadata.RecordingID]
	assert.True(t, registered, "Metal should resolve through the recording registry")
}
