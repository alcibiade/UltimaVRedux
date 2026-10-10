package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// DungeonKlimbDirection identifies vertical dungeon travel.
type DungeonKlimbDirection byte

const (
	// DungeonKlimbUp ascends toward the next lower internal floor number.
	DungeonKlimbUp DungeonKlimbDirection = iota
	// DungeonKlimbDown descends toward the next higher internal floor number.
	DungeonKlimbDown
)

// DungeonKlimbOptions reports routes available from the current packed cell.
type DungeonKlimbOptions struct {
	Up              bool
	Down            bool
	GrappleRequired bool
}

type dungeonKlimbPlan struct {
	direction         DungeonKlimbDirection
	targetFloor       references.FloorNumber
	blocked           bool
	exit              bool
	triggersHiddenPit bool
	completion        DungeonCommandCompletion
}

// DungeonKlimbOptionsAtCurrentCell examines the authoritative current cell.
func (g *GameState) DungeonKlimbOptionsAtCurrentCell() DungeonKlimbOptions {
	dungeon := g.requireActiveDungeon()
	cell, ok := dungeon.Cell(g.MapState.PlayerLocation.Floor, g.MapState.PlayerLocation.Position)
	if !ok {
		panic("current dungeon position is invalid")
	}

	options := DungeonKlimbOptions{}
	switch cell.Kind() {
	case references.DungeonLadderUp:
		options.Up = true
	case references.DungeonLadderDown, references.DungeonPitOrTrap:
		options.Down = true
	case references.DungeonLadderBoth:
		options.Up = true
		options.Down = true
	case references.DungeonPassage,
		references.DungeonClosedChest,
		references.DungeonFountain,
		references.DungeonOpenChest,
		references.DungeonEnergyField,
		references.DungeonReserved,
		references.DungeonClearedRoom,
		references.DungeonWall,
		references.DungeonSpecialWall,
		references.DungeonHiddenDoor,
		references.DungeonRevealedDoor,
		references.DungeonLiveRoom:
	}

	if cell.HasBit08() && !options.Up {
		if g.PartyState.Inventory.QuestItems.HasSome(references.Grapple) {
			options.Up = true
		} else {
			options.GrappleRequired = true
		}
	}

	return options
}

// CompleteDungeonKlimbPass resolves Pass at a dual-route prompt as a full command.
func (g *GameState) CompleteDungeonKlimbPass(token DungeonCommandCycle) {
	g.claimDungeonCommandAction(DungeonFullEndPlayer)
	g.SystemCallbacks.Message.AddRowStr("Pass")
	g.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
}

// ActionKlimbDungeonMap starts one input-blocking vertical transition workflow.
func (g *GameState) ActionKlimbDungeonMap(token DungeonCommandCycle, direction DungeonKlimbDirection) {
	if g.EventQueueManager == nil {
		panic("GameState.ActionKlimbDungeonMap: EventQueueManager is nil")
	}
	if g.SystemCallbacks == nil {
		panic("GameState.ActionKlimbDungeonMap: SystemCallbacks is nil")
	}
	if g.EventQueueManager.HasWorkflow("dungeon_klimb") {
		panic("dungeon Klimb workflow is already active")
	}

	plan := g.planDungeonKlimb(direction)
	g.claimDungeonCommandAction(plan.completion)
	var pitCycle *dungeonCommandCycleState
	if plan.triggersHiddenPit {
		pitCycle = g.beginDungeonCommandCompletion(token, plan.completion)
	}
	tokenCompleted := false
	message := "Down!"
	if direction == DungeonKlimbUp {
		message = "Up!"
	}

	workflow := g.EventQueueManager.Workflow("dungeon_klimb").
		Priority(event_queue.PriorityCritical).
		Sequential().BlockInput().
		First(func() {
			g.SystemCallbacks.Message.AddRowStr(message)
		})
	if !plan.blocked {
		workflow.Animate(KlimbFadeOutMs, func(progress float32) {
			g.SystemCallbacks.Visual.SetMapFadeLevel(float64(progress))
		})
	}
	workflow.Then(func() {
		if plan.exit {
			g.CompleteDungeonCommandCycle(token, plan.completion)
			tokenCompleted = true
		}
		g.executeDungeonKlimbPlan(plan)
		if plan.blocked {
			g.SystemCallbacks.Message.AddRowStr("Failed!")
		} else if plan.exit {
			if direction == DungeonKlimbUp {
				g.SystemCallbacks.Message.AddRowStr("Exit to Britannia!")
			} else {
				g.SystemCallbacks.Message.AddRowStr("Exit to Underworld!")
			}
		}
	})
	if !plan.blocked {
		workflow.Animate(KlimbFadeInMs, func(progress float32) {
			g.SystemCallbacks.Visual.SetMapFadeLevel(1.0 - float64(progress))
		})
	}
	if plan.triggersHiddenPit {
		workflow.Then(func() {
			g.processDungeonEndPlayerBeforeCell(pitCycle.dungeonActorsMove)
			g.requireActiveDungeon().setLastDirection(references.DungeonArrivedFromAbove)
		}).ThenSequence(g.thenDungeonHiddenPitFalls())
	}
	workflow.Finally(func() {
		g.SystemCallbacks.Visual.SetMapFadeLevel(0.0)
		if plan.triggersHiddenPit {
			if !g.queueCurrentDungeonRoomEncounter(true, false, true, dungeonRoomGenerateWanderer) {
				g.processDungeonEndPlayerAfterCell()
			}
			g.finishDungeonCommandCycle(token)
		} else if !tokenCompleted {
			g.CompleteDungeonCommandCycle(token, plan.completion)
		}
	}).Start()
}

