package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// REDUX DEVIATION: CRAFT-0020: the original has no deed maps, so nothing raids them. With the
// option on, monsters suited to the deed's terrain occasionally appear on the party's own deed
// map while the party is there (ADR-014 step 6). The rolls draw the game RNG only on a deed map,
// so every original RNG draw is unchanged with the option off and everywhere else.

// processDeedRaidTurn runs once per deed-map actor phase, after the actors have moved. It
// counts actor turns and, every DeedRaidCheckTurns of them, rolls for a raid (ref:CRAFT-0020).
// Off a deed map it does nothing and draws nothing.
func (g *GameState) processDeedRaidTurn() {
	if !g.IsOnDeedMap() {
		return
	}
	if !g.IsCraftingEnabled() {
		// Switching the option off ends any raid in progress; no new raid starts.
		g.dismissDeedRaiders()
		g.crafting.raidTurns = 0

		return
	}
	if g.crafting.raidsSuppressed || g.HasPendingOverworldContact() || g.CombatScene != nil ||
		len(g.deedRaiders()) > 0 {
		return
	}
	g.crafting.raidTurns++
	if g.crafting.raidTurns < references.DeedRaidCheckTurns {
		return
	}
	g.crafting.raidTurns = 0
	if !g.OneInXOdds(references.DeedRaidOdds) {
		return
	}
	g.startDeedRaid()
}

// startDeedRaid spawns one raid on the loaded deed map and announces it. Game RNG draws, in
// order: the monster (weighted), the number of raiders, then one draw per raider for its square.
// It returns the number of raiders placed; with no free spawn square, none appear and nothing
// is announced.
func (g *GameState) startDeedRaid() int {
	deed := g.crafting.active
	raiders := references.DeedRaiders(deed.Terrain)
	total := 0
	for _, raider := range raiders {
		total += raider.Weight
	}
	roll := g.RandomIntInRange(1, total)
	monster := raiders[len(raiders)-1].Monster
	for _, raider := range raiders {
		if roll <= raider.Weight {
			monster = raider.Monster

			break
		}
		roll -= raider.Weight
	}
	count := g.RandomIntInRange(references.DeedRaidMinRaiders, references.DeedRaidMaxRaiders)

	return g.spawnDeedRaid(monster, count)
}

// spawnDeedRaid places up to count raiders of one monster on random free spawn squares.
func (g *GameState) spawnDeedRaid(monster references.MonsterType, count int) int {
	enemyRef, err := g.enemyReferenceForCombat(monster)
	if err != nil {
		panic("deed raider table names a monster without a reference")
	}
	squares := g.DeedRaidSpawnSquares()
	units := g.CurrentNPCAIController.GetNpcs()
	placed := 0
	for ; placed < count && len(squares) > 0 && units.HasCapacity(); placed++ {
		index := g.RandomIntInRange(0, len(squares)-1)
		square := squares[index]
		squares = append(squares[:index], squares[index+1:]...)
		raider := map_units.NewEnemyNPC(*enemyRef, references.DynamicSpawnIndex)
		raider.MarkAsDeedRaider()
		raider.SetPos(square)
		raider.SetFloor(g.MapState.PlayerLocation.Floor)
		raider.SetVisible(true)
		if !units.AddEnemy(raider) {
			panic("deed raider could not join a map with capacity")
		}
	}
	if placed == 0 {
		return 0
	}
	g.CurrentNPCAIController.FreshenExistingNPCsOnMap()
	g.SystemCallbacks.Message.AddRowStr("Raiders approach!")

	return placed
}

// DeedRaidSpawnSquares lists, in row-major order, the squares of the loaded deed map where a
// raider may appear (ref:CRAFT-0020): within DeedRaidSpawnDepth of the edge (the decorative ring
// and its fringe, which no piece can be built on, so never inside a built room), passable to
// raiders, no resource node or placement, no party, creature or item stack, and at least
// DeedRaidMinSpawnDistance from the party.
func (g *GameState) DeedRaidSpawnSquares() []references.Position {
	deed := g.crafting.active
	if deed == nil || !g.IsOnDeedMap() {
		return nil
	}
	layeredMap := g.GetLayeredMapByCurrentLocation()
	party := g.MapState.PlayerLocation.Position
	units := g.CurrentNPCAIController.GetNpcs()
	floor := g.MapState.PlayerLocation.Floor
	squares := make([]references.Position, 0)
	for y := range references.YSmallMapTiles {
		for x := range references.XSmallMapTiles {
			square := references.Position{X: x, Y: y}
			if references.DeedSquareDepth(square) > references.DeedRaidSpawnDepth ||
				absInt(int(square.X-party.X))+absInt(int(square.Y-party.Y)) < references.DeedRaidMinSpawnDistance {
				continue
			}
			if _, node := g.crafting.activeLayout.NodeAt(square); node {
				continue
			}
			if _, built := deed.Placement(square); built {
				continue
			}
			if units.GetMapUnitAtPositionAndFloorOrNil(square, floor) != nil ||
				g.ItemStacksMap.HasItemStackAtPosition(&square) {
				continue
			}
			if !ai.IsDeedRaiderTerrainPassable(layeredMap.GetSmallMapPersistentTerrainTile(&square, g.DateTime.Hour)) {
				continue
			}
			squares = append(squares, square)
		}
	}

	return squares
}

