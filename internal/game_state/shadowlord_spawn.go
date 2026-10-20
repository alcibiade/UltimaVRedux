package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	rng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

// SpawnShadowlordNPC creates and spawns a Shadowlord enemy NPC at the specified position.
// Called when a Shadowlord is successfully summoned at a Keep.
// The Shadowlord spawns 2 tiles north of the player (at playerY - 2).
//
// Based on original command rules - Shadowlord appears above player after summoning.
// See ref:SHADOW-0002.
func (g *GameState) SpawnShadowlordNPC(lord references.Shadowlord) bool {
	return g.insertShadowlordNPC(lord) != nil
}

func (g *GameState) insertShadowlordNPC(lord references.Shadowlord) *map_units.NPCEnemy {
	playerPos := g.MapState.PlayerLocation.Position

	// Calculate spawn position: 2 tiles north of player
	spawnPos := references.Position{
		X: playerPos.X,
		Y: playerPos.Y - 2,
	}

	return g.insertShadowlordNPCAtPosition(lord, spawnPos, true)
}

// SpawnShadowlordNPCAtPosition creates a Shadowlord enemy NPC at an explicit
// passable position. Source-fixed Yell and automatic placement use their
// unexported terrain policy instead.
//
// Parameters:
//   - lord: Which Shadowlord to spawn
//   - spawnPos: The position to spawn at
//
// Returns true if spawn was successful, false otherwise.
func (g *GameState) SpawnShadowlordNPCAtPosition(lord references.Shadowlord, spawnPos references.Position) bool {
	return g.insertShadowlordNPCAtPosition(lord, spawnPos, false) != nil
}

func (g *GameState) insertShadowlordNPCAtPosition(
	lord references.Shadowlord,
	spawnPos references.Position,
	sourceFixedPosition bool,
) *map_units.NPCEnemy {
	return g.insertShadowlordNPCAtFloorPosition(
		lord,
		spawnPos,
		g.MapState.PlayerLocation.Floor,
		sourceFixedPosition,
	)
}

