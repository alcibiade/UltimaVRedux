package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// SpritePresentation describes sprite selection and terrain composition for an actor.
type SpritePresentation struct {
	Sprite                  indexes.SpriteIndex
	BackgroundIntegrated    bool
	SuppressTerrainUnderlay bool
}

// GetAvatarSpriteAtPosition returns the correct avatar presentation at the given position.
func (g *GameState) GetAvatarSpriteAtPosition(pos references.Position) SpritePresentation {
	// Check if we're in camping mode and avatar is sleeping
	if g.IsInCamping() && g.IsAvatarSleepingInCamp() {
		return SpritePresentation{Sprite: indexes.DeadBody} // Original Ultima V uses DEAD_PC for sleeping characters during camping
	}

	vehicle := g.PartyVehicle.GetVehicleDetails()
	currentMap := g.GetLayeredMapByCurrentLocation()
	baseTile := currentMap.GetTileTopMapOnlyTile(&pos)
	if vehicle.VehicleType == references.CarpetVehicle && baseTile != nil {
		// Tile-context sprite projection also substitutes IN_BED for a ridden carpet on LeftBed.
		//nolint:exhaustive // Only ladders and the bed head replace a boarded carpet.
		// See ref:TILE-0008.
		switch baseTile.Index {
		case indexes.LeftBed:
			return SpritePresentation{Sprite: indexes.SleepingInBed, BackgroundIntegrated: true}
		case indexes.LadderUp:
			return SpritePresentation{Sprite: indexes.KlimbLadderUp, BackgroundIntegrated: true}
		case indexes.LadderDown:
			return SpritePresentation{Sprite: indexes.KlimbLadderDown, BackgroundIntegrated: true}
		}
	}
	if vehicle.VehicleType != references.NoPartyVehicle {
		return SpritePresentation{Sprite: vehicle.GetBoardedSpriteIndex()}
	}

	if baseTile == nil {
		return SpritePresentation{Sprite: indexes.AvatarKeyIndex}
	}

	// Apply avatar sprite replacements based on the tile the avatar is standing on
	return g.GetNPCSpriteReplacement(baseTile.Index, indexes.AvatarKeyIndex, pos)
}

// GetNPCSpriteReplacement handles sprite replacements for NPCs/avatar based on the tile they're on.
// This includes ladders, beds, chairs, manacles, camping sleep, bard playing, etc.
// Per original world-state rules: Bard→Merchant conversion happens first, then tile-context sprite projection applies tile-based transformations.
//
// See ref:TILE-0008.
//
//nolint:cyclop // Tile-based switch naturally has many cases - matches original tile-context sprite projection function structure
func (g *GameState) GetNPCSpriteReplacement(
	baseTileIndex, npcSpriteIndex indexes.SpriteIndex,
	pos references.Position,
) SpritePresentation {
	// Check if we're in camping mode and this is a sleeping character
	if g.IsInCamping() && g.isCharacterSleepingInCamp(npcSpriteIndex) {
		return SpritePresentation{Sprite: indexes.DeadBody} // Original Ultima V uses DEAD_PC for sleeping characters during camping
	}

	// BANJO_PLAYER sprite switching (Original: original world-state rules)
	// When BANJO_PLAYER is on south-facing chair (ChairBackBack): show BardPlaying sprite (playing music)
	// When NOT on that chair: convert to Merchant sprite, then continue to tile-based checks
	// This matches original behavior where tile-context sprite projection is still called after bard→merchant conversion
	// See ref:TILE-0008.
	if npcSpriteIndex.IsBardPlaying() {
		if baseTileIndex == indexes.ChairBackBack {
			return SpritePresentation{
				Sprite:                  npcSpriteIndex,
				BackgroundIntegrated:    true,
				SuppressTerrainUnderlay: true,
			}
		}
		// Not seated on south-facing chair - convert to Merchant sprite, preserving animation frame
		// Then fall through to tile-based replacements (beds, ladders, manacles, etc.)
		frameOffset := npcSpriteIndex - indexes.BardPlayingKeyIndex
		npcSpriteIndex = indexes.MerchantKeyIndex + frameOffset
		// Continue to the tile checks below - don't return early
	}

	if baseTileIndex == indexes.LeftBed {
		if npcSpriteIndex.IsMagicCarpetUnboarded() {
			return SpritePresentation{Sprite: npcSpriteIndex}
		}

		return SpritePresentation{Sprite: indexes.SleepingInBed, BackgroundIntegrated: true}
	}

	if baseTileIndex == indexes.ChairBackRight {
		return SpritePresentation{Sprite: indexes.SitChairLeft, BackgroundIntegrated: true}
	}

	if baseTileIndex == indexes.ChairBackLeft {
		return SpritePresentation{Sprite: indexes.SitChairRight, BackgroundIntegrated: true}
	}

	if baseTileIndex == indexes.ChairBackForward {
		// Check for table with food above
		currentMap := g.GetLayeredMapByCurrentLocation()
		upPos := pos.GetPositionUp()
		upPosTile := currentMap.GetTopTile(upPos)
		if upPosTile != nil && (upPosTile.Is(indexes.TableFoodBoth) || upPosTile.Is(indexes.TableFoodBottom)) {
			// For now, return static sprite - we'll add animation support later
			return SpritePresentation{Sprite: indexes.SitChairUpEat1, BackgroundIntegrated: true}
		}

		return SpritePresentation{Sprite: indexes.SitChairUp, BackgroundIntegrated: true}
	}

	if baseTileIndex == indexes.ChairBackBack {
		// Check for table with food below
		currentMap := g.GetLayeredMapByCurrentLocation()
		downPos := pos.GetPositionDown()
		downPosTile := currentMap.GetTopTile(downPos)
		if downPosTile != nil && (downPosTile.Is(indexes.TableFoodBoth) || downPosTile.Is(indexes.TableFoodTop)) {
			// For now, return static sprite - we'll add animation support later
			return SpritePresentation{Sprite: indexes.SitChairDownEat1, BackgroundIntegrated: true}
		}

		return SpritePresentation{Sprite: indexes.SitChairDown, BackgroundIntegrated: true}
	}

	if baseTileIndex == indexes.LadderUp {
		return SpritePresentation{Sprite: indexes.KlimbLadderUp, BackgroundIntegrated: true}
	}

	if baseTileIndex == indexes.LadderDown {
		return SpritePresentation{Sprite: indexes.KlimbLadderDown, BackgroundIntegrated: true}
	}

	if baseTileIndex == indexes.Manacles {
		return SpritePresentation{Sprite: indexes.ManaclesPrisoner, BackgroundIntegrated: true}
	}

	if baseTileIndex == indexes.Stocks {
		return SpritePresentation{Sprite: indexes.PersonStocks1, BackgroundIntegrated: true}
	}

	return SpritePresentation{Sprite: npcSpriteIndex}
}

