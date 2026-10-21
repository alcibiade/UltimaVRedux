package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	dungeonWandererContactWorkflowName = "dungeon_wanderer_contact"
	dungeonWandererContactDelayMs      = int64(2 * references.OriginalTickMs)
)

// DungeonWandererEncounterKind distinguishes the original direct-Attack and
// automatic-contact return policies.
type DungeonWandererEncounterKind byte

const (
	// DungeonWandererEncounterDirect is initiated by the dungeon Attack command.
	DungeonWandererEncounterDirect DungeonWandererEncounterKind = iota
	// DungeonWandererEncounterContact interrupts full end-player processing.
	DungeonWandererEncounterContact
)

// DungeonWandererEncounterOrigin retains the dungeon continuation owned by a
// generated tactical encounter.
type DungeonWandererEncounterOrigin struct {
	Kind     DungeonWandererEncounterKind
	Location references.Location
	Floor    references.FloorNumber
	Position references.Position

	commandToken DungeonCommandCycle
}

func (g *GameState) startDirectDungeonWandererCombat(monster references.MonsterType) error {
	combat, err := g.PrepareDungeonWandererCombatState(monster)
	if err != nil {
		return err
	}
	player := g.MapState.PlayerLocation
	combat.DungeonWandererOrigin = &DungeonWandererEncounterOrigin{
		Kind: DungeonWandererEncounterDirect, Location: player.Location,
		Floor: player.Floor, Position: player.Position,
	}
	g.commitCombatTransition(combat, references.DungeonCombatTerritory)

	return nil
}

func (g *GameState) startContactDungeonWandererCombat(
	token DungeonCommandCycle,
	movement DungeonWandererMovementResult,
) bool {
	if g.EventQueueManager == nil {
		panic("dungeon wanderer contact requires an event queue manager")
	}
	if g.SystemCallbacks == nil {
		panic("dungeon wanderer contact requires system callbacks")
	}
	if g.EventQueueManager.HasWorkflow(dungeonWandererContactWorkflowName) {
		panic("dungeon wanderer contact workflow is already active")
	}
	dungeon := g.requireActiveDungeon()
	wanderer := dungeon.wanderer
	if !movement.Contact || wanderer == nil || wanderer.Monster != movement.Monster ||
		wanderer.Floor != g.MapState.PlayerLocation.Floor || wanderer.Position != movement.AttackOrigin {
		panic("dungeon wanderer contact no longer matches authoritative state")
	}
	attackFacing, ok := dungeonWandererAttackFacing(g.MapState.PlayerLocation.Position, movement.AttackOrigin)
	if !ok {
		panic("dungeon wanderer contact origin is not adjacent to the party")
	}
	combat, err := g.prepareDungeonWandererCombatStateForFacing(
		movement.Monster,
		attackFacing,
		g.AsRandomProvider(),
	)
	if err != nil {
		logging.Errorf(logging.CategoryGameState, "Unable to prepare dungeon wanderer contact: %v", err)
		g.SystemCallbacks.UI.ShowActionFeedback("Combat could not begin.", true)

		return false
	}

	previousFacing := dungeon.facing
	dungeon.setFacingAndLastDirection(attackFacing)
	player := g.MapState.PlayerLocation
	combat.DungeonWandererOrigin = &DungeonWandererEncounterOrigin{
		Kind: DungeonWandererEncounterContact, Location: player.Location,
		Floor: player.Floor, Position: player.Position, commandToken: token,
	}
	message := "Attacked!"
	if attackFacing != previousFacing {
		message = fmt.Sprintf("Attacked from the %s!", dungeonWandererFacingName(attackFacing))
	}

	g.EventQueueManager.Workflow(dungeonWandererContactWorkflowName).
		Priority(event_queue.PriorityHigh).
		Sequential().BlockInput().
		First(func() {
			g.SystemCallbacks.Message.AddRowStr(message)
			g.SystemCallbacks.Screen.UpdateDisplay()
		}).
		Wait(dungeonWandererContactDelayMs).
		Then(func() {
			g.commitCombatTransition(combat, references.DungeonCombatTerritory)
		}).
		Finally(func() {}).
		Start()

	return true
}

