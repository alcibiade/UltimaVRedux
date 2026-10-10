package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// checkSpecialAreas handles all special regenerating and condition-based search areas
// This includes source-table skull keys, glass swords, castle keys, and ordinary records.
func (ssm *SearchSystemManager) checkSpecialAreas(searchPos *references.Position) *SearchResponse {
	onmap := ssm.gameState.MapState.PlayerLocation.Location
	level := ssm.gameState.MapState.PlayerLocation.Floor

	// Use the authentic search location manager to check for special items
	if authenticResult := ssm.authenticLocationManager.FindAtPosition(searchPos.X, searchPos.Y, level, onmap); authenticResult != nil {
		// Dispatch to specific handler based on the special item type
		switch authenticResult.Index {
		case SpecialItemSkullKeys:
			return ssm.checkSkullKeysTree(searchPos, authenticResult)
		case SpecialItemCastleKeys:
			return ssm.checkCastleKeysSpawn(searchPos, authenticResult)
		case SpecialItemGlassSword:
			return ssm.checkGlassSwordSpawn(searchPos, authenticResult)
		default:
			// Handle other normal search locations found by authentic manager
			return ssm.handleNormalSearchLocation(searchPos, authenticResult)
		}
	}

	return nil
}

// checkSkullKeysTree handles skull key daily regeneration at Minoc tree location
func (ssm *SearchSystemManager) checkSkullKeysTree(searchPos *references.Position, result *AuthenticSearchResult) *SearchResponse {
	// Create items to place on map
	items := []references.ItemAndQuantity{
		{
			Item:     result.ItemType,
			Quantity: result.Quantity,
		},
	}

	// Add items to ItemStacksMap for retrieval via Get command
	err := ssm.addFoundItemsToMap(searchPos, items)
	if err != nil {
		return &SearchResponse{
			Result:    SearchResultNotFound,
			Message:   MessageCantPlaceItems + err.Error(),
			TimeSpent: 1,
		}
	}
	ssm.authenticLocationManager.CommitFound(result)

	// Generate success message
	message := ssm.buildDiscoveryMessage("", result.DiscoveryText)

	// Trigger screen update to show new items visually
	ssm.gameState.SystemCallbacks.Screen.UpdateDisplay()

	return &SearchResponse{
		Result:    SearchResultFoundItem,
		Items:     items,
		Message:   message,
		TimeSpent: 1,
	}
}

// checkCastleKeysSpawn handles keys in Blackthorn's Castle (requires no keys in inventory)
func (ssm *SearchSystemManager) checkCastleKeysSpawn(searchPos *references.Position, result *AuthenticSearchResult) *SearchResponse {
	// This logic is handled by the authentic search location manager's requirements checking
	// If we get here, the requirements are already met
	return ssm.handleNormalSearchLocation(searchPos, result)
}

// checkGlassSwordSpawn handles glass sword discovery (requires no glass swords in inventory)
func (ssm *SearchSystemManager) checkGlassSwordSpawn(searchPos *references.Position, result *AuthenticSearchResult) *SearchResponse {
	// This logic is handled by the authentic search location manager's requirements checking
	// If we get here, the requirements are already met
	return ssm.handleNormalSearchLocation(searchPos, result)
}

