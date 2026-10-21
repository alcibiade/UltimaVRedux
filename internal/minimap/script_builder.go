package minimap

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// ScriptBuilder provides a fluent API for creating mini-map scripts
type ScriptBuilder struct {
	script *Script
}

// NewMiniMapScript creates a new script builder with the given name
func NewMiniMapScript(name string) *ScriptBuilder {
	return &ScriptBuilder{
		script: &Script{
			Name:  name,
			Steps: make([]ScriptStep, 0),
		},
	}
}

// newStep returns an otherwise empty step of the given type; callers set only the fields that type uses.
func newStep(stepType StepType) ScriptStep {
	return ScriptStep{
		Type:              stepType,
		ActorIDs:          nil,
		Target:            nil,
		Direction:         references.NoneDirection,
		DelayMs:           0,
		Duration:          0,
		Visible:           false,
		Sprite:            indexes.NoSprites,
		PlayWalkingSounds: false,
		Frame:             "",
		TileCommand:       nil,
	}
}

// WalkTo adds a walk step for a single actor to reach the target position
func (b *ScriptBuilder) WalkTo(actor ActorID, target references.Position, delayMs int) *ScriptBuilder {
	return b.WalkActors([]ActorID{actor}, target, delayMs)
}

// WalkActors adds a walk step for multiple actors to reach the target position
func (b *ScriptBuilder) WalkActors(
	actors []ActorID,
	target references.Position,
	delayMs int,
) *ScriptBuilder {
	step := newStep(StepWalk)
	step.ActorIDs = actors
	step.Target = &target
	step.DelayMs = delayMs
	b.script.Steps = append(b.script.Steps, step)

	return b
}

// SetVisible adds a visibility step for a single actor
func (b *ScriptBuilder) SetVisible(actor ActorID, visible bool) *ScriptBuilder {
	step := newStep(StepSetVisible)
	step.ActorIDs = []ActorID{actor}
	step.Visible = visible
	b.script.Steps = append(b.script.Steps, step)

	return b
}

// SetSprite adds a sprite change step for a single actor
func (b *ScriptBuilder) SetSprite(actor ActorID, sprite indexes.SpriteIndex) *ScriptBuilder {
	step := newStep(StepSetSprite)
	step.ActorIDs = []ActorID{actor}
	step.Sprite = sprite
	b.script.Steps = append(b.script.Steps, step)

	return b
}

// SetFrame selects scene-owned presentation content. The script controls when
// content changes while the rendering scene controls how that frame is drawn.
func (b *ScriptBuilder) SetFrame(frame FrameID) *ScriptBuilder {
	step := newStep(StepSetFrame)
	step.Frame = frame
	b.script.Steps = append(b.script.Steps, step)

	return b
}

// AwaitAdvance pauses script progression until ScriptExecutor.Advance is called.
func (b *ScriptBuilder) AwaitAdvance() *ScriptBuilder {
	b.script.Steps = append(b.script.Steps, newStep(StepAwaitAdvance))

	return b
}

// TileCommand appends a typed operation for a script-owned tile scene.
func (b *ScriptBuilder) TileCommand(command TileCommand) *ScriptBuilder {
	step := newStep(StepTileCommand)
	step.TileCommand = &command
	b.script.Steps = append(b.script.Steps, step)

	return b
}

// WithWalkingSounds enables walking sound effects for the last walk step.
// The executor will play sounds via WalkingSoundPlayer on each step if configured.
func (b *ScriptBuilder) WithWalkingSounds() *ScriptBuilder {
	if len(b.script.Steps) > 0 {
		lastIdx := len(b.script.Steps) - 1
		b.script.Steps[lastIdx].PlayWalkingSounds = true
	}

	return b
}

// Build returns the completed script
func (b *ScriptBuilder) Build() *Script {
	return b.script
}
