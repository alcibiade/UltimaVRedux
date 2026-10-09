package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

type dungeonPackedCellTarget struct {
	floor    references.FloorNumber
	position references.Position
	cell     references.DungeonCell
}

// DungeonPackedSpellResult reports one authoritative dungeon spell mutation.
type DungeonPackedSpellResult struct {
	Change   DungeonCellChange
	Applied  bool
	Disarmed bool
}

// DungeonFloorSpellOutcome identifies one prepared vertical-travel result.
type DungeonFloorSpellOutcome byte

const (
	// DungeonFloorSpellFailed reports that no vertical travel occurred.
	DungeonFloorSpellFailed DungeonFloorSpellOutcome = iota
	// DungeonFloorSpellChangedFloor reports an internal dungeon floor transition.
	DungeonFloorSpellChangedFloor
	// DungeonFloorSpellExited reports a transition out of dungeon authority.
	DungeonFloorSpellExited
)

// DungeonFloorSpellResult reports authoritative vertical travel and command ownership.
type DungeonFloorSpellResult struct {
	Outcome          DungeonFloorSpellOutcome
	Direction        DungeonKlimbDirection
	CommandCompleted bool
}

type dungeonFloorSpellPlan struct {
	direction   DungeonKlimbDirection
	targetFloor references.FloorNumber
	valid       bool
	exit        bool
}

func isDungeonPackedSpell(spell references.Spell) bool {
	//nolint:exhaustive // Only packed-cell dungeon spells return true.
	switch spell {
	case references.AnSanct,
		references.InFlamGrav,
		references.InNoxGrav,
		references.InZuGrav,
		references.InSanctGrav,
		references.AnGrav:
		return true
	default:
		return false
	}
}

func isDungeonFloorSpell(spell references.Spell) bool {
	return spell == references.UusPor || spell == references.DesPor
}

// CanCastDungeonFloorSpell validates vertical travel before prepared-spell commitment.
func (g *GameState) CanCastDungeonFloorSpell(spell references.Spell) bool {
	return g.planDungeonFloorSpell(spell).valid
}

// CastDungeonFloorSpell applies one validated vertical travel spell. Boundary
// exits consume the precharged dungeon command before leaving dungeon authority.
func (g *GameState) CastDungeonFloorSpell(
	token DungeonCommandCycle,
	spell references.Spell,
) DungeonFloorSpellResult {
	plan := g.planDungeonFloorSpell(spell)
	result := DungeonFloorSpellResult{Outcome: DungeonFloorSpellFailed, Direction: plan.direction}
	if !plan.valid {
		return result
	}
	if plan.exit {
		g.CompleteDungeonCommandCycle(token, DungeonCommandOnly)
		g.exitDungeonToWorld(plan.direction)
		result.Outcome = DungeonFloorSpellExited
		result.CommandCompleted = true

		return result
	}

	lastDirection := references.DungeonArrivedFromAbove
	if plan.direction == DungeonKlimbUp {
		lastDirection = references.DungeonArrivedFromBelow
	}
	g.transitionDungeonFloor(plan.targetFloor, lastDirection)
	g.generateDungeonWanderer()
	result.Outcome = DungeonFloorSpellChangedFloor

	return result
}

func (g *GameState) planDungeonFloorSpell(spell references.Spell) dungeonFloorSpellPlan {
	direction := DungeonKlimbUp
	//nolint:exhaustive // This planner accepts only the two floor spells and panics otherwise.
	switch spell {
	case references.UusPor:
	case references.DesPor:
		direction = DungeonKlimbDown
	default:
		panic("dungeon floor spell plan requested for unsupported spell")
	}

	plan := dungeonFloorSpellPlan{direction: direction, targetFloor: g.MapState.PlayerLocation.Floor}
	dungeon := g.requireActiveDungeon()
	if dungeon.location == references.Doom {
		return plan
	}

	if direction == DungeonKlimbUp {
		if plan.targetFloor == 0 {
			plan.valid = true
			plan.exit = true

			return plan
		}
		plan.targetFloor--
	} else {
		if plan.targetFloor == references.DungeonFloorCount-1 {
			plan.valid = true
			plan.exit = true

			return plan
		}
		plan.targetFloor++
	}

	cell, ok := dungeon.Cell(plan.targetFloor, g.MapState.PlayerLocation.Position)
	if !ok {
		panic("dungeon floor spell target is invalid")
	}
	plan.valid = cell.Raw()&0xf0 == 0

	return plan
}