// checkReagentCaches handles special reagent cache locations from original Search/Get/Open/Jimmy rules reagent-cache search
// These include mandrake root and nightshade at specific overworld coordinates (midnight only, daily regeneration)
// See ref:INV-0008.
func (ssm *SearchSystemManager) checkReagentCaches(searchPos *references.Position) *SearchResponse {
	// Must be in the overworld (BritanniaUnderworld)
	onmap := ssm.gameState.MapState.PlayerLocation.Location
	if onmap != references.BritanniaUnderworld {
		return nil
	}

	// Must be at surface level (level 0)
	level := ssm.gameState.MapState.PlayerLocation.Floor
	if level != 0 {
		return nil
	}

	// Must be midnight (hour 0)
	if ssm.gameState.DateTime.Hour != 0 {
		return nil
	}

	// Check each reagent cache location
	for i, cacheLocation := range reagentCacheLocations {
		// Check if this is the right coordinates
		if searchPos.X != cacheLocation.X || searchPos.Y != cacheLocation.Y {
			continue
		}

		// Check if this reagent cache was already found today (daily regeneration)
		currentDay := ssm.gameState.DateTime.Day
		if ssm.reagentCachesFoundDay[i] == currentDay {
			return nil
		}

		// Mark this reagent cache as found today
		ssm.reagentCachesFoundDay[i] = currentDay

		// Generate random quantity within the cache's range
		quantity := ssm.gameState.RandomIntInRange(cacheLocation.MinQuantity, cacheLocation.MaxQuantity)

		// Add reagents directly to inventory (authentic Ultima V behavior)
		// IncrementBy automatically clamps to 99 for reagents (itemQuantitySmallMax)
		ssm.gameState.PartyState.Inventory.Reagent.IncrementBy(cacheLocation.ReagentType, uint16(quantity))

		// Create items list for message formatting (but don't place on map)
		items := []references.ItemAndQuantity{
			{
				Item:     cacheLocation.ReagentType,
				Quantity: uint16(quantity),
			},
		}

		// Generate success message using standard formatting
		message := ssm.buildFoundMessage(ssm.getTileLocationName(topTileAt(ssm.gameState, searchPos).Index), items)

		// Trigger screen update to show new items visually
		ssm.gameState.SystemCallbacks.Screen.UpdateDisplay()

		return &SearchResponse{
			Result:    SearchResultFoundItem,
			Items:     items,
			Message:   message,
			TimeSpent: 1,
		}
	}

	// No reagent cache found at this location
	return nil
}

// handleNormalSearchLocation handles normal search locations found by the authentic manager
func (ssm *SearchSystemManager) handleNormalSearchLocation(searchPos *references.Position, result *AuthenticSearchResult) *SearchResponse {
	if result.ItemType == nil {
		if !ssm.gameState.addSurfaceCorpse(result) {
			return &SearchResponse{
				Result:    SearchResultNotFound,
				Message:   MessageCantPlaceItems,
				TimeSpent: 1,
			}
		}
		ssm.authenticLocationManager.CommitFound(result)
		layeredMap := ssm.gameState.GetLayeredMapByCurrentLocation()
		topTile := layeredMap.GetTopTile(searchPos)
		message := ssm.buildDiscoveryMessage(ssm.getTileLocationName(topTile.Index), result.DiscoveryText)
		ssm.gameState.SystemCallbacks.Screen.UpdateDisplay()

		return &SearchResponse{
			Result:    SearchResultFoundItem,
			Message:   message,
			TimeSpent: 1,
		}
	}

	// Create items to place on map
	items := []references.ItemAndQuantity{
		{
			Item:     result.ItemType,
			Quantity: result.Quantity,
		},
	}

	// Add items to ItemStacksMap for retrieval via Get command
	err := ssm.addFoundItemsToMap(searchPos, items)
	if err != nil {
		return &SearchResponse{
			Result:    SearchResultNotFound,
			Message:   MessageCantPlaceItems + err.Error(),
			TimeSpent: 1,
		}
	}
	ssm.authenticLocationManager.CommitFound(result)

	// Generate success message based on the tile at this location
	layeredMap := ssm.gameState.GetLayeredMapByCurrentLocation()
	topTile := layeredMap.GetTopTile(searchPos)
	location := ssm.getTileLocationName(topTile.Index)
	message := ssm.buildDiscoveryMessage(location, result.DiscoveryText)

	// Trigger screen update to show new items visually
	ssm.gameState.SystemCallbacks.Screen.UpdateDisplay()

	return &SearchResponse{
		Result:    SearchResultFoundItem,
		Items:     items,
		Message:   message,
		TimeSpent: 1,
	}
}
