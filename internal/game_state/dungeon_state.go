package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

var (
	errInvalidDungeonCell = errors.New("invalid dungeon cell coordinate")
)

// DungeonWanderer is the semantic state required for one original dungeon
// wanderer. Combat transition orchestration belongs to the encounter workflow.
// See ref:DNG-0001.
type DungeonWanderer struct {
	Monster          references.MonsterType
	Floor            references.FloorNumber
	Position         references.Position
	PreviousPosition references.Position
	OnCeiling        bool
}

// DungeonState owns one active visit's mutable semantic state.
type DungeonState struct {
	location         references.Location
	cells            references.DungeonCells
	chestAssessments DungeonChestAssessments

	facing        references.DungeonFacing
	lastDirection references.DungeonLastDirection
	wanderer      *DungeonWanderer
	// A pit chain that lands in a fixed room regenerates only after that room returns.
	generateWandererAfterRoom bool
	roomContinuation          dungeonRoomContinuation

	quicknessSkipNext bool
	revision          uint64
}

// DungeonStateSnapshot is a detached complete view for rendering, debug, and
// future persistence. Projection caches are deliberately excluded.
type DungeonStateSnapshot struct {
	Location          references.Location
	Cells             references.DungeonCells
	ChestAssessments  DungeonChestAssessments
	Facing            references.DungeonFacing
	LastDirection     references.DungeonLastDirection
	Wanderer          *DungeonWanderer
	QuicknessSkipNext bool
	Revision          uint64
}

// DungeonCellChange records one authoritative packed-cell transformation.
type DungeonCellChange struct {
	Floor    references.FloorNumber
	Position references.Position
	Before   references.DungeonCell
	After    references.DungeonCell
	Revision uint64
}

func (d *DungeonState) hasPendingRoomContinuation() bool {
	return d != nil && (d.generateWandererAfterRoom ||
		d.roomContinuation.encounterPending ||
		d.roomContinuation.finishEndPlayer)
}

func newDungeonState(
	location references.Location,
	cells references.DungeonCells,
	facing references.DungeonFacing,
	lastDirection references.DungeonLastDirection,
) (*DungeonState, error) {
	if location.GetMapType() != references.DungeonMapType {
		return nil, fmt.Errorf("location %s is not a dungeon", location)
	}
	if !facing.IsValid() {
		return nil, fmt.Errorf("invalid dungeon facing %d", facing)
	}
	if !lastDirection.IsValid() {
		return nil, fmt.Errorf("invalid dungeon last direction %d", lastDirection)
	}

	return &DungeonState{
		location:                  location,
		cells:                     cells,
		facing:                    facing,
		lastDirection:             lastDirection,
		wanderer:                  nil,
		generateWandererAfterRoom: false,
		roomContinuation:          dungeonRoomContinuation{},
		quicknessSkipNext:         false,
		revision:                  0,
	}, nil
}

// PrepareDungeonState clones immutable source cells without mutating runtime
// location state. Callers install the complete candidate only after validation.
func (g *GameState) PrepareDungeonState(
	location references.Location,
	facing references.DungeonFacing,
	lastDirection references.DungeonLastDirection,
) (*DungeonState, error) {
	if location.GetMapType() != references.DungeonMapType {
		return nil, fmt.Errorf("location %s is not a dungeon", location)
	}
	if g.GameReferences == nil || g.GameReferences.LocationReferences == nil {
		return nil, errors.New("dungeon location references are unavailable")
	}
	reference := g.GameReferences.LocationReferences.GetLocationReference(location)
	cells, ok := reference.CopyDungeonCells()
	if !ok {
		return nil, fmt.Errorf("dungeon source cells are unavailable for %s", location)
	}
	if err := g.applyPersistentDungeonDiscoveries(location, &cells); err != nil {
		return nil, fmt.Errorf("applying persistent discoveries for %s: %w", location, err)
	}
	g.applyPersistentDungeonRooms(location, &cells)

	return newDungeonState(location, cells, facing, lastDirection)
}

func (g *GameState) installDungeonState(state *DungeonState) {
	if state == nil {
		panic("cannot install nil dungeon state")
	}
	if g.MapState.PlayerLocation.Location.GetMapType() != references.DungeonMapType {
		panic("cannot install dungeon state outside a dungeon location")
	}
	if state.location != g.MapState.PlayerLocation.Location {
		panic("dungeon state source does not match player location")
	}
	g.installPreparedDungeonState(state)
	g.synchronizeDungeonRoomEncounterForStablePosition()
}