// CanCastDungeonPackedSpell reports whether the spell's implicit dungeon target
// can be mutated without committing prepared-spell resources.
func (g *GameState) CanCastDungeonPackedSpell(spell references.Spell) bool {
	_, ok := g.dungeonPackedSpellTarget(spell)

	return ok
}

// CastDungeonPackedSpell applies one prepared spell to its implicit packed-cell target.
func (g *GameState) CastDungeonPackedSpell(spell references.Spell) DungeonPackedSpellResult {
	target, ok := g.dungeonPackedSpellTarget(spell)
	if !ok {
		return DungeonPackedSpellResult{}
	}

	if spell == references.AnGrav {
		result := g.DispelCurrentOrFacingDungeonField()
		if !result.Cleared {
			panic("validated dungeon An Grav target disappeared")
		}

		return DungeonPackedSpellResult{Change: result.Change, Applied: true, Disarmed: false}
	}

	after := references.DungeonCell(0)
	disarmed := false
	if spell == references.AnSanct {
		after = references.DungeonCell(0x70 | (target.cell.Raw() & 0x08))
		disarmed = target.cell.Raw()&0x01 != 0
	} else {
		after = references.DungeonCell(dungeonFieldCellForSpell(spell) | (target.cell.Raw() & 0x08))
	}

	change, err := g.TransformDungeonCell(target.floor, target.position, func(before references.DungeonCell) references.DungeonCell {
		if before != target.cell {
			panic("validated dungeon spell target changed before mutation")
		}

		return after
	})
	if err != nil {
		panic(err)
	}

	return DungeonPackedSpellResult{Change: change, Applied: true, Disarmed: disarmed}
}

func (g *GameState) dungeonPackedSpellTarget(spell references.Spell) (dungeonPackedCellTarget, bool) {
	//nolint:exhaustive // Every packed-cell dungeon spell is handled explicitly.
	switch spell {
	case references.AnSanct:
		return g.currentOrFacingDungeonCell(references.DungeonClosedChest)
	case references.AnGrav:
		return g.currentOrFacingDungeonCell(references.DungeonEnergyField)
	case references.InFlamGrav, references.InNoxGrav, references.InZuGrav, references.InSanctGrav:
		target := g.facingDungeonCell()

		return target, target.cell.Raw()&0xf7 == 0
	default:
		panic("dungeon packed spell target requested for unsupported spell")
	}
}

func (g *GameState) currentOrFacingDungeonCell(kind references.DungeonCellKind) (dungeonPackedCellTarget, bool) {
	dungeon := g.requireActiveDungeon()
	floor := g.MapState.PlayerLocation.Floor
	current := g.MapState.PlayerLocation.Position
	cell, ok := dungeon.Cell(floor, current)
	if !ok {
		panic("current dungeon spell target is invalid")
	}
	if cell.Kind() == kind {
		return dungeonPackedCellTarget{floor: floor, position: current, cell: cell}, true
	}

	target := g.facingDungeonCell()

	return target, target.cell.Kind() == kind
}

func (g *GameState) facingDungeonCell() dungeonPackedCellTarget {
	dungeon := g.requireActiveDungeon()
	floor := g.MapState.PlayerLocation.Floor
	current := g.MapState.PlayerLocation.Position
	facing := dungeon.facing.Direction().GetNewPositionInDirection(&current)
	if facing == nil {
		panic("dungeon facing produced no spell target")
	}
	position := *facing.GetWrapped(references.XDungeonMapTiles, references.YDungeonMapTiles)
	cell, ok := dungeon.Cell(floor, position)
	if !ok {
		panic("facing dungeon spell target is invalid")
	}

	return dungeonPackedCellTarget{floor: floor, position: position, cell: cell}
}

func dungeonFieldCellForSpell(spell references.Spell) byte {
	// Original dungeon field creation stores fire, poison, sleep, then electric.
	//nolint:exhaustive // Only dungeon field-creation spells are accepted.
	// See ref:DNG-0008.
	switch spell {
	case references.InFlamGrav:
		return 0x82
	case references.InNoxGrav:
		return 0x81
	case references.InZuGrav:
		return 0x80
	case references.InSanctGrav:
		return 0x83
	default:
		panic("dungeon field cell requested for unsupported spell")
	}
}
