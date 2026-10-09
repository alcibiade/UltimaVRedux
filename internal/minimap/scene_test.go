package minimap

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestNewMiniMapScene(t *testing.T) {
	scene := NewMiniMapScene()

	assert.NotNil(t, scene)
	assert.NotNil(t, scene.actors)
	assert.Empty(t, scene.actors)
}

func TestMiniMapScene_AddActor(t *testing.T) {
	scene := NewMiniMapScene()

	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)

	actor := scene.GetActor(ActorAvatar)
	assert.NotNil(t, actor)
	assert.Equal(t, references.Position{X: 5, Y: 10}, actor.Position)
	assert.Equal(t, indexes.AvatarKeyIndex, actor.Sprite)
}

func TestMiniMapScene_AddActor_Chaining(t *testing.T) {
	scene := NewMiniMapScene().
		AddActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex).
		AddActor(ActorLordBritish, references.Position{X: 5, Y: 8}, indexes.LordBritish1)

	assert.Equal(t, 2, len(scene.actors))
	assert.NotNil(t, scene.GetActor(ActorAvatar))
	assert.NotNil(t, scene.GetActor(ActorLordBritish))
}

func TestMiniMapScene_GetActor_NotFound(t *testing.T) {
	scene := NewMiniMapScene()

	actor := scene.GetActor(ActorAvatar)

	assert.Nil(t, actor)
}

func TestMiniMapScene_GetActorPosition(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)

	pos := scene.GetActorPosition(ActorAvatar)

	assert.Equal(t, references.Position{X: 5, Y: 10}, pos)
}

func TestMiniMapScene_GetActorPosition_NotFound(t *testing.T) {
	scene := NewMiniMapScene()

	pos := scene.GetActorPosition(ActorAvatar)

	// Returns zero position for non-existent actor
	assert.Equal(t, references.Position{X: 0, Y: 0}, pos)
}

func TestMiniMapScene_SetActorPosition(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)

	scene.SetActorPosition(ActorAvatar, references.Position{X: 5, Y: 5})

	assert.Equal(t, references.Position{X: 5, Y: 5}, scene.GetActorPosition(ActorAvatar))
}

func TestMiniMapScene_SetActorPosition_NotFound(t *testing.T) {
	scene := NewMiniMapScene()

	// Should not panic when setting position for non-existent actor
	scene.SetActorPosition(ActorAvatar, references.Position{X: 5, Y: 5})

	// Actor still doesn't exist
	assert.Nil(t, scene.GetActor(ActorAvatar))
}

func TestMiniMapScene_SetActorSprite(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)

	scene.SetActorSprite(ActorAvatar, indexes.SpriteIndex(indexes.LordBritish1))

	actor := scene.GetActor(ActorAvatar)
	assert.Equal(t, indexes.SpriteIndex(indexes.LordBritish1), actor.Sprite)
}

func TestMiniMapScene_SetActorVisible(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)

	scene.SetActorVisible(ActorAvatar, false)

	actor := scene.GetActor(ActorAvatar)
	assert.False(t, actor.Visible)

	scene.SetActorVisible(ActorAvatar, true)

	assert.True(t, actor.Visible)
}

func TestMiniMapScene_SetActorFacing(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)

	scene.SetActorFacing(ActorAvatar, references.Up)

	actor := scene.GetActor(ActorAvatar)
	assert.Equal(t, references.Up, actor.Facing)
}

func TestMiniMapScene_CameraCenter_DefaultsTo5_5(t *testing.T) {
	scene := NewMiniMapScene()

	center := scene.GetCameraCenter()

	assert.Equal(t, references.Position{X: 5, Y: 5}, center)
}

func TestMiniMapScene_SetCameraCenter(t *testing.T) {
	scene := NewMiniMapScene()

	scene.SetCameraCenter(references.Position{X: 3, Y: 7})

	assert.Equal(t, references.Position{X: 3, Y: 7}, scene.GetCameraCenter())
}

func TestMiniMapScene_SetCurrentFrame(t *testing.T) {
	scene := NewMiniMapScene()

	scene.SetCurrentFrame("arrival-1")

	assert.Equal(t, FrameID("arrival-1"), scene.GetCurrentFrame())
}

func TestMiniMapScene_GetAllActors(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)
	scene.AddActor(ActorLordBritish, references.Position{X: 5, Y: 8}, indexes.LordBritish1)

	actors := scene.GetAllActors()

	assert.Len(t, actors, 2)
}

func TestMiniMapScene_GetAllActors_Empty(t *testing.T) {
	scene := NewMiniMapScene()

	actors := scene.GetAllActors()

	assert.Len(t, actors, 0)
}

func TestMiniMapScene_GetVisibleActors(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)
	scene.AddActor(ActorLordBritish, references.Position{X: 5, Y: 8}, indexes.LordBritish1)
	scene.SetActorVisible(ActorLordBritish, false)

	visibleActors := scene.GetVisibleActors()

	assert.Len(t, visibleActors, 1)
	assert.Equal(t, ActorAvatar, visibleActors[0].ID)
}
