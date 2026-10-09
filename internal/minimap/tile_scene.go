package minimap

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

var errTileMapInvalid = errors.New("tile map is invalid")

// TileMap is presentation-owned terrain with an optional caption.
type TileMap struct {
	Width   int
	Height  int
	Tiles   []indexes.SpriteIndex
	Caption string
}

// TileEffectState exposes the active scripted effect to the presentation adapter.
type TileEffectState struct {
	Command  TileCommand
	Progress float64
}

// RegisterTileMap adds immutable source terrain to the scene.
func (s *MiniMapScene) RegisterTileMap(mapID FrameID, tileMap TileMap) error {
	if mapID == "" || tileMap.Width <= 0 || tileMap.Height <= 0 || len(tileMap.Tiles) != tileMap.Width*tileMap.Height {
		return fmt.Errorf("%w: map %q has invalid dimensions or tile count", errTileMapInvalid, mapID)
	}
	copyMap := tileMap
	copyMap.Tiles = append([]indexes.SpriteIndex(nil), tileMap.Tiles...)
	s.tileMaps[mapID] = copyMap

	return nil
}

// CurrentTileMap returns the active map ID and its source metadata.
func (s *MiniMapScene) CurrentTileMap() (FrameID, TileMap, bool) {
	tileMap, ok := s.tileMaps[s.currentMap]

	return s.currentMap, tileMap, ok
}

// TileDimensions returns the active presentation map dimensions.
func (s *MiniMapScene) TileDimensions() (int, int) {
	return s.mapWidth, s.mapHeight
}

// TileAt returns current mutable terrain.
func (s *MiniMapScene) TileAt(position references.Position) indexes.SpriteIndex {
	if !s.validTilePosition(position) {
		return 0
	}

	return s.tiles[int(position.Y)*s.mapWidth+int(position.X)]
}

// IsColumnRevealed reports whether the current center-out map transition has exposed a column.
func (s *MiniMapScene) IsColumnRevealed(x int) bool {
	return x >= s.revealMinX && x <= s.revealMaxX
}

// ActiveTileEffect returns the current effect, if any.
func (s *MiniMapScene) ActiveTileEffect() *TileEffectState {
	return s.tileEffect
}

func (s *MiniMapScene) applyTileCommand(command *TileCommand) {
	switch command.Type {
	case TileCommandSetMap:
		s.setTileMap(command.MapID)
	case TileCommandClearActors:
		s.clearActors()
	case TileCommandAddActor:
		s.AddActor(command.ActorID, command.Position, command.Sprite)
	case TileCommandRemoveActor:
		s.removeActor(command.ActorID)
	case TileCommandMoveActor:
		s.moveActor(command.ActorID, command.Direction)
	case TileCommandSetTile:
		s.setTile(command.Position, command.Sprite)
	case TileCommandRefresh:
		s.tileEffect = nil
		s.commandCycles = 0
	case TileCommandRaiseGate, TileCommandDropGate, TileCommandFizzActor, TileCommandFadeActor, TileCommandNukActor:
		s.startTileEffect(command)
	case TileCommandReset:
		s.tileEffect = nil
	}
}

func (s *MiniMapScene) startTileEffect(command *TileCommand) {
	if command.Type == TileCommandDropGate {
		s.setTile(command.Position, indexes.Grass)
	}
	s.commandCycles = 0
	s.tileEffect = &TileEffectState{Command: *command, Progress: 0}
}

func (s *MiniMapScene) updateTileCommand(command *TileCommand, elapsedMs int) {
	if command.Cycles > 0 && command.CycleMs > 0 {
		targetCycles := min(elapsedMs/command.CycleMs, command.Cycles)
		for s.commandCycles < targetCycles {
			s.advanceReveal()
			s.commandCycles++
		}
	}
	if command.Type == TileCommandRefresh {
		return
	}
	if s.tileEffect != nil && command.DurationMs > 0 {
		s.tileEffect.Progress = min(float64(elapsedMs)/float64(command.DurationMs), 1)
	}
}

func (s *MiniMapScene) finishTileCommand(command *TileCommand) {
	switch command.Type {
	case TileCommandRaiseGate:
		s.setTile(command.Position, indexes.Moongate)
	case TileCommandDropGate:
		s.setTile(command.Position, indexes.Grass)
	case TileCommandFadeActor:
		s.removeActor(command.ActorID)
	case TileCommandSetMap, TileCommandClearActors, TileCommandAddActor, TileCommandRemoveActor,
		TileCommandMoveActor, TileCommandRefresh, TileCommandSetTile, TileCommandFizzActor,
		TileCommandNukActor, TileCommandReset:
	}
	s.tileEffect = nil
}

func (s *MiniMapScene) setTileMap(id FrameID) {
	tileMap, ok := s.tileMaps[id]
	if !ok {
		panic(fmt.Sprintf("MiniMapScene: unknown tile map %q", id))
	}
	s.currentMap = id
	s.mapWidth, s.mapHeight = tileMap.Width, tileMap.Height
	s.tiles = append(s.tiles[:0], tileMap.Tiles...)
	const centerDivisor = 2
	center := tileMap.Width / centerDivisor
	s.revealMinX, s.revealMaxX = center, center
	s.revealPhase = 0
	s.commandCycles = 0
}

func (s *MiniMapScene) setTile(position references.Position, sprite indexes.SpriteIndex) {
	if s.validTilePosition(position) {
		s.tiles[int(position.Y)*s.mapWidth+int(position.X)] = sprite
	}
}

func (s *MiniMapScene) validTilePosition(position references.Position) bool {
	return int(position.X) >= 0 && int(position.X) < s.mapWidth &&
		int(position.Y) >= 0 && int(position.Y) < s.mapHeight
}

func (s *MiniMapScene) moveActor(id ActorID, direction references.Direction) {
	actor := s.actors[id]
	if actor == nil {
		return
	}
	switch direction {
	case references.Up:
		actor.Position.Y--
	case references.Right:
		actor.Position.X++
	case references.Down:
		actor.Position.Y++
	case references.Left:
		actor.Position.X--
	case references.NoneDirection:
	}
}

func (s *MiniMapScene) removeActor(actorID ActorID) {
	delete(s.actors, actorID)
	for index, orderedID := range s.actorOrder {
		if orderedID == actorID {
			s.actorOrder = append(s.actorOrder[:index], s.actorOrder[index+1:]...)

			return
		}
	}
}

func (s *MiniMapScene) clearActors() {
	s.actors = make(map[ActorID]*Actor)
	s.actorOrder = s.actorOrder[:0]
}

func (s *MiniMapScene) advanceReveal() {
	if s.revealMinX == 0 && s.revealMaxX == s.mapWidth-1 {
		return
	}
	if s.revealPhase%2 == 1 {
		s.revealMinX = max(0, s.revealMinX-1)
		s.revealMaxX = min(s.mapWidth-1, s.revealMaxX+1)
	}
	s.revealPhase++
}
