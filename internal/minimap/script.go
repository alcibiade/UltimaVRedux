package minimap

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

var errInvalidScript = errors.New("invalid mini-map script")

// FrameID identifies presentation content owned by the scene rendering the script.
// Tile scenes may ignore frames; illustrated scenes resolve them to their own assets.
type FrameID string

// StepType defines the type of action in a script step
type StepType int

const (
	// StepWalk moves actors to a target position.
	StepWalk StepType = iota
	// StepWait delays script progression.
	StepWait
	// StepFace changes actor facing.
	StepFace
	// StepSetVisible shows or hides actors.
	StepSetVisible
	// StepSetSprite changes actor sprites.
	StepSetSprite
	// StepSetFrame selects scene-owned presentation content.
	StepSetFrame
	// StepAwaitAdvance waits for a fresh user advance signal.
	StepAwaitAdvance
	// StepTileCommand mutates or animates a script-owned tile scene.
	StepTileCommand
)

// TileCommandType identifies a concrete operation on a script-owned tile scene.
type TileCommandType uint8

const (
	// TileCommandSetMap selects source terrain and begins its reveal.
	TileCommandSetMap TileCommandType = iota
	// TileCommandClearActors removes every scene actor.
	TileCommandClearActors
	// TileCommandAddActor creates or replaces one actor.
	TileCommandAddActor
	// TileCommandRemoveActor removes one actor.
	TileCommandRemoveActor
	// TileCommandMoveActor moves one actor one cardinal tile.
	TileCommandMoveActor
	// TileCommandRefresh advances scene animation and timing.
	TileCommandRefresh
	// TileCommandSetTile mutates one terrain tile.
	TileCommandSetTile
	// TileCommandRaiseGate animates a rising moongate.
	TileCommandRaiseGate
	// TileCommandDropGate animates a descending moongate.
	TileCommandDropGate
	// TileCommandFizzActor reveals an actor progressively.
	TileCommandFizzActor
	// TileCommandFadeActor removes an actor progressively.
	TileCommandFadeActor
	// TileCommandNukActor applies the Return-to-the-View strike effect.
	TileCommandNukActor
	// TileCommandReset loops the script to its first instruction.
	TileCommandReset
)

// TileCommand is a typed presentation operation decoded from original data or
// authored through NewMiniMapScript. Unused fields remain at their zero values.
type TileCommand struct {
	Type       TileCommandType
	ActorID    ActorID
	Sprite     indexes.SpriteIndex
	Position   references.Position
	Direction  references.Direction
	MapID      FrameID
	Cycles     int
	CycleMs    int
	DurationMs int
}

// ScriptStep represents a single action in a mini-map script
type ScriptStep struct {
	Type              StepType
	ActorIDs          []ActorID            // Actors affected by this step
	Target            *references.Position // Target position for walk steps
	Direction         references.Direction // Direction for face steps
	DelayMs           int                  // Delay between walk steps in milliseconds
	Duration          int                  // Duration for wait steps in milliseconds
	Visible           bool                 // Visibility for show/hide steps
	Sprite            indexes.SpriteIndex  // Sprite for set sprite steps
	PlayWalkingSounds bool                 // If true, play walking sounds on each step (walk steps only)
	Frame             FrameID              // Scene-owned frame for presentation steps
	TileCommand       *TileCommand         // Typed tile-scene operation
}

// MiniMapScript represents a sequence of scripted actions for a mini-map scene
type MiniMapScript struct {
	Name  string       // Script identifier for debugging
	Steps []ScriptStep // Ordered list of steps to execute
}

// Validate rejects malformed scripts before they can leave an executor stalled
// or panic partway through a presentation.
func (s *MiniMapScript) Validate() error {
	if s == nil {
		return fmt.Errorf("%w: script is nil", errInvalidScript)
	}
	for index := range s.Steps {
		if err := validateScriptStep(&s.Steps[index], index); err != nil {
			return err
		}
	}

	return nil
}

func validateScriptStep(step *ScriptStep, index int) error {
	switch step.Type {
	case StepWalk:
		return validateWalkStep(step, index)
	case StepWait:
		return validateWaitStep(step, index)
	case StepFace, StepSetVisible, StepSetSprite:
		return validateActorStep(step, index)
	case StepSetFrame:
		return validateFrameStep(step, index)
	case StepAwaitAdvance:
		return nil
	case StepTileCommand:
		return validateTileCommand(step.TileCommand, index)
	default:
		return fmt.Errorf("%w: step %d has unknown type %d", errInvalidScript, index, step.Type)
	}
}

//nolint:cyclop // Validation mirrors the finite typed command set.
func validateTileCommand(command *TileCommand, index int) error {
	if command == nil {
		return fmt.Errorf("%w: tile command step %d has no command", errInvalidScript, index)
	}
	switch command.Type {
	case TileCommandSetMap:
		if command.MapID == "" {
			return fmt.Errorf("%w: tile command step %d has no map ID", errInvalidScript, index)
		}
	case TileCommandAddActor, TileCommandRemoveActor, TileCommandMoveActor,
		TileCommandFizzActor, TileCommandFadeActor, TileCommandNukActor:
		if command.ActorID == "" {
			return fmt.Errorf("%w: tile command step %d has no actor ID", errInvalidScript, index)
		}
	case TileCommandRefresh:
		if command.Cycles < 0 || command.CycleMs <= 0 {
			return fmt.Errorf("%w: tile command step %d has invalid refresh timing", errInvalidScript, index)
		}
	case TileCommandRaiseGate, TileCommandDropGate:
		if command.DurationMs <= 0 {
			return fmt.Errorf("%w: tile command step %d has invalid gate duration", errInvalidScript, index)
		}
	case TileCommandClearActors, TileCommandSetTile, TileCommandReset:
	default:
		return fmt.Errorf("%w: tile command step %d has unknown command %d", errInvalidScript, index, command.Type)
	}

	return nil
}

func validateWalkStep(step *ScriptStep, index int) error {
	if len(step.ActorIDs) == 0 || step.Target == nil || step.DelayMs <= 0 {
		return fmt.Errorf("%w: walk step %d requires actors, target, and positive delay", errInvalidScript, index)
	}

	return nil
}

func validateWaitStep(step *ScriptStep, index int) error {
	if step.Duration < 0 {
		return fmt.Errorf("%w: wait step %d has negative duration", errInvalidScript, index)
	}

	return nil
}

func validateActorStep(step *ScriptStep, index int) error {
	if len(step.ActorIDs) == 0 {
		return fmt.Errorf("%w: actor step %d requires at least one actor", errInvalidScript, index)
	}

	return nil
}

func validateFrameStep(step *ScriptStep, index int) error {
	if step.Frame == "" {
		return fmt.Errorf("%w: frame step %d has no frame ID", errInvalidScript, index)
	}

	return nil
}
