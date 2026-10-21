package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	searchLocationCount       = 113
	surfaceCorpseOrderBitBase = searchLocationCount
	surfaceCorpseOrderBits    = 4
	searchBitsPerByte         = 8
	searchBitMask             = searchBitsPerByte - 1
	searchFoundBit            = 1
)

func surfaceCorpseOrdinal(searchIndex int) (int, bool) {
	ordinal := 0
	for _, location := range references.GetAllSearchLocations() {
		if location.ObjectClass != references.SearchObjectDeadPC && location.ObjectClass != references.SearchObjectDeadMonster {
			continue
		}
		if location.Index == searchIndex {
			return ordinal, true
		}
		ordinal++
	}

	return 0, false
}

// AuthenticSearchLocationManager implements the original Search/Get/Open/Jimmy rules search algorithm
// See ref:INV-0008.
type AuthenticSearchLocationManager struct {
	gameState *GameState
	// Found status tracking using bitmask array (like original Search/Get/Open/Jimmy rules discovery bits)
	// See ref:INV-0008.
	objFound [24]byte // 24 bytes = 192 bits for 113+ search locations
	// Day tracking for skull keys (original Search/Get/Open/Jimmy rules last reveal day variable)
	// See ref:INV-0008.
	skullKeysFoundDay byte // Day when skull keys were last found (1-28)
}

// NewAuthenticSearchLocationManager creates the authentic original Search/Get/Open/Jimmy rules-based search manager
// See ref:INV-0008.
func NewAuthenticSearchLocationManager(gameState *GameState) *AuthenticSearchLocationManager {
	aslm := &AuthenticSearchLocationManager{
		gameState: gameState,
		// objFound array starts as all zeros (no locations found initially)
	}

	// TODO: Load found status from save file when save/load system is implemented
	// For now, all locations start as not found

	return aslm
}

func (aslm *AuthenticSearchLocationManager) snapshotState() ([24]byte, byte) {
	return aslm.objFound, aslm.skullKeysFoundDay
}

func (aslm *AuthenticSearchLocationManager) restoreState(found [24]byte, skullKeysFoundDay byte) {
	aslm.objFound = found
	aslm.skullKeysFoundDay = skullKeysFoundDay
}

func (aslm *AuthenticSearchLocationManager) surfaceCorpseOrder() []int {
	locations := references.GetAllSearchLocations()
	result := make([]int, 0)
	capacity := (len(aslm.objFound)*searchBitsPerByte - surfaceCorpseOrderBitBase) / surfaceCorpseOrderBits
	for slot := range capacity {
		value := aslm.readSearchStateBits(surfaceCorpseOrderBitBase+slot*surfaceCorpseOrderBits, surfaceCorpseOrderBits)
		if value == 0 {
			return result
		}
		ordinal := value - 1
		for _, location := range locations {
			if location.ObjectClass != references.SearchObjectDeadPC && location.ObjectClass != references.SearchObjectDeadMonster {
				continue
			}
			if ordinal == 0 {
				result = append(result, location.Index)

				break
			}
			ordinal--
		}
	}

	return result
}

func (aslm *AuthenticSearchLocationManager) setSurfaceCorpseActive(searchIndex int, active bool) {
	ordinal, ok := surfaceCorpseOrdinal(searchIndex)
	if !ok || ordinal >= (1<<surfaceCorpseOrderBits)-1 {
		panic("cannot encode surface corpse search location")
	}
	order := aslm.surfaceCorpseOrder()
	for index, existing := range order {
		if existing != searchIndex {
			continue
		}
		if active {
			return
		}
		order = append(order[:index], order[index+1:]...)
		aslm.writeSurfaceCorpseOrder(order)

		return
	}
	if active {
		order = append(order, searchIndex)
		aslm.writeSurfaceCorpseOrder(order)
	}
}

func (aslm *AuthenticSearchLocationManager) writeSurfaceCorpseOrder(order []int) {
	capacity := (len(aslm.objFound)*searchBitsPerByte - surfaceCorpseOrderBitBase) / surfaceCorpseOrderBits
	if len(order) > capacity {
		panic("surface corpse order exceeds search-state capacity")
	}
	for slot := range capacity {
		value := 0
		if slot < len(order) {
			ordinal, ok := surfaceCorpseOrdinal(order[slot])
			if !ok {
				panic("cannot encode non-corpse search location")
			}
			value = ordinal + 1
		}
		aslm.writeSearchStateBits(surfaceCorpseOrderBitBase+slot*surfaceCorpseOrderBits, surfaceCorpseOrderBits, value)
	}
}

func (aslm *AuthenticSearchLocationManager) readSearchStateBits(start, count int) int {
	value := 0
	for offset := range count {
		bit := start + offset
		if aslm.objFound[bit/searchBitsPerByte]&(1<<(bit&searchBitMask)) != 0 {
			value |= 1 << offset
		}
	}

	return value
}

