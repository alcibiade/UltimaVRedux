package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// AutoOpenDoorState tracks the state of an auto-open door animation sequence
type AutoOpenDoorState struct {
	isActive       bool
	direction      references.Direction
	startElapsedMs int64 // Start time in milliseconds
	phase          AutoOpenPhase
	firstTurnDone  bool
	secondTurnDone bool
	movePlayerDone bool
	targetPosition *references.Position
}

// AutoOpenPhase represents the current phase of the auto-open animation
type AutoOpenPhase int

const (
	AutoOpenPhaseNone AutoOpenPhase = iota
	AutoOpenPhaseOpenDoor
	AutoOpenPhaseFirstNPCMove
	AutoOpenPhaseSecondNPCMove
	AutoOpenPhaseMovePlayer
	AutoOpenPhaseComplete
)

const autoOpenPauseDurationMs = int64(150) // Pause duration in milliseconds

// StartAutoOpenSequence begins an auto-open door animation sequence
func (a *AutoOpenDoorState) StartAutoOpenSequence(
	direction references.Direction,
	targetPos *references.Position,
	currentElapsedMs int64,
) {
	a.isActive = true
	a.direction = direction
	a.startElapsedMs = currentElapsedMs
	a.phase = AutoOpenPhaseOpenDoor
	a.firstTurnDone = false
	a.secondTurnDone = false
	a.movePlayerDone = false
	a.targetPosition = targetPos
}

// AutoOpenResult represents the result of an auto-open update
type AutoOpenResult struct {
	ShouldShowMessage bool
	Message           string
	ShouldMovePlayer  bool
	MoveFailed        bool
}

// Update processes the auto-open animation state and returns status information
func (a *AutoOpenDoorState) Update(gameScene *GameScene, currentElapsedMs int64) *AutoOpenResult {
	if !a.isActive {
		return nil
	}

	elapsedSinceStart := currentElapsedMs - a.startElapsedMs
	result := &AutoOpenResult{ShouldShowMessage: false, Message: "", ShouldMovePlayer: false, MoveFailed: false}

	switch a.phase {
	case AutoOpenPhaseNone:
		// Inactive; nothing to advance.

	case AutoOpenPhaseOpenDoor:
		// Open the door - ActionOpenSmallMap handles time advancement and "Opened!" message
		if gameScene.gameState.ActionOpenSmallMap(a.direction) {
			a.phase = AutoOpenPhaseFirstNPCMove
			a.startElapsedMs = currentElapsedMs
		} else {
			// Failed to open, abort
			result.MoveFailed = true
			a.Reset()
		}

	case AutoOpenPhaseFirstNPCMove:
		if !a.firstTurnDone {
			// Process first NPC movement only (no time advancement - ActionOpenSmallMap already did that)
			gameScene.gameState.SmallMapProcessNPCs()
			a.firstTurnDone = true
			a.startElapsedMs = currentElapsedMs
		} else if elapsedSinceStart >= autoOpenPauseDurationMs {
			// After pause, move to second NPC turn
			a.phase = AutoOpenPhaseSecondNPCMove
			a.startElapsedMs = currentElapsedMs
		}

	case AutoOpenPhaseSecondNPCMove:
		if !a.secondTurnDone {
			// Process second NPC movement only (no additional time advancement)
			gameScene.gameState.SmallMapProcessNPCs()
			a.secondTurnDone = true
			a.startElapsedMs = currentElapsedMs
		} else if elapsedSinceStart >= autoOpenPauseDurationMs {
			// After pause, move player through door
			a.phase = AutoOpenPhaseMovePlayer
		}

	case AutoOpenPhaseMovePlayer:
		if !a.movePlayerDone {
			// Signal that player should be moved
			result.ShouldMovePlayer = true
			result.ShouldShowMessage = true
			result.Message = "" // Blank line for cleaner output
			a.movePlayerDone = true
		}
		a.phase = AutoOpenPhaseComplete

	case AutoOpenPhaseComplete:
		a.Reset()
	}

	return result
}

// IsActive returns whether an auto-open sequence is currently active
func (a *AutoOpenDoorState) IsActive() bool {
	return a.isActive
}

// Reset clears the auto-open state
func (a *AutoOpenDoorState) Reset() {
	a.isActive = false
	a.phase = AutoOpenPhaseNone
	a.firstTurnDone = false
	a.secondTurnDone = false
	a.movePlayerDone = false
	a.targetPosition = nil
	a.startElapsedMs = 0
}