// deedRaiders returns the raiders on the loaded small map.
func (g *GameState) deedRaiders() []*map_units.NPCEnemy {
	if g.MapState.PlayerLocation.Location.GetMapType() != references.SmallMapType ||
		g.CurrentNPCAIController == nil || g.CurrentNPCAIController.GetNpcs() == nil {
		return nil
	}

	return g.CurrentNPCAIController.GetNpcs().GetEnemiesMatching(func(enemy *map_units.NPCEnemy) bool {
		return enemy.IsDeedRaider()
	})
}

// DeedRaiderCount returns the number of raiders on the map the party is on.
func (g *GameState) DeedRaiderCount() int {
	return len(g.deedRaiders())
}

// dismissDeedRaiders removes every raider from the loaded small map and returns how many left.
func (g *GameState) dismissDeedRaiders() int {
	raiders := g.deedRaiders()
	for _, raider := range raiders {
		if !g.CurrentNPCAIController.GetNpcs().RemoveEnemy(raider) {
			panic("listed deed raider could not be removed")
		}
	}
	if len(raiders) > 0 {
		g.CurrentNPCAIController.FreshenExistingNPCsOnMap()
	}

	return len(raiders)
}

// queueAdjacentDeedRaiderEncounter queues combat with the first raider next to the party, as the
// Shadowlord adjacency check does, before the actors move (ref:CRAFT-0021).
func (g *GameState) queueAdjacentDeedRaiderEncounter() *map_units.NPCEnemy {
	if g.HasPendingOverworldContact() || !g.IsOnDeedMap() {
		return nil
	}
	player := g.MapState.PlayerLocation
	for _, raider := range g.deedRaiders() {
		position := raider.Pos()
		if !raider.IsVisible() || !position.IsNextTo(player.Position, raider.Floor(), player.Floor) {
			continue
		}
		if err := g.QueueSmallMapCombatEncounter(raider, SmallMapEncounterDeedRaiderAdjacency); err != nil {
			return nil
		}

		return raider
	}

	return nil
}

// buildDeedRaidOrigin fills a small-map encounter origin for a raider (ref:CRAFT-0021). The
// whole raid fights together: every raider on the map joins the battle. The battlefield follows
// the deed's terrain, or the brick floor of a town interior for a raider standing on a built
// piece.
func (g *GameState) buildDeedRaidOrigin(
	origin *SmallMapEncounterOrigin, raider *map_units.NPCEnemy, cause SmallMapEncounterCause,
) error {
	if cause != SmallMapEncounterAttack && cause != SmallMapEncounterDeedRaiderAdjacency {
		return ErrInvalidSmallMapEncounter
	}
	deed := g.crafting.active
	if deed == nil || !g.IsOnDeedMap() {
		return ErrInvalidSmallMapEncounter
	}
	origin.SourceEnemy = raider
	origin.SourceKind = SmallMapEncounterDeedRaid
	origin.Position = raider.Pos()
	origin.NPCNumber = raider.MapUnitDetails().NPCNum
	origin.MonsterType = raider.EnemyReference.MonsterType
	origin.CombatMap = references.DeedRaidCombatMap(deed.Terrain)
	if _, built := deed.Placement(origin.Position); built {
		origin.CombatMap = references.BrickCombatMap
	}
	origin.RaiderCount = len(g.deedRaiders())
	origin.Consequences.CompleteSourceTurn = cause == SmallMapEncounterAttack

	return nil
}

// validateDeedRaidOrigin checks that a raid origin still matches the loaded deed map.
func (g *GameState) validateDeedRaidOrigin(origin *SmallMapEncounterOrigin) error {
	if origin.SourceNPC != nil || origin.SourceEnemy == nil || !origin.SourceEnemy.IsDeedRaider() ||
		!g.IsOnDeedMap() || origin.MonsterType != origin.SourceEnemy.EnemyReference.MonsterType ||
		origin.RaiderCount < 1 || origin.RaiderCount != len(g.deedRaiders()) {
		return ErrInvalidSmallMapEncounter
	}

	return nil
}