func (aslm *AuthenticSearchLocationManager) writeSearchStateBits(start, count, value int) {
	for offset := range count {
		bit := start + offset
		mask := byte(1 << (bit & searchBitMask))
		if value&(1<<offset) != 0 {
			aslm.objFound[bit/searchBitsPerByte] |= mask
		} else {
			aslm.objFound[bit/searchBitsPerByte] &^= mask
		}
	}
}

// ResetAllFoundStatus clears all found flags (useful for new games)
func (aslm *AuthenticSearchLocationManager) ResetAllFoundStatus() {
	aslm.objFound = [24]byte{}
}

// GetTotalSearchLocations returns the total number of search locations in the system
func (aslm *AuthenticSearchLocationManager) GetTotalSearchLocations() int {
	return len(references.GetAllSearchLocations())
}

// GetFoundCount returns the number of locations currently marked as found
func (aslm *AuthenticSearchLocationManager) GetFoundCount() int {
	count := 0
	for i := range searchLocationCount {
		if aslm.isAlreadyFound(i) {
			count++
		}
	}

	return count
}

func isOneTimeSearchLocation(location references.SearchLocationReference) bool {
	return !location.Requirements.IsSpecialItem && !location.Requirements.IsRepeatable
}

func (aslm *AuthenticSearchLocationManager) getOneTimeFoundCount() int {
	count := 0
	for _, location := range references.GetAllSearchLocations() {
		if isOneTimeSearchLocation(location) && aslm.isAlreadyFound(location.Index) {
			count++
		}
	}

	return count
}

func (aslm *AuthenticSearchLocationManager) resetOneTimeFoundStatus() (int, int) {
	before := aslm.getOneTimeFoundCount()
	for _, location := range references.GetAllSearchLocations() {
		if !isOneTimeSearchLocation(location) || location.Index < 0 || location.Index >= searchLocationCount {
			continue
		}
		byteIndex := location.Index / searchBitsPerByte
		bitIndex := location.Index & searchBitMask
		aslm.objFound[byteIndex] &^= byte(searchFoundBit << bitIndex)
	}

	return before, aslm.getOneTimeFoundCount()
}

// GetFoundLocationsByMap returns a map of found locations grouped by map location
func (aslm *AuthenticSearchLocationManager) GetFoundLocationsByMap() map[references.Location][]int {
	result := make(map[references.Location][]int)
	allLocations := references.GetAllSearchLocations()

	for _, loc := range allLocations {
		if aslm.isAlreadyFound(loc.Index) {
			mapLoc := references.Location(loc.MapNumber)
			result[mapLoc] = append(result[mapLoc], loc.Index)
		}
	}

	return result
}

// AuthenticSearchLocationManager methods implementing original Search/Get/Open/Jimmy rules algorithm
// See ref:INV-0008.

// FindAtPosition returns the first eligible source-order record without mutating
// found state. The caller commits only after creating the world object.
func (aslm *AuthenticSearchLocationManager) FindAtPosition(x, y references.Coordinate, level references.FloorNumber, onmap references.Location) *AuthenticSearchResult {
	// Get all search locations from references
	allLocations := references.GetAllSearchLocations()

	// Scan through all search locations (equivalent to original Search/Get/Open/Jimmy rules for loop)
	// See ref:INV-0008.
	for _, location := range allLocations {
		// Require the search record’s map, floor, X and Y to match the target. ref:INV-0008
		if references.Location(location.MapNumber) == onmap && references.FloorNumber(location.Level) == level && references.Coordinate(location.X) == x && references.Coordinate(location.Y) == y {
			// Handle special items (indices 13-15 in original Search/Get/Open/Jimmy rules)
			// See ref:INV-0008.
			if location.Index == 13 { // Keys in Blackthorn's Castle
				if !aslm.meetsKeysRequirement(location) || aslm.alreadyFoundAt(int(x), int(y)) {
					continue
				}

				return aslm.createFoundResult(location)
			}

			if location.Index == 14 { // Skull keys
				if !aslm.meetsSkullKeysRequirement(location) {
					continue
				}

				return aslm.createFoundResult(location)
			}

			if location.Index == 15 { // Glass swords
				if !aslm.meetsGlassSwordsRequirement(location) {
					continue
				}
				// Check if items already exist at this location
				if aslm.alreadyFoundAt(int(x), int(y)) {
					// Items exist at location - this should be handled by higher level search logic
					continue
				}

				return aslm.createFoundResult(location)
			}

			// Regular items (indices < 13 or > 15)
			if (location.Index < 13 || location.Index > 15) && !aslm.isAlreadyFound(location.Index) {
				return aslm.createFoundResult(location)
			}
		}
	}

	// Nothing found
	return nil
}

// SearchAtPosition preserves the complete lookup-and-commit operation used by
// direct callers. Production world placement uses FindAtPosition and CommitFound.
func (aslm *AuthenticSearchLocationManager) SearchAtPosition(x, y references.Coordinate, level references.FloorNumber, onmap references.Location) *AuthenticSearchResult {
	result := aslm.FindAtPosition(x, y, level, onmap)
	if result != nil {
		aslm.CommitFound(result)
	}

	return result
}

