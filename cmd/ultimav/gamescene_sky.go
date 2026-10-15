package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

const (
	skyCells            = 12
	skyWidthNumerator   = 3
	skyWidthDenominator = 2
	skySunGlyph         = 0x2a
	skyMoonGlyph        = 0x30
	skySunSetHour       = 17
	skyTrammelSetHour   = 8
	skyFeluccaSetHour   = 2
)

// skyStrip is a presentation snapshot, never a second calendar. A zero glyph
// means an empty cell, not phase zero (whose actual RUNES.CH glyph is 0x30).
type skyStrip struct {
	visible bool
	glyphs  [skyCells]int
}

type skyPresentation struct {
	owner       *game_state.GameState
	strip       skyStrip
	initialized bool
}

// ref:ASTRO-0001: positions are hourly and independent of the 28-day phases.
func projectSky(date datetime.UltimaDate) skyStrip {
	strip := skyStrip{visible: true}
	place := func(position, glyph int) {
		if position >= 0 && position < skyCells {
			strip.glyphs[position] = glyph
		}
	}
	place(skySunSetHour-int(date.Hour), skySunGlyph)
	moonPosition := func(setHour int) int {
		position := setHour - int(date.Hour)
		if position < -skyCells {
			position += datetime.HoursPerDay
		}

		return position
	}
	place(moonPosition(skyTrammelSetHour), skyMoonGlyph+int(date.GetTrammelMoonPhase()))
	place(moonPosition(skyFeluccaSetHour), skyMoonGlyph+int(date.GetFeluccaMoonPhase()))

	return strip
}

func skyEligible(player references.PlayerLocation) bool {
	return player.Location <= references.SerpentsHold && player.Location != references.Ararat && player.Floor >= 0
}

// skyOrigin retains the real world/floor when the viewport is a tactical map.
func (g *GameScene) skyOrigin() references.PlayerLocation {
	if combat := g.gameState.CombatScene; combat != nil && g.gameState.IsInCombat() {
		return combat.PreCombatPlayerLocation
	}
	if camp := g.gameState.CampingScene; camp != nil && g.gameState.IsInCamping() {
		return camp.OriginalPlayerLocation
	}
	if shrine := g.gameState.ShrineScene; shrine != nil && g.gameState.IsInShrine() {
		return shrine.OriginalPlayerLocation
	}

	return g.gameState.MapState.PlayerLocation
}

func (g *GameScene) suppressSkyPresentation() bool {
	if g.gameState == nil || g.finalePresentation != nil || g.usesCinematicSplitLayout() {
		return true
	}
	if g.miniMapScene != nil && !g.gameState.IsInShrine() {
		return true
	}
	queues := g.gameState.EventQueueManager

	return queues != nil && queues.HasWorkflow(deathResurrectionWorkflowName)
}

// Called before input/workflows and at drawing, so retention also works when
// Ebitengine runs multiple updates without a Draw. New/load states cannot inherit
// another GameState's presentation. Debug date edits are read directly.
func (g *GameScene) syncSkyPresentation() {
	if g.sky.owner != g.gameState {
		g.sky = skyPresentation{owner: g.gameState}
	}
	if g.suppressSkyPresentation() {
		g.sky.strip = skyStrip{}
		g.sky.initialized = false

		return
	}
	if !skyEligible(g.skyOrigin()) {
		g.sky.strip = skyStrip{}
		g.sky.initialized = true

		return
	}
	if g.sky.initialized && (g.gameState.IsInCombat() || g.gameState.IsInShrine()) {
		return
	}
	// REDUX DEVIATION: ASTRO-0002: original outdoor return charges zero minutes and
	// refreshes only when the hour changes, sometimes leaving outdoor combat's
	// inherited sky stale. Refresh immediately from the current authoritative
	// date on return, so navigation information is reliable.
	g.sky.strip = projectSky(g.gameState.DateTime)
	g.sky.initialized = true
}