func (g *GameState) insertShadowlordNPCAtFloorPosition(
	lord references.Shadowlord,
	spawnPos references.Position,
	spawnFloor references.FloorNumber,
	sourceFixedPosition bool,
) *map_units.NPCEnemy {
	logging.Debug(logging.CategoryGameState,
		"SpawnShadowlordNPCAtPosition: spawning %s at (%d, %d) floor %d",
		lord.String(), spawnPos.X, spawnPos.Y, spawnFloor)

	if g.ShadowlordState == nil || !g.ShadowlordState.IsAlive(lord) ||
		g.GameReferences == nil || g.GameReferences.EnemyReferences == nil ||
		g.GameReferences.LocationReferences == nil {
		logging.Error(logging.CategoryGameState,
			"SpawnShadowlordNPCAtPosition: Shadowlord state or references are unavailable")

		return nil
	}

	shadowlordEnemy := g.enemyReference(references.MonsterShadowlord)
	if shadowlordEnemy == nil || shadowlordEnemy.KeyFrameTile == nil {
		logging.Error(logging.CategoryGameState,
			"SpawnShadowlordNPCAtPosition: static Shadowlord reference is unavailable")

		return nil
	}

	if g.CurrentNPCAIController == nil {
		logging.Error(logging.CategoryGameState,
			"SpawnShadowlordNPCAtPosition: CurrentNPCAIController is nil")

		return nil
	}
	units := g.CurrentNPCAIController.GetNpcs()
	if units == nil || !units.HasCapacity() {
		logging.Error(logging.CategoryGameState,
			"SpawnShadowlordNPCAtPosition: map-unit controller is unavailable or full")

		return nil
	}

	location := g.MapState.PlayerLocation.Location
	if location.GetMapType() != references.SmallMapType {
		logging.Error(logging.CategoryGameState,
			"SpawnShadowlordNPCAtPosition: Shadowlords require a small map")

		return nil
	}
	locationReference := g.GameReferences.LocationReferences.GetLocationReference(location)
	if locationReference == nil {
		logging.Error(logging.CategoryGameState,
			"SpawnShadowlordNPCAtPosition: current location reference is unavailable")

		return nil
	}
	minimumFloor, maximumFloor := locationReference.GetFloorMinMax()
	if spawnFloor < minimumFloor || spawnFloor > maximumFloor {
		logging.Error(logging.CategoryGameState,
			"SpawnShadowlordNPCAtPosition: floor %d is invalid for %s", spawnFloor, location.String())

		return nil
	}
	if spawnPos.X < 0 || spawnPos.Y < 0 ||
		spawnPos.X > locationReference.GetMaxX() || spawnPos.Y > locationReference.GetMaxY() {
		logging.Error(logging.CategoryGameState,
			"SpawnShadowlordNPCAtPosition: position (%d, %d) is out of bounds", spawnPos.X, spawnPos.Y)

		return nil
	}

	layeredMap := g.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, spawnFloor)
	if layeredMap == nil {
		logging.Error(logging.CategoryGameState,
			"SpawnShadowlordNPCAtPosition: current floor is not materialized")

		return nil
	}
	spawnTile := layeredMap.GetTopTile(&spawnPos)
	// Original Yell and town initialization force their exact source coordinates;
	// arbitrary callers must use monster spawn terrain. The ritual's blue Flame
	// remains valid because original Shadowlord collision explicitly permits it.
	if spawnTile == nil || (!sourceFixedPosition &&
		!shadowlordEnemy.CanSpawnToTile(spawnTile) && !spawnTile.Is(indexes.BlueFlame)) {
		logging.Error(logging.CategoryGameState,
			"SpawnShadowlordNPCAtPosition: position (%d, %d) is not passable", spawnPos.X, spawnPos.Y)

		return nil
	}
	if spawnFloor == g.MapState.PlayerLocation.Floor && spawnPos == g.MapState.PlayerLocation.Position {
		logging.Error(logging.CategoryGameState,
			"SpawnShadowlordNPCAtPosition: spawn position is occupied by the party")

		return nil
	}
	for _, unit := range *units {
		if unit == nil || !unit.IsVisible() || unit.IsEmptyMapUnit() {
			continue
		}
		if unit.Pos() == spawnPos && unit.Floor() == spawnFloor {
			logging.Error(logging.CategoryGameState,
				"SpawnShadowlordNPCAtPosition: spawn position is occupied")

			return nil
		}
		enemy, ok := unit.(*map_units.NPCEnemy)
		if ok && enemy.Variant() == map_units.EnemyVariantShadowlord {
			logging.Error(logging.CategoryGameState,
				"SpawnShadowlordNPCAtPosition: a physical Shadowlord is already present")

			return nil
		}
	}

	npcEnemy := map_units.NewEnemyNPC(*shadowlordEnemy, references.DynamicSpawnIndex)
	npcEnemy.SetShadowlordIdentity(lord)
	npcEnemy.SetPos(spawnPos)
	npcEnemy.SetFloor(spawnFloor)
	npcEnemy.SetVisible(true)

	if !units.AddEnemy(npcEnemy) {
		logging.Error(logging.CategoryGameState,
			"SpawnShadowlordNPCAtPosition: failed to add Shadowlord enemy to map")

		return nil
	}
	inserted := matchingShadowlordAt(units, spawnPos, spawnFloor, lord)
	if inserted == nil {
		panic("inserted Shadowlord cannot be recovered from map-unit authority")
	}

	logging.Info(logging.CategoryGameState,
		"SpawnShadowlordNPCAtPosition: %s spawned at (%d, %d)",
		lord.String(), spawnPos.X, spawnPos.Y)

	return inserted
}

// IsShadowlordNPCPresent checks if a Shadowlord NPC is currently on the map.
func (g *GameState) IsShadowlordNPCPresent() bool {
	if g.CurrentNPCAIController == nil {
		return false
	}

	enemies := g.CurrentNPCAIController.GetNpcs().GetEnemies()

	for i := range enemies {
		enemy := &enemies[i]
		if enemy.Variant() == map_units.EnemyVariantShadowlord {
			return true
		}
	}

	return false
}

