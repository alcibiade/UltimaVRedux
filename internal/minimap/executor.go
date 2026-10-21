package minimap

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// WalkingSoundPlayer plays walking sounds based on position.
// Implemented by GameScene to provide automatic surface detection.
type WalkingSoundPlayer interface {
	PlayWalkingSoundAt(pos references.Position)
}

// ScriptExecutor animates a script on a scene, frame by frame
type ScriptExecutor struct {
	scene              *MiniMapScene
	script             *MiniMapScript
	currentStepIndex   int
	elapsedMs          int // Time elapsed for current step
	walkProgress       map[ActorID]*walkState
	walkingSoundPlayer WalkingSoundPlayer // Optional - plays sounds during walks
	awaitingAdvance    bool
	tileCommandStarted bool
	interrupted        bool
}

// walkState tracks the walking progress for an actor
type walkState struct {
	startPos   references.Position
	targetPos  references.Position
	currentPos references.Position
	stepsTaken int
	totalSteps int
}

// NewScriptExecutor creates a new executor for the given scene and script
func NewScriptExecutor(scene *MiniMapScene, script *MiniMapScript) *ScriptExecutor {
	return NewScriptExecutorWithSounds(scene, script, nil)
}

// NewScriptExecutorWithSounds creates a new executor with optional walking sound support.
// If walkingSoundPlayer is provided and a walk step has PlayWalkingSounds=true,
// the player will be called on each step to play appropriate walking sounds.
func NewScriptExecutorWithSounds(
	scene *MiniMapScene,
	script *MiniMapScript,
	player WalkingSoundPlayer,
) *ScriptExecutor {
	if scene == nil {
		panic("ScriptExecutor: scene cannot be nil")
	}

	if script == nil {
		panic("ScriptExecutor: script cannot be nil")
	}
	if err := script.Validate(); err != nil {
		panic(err)
	}

	return &ScriptExecutor{
		scene:              scene,
		script:             script,
		currentStepIndex:   0,
		elapsedMs:          0,
		walkProgress:       make(map[ActorID]*walkState),
		walkingSoundPlayer: player,
		awaitingAdvance:    false, tileCommandStarted: false, interrupted: false,
	}
}

// IsComplete returns true if the script has finished executing
func (e *ScriptExecutor) IsComplete() bool {
	return e.interrupted || e.currentStepIndex >= len(e.script.Steps)
}

// Interrupt completes an interruptible presentation without executing further steps.
func (e *ScriptExecutor) Interrupt() {
	e.interrupted = true
	e.awaitingAdvance = false
}

// IsWaitingForAdvance reports whether the script is paused at an explicit input gate.
func (e *ScriptExecutor) IsWaitingForAdvance() bool {
	return e.awaitingAdvance
}

// Advance releases one explicit input gate. It returns false when the script is
// not currently awaiting input, allowing callers to consume keys only when used.
func (e *ScriptExecutor) Advance() bool {
	if !e.awaitingAdvance {
		return false
	}
	e.awaitingAdvance = false
	e.advanceToNextStep()
	e.Update(0)

	return true
}

// Update advances the script execution by deltaMs milliseconds
func (e *ScriptExecutor) Update(deltaMs int) {
	if e.IsComplete() || e.awaitingAdvance {
		return
	}

	for !e.IsComplete() && !e.awaitingAdvance {
		step := &e.script.Steps[e.currentStepIndex]
		switch step.Type {
		case StepWalk:
			e.executeWalk(step, deltaMs)

			return
		case StepWait:
			e.executeWait(step, deltaMs)

			return
		case StepFace:
			e.executeFace(step)
		case StepSetVisible:
			e.executeSetVisible(step)
		case StepSetSprite:
			e.executeSetSprite(step)
		case StepSetFrame:
			e.scene.SetCurrentFrame(step.Frame)
			e.advanceToNextStep()
		case StepAwaitAdvance:
			e.awaitingAdvance = true
		case StepTileCommand:
			remainingMs, stop := e.executeTileCommand(step.TileCommand, deltaMs)
			deltaMs = remainingMs
			if stop {
				return
			}
		default:
			panic("ScriptExecutor: validated script contains an unknown step")
		}
	}
}

func (e *ScriptExecutor) executeTileCommand(command *TileCommand, deltaMs int) (int, bool) {
	if !e.tileCommandStarted {
		e.scene.applyTileCommand(command)
		e.tileCommandStarted = true
	}
	duration := tileCommandDuration(command)
	if duration <= 0 {
		if command.Type == TileCommandReset {
			e.currentStepIndex = 0
			e.resetStepState()

			return deltaMs, false
		}
		e.scene.finishTileCommand(command)
		e.advanceToNextStep()

		return deltaMs, false
	}
	remainingForStep := duration - e.elapsedMs
	consumedMs := min(deltaMs, remainingForStep)
	e.elapsedMs += consumedMs
	e.scene.updateTileCommand(command, e.elapsedMs)
	if e.elapsedMs < duration {
		return 0, true
	}
	e.scene.finishTileCommand(command)
	e.advanceToNextStep()

	return deltaMs - consumedMs, false
}

