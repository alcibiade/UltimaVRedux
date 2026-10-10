package main

import (
	"strconv"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func (g *GameScene) executeWorldSetupDebugCommand(request debugging.Request) (debugging.Result, bool) {
	switch request.CommandID {
	case debugging.CommandMovementEnterLocation:
		if result, blocked := g.worldSetupBlockedResult(request); blocked {
			return result, true
		}
		locationID, err := strconv.Atoi(mustDebugStringArgument(request, "location"))
		if err != nil {
			panic("validated world setup location is not an integer")
		}
		operation, err := g.gameState.DebugEnterSmallLocation(references.Location(locationID))

		return worldSetupOperationResult(request, "Named location entered.", g.gameState, operation, err), true
	case debugging.CommandMovementFloorSet:
		if result, blocked := g.worldSetupBlockedResult(request); blocked {
			return result, true
		}
		operation, err := g.gameState.DebugSetFloor(
			references.FloorNumber(mustDebugIntegerArgument(request, "floor")),
		)

		return worldSetupOperationResult(request, "Exact floor selected.", g.gameState, operation, err), true
	case debugging.CommandMovementLandmarkTravel:
		if result, blocked := g.worldSetupBlockedResult(request); blocked {
			return result, true
		}
		operation, err := g.gameState.DebugTravelToLandmark(mustDebugStringArgument(request, "landmark"))

		return worldSetupOperationResult(request, "Landmark setup loaded.", g.gameState, operation, err), true
	default:
		return debugging.Result{}, false
	}
}

func (g *GameScene) worldSetupBlockedResult(request debugging.Request) (debugging.Result, bool) {
	code, message, blocked := g.worldSetupBlock()
	if blocked {
		return debugging.FailureResult(request, code, message), true
	}

	return debugging.Result{}, false
}

func (g *GameScene) worldSetupBlock() (string, string, bool) {
	if g.dialogStack.HasOpenDialog() {
		return "dialog_active", "world setup is unavailable while a dialog is open", true
	}
	if g.gameState.EventQueueManager.HasActiveWorkflows() {
		return "gameplay_workflow_active", "world setup is unavailable during a gameplay workflow", true
	}
	if g.miniMapScene != nil || g.gameState.ShrineScene != nil {
		return "scripted_scene_active", "world setup is unavailable during a shrine or Codex scene", true
	}

	return "", "", false
}

func worldSetupOperationResult(
	request debugging.Request,
	summary string,
	state *game_state.GameState,
	operation game_state.WorldSetupResult,
	err error,
) debugging.Result {
	if err != nil {
		return debugging.FailureResult(request, "invalid_world_setup", err.Error())
	}
	result := debugSuccessResultWithMode(
		request,
		debugging.ExecutionSynchronized,
		summary,
		newWorldPositionResult(state),
	)
	appendWorldSetupChanges(&result.Changes, operation)

	return result
}
