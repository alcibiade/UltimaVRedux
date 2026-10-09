package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

const dungeonWandererPlacementAttempts = 8

type dungeonRandomInt func(minimum, maximum int) int

// DungeonWandererMovementResult describes one actor phase without initiating
// the combat transition owned by the encounter workflow.
type DungeonWandererMovementResult struct {
	Monster       references.MonsterType
	From          references.Position
	Target        references.Position
	FinalPosition references.Position
	AttackOrigin  references.Position
	Attempts      int
	Moved         bool
	Contact       bool
}

// generateDungeonWanderer replaces the active occupant using dungeon creature selection
// and genmonxy ordering: type first, then at most eight packed-cell samples.
// See ref:DNG-0010.
func (g *GameState) generateDungeonWanderer() {
	g.generateDungeonWandererWithRandom(g.RandomIntInRange)
}

func (g *GameState) generateDungeonWandererWithRandom(randomInt dungeonRandomInt) bool {
	if randomInt == nil {
		panic("dungeon wanderer generation requires a random source")
	}
	dungeon := g.requireActiveDungeon()
	definitionIndex := randomInt(0, references.DungeonWanderingMonsterCount-1)
	definition, ok := references.DungeonWanderingMonsterDefinitionAt(definitionIndex)
	if !ok {
		panic("dungeon wanderer definition selection is out of range")
	}

	var generated *DungeonWanderer
	party := g.MapState.PlayerLocation
	for range dungeonWandererPlacementAttempts {
		packedPosition := randomInt(0, references.DungeonCellCount/references.DungeonFloorCount-1)
		position := references.Position{
			X: references.Coordinate(packedPosition % int(references.XDungeonMapTiles)),
			Y: references.Coordinate(packedPosition / int(references.XDungeonMapTiles)),
		}
		cell, valid := dungeon.Cell(party.Floor, position)
		if !valid {
			panic("dungeon wanderer placement sample is out of range")
		}
		// genmonxy's "not visible" test is this row/column heuristic, not facing or LOS.
		if !dungeonCellAllowsWandererPlacement(cell.Kind()) ||
			position.X == party.Position.X || position.Y == party.Position.Y {
			continue
		}

		onCeiling := false
		if definition.Monster == references.MonsterSpider || definition.Monster == references.MonsterSlime {
			onCeiling = randomInt(0, 99) > 48
		}
		generated = &DungeonWanderer{
			Monster:          definition.Monster,
			Floor:            party.Floor,
			Position:         position,
			PreviousPosition: position,
			OnCeiling:        onCeiling,
		}

		break
	}
	dungeon.replaceWanderer(generated)

	return generated != nil
}

func dungeonCellAllowsWandererPlacement(kind references.DungeonCellKind) bool {
	return kind < references.DungeonPitOrTrap || kind == references.DungeonOpenChest
}

func (d *DungeonState) replaceWanderer(wanderer *DungeonWanderer) {
	if d.wanderer == nil && wanderer == nil {
		return
	}
	if d.wanderer != nil && wanderer != nil && *d.wanderer == *wanderer {
		return
	}
	d.wanderer = wanderer
	d.revision++
}

// moveDungeonWanderer performs one dungeon creature movement actor phase.
// See ref:DNG-0010.
func (g *GameState) moveDungeonWanderer() DungeonWandererMovementResult {
	return g.moveDungeonWandererWithRandom(g.RandomIntInRange)
}

func (g *GameState) moveDungeonWandererWithRandom(randomInt dungeonRandomInt) DungeonWandererMovementResult {
	if randomInt == nil {
		panic("dungeon wanderer movement requires a random source")
	}
	dungeon := g.requireActiveDungeon()
	wanderer := dungeon.wanderer
	result := DungeonWandererMovementResult{Monster: references.MonsterNone}
	if wanderer == nil || wanderer.Floor != g.MapState.PlayerLocation.Floor {
		return result
	}

	result.Monster = wanderer.Monster
	result.From = wanderer.Position
	result.Target = wanderer.Position
	result.FinalPosition = wanderer.Position
	if wanderer.Monster == references.MonsterReaper {
		return result
	}

	for attempt := 1; attempt <= dungeonWandererPlacementAttempts; attempt++ {
		result.Attempts = attempt
		direction := dungeonWandererDirection(randomInt(0, 3))
		target := direction.GetNewPositionInDirection(&wanderer.Position).GetWrapped(
			references.XDungeonMapTiles,
			references.YDungeonMapTiles,
		)
		cell, valid := dungeon.Cell(wanderer.Floor, *target)
		if !valid {
			panic("wrapped dungeon wanderer movement produced invalid target")
		}
		if !dungeonCellAllowsWandererMovement(cell.Kind()) {
			continue
		}
		if *target == wanderer.PreviousPosition && randomInt(0, 7) != 1 {
			continue
		}

		wanderer.PreviousPosition = wanderer.Position
		wanderer.Position = *target
		result.Target = *target
		result.Moved = true
		if wanderer.Position == g.MapState.PlayerLocation.Position {
			result.Contact = true
			result.AttackOrigin = wanderer.PreviousPosition
			wanderer.Position = wanderer.PreviousPosition
		}
		result.FinalPosition = wanderer.Position
		dungeon.revision++

		return result
	}

	return result
}

func dungeonCellAllowsWandererMovement(kind references.DungeonCellKind) bool {
	// monster_move accepts reserved 0x9* cells even though genmonxy cannot spawn there.
	return kind < references.DungeonClearedRoom &&
		kind != references.DungeonPitOrTrap &&
		kind != references.DungeonEnergyField
}

func dungeonWandererDirection(value int) references.Direction {
	switch value {
	case 0:
		return references.Up
	case 1:
		return references.Right
	case 2:
		return references.Down
	case 3:
		return references.Left
	default:
		panic("dungeon wanderer direction is out of range")
	}
}
