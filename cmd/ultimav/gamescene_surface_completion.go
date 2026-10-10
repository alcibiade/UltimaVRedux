package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

type surfaceCommandState uint8

const (
	surfaceCommandCompleted surfaceCommandState = iota
	surfaceCommandRejected
	surfaceCommandCancelled
	surfaceCommandDeferred
	surfaceCommandStillActive
)

type surfaceCommandResult struct {
	State               surfaceCommandState
	OriginMapType       references.GeneralMapType
	ConsumesNormalCycle bool
	MovementOccurred    bool
	MapTransitioned     bool
	Pass                bool
}

func newSurfaceCommandResult(state surfaceCommandState, origin references.GeneralMapType) surfaceCommandResult {
	return surfaceCommandResult{
		State:               state,
		OriginMapType:       origin,
		ConsumesNormalCycle: state != surfaceCommandDeferred && state != surfaceCommandStillActive,
	}
}

func surfaceResultFromSuccess(success bool, origin references.GeneralMapType) surfaceCommandResult {
	if success {
		return newSurfaceCommandResult(surfaceCommandCompleted, origin)
	}

	return newSurfaceCommandResult(surfaceCommandRejected, origin)
}

func (g *GameScene) completeSurfaceCommand(result surfaceCommandResult) {
	if result.MovementOccurred {
		panic("surface movement must use movement completion")
	}
	if result.MapTransitioned {
		return
	}
	switch result.State {
	case surfaceCommandCompleted, surfaceCommandRejected, surfaceCommandCancelled:
		if result.ConsumesNormalCycle {
			if result.Pass {
				g.gameState.CompleteSurfacePassCycle(result.OriginMapType)
			} else {
				g.gameState.CompleteSurfaceCommandCycle(result.OriginMapType)
			}
		}
	case surfaceCommandDeferred, surfaceCommandStillActive:
		if result.ConsumesNormalCycle {
			panic("active surface command cannot consume its normal cycle")
		}
	default:
		panic("unknown surface command completion state")
	}
}

func (g *GameScene) completeSurfaceMovement(result surfaceCommandResult) {
	if result.MapTransitioned {
		return
	}
	if result.State == surfaceCommandDeferred || result.State == surfaceCommandStillActive {
		if result.ConsumesNormalCycle {
			panic("active surface movement cannot consume its normal cycle")
		}

		return
	}
	if !result.ConsumesNormalCycle {
		return
	}
	if result.OriginMapType == references.LargeMapType && !result.MovementOccurred && result.State == surfaceCommandCompleted {
		// A completed heading change is an ordinary command, not a blocked step.
		g.gameState.CompleteSurfaceCommandCycle(result.OriginMapType)

		return
	}

	g.gameState.CompleteSurfaceMovementCycle(result.OriginMapType, result.MovementOccurred)
}

// deferSurfaceCommandCompletion transfers completion to the workflow lane when
// the action started a GameState-owned workflow that cannot accept a callback.
func (g *GameScene) deferSurfaceCommandCompletion(result surfaceCommandResult) {
	result.State = surfaceCommandCompleted
	result.ConsumesNormalCycle = true
	g.gameState.EventQueueManager.Workflow("surface_command_completion").
		Priority(event_queue.PriorityLow).
		Sequential().
		First(func() {}).
		Finally(func() {
			g.completeSurfaceCommand(result)
		}).
		Start()
}
