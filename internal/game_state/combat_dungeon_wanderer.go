package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ErrInvalidDungeonWandererCombat indicates invalid generated encounter inputs.
var ErrInvalidDungeonWandererCombat = errors.New("invalid dungeon wanderer combat")

// PrepareDungeonWandererCombatState builds a complete generated encounter but
// leaves transition, origin, and return policy to the caller.
func (g *GameState) PrepareDungeonWandererCombatState(monsterType references.MonsterType) (*CombatState, error) {
	return g.prepareDungeonWandererCombatState(monsterType, g.AsRandomProvider())
}

func (g *GameState) prepareDungeonWandererCombatState(
	monsterType references.MonsterType,
	rng references.RandomProvider,
) (*CombatState, error) {
	if g.activeDungeon == nil {
		return nil, fmt.Errorf("%w: active dungeon is unavailable", ErrInvalidDungeonWandererCombat)
	}

	return g.prepareDungeonWandererCombatStateForFacing(monsterType, g.activeDungeon.facing, rng)
}

func (g *GameState) prepareDungeonWandererCombatStateForFacing(
	monsterType references.MonsterType,
	facing references.DungeonFacing,
	rng references.RandomProvider,
) (*CombatState, error) {
	if g.activeDungeon == nil || g.MapState.PlayerLocation.Location != g.activeDungeon.location {
		return nil, fmt.Errorf("%w: active dungeon is unavailable", ErrInvalidDungeonWandererCombat)
	}
	if !references.IsDungeonWanderingMonster(monsterType) {
		return nil, fmt.Errorf("%w: monster %d is not a dungeon wanderer", ErrInvalidDungeonWandererCombat, monsterType)
	}
	if !facing.IsValid() {
		return nil, fmt.Errorf("%w: invalid dungeon facing %d", ErrInvalidDungeonWandererCombat, facing)
	}
	if rng == nil {
		panic("dungeon wanderer combat requires a random source")
	}
	enemyReference, err := g.enemyReferenceForCombat(monsterType)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidDungeonWandererCombat, err)
	}

	player := g.MapState.PlayerLocation
	combatMap, err := references.BuildGeneratedDungeonCombatMap(
		g.activeDungeon.location,
		&g.activeDungeon.cells,
		player.Floor,
		player.Position,
		facing,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidDungeonWandererCombat, err)
	}

	selectedSlots, monsterCount := dungeonWandererCombatSlots(monsterType, rng)
	combat := NewCombatState(combatMap, dungeonFacingCombatEntry(facing), &g.PartyState)
	g.configureCombatTurnQueue(combat)
	g.initializeCombatPartyRings(combat)
	materializeGeneratedDungeonChest(combat)
	// Dungeon encounter setup fills replacement-monster choices even though homogeneous generated
	// encounters never consume those four replacement choices.
	// See ref:DNG-0010.
	for range 4 {
		_ = rng.Intn(8)
	}
	for slot, selected := range selectedSlots {
		if !selected {
			continue
		}
		combat.AddMonsterToQueue(NewCombatMonster(
			monsterType,
			enemyReference,
			combatMap.MonsterPositions[slot],
			rng,
		))
	}
	if combat.MonsterManager.Count() != monsterCount {
		panic("generated dungeon monster manager and selected slots are out of sync")
	}
	_, queued := combat.TurnQueue.CountRemaining()
	if queued != monsterCount {
		panic("generated dungeon monster manager and turn queue are out of sync")
	}

	return combat, nil
}

func materializeGeneratedDungeonChest(combat *CombatState) {
	chest, found := combat.CombatMapRef.MaterializeGeneratedDungeonChest()
	if !found {
		return
	}
	treasureValue := 3*int(chest.SourceFloor) + 7
	if !combat.AddObject(NewCombatChest(chest.Position, treasureValue, false)) {
		panic("generated dungeon chest has no original object slot")
	}
}

func dungeonWandererCombatSlots(
	monsterType references.MonsterType,
	rng references.RandomProvider,
) ([references.MaxMonstersPerMap]bool, int) {
	groupMaximum, forcedMaximum := dungeonWandererGroupSize(monsterType)
	order := [references.MaxMonstersPerMap]int{}
	for index := range order {
		order[index] = index
	}
	for index := range order {
		randomIndex := rng.Intn(references.MaxMonstersPerMap)
		order[index], order[randomIndex] = order[randomIndex], order[index]
	}

	monsterCount := rng.Intn(groupMaximum) + 1
	if forcedMaximum {
		monsterCount = groupMaximum
	}
	selected := [references.MaxMonstersPerMap]bool{}
	for _, slot := range order[:monsterCount] {
		selected[slot] = true
	}

	return selected, monsterCount
}

func dungeonWandererGroupSize(monsterType references.MonsterType) (int, bool) {
	//nolint:exhaustive // The caller validates membership in the eight-entry dungeon wanderer table.
	switch monsterType {
	case references.MonsterRat:
		return 10, false
	case references.MonsterBat:
		return 16, true
	case references.MonsterSpider:
		return 4, false
	case references.MonsterGhost:
		return 6, false
	case references.MonsterSlime:
		return 16, true
	case references.MonsterGremlin:
		return 13, false
	case references.MonsterGazer:
		return 4, false
	case references.MonsterReaper:
		return 3, false
	default:
		panic("non-wanderer has no dungeon combat group size")
	}
}

func dungeonFacingCombatEntry(facing references.DungeonFacing) CombatEntryDirection {
	switch facing {
	case references.DungeonFacingNorth:
		return CombatEntryNorth
	case references.DungeonFacingEast:
		return CombatEntryEast
	case references.DungeonFacingSouth:
		return CombatEntrySouth
	case references.DungeonFacingWest:
		return CombatEntryWest
	default:
		panic("invalid dungeon combat facing")
	}
}
