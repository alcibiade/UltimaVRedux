//nolint:mnd // Original 19x4 tile-strip dimensions and presentation coordinates.
package main

import (
	"errors"
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/minimap"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/visual"
	"github.com/bradhannah/UltimaVRedux/pkg/color"
	rng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

var errReturnViewPresentationInvalid = errors.New("return to the View presentation is invalid")

const (
	returnViewWorkflowName = "return_to_the_view"
	returnViewStripX       = 8
	returnViewStripY       = 128
	returnViewFizzSeed     = 0x5254545646495A5A
	returnViewCaptionY     = 192
	returnViewTextColumns  = 36
)

type returnViewPresentation struct {
	scene          *minimap.MiniMapScene
	executor       *minimap.ScriptExecutor
	spriteSheet    *sprites.SpriteSheet
	dissolveType   minimap.TileCommandType
	dissolveActor  minimap.ActorID
	dissolve       *visual.PixelDissolveEffect
	dissolveBuffer *ebiten.Image
}

func newReturnViewPresentation(data *minimap.ReturnViewData) (*returnViewPresentation, error) {
	if data == nil || data.Script == nil {
		return nil, fmt.Errorf("creating Return to the View presentation: %w", errReturnViewPresentationInvalid)
	}
	scene := minimap.NewMiniMapScene()
	for id, tileMap := range data.Maps {
		if err := scene.RegisterTileMap(id, tileMap); err != nil {
			return nil, fmt.Errorf("creating Return to the View presentation: %w", err)
		}
	}
	executor := minimap.NewScriptExecutor(scene, data.Script)
	executor.Update(0)

	return &returnViewPresentation{
		scene: scene, executor: executor, spriteSheet: sprites.NewSpriteSheet(),
		dissolveType: 0, dissolveActor: "", dissolve: nil, dissolveBuffer: nil,
	}, nil
}

func (p *returnViewPresentation) caption() string {
	_, tileMap, ok := p.scene.CurrentTileMap()
	if !ok {
		return ""
	}

	return tileMap.Caption
}

func (p *returnViewPresentation) draw(canvas *ebiten.Image, elapsedMs int64) {
	canvas.Clear()
	width, height := p.scene.TileDimensions()
	for tileY := range height {
		for tileX := range width {
			if !p.scene.IsColumnRevealed(tileX) {
				continue
			}
			position := minimapPosition(tileX, tileY)
			tile := p.scene.TileAt(position)
			positionHash := int32(tileY*width + tileX) //nolint:gosec // Tiny fixed grid.
			sprite := p.spriteSheet.GetRawAnimatedSprite(tile, positionHash, elapsedMs)
			drawIntroImage(
				canvas, sprite,
				returnViewStripX+tileX*sprites.TileSize,
				returnViewStripY+tileY*sprites.TileSize,
			)
		}
	}
	for _, actor := range p.scene.GetVisibleActors() {
		p.drawActor(canvas, actor, elapsedMs)
	}
	p.drawEffect(canvas)
	p.drawCaptionBackground(canvas)
}

func (p *returnViewPresentation) drawActor(canvas *ebiten.Image, actor *minimap.Actor, elapsedMs int64) {
	alpha := float32(1)
	effect := p.scene.ActiveTileEffect()
	if effect != nil && effect.Command.ActorID == actor.ID {
		switch effect.Command.Type {
		case minimap.TileCommandFizzActor, minimap.TileCommandFadeActor:
			p.drawDissolvingActor(canvas, actor, elapsedMs, effect)

			return
		case minimap.TileCommandNukActor:
			if effect.Progress > 0.6 {
				alpha = 0
			}
		case minimap.TileCommandSetMap, minimap.TileCommandClearActors, minimap.TileCommandAddActor,
			minimap.TileCommandRemoveActor, minimap.TileCommandMoveActor, minimap.TileCommandRefresh,
			minimap.TileCommandSetTile, minimap.TileCommandRaiseGate, minimap.TileCommandDropGate,
			minimap.TileCommandReset:
		}
	}
	positionHash := int32(int(actor.Position.Y)*19 + int(actor.Position.X)) //nolint:gosec // Tiny fixed grid.
	sprite := p.spriteSheet.GetRawAnimatedSprite(actor.Sprite, positionHash, elapsedMs)
	options := introImageDrawOptions(
		returnViewStripX+int(actor.Position.X)*sprites.TileSize,
		returnViewStripY+int(actor.Position.Y)*sprites.TileSize,
	)
	options.ColorScale.ScaleAlpha(alpha)
	canvas.DrawImage(sprite, options)
}

func (p *returnViewPresentation) drawDissolvingActor(
	canvas *ebiten.Image,
	actor *minimap.Actor,
	elapsedMs int64,
	effect *minimap.TileEffectState,
) {
	if p.dissolve == nil || p.dissolveType != effect.Command.Type || p.dissolveActor != actor.ID {
		mode := visual.DissolveModeReveal
		if effect.Command.Type == minimap.TileCommandFadeActor {
			mode = visual.DissolveModeHide
		}
		p.dissolveType = effect.Command.Type
		p.dissolveActor = actor.ID
		p.dissolve = visual.NewPixelDissolveEffect(
			sprites.TileSize, sprites.TileSize, sprites.TileSize, sprites.TileSize,
			mode,
			rng.New(returnViewFizzSeed),
		)
		p.dissolveBuffer = ebiten.NewImage(sprites.TileSize, sprites.TileSize)
	}
	p.dissolve.UpdateProgress(effect.Progress)
	positionHash := int32(int(actor.Position.Y)*19 + int(actor.Position.X)) //nolint:gosec // Tiny fixed grid.
	sprite := p.spriteSheet.GetRawAnimatedSprite(actor.Sprite, positionHash, elapsedMs)
	drawIntroDissolve(
		canvas,
		sprite,
		p.dissolve,
		p.dissolveBuffer,
		introImageDrawOptions(
			returnViewStripX+int(actor.Position.X)*sprites.TileSize,
			returnViewStripY+int(actor.Position.Y)*sprites.TileSize,
		),
	)
}

func (p *returnViewPresentation) drawCaptionBackground(canvas *ebiten.Image) {
	leftCell, rightCell := returnViewCaptionCells(p.caption())
	leftX, rightX := float32(leftCell*8), float32(rightCell*8)
	vector.FillRect(canvas, leftX, returnViewCaptionY, rightX-leftX, 8, color.Black, false)
}

func returnViewCaptionCells(caption string) (int, int) {
	left := returnViewTextColumns/2 - len(caption)/2

	return left, left + len(caption) + 2
}

func (p *returnViewPresentation) drawEffect(canvas *ebiten.Image) {
	effect := p.scene.ActiveTileEffect()
	if effect == nil {
		return
	}
	switch effect.Command.Type {
	case minimap.TileCommandRaiseGate, minimap.TileCommandDropGate:
		sprite := p.spriteSheet.GetRawSprite(indexes.Moongate)
		moongate, offsetY := verticallyRevealMoongate(sprite, returnViewMoongateRows(effect))
		if moongate == nil {
			return
		}
		options := introImageDrawOptions(
			returnViewStripX+int(effect.Command.Position.X)*sprites.TileSize,
			returnViewStripY+int(effect.Command.Position.Y)*sprites.TileSize+int(offsetY),
		)
		canvas.DrawImage(moongate, options)
	case minimap.TileCommandNukActor:
		actor := p.scene.GetActor(effect.Command.ActorID)
		if actor == nil {
			return
		}
		targetX := float32(returnViewStripX + int(actor.Position.X)*sprites.TileSize + sprites.TileSize/2)
		targetY := float32(returnViewStripY + int(actor.Position.Y)*sprites.TileSize + sprites.TileSize/2)
		spread := float32(16 * effect.Progress)
		vector.StrokeLine(canvas, targetX-spread, targetY-spread, targetX+spread, targetY+spread, 1, color.White, false)
		vector.StrokeLine(canvas, targetX+spread, targetY-spread, targetX-spread, targetY+spread, 1, color.White, false)
	case minimap.TileCommandSetMap, minimap.TileCommandClearActors, minimap.TileCommandAddActor,
		minimap.TileCommandRemoveActor, minimap.TileCommandMoveActor, minimap.TileCommandRefresh,
		minimap.TileCommandSetTile, minimap.TileCommandFizzActor, minimap.TileCommandFadeActor,
		minimap.TileCommandReset:
	}
}

func returnViewMoongateRows(effect *minimap.TileEffectState) int {
	rowsChanged := int(math.Ceil(effect.Progress * sprites.TileSize))
	if effect.Command.Type == minimap.TileCommandDropGate {
		return sprites.TileSize - rowsChanged
	}

	return rowsChanged
}

func minimapPosition(x, y int) references.Position {
	return references.Position{
		X: references.Coordinate(x), //nolint:gosec // Bounded by the 19x4 map.
		Y: references.Coordinate(y), //nolint:gosec // Bounded by the 19x4 map.
	}
}