func tileCommandDuration(command *TileCommand) int {
	if command == nil {
		return 0
	}
	if command.Type == TileCommandRefresh {
		return command.Cycles * command.CycleMs
	}

	return command.DurationMs
}

// executeWalk handles walking actors toward their target positions
func (e *ScriptExecutor) executeWalk(step *ScriptStep, deltaMs int) {
	e.elapsedMs += deltaMs

	// Initialize walk states if not already done
	for _, actorID := range step.ActorIDs {
		if _, exists := e.walkProgress[actorID]; !exists {
			actor := e.scene.GetActor(actorID)
			if actor == nil {
				continue
			}

			startPos := actor.Position
			targetPos := *step.Target
			totalSteps := calculateTotalSteps(startPos, targetPos)

			e.walkProgress[actorID] = &walkState{
				startPos:   startPos,
				targetPos:  targetPos,
				currentPos: startPos,
				stepsTaken: 0,
				totalSteps: totalSteps,
			}
		}
	}

	// Check if enough time has passed for a step
	if e.elapsedMs >= step.DelayMs {
		e.elapsedMs -= step.DelayMs

		allComplete := true

		for _, actorID := range step.ActorIDs {
			walker := e.walkProgress[actorID]
			if walker == nil {
				continue
			}

			e.stepWalker(step, actorID, walker)

			if walker.stepsTaken < walker.totalSteps {
				allComplete = false
			}
		}

		if allComplete {
			e.advanceToNextStep()
		}
	}
}

// stepWalker takes one step toward the target if the actor has not arrived yet.
func (e *ScriptExecutor) stepWalker(step *ScriptStep, actorID ActorID, walker *walkState) {
	if walker.stepsTaken >= walker.totalSteps {
		return
	}

	walker.currentPos = takeOneStep(walker.currentPos, walker.targetPos)
	walker.stepsTaken++
	e.scene.SetActorPosition(actorID, walker.currentPos)

	// Play walking sound if enabled for this step
	if step.PlayWalkingSounds && e.walkingSoundPlayer != nil {
		e.walkingSoundPlayer.PlayWalkingSoundAt(walker.currentPos)
	}
}

// executeWait handles waiting for a duration
func (e *ScriptExecutor) executeWait(step *ScriptStep, deltaMs int) {
	e.elapsedMs += deltaMs

	if e.elapsedMs >= step.Duration {
		e.advanceToNextStep()
	}
}

// executeFace handles changing actor facing directions
func (e *ScriptExecutor) executeFace(step *ScriptStep) {
	for _, actorID := range step.ActorIDs {
		e.scene.SetActorFacing(actorID, step.Direction)
	}

	e.advanceToNextStep()
}

// executeSetVisible handles showing/hiding actors
func (e *ScriptExecutor) executeSetVisible(step *ScriptStep) {
	for _, actorID := range step.ActorIDs {
		e.scene.SetActorVisible(actorID, step.Visible)
	}

	e.advanceToNextStep()
}

// executeSetSprite handles changing actor sprites
func (e *ScriptExecutor) executeSetSprite(step *ScriptStep) {
	for _, actorID := range step.ActorIDs {
		e.scene.SetActorSprite(actorID, step.Sprite)
	}

	e.advanceToNextStep()
}

// advanceToNextStep moves to the next step and resets state
func (e *ScriptExecutor) advanceToNextStep() {
	e.currentStepIndex++
	e.resetStepState()
}

func (e *ScriptExecutor) resetStepState() {
	e.elapsedMs = 0
	e.walkProgress = make(map[ActorID]*walkState)
	e.tileCommandStarted = false
}

// calculateTotalSteps returns the number of steps needed to reach target
func calculateTotalSteps(from, to references.Position) int {
	dx := abs(int(to.X) - int(from.X))
	dy := abs(int(to.Y) - int(from.Y))

	// Manhattan-ish distance - we move one axis at a time
	return dx + dy
}

// takeOneStep moves one step toward the target position
func takeOneStep(current, target references.Position) references.Position {
	result := current

	// Prioritize vertical movement first (Y axis), then horizontal (X axis)
	// This matches the shrine walking pattern (north first)
	switch {
	case current.Y < target.Y:
		result.Y++
	case current.Y > target.Y:
		result.Y--
	case current.X < target.X:
		result.X++
	case current.X > target.X:
		result.X--
	}

	return result
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}

	return x
}