// IsAvatarSleepingInCamp checks if the avatar is sleeping during camping
func (g *GameState) IsAvatarSleepingInCamp() bool {
	if !g.IsInCamping() {
		return false
	}

	// During camping, the avatar sleeps unless they are the guard
	// We need to check if there's an active camping scene and if avatar is the guard
	// For now, assume avatar sleeps if we're in camping mode and no guard is avatar
	// The guard selection is not modelled, so this may need refinement once it is.
	return true // Simplified logic - avatar sleeps during camping unless they're guard
}

// isCharacterSleepingInCamp checks if a party member sprite represents a character sleeping during camping
func (g *GameState) isCharacterSleepingInCamp(npcSpriteIndex indexes.SpriteIndex) bool {
	if !g.IsInCamping() {
		return false
	}

	// Check if this is a party member sprite (not avatar, since avatar is handled separately)
	// Party members sleep during camping unless they are the guard
	if g.IsPartyMemberSprite(npcSpriteIndex) {
		// Guard selection is not modelled: all party members sleep.
		return true
	}

	return false
}

// IsPartyMemberSprite checks if the given sprite index represents a party member
func (g *GameState) IsPartyMemberSprite(spriteIndex indexes.SpriteIndex) bool {
	// Check if this sprite matches any of the party member sprites
	for i := range g.PartyState.GetCharacterCount() {
		character := g.PartyState.GetCharacter(i)
		if character.PartyMap == party_state.PartyMapActive {
			characterSpriteIndex := character.GetKeySpriteIndex()
			if spriteIndex == characterSpriteIndex {
				return true
			}
		}
	}

	return false
}

// isQuestItemNPCHidden checks if an NPC represents a quest item that should be hidden
// Quest item NPCs are now ALWAYS hidden because quest items are rendered as overlays
// Debug logging helps identify when NPCs are being hidden
func (g *GameState) isQuestItemNPCHidden(mapUnit map_units.MapUnit) bool {
	if mapUnit == nil {
		return false
	}

	// Check if this is a friendly NPC (quest items are NPCFriendly type)
	friendly, ok := mapUnit.(*map_units.NPCFriendly)
	if !ok {
		return false
	}

	// Get the sprite index for this NPC to identify which quest item it represents
	npcSpriteIndex := friendly.GetKeySpriteIndex()

	// Quest item NPCs are ALWAYS hidden now - they're rendered as overlays instead
	for _, questItem := range [...]struct {
		sprite indexes.SpriteIndex
		name   string
	}{{indexes.Crown, "Crown"}, {indexes.Sceptre, "Sceptre"}, {indexes.Amulet, "Amulet"}} {
		if npcSpriteIndex == questItem.sprite {
			logging.Debugf(logging.CategoryGameState,
				"Hiding %s NPC at position - using overlay rendering instead", questItem.name)

			return true
		}
	}

	return false
}

// IsQuestItemNPCHidden is a public wrapper for the UI layer to check quest item NPC visibility
func (g *GameState) IsQuestItemNPCHidden(mapUnit map_units.MapUnit) bool {
	return g.isQuestItemNPCHidden(mapUnit)
}

// GetSeatedBardPlayingPositions returns positions of bards actively playing music.
// See ref:TILE-0008.
func (g *GameState) GetSeatedBardPlayingPositions() []references.Position {
	if g.MapState.PlayerLocation.Location.GetMapType() != references.SmallMapType {
		return nil
	}

	npcs := g.CurrentNPCAIController.GetNpcs()
	if npcs == nil {
		return nil
	}

	currentMap := g.GetLayeredMapByCurrentLocation()
	playerFloor := g.MapState.PlayerLocation.Floor

	bards := npcs.GetFriendliesMatching(func(f *map_units.NPCFriendly) bool {
		if !f.IsVisible() || f.Floor() != playerFloor || !f.GetKeySpriteIndex().IsBardPlaying() {
			return false
		}
		pos := f.MapUnitDetails().Position
		tile := currentMap.GetTileTopMapOnlyTile(&pos)

		return tile != nil && tile.Index == indexes.ChairBackBack
	})

	positions := make([]references.Position, len(bards))
	for i, b := range bards {
		positions[i] = b.MapUnitDetails().Position
	}

	return positions
}