func (g *GameState) ensureSingleAutomaticShadowlordNPC(
	lord references.Shadowlord,
	location references.Location,
) bool {
	if g.CurrentNPCAIController == nil || g.CurrentNPCAIController.GetNpcs() == nil {
		return false
	}

	units := g.CurrentNPCAIController.GetNpcs()
	unitsBefore := append([]map_units.MapUnit(nil), (*units)...)
	type removedShadowlord struct {
		enemy   *map_units.NPCEnemy
		visible bool
	}
	removed := make([]removedShadowlord, 0)
	var retained *map_units.NPCEnemy
	for _, unit := range unitsBefore {
		enemy, ok := unit.(*map_units.NPCEnemy)
		if !ok || enemy.Variant() != map_units.EnemyVariantShadowlord {
			continue
		}
		identity, identified := enemy.ShadowlordIdentity()
		if retained == nil && enemy.IsVisible() && identified && identity == lord {
			retained = enemy

			continue
		}
		wasVisible := enemy.IsVisible()
		if !units.RemoveEnemy(enemy) {
			panic("duplicate physical Shadowlord could not be removed")
		}
		removed = append(removed, removedShadowlord{enemy: enemy, visible: wasVisible})
	}
	if retained != nil {
		return true
	}

	if g.SpawnAutomaticShadowlordNPC(lord, location) {
		return true
	}
	for _, removedLord := range removed {
		removedLord.enemy.SetVisible(removedLord.visible)
	}
	units.ReplaceWith(unitsBefore)

	return false
}

func (g *GameState) clearPhysicalShadowlords() {
	if g.CurrentNPCAIController == nil || g.CurrentNPCAIController.GetNpcs() == nil {
		return
	}
	units := g.CurrentNPCAIController.GetNpcs()
	for _, unit := range append([]map_units.MapUnit(nil), (*units)...) {
		enemy, ok := unit.(*map_units.NPCEnemy)
		if !ok || enemy.Variant() != map_units.EnemyVariantShadowlord {
			continue
		}
		if !units.RemoveEnemy(enemy) {
			panic("physical Shadowlord could not be cleared")
		}
	}
}

// ApplyShadowlordEffectsToNPCs applies Shadowlord presence effects to nearby NPCs.
// Based on original Ultima V town rules NPC influence:
//   - Astaroth (Hatred): ~50% of regular NPCs become hostile
//   - Nosfentor (Cowardice): ~50% of regular NPCs flee
//   - Faulinei (Falsehood): No NPC behavior change
//
// This only works on small maps where Shadowlords can be summoned.
// Note: This function is for automatic assigned-town presence, not yell summoning.
// The AutomaticPresence state is tracked so NPC effects persist across floor changes.
// See ref:SHADOW-0002.
func (g *GameState) ApplyShadowlordEffectsToNPCs(lord references.Shadowlord) {
	if g.CurrentNPCAIController == nil {
		logging.Warn(logging.CategoryGameState,
			"ApplyShadowlordEffectsToNPCs: CurrentNPCAIController is nil, NPC effects skipped")

		return
	}

	// Shadowlord effects only apply on small maps
	smallMapController, ok := g.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
	if !ok {
		logging.Debug(logging.CategoryGameState,
			"ApplyShadowlordEffectsToNPCs: not on small map, skipping NPC effects")

		return
	}

	// Apply effects with 50% random chance per NPC (matching original game)
	smallMapController.SetNPCsReactToShadowlord(lord, func() bool {
		return g.OneInXOdds(2) // 50% chance
	})

	logging.Info(logging.CategoryGameState,
		"ApplyShadowlordEffectsToNPCs: applied %s effects to NPCs (automatic presence)", lord.String())
}

