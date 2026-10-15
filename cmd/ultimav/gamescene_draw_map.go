package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

// drawItemStacksOverlay renders all items on ground (equipment/provisions) as an overlay with proper outline support
func (g *GameScene) drawItemStacksOverlay(screen *ebiten.Image) {
	// Get current map info
	layer := g.gameState.GetLayeredMapByCurrentLocation()
	mapType := g.gameState.MapState.PlayerLocation.Location.GetMapType()

	// Animation timing
	animationTime := g.getElapsedMs()

	// Iterate through all visible tile positions
	margin := g.surfaceOverscanTiles()
	for tileY := -margin; tileY < references.Coordinate(yTilesVisibleOnGameScreen)+margin; tileY++ {
		for tileX := -margin; tileX < references.Coordinate(xTilesVisibleOnGameScreen)+margin; tileX++ {
			// Calculate world position for this screen tile
			worldPos := &references.Position{
				X: tileX + g.gameState.MapState.PlayerLocation.Position.X - xCenter,
				Y: tileY + g.gameState.MapState.PlayerLocation.Position.Y - yCenter,
			}

			// Handle coordinate wrapping for large maps
			if mapType == references.LargeMapType {
				worldPos = worldPos.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
			}

			// Skip if position not visible
			if !g.isPositionVisible(layer, worldPos) {
				continue
			}

			// Check for equipment/provisions at this position via ItemStacksMap
			if !g.gameState.ItemStacksMap.HasItemStackAtPosition(worldPos) {
				continue
			}

			// Get the top item from the stack
			item := g.gameState.ItemStacksMap.Peek(worldPos)
			if item == nil {
				continue
			}

			// Get the sprite index for this item
			itemRef := g.gameReferences.InventoryItemReferences.GetReferenceByItem(item.Item)
			itemSpriteIndex := itemRef.ItemSprite

			// Position-based hash for animation variety
			posHash := worldPos.AnimationHash()

			// Get animated sprite for the item
			itemSprite := g.spriteSheet.GetAnimatedSprite(itemSpriteIndex, posHash, animationTime)

			// Apply yellow outline to ItemStacks for visibility (except gems which have good contrast)
			// Skip outline for gems - they're already visually distinct and yellow clashes
			if provision, ok := item.Item.(references.Provision); !ok || provision != references.Gem {
				itemSprite = g.spriteSheet.GetTileEffectProcessor().ApplyDynamicSimple(itemSprite, sprites.YellowOutline)
			}

			// Calculate screen position
			screenX, screenY := sprites.ConvertPosition16to18(int(tileX), int(tileY))

			// Setup draw options with scaling
			itemDo := ebiten.DrawImageOptions{}
			if _, moonstone := item.Item.(references.Moonstone); moonstone && g.gameState.IsMoongateActiveAt(*worldPos) {
				// Items draw after gates. Keep the gate visible beneath its loose stone.
				itemDo.ColorScale.ScaleAlpha(0.6)
			}
			itemDo.GeoM.Translate(screenX, screenY)

			// Apply game screen scaling
			mapOp := g.worldDrawOptions()

			itemDo.GeoM.Concat(mapOp.GeoM)

			// Draw to screen
			screen.DrawImage(itemSprite, &itemDo)
		}
	}
}

// drawMapUnitsOverlay renders all NPCs, enemies, and vehicles as an overlay with proper outline support
func (g *GameScene) drawMapUnitsOverlay(screen *ebiten.Image) {
	// Skip if no NPC controller
	if g.gameState.CurrentNPCAIController == nil || g.gameState.CurrentNPCAIController.GetNpcs() == nil {
		return
	}

	// Get current map info
	layer := g.gameState.GetLayeredMapByCurrentLocation()
	mapType := g.gameState.MapState.PlayerLocation.Location.GetMapType()

	// Animation timing
	animationTime := g.getElapsedMs()

	// Iterate through all visible tile positions
	margin := g.surfaceOverscanTiles()
	for tileY := -margin; tileY < references.Coordinate(yTilesVisibleOnGameScreen)+margin; tileY++ {
		for tileX := -margin; tileX < references.Coordinate(xTilesVisibleOnGameScreen)+margin; tileX++ {
			g.drawMapUnitAtPosition(screen, layer, mapType, tileX, tileY, animationTime)
		}
	}
}

// drawMapUnitAtPosition renders a single map unit if one exists at the given tile position
func (g *GameScene) drawMapUnitAtPosition(
	screen *ebiten.Image, layer *map_state.LayeredMap, mapType references.GeneralMapType,
	tileX, tileY references.Coordinate, animationTime int64,
) {
	// Calculate world position for this screen tile
	worldPos := &references.Position{
		X: tileX + g.gameState.MapState.PlayerLocation.Position.X - xCenter,
		Y: tileY + g.gameState.MapState.PlayerLocation.Position.Y - yCenter,
	}

	// Handle coordinate wrapping for large maps
	if mapType == references.LargeMapType {
		worldPos = worldPos.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
	}

	mapUnit, presentation, ok := g.mapUnitPresentationAtPosition(layer, mapType, *worldPos)
	if !ok {
		return
	}

	// Render the map unit sprite
	g.renderMapUnitSprite(screen, mapUnit, worldPos, tileX, tileY, animationTime, presentation)
}

