package minimap

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// MiniMapScriptBuilder provides a fluent API for creating mini-map scripts
type MiniMapScriptBuilder struct {
	script *MiniMapScript
}

// NewMiniMapScript creates a new script builder with the given name
func NewMiniMapScript(name string) *MiniMapScriptBuilder {
	return &MiniMapScriptBuilder{
		script: &MiniMapScript{
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
func (b *MiniMapScriptBuilder) WalkTo(actor ActorID, target references.Position, delayMs int) *MiniMapScriptBuilder {
	return b.WalkActors([]ActorID{actor}, target, delayMs)
}

// WalkActors adds a walk step for multiple actors to reach the target position
func (b *MiniMapScriptBuilder) WalkActors(
	actors []ActorID,
	target references.Position,
	delayMs int,
) *MiniMapScriptBuilder {
	step := newStep(StepWalk)
	step.ActorIDs = actors
	step.Target = &target
	step.DelayMs = delayMs
	b.script.Steps = append(b.script.Steps, step)

	return b
}

// SetVisible adds a visibility step for a single actor
func (b *MiniMapScriptBuilder) SetVisible(actor ActorID, visible bool) *MiniMapScriptBuilder {
	step := newStep(StepSetVisible)
	step.ActorIDs = []ActorID{actor}
	step.Visible = visible
	b.script.Steps = append(b.script.Steps, step)

	return b
}

// SetSprite adds a sprite change step for a single actor
func (b *MiniMapScriptBuilder) SetSprite(actor ActorID, sprite indexes.SpriteIndex) *MiniMapScriptBuilder {
	step := newStep(StepSetSprite)
	step.ActorIDs = []ActorID{actor}
	step.Sprite = sprite
	b.script.Steps = append(b.script.Steps, step)

	return b
}

// SetFrame selects scene-owned presentation content. The script controls when
// content changes while the rendering scene controls how that frame is drawn.
func (b *MiniMapScriptBuilder) SetFrame(frame FrameID) *MiniMapScriptBuilder {
	step := newStep(StepSetFrame)
	step.Frame = frame
	b.script.Steps = append(b.script.Steps, step)

	return b
}

// AwaitAdvance pauses script progression until ScriptExecutor.Advance is called.
func (b *MiniMapScriptBuilder) AwaitAdvance() *MiniMapScriptBuilder {
	b.script.Steps = append(b.script.Steps, newStep(StepAwaitAdvance))

	return b
}

// TileCommand appends a typed operation for a script-owned tile scene.
func (b *MiniMapScriptBuilder) TileCommand(command TileCommand) *MiniMapScriptBuilder {
	step := newStep(StepTileCommand)
	step.TileCommand = &command
	b.script.Steps = append(b.script.Steps, step)

	return b
}

// WithWalkingSounds enables walking sound effects for the last walk step.
// The executor will play sounds via WalkingSoundPlayer on each step if configured.
func (b *MiniMapScriptBuilder) WithWalkingSounds() *MiniMapScriptBuilder {
	if len(b.script.Steps) > 0 {
		lastIdx := len(b.script.Steps) - 1
		b.script.Steps[lastIdx].PlayWalkingSounds = true
	}

	return b
}

// Build returns the completed script
func (b *MiniMapScriptBuilder) Build() *MiniMapScript {
	return b.script
}
