package minimap

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// ActorID is a unique identifier for an actor in a mini-map scene
type ActorID string

// Standard actor IDs for common characters
const (
	ActorAvatar      ActorID = "avatar"
	ActorLordBritish ActorID = "lb"
	ActorShamino     ActorID = "shamino"
	ActorIolo        ActorID = "iolo"
	ActorDupre       ActorID = "dupre"
	ActorGwenno      ActorID = "gwenno"
)

// Actor represents a character or entity in a mini-map scene
type Actor struct {
	ID       ActorID
	Position references.Position
	Sprite   indexes.SpriteIndex
	Facing   references.Direction
	Visible  bool
}

// NewActor creates a new actor with the given ID, position, and sprite
func NewActor(id ActorID, pos references.Position, sprite indexes.SpriteIndex) *Actor {
	return &Actor{
		ID:       id,
		Position: pos,
		Sprite:   sprite,
		Facing:   references.NoneDirection,
		Visible:  true,
	}
}
