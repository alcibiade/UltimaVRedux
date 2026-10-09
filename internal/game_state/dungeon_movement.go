package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

// DungeonCommandCompletion controls whether a command only consumes its
// precharge or also runs original dungeon end-player processing.
type DungeonCommandCompletion byte

// Dungeon command completion modes separate precharge-only turns from full processing.
const (
	DungeonCommandOnly DungeonCommandCompletion = iota
	DungeonFullEndPlayer
)

// DungeonMovement identifies relative forward or backward movement.
type DungeonMovement byte

// Dungeon movement directions are relative to the party's facing.
const (
	DungeonAdvance DungeonMovement = iota
	DungeonBackUp
)

// DungeonMoveOutcome identifies the semantic result of a movement attempt.
type DungeonMoveOutcome byte

// Dungeon movement outcomes preserve blocked causes for presentation and follow-up mechanics.
const (
	DungeonMoveMoved DungeonMoveOutcome = iota
	DungeonMoveTerrainBlocked
	DungeonMoveWandererBlocked
	DungeonMoveElectricField
)

// DungeonMoveResult is the detached result of one relative movement attempt.
type DungeonMoveResult struct {
	Movement          DungeonMovement
	Outcome           DungeonMoveOutcome
	From              references.Position
	Target            references.Position
	Completion        DungeonCommandCompletion
	TriggersHiddenPit bool
}

// DungeonTurn identifies a relative facing change.
type DungeonTurn byte

// Dungeon turn directions use the original left, right, and turnaround controls.
const (
	DungeonTurnLeft DungeonTurn = iota
	DungeonTurnRight
	DungeonTurnAround
)

// DungeonTurnOutcome identifies whether facing changed.
type DungeonTurnOutcome byte

// Dungeon turn outcomes distinguish doorway refusal from a completed turn.
const (
	DungeonTurned DungeonTurnOutcome = iota
	DungeonTurnBlockedInDoorway
)

// DungeonTurnResult is the detached result of one facing change.
type DungeonTurnResult struct {
	Turn       DungeonTurn
	Outcome    DungeonTurnOutcome
	Before     references.DungeonFacing
	After      references.DungeonFacing
	Completion DungeonCommandCompletion
}

func (g *GameState) requireActiveDungeon() *DungeonState {
	if g.activeDungeon == nil || g.MapState.PlayerLocation.Location.GetMapType() != references.DungeonMapType {
		panic("dungeon action requires active dungeon state")
	}
	if g.activeDungeon.location != g.MapState.PlayerLocation.Location {
		panic("active dungeon does not match player location")
	}

	return g.activeDungeon
}

// TryDungeonAdvance attempts to move one wrapped cell in the facing direction.
func (g *GameState) TryDungeonAdvance() DungeonMoveResult {
	return g.tryDungeonMove(DungeonAdvance)
}

// TryDungeonBackUp attempts to move one wrapped cell opposite the facing direction.
func (g *GameState) TryDungeonBackUp() DungeonMoveResult {
	return g.tryDungeonMove(DungeonBackUp)
}

func (g *GameState) tryDungeonMove(movement DungeonMovement) DungeonMoveResult {
	dungeon := g.requireActiveDungeon()
	g.claimDungeonCommandAction(DungeonFullEndPlayer)
	from := g.MapState.PlayerLocation.Position
	direction := dungeon.facing.Direction()
	if movement == DungeonBackUp {
		direction = direction.GetOppositeDirection()
	}
	target := direction.GetNewPositionInDirection(&from).GetWrapped(
		references.XDungeonMapTiles,
		references.YDungeonMapTiles,
	)
	result := DungeonMoveResult{
		Movement:   movement,
		Outcome:    DungeonMoveTerrainBlocked,
		From:       from,
		Target:     *target,
		Completion: DungeonFullEndPlayer,
	}
	dungeon.recordLastDirectionFromFacing()

	if dungeon.occupied(g.MapState.PlayerLocation.Floor, *target) {
		result.Outcome = DungeonMoveWandererBlocked

		return result
	}
	cell, ok := dungeon.Cell(g.MapState.PlayerLocation.Floor, *target)
	if !ok {
		panic("wrapped dungeon movement produced invalid target")
	}
	if cell.Raw() == 0x83 {
		result.Outcome = DungeonMoveElectricField
		g.processDungeonElectricField()

		return result
	}
	if !dungeonCellAllowsMovement(cell.Kind(), movement) {
		return result
	}

	result.Outcome = DungeonMoveMoved
	result.TriggersHiddenPit = isDungeonHiddenPit(cell)
	g.TransitionToLocation(
		g.MapState.PlayerLocation.Location,
		g.MapState.PlayerLocation.Floor,
		*target,
		TransitionPositionOnly,
		false,
	)

	return result
}

func dungeonCellAllowsMovement(kind references.DungeonCellKind, movement DungeonMovement) bool {
	switch kind {
	case references.DungeonWall, references.DungeonSpecialWall, references.DungeonHiddenDoor:
		return false
	case references.DungeonClearedRoom, references.DungeonLiveRoom:
		return movement == DungeonAdvance
	case references.DungeonPassage,
		references.DungeonLadderUp,
		references.DungeonLadderDown,
		references.DungeonLadderBoth,
		references.DungeonClosedChest,
		references.DungeonFountain,
		references.DungeonPitOrTrap,
		references.DungeonOpenChest,
		references.DungeonEnergyField,
		references.DungeonReserved,
		references.DungeonRevealedDoor:
		return true
	}

	panic("invalid dungeon cell kind")
}

// TurnDungeon applies one original relative facing change.
func (g *GameState) TurnDungeon(turn DungeonTurn) DungeonTurnResult {
	dungeon := g.requireActiveDungeon()
	g.claimDungeonCommandAction(DungeonCommandOnly)
	result := DungeonTurnResult{
		Turn:       turn,
		Outcome:    DungeonTurned,
		Before:     dungeon.facing,
		After:      dungeon.facing,
		Completion: DungeonCommandOnly,
	}
	cell, ok := dungeon.Cell(g.MapState.PlayerLocation.Floor, g.MapState.PlayerLocation.Position)
	if !ok {
		panic("current dungeon position is invalid")
	}
	if turn != DungeonTurnAround && cell.Kind() == references.DungeonRevealedDoor {
		dungeon.recordLastDirectionFromFacing()
		result.Outcome = DungeonTurnBlockedInDoorway

		return result
	}

	switch turn {
	case DungeonTurnLeft:
		dungeon.setFacingAndLastDirection(dungeon.facing.TurnLeft())
	case DungeonTurnRight:
		dungeon.setFacingAndLastDirection(dungeon.facing.TurnRight())
	case DungeonTurnAround:
		dungeon.setFacingAndLastDirection(dungeon.facing.TurnAround())
	default:
		panic("invalid dungeon turn")
	}
	result.After = dungeon.facing

	return result
}
