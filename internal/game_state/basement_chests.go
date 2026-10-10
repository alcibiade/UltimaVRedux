package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// BasementChestDailyRecord is durable independently of the active map. Nil means
// no known opening; each array entry corresponds to LordBritishBasementChests.
type BasementChestDailyRecord struct {
	LastOpened *datetime.BritannianDate
}

func (record BasementChestDailyRecord) eligible(date datetime.BritannianDate) bool {
	if record.LastOpened == nil {
		return true
	}
	last := *record.LastOpened

	return date.Year > last.Year || date.Year == last.Year &&
		(date.Month > last.Month || date.Month == last.Month && date.Day > last.Day)
}

func dailyBasementChestIndex(chest *map_units.NPCFriendly) (int, bool) {
	if chest == nil || !chest.IsAuthoredSurfaceChest() {
		return 0, false
	}

	return references.BasementChestIndex(chest.NPCReference, chest.MapUnitDetails().NPCNum)
}

// Population is not entry. Once consumed, a definition may only be recreated by
// reconcileBasementChestsOnEntry, even when its last opening was on an older date.
type basementChestPopulationChecker struct {
	state          *GameState
	preserveAbsent bool
}

func (checker basementChestPopulationChecker) IsNPCDead(location references.Location, number int) bool {
	g := checker.state
	if location == references.LordBritishsCastle {
		for index, source := range references.LordBritishBasementChests() {
			if number == source.NPCNumber && (checker.preserveAbsent || g.BasementChestHistory[index].LastOpened != nil) {
				return true
			}
		}
	}

	return g.PartyState.IsNPCDead(location, number)
}

// REDUX DEVIATION: terrain reconstruction and scheduled NPC placement
// recreate ordinary town chests on floor return. Lord British's basement
// chests replenish at most once per calendar day on basement entry to
// discourage repetitive stair-loop farming while retaining repeat visits.
// See ref:CMD-0009.
func (g *GameState) reconcileBasementChestsOnEntry() {
	if g.MapState.PlayerLocation.Location != references.LordBritishsCastle ||
		g.MapState.PlayerLocation.Floor != references.Basement {
		return
	}
	units := g.CurrentNPCAIController.GetNpcs()
	refs := *g.GameReferences.LocationReferences.GetLocationReference(references.LordBritishsCastle).GetNPCReferences()
	for index, source := range references.LordBritishBasementChests() {
		if !g.BasementChestHistory[index].eligible(g.DateTime.DateOnly()) {
			continue
		}
		occupiedIdentity := false
		for _, unit := range *units {
			if unit.MapUnitDetails().NPCNum == source.NPCNumber {
				occupiedIdentity = true

				break
			}
		}
		if occupiedIdentity || !g.canPlaceBasementChest(source.Position, nil) ||
			g.surfaceLootObjectCount() >= map_units.SurfaceLootObjectCapacity {
			continue
		}
		if verified, ok := references.BasementChestIndex(refs[source.NPCNumber], source.NPCNumber); !ok || verified != index {
			panic("Lord British basement chest source identity changed")
		}
		chest := map_units.NewNPCFriendly(refs[source.NPCNumber], source.NPCNumber)
		chest.PlaceAtScheduleSlot(chest.NPCReference.Schedule.DesiredSlot(g.DateTime))
		*units = append(*units, chest)
	}
	g.CurrentNPCAIController.FreshenExistingNPCsOnMap()
}

func (g *GameState) canPlaceBasementChest(position references.Position, except map_units.MapUnit) bool {
	if g.MapState.PlayerLocation.Position == position || g.ItemStacksMap.HasItemStackAtPosition(&position) {
		return false
	}
	tile := g.GetCurrentLayeredMap().GetSmallMapPersistentTerrainTile(&position, g.DateTime.Hour)
	if tile == nil || !tile.IsWalkingPassable() {
		return false
	}
	for _, unit := range *g.CurrentNPCAIController.GetNpcs() {
		if unit != except && unit.IsVisible() && unit.Floor() == references.Basement && unit.Pos() == position {
			return false
		}
	}

	return true
}

