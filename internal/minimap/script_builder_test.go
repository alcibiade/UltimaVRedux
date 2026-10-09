package minimap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestScriptBuilder_Creation(t *testing.T) {
	builder := NewMiniMapScript("test")

	assert.NotNil(t, builder)
}

func TestScriptBuilder_WalkTo(t *testing.T) {
	script := NewMiniMapScript("test").
		WalkTo(ActorAvatar, references.Position{X: 5, Y: 5}, 300).
		Build()

	assert.Equal(t, "test", script.Name)
	assert.Len(t, script.Steps, 1)
	assert.Equal(t, StepWalk, script.Steps[0].Type)
	assert.Equal(t, []ActorID{ActorAvatar}, script.Steps[0].ActorIDs)
	assert.Equal(t, references.Position{X: 5, Y: 5}, *script.Steps[0].Target)
	assert.Equal(t, 300, script.Steps[0].DelayMs)
}

func TestScriptBuilder_Wait(t *testing.T) {
	script := NewMiniMapScript("test").
		Wait(500).
		Build()

	assert.Len(t, script.Steps, 1)
	assert.Equal(t, StepWait, script.Steps[0].Type)
	assert.Equal(t, 500, script.Steps[0].Duration)
}

func TestScriptBuilder_Face(t *testing.T) {
	script := NewMiniMapScript("test").
		Face(ActorAvatar, references.Up).
		Build()

	assert.Len(t, script.Steps, 1)
	assert.Equal(t, StepFace, script.Steps[0].Type)
	assert.Equal(t, []ActorID{ActorAvatar}, script.Steps[0].ActorIDs)
	assert.Equal(t, references.Up, script.Steps[0].Direction)
}

func TestScriptBuilder_SetVisible(t *testing.T) {
	script := NewMiniMapScript("test").
		SetVisible(ActorAvatar, false).
		Build()

	assert.Len(t, script.Steps, 1)
	assert.Equal(t, StepSetVisible, script.Steps[0].Type)
	assert.Equal(t, []ActorID{ActorAvatar}, script.Steps[0].ActorIDs)
	assert.False(t, script.Steps[0].Visible)
}

func TestScriptBuilder_SetSprite(t *testing.T) {
	script := NewMiniMapScript("test").
		SetSprite(ActorAvatar, indexes.SpriteIndex(indexes.LordBritish1)).
		Build()

	assert.Len(t, script.Steps, 1)
	assert.Equal(t, StepSetSprite, script.Steps[0].Type)
	assert.Equal(t, []ActorID{ActorAvatar}, script.Steps[0].ActorIDs)
	assert.Equal(t, indexes.SpriteIndex(indexes.LordBritish1), script.Steps[0].Sprite)
}

func TestScriptBuilder_PresentationFrames(t *testing.T) {
	script := NewMiniMapScript("introduction").
		SetFrame("summoning-1").
		AwaitAdvance().
		SetFrame("summoning-2").
		Build()

	require.NoError(t, script.Validate())
	assert.Equal(t, StepSetFrame, script.Steps[0].Type)
	assert.Equal(t, FrameID("summoning-1"), script.Steps[0].Frame)
	assert.Equal(t, StepAwaitAdvance, script.Steps[1].Type)
}

func TestScriptBuilder_ChainedSteps(t *testing.T) {
	script := NewMiniMapScript("shrine_walk").
		WalkTo(ActorAvatar, references.Position{X: 5, Y: 5}, 300).
		Wait(200).
		Face(ActorAvatar, references.Up).
		Build()

	assert.Len(t, script.Steps, 3)
	assert.Equal(t, StepWalk, script.Steps[0].Type)
	assert.Equal(t, StepWait, script.Steps[1].Type)
	assert.Equal(t, StepFace, script.Steps[2].Type)
}

func TestScriptBuilder_WalkActors_MultipleActors(t *testing.T) {
	script := NewMiniMapScript("party_enter").
		WalkActors([]ActorID{ActorAvatar, ActorShamino, ActorIolo}, references.Position{X: 5, Y: 5}, 300).
		Build()

	assert.Len(t, script.Steps, 1)
	assert.Len(t, script.Steps[0].ActorIDs, 3)
	assert.Contains(t, script.Steps[0].ActorIDs, ActorAvatar)
	assert.Contains(t, script.Steps[0].ActorIDs, ActorShamino)
	assert.Contains(t, script.Steps[0].ActorIDs, ActorIolo)
}

func TestScriptBuilder_FaceActors_MultipleActors(t *testing.T) {
	script := NewMiniMapScript("party_face").
		FaceActors([]ActorID{ActorAvatar, ActorShamino}, references.Down).
		Build()

	assert.Len(t, script.Steps, 1)
	assert.Equal(t, StepFace, script.Steps[0].Type)
	assert.Len(t, script.Steps[0].ActorIDs, 2)
	assert.Equal(t, references.Down, script.Steps[0].Direction)
}

func TestScriptBuilder_ComplexSequence(t *testing.T) {
	// Simulate endgame-like sequence with multiple actors
	script := NewMiniMapScript("endgame_intro").
		WalkActors([]ActorID{ActorAvatar, ActorShamino, ActorIolo}, references.Position{X: 5, Y: 8}, 200).
		Wait(500).
		WalkTo(ActorLordBritish, references.Position{X: 5, Y: 6}, 200).
		Wait(300).
		FaceActors([]ActorID{ActorAvatar, ActorShamino, ActorIolo}, references.Up).
		Face(ActorLordBritish, references.Down).
		Build()

	assert.Equal(t, "endgame_intro", script.Name)
	assert.Len(t, script.Steps, 6)
}