func (g *GameScene) mapUnitPresentationAtPosition(
	layer *map_state.LayeredMap,
	mapType references.GeneralMapType,
	position references.Position,
) (*map_units.MapUnit, game_state.SpritePresentation, bool) {
	if !g.shouldDrawMapUnitsOverlay() || !g.shouldRenderMapUnitAtPosition(layer, &position) ||
		g.gameState.CurrentNPCAIController == nil || g.gameState.CurrentNPCAIController.GetNpcs() == nil {
		return nil, game_state.SpritePresentation{}, false
	}
	mapUnit := mapUnitAtPositionForRender(
		g.gameState.CurrentNPCAIController.GetNpcs(),
		mapType,
		position,
		g.gameState.MapState.PlayerLocation.Floor,
	)
	if mapUnit == nil || !(*mapUnit).IsVisible() || g.isShadowlordSummonFizzTarget(*mapUnit) ||
		g.gameState.IsQuestItemNPCHidden(*mapUnit) ||
		(*mapUnit).Floor() != g.gameState.MapState.PlayerLocation.Floor {
		return nil, game_state.SpritePresentation{}, false
	}
	baseTile := layer.GetTileTopMapOnlyTile(&position)
	if baseTile == nil {
		return nil, game_state.SpritePresentation{}, false
	}
	presentation := g.gameState.GetNPCSpriteReplacement(baseTile.Index, (*mapUnit).GetKeySpriteIndex(), position)

	return mapUnit, presentation, true
}

func mapUnitAtPositionForRender(
	mapUnits *map_units.MapUnits,
	mapType references.GeneralMapType,
	position references.Position,
	floor references.FloorNumber,
) *map_units.MapUnit {
	if mapType == references.SmallMapType {
		return mapUnits.GetMapUnitAtPositionAndFloorOrNil(position, floor)
	}

	// Each large-map controller owns one flat world floor.
	return mapUnits.GetMapUnitAtPositionOrNil(position)
}

// shouldRenderMapUnitAtPosition checks if a map unit at the given position should be rendered
func (g *GameScene) shouldRenderMapUnitAtPosition(layer *map_state.LayeredMap, worldPos *references.Position) bool {
	// Skip if position not visible
	if !g.isPositionVisible(layer, worldPos) {
		return false
	}

	// Skip avatar position (handled separately)
	if g.gameState.IsAvatarAtPosition(worldPos) {
		return false
	}

	return true
}

// renderMapUnitSprite renders the sprite for a map unit at the given position
func (g *GameScene) renderMapUnitSprite(
	screen *ebiten.Image, mapUnit *map_units.MapUnit,
	worldPos *references.Position, tileX, tileY references.Coordinate, animationTime int64,
	presentation game_state.SpritePresentation,
) {
	// Position-based hash for animation variety
	posHash := worldPos.AnimationHash()

	// Only the actor body follows Time Stop. Tinting and glow keep ordinary time.
	bodyTime := g.surfaceActorAnimationTime(*mapUnit, animationTime)
	npcSpriteImage := g.spriteSheet.GetAnimatedSprite(presentation.Sprite, posHash, bodyTime)

	// Apply the presentation-owned NPC pipeline.
	// Skip for background-integrated sprites (sleeping in bed, sitting in chairs, climbing ladders)
	// These sprites include the background baked in, so applying effects looks wrong
	if !presentation.BackgroundIntegrated {
		npcSpriteImage = g.spriteSheet.GetTileEffectProcessor().ApplyDynamic(
			npcSpriteImage,
			sprites.StandardNPCPipeline,
		)
	}

	// Calculate screen position
	screenX, screenY := sprites.ConvertPosition16to18(int(tileX), int(tileY))

	// Setup draw options with scaling
	npcDo := ebiten.DrawImageOptions{}
	npcDo.GeoM.Translate(screenX, screenY)

	// Apply light source tinting to NPC (same visual treatment as tiles)
	g.applyLightTinting(*worldPos, &npcDo, animationTime)

	// Apply game screen scaling
	mapOp := g.worldDrawOptions()

	npcDo.GeoM.Concat(mapOp.GeoM)

	// Draw the base NPC sprite to screen
	screen.DrawImage(npcSpriteImage, &npcDo)

	// Apply pulsing glow overlay for light-emitting NPCs (wisps, seahorses, daemons, ghosts)
	// Uses additive blending to create a colored aura effect that pulses
	// Works both day and night - magical creatures always glow
	glowOverlay := sprites.GetNPCGlowOverlay(presentation.Sprite)
	if glowOverlay != nil {
		// Generate sprite mask for shaped glow (cached for performance)
		spriteMask := sprites.GetSpriteMask(npcSpriteImage)
		glowOverlay.ApplyOverlayAtPosition(
			screen,
			&npcDo,
			int(worldPos.X), int(worldPos.Y),
			1.0, // Full opacity
			spriteMask,
		)
	}

	if g.gameState.MapState.PlayerLocation.Location.GetMapType() == references.LargeMapType {
		if currentHP, maxHP, ok := largeMapShipHealth(*mapUnit); ok {
			drawCombatHealthBar(screen, &npcDo, currentHP, maxHP)
		}
	}
}

