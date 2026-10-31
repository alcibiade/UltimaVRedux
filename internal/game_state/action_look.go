package game_state

import (
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const (
	wantedPosterX             = 0x11
	wantedPosterY             = 0x15
	wantedPosterInteriorWidth = 13
	wantedPosterNameRows      = 3
)

// ActionLookSmallMap looks one tile in a direction on a town or keep map and reports whether it saw something.
func (g *GameState) ActionLookSmallMap(direction references.Direction) bool {
	newPosition := direction.GetNewPositionInDirection(&g.MapState.PlayerLocation.Position)
	if newPosition == nil {
		g.SystemCallbacks.Message.AddRowStr("Look-Thou dost see nothing")
		g.AdvanceOrdinarySurfaceCommandMinutes(1)

		return true
	}

	// Check for moongate first (it overlays the tile)
	if g.LookAtMoongate(*newPosition) {
		g.AdvanceOrdinarySurfaceCommandMinutes(1)

		return true
	}

	// Check for wishing well (modern UI interaction)
	if g.DetectWishingWellAtPosition(newPosition) {
		// The workflow owns interaction; the surface dispatcher completes the one Look cycle.
		return g.TriggerWishingWellModern(newPosition)
	}

	// Check for fountain (cosmetic drinking interaction)
	if g.DetectFountainAtPosition(newPosition) {
		// TriggerFountainInteraction handles dialog sequence and time advancement internally
		return g.TriggerFountainInteraction()
	}

	currentMap := g.GetLayeredMapByCurrentLocation()
	topTile := currentMap.GetTopTile(newPosition)
	if topTile == nil {
		g.SystemCallbacks.Message.AddRowStr("Look-Thou dost see nothing")
		g.AdvanceOrdinarySurfaceCommandMinutes(1)

		return true
	}
	lookTileIndex := topTile.Index
	redirectedSignPiece := false
	hasMapUnit := currentMap.GetTileByLayer(map_state.MapUnitLayer, newPosition).Index > 0
	if !hasMapUnit {
		lookTileIndex = currentMap.GetSmallMapTerrainTile(newPosition, g.DateTime.Hour).Index
		lookTileIndex, redirectedSignPiece = g.resolveLookSignPiece(newPosition, lookTileIndex)
	}

	// Check for sign tile - show sign dialog with runic text
	logging.Debugf(logging.CategoryUI, "Look: topTile.Index=%d, IsSign=%v", lookTileIndex, lookTileIndex.IsSign())
	if !hasMapUnit && !redirectedSignPiece && lookTileIndex.IsSign() {
		logging.Debugf(logging.CategoryUI, "Look: Detected sign, calling handleLookAtSign")

		return g.handleLookAtSign(newPosition, topTile)
	}

	lookDescription := g.GameReferences.LookReferences.GetTileLookDescription(lookTileIndex)
	if !hasMapUnit {
		// REDUX DEVIATION: CRAFT-0005: deed-map resource nodes reuse original tiles whose Look
		// text would mislead ("bountiful crops", "mountains"); Look names the node instead.
		if nodeDescription, isNode := g.deedNodeLookDescription(*newPosition); isNode {
			lookDescription = nodeDescription
		}
	}
	fullMessage := "Thou dost see " + lookDescription
	flameKnowledgeID := ""

	// Special cases for interactive tiles - append time to message or override description
	//nolint:exhaustive // Only tiles with contextual Look behavior need cases.
	switch lookTileIndex {
	case indexes.Clock1, indexes.Clock2:
		fullMessage += g.DateTime.GetTimeAsString()
	case indexes.BlueFlame:
		// Blue Flames at Keeps show the Flame of Principle name
		flame := references.GetFlameAtLocation(g.MapState.PlayerLocation.Location)
		if flame >= 0 {
			fullMessage = "Thou dost see the " + flame.GetPrinciple()
			flameKnowledgeID = "flame." + strings.ToLower(flame.String())
		}
	case indexes.Telescope:
		// Telescope triggers astronomy viewing - day/night branching handled by workflow
		g.SystemCallbacks.Flow.Observation.ShowTelescopeInteraction()

		return true // Workflow handles time advancement
	case indexes.CrystalBall:
		// Crystal ball triggers overhead gem view with intelligence check
		g.SystemCallbacks.Flow.Observation.ShowCrystalBallInteraction()

		return true // Workflow handles time advancement
	}

	g.SystemCallbacks.UI.ShowActionFeedback(fullMessage, false)
	g.SystemCallbacks.Message.AddRowStr("Look-" + fullMessage)
	if flameKnowledgeID != "" {
		g.DiscoverJournalKnowledge(flameKnowledgeID, ObservationKnowledgeSource("Look ("+questJournalKeepName(g.MapState.PlayerLocation.Location)+")"))
	}
	g.AdvanceOrdinarySurfaceCommandMinutes(1)

	return true
}

// resolveLookSignPiece reproduces terrain description for the directional
// E0-E2 sign structure. The terminal tile is described normally and is not
// redispatched as an inscription.
// See ref:CMD-0004.
func (g *GameState) resolveLookSignPiece(position *references.Position, tileIndex indexes.SpriteIndex) (indexes.SpriteIndex, bool) {
	if tileIndex < indexes.LargeSignBottom || tileIndex > indexes.LargeSignRight {
		return tileIndex, false
	}

	currentPosition := *position
	visited := make(map[references.Position]struct{})
	currentMap := g.GetLayeredMapByCurrentLocation()
	for tileIndex >= indexes.LargeSignBottom && tileIndex <= indexes.LargeSignRight {
		if _, exists := visited[currentPosition]; exists {
			logging.Warnf(logging.CategoryGameState, "Look: directional sign cycle at (%d,%d)", currentPosition.X, currentPosition.Y)

			return tileIndex, true
		}
		visited[currentPosition] = struct{}{}

		//nolint:exhaustive // The loop admits only the three directional sign pieces.
		switch tileIndex {
		case indexes.LargeSignBottom:
			currentPosition.Y--
		case indexes.LargeSignLeft:
			currentPosition.X++
		case indexes.LargeSignRight:
			currentPosition.X--
		}
		if g.MapState.PlayerLocation.Location.GetMapType() == references.LargeMapType {
			currentPosition.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
		}

		tile := currentMap.GetTileTopMapOnlyTile(&currentPosition)
		if tile == nil || tile.Index <= 0 {
			return tileIndex, true
		}
		tileIndex = tile.Index
	}

	return tileIndex, true
}

// handleLookAtSign handles looking at a sign tile - shows sign dialog with runic text.
func (g *GameState) handleLookAtSign(position *references.Position, tile *references.Tile) bool {
	logging.Debugf(logging.CategoryUI, "handleLookAtSign: position=(%d,%d), tile.Index=%d", position.X, position.Y, tile.Index)

	// The Yew wanted poster is generated from the active party before SIGNS.DAT lookup.
	sign := g.wantedPosterSign(position)
	switch {
	case sign != nil:
		logging.Debugf(logging.CategoryUI, "handleLookAtSign: Generated Yew wanted poster")
	case g.GameReferences.SignReferences != nil:
		sign = g.GameReferences.SignReferences.GetSignAt(
			g.MapState.PlayerLocation.Location,
			int(g.MapState.PlayerLocation.Floor),
			int(position.X),
			int(position.Y),
		)
		if sign != nil {
			logging.Debugf(logging.CategoryUI, "handleLookAtSign: Found sign record with %d bytes of text", len(sign.RawText))
		} else {
			logging.Debugf(logging.CategoryUI, "handleLookAtSign: No sign record found, will use default text")
		}
	default:
		logging.Debugf(logging.CategoryUI, "handleLookAtSign: SignReferences is nil")
	}

	// Show sign dialog (nil sign will display default message)
	logging.Debugf(logging.CategoryUI, "handleLookAtSign: Calling ShowSignDialog callback")
	g.SystemCallbacks.UI.ShowSignDialog(sign)

	// Also output normal look description
	lookDescription := g.GameReferences.LookReferences.GetTileLookDescription(tile.Index)
	g.SystemCallbacks.Message.AddRowStr("Look-Thou dost see " + lookDescription)
	g.AdvanceOrdinarySurfaceCommandMinutes(1)

	return true
}

func (g *GameState) wantedPosterSign(position *references.Position) *references.SignRecord {
	if g.MapState.PlayerLocation.Location != references.Yew ||
		g.MapState.PlayerLocation.Floor != references.MainFloor ||
		int(position.X) != wantedPosterX || int(position.Y) != wantedPosterY {
		return nil
	}

	names := make([]string, 0, wantedPosterNameRows)
	for index := range min(g.PartyState.GetActivePartyCount(), wantedPosterNameRows) {
		names = append(names, g.PartyState.GetCharacter(index).GetNameAsString())
	}

	return &references.SignRecord{
		Location: references.Yew,
		Floor:    int(references.MainFloor),
		X:        wantedPosterX,
		Y:        wantedPosterY,
		RawText:  buildWantedPosterText(names),
	}
}

func buildWantedPosterText(names []string) []byte {
	rawText := make([]byte, 0, 9*16)
	rawText = append(rawText, "abbbbbbbbbbbbbc\n"...)
	rawText = appendWantedPosterLine(rawText, "Wanted:")
	rawText = appendWantedPosterLine(rawText, "")
	for row := range wantedPosterNameRows {
		name := ""
		if row < len(names) {
			name = names[row]
		}
		rawText = appendWantedPosterLine(rawText, name)
	}
	rawText = appendWantedPosterLine(rawText, "")
	rawText = appendWantedPosterLine(rawText, "Dead or Alive")
	rawText = append(rawText, "deeeeeeeeeeeeef"...)

	return rawText
}

func appendWantedPosterLine(rawText []byte, content string) []byte {
	if len(content) > wantedPosterInteriorWidth {
		panic("wanted poster line exceeds interior width")
	}
	line := make([]byte, wantedPosterInteriorWidth)
	for index := range line {
		line[index] = ' ' | references.SignHighBitMask
	}
	start := (wantedPosterInteriorWidth - len(content)) / 2
	for index := range len(content) {
		line[start+index] = content[index] | references.SignHighBitMask
	}

	rawText = append(rawText, 'g')
	rawText = append(rawText, line...)
	rawText = append(rawText, 'g', '\n')

	return rawText
}

// ActionLookLargeMap looks one tile in a direction on the overworld or underworld and reports whether it saw something.
func (g *GameState) ActionLookLargeMap(direction references.Direction) bool {
	newPosition := direction.GetNewPositionInDirection(&g.MapState.PlayerLocation.Position)
	if newPosition == nil {
		g.SystemCallbacks.Message.AddRowStr("Look-Thou dost see nothing")
		g.AdvanceOrdinarySurfaceCommandMinutes(1)

		return true
	}
	terrainPosition := *newPosition
	terrainPosition.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
	wrappedAcrossSeam := terrainPosition != *newPosition

	// Dynamic overlays retain their raw lookup coordinate; only terrain wraps.
	if g.LookAtMoongate(*newPosition) {
		g.AdvanceOrdinarySurfaceCommandMinutes(1)

		return true
	}

	// Check for quest items (shards + artifacts) - they are now rendered as overlays, not tiles
	hasQuestItemHere, questItem := g.IsQuestItemAtPosition(*newPosition)
	if hasQuestItemHere && g.IsQuestItemVisible(*newPosition, questItem) {
		// Get the quest item sprite index for description lookup
		questItemSpriteIndex := g.GameReferences.InventoryItemReferences.GetQuestItemSpriteIndex(questItem)
		questItemDescription := g.GameReferences.LookReferences.GetTileLookDescription(questItemSpriteIndex)
		fullMessage := "Thou dost see " + questItemDescription
		g.SystemCallbacks.UI.ShowActionFeedback(fullMessage, false)
		g.SystemCallbacks.Message.AddRowStr("Look-" + fullMessage)
		g.AdvanceOrdinarySurfaceCommandMinutes(1)

		return true
	}

	currentMap := g.GetLayeredMapByCurrentLocation()
	topTile := currentMap.GetTileTopMapOnlyTile(&terrainPosition)
	hasMapUnit := false
	if !wrappedAcrossSeam {
		topTile = currentMap.GetTopTile(newPosition)
		hasMapUnit = currentMap.GetTileByLayer(map_state.MapUnitLayer, newPosition).Index > 0
	}
	if topTile == nil {
		g.SystemCallbacks.Message.AddRowStr("Look-Thou dost see nothing")
		g.AdvanceOrdinarySurfaceCommandMinutes(1)

		return true
	}
	lookTileIndex := topTile.Index
	redirectedSignPiece := false
	if !hasMapUnit {
		if wrappedAcrossSeam {
			lookTileIndex = g.getEffectiveMapTerrainTile(currentMap, &terrainPosition)
		} else {
			lookTileIndex = g.GetEffectiveTile(newPosition)
		}
		lookTileIndex, redirectedSignPiece = g.resolveLookSignPiece(&terrainPosition, lookTileIndex)
	}

	// Check for sign tile - show sign dialog with runic text
	logging.Debugf(logging.CategoryUI, "LargeMap Look: topTile.Index=%d, IsSign=%v", lookTileIndex, lookTileIndex.IsSign())
	if !hasMapUnit && !redirectedSignPiece && lookTileIndex.IsSign() {
		return g.handleLookAtSign(&terrainPosition, topTile)
	}

	lookDescription := g.GameReferences.LookReferences.GetTileLookDescription(lookTileIndex)
	if !hasMapUnit && !redirectedSignPiece {
		if dungeon, found := g.dungeonEntranceAt(terrainPosition, lookTileIndex); found {
			if lookTileIndex == indexes.BlockEntrance {
				lookDescription += dungeon.String()
			} else {
				// REDUX DEVIATION: terrain description names only sealed dungeon
				// entrances. Redux names an observed open portal as well so both
				// exterior sides identify the dungeon without requiring entry.
				// See ref:CMD-0004.
				lookDescription = "the entrance to the dungeon " + dungeon.String()
			}
			g.observeDungeonEntrance(dungeon, terrainPosition, g.sampleDungeonObservation(), false)
		}
	}
	fullMessage := "Thou dost see " + lookDescription
	logging.Debugf(logging.CategoryUI, "LargeMap Look: NOT a sign, showing feedback: %s", fullMessage)

	// Special cases for interactive tiles - append time to message
	//nolint:exhaustive // Only clock tiles add contextual text.
	switch lookTileIndex {
	case indexes.Clock1, indexes.Clock2:
		fullMessage += g.DateTime.GetTimeAsString()
	}

	g.SystemCallbacks.UI.ShowActionFeedback(fullMessage, false)
	g.SystemCallbacks.Message.AddRowStr("Look-" + fullMessage)
	if !hasMapUnit && !redirectedSignPiece {
		g.observeAtlasLook(terrainPosition, lookTileIndex)
	}
	g.AdvanceOrdinarySurfaceCommandMinutes(1)

	return true
}

func (g *GameState) dungeonEntranceAt(position references.Position, tile indexes.SpriteIndex) (references.Location, bool) {
	position.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
	entrances := g.GetDungeonEntranceCoordinates()
	for _, candidate := range canonicalDungeonWords {
		if !DungeonEntranceExistsOnFloor(candidate.dungeon, g.MapState.PlayerLocation.Floor) ||
			position != entrances[candidate.dungeon] ||
			(tile != candidate.open && tile != indexes.BlockEntrance) {
			continue
		}

		// REDUX DEVIATION: original Look rules selected a sealed entrance name by target
		// X alone. Redux verifies the full coordinate, floor and visible tile.
		// See ref:CMD-0004.
		return candidate.dungeon, true
	}

	return 0, false
}

// ActionLookDungeonMap describes one relative dungeon cell for the selected character.
func (g *GameState) ActionLookDungeonMap(
	direction references.Direction,
	characterIndex int,
) (DungeonLookResult, bool) {
	return g.actionLookDungeonMap(direction, characterIndex)
}