// ApplyFaulineiShopTheft removes a draw of 1..64 gold, floored at zero,
// only for automatic assigned-town Faulinei presence. Yell summons do not
// qualify. Return the amount actually removed. Concrete transactions own
// the payment/theft/grant boundary and exclusions. ref:MERCH-0003
func (g *GameState) ApplyFaulineiShopTheft() uint16 {
	// Only steal during Faulinei's AUTOMATIC presence
	// Yell summoning (ShadowlordState.Summoned) does NOT trigger shop theft
	if g.ShadowlordState.AutomaticPresence != references.Faulinei {
		return 0
	}

	// Calculate theft amount: 1-64 gold (random)
	// Original: random(1,64) in original shop rules
	// See ref:SHADOW-0002.
	theftAmount := uint16(g.RollDice(64))

	// Don't steal more than the player has
	currentGold := g.PartyState.Inventory.Gold.Get()
	if theftAmount > currentGold {
		theftAmount = currentGold
	}

	// No gold to steal
	if theftAmount == 0 {
		return 0
	}

	// Deduct the gold
	g.PartyState.Inventory.Gold.DecrementBy(theftAmount)

	logging.Info(logging.CategoryGameState,
		"ApplyFaulineiShopTheft: Faulinei stole %d gold during shop transaction", theftAmount)

	return theftAmount
}

// FaulineiConversationTheftKind identifies the inventory branch selected by
// conversation theft without exposing more detail in player-facing output.
// See ref:SHADOW-0002.
type FaulineiConversationTheftKind string

// Faulinei conversation theft kinds identify the selected original inventory branch.
const (
	FaulineiConversationTheftNone      FaulineiConversationTheftKind = ""
	FaulineiConversationTheftKey       FaulineiConversationTheftKind = "key"
	FaulineiConversationTheftGem       FaulineiConversationTheftKind = "gem"
	FaulineiConversationTheftTorch     FaulineiConversationTheftKind = "torch"
	FaulineiConversationTheftEquipment FaulineiConversationTheftKind = "equipment"
	FaulineiConversationTheftPotion    FaulineiConversationTheftKind = "potion"
	FaulineiConversationTheftScroll    FaulineiConversationTheftKind = "scroll"
	FaulineiConversationTheftGold      FaulineiConversationTheftKind = "gold"
)

// FaulineiConversationTheftResult reports the authoritative inventory mutation.
// ItemID is populated for equipment, potion, and scroll results.
type FaulineiConversationTheftResult struct {
	Kind     FaulineiConversationTheftKind
	ItemID   int
	Quantity uint16
}

// Occurred reports whether automatic Faulinei presence invoked the theft.
// The original still reports theft when the final gold fallback clamps to zero.
func (result FaulineiConversationTheftResult) Occurred() bool {
	return result.Kind != FaulineiConversationTheftNone
}

// ApplyFaulineiConversationTheft applies conversation theft after a completed
// normal conversation. Workflow routing owns the completion check; this method
// rejects every presence source except automatic Faulinei.
// See ref:SHADOW-0002.
func (g *GameState) ApplyFaulineiConversationTheft() FaulineiConversationTheftResult {
	if g.ShadowlordState == nil ||
		g.ShadowlordState.AutomaticPresence != references.Faulinei {
		return FaulineiConversationTheftResult{}
	}

	inventory := &g.PartyState.Inventory
	if inventory.Provisions.Keys.Get() != 0 ||
		inventory.Provisions.Gems.Get() != 0 ||
		inventory.Provisions.Torches.Get() != 0 {
		for {
			switch g.RandomIntInRange(0, 2) {
			case 0:
				if inventory.Provisions.Keys.DecrementByOne() {
					return FaulineiConversationTheftResult{Kind: FaulineiConversationTheftKey, Quantity: 1}
				}
			case 1:
				if inventory.Provisions.Gems.DecrementByOne() {
					return FaulineiConversationTheftResult{Kind: FaulineiConversationTheftGem, Quantity: 1}
				}
			case 2:
				if inventory.Provisions.Torches.DecrementByOne() {
					return FaulineiConversationTheftResult{Kind: FaulineiConversationTheftTorch, Quantity: 1}
				}
			}
		}
	}

	for equipment := references.Ankh; equipment >= references.LeatherHelm; equipment-- {
		if inventory.Equipment.DecrementByOne(equipment) {
			return FaulineiConversationTheftResult{
				Kind: FaulineiConversationTheftEquipment, ItemID: int(equipment), Quantity: 1,
			}
		}
	}
	for potion := references.White; potion >= references.Blue; potion-- {
		if inventory.Potions.DecrementByOne(potion) {
			return FaulineiConversationTheftResult{
				Kind: FaulineiConversationTheftPotion, ItemID: int(potion), Quantity: 1,
			}
		}
	}
	for scroll := references.ScrollAnTym; scroll >= references.ScrollVasLor; scroll-- {
		if inventory.Scrolls.DecrementByOne(scroll) {
			return FaulineiConversationTheftResult{
				Kind: FaulineiConversationTheftScroll, ItemID: int(scroll), Quantity: 1,
			}
		}
	}

	quantity := min(uint16(g.RandomIntInRange(1, 15)), inventory.Gold.Get())
	inventory.Gold.DecrementBy(quantity)

	return FaulineiConversationTheftResult{Kind: FaulineiConversationTheftGold, Quantity: quantity}
}

