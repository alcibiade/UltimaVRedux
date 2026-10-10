package minimap

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestActorID_Constants(t *testing.T) {
	assert.Equal(t, ActorID("avatar"), ActorAvatar)
	assert.Equal(t, ActorID("lb"), ActorLordBritish)
	assert.Equal(t, ActorID("shamino"), ActorShamino)
	assert.Equal(t, ActorID("iolo"), ActorIolo)
}

func TestNewActor(t *testing.T) {
	actor := NewActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)

	assert.Equal(t, ActorAvatar, actor.ID)
	assert.Equal(t, references.Position{X: 5, Y: 10}, actor.Position)
	assert.Equal(t, indexes.AvatarKeyIndex, actor.Sprite)
	assert.True(t, actor.Visible)
	assert.Equal(t, references.NoneDirection, actor.Facing)
}

func TestActor_SetPosition(t *testing.T) {
	actor := NewActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)

	actor.Position = references.Position{X: 5, Y: 5}

	assert.Equal(t, references.Position{X: 5, Y: 5}, actor.Position)
}

func TestActor_SetVisible(t *testing.T) {
	actor := NewActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)
	assert.True(t, actor.Visible)

	actor.Visible = false

	assert.False(t, actor.Visible)
}

func TestActor_SetFacing(t *testing.T) {
	actor := NewActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)

	actor.Facing = references.Up

	assert.Equal(t, references.Up, actor.Facing)
}

func TestActor_SetSprite(t *testing.T) {
	actor := NewActor(ActorAvatar, references.Position{X: 5, Y: 10}, indexes.AvatarKeyIndex)

	actor.Sprite = indexes.SpriteIndex(indexes.LordBritish1)

	assert.Equal(t, indexes.SpriteIndex(indexes.LordBritish1), actor.Sprite)
}