func (g *GameState) planDungeonKlimb(direction DungeonKlimbDirection) dungeonKlimbPlan {
	options := g.DungeonKlimbOptionsAtCurrentCell()
	if direction == DungeonKlimbUp && !options.Up || direction == DungeonKlimbDown && !options.Down {
		panic("selected dungeon Klimb route is unavailable")
	}
	if direction != DungeonKlimbUp && direction != DungeonKlimbDown {
		panic("invalid dungeon Klimb direction")
	}

	currentFloor := g.MapState.PlayerLocation.Floor
	plan := dungeonKlimbPlan{
		direction:         direction,
		targetFloor:       currentFloor,
		blocked:           false,
		exit:              false,
		triggersHiddenPit: false,
		completion:        DungeonFullEndPlayer,
	}
	if direction == DungeonKlimbUp {
		if currentFloor == 0 {
			plan.exit = true
			plan.completion = DungeonCommandOnly

			return plan
		}
		plan.targetFloor--
	} else {
		if currentFloor == references.DungeonFloorCount-1 {
			plan.exit = true
			plan.completion = DungeonCommandOnly

			return plan
		}
		plan.targetFloor++
	}

	cell, ok := g.requireActiveDungeon().Cell(plan.targetFloor, g.MapState.PlayerLocation.Position)
	if !ok {
		panic("dungeon Klimb target is invalid")
	}
	plan.blocked = dungeonKlimbTargetBlocked(cell.Kind())
	plan.triggersHiddenPit = !plan.blocked && isDungeonHiddenPit(cell)

	return plan
}

func dungeonKlimbTargetBlocked(kind references.DungeonCellKind) bool {
	return kind >= references.DungeonWall && kind <= references.DungeonRevealedDoor
}

func (g *GameState) executeDungeonKlimbPlan(plan dungeonKlimbPlan) {
	if plan.blocked {
		return
	}
	if plan.exit {
		g.exitDungeonToWorld(plan.direction)

		return
	}

	lastDirection := references.DungeonArrivedFromAbove
	if plan.direction == DungeonKlimbUp {
		lastDirection = references.DungeonArrivedFromBelow
	}
	g.transitionDungeonFloor(plan.targetFloor, lastDirection)
	g.generateDungeonWanderer()
}

func (g *GameState) transitionDungeonFloor(
	targetFloor references.FloorNumber,
	lastDirection references.DungeonLastDirection,
) {
	dungeon := g.requireActiveDungeon()
	g.TransitionToLocation(
		g.MapState.PlayerLocation.Location,
		targetFloor,
		g.MapState.PlayerLocation.Position,
		TransitionFloorOnly,
		false,
	)
	dungeon.setLastDirection(lastDirection)
}

func (g *GameState) exitDungeonToWorld(direction DungeonKlimbDirection) {
	dungeon := g.MapState.PlayerLocation.Location
	position, exists := g.GetDungeonEntranceCoordinates()[dungeon]
	if !exists {
		panic("active dungeon entrance coordinate is unavailable")
	}
	floor := references.FloorNumber(references.OVERWORLD)
	if direction == DungeonKlimbDown {
		floor = references.FloorNumber(references.UNDERWORLD)
	}
	g.TransitionToLocation(
		references.BritanniaUnderworld,
		floor,
		position,
		TransitionFullLocation,
		false,
	)
}