// Prefer whole-pixel magnification; the smallest frame band needs a fractional
// nearest-neighbour reduction to keep all fourteen cells inside the border.
func skyStripBounds(outer, aperture image.Rectangle) image.Rectangle {
	cellSize := min(aperture.Min.Y-outer.Min.Y, outer.Dx()/(skyCells+2))
	if cellSize < 1 {
		return image.Rectangle{}
	}
	if cellSize >= text.BitmapCharHeight {
		cellSize -= cellSize % text.BitmapCharHeight
	}
	// Keep the existing glyph run and the rounded frame corners at narrow
	// resolutions; use the full 50% widening wherever the border has room.
	oldWidth := (skyCells + 2) * cellSize
	cornerInset := max(1, (aperture.Min.Y-outer.Min.Y)/2)
	maxWidth := max(oldWidth, outer.Dx()-2*cornerInset)
	width, height := min(maxWidth, oldWidth*skyWidthNumerator/skyWidthDenominator), cellSize
	x := mapFrameCenteredX(outer, width)
	y := outer.Min.Y + (aperture.Min.Y-outer.Min.Y-height)/2

	return image.Rect(x, y, x+width, y+height)
}

func skyGlyphX(bounds image.Rectangle, cell int) float64 {
	pitch := float64(bounds.Dx()) / float64(skyCells+2)

	return float64(bounds.Min.X) + float64(cell+1)*pitch + (pitch-float64(bounds.Dy()))/2
}

func (g *GameScene) drawSkyStrip(screen *ebiten.Image) {
	g.syncSkyPresentation()
	if !g.sky.strip.visible || g.borders.skyBounds.Empty() {
		g.drawInheritedDungeonHeader(screen)

		return
	}
	bounds := g.borders.skyBounds
	backingBounds := g.borders.skyBackingBounds
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(backingBounds.Min.X), float64(backingBounds.Min.Y))
	screen.DrawImage(g.borders.skyBacking, op)
	scale := float64(bounds.Dy()) / text.BitmapCharHeight
	font := text.GetBitmapFonts().Runic
	for cell, glyph := range g.sky.strip.glyphs {
		if glyph == 0 {
			continue
		}
		colour := ucolor.EGALightGray
		if glyph == skySunGlyph {
			colour = ucolor.EGAYellow
		}
		red, green, blue := ucolor.ToRGBFloats(colour)
		font.DrawChar(screen, glyph, skyGlyphX(bounds, cell),
			float64(bounds.Min.Y), scale, red, green, blue)
	}
}

// ref:ASTRO-0002: the ordinary dungeon header is retained
// during camping; tactical dungeon rooms likewise must not gain a surface sky.
func (g *GameScene) drawInheritedDungeonHeader(screen *ebiten.Image) {
	if g.suppressSkyPresentation() || (!g.gameState.IsInCamping() && !g.gameState.IsInCombat()) {
		return
	}
	origin := g.skyOrigin()
	if origin.Location.GetMapType() != references.DungeonMapType {
		return
	}
	g.drawDungeonLevelLabel(screen, dungeonHUDStatus{floor: origin.Floor})
}

func (g *GameScene) initializeSkyBorder() {
	if g.borders.skyBacking != nil {
		g.borders.skyBacking.Deallocate()
		g.borders.skyBacking = nil
	}
	bounds := skyStripBounds(g.borders.outsideBorder.Bounds(), g.borders.surfaceAperture)
	g.borders.skyBounds = bounds
	g.borders.skyBackingBounds = image.Rectangle{}
	if bounds.Empty() {
		return
	}
	// Glyphs stay at their original height. The opaque backing fills the whole
	// top border band, hiding its lower white rule beneath the sky readout.
	backingBounds := image.Rect(bounds.Min.X, g.borders.outsideBorder.Bounds().Min.Y,
		bounds.Max.X, g.borders.surfaceAperture.Min.Y)
	g.borders.skyBackingBounds = backingBounds
	backing := ebiten.NewImage(backingBounds.Dx(), backingBounds.Dy())
	backing.Fill(ucolor.EGABlack)
	// Match the wind display's frame-coloured, white-edged inward bookends.
	stroke, blue := windBorderStyle(g.borders.outsideBorder.Bounds(), g.borders.surfaceAperture)
	drawMapBorderBookends(backing, stroke, blue)
	g.borders.skyBacking = backing
}
