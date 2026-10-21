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
	// Initialize keyboard debouncing - wait for current keys to be released
	if config.Keyboard != nil {
		config.Keyboard.PrepareForTransition(KeyboardTransitionDelayMs)
	}

	// Create border with opaque black interior
	border := newDialogBorder(config.Placement)
	bounds := border.GetBounds()
	screenWidth, screenHeight := display.GetManager().GetCurrentSize()

	dialog := &GemViewDialog{
		buffer:            ebiten.NewImage(bounds.Dx(), bounds.Dy()), // Static render target
		flashBuffer:       nil,                                       // Rendered after buffer
		drawOptions:       &ebiten.DrawImageOptions{},
		border:            border,
		placement:         config.Placement,
		bounds:            bounds,
		pixelUnit:         getClassicUIPixelUnit(screenWidth, screenHeight),
		hint:              staticViewerHint{output: nil, width: 0},
		dismissed:         false,
		waitingForRelease: true, // Start waiting for trigger keys to be released
		keyboard:          config.Keyboard,
		playerWorldX:      config.PlayerWorldX,
		playerWorldY:      config.PlayerWorldY,
		viewOriginX:       gemViewOrigin(config.PlayerWorldX, config.WrapMap),
		viewOriginY:       gemViewOrigin(config.PlayerWorldY, config.WrapMap),
		getTileAt:         config.GetTileAt,
		wrapMap:           config.WrapMap,
		flashVisible:      false,
	}

	// Position draw options for the buffer
	dialog.drawOptions.GeoM.Translate(float64(bounds.Min.X), float64(bounds.Min.Y))

	// Render everything to buffer once
	dialog.renderToBuffer()
	dialog.renderFlashBuffer()

	return dialog
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
	layers := gemPlotLayerTable()
	for tileY := range gemViewGridSize {
		for tileX := range gemViewGridSize {
			worldX, worldY := g.worldPosition(tileX, tileY)

			// Get tile sprite index at this world position
			tileIndex := 0
			if g.getTileAt != nil {
				tileIndex = g.getTileAt(worldX, worldY)
			}

			// Clamp to valid range
			if tileIndex < 0 || tileIndex >= len(gemTile) {
				tileIndex = 0
			}

			// Map the terrain byte to its compact plot type; ref:VIEW-0002.
			plotType := gemTile[tileIndex]

			// Render the plot type (pass tileIndex for directional rivers and roads)
			tile := gemPlotTile{
				originX: float32(tileX) * tileWidth,
				originY: float32(tileY) * tileHeight,
				width:   tileWidth,
				height:  tileHeight,
			}
			g.renderPlotType(tile, layers[plotType], plotType, tileIndex)
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

// Plot types with tile-dependent artwork; every other type draws its fixed
// gemPlotLayerTable pattern. Original plot numbering; ref:VIEW-0002.
const (
	gemPlotRiver     = 10
	gemPlotRoad      = 16
	gemPlotTypeCount = 17
)

// gemPlotRect is a filled rectangle measured in cells of the original 4x4 plot.
type gemPlotRect struct {
	column, row, columns, rows int
}

// gemPlotLayer fills a set of plot rectangles with one EGA colour.
type gemPlotLayer struct {
	color stdcolor.Color
	rects []gemPlotRect
}

// gemPlotLayerTable returns the fixed artwork of plot types 0-16 in original
// 4x4 cells; ref:VIEW-0002. Plot 0 is the black background. River plots use
// gemViewRiverWaterMask instead, and roads add gemViewRoadConnectionMask.
func gemPlotLayerTable() [gemPlotTypeCount][]gemPlotLayer {
	forestDots := []gemPlotRect{{1, 0, 1, 1}, {1, 2, 1, 1}, {3, 1, 1, 1}, {3, 3, 1, 1}}
	solid := []gemPlotRect{{0, 0, 4, 4}}
	centerBlock := []gemPlotRect{{1, 1, 2, 1}, {1, 2, 2, 1}}

	return [gemPlotTypeCount][]gemPlotLayer{
		1: {{gemColorLGreen, forestDots}},                               // plotv1: forest/trees
		2: {{gemColorLGreen, solid}},                                    // plotv2: dense forest
		3: {{gemColorRed, solid}},                                       // mountains/rocks
		4: {{gemColorWhite, []gemPlotRect{{0, 0, 4, 1}, {0, 3, 4, 1}}}}, // plotv4: bridges
		5: {{gemColorWhite, centerBlock}},                               // plotv5: buildings/towns
		6: {{gemColorWhite, []gemPlotRect{ // plotv6: hollow walls/castles
			{0, 0, 4, 1}, {0, 3, 4, 1}, {0, 1, 1, 2}, {3, 1, 1, 2}}}},
		7: {{gemColorWhite, solid}}, // towns/structures
		8: {{gemColorYellow, []gemPlotRect{ // plotv8: swamp checkerboard
			{0, 0, 2, 1}, {0, 1, 2, 1}, {2, 2, 2, 1}, {2, 3, 2, 1}}}},
		9: {{gemColorLGreen, []gemPlotRect{ // plotv9: brush/scrub
			{0, 0, 4, 1}, {0, 2, 4, 1}, {2, 1, 1, 1}, {0, 3, 1, 1}}}},
		11: {{gemColorLBlue, []gemPlotRect{{0, 0, 1, 1}, {2, 2, 1, 1}}}}, // plotv11: shallow water
		12: {{gemColorLBlue, []gemPlotRect{{2, 2, 1, 1}}}},               // water feature
		13: { // plotv13: coast/shore
			{gemColorLGreen, []gemPlotRect{{1, 0, 1, 1}, {3, 1, 1, 1}}},
			{gemColorLBlue, []gemPlotRect{{0, 2, 1, 1}, {2, 3, 1, 1}}},
		},
		14: {{gemColorWhite, []gemPlotRect{{1, 0, 1, 4}, {2, 0, 1, 4}}}}, // ladder/passage
		15: {{gemColorLBlue, solid}},                                     // ocean/deep water
		16: {{gemColorLGreen, forestDots}, {gemColorRed, centerBlock}},   // plotv16: road centre
	}
}

// renderPlotType draws one plot type, scaled from the original 4x4 cells to the
// tile size; ref:VIEW-0002. tileIndex selects river and road directions.
func (g *GemViewDialog) renderPlotType(tile gemPlotTile, layers []gemPlotLayer, plotType, tileIndex int) {
	if plotType == gemPlotRiver {
		g.drawRiverPlot(tile, tileIndex)

		return
	}
	for _, layer := range layers {
		for _, rect := range layer.rects {
			g.drawPlotRect(tile, rect, layer.color)
		}
	}
	if plotType == gemPlotRoad {
		g.drawRoadConnections(tile, tileIndex)
	}
}

// drawRiverPlot keeps green banks around the blue water quadrants of plotv10.
func (g *GemViewDialog) drawRiverPlot(tile gemPlotTile, tileIndex int) {
	waterMask := gemViewRiverWaterMask(tileIndex)
	quadrants := [4]gemPlotRect{{1, 0, 1, 1}, {3, 1, 1, 1}, {1, 2, 1, 1}, {3, 3, 1, 1}}
	for quadrant, rect := range quadrants {
		quadrantColor := gemColorLGreen
		if waterMask&(gemViewNorthMask>>quadrant) != 0 {
			quadrantColor = gemColorLBlue
		}
		g.drawPlotRect(tile, rect, quadrantColor)
	}
}

// drawRoadConnections reads road-connection bits relative to base tile 0x20; ref:VIEW-0002.
func (g *GemViewDialog) drawRoadConnections(tile gemPlotTile, tileIndex int) {
	directionMask, ok := gemViewRoadConnectionMask(tileIndex)
	if !ok {
		return
	}
	connections := [4]struct {
		mask uint8
		rect gemPlotRect
	}{
		{gemViewNorthMask, gemPlotRect{1, 0, 2, 1}},
		{gemViewEastMask, gemPlotRect{3, 1, 1, 2}},
		{gemViewSouthMask, gemPlotRect{1, 3, 2, 1}},
		{gemViewWestMask, gemPlotRect{0, 1, 1, 2}},
	}
	for _, connection := range connections {
		if directionMask&connection.mask != 0 {
			g.drawPlotRect(tile, connection.rect, gemColorRed)
		}
	}
	if cutoutX, cutoutY, hasCutout := gemViewRoadCutout(tileIndex); hasCutout {
		g.drawPlotRect(tile, gemPlotRect{cutoutX, cutoutY, 1, 1}, gemColorBlack)
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

// gemPlotTile is the screen rectangle of one 4x4 original plot.
type gemPlotTile struct {
	originX, originY, width, height float32
}

// drawPlotRect fills rect, given in original 4x4 cells, scaled to the tile size.
func (g *GemViewDialog) drawPlotRect(tile gemPlotTile, rect gemPlotRect, fill stdcolor.Color) {
	cellW := tile.width / gemViewOriginalGridSize
	cellH := tile.height / gemViewOriginalGridSize
	vector.FillRect(g.buffer,
		tile.originX+float32(rect.column)*cellW, tile.originY+float32(rect.row)*cellH,
		float32(rect.columns)*cellW, float32(rect.rows)*cellH, fill, false)
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
	g.hint.draw(screen, g.bounds, ControlHint{"Any key", hintActionClose})
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