func largeMapShipHealth(mapUnit map_units.MapUnit) (int, int, bool) {
	switch unit := mapUnit.(type) {
	case *map_units.NPCEnemy:
		if references.IsPirateShipSprite(unit.GetKeySpriteIndex()) {
			return unit.CurrentHP(), references.PirateShipMaxHP, true
		}
	case *map_units.NPCFriendly:
		vehicle := unit.GetVehicleDetails()
		if vehicle.VehicleType == references.FrigateVehicle {
			return vehicle.GetHullHP(), map_units.MaxHullHP, true
		}
	}

	return 0, 0, false
}

// drawQuestItemsOverlay renders all quest items (shards + artifacts) as overlays with differentiated visual effects:
// - Lord British's artifacts (Crown, Sceptre, Amulet): black outline + pulsing golden-white glow
// - Shards: pulsing red-purple glow only (no outline)
func (g *GameScene) drawQuestItemsOverlay(screen *ebiten.Image) {
	layer := g.gameState.GetLayeredMapByCurrentLocation()
	mapType := g.gameState.MapState.PlayerLocation.Location.GetMapType()
	currentLocation := g.gameState.MapState.PlayerLocation.Location
	currentFloor := g.gameState.MapState.PlayerLocation.Floor

	// Iterate through all quest items (shards + artifacts)
	questItemPlacements := game_state.GetAllQuestItemPlacements()
	for _, itemData := range questItemPlacements {
		// Skip items not in current location/floor
		if currentLocation != itemData.Location || currentFloor != itemData.Floor {
			continue
		}

		worldPos := &itemData.Position

		// Handle coordinate wrapping based on map type
		if mapType == references.LargeMapType {
			worldPos = worldPos.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
		} else if worldPos.X < 0 || worldPos.Y < 0 {
			// Small maps don't wrap, but ensure coordinates are valid
			continue
		}

		// Calculate screen tile position relative to player
		relative := g.surfaceWorldRelativePosition(*worldPos)
		screenTileX := relative.X + xCenter
		screenTileY := relative.Y + yCenter

		// Check if quest item is within visible screen bounds
		margin := g.surfaceOverscanTiles()
		if screenTileX < -margin || screenTileX >= xTilesVisibleOnGameScreen+margin ||
			screenTileY < -margin || screenTileY >= yTilesVisibleOnGameScreen+margin {
			continue // Quest item is off-screen
		}

		// Skip if position not visible (lighting/line-of-sight check)
		if !g.isPositionVisible(layer, worldPos) {
			continue
		}

		// Check if this quest item should be visible (not in inventory)
		if !g.gameState.IsQuestItemVisible(*worldPos, itemData.QuestItemType) {
			continue
		}

		// Get the appropriate sprite for this quest item (from JSON data)
		questItemSprite := g.spriteSheet.GetSprite(g.gameState.GameReferences.InventoryItemReferences.GetQuestItemSpriteIndex(itemData.QuestItemType))

		// Determine which overlay to use based on item type
		var glowOverlay *sprites.OverlayPipeline
		isArtifact := game_state.IsLordBritishArtifact(itemData.QuestItemType)

		if isArtifact {
			// Lord British's artifacts get black outline + golden-white glow
			glowOverlay = sprites.GetArtifactGlowOverlay()
			// Apply black outline effect to sprite
			effectProcessor := g.spriteSheet.GetTileEffectProcessor()
			pipeline := sprites.Effect().
				AddOutline(sprites.BlackOutline).
				Build()
			questItemSprite = effectProcessor.ApplyDynamic(questItemSprite, pipeline)
		} else {
			// Shards get red-purple glow (no outline)
			glowOverlay = sprites.GetShardGlowOverlay()
		}

		// Calculate screen position
		screenX, screenY := sprites.ConvertPosition16to18(int(screenTileX), int(screenTileY))

		// Setup draw options with scaling
		questItemDo := ebiten.DrawImageOptions{}
		questItemDo.GeoM.Translate(screenX, screenY)

		// Apply game screen scaling
		mapOp := g.worldDrawOptions()

		questItemDo.GeoM.Concat(mapOp.GeoM)

		// Get elapsed time for animations
		elapsedMs := g.getElapsedMs()

		// Apply pulsing alpha to base sprite (modulates visibility over time)
		if glowOverlay != nil {
			pulsingAlpha := glowOverlay.GetPulsingAlpha(elapsedMs)
			questItemDo.ColorScale.ScaleAlpha(pulsingAlpha)
		}

		// Draw base sprite to screen (with pulsing alpha applied)
		screen.DrawImage(questItemSprite, &questItemDo)

		// Apply static glow overlay on top using sprite mask (glow follows sprite shape)
		if glowOverlay != nil {
			spriteMask := sprites.GetSpriteMask(questItemSprite)
			glowOverlay.ApplyOverlayWithMask(screen, &questItemDo, elapsedMs, 1.0, spriteMask)
		}
	}
}
