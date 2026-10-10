package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	moongatePulseCycle     = 2000
	moongateBaseOpacity    = .85
	moongatePulseAmplitude = 0.25
	fullCircleRadians      = 2
)

// MoongateRenderer owns only presentation resources and source identity. The
// shared GameState stage determines whether and how large every gate renders.
type MoongateRenderer struct {
	position       references.Position
	phaseIndex     int
	moongateSprite *ebiten.Image
}

func NewMoongateRenderer(position references.Position, phaseIndex int) *MoongateRenderer {
	return &MoongateRenderer{position: position, phaseIndex: phaseIndex, moongateSprite: nil}
}

func (mr *MoongateRenderer) CalculateCurrentOpacity(elapsedMs uint64) float32 {
	phase := float64(elapsedMs%uint64(moongatePulseCycle)) /
		float64(moongatePulseCycle) * fullCircleRadians * math.Pi

	return moongateBaseOpacity + float32(math.Sin(phase)*moongatePulseAmplitude)
}

func (mr *MoongateRenderer) GetPosition() references.Position {
	return mr.position
}

func (mr *MoongateRenderer) GetPhaseIndex() int {
	return mr.phaseIndex
}

// GetDrawInfo derives visible rows directly from the authoritative stage.
func (mr *MoongateRenderer) GetDrawInfo(
	stage game_state.MoongateStage,
	elapsedMs uint64,
) (float32, int, bool) {
	if stage == game_state.MoongateStageHidden || mr.moongateSprite == nil {
		return 0, 0, false
	}

	return mr.CalculateCurrentOpacity(elapsedMs), int(stage), true
}

func (mr *MoongateRenderer) SetSprite(sprite *ebiten.Image) {
	mr.moongateSprite = sprite
}

// MoongateVisualSystem owns renderer resources, never moongate stage.
type MoongateVisualSystem struct {
	activeRenderers map[int]*MoongateRenderer
	gameState       *game_state.GameState
}

func NewMoongateVisualSystem(gameState *game_state.GameState) *MoongateVisualSystem {
	return &MoongateVisualSystem{activeRenderers: make(map[int]*MoongateRenderer), gameState: gameState}
}

// Update synchronizes nearby renderer resources with authoritative gameplay state.
func (mvs *MoongateVisualSystem) Update() {
	if mvs.gameState == nil {
		panic("MoongateVisualSystem.Update: gameState is nil")
	}
	if mvs.gameState.MoongateStage() == game_state.MoongateStageHidden {
		if len(mvs.activeRenderers) != 0 {
			mvs.activeRenderers = make(map[int]*MoongateRenderer)
		}

		return
	}

	const maxMoongateProcessingDistance = 30
	playerPos := mvs.gameState.MapState.PlayerLocation.Position
	moonstones := mvs.gameState.GetMoonstones()
	desired := make(map[int]bool)
	for moonstoneID, moonstone := range moonstones {
		position := references.Position{X: references.Coordinate(moonstone.X), Y: references.Coordinate(moonstone.Y)}
		distance := playerPos.GetWrappedDistanceBetweenWrapped(
			&position,
			references.XLargeMapTiles,
			references.YLargeMapTiles,
		)
		if distance > maxMoongateProcessingDistance || !mvs.isCurrentSource(moonstone) {
			continue
		}
		desired[moonstoneID] = true
		if renderer := mvs.activeRenderers[moonstoneID]; renderer == nil || renderer.position != position {
			mvs.activeRenderers[moonstoneID] = NewMoongateRenderer(position, moonstoneID)
		}
	}

	for id := range mvs.activeRenderers {
		if !desired[id] {
			delete(mvs.activeRenderers, id)
		}
	}
}

func (mvs *MoongateVisualSystem) isCurrentSource(stone game_state.MoonstoneLocation) bool {
	location := mvs.gameState.MapState.PlayerLocation
	level := uint8(location.Floor) //nolint:gosec // Moonstone save levels encode signed floor -1 as 0xff.

	return stone.Map == uint8(location.Location) && stone.Level == level && stone.CanCreateMoongate()
}

// Consumers include drawing, light collection, and ambient audio. Recheck the
// source here so a layer transition cannot expose stale resources before Update.
func (mvs *MoongateVisualSystem) isCurrentRenderer(renderer *MoongateRenderer) bool {
	stone := mvs.gameState.GetMoonstones()[renderer.phaseIndex]

	return mvs.isCurrentSource(stone) && renderer.position == (references.Position{
		X: references.Coordinate(stone.X), Y: references.Coordinate(stone.Y),
	})
}

func (mvs *MoongateVisualSystem) ResetForNewLocation() {
	logging.Debug(
		logging.CategoryMoongate,
		"Clearing %d moongate renderers for location transition",
		len(mvs.activeRenderers),
	)
	mvs.activeRenderers = make(map[int]*MoongateRenderer)
}

func (mvs *MoongateVisualSystem) GetActiveRenderers() []*MoongateRenderer {
	if mvs.gameState.MoongateStage() == game_state.MoongateStageHidden {
		return nil
	}
	renderers := make([]*MoongateRenderer, 0, len(mvs.activeRenderers))
	for _, renderer := range mvs.activeRenderers {
		if mvs.isCurrentRenderer(renderer) {
			renderers = append(renderers, renderer)
		}
	}

	return renderers
}

func (mvs *MoongateVisualSystem) GetActiveMoongateAt(pos references.Position) *MoongateRenderer {
	if mvs.gameState.MoongateStage() == game_state.MoongateStageHidden {
		return nil
	}
	for _, renderer := range mvs.activeRenderers {
		if renderer.GetPosition() == pos && mvs.isCurrentRenderer(renderer) {
			return renderer
		}
	}

	return nil
}

func (mvs *MoongateVisualSystem) GetVisibleMoongatePositions() []references.Position {
	renderers := mvs.GetActiveRenderers()
	positions := make([]references.Position, 0, len(renderers))
	for _, renderer := range renderers {
		positions = append(positions, renderer.GetPosition())
	}

	return positions
}