type AuthenticSearchResult struct {
	ItemType      references.Item
	Quantity      uint16
	Index         int
	Position      references.Position
	ObjectClass   references.SearchObjectClass
	DiscoveryText string
}

// CommitFound records a discovery after its world representation is created.
func (aslm *AuthenticSearchLocationManager) CommitFound(result *AuthenticSearchResult) {
	if result == nil || result.Index < 0 || result.Index >= searchLocationCount {
		panic("cannot commit invalid search result")
	}
	if result.Index == SpecialItemSkullKeys {
		aslm.setSkullKeysDay()

		return
	}
	if result.Index < SpecialItemCastleKeys || result.Index > SpecialItemGlassSword {
		aslm.markAsFound(result.Index)
	}
}

// isAlreadyFound checks if location has been found using bitmask array
func (aslm *AuthenticSearchLocationManager) isAlreadyFound(index int) bool {
	if index < 0 || index >= searchLocationCount {
		return true // Invalid index
	}

	// Equivalent to: (obj_found[i>>3] & mask) == 0
	byteIndex := index / searchBitsPerByte
	bitIndex := index & searchBitMask
	mask := byte(searchFoundBit << bitIndex)

	return (aslm.objFound[byteIndex] & mask) != 0
}

// IsLocationFound is the public method to check if a search location has been found
// Used by the overlay system to determine which locations are still searchable
func (aslm *AuthenticSearchLocationManager) IsLocationFound(index int) bool {
	return aslm.isAlreadyFound(index)
}

// markAsFound sets the found bit for this location
func (aslm *AuthenticSearchLocationManager) markAsFound(index int) {
	if index < 0 || index >= searchLocationCount {
		return // Invalid index
	}

	// Equivalent to: obj_found[i>>3] |= mask
	byteIndex := index / searchBitsPerByte
	bitIndex := index & searchBitMask
	mask := byte(searchFoundBit << bitIndex)

	aslm.objFound[byteIndex] |= mask
}

// Special requirement checks

func (aslm *AuthenticSearchLocationManager) meetsKeysRequirement(location references.SearchLocationReference) bool {
	if !location.Requirements.RequiresNoKeys {
		return true
	}

	// Player must have NO keys in inventory
	keys := aslm.gameState.PartyState.Inventory.Provisions.Keys.Get()

	return keys == 0
}

func (aslm *AuthenticSearchLocationManager) meetsSkullKeysRequirement(location references.SearchLocationReference) bool {
	if !location.Requirements.IsSkullKeys {
		return true
	}

	// Skull Key search is eligible only when today differs from the saved
	// last-reveal day; pickup is a separate operation. ref:PROV-0006
	currentDay := aslm.gameState.DateTime.Day

	// If skull keys have never been found (skullKeysFoundDay == 0), they're available
	// If current day != day they were found, they're available again
	return aslm.skullKeysFoundDay == 0 || currentDay != aslm.skullKeysFoundDay
}

func (aslm *AuthenticSearchLocationManager) meetsGlassSwordsRequirement(location references.SearchLocationReference) bool {
	if !location.Requirements.RequiresNoGlassSwords {
		return true
	}

	// Glass Sword discovery requires no carried Glass Sword. ref:QUEST-0002
	glassSwordCount := aslm.gameState.PartyState.Inventory.Equipment.Get(references.GlassSword)

	return glassSwordCount == 0
}

func (aslm *AuthenticSearchLocationManager) alreadyFoundAt(x, y int) bool {
	pos := &references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
	if aslm.gameState.ItemStacksMap.HasItemStackAtPosition(pos) || aslm.gameState.ObjectPresentAt(pos) {
		return true
	}
	for _, corpse := range aslm.gameState.SurfaceCorpses {
		if corpse.Location == aslm.gameState.MapState.PlayerLocation.Location &&
			corpse.Floor == aslm.gameState.MapState.PlayerLocation.Floor && corpse.Position == *pos {
			return true
		}
	}

	return false
}

func (aslm *AuthenticSearchLocationManager) setSkullKeysDay() {
	// Set last reveal day = current game day (original last reveal day = day)
	// See ref:INV-0008.
	aslm.skullKeysFoundDay = aslm.gameState.DateTime.Day
}

func (aslm *AuthenticSearchLocationManager) createFoundResult(location references.SearchLocationReference) *AuthenticSearchResult {
	decoded := references.DecodeSearchLocationItem(location)

	return &AuthenticSearchResult{
		ItemType:      decoded.Item,
		Quantity:      uint16(decoded.PickupQuantity), //nolint:gosec // Original payloads are byte-sized.
		Index:         location.Index,
		Position:      references.Position{X: references.Coordinate(location.X), Y: references.Coordinate(location.Y)},
		ObjectClass:   location.ObjectClass,
		DiscoveryText: decoded.DiscoveryText,
	}
}