func dungeonWandererAttackFacing(
	party references.Position,
	origin references.Position,
) (references.DungeonFacing, bool) {
	for facing := references.DungeonFacingNorth; facing <= references.DungeonFacingWest; facing++ {
		target := facing.Direction().GetNewPositionInDirection(&party).GetWrapped(
			references.XDungeonMapTiles,
			references.YDungeonMapTiles,
		)
		if *target == origin {
			return facing, true
		}
	}

	return references.DungeonFacingNorth, false
}

func dungeonWandererFacingName(facing references.DungeonFacing) string {
	switch facing {
	case references.DungeonFacingNorth:
		return "north"
	case references.DungeonFacingEast:
		return "east"
	case references.DungeonFacingSouth:
		return "south"
	case references.DungeonFacingWest:
		return "west"
	default:
		panic("invalid dungeon wanderer facing")
	}
}

func (g *GameState) resumeDungeonWandererEncounter(
	origin *DungeonWandererEncounterOrigin,
	route CombatExitRoute,
) {
	if origin == nil {
		return
	}
	if g.MapState.PlayerLocation.Location != origin.Location ||
		g.MapState.PlayerLocation.Floor != origin.Floor ||
		g.MapState.PlayerLocation.Position != origin.Position {
		panic("dungeon wanderer combat returned to an unexpected origin")
	}
	if origin.Kind != DungeonWandererEncounterDirect && origin.Kind != DungeonWandererEncounterContact {
		panic("invalid dungeon wanderer encounter origin")
	}

	contactTokenFinished := false
	if origin.Kind == DungeonWandererEncounterContact &&
		dungeonWandererExitLeavesDungeon(route, g.MapState.PlayerLocation.Floor) {
		// Boundary exits must release the retained command before the full map
		// transition rejects active dungeon command ownership.
		g.finishDungeonCommandCycle(origin.commandToken)
		contactTokenFinished = true
	}

	g.applyDungeonWandererExitRoute(origin.Kind, route)
	if g.MapState.PlayerLocation.Location.GetMapType() == references.DungeonMapType {
		g.generateDungeonWanderer()
	}
	if origin.Kind == DungeonWandererEncounterDirect {
		return
	}
	if origin.Kind != DungeonWandererEncounterContact || origin.commandToken.id == 0 {
		panic("invalid dungeon wanderer encounter continuation")
	}
	if g.MapState.PlayerLocation.Location.GetMapType() == references.DungeonMapType {
		if g.processDungeonEndPlayerAfterWanderer(origin.commandToken) {
			return
		}
	}
	if !contactTokenFinished {
		g.finishDungeonCommandCycle(origin.commandToken)
	}
}

func dungeonWandererExitLeavesDungeon(route CombatExitRoute, floor references.FloorNumber) bool {
	return route == CombatExitUp && floor == 0 ||
		route == CombatExitDown && floor == references.DungeonFloorCount-1
}

func (g *GameState) applyDungeonWandererExitRoute(
	kind DungeonWandererEncounterKind,
	route CombatExitRoute,
) {
	switch route {
	case CombatExitNorth, CombatExitEast, CombatExitSouth, CombatExitWest:
		if kind != DungeonWandererEncounterContact {
			return
		}
		position, moved := applyDungeonCardinalExit(g.MapState.PlayerLocation.Position, route)
		if !moved {
			panic("cardinal dungeon wanderer exit did not move")
		}
		g.MapState.PlayerLocation.Position = position
		g.requireActiveDungeon().setFacingAndLastDirection(combatExitRouteFacing(route))
	case CombatExitUp, CombatExitDown:
		g.applyDungeonVerticalExit(route)
		if kind == DungeonWandererEncounterDirect && g.activeDungeon != nil {
			// attack_dng() handles only the vertical level change and never enters
			// end_player(), even when the destination is an A*/F* room.
			g.activeDungeon.roomContinuation = dungeonRoomContinuation{}
		}
	case CombatExitNone:
	case CombatExitDoomMirror:
		fallthrough
	default:
		panic("invalid dungeon wanderer exit route")
	}
}