// ApplyShadowlordMapEffects applies Shadowlord presence map effects to the current small map.
// Based on original Ultima V town rules vegetation effects:
//   - Crops (WheatInField) become plowed fields (PlowedField)
//   - Fruit trees (Tree) become stumps (DeadTree)
//   - 7/8 chance per affected tile
//
// This only applies during automatic assigned-town presence, not yell summoning.
// Returns (destroyed count, total affected tiles found).
//
// See ref:SHADOW-0002.
//
//nolint:unused // Will be called by moon-based presence system when implemented
func (g *GameState) ApplyShadowlordMapEffects(lord references.Shadowlord) (int, int) {
	// Only apply on small maps during automatic presence
	if g.MapState.PlayerLocation.Location.GetMapType() != references.SmallMapType {
		return 0, 0
	}

	layeredMap := g.GetCurrentLayeredMap()
	if layeredMap == nil {
		return 0, 0
	}

	destroyed := 0
	total := 0
	// REDUX DEVIATION: vegetation effects reseeds from wall time after its
	// day-seeded pass. Redux uses an isolated deterministic stream so later gameplay RNG is unchanged.
	// See ref:SHADOW-0002.
	effectRNG := rng.New(uint64(g.DateTime.Day))

	// Scan the entire map for affected tiles
	// Use the map dimensions from LayeredMap
	mapWidth := int(layeredMap.XMaxTilesPerMap)
	mapHeight := int(layeredMap.YMaxTilesPerMap)

	for y := range mapHeight {
		for x := range mapWidth {
			pos := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
			tile := layeredMap.GetTileByLayer(map_state.MapLayer, &pos)

			if tile == nil {
				continue
			}

			// Check if this tile is affected
			replacement := getShadowlordMapEffectReplacement(tile.Index)
			if replacement == tile.Index {
				// Not an affected tile
				continue
			}

			total++

			// Original random(0,7): zero survives and every nonzero result destroys.
			if effectRNG.Intn(8) == 0 {
				// 1/8 chance - tile survives
				continue
			}

			// Replace the tile using SetTileByLayer
			layeredMap.SetTileByLayer(map_state.MapLayer, &pos, replacement)
			destroyed++
		}
	}

	if destroyed > 0 {
		logging.Info(logging.CategoryGameState,
			"ApplyShadowlordMapEffects: %s destroyed %d of %d crop/tree tiles",
			lord.String(), destroyed, total)
	}

	return destroyed, total
}

// getShadowlordMapEffectReplacement returns the replacement tile for Shadowlord map effects.
// Based on original town rules:
//   - WheatInField -> PlowedField (crops destroyed)
//   - Tree -> DeadTree (fruit trees become stumps)
//
// Returns the same index if the tile is not affected.
// See ref:SHADOW-0002.
func getShadowlordMapEffectReplacement(tileIndex indexes.SpriteIndex) indexes.SpriteIndex {
	//nolint:exhaustive // Only crops and fruit trees change; every other tile is preserved.
	switch tileIndex {
	case indexes.WheatInField:
		return indexes.PlowedField
	case indexes.Tree:
		return indexes.DeadTree
	default:
		return tileIndex
	}
}