// installPreparedDungeonState installs semantic authority and every derived
// projection before a transition can access the destination map.
func (g *GameState) installPreparedDungeonState(state *DungeonState) {
	if state == nil {
		panic("cannot install nil dungeon state")
	}
	if state.location.GetMapType() != references.DungeonMapType {
		panic("cannot install non-dungeon state")
	}

	g.activeDungeon = state
	g.MapState.LayeredMaps.ResetAndCreateDungeonMap(
		state.cells,
		g.GameReferences.TileReferences,
		g.MapState.XTilesVisibleOnGameScreen,
		g.MapState.YTilesVisibleOnGameScreen,
	)
}

// Cell returns one authoritative active-visit cell.
func (d *DungeonState) Cell(floor references.FloorNumber, position references.Position) (references.DungeonCell, bool) {
	if d == nil {
		return 0, false
	}

	return d.cells.Cell(floor, position)
}

// Snapshot returns a detached complete semantic view.
func (d *DungeonState) Snapshot() DungeonStateSnapshot {
	if d == nil {
		return DungeonStateSnapshot{}
	}
	var wanderer *DungeonWanderer
	if d.wanderer != nil {
		wandererCopy := *d.wanderer
		wanderer = &wandererCopy
	}

	return DungeonStateSnapshot{
		Location:          d.location,
		Cells:             d.cells,
		ChestAssessments:  d.chestAssessments,
		Facing:            d.facing,
		LastDirection:     d.lastDirection,
		Wanderer:          wanderer,
		QuicknessSkipNext: d.quicknessSkipNext,
		Revision:          d.revision,
	}
}

func (d *DungeonState) occupied(floor references.FloorNumber, position references.Position) bool {
	return d != nil && d.wanderer != nil && d.wanderer.Floor == floor && d.wanderer.Position == position
}

func (d *DungeonState) recordLastDirectionFromFacing() {
	lastDirection := references.DungeonLastDirection(d.facing)
	if d.lastDirection == lastDirection {
		return
	}
	d.lastDirection = lastDirection
	d.revision++
}

func (d *DungeonState) setFacingAndLastDirection(facing references.DungeonFacing) {
	lastDirection := references.DungeonLastDirection(facing)
	if d.facing == facing && d.lastDirection == lastDirection {
		return
	}
	d.facing = facing
	d.lastDirection = lastDirection
	d.revision++
}

func (d *DungeonState) setLastDirection(lastDirection references.DungeonLastDirection) {
	if !lastDirection.IsValid() {
		panic("invalid dungeon last direction")
	}
	if d.lastDirection == lastDirection {
		return
	}
	d.lastDirection = lastDirection
	d.revision++
}

// DungeonStateSnapshot returns the current detached semantic dungeon state.
func (g *GameState) DungeonStateSnapshot() (DungeonStateSnapshot, bool) {
	if g.activeDungeon == nil {
		return DungeonStateSnapshot{}, false
	}

	return g.activeDungeon.Snapshot(), true
}

// TransformDungeonCell applies one authoritative packed-cell mutation and rebuilds its projection floor.
func (g *GameState) TransformDungeonCell(
	floor references.FloorNumber,
	position references.Position,
	transform func(references.DungeonCell) references.DungeonCell,
) (DungeonCellChange, error) {
	dungeon := g.requireActiveDungeon()
	if transform == nil {
		panic("dungeon cell transform is nil")
	}
	before, ok := dungeon.cells.Cell(floor, position)
	if !ok {
		return DungeonCellChange{}, errInvalidDungeonCell
	}
	after := transform(before)
	change := DungeonCellChange{Floor: floor, Position: position, Before: before, After: after}
	if after == before {
		change.Revision = dungeon.revision

		return change, nil
	}
	if !dungeon.cells.SetCell(floor, position, after) {
		return DungeonCellChange{}, errInvalidDungeonCell
	}
	// A ceiling-bit change does not replace or rearm the chest. All other
	// mutations retire its old assessment; successful Jimmy then records safety.
	if before.Raw()&0xf7 != after.Raw()&0xf7 {
		dungeon.chestAssessments[floor][position.Y][position.X] = references.ChestUninspected
	}
	dungeon.revision++
	change.Revision = dungeon.revision
	g.MapState.LayeredMaps.RebuildDungeonFloor(dungeon.cells, floor)
	// Reconcile cell-owned presentation synchronously. A replacement followed by
	// another removal in the same update must not revive the earlier remnant.
	g.SystemCallbacks.Screen.UpdateDisplay()

	return change, nil
}
