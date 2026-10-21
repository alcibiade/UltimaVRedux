package minimap

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Default camera center for 11x11 mini-maps
const defaultCameraCenter = 5

// MiniMapScene represents a mini-map scene (shrine, codex, endgame, etc.)
// It holds the current state of all actors and the camera position.
type MiniMapScene struct {
	actors        map[ActorID]*Actor
	actorOrder    []ActorID
	cameraCenter  references.Position
	currentFrame  FrameID
	tileMaps      map[FrameID]TileMap
	currentMap    FrameID
	tiles         []indexes.SpriteIndex
	mapWidth      int
	mapHeight     int
	revealMinX    int
	revealMaxX    int
	revealPhase   int
	commandCycles int
	tileEffect    *TileEffectState
}

// GetCurrentFrame returns the scene-owned presentation frame selected by the script.
func (s *MiniMapScene) GetCurrentFrame() FrameID {
	return s.currentFrame
}

// SetCurrentFrame selects scene-owned presentation content.
func (s *MiniMapScene) SetCurrentFrame(frame FrameID) {
	s.currentFrame = frame
}

// NewMiniMapScene creates a new mini-map scene with default camera at (5,5)
func NewMiniMapScene() *MiniMapScene {
	return &MiniMapScene{
		actors:     make(map[ActorID]*Actor),
		actorOrder: make([]ActorID, 0),
		cameraCenter: references.Position{
			X: defaultCameraCenter,
			Y: defaultCameraCenter,
		},
		currentFrame: "", tileMaps: make(map[FrameID]TileMap), currentMap: "", tiles: nil,
		mapWidth: 0, mapHeight: 0, revealMinX: 0, revealMaxX: -1,
		revealPhase: 0, commandCycles: 0, tileEffect: nil,
	}
}

// AddActor adds an actor to the scene at the given position with the given sprite.
// Returns the scene for method chaining.
func (s *MiniMapScene) AddActor(id ActorID, pos references.Position, sprite indexes.SpriteIndex) *MiniMapScene {
	if _, exists := s.actors[id]; !exists {
		s.actorOrder = append(s.actorOrder, id)
	}
	s.actors[id] = NewActor(id, pos, sprite)

	return s
}

// GetActor returns the actor with the given ID, or nil if not found
func (s *MiniMapScene) GetActor(id ActorID) *Actor {
	return s.actors[id]
}

// SetActorPosition updates the position of the actor with the given ID.
// Does nothing if actor not found.
func (s *MiniMapScene) SetActorPosition(id ActorID, pos references.Position) {
	actor := s.actors[id]
	if actor == nil {
		return
	}

	actor.Position = pos
}

// SetActorSprite updates the sprite of the actor with the given ID.
// Does nothing if actor not found.
func (s *MiniMapScene) SetActorSprite(id ActorID, sprite indexes.SpriteIndex) {
	actor := s.actors[id]
	if actor == nil {
		return
	}

	actor.Sprite = sprite
}

// SetActorVisible updates the visibility of the actor with the given ID.
// Does nothing if actor not found.
func (s *MiniMapScene) SetActorVisible(id ActorID, visible bool) {
	actor := s.actors[id]
	if actor == nil {
		return
	}

	actor.Visible = visible
}

// SetActorFacing updates the facing direction of the actor with the given ID.
// Does nothing if actor not found.
func (s *MiniMapScene) SetActorFacing(id ActorID, facing references.Direction) {
	actor := s.actors[id]
	if actor == nil {
		return
	}

	actor.Facing = facing
}

// GetCameraCenter returns the camera center position
func (s *MiniMapScene) GetCameraCenter() references.Position {
	return s.cameraCenter
}

// GetVisibleActors returns a slice of all visible actors in the scene
func (s *MiniMapScene) GetVisibleActors() []*Actor {
	actors := make([]*Actor, 0, len(s.actorOrder))
	for _, id := range s.actorOrder {
		if actor := s.actors[id]; actor != nil && actor.Visible {
			actors = append(actors, actor)
		}
	}

	return actors
}
