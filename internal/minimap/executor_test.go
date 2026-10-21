package minimap

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestScriptExecutor_Creation(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)

	script := NewMiniMapScript("test").
		WalkTo(ActorAvatar, references.Position{X: 5, Y: 5}, 300).
		Build()

	executor := NewScriptExecutor(scene, script)

	assert.NotNil(t, executor)
	assert.False(t, executor.IsComplete())
}

func TestScriptExecutor_EmptyScript_CompletesImmediately(t *testing.T) {
	scene := NewMiniMapScene()
	script := NewMiniMapScript("empty").Build()

	executor := NewScriptExecutor(scene, script)

	assert.True(t, executor.IsComplete())
}

func TestScriptExecutor_WalkOneStep(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)

	// Walk one step north (from Y=10 to Y=9)
	script := NewMiniMapScript("test").
		WalkTo(ActorAvatar, references.Position{X: 5, Y: 9}, 100).
		Build()

	executor := NewScriptExecutor(scene, script)

	// Simulate enough time for one step (100ms delay per step)
	for !executor.IsComplete() {
		executor.Update(16) // ~60fps
	}

	assert.Equal(t, references.Position{X: 5, Y: 9}, scene.GetActorPosition(ActorAvatar))
}

func TestScriptExecutor_WalkMultipleSteps(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)

	// Walk 5 steps north (from Y=10 to Y=5)
	script := NewMiniMapScript("test").
		WalkTo(ActorAvatar, references.Position{X: 5, Y: 5}, 50).
		Build()

	executor := NewScriptExecutor(scene, script)

	// Run to completion
	for !executor.IsComplete() {
		executor.Update(16)
	}

	assert.Equal(t, references.Position{X: 5, Y: 5}, scene.GetActorPosition(ActorAvatar))
}

func TestScriptExecutor_WalkDiagonal(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 3, Y: 10}, indexes.AvatarKeyIndex)

	// Walk to diagonal position (from 3,10 to 5,5)
	script := NewMiniMapScript("test").
		WalkTo(ActorAvatar, references.Position{X: 5, Y: 5}, 50).
		Build()

	executor := NewScriptExecutor(scene, script)

	for !executor.IsComplete() {
		executor.Update(16)
	}

	assert.Equal(t, references.Position{X: 5, Y: 5}, scene.GetActorPosition(ActorAvatar))
}

func TestScriptExecutor_Wait(t *testing.T) {
	scene := NewMiniMapScene()
	script := NewMiniMapScript("test").
		Wait(500).
		Build()

	executor := NewScriptExecutor(scene, script)

	// Not complete after 400ms
	executor.Update(400)
	assert.False(t, executor.IsComplete())

	// Complete after 500ms total
	executor.Update(100)
	assert.True(t, executor.IsComplete())
}

func TestScriptExecutor_Face(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 5}, indexes.AvatarKeyIndex)

	script := NewMiniMapScript("test").
		Face(ActorAvatar, references.Right).
		Build()

	executor := NewScriptExecutor(scene, script)
	executor.Update(16)

	assert.Equal(t, references.Right, scene.GetActor(ActorAvatar).Facing)
	assert.True(t, executor.IsComplete())
}

func TestScriptExecutor_SetVisible(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 5}, indexes.AvatarKeyIndex)

	script := NewMiniMapScript("test").
		SetVisible(ActorAvatar, false).
		Build()

	executor := NewScriptExecutor(scene, script)
	executor.Update(16)

	assert.False(t, scene.GetActor(ActorAvatar).Visible)
	assert.True(t, executor.IsComplete())
}

func TestScriptExecutor_SetSprite(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 5}, indexes.AvatarKeyIndex)

	script := NewMiniMapScript("test").
		SetSprite(ActorAvatar, indexes.SpriteIndex(indexes.LordBritish1)).
		Build()

	executor := NewScriptExecutor(scene, script)
	executor.Update(16)

	assert.Equal(t, indexes.SpriteIndex(indexes.LordBritish1), scene.GetActor(ActorAvatar).Sprite)
	assert.True(t, executor.IsComplete())
}

func TestScriptExecutor_MultipleActorsWalkSimultaneously(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)
	scene.AddActor(ActorShamino, references.Position{X: 4, Y: 10}, indexes.SpriteIndex(indexes.Bard1))

	script := NewMiniMapScript("test").
		WalkActors([]ActorID{ActorAvatar, ActorShamino}, references.Position{X: 5, Y: 5}, 50).
		Build()

	executor := NewScriptExecutor(scene, script)

	for !executor.IsComplete() {
		executor.Update(16)
	}

	assert.Equal(t, references.Position{X: 5, Y: 5}, scene.GetActorPosition(ActorAvatar))
	assert.Equal(t, references.Position{X: 5, Y: 5}, scene.GetActorPosition(ActorShamino))
}

func TestScriptExecutor_SequentialSteps(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)

	script := NewMiniMapScript("test").
		WalkTo(ActorAvatar, references.Position{X: 5, Y: 8}, 50).
		Wait(100).
		Face(ActorAvatar, references.Up).
		Build()

	executor := NewScriptExecutor(scene, script)

	for !executor.IsComplete() {
		executor.Update(16)
	}

	assert.Equal(t, references.Position{X: 5, Y: 8}, scene.GetActorPosition(ActorAvatar))
	assert.Equal(t, references.Up, scene.GetActor(ActorAvatar).Facing)
}

func TestScriptExecutor_GetCurrentStepIndex(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)

	script := NewMiniMapScript("test").
		WalkTo(ActorAvatar, references.Position{X: 5, Y: 9}, 50).
		Wait(100).
		Build()

	executor := NewScriptExecutor(scene, script)

	assert.Equal(t, 0, executor.GetCurrentStepIndex())

	// Complete first step
	for executor.GetCurrentStepIndex() == 0 && !executor.IsComplete() {
		executor.Update(16)
	}

	// Should be on second step
	if !executor.IsComplete() {
		assert.Equal(t, 1, executor.GetCurrentStepIndex())
	}
}

func TestScriptExecutor_PresentationFramesWaitForAdvance(t *testing.T) {
	scene := NewMiniMapScene()
	script := NewMiniMapScript("introduction").
		SetFrame("frame-1").
		AwaitAdvance().
		SetFrame("frame-2").
		AwaitAdvance().
		Build()
	executor := NewScriptExecutor(scene, script)

	executor.Update(16)

	assert.Equal(t, FrameID("frame-1"), scene.GetCurrentFrame())
	assert.True(t, executor.IsWaitingForAdvance())
	assert.True(t, executor.Advance())
	assert.Equal(t, FrameID("frame-2"), scene.GetCurrentFrame())
	assert.True(t, executor.IsWaitingForAdvance())
	assert.True(t, executor.Advance())
	assert.True(t, executor.IsComplete())
	assert.False(t, executor.Advance())
}

func TestScriptExecutor_RejectsInvalidScript(t *testing.T) {
	scene := NewMiniMapScene()
	script := &Script{Steps: []ScriptStep{{Type: StepWalk}}}

	assert.Panics(t, func() {
		NewScriptExecutor(scene, script)
	})
}

func TestScriptExecutor_NilScene_Panics(t *testing.T) {
	script := NewMiniMapScript("test").Build()

	assert.Panics(t, func() {
		NewScriptExecutor(nil, script)
	})
}

func TestScriptExecutor_NilScript_Panics(t *testing.T) {
	scene := NewMiniMapScene()

	assert.Panics(t, func() {
		NewScriptExecutor(scene, nil)
	})
}
