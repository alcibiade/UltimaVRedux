// Package game_state provides NPC light source calculation for magical creatures.
//
// Magical creatures that emit light include:
//   - Wisps: Ethereal energy beings with blue-white glow
//   - Seahorses: Magical sea creatures with bright blue glow
//   - Daemons: Demonic fire creatures with red glow
//   - Ghosts: Spectral beings with pale blue glow
package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// GetVisibleNPCLights returns light information for magical NPCs in the current location.
// Returns data needed for the lighting system to add NPCs as light sources.
func (gs *GameState) GetVisibleNPCLights() []map_state.NPCLightInfo {
	if gs.CurrentNPCAIController == nil {
		return nil
	}

	npcs := gs.CurrentNPCAIController.GetNpcs()
	if npcs == nil {
		return nil
	}

	var npcLights []map_state.NPCLightInfo

	// Check all enemies for light-emitting creatures
	enemies := npcs.GetEnemies()
	for i := range enemies {
		enemy := &enemies[i]
		if !enemy.IsVisible() {
			continue
		}

		// Check if this enemy's floor matches the player's floor (for multi-floor maps)
		if enemy.Floor() != gs.MapState.PlayerLocation.Floor {
			continue
		}

		lightInfo := gs.getNPCLightInfoIfEmitsLight(enemy)
		if lightInfo != nil {
			npcLights = append(npcLights, *lightInfo)
		}
	}

	return npcLights
}

// getNPCLightInfoIfEmitsLight checks if an NPC enemy emits light and returns its light info.
// Returns nil if the NPC does not emit light.
func (gs *GameState) getNPCLightInfoIfEmitsLight(enemy *map_units.NPCEnemy) *map_state.NPCLightInfo {
	spriteIndex := enemy.GetKeySpriteIndex()

	// Check against known light-emitting creature ranges
	switch {
	case isWispSprite(spriteIndex):
		return &map_state.NPCLightInfo{
			Position:    enemy.Pos(),
			MarkerIndex: map_state.WispMarkerIndex,
			LightRadius: map_state.WispLightDistance,
		}
	case isSeahorseSprite(spriteIndex):
		return &map_state.NPCLightInfo{
			Position:    enemy.Pos(),
			MarkerIndex: map_state.SeahorseMarkerIndex,
			LightRadius: map_state.SeahorseLightDistance,
		}
	case isDaemonSprite(spriteIndex):
		return &map_state.NPCLightInfo{
			Position:    enemy.Pos(),
			MarkerIndex: map_state.DaemonMarkerIndex,
			LightRadius: map_state.DaemonLightDistance,
		}
	case isGhostSprite(spriteIndex):
		return &map_state.NPCLightInfo{
			Position:    enemy.Pos(),
			MarkerIndex: map_state.GhostMarkerIndex,
			LightRadius: map_state.GhostLightDistance,
		}
	default:
		return nil
	}
}

// Sprite range constants for light-emitting creatures.
// Each creature has 4 animation frames (e.g., Wisp1-Wisp4).
const (
	wispStartSprite     = indexes.Wisp1     // 468
	wispEndSprite       = indexes.Wisp1 + 3 // 471 (Wisp4)
	seahorseStartSprite = indexes.Seahorse1 // 384
	seahorseEndSprite   = indexes.Seahorse1 + 3
	daemonStartSprite   = indexes.Daemon1 // 472
	daemonEndSprite     = indexes.Daemon1 + 3
	ghostStartSprite    = indexes.Ghost1 // 412
	ghostEndSprite      = indexes.Ghost1 + 3
)

// isWispSprite returns true if the sprite index is any Wisp animation frame.
func isWispSprite(spriteIndex indexes.SpriteIndex) bool {
	return spriteIndex >= wispStartSprite && spriteIndex <= wispEndSprite
}

// isSeahorseSprite returns true if the sprite index is any Seahorse animation frame.
func isSeahorseSprite(spriteIndex indexes.SpriteIndex) bool {
	return spriteIndex >= seahorseStartSprite && spriteIndex <= seahorseEndSprite
}

// isDaemonSprite returns true if the sprite index is any Daemon animation frame.
func isDaemonSprite(spriteIndex indexes.SpriteIndex) bool {
	return spriteIndex >= daemonStartSprite && spriteIndex <= daemonEndSprite
}

// isGhostSprite returns true if the sprite index is any Ghost animation frame.
func isGhostSprite(spriteIndex indexes.SpriteIndex) bool {
	return spriteIndex >= ghostStartSprite && spriteIndex <= ghostEndSprite
}

// IsLightEmittingNPC returns true if the sprite index represents a creature that emits light.
// This is useful for external systems that need to check NPC light emission.
func IsLightEmittingNPC(spriteIndex indexes.SpriteIndex) bool {
	return isWispSprite(spriteIndex) ||
		isSeahorseSprite(spriteIndex) ||
		isDaemonSprite(spriteIndex) ||
		isGhostSprite(spriteIndex)
}

// GetNPCLightMarkerIndex returns the marker index for a light-emitting NPC sprite.
// Returns 0 if the sprite does not emit light.
func GetNPCLightMarkerIndex(spriteIndex indexes.SpriteIndex) indexes.SpriteIndex {
	switch {
	case isWispSprite(spriteIndex):
		return map_state.WispMarkerIndex
	case isSeahorseSprite(spriteIndex):
		return map_state.SeahorseMarkerIndex
	case isDaemonSprite(spriteIndex):
		return map_state.DaemonMarkerIndex
	case isGhostSprite(spriteIndex):
		return map_state.GhostMarkerIndex
	default:
		return 0
	}
}

// GetNPCLightRadius returns the light radius for a light-emitting NPC sprite.
// Returns 0 if the sprite does not emit light.
func GetNPCLightRadius(spriteIndex indexes.SpriteIndex) int {
	switch {
	case isWispSprite(spriteIndex):
		return map_state.WispLightDistance
	case isSeahorseSprite(spriteIndex):
		return map_state.SeahorseLightDistance
	case isDaemonSprite(spriteIndex):
		return map_state.DaemonLightDistance
	case isGhostSprite(spriteIndex):
		return map_state.GhostLightDistance
	default:
		return 0
	}
}
