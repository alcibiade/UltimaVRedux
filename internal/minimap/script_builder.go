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

// WalkTo adds a walk step for a single actor to reach the target position
func (b *MiniMapScriptBuilder) WalkTo(actor ActorID, target references.Position, delayMs int) *MiniMapScriptBuilder {
	return b.WalkActors([]ActorID{actor}, target, delayMs)
}

// WalkActors adds a walk step for multiple actors to reach the target position
func (b *MiniMapScriptBuilder) WalkActors(actors []ActorID, target references.Position, delayMs int) *MiniMapScriptBuilder {
	b.script.Steps = append(b.script.Steps, ScriptStep{
		Type:     StepWalk,
		ActorIDs: actors,
		Target:   &target,
		DelayMs:  delayMs,
	})

	return b
}

// SetVisible adds a visibility step for a single actor
func (b *MiniMapScriptBuilder) SetVisible(actor ActorID, visible bool) *MiniMapScriptBuilder {
	b.script.Steps = append(b.script.Steps, ScriptStep{
		Type:     StepSetVisible,
		ActorIDs: []ActorID{actor},
		Visible:  visible,
	})

	return b
}

// SetSprite adds a sprite change step for a single actor
func (b *MiniMapScriptBuilder) SetSprite(actor ActorID, sprite indexes.SpriteIndex) *MiniMapScriptBuilder {
	b.script.Steps = append(b.script.Steps, ScriptStep{
		Type:     StepSetSprite,
		ActorIDs: []ActorID{actor},
		Sprite:   sprite,
	})

	return b
}

// SetFrame selects scene-owned presentation content. The script controls when
// content changes while the rendering scene controls how that frame is drawn.
func (b *MiniMapScriptBuilder) SetFrame(frame FrameID) *MiniMapScriptBuilder {
	b.script.Steps = append(b.script.Steps, ScriptStep{ //nolint:exhaustruct // Step payload uses only frame fields.
		Type:  StepSetFrame,
		Frame: frame,
	})

	return b
}

// AwaitAdvance pauses script progression until ScriptExecutor.Advance is called.
func (b *MiniMapScriptBuilder) AwaitAdvance() *MiniMapScriptBuilder {
	b.script.Steps = append(b.script.Steps, ScriptStep{ //nolint:exhaustruct // Input gate has no payload.
		Type: StepAwaitAdvance,
	})

	return b
}

// TileCommand appends a typed operation for a script-owned tile scene.
func (b *MiniMapScriptBuilder) TileCommand(command TileCommand) *MiniMapScriptBuilder {
	b.script.Steps = append(b.script.Steps, ScriptStep{ //nolint:exhaustruct // Step payload is the tile command.
		Type:        StepTileCommand,
		TileCommand: &command,
	})

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
