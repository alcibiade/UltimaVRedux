package widgets

import (
	"image"
	stdcolor "image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	pkgcolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

// GemViewDialog displays an overhead map view using the gem/crystal ball viewing system.
// Uses static rendering - draws once to buffer, then blits buffer each frame.
// Maps terrain families to compact surface-view plot types; ref:VIEW-0002.
type GemViewDialog struct {
	// Display buffer (rendered once, reused every frame)
	buffer      *ebiten.Image
	flashBuffer *ebiten.Image
	drawOptions *ebiten.DrawImageOptions

	// Border and positioning
	border    *Border
	placement sprites.PercentBasedPlacement
	bounds    image.Rectangle
	pixelUnit int
	hint      staticViewerHint

	// State
	dismissed         bool
	waitingForRelease bool // Debouncing: wait for trigger keys to be released
	keyboard          *input.Keyboard

	// Map data for rendering (set on creation, used during initial render)
	playerWorldX int // Player's world X coordinate used for the marker
	playerWorldY int // Player's world Y coordinate used for the marker
	viewOriginX  int
	viewOriginY  int

	// Tile lookup function provided by caller
	getTileAt func(worldX, worldY int) int // Returns sprite index at world position

	wrapMap bool

	// Flash state for player position indicator
	flashVisible bool
}

// GemViewDialogConfig provides configuration for creating a gem view dialog.
type GemViewDialogConfig struct {
	Placement    sprites.PercentBasedPlacement
	Keyboard     *input.Keyboard
	PlayerWorldX int                          // Player's world X coordinate
	PlayerWorldY int                          // Player's world Y coordinate
	GetTileAt    func(worldX, worldY int) int // Function to get tile sprite index at world position
	WrapMap      bool                         // True for the wrapped 256x256 overworld
}

const (
	gemViewTileSize = 12 // Pixels per tile in gem view (12x12)
	gemViewGridSize = 32 // 32x32 tiles visible
	gemViewMapSize  = 256
	gemViewRiver    = 0x60
	gemViewRoad     = 0x20

	gemViewOriginalGridSize = 4
	gemViewHalfWindow       = gemViewGridSize / 4
	gemViewNibbleMask       = 0x0f
	gemViewHighNibbleMask   = 0xf0
	gemViewWindowSize       = 0x10

	gemViewNorthMask = 8
	gemViewEastMask  = 4
	gemViewSouthMask = 2
	gemViewWestMask  = 1
)

// Original masks for river quadrants and road connections; ref:VIEW-0002.
//
//nolint:gochecknoglobals // Fixed source data from the original game.
var (
	gemViewRiverMasks = [16]uint8{5, 6, 4, 1, 3, 6, 5, 7, 7, 14, 0, 0, 1, 2, 8, 4}
	gemViewRoadMasks  = [7]uint8{10, 5, 12, 6, 3, 9, 15}
)

// EGA color palette for the original surface View; ref:VIEW-0002.
// Using centralized EGA palette from pkg/color
//
//nolint:gochecknoglobals // EGA palette aliases
var (
	gemColorBlack  = stdcolor.Black         // EGA 0
	gemColorRed    = pkgcolor.EGARed        // EGA 4
	gemColorWhite  = stdcolor.White         // EGA 15
	gemColorYellow = pkgcolor.EGAYellow     // EGA 14
	gemColorLBlue  = pkgcolor.EGALightBlue  // EGA 9 (light blue)
	gemColorLGreen = pkgcolor.EGALightGreen // EGA 10 (light green)
)

// gemTile maps sprite indices to plot types (0-16).
// Original terrain-to-plot mapping; ref:VIEW-0002.
//
//nolint:gochecknoglobals // Data table from original game
var gemTile = [256]int{
	0, 12, 11, 10, 13, 1, 9, 3, 9, 2, 2, 8, 7, 6, 8, 8,
	5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 3, 4, 3, 3,
	16, 16, 16, 16, 16, 16, 16, 7, 7, 5, 5, 5, 9, 2, 5, 5,
	1, 1, 1, 1, 1, 1, 1, 1, 4, 7, 7, 7, 7, 7, 7, 7,
	3, 5, 5, 5, 3, 6, 7, 4, 3, 3, 6, 6, 5, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7, 5, 5, 4, 5, 4, 4, 5, 5,
	10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 3, 3, 10, 10, 10, 10,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	5, 5, 5, 5, 5, 5, 6, 3, 5, 5, 5, 5, 3, 5, 5, 3,
	5, 5, 5, 5, 4, 4, 4, 6, 6, 6, 4, 4, 4, 5, 5, 5,
	5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 3, 4, 4, 5, 5, 5,
	5, 5, 5, 5, 5, 5, 5, 5, 6, 6, 6, 6, 3, 5, 4, 5,
	0, 0, 0, 0, 6, 6, 6, 6, 5, 5, 5, 5, 0, 0, 0, 0,
	7, 7, 7, 7, 11, 11, 11, 11, 15, 15, 15, 15, 15, 3, 5, 5,
	14, 14, 14, 14, 10, 10, 10, 10, 5, 5, 5, 5, 6, 6, 6, 6,
	6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 5, 5, 5, 5, 7, 0,
}

// NewGemViewDialog creates a new gem view display dialog.
// The dialog renders once on creation (static), then simply blits the buffer on each Draw().
func NewGemViewDialog(config GemViewDialogConfig) *GemViewDialog {
	g := &GemViewDialog{
		placement:         config.Placement,
		keyboard:          config.Keyboard,
		playerWorldX:      config.PlayerWorldX,
		playerWorldY:      config.PlayerWorldY,
		viewOriginX:       gemViewOrigin(config.PlayerWorldX, config.WrapMap),
		viewOriginY:       gemViewOrigin(config.PlayerWorldY, config.WrapMap),
		getTileAt:         config.GetTileAt,
		wrapMap:           config.WrapMap,
		dismissed:         false,
		waitingForRelease: true, // Start waiting for trigger keys to be released
		drawOptions:       &ebiten.DrawImageOptions{},
	}

	// Initialize keyboard debouncing - wait for current keys to be released
	if config.Keyboard != nil {
		config.Keyboard.PrepareForTransition(KeyboardTransitionDelayMs)
	}

	// Create border with opaque black interior
	g.border = newDialogBorder(config.Placement, DialogBackgroundBlack)
	g.bounds = g.border.GetBounds()
	screenWidth, screenHeight := display.GetManager().GetCurrentSize()
	g.pixelUnit = getClassicUIPixelUnit(screenWidth, screenHeight)

	// Create buffer for static rendering
	g.buffer = ebiten.NewImage(g.bounds.Dx(), g.bounds.Dy())

	// Position draw options for the buffer
	g.drawOptions.GeoM.Translate(float64(g.bounds.Min.X), float64(g.bounds.Min.Y))

	// Render everything to buffer once
	g.renderToBuffer()
	g.renderFlashBuffer()

	return g
}

// renderToBuffer draws the gem view map to the internal buffer.
// This is called once during initialization - not per-frame.
func (g *GemViewDialog) renderToBuffer() {
	// Fill with solid black (opaque)
	vector.FillRect(g.buffer,
		0, 0,
		float32(g.bounds.Dx()), float32(g.bounds.Dy()),
		stdcolor.Black,
		false) // false = no anti-aliasing for pixel-perfect

	// Calculate scale factors to fit 32x32 tiles into dialog bounds
	// Original: 32 tiles * 12 pixels = 384x384
	// We scale to fit the dialog bounds
	tileWidth := float32(g.bounds.Dx()) / float32(gemViewGridSize)
	tileHeight := float32(g.bounds.Dy()) / float32(gemViewGridSize)

	// Render each tile in the 32x32 grid
	for ty := range gemViewGridSize {
		for tx := range gemViewGridSize {
			worldX, worldY := g.worldPosition(tx, ty)

			// Get tile sprite index at this world position
			tileIndex := 0
			if g.getTileAt != nil {
				tileIndex = g.getTileAt(worldX, worldY)
			}

			// Clamp to valid range
			if tileIndex < 0 || tileIndex >= 256 {
				tileIndex = 0
			}

			// Map the terrain byte to its compact plot type; ref:VIEW-0002.
			plotType := gemTile[tileIndex]

			// Calculate screen position for this tile
			screenX := float32(tx) * tileWidth
			screenY := float32(ty) * tileHeight

			// Render the plot type (pass tileIndex for directional roads)
			g.renderPlotType(screenX, screenY, tileWidth, tileHeight, plotType, tileIndex)
		}
	}
	applyClassicPixelRoundedMask(g.buffer, g.pixelUnit)
}

func (g *GemViewDialog) renderFlashBuffer() {
	g.flashBuffer = ebiten.NewImage(g.bounds.Dx(), g.bounds.Dy())
	g.flashBuffer.DrawImage(g.buffer, nil)
	g.drawPlayerFlash(g.flashBuffer, image.Pt(0, 0))
	applyClassicPixelRoundedMask(g.flashBuffer, g.pixelUnit)
}

// renderPlotType draws a single plot type at the given position.
// Original plot types 0-16 with scaled tile dimensions; ref:VIEW-0002.
// Original uses 4x4 pixel tiles; we scale proportionally.
// tileIndex is needed for directional tiles like roads (plotv16).
func (g *GemViewDialog) renderPlotType(x, y, width, height float32, plotType, tileIndex int) {
	switch plotType {
	case 0:
		// Nothing (black background already filled)
		return

	case 1:
		// plotv1: Light green 4 scattered pixels (forest/trees)
		// Place four dots at (1,0), (1,2), (3,1), and (3,3); ref:VIEW-0002.
		g.drawPixel(x, y, width, height, 1, 0, gemColorLGreen)
		g.drawPixel(x, y, width, height, 1, 2, gemColorLGreen)
		g.drawPixel(x, y, width, height, 3, 1, gemColorLGreen)
		g.drawPixel(x, y, width, height, 3, 3, gemColorLGreen)

	case 2:
		// plotv2: Light green solid fill (dense forest)
		g.drawFilledBox(x, y, width, height, gemColorLGreen)

	case 3:
		// RED solid fill (mountains/rocks)
		g.drawFilledBox(x, y, width, height, gemColorRed)

	case 4:
		// plotv4: WHITE 2 horizontal lines (bridges/structures)
		// Draw full top and bottom rows; ref:VIEW-0002.
		g.drawHLine(x, y, width, height, 0, gemColorWhite) // y=0
		g.drawHLine(x, y, width, height, 3, gemColorWhite) // y=3

	case 5:
		// plotv5: WHITE 2x2 center block (buildings/towns)
		// Fill the central 2x2 block; ref:VIEW-0002.
		g.drawHLineSegment(x, y, width, height, 1, 2, 1, gemColorWhite) // y=1, x=1-2
		g.drawHLineSegment(x, y, width, height, 1, 2, 2, gemColorWhite) // y=2, x=1-2

	case 6:
		// plotv6: WHITE hollow rectangle (walls/castles)
		// Original: top line, bottom line, left segment, right segment
		g.drawHLine(x, y, width, height, 0, gemColorWhite)                               // top
		g.drawHLine(x, y, width, height, gemViewOriginalGridSize-1, gemColorWhite)       // bottom
		g.drawMiddleVLine(x, y, width, height, 0, gemColorWhite)                         // left, y=1-2
		g.drawMiddleVLine(x, y, width, height, gemViewOriginalGridSize-1, gemColorWhite) // right, y=1-2

	case 7:
		// WHITE solid fill (towns/structures)
		g.drawFilledBox(x, y, width, height, gemColorWhite)

	case 8:
		// plotv8: YELLOW checkerboard pattern (swamp)
		// Original: 2x2 block at top-left, 2x2 block at bottom-right
		g.drawHLineSegment(x, y, width, height, 0, 1, 0, gemColorYellow) // y=0, x=0-1
		g.drawHLineSegment(x, y, width, height, 0, 1, 1, gemColorYellow) // y=1, x=0-1
		g.drawHLineSegment(x, y, width, height, 2, 3, 2, gemColorYellow) // y=2, x=2-3
		g.drawHLineSegment(x, y, width, height, 2, 3, 3, gemColorYellow) // y=3, x=2-3

	case 9:
		// plotv9: Light green pattern (brush/scrub)
		// Original: 2 horizontal lines + 2 pixels
		g.drawHLine(x, y, width, height, 0, gemColorLGreen)    // y=0
		g.drawHLine(x, y, width, height, 2, gemColorLGreen)    // y=2
		g.drawPixel(x, y, width, height, 2, 1, gemColorLGreen) // pixel at (2,1)
		g.drawPixel(x, y, width, height, 0, 3, gemColorLGreen) // pixel at (0,3)

	case 10:
		// plotv10: River direction masks retain green banks around blue water.
		waterMask := gemViewRiverWaterMask(tileIndex)
		points := [4][2]int{{1, 0}, {3, 1}, {1, 2}, {3, 3}}
		for i, point := range points {
			c := gemColorLGreen
			if waterMask&(gemViewNorthMask>>i) != 0 {
				c = gemColorLBlue
			}
			g.drawPixel(x, y, width, height, point[0], point[1], c)
		}

	case 11:
		// plotv11: Light blue 2 diagonal pixels (shallow water)
		// Place dots at (0,0) and (2,2); ref:VIEW-0002.
		g.drawPixel(x, y, width, height, 0, 0, gemColorLBlue)
		g.drawPixel(x, y, width, height, 2, 2, gemColorLBlue)

	case 12:
		// Light blue single pixel (water feature)
		// Place a dot at (2,2); ref:VIEW-0002.
		g.drawPixel(x, y, width, height, 2, 2, gemColorLBlue)

	case 13:
		// plotv13: Mixed light green + light blue pattern (coast/shore)
		// Original: green at (1,0), (3,1); blue at (0,2), (2,3)
		g.drawPixel(x, y, width, height, 1, 0, gemColorLGreen)
		g.drawPixel(x, y, width, height, 3, 1, gemColorLGreen)
		g.drawPixel(x, y, width, height, 0, 2, gemColorLBlue)
		g.drawPixel(x, y, width, height, 2, 3, gemColorLBlue)

	case 14:
		// WHITE 2 vertical lines (ladder/passage)
		// Fill the middle two columns; ref:VIEW-0002.
		g.drawVLine(x, y, width, height, 1, gemColorWhite) // x=1
		g.drawVLine(x, y, width, height, 2, gemColorWhite) // x=2

	case 15:
		// Light blue solid fill (ocean/deep water)
		g.drawFilledBox(x, y, width, height, gemColorLBlue)

	case 16:
		// plotv16: Road pattern with directional connections
		// plotv1 background (light green scattered pixels)
		g.drawPixel(x, y, width, height, 1, 0, gemColorLGreen)
		g.drawPixel(x, y, width, height, 1, 2, gemColorLGreen)
		g.drawPixel(x, y, width, height, 3, 1, gemColorLGreen)
		g.drawPixel(x, y, width, height, 3, 3, gemColorLGreen)

		// Red center 2x2 block
		g.drawHLineSegment(x, y, width, height, 1, 2, 1, gemColorRed)
		g.drawHLineSegment(x, y, width, height, 1, 2, 2, gemColorRed)

		// Read road-connection bits relative to base tile 0x20; ref:VIEW-0002.
		g.drawRoadConnections(x, y, width, height, tileIndex)
	}
}

//nolint:mnd,varnamelen // Original 4x4 plot coordinates and x/y drawing origins are idiomatic here; ref:VIEW-0002.
func (g *GemViewDialog) drawRoadConnections(x, y, width, height float32, tileIndex int) {
	directionMask, ok := gemViewRoadConnectionMask(tileIndex)
	if !ok {
		return
	}
	if directionMask&gemViewNorthMask != 0 {
		g.drawHLineSegment(x, y, width, height, 1, 2, 0, gemColorRed)
	}
	if directionMask&gemViewEastMask != 0 {
		g.drawMiddleVLine(x, y, width, height, gemViewOriginalGridSize-1, gemColorRed)
	}
	if directionMask&gemViewSouthMask != 0 {
		g.drawHLineSegment(x, y, width, height, 1, 2, 3, gemColorRed)
	}
	if directionMask&gemViewWestMask != 0 {
		g.drawMiddleVLine(x, y, width, height, 0, gemColorRed)
	}
	if cutoutX, cutoutY, hasCutout := gemViewRoadCutout(tileIndex); hasCutout {
		g.drawPixel(x, y, width, height, cutoutX, cutoutY, gemColorBlack)
	}
}

func gemViewOrigin(player int, wrapMap bool) int {
	if !wrapMap {
		return 0
	}
	origin := player & gemViewHighNibbleMask
	if player&gemViewNibbleMask < gemViewHalfWindow {
		origin = (origin - gemViewWindowSize) & (gemViewMapSize - 1)
	}

	return origin
}

func (g *GemViewDialog) worldPosition(tileX, tileY int) (int, int) {
	worldX := g.viewOriginX + tileX
	worldY := g.viewOriginY + tileY
	if g.wrapMap {
		worldX &= gemViewMapSize - 1
		worldY &= gemViewMapSize - 1
	}

	return worldX, worldY
}

func (g *GemViewDialog) markerGridPosition() (int, int, bool) {
	gridX := g.playerWorldX - g.viewOriginX
	gridY := g.playerWorldY - g.viewOriginY
	if g.wrapMap {
		gridX &= gemViewMapSize - 1
		gridY &= gemViewMapSize - 1
	}

	return gridX, gridY, gridX >= 0 && gridX < gemViewGridSize && gridY >= 0 && gridY < gemViewGridSize
}

func gemViewRiverWaterMask(tile int) uint8 {
	if tile&gemViewHighNibbleMask != gemViewRiver {
		return gemViewNibbleMask
	}

	return gemViewRiverMasks[tile&gemViewNibbleMask]
}

func gemViewRoadConnectionMask(tile int) (uint8, bool) {
	index := tile - gemViewRoad
	if index < 0 || index >= len(gemViewRoadMasks) {
		return 0, false
	}

	return gemViewRoadMasks[index], true
}

//nolint:mnd // Original road codes and 4x4 cutout coordinates are clearer as a direct table; ref:VIEW-0002.
func gemViewRoadCutout(tile int) (int, int, bool) {
	switch tile {
	case 0x22:
		return 1, 2, true
	case 0x23:
		return 1, 1, true
	case 0x24:
		return 2, 1, true
	case 0x25:
		return 2, 2, true
	default:
		return 0, 0, false
	}
}

// Drawing helpers that scale 4x4 original pixel coordinates to our tile size.
// Original surface View uses 4x4 pixel tiles; we scale proportionally; ref:VIEW-0002.

// drawFilledBox fills a tile with a solid color.
func (g *GemViewDialog) drawFilledBox(x, y, width, height float32, c stdcolor.Color) {
	vector.FillRect(g.buffer, x, y, width, height, c, false)
}

// drawPixel draws a single pixel at the given 4x4 grid position, scaled to tile size.
// px, py are coordinates in the original 4x4 tile (0-3).
func (g *GemViewDialog) drawPixel(x, y, width, height float32, px, py int, c stdcolor.Color) {
	// Scale the pixel position and size
	pixelW := width / gemViewOriginalGridSize
	pixelH := height / gemViewOriginalGridSize

	screenX := x + float32(px)*pixelW
	screenY := y + float32(py)*pixelH

	vector.FillRect(g.buffer, screenX, screenY, pixelW, pixelH, c, false)
}

// drawHLine draws a full-width horizontal line at the given y position (0-3).
func (g *GemViewDialog) drawHLine(x, y, width, height float32, py int, c stdcolor.Color) {
	lineH := height / gemViewOriginalGridSize
	screenY := y + float32(py)*lineH

	vector.FillRect(g.buffer, x, screenY, width, lineH, c, false)
}

// drawVLine draws a full-height vertical line at the given x position (0-3).
func (g *GemViewDialog) drawVLine(x, y, width, height float32, px int, c stdcolor.Color) {
	lineW := width / gemViewOriginalGridSize
	screenX := x + float32(px)*lineW

	vector.FillRect(g.buffer, screenX, y, lineW, height, c, false)
}

// drawHLineSegment draws a horizontal line segment from x1 to x2 (inclusive) at y position py.
// x1, x2, py are coordinates in the original 4x4 tile (0-3).
func (g *GemViewDialog) drawHLineSegment(x, y, width, height float32, x1, x2, py int, c stdcolor.Color) {
	pixelW := width / gemViewOriginalGridSize
	pixelH := height / gemViewOriginalGridSize

	screenX := x + float32(x1)*pixelW
	screenY := y + float32(py)*pixelH
	segmentW := float32(x2-x1+1) * pixelW

	vector.FillRect(g.buffer, screenX, screenY, segmentW, pixelH, c, false)
}

// drawMiddleVLine draws the middle two vertical pixels of the original 4x4 tile at the given column.
func (g *GemViewDialog) drawMiddleVLine(originX, originY, width, height float32, column int, lineColor stdcolor.Color) {
	const segmentPixelCount = 2
	pixelW := width / gemViewOriginalGridSize
	pixelH := height / gemViewOriginalGridSize

	screenX := originX + float32(column)*pixelW
	screenY := originY + pixelH
	segmentH := segmentPixelCount * pixelH

	vector.FillRect(g.buffer, screenX, screenY, pixelW, segmentH, lineColor, false)
}

// Draw implements Widget interface.
// Simply blits the pre-rendered buffer - no per-frame rendering.
func (g *GemViewDialog) Draw(screen *ebiten.Image) {
	buffer := g.buffer
	if g.flashVisible {
		buffer = g.flashBuffer
	}
	screen.DrawImage(buffer, g.drawOptions)

	DrawClassicPixelRoundedFrame(screen, g.bounds, g.pixelUnit, DialogRuleColor)
	g.hint.draw(screen, g.bounds, "Any key: Close")
}

// drawPlayerFlash draws a white rectangle at the player's position.
// Highlights the player position with an inverted box; ref:VIEW-0002.
func (g *GemViewDialog) drawPlayerFlash(screen *ebiten.Image, origin image.Point) {
	flashTileX, flashTileY, visible := g.markerGridPosition()
	if !visible {
		return
	}

	// Calculate tile dimensions
	tileWidth := float32(g.bounds.Dx()) / float32(gemViewGridSize)
	tileHeight := float32(g.bounds.Dy()) / float32(gemViewGridSize)

	// Calculate screen position for player tile
	screenX := float32(origin.X) + float32(flashTileX)*tileWidth
	screenY := float32(origin.Y) + float32(flashTileY)*tileHeight

	// Draw white filled rectangle at player position
	vector.FillRect(screen, screenX, screenY, tileWidth, tileHeight, gemColorWhite, false)
}

// Update implements Widget interface.
// Handles keyboard input for dismissal with proper debouncing.
func (g *GemViewDialog) Update() {
	if g.keyboard == nil {
		return
	}

	// Phase 1: Wait for trigger keys to be released
	if g.waitingForRelease {
		if g.keyboard.IsWaitingForRelease() {
			// Still waiting for initial keys to be released - poll keyboard state
			g.keyboard.TryAcceptKey(0) // Trigger state update check

			return
		}
		// Keys released - transition to accepting new input
		g.waitingForRelease = false
	}

	// Phase 2: Accept any new keypress to dismiss
	if g.keyboard.IsAnyKeyPressed() {
		g.keyboard.ResetReleaseDebouncing()
		g.dismissed = true
	}
}

// IsDismissed returns true if the dialog has been dismissed by user input.
func (g *GemViewDialog) IsDismissed() bool {
	return g.dismissed
}

// Dismiss closes the dialog through its normal workflow lifecycle.
func (g *GemViewDialog) Dismiss() {
	g.dismissed = true
}

// ToggleFlash toggles the flash visibility state for the player position indicator.
// Called by event queue at regular intervals to create flashing effect.
func (g *GemViewDialog) ToggleFlash() {
	g.flashVisible = !g.flashVisible
}