// basementChestLifetimes retains actual objects through non-entry controller
// refreshes. Real departure has already discarded transient children.
func (g *GameState) basementChestLifetimes() []*map_units.NPCFriendly {
	if g.MapState.PlayerLocation.Location != references.LordBritishsCastle || g.CurrentNPCAIController == nil {
		return nil
	}

	return g.CurrentNPCAIController.GetNpcs().GetFriendliesMatching(func(chest *map_units.NPCFriendly) bool {
		_, daily := dailyBasementChestIndex(chest)

		return daily || chest.IsTransientSurfaceChest() && chest.NPCReference.Location == references.LordBritishsCastle
	})
}

func (g *GameState) restoreBasementChestLifetimes(chests []*map_units.NPCFriendly) {
	if g.MapState.PlayerLocation.Location != references.LordBritishsCastle {
		return
	}
	changed := len(chests) > 0
	units := g.CurrentNPCAIController.GetNpcs()
	for _, chest := range chests {
		units.RemoveNPCByNumber(chest.MapUnitDetails().NPCNum)
		*units = append(*units, chest)
	}
	if g.MapState.PlayerLocation.Floor == references.Basement {
		for _, unit := range append(map_units.MapUnits(nil), (*units)...) {
			chest, ok := unit.(*map_units.NPCFriendly)
			if !ok {
				continue
			}
			_, daily := dailyBasementChestIndex(chest)
			preserved := false
			for _, previous := range chests {
				preserved = preserved || previous == chest
			}
			if daily && !preserved && !g.canPlaceBasementChest(chest.Pos(), chest) {
				units.RemoveNPCByNumber(chest.MapUnitDetails().NPCNum)
				changed = true
			}
		}
	}
	// Population already placed the generated objects. Refresh only when this
	// Castle-specific reconciliation actually changed their authority.
	if changed {
		g.CurrentNPCAIController.FreshenExistingNPCsOnMap()
	}
}

func (g *GameState) discardSurfaceChestLootOnDeparture() {
	if g.CurrentNPCAIController != nil {
		g.CurrentNPCAIController.GetNpcs().DiscardTransientSurfaceChests()
	}
}

// NPC placement allocates objects only for the active floor. Redux retains
// schedules for all floors; those off-floor records do not consume this budget.
// REDUX DEVIATION: free object-slot selection uses one shared object array. This bounded
// adapter counts active units and unmerged loose entries across Redux's existing
// authorities, retaining their pickup ordering rather than redesigning storage.
// See ref:CMD-0009.
func (g *GameState) surfaceLootObjectCount() int {
	count := 0
	for _, unit := range *g.CurrentNPCAIController.GetNpcs() {
		if unit.IsVisible() && !unit.IsEmptyMapUnit() && unit.Floor() == g.MapState.PlayerLocation.Floor {
			count++
		}
	}
	for _, entry := range g.ItemStacksMap.Entries() {
		count += len(entry.Stack.Items)
	}

	return count
}

func (g *GameState) revealSurfaceChestLoot(position references.Position, value int) {
	found := false
	for _, entry := range references.RollChestLoot(value, g.AsRandomProvider()) {
		if g.surfaceLootObjectCount() >= map_units.SurfaceLootObjectCapacity {
			continue
		}
		message := "a chest!"
		if entry.ChestValue != 0 {
			if g.CurrentNPCAIController.GetNpcs().InsertTransientSurfaceChest(
				g.MapState.PlayerLocation.Location, position, g.MapState.PlayerLocation.Floor, entry.ChestValue) == nil {
				continue
			}
		} else {
			// Preserve each allocation and the existing pile's pickup priority.
			// Same-tile children remain separately actionable by Open.
			stack := references.ItemStack{Items: references.ItemStacks{entry.Item}}
			if err := g.ItemStacksMap.PushUnder(&position, &stack); err != nil {
				panic("preflighted surface chest item placement failed: " + err.Error())
			}
			message = g.GameReferences.InventoryItemReferences.GetListOfItems(&stack)
		}
		if !found {
			g.SystemCallbacks.Message.AddRowStr("Found:")
			found = true
		}
		g.SystemCallbacks.Message.AddRowStr(message)
	}
	if !found {
		g.SystemCallbacks.Message.AddRowStr("Chest empty!")
	}
}

func (g *GameState) surfaceChestPlacementBlocked(position *references.Position, chest *map_units.NPCFriendly) bool {
	if position == nil || *position == g.MapState.PlayerLocation.Position {
		return true
	}
	for _, unit := range *g.CurrentNPCAIController.GetNpcs() {
		if unit != chest && unit.IsVisible() && unit.Floor() == chest.Floor() && unit.Pos() == *position {
			return true
		}
	}

	return false
}
