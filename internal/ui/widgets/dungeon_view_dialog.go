package widgets

import (
	"image"
	stdcolor "image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	utext "github.com/bradhannah/UltimaVRedux/internal/text"
	pkgcolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

const (
	dungeonViewFloorCells     = 8
	dungeonViewCellSize       = 16
	dungeonViewMarkerGutter   = 8
	dungeonViewRightMarkerPad = 1
	dungeonViewGridOrigin     = dungeonViewMarkerGutter
	dungeonViewNativeSize     = dungeonViewMarkerGutter*2 + dungeonViewFloorCells*dungeonViewCellSize +
		dungeonViewRightMarkerPad
	dungeonViewLegendIconSize    = 12
	dungeonViewWideGap           = 2
	dungeonViewNarrowGap         = 8
	dungeonViewCompositionGap    = 12
	dungeonViewWideHeaderHeight  = 20
	dungeonViewWideFooterHeight  = 14
	dungeonViewNarrowHeaderMin   = 48
	dungeonViewNarrowFooterMin   = 48
	dungeonViewLegendMin         = 150
	dungeonViewLegendLabelMin    = 200
	dungeonViewLegendLabelWidth  = 240
	dungeonViewLegendColumnGap   = 12
	dungeonViewNarrowColumnGap   = 4
	dungeonViewMinimumTextSize   = 10
	dungeonViewMinimumTextSpace  = 11
	dungeonViewNarrowFirstColumn = 43
	dungeonViewReferenceWidth    = 1920
	dungeonViewFloorDim          = 0.72
	dungeonViewPlainWall         = 0xb0
	dungeonViewHorizontalRuneA   = int('A')
	dungeonViewVerticalRuneI     = int('I')
	dungeonViewLegendColumnCount = 2
	dungeonViewLegendRowCount    = 9
)

type dungeonViewSegment struct {
	fromX float32
	fromY float32
	toX   float32
	toY   float32
}

type dungeonViewSurface uint8

const (
	dungeonViewSurfaceNone dungeonViewSurface = iota
	dungeonViewSurfaceFloor
	dungeonViewSurfaceWall
)

type dungeonViewIcon uint8

const (
	dungeonViewIconNone dungeonViewIcon = iota
	dungeonViewIconCeilingPit
	dungeonViewIconLadderUp
	dungeonViewIconLadderDown
	dungeonViewIconLadderBoth
	dungeonViewIconChest
	dungeonViewIconFountain
	dungeonViewIconOpenPit
	dungeonViewIconHiddenPit
	dungeonViewIconPitCeiling
	dungeonViewIconTrap
	dungeonViewIconField
	dungeonViewIconRoom
	dungeonViewIconPlaque
	dungeonViewIconSpecialWall
	dungeonViewIconHiddenDoor
	dungeonViewIconDoor
)

type dungeonViewAxis uint8

const (
	dungeonViewAxisNone dungeonViewAxis = iota
	dungeonViewAxisHorizontal
	dungeonViewAxisVertical
)

type dungeonViewTopology uint8

const (
	dungeonViewNorth dungeonViewTopology = 1 << iota
	dungeonViewEast
	dungeonViewSouth
	dungeonViewWest
)

type dungeonViewCellPlan struct {
	visible  bool
	player   bool
	surface  dungeonViewSurface
	icon     dungeonViewIcon
	topology dungeonViewTopology
	doorAxis dungeonViewAxis
}

type dungeonViewKnownEdges struct {
	east  [dungeonViewFloorCells][dungeonViewFloorCells]bool
	south [dungeonViewFloorCells][dungeonViewFloorCells]bool
}

type dungeonViewRenderPlan struct {
	cells [dungeonViewFloorCells][dungeonViewFloorCells]dungeonViewCellPlan
	edges dungeonViewKnownEdges
}

type dungeonViewLayout struct {
	header      image.Rectangle
	mapArea     image.Rectangle
	legend      image.Rectangle
	footer      image.Rectangle
	legendBelow bool
	mapScale    int
}

type dungeonViewLegendEntry struct {
	label        string
	narrowLabel  string
	cell         references.DungeonCell
	player       bool
	continuation bool
}

//nolint:gochecknoglobals,exhaustruct,mnd // Immutable presentation key; omitted fields use zero values.
var dungeonViewLegend = [...]dungeonViewLegendEntry{
	{label: "You", player: true},
	{label: "Passage", cell: 0x00},
	{label: "Wall", cell: 0xb0},
	{label: "Wall writing", narrowLabel: "Writing", cell: 0xb1},
	{label: "Special wall", narrowLabel: "Special", cell: 0xc0},
	{label: "Door", cell: 0xe0},
	{label: "Secret door", narrowLabel: "Secret", cell: 0xd0},
	{label: "Room", cell: 0xa0},
	{label: "Ladder up", narrowLabel: "Up", cell: 0x10},
	{label: "Ladder down", cell: 0x20},
	{label: "Ladder both", narrowLabel: "Both", cell: 0x30},
	{label: "Chest", cell: 0x4f},
	{label: "Fountain", cell: 0x50},
	{label: "Pit / hole", cell: 0x60},
	{label: "Hidden pit", cell: 0x61},
	{label: "Trap / bomb", cell: 0x62},
	{label: "Energy field", narrowLabel: "Field", cell: 0x80},
	{label: "Continuation", continuation: true},
}

// DungeonViewDialog renders the source-faithful dungeon projection as a responsive vector floor plan.
type DungeonViewDialog struct {
	model        game_state.DungeonViewModel
	keyboard     *input.Keyboard
	placement    sprites.PercentBasedPlacement
	border       *Border
	buffer       *ebiten.Image
	drawOptions  *ebiten.DrawImageOptions
	bounds       image.Rectangle
	layout       dungeonViewLayout
	screenWidth  int
	screenHeight int

	titleOutput   *utext.Output
	legendOutputs [2]*utext.Output
	footerOutput  *utext.Output

	dismissed         bool
	waitingForRelease bool
}

// DungeonViewDialogConfig contains immutable state for one symbolic dungeon view.
type DungeonViewDialogConfig struct {
	Placement sprites.PercentBasedPlacement
	Keyboard  *input.Keyboard
	Model     game_state.DungeonViewModel
}

// NewDungeonViewDialog creates a static symbolic map with responsive framing and legend placement.
//
// REDUX DEVIATION: Original dungeon View draws IBM/runic cell glyphs across display-space copies (ref:VIEW-0003). Redux
// collapses visible occurrences into a canonical north-up atlas with connected vector topology,
// geometric EGA icons, matching runic edge marks, a fixed legend, and a dismissal hint.
//
//nolint:exhaustruct // Rendering fields are initialized together by rebuildLayout.
func NewDungeonViewDialog(config DungeonViewDialogConfig) *DungeonViewDialog {
	dialog := &DungeonViewDialog{
		model:             config.Model,
		keyboard:          config.Keyboard,
		placement:         config.Placement,
		dismissed:         false,
		waitingForRelease: true,
	}
	if dialog.keyboard != nil {
		dialog.keyboard.PrepareForTransition(0)
	}
	screenWidth, screenHeight := display.GetManager().GetCurrentSize()
	dialog.rebuildLayout(screenWidth, screenHeight)

	return dialog
}

//nolint:exhaustruct,mnd // Widget option zero values and compact typography ratios are intentional.
func (d *DungeonViewDialog) rebuildLayout(screenWidth, screenHeight int) {
	if d.border != nil && screenWidth == d.screenWidth && screenHeight == d.screenHeight {
		return
	}
	d.screenWidth = screenWidth
	d.screenHeight = screenHeight
	d.border = newDialogBorder(d.placement, DialogBackgroundBlack)
	d.drawOptions = &ebiten.DrawImageOptions{}
	d.bounds = d.border.GetBounds()
	d.layout = calculateDungeonViewLayout(localDungeonViewBounds(d.border), screenWidth, screenHeight)
	d.buffer = ebiten.NewImage(d.bounds.Dx(), d.bounds.Dy())
	d.drawOptions.GeoM.Translate(float64(d.bounds.Min.X), float64(d.bounds.Min.Y))

	d.titleOutput = newDungeonViewOutput(FontConfig{
		Size: FontSizeLarge, LineSpacingMultiplier: 1.1, MaxLines: 2, MaxChars: 90,
	})
	for index := range d.legendOutputs {
		d.legendOutputs[index] = newDungeonViewOutputWithMinimum(
			FontConfig{
				Size: FontSizeLarge, LineSpacingMultiplier: 1.1, MaxLines: len(dungeonViewLegend) + 1, MaxChars: 30,
			},
			dungeonViewMinimumTextSize,
			dungeonViewMinimumTextSpace,
		)
	}
	d.footerOutput = newDungeonViewOutput(FontConfig{
		Size: FontSizeLarge, LineSpacingMultiplier: 1.1, MaxLines: 2, MaxChars: 90,
	})

	d.renderToBuffer()
}

func newDungeonViewOutput(config FontConfig) *utext.Output {
	return newDungeonViewOutputWithMinimum(
		config,
		dungeonViewMinimumTextSize,
		dungeonViewMinimumTextSpace,
	)
}

func newDungeonViewOutputWithMinimum(config FontConfig, minimumSize, minimumSpacing float64) *utext.Output {
	_, output := GetScaledFontWithOutput(config)
	if output.Font.TextFace.Size >= minimumSize {
		return output
	}
	font := utext.NewUltimaFont(minimumSize)
	output.SetFont(font, max(output.GetLineSpacing(), minimumSpacing))

	return output
}

func localDungeonViewBounds(border *Border) image.Rectangle {
	outer := border.GetBounds()
	interior := border.GetInteriorBounds()

	return interior.Sub(outer.Min)
}

//nolint:exhaustruct,mnd // False is the landscape zero value; halves center fixed rectangles.
func calculateDungeonViewLayout(bounds image.Rectangle, screenWidth, screenHeight int) dungeonViewLayout {
	landscape := bounds.Dx() >= bounds.Dy()+dungeonViewLegendMin
	headerHeight, footerHeight, gap := dungeonViewNarrowHeaderMin, dungeonViewNarrowFooterMin, dungeonViewNarrowGap
	if landscape {
		textHeight := dungeonViewWideTextHeight(screenWidth)
		headerHeight = max(dungeonViewWideHeaderHeight, textHeight)
		footerHeight = max(dungeonViewWideFooterHeight, textHeight)
		gap = dungeonViewWideGap
	}
	header := image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Min.Y+headerHeight)
	footer := image.Rect(bounds.Min.X, bounds.Max.Y-footerHeight, bounds.Max.X, bounds.Max.Y)
	body := image.Rect(bounds.Min.X, header.Max.Y+gap, bounds.Max.X, footer.Min.Y-gap)

	if landscape {
		mapScale := dungeonViewMapScaleForArea(body, screenWidth, screenHeight)
		for mapScale > 1 && dungeonViewWideGroupWidth(mapScale, screenWidth) > body.Dx() {
			mapScale--
		}
		mapSize := dungeonViewNativeSize * mapScale
		legendWidth := dungeonViewWideLegendWidth(mapScale, screenWidth)
		legendHeight := dungeonViewLegendRowCount * dungeonViewLegendIconSize * mapScale
		groupWidth := mapSize + dungeonViewCompositionGap + legendWidth
		groupX := body.Min.X + (body.Dx()-groupWidth)/2
		groupY := body.Min.Y + (body.Dy()-mapSize)/2
		mapArea := image.Rect(groupX, groupY, groupX+mapSize, groupY+mapSize)
		legend := image.Rect(
			mapArea.Max.X+dungeonViewCompositionGap,
			mapArea.Min.Y+(mapSize-legendHeight)/2,
			mapArea.Max.X+dungeonViewCompositionGap+legendWidth,
			mapArea.Min.Y+(mapSize+legendHeight)/2,
		)

		return dungeonViewLayout{
			header: header, mapArea: mapArea, legend: legend, footer: footer, mapScale: mapScale,
		}
	}

	mapScale := dungeonViewMapScaleForArea(body, screenWidth, screenHeight)
	for mapScale > 1 && dungeonViewNarrowGroupHeight(mapScale) > body.Dy() {
		mapScale--
	}
	mapSize := dungeonViewNativeSize * mapScale
	legendHeight := dungeonViewLegendRowCount * dungeonViewLegendIconSize * mapScale
	groupHeight := mapSize + dungeonViewNarrowGap + legendHeight
	groupY := body.Min.Y + (body.Dy()-groupHeight)/2
	mapArea := image.Rect(
		body.Min.X+(body.Dx()-mapSize)/2,
		groupY,
		body.Min.X+(body.Dx()+mapSize)/2,
		groupY+mapSize,
	)
	legend := image.Rect(
		body.Min.X,
		mapArea.Max.Y+dungeonViewNarrowGap,
		body.Max.X,
		mapArea.Max.Y+dungeonViewNarrowGap+legendHeight,
	)

	return dungeonViewLayout{
		header: header, mapArea: mapArea, legend: legend, footer: footer, legendBelow: true, mapScale: mapScale,
	}
}

func dungeonViewWideTextHeight(screenWidth int) int {
	scaledLineSpacing := float64(FontSizeLarge) * 1.1 *
		float64(max(1, screenWidth)) / dungeonViewReferenceWidth

	return int(math.Ceil(scaledLineSpacing))
}

func dungeonViewMapScaleForArea(area image.Rectangle, screenWidth, screenHeight int) int {
	availableScale := min(area.Dx()/dungeonViewNativeSize, area.Dy()/dungeonViewNativeSize)

	return max(1, min(getClassicUIPixelUnit(screenWidth, screenHeight), availableScale))
}

func dungeonViewWideLegendWidth(mapScale, screenWidth int) int {
	physicalIconSize := dungeonViewLegendIconSize * mapScale
	labelWidth := max(
		dungeonViewLegendLabelMin,
		(dungeonViewLegendLabelWidth*screenWidth+dungeonViewReferenceWidth/2)/dungeonViewReferenceWidth,
	)

	return dungeonViewLegendColumnCount*(physicalIconSize+labelWidth) + dungeonViewLegendColumnGap
}

func dungeonViewWideGroupWidth(mapScale, screenWidth int) int {
	return dungeonViewNativeSize*mapScale + dungeonViewCompositionGap +
		dungeonViewWideLegendWidth(mapScale, screenWidth)
}

func dungeonViewNarrowGroupHeight(mapScale int) int {
	return (dungeonViewNativeSize+dungeonViewLegendRowCount*dungeonViewLegendIconSize)*mapScale +
		dungeonViewNarrowGap
}

func (d *DungeonViewDialog) renderToBuffer() {
	FillClassicPixelRoundedRect(
		d.buffer,
		d.buffer.Bounds(),
		getClassicUIPixelUnit(d.screenWidth, d.screenHeight),
		pkgcolor.UIBackground.RGBA(),
	)
	d.renderHeader()
	d.renderMap()
	d.renderLegend()
	d.renderFooter()
}

//nolint:mnd // Halving centers text vertically in the fixed header.
func (d *DungeonViewDialog) renderHeader() {
	d.titleOutput.Reset()
	lineCount := 1
	if d.layout.legendBelow {
		lineCount = 2
		d.titleOutput.AppendLine().LightGreen("DUNGEON VIEW").LightCyan("   NORTH UP").Done()
		d.titleOutput.AppendLine().
			Yellow(strings.ToUpper(d.model.Location.String())).
			Whitef("   LEVEL %d OF %d", int(d.model.Floor)+1, references.DungeonFloorCount).
			Done()
	} else {
		d.titleOutput.AppendLine().
			LightGreen("DUNGEON VIEW").
			Yellow("   "+strings.ToUpper(d.model.Location.String())).
			Whitef("   LEVEL %d OF %d", int(d.model.Floor)+1, references.DungeonFloorCount).
			LightCyan("   NORTH UP").
			Done()
	}
	y := d.layout.header.Min.Y +
		(d.layout.header.Dy()-int(d.titleOutput.GetTotalHeightForLines(lineCount)))/2
	d.titleOutput.DrawPanelContent(d.buffer, image.Pt(d.layout.header.Min.X, y), d.layout.header.Dx(), text.AlignCenter)
}

//nolint:exhaustruct,mnd // Native pixel-grid passes keep topology crisp and deterministic.
func (d *DungeonViewDialog) renderMap() {
	mapBuffer := ebiten.NewImage(dungeonViewNativeSize, dungeonViewNativeSize)
	mapBuffer.Fill(pkgcolor.EGABlack)
	plan := buildDungeonViewRenderPlan(d.model)
	floorBounds := dungeonViewAtlasBounds()
	vector.FillRect(
		mapBuffer,
		float32(floorBounds.Min.X),
		float32(floorBounds.Min.Y),
		float32(floorBounds.Dx()),
		float32(floorBounds.Dy()),
		pkgcolor.From(pkgcolor.EGABlue).Intensity(0.25).RGBA(),
		false,
	)

	for sourceY := range dungeonViewFloorCells {
		for sourceX := range dungeonViewFloorCells {
			cellPlan := plan.cells[sourceY][sourceX]
			if !cellPlan.visible {
				continue
			}
			originX, originY := dungeonViewCellOrigin(sourceX, sourceY)
			drawDungeonViewSurface(
				mapBuffer,
				cellPlan.surface,
				originX,
				originY,
			)
		}
	}
	for sourceY := range dungeonViewFloorCells {
		for sourceX := range dungeonViewFloorCells {
			cellPlan := plan.cells[sourceY][sourceX]
			if !cellPlan.visible {
				continue
			}
			originX, originY := dungeonViewCellOrigin(sourceX, sourceY)
			drawDungeonViewDetail(
				mapBuffer,
				cellPlan,
				originX,
				originY,
			)
		}
	}
	drawDungeonViewContinuations(mapBuffer, plan)
	for sourceY := range dungeonViewFloorCells {
		for sourceX := range dungeonViewFloorCells {
			if !plan.cells[sourceY][sourceX].player {
				continue
			}
			originX, originY := dungeonViewCellOrigin(sourceX, sourceY)
			drawDungeonViewPlayerMarker(mapBuffer, originX, originY, d.model.Facing)
		}
	}
	vector.StrokeRect(
		mapBuffer,
		float32(floorBounds.Min.X)-0.5,
		float32(floorBounds.Min.Y)-0.5,
		float32(floorBounds.Dx())+1,
		float32(floorBounds.Dy())+1,
		1,
		pkgcolor.EGALightGray,
		false,
	)

	scale := float64(d.layout.mapScale)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(d.layout.mapArea.Min.X), float64(d.layout.mapArea.Min.Y))
	d.buffer.DrawImage(mapBuffer, op)
}

func dungeonViewAtlasBounds() image.Rectangle {
	return image.Rect(
		dungeonViewGridOrigin,
		dungeonViewGridOrigin,
		dungeonViewGridOrigin+dungeonViewFloorCells*dungeonViewCellSize,
		dungeonViewGridOrigin+dungeonViewFloorCells*dungeonViewCellSize,
	)
}

func dungeonViewCellOrigin(sourceX, sourceY int) (int, int) {
	return dungeonViewGridOrigin + sourceX*dungeonViewCellSize,
		dungeonViewGridOrigin + sourceY*dungeonViewCellSize
}

func wrappedDungeonViewAxis(value int) int {
	size := int(references.XDungeonMapTiles)
	value %= size
	if value < 0 {
		value += size
	}

	return value
}

//nolint:exhaustruct,mnd // Native legend icons use the exact map plan and integer scale.
func (d *DungeonViewDialog) renderLegend() {
	physicalIconSize := dungeonViewLegendPhysicalIconSize(d.layout)
	columnGap := dungeonViewLegendColumnGap
	if d.layout.legendBelow {
		columnGap = dungeonViewNarrowColumnGap
	}
	for column := range dungeonViewLegendColumnCount {
		columnRect := dungeonViewLegendColumnRect(d.layout, column, columnGap)
		columnX := columnRect.Min.X
		output := d.legendOutputs[column]
		for row := range dungeonViewLegendRowCount {
			index := column*dungeonViewLegendRowCount + row
			entry := dungeonViewLegend[index]
			rowY := d.layout.legend.Min.Y + row*physicalIconSize
			icon := dungeonViewLegendNativeIcon(entry)
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
			op.GeoM.Scale(float64(d.layout.mapScale), float64(d.layout.mapScale))
			op.GeoM.Translate(float64(columnX), float64(rowY))
			d.buffer.DrawImage(icon, op)

			output.Reset()
			output.AppendLine().LightGray(dungeonViewLegendLabel(entry, d.layout.legendBelow)).Done()
			textX := columnX + physicalIconSize + max(2, d.layout.mapScale*2)
			textY := rowY + (physicalIconSize-int(output.GetFontHeight()))/2
			output.DrawPanelContent(
				d.buffer,
				image.Pt(textX, textY),
				columnRect.Max.X-textX,
				text.AlignStart,
			)
		}
	}
}

func dungeonViewLegendLabel(entry dungeonViewLegendEntry, narrow bool) string {
	if narrow && entry.narrowLabel != "" {
		return entry.narrowLabel
	}

	return entry.label
}

//nolint:mnd // Percentage split reserves more narrow width for the longer second-column labels.
func dungeonViewLegendColumnRect(layout dungeonViewLayout, column, columnGap int) image.Rectangle {
	availableWidth := layout.legend.Dx() - columnGap
	firstWidth := availableWidth / dungeonViewLegendColumnCount
	if layout.legendBelow {
		firstWidth = availableWidth * dungeonViewNarrowFirstColumn / 100
	}
	if column == 0 {
		return image.Rect(
			layout.legend.Min.X,
			layout.legend.Min.Y,
			layout.legend.Min.X+firstWidth,
			layout.legend.Max.Y,
		)
	}

	return image.Rect(
		layout.legend.Min.X+firstWidth+columnGap,
		layout.legend.Min.Y,
		layout.legend.Max.X,
		layout.legend.Max.Y,
	)
}

func dungeonViewLegendPhysicalIconSize(layout dungeonViewLayout) int {
	return dungeonViewLegendIconSize * layout.mapScale
}

//nolint:mnd // Compact native symbols preserve every deliberate legend pixel at 75% map size.
func dungeonViewLegendNativeIcon(entry dungeonViewLegendEntry) *ebiten.Image {
	icon := ebiten.NewImage(dungeonViewLegendIconSize, dungeonViewLegendIconSize)
	switch {
	case entry.continuation:
		runeID := dungeonViewContinuationRune(dungeonViewAxisHorizontal, 0)
		drawDungeonViewRune(icon, runeID, 2, 2)
	case entry.player:
		drawDungeonViewLegendPlayer(icon)
	default:
		plan := dungeonViewLegendCellPlan(entry.cell)
		drawDungeonViewLegendCell(icon, plan)
	}

	return icon
}

//nolint:cyclop,mnd // Compact legend forms intentionally differ from the 16x16 map vocabulary.
func drawDungeonViewLegendCell(dst *ebiten.Image, plan dungeonViewCellPlan) {
	drawDungeonViewLegendSurface(dst, plan.surface)
	switch plan.icon {
	case dungeonViewIconNone:
		if plan.surface == dungeonViewSurfaceWall {
			drawDungeonViewLegendWall(dst)
		}
	case dungeonViewIconCeilingPit:
		drawDungeonViewLegendCeilingPit(dst)
	case dungeonViewIconLadderUp:
		drawDungeonViewLegendLadder(dst, true, false)
	case dungeonViewIconLadderDown:
		drawDungeonViewLegendLadder(dst, false, true)
	case dungeonViewIconLadderBoth:
		drawDungeonViewLegendLadder(dst, true, true)
	case dungeonViewIconChest:
		drawDungeonViewLegendChest(dst)
	case dungeonViewIconFountain:
		drawDungeonViewLegendFountain(dst)
	case dungeonViewIconOpenPit:
		drawDungeonViewLegendOpenPit(dst)
	case dungeonViewIconHiddenPit:
		drawDungeonViewLegendHiddenPit(dst)
	case dungeonViewIconPitCeiling:
		drawDungeonViewLegendCeilingPit(dst)
	case dungeonViewIconTrap:
		drawDungeonViewLegendTrap(dst)
	case dungeonViewIconField:
		drawDungeonViewLegendField(dst)
	case dungeonViewIconRoom:
		drawDungeonViewLegendRoom(dst)
	case dungeonViewIconPlaque:
		drawDungeonViewLegendWall(dst)
		vector.FillRect(dst, 4, 4, 4, 4, pkgcolor.EGABrown, false)
		vector.StrokeRect(dst, 3.5, 3.5, 5, 5, 1, pkgcolor.EGAWhite, false)
	case dungeonViewIconSpecialWall:
		drawDungeonViewLegendSpecialWall(dst)
	case dungeonViewIconHiddenDoor:
		drawDungeonViewLegendWall(dst)
		vector.StrokeLine(dst, 3, 9, 9, 3, 1, pkgcolor.EGALightCyan, false)
	case dungeonViewIconDoor:
		drawDungeonViewLegendDoor(dst)
	}
}

func drawDungeonViewLegendSurface(dst *ebiten.Image, surface dungeonViewSurface) {
	if surface == dungeonViewSurfaceFloor {
		dst.Fill(pkgcolor.From(pkgcolor.EGABlue).Intensity(dungeonViewFloorDim).RGBA())
	}
}

//nolint:mnd // Native 12x12 legend geometry is intentionally hand-placed.
func drawDungeonViewLegendWall(dst *ebiten.Image) {
	vector.FillRect(dst, 0, 5, dungeonViewLegendIconSize, 3, pkgcolor.EGALightGray, false)
	vector.FillRect(dst, 5, 5, 2, 3, pkgcolor.EGAWhite, false)
}

//nolint:mnd // Native 12x12 legend geometry is intentionally hand-placed.
func drawDungeonViewLegendSpecialWall(dst *ebiten.Image) {
	dst.Fill(pkgcolor.EGABlue)
	drawDungeonViewLegendWall(dst)
	vector.FillRect(dst, 5, 2, 2, 8, pkgcolor.EGALightCyan, false)
	vector.FillRect(dst, 3, 5, 6, 2, pkgcolor.EGALightCyan, false)
}

//nolint:mnd // Native 12x12 legend geometry is intentionally hand-placed.
func drawDungeonViewLegendDoor(dst *ebiten.Image) {
	vector.FillRect(dst, 0, 5, 4, 3, pkgcolor.EGALightGray, false)
	vector.FillRect(dst, 8, 5, 4, 3, pkgcolor.EGALightGray, false)
	vector.FillRect(dst, 4, 4, 1, 5, pkgcolor.EGALightGray, false)
	vector.FillRect(dst, 7, 4, 1, 5, pkgcolor.EGALightGray, false)
	vector.StrokeLine(dst, 6, 6, 4, 2, 1, pkgcolor.EGAYellow, false)
}

//nolint:mnd // Native 12x12 legend geometry is intentionally hand-placed.
func drawDungeonViewLegendPlayer(dst *ebiten.Image) {
	drawDungeonViewLegendSurface(dst, dungeonViewSurfaceFloor)
	marker := pkgcolor.EGALightGreen
	drawDungeonViewLine(dst, 0, 0, 3, marker)
	drawDungeonViewLine(dst, 9, 0, 3, marker)
	drawDungeonViewLine(dst, 0, 11, 3, marker)
	drawDungeonViewLine(dst, 9, 11, 3, marker)
	vector.FillRect(dst, 0, 0, 1, 3, marker, false)
	vector.FillRect(dst, 11, 0, 1, 3, marker, false)
	vector.FillRect(dst, 0, 9, 1, 3, marker, false)
	vector.FillRect(dst, 11, 9, 1, 3, marker, false)
	vector.StrokeLine(dst, 6, 9, 6, 2, 1, pkgcolor.EGAYellow, false)
	vector.StrokeLine(dst, 6, 2, 3, 5, 1, pkgcolor.EGAYellow, false)
	vector.StrokeLine(dst, 6, 2, 9, 5, 1, pkgcolor.EGAYellow, false)
}

//nolint:mnd // Native 12x12 legend geometry is intentionally hand-placed.
func drawDungeonViewLegendLadder(dst *ebiten.Image, hasUp, hasDown bool) {
	vector.FillRect(dst, 3, 2, 1, 8, pkgcolor.EGALightGray, false)
	vector.FillRect(dst, 8, 2, 1, 8, pkgcolor.EGALightGray, false)
	for rungY := 3; rungY <= 9; rungY += 2 {
		drawDungeonViewLine(dst, 3, rungY, 6, pkgcolor.EGAWhite)
	}
	if hasUp {
		vector.StrokeLine(dst, 6, 1, 4, 3, 1, pkgcolor.EGAYellow, false)
		vector.StrokeLine(dst, 6, 1, 8, 3, 1, pkgcolor.EGAYellow, false)
	}
	if hasDown {
		vector.StrokeLine(dst, 6, 10, 4, 8, 1, pkgcolor.EGAYellow, false)
		vector.StrokeLine(dst, 6, 10, 8, 8, 1, pkgcolor.EGAYellow, false)
	}
}

//nolint:mnd // Native 12x12 legend geometry is intentionally hand-placed.
func drawDungeonViewLegendChest(dst *ebiten.Image) {
	vector.FillRect(dst, 2, 4, 8, 6, pkgcolor.EGABrown, false)
	vector.StrokeRect(dst, 2.5, 4.5, 7, 5, 1, pkgcolor.EGAYellow, false)
	drawDungeonViewLine(dst, 2, 7, 8, pkgcolor.EGAYellow)
	vector.FillRect(dst, 6, 7, 1, 2, pkgcolor.EGAWhite, false)
}

//nolint:mnd // Native 12x12 legend geometry is intentionally hand-placed.
func drawDungeonViewLegendFountain(dst *ebiten.Image) {
	drawDungeonViewLine(dst, 2, 8, 8, pkgcolor.EGAWhite)
	drawDungeonViewLine(dst, 3, 10, 6, pkgcolor.EGALightGray)
	vector.StrokeLine(dst, 3, 7, 6, 3, 1, pkgcolor.EGALightCyan, false)
	vector.StrokeLine(dst, 6, 3, 9, 7, 1, pkgcolor.EGALightCyan, false)
	vector.FillRect(dst, 5, 5, 3, 3, pkgcolor.EGALightBlue, false)
}

//nolint:mnd // Native 12x12 legend geometry is intentionally hand-placed.
func drawDungeonViewLegendOpenPit(dst *ebiten.Image) {
	vector.FillRect(dst, 3, 3, 6, 6, pkgcolor.EGABlack, false)
	vector.StrokeRect(dst, 2.5, 2.5, 7, 7, 1, pkgcolor.EGALightGray, false)
}

//nolint:mnd // Native 12x12 legend geometry is intentionally hand-placed.
func drawDungeonViewLegendHiddenPit(dst *ebiten.Image) {
	vector.StrokeRect(dst, 4.5, 4.5, 3, 3, 1, pkgcolor.EGADarkGray, false)
	vector.StrokeLine(dst, 3, 9, 9, 3, 1, pkgcolor.EGALightGray, false)
}

//nolint:mnd // Native 12x12 legend geometry is intentionally hand-placed.
func drawDungeonViewLegendCeilingPit(dst *ebiten.Image) {
	vector.StrokeRect(dst, 2.5, 2.5, 7, 7, 1, pkgcolor.EGALightGray, false)
	drawDungeonViewLine(dst, 3, 6, 6, pkgcolor.EGADarkGray)
}

//nolint:mnd // Native 12x12 legend geometry is intentionally hand-placed.
func drawDungeonViewLegendTrap(dst *ebiten.Image) {
	vector.StrokeLine(dst, 2, 2, 10, 10, 1, pkgcolor.EGALightRed, false)
	vector.StrokeLine(dst, 10, 2, 2, 10, 1, pkgcolor.EGALightRed, false)
	vector.FillRect(dst, 5, 5, 2, 2, pkgcolor.EGAYellow, false)
}

//nolint:mnd // Native 12x12 legend geometry is intentionally hand-placed.
func drawDungeonViewLegendField(dst *ebiten.Image) {
	for index, fieldColor := range [...]stdcolor.Color{pkgcolor.EGALightMagenta, pkgcolor.EGALightCyan} {
		x := float32(4 + index*4)
		vector.StrokeLine(dst, x, 1, x-1, 4, 1, fieldColor, false)
		vector.StrokeLine(dst, x-1, 4, x, 7, 1, fieldColor, false)
		vector.StrokeLine(dst, x, 7, x-1, 10, 1, fieldColor, false)
	}
}

//nolint:mnd // Native 12x12 legend geometry is intentionally hand-placed.
func drawDungeonViewLegendRoom(dst *ebiten.Image) {
	vector.StrokeRect(dst, 2.5, 2.5, 7, 7, 1, pkgcolor.EGAYellow, false)
	vector.FillRect(dst, 5, 5, 2, 2, pkgcolor.EGAWhite, false)
}

//nolint:mnd // Halving centers text vertically in the fixed footer.
func (d *DungeonViewDialog) renderFooter() {
	d.footerOutput.Reset()
	lines := dungeonViewFooterLines(d.layout.legendBelow)
	if d.layout.legendBelow {
		d.footerOutput.AppendLine().Cyan(lines[0]).Done()
		d.footerOutput.AppendLine().LightGray(lines[1]).Done()
	} else {
		d.footerOutput.AppendLine().
			Cyan("MATCHING EDGE MARKS CONNECT").
			LightGray("     PRESS ENTER TO RETURN").
			Done()
	}
	y := d.layout.footer.Min.Y + (d.layout.footer.Dy()-int(d.footerOutput.GetTotalHeightForLines(len(lines))))/2
	d.footerOutput.DrawPanelContent(d.buffer, image.Pt(d.layout.footer.Min.X, y), d.layout.footer.Dx(), text.AlignCenter)
}

func dungeonViewFooterLines(narrow bool) []string {
	if narrow {
		return []string{"MATCHING EDGE MARKS CONNECT", "PRESS ENTER TO RETURN"}
	}

	return []string{"MATCHING EDGE MARKS CONNECT     PRESS ENTER TO RETURN"}
}

//nolint:mnd // Pixel-snapped corner brackets leave the planned topology visible beneath the marker.
func drawDungeonViewPlayerMarker(
	destination *ebiten.Image,
	originX int,
	originY int,
	facing references.DungeonFacing,
) {
	markerColor := pkgcolor.EGALightGreen
	drawDungeonViewLine(destination, originX, originY, 4, markerColor)
	drawDungeonViewLine(destination, originX+12, originY, 4, markerColor)
	drawDungeonViewLine(destination, originX, originY+15, 4, markerColor)
	drawDungeonViewLine(destination, originX+12, originY+15, 4, markerColor)
	vector.FillRect(destination, float32(originX), float32(originY), 1, 4, markerColor, false)
	vector.FillRect(destination, float32(originX+15), float32(originY), 1, 4, markerColor, false)
	vector.FillRect(destination, float32(originX), float32(originY+12), 1, 4, markerColor, false)
	vector.FillRect(destination, float32(originX+15), float32(originY+12), 1, 4, markerColor, false)
	for _, segment := range dungeonViewPlayerSegments(facing) {
		vector.StrokeLine(
			destination,
			float32(originX)+segment.fromX,
			float32(originY)+segment.fromY,
			float32(originX)+segment.toX,
			float32(originY)+segment.toY,
			2,
			pkgcolor.EGABlack,
			false,
		)
		vector.StrokeLine(
			destination,
			float32(originX)+segment.fromX,
			float32(originY)+segment.fromY,
			float32(originX)+segment.toX,
			float32(originY)+segment.toY,
			1,
			pkgcolor.EGAYellow,
			false,
		)
	}
}

func dungeonViewPlayerSegments(facing references.DungeonFacing) []dungeonViewSegment {
	const (
		minimum = float32(2)
		center  = float32(8)
		maximum = float32(14)
	)

	switch facing {
	case references.DungeonFacingNorth:
		return []dungeonViewSegment{
			{fromX: center, fromY: maximum, toX: center, toY: minimum},
			{fromX: center, fromY: minimum, toX: minimum, toY: center},
			{fromX: center, fromY: minimum, toX: maximum, toY: center},
		}
	case references.DungeonFacingEast:
		return []dungeonViewSegment{
			{fromX: minimum, fromY: center, toX: maximum, toY: center},
			{fromX: maximum, fromY: center, toX: center, toY: minimum},
			{fromX: maximum, fromY: center, toX: center, toY: maximum},
		}
	case references.DungeonFacingSouth:
		return []dungeonViewSegment{
			{fromX: center, fromY: minimum, toX: center, toY: maximum},
			{fromX: center, fromY: maximum, toX: minimum, toY: center},
			{fromX: center, fromY: maximum, toX: maximum, toY: center},
		}
	case references.DungeonFacingWest:
		return []dungeonViewSegment{
			{fromX: maximum, fromY: center, toX: minimum, toY: center},
			{fromX: minimum, fromY: center, toX: center, toY: minimum},
			{fromX: minimum, fromY: center, toX: center, toY: maximum},
		}
	default:
		panic("invalid dungeon View facing")
	}
}

//nolint:cyclop,exhaustruct,funlen // Packed cell variants map directly to compact vector symbols.
func dungeonViewCellPlanFor(cell references.DungeonCell) dungeonViewCellPlan {
	plan := dungeonViewCellPlan{surface: dungeonViewSurfaceFloor}
	switch cell.Kind() {
	case references.DungeonPassage:
		if cell.HasBit08() {
			plan.icon = dungeonViewIconCeilingPit
		}
	case references.DungeonLadderUp:
		plan.icon = dungeonViewIconLadderUp
	case references.DungeonLadderDown:
		plan.icon = dungeonViewIconLadderDown
	case references.DungeonLadderBoth:
		plan.icon = dungeonViewIconLadderBoth
	case references.DungeonClosedChest:
		plan.icon = dungeonViewIconChest
	case references.DungeonFountain:
		plan.icon = dungeonViewIconFountain
	case references.DungeonPitOrTrap:
		switch cell.Raw() {
		case 0x60:
			plan.icon = dungeonViewIconOpenPit
		case 0x61, 0x69:
			plan.icon = dungeonViewIconHiddenPit
		case 0x68:
			plan.icon = dungeonViewIconPitCeiling
		default:
			plan.icon = dungeonViewIconTrap
		}
	case references.DungeonOpenChest, references.DungeonReserved:
	case references.DungeonEnergyField:
		plan.icon = dungeonViewIconField
	case references.DungeonClearedRoom, references.DungeonLiveRoom:
		plan.icon = dungeonViewIconRoom
	case references.DungeonWall:
		plan.surface = dungeonViewSurfaceWall
		if cell.Raw() != dungeonViewPlainWall {
			plan.icon = dungeonViewIconPlaque
		}
	case references.DungeonSpecialWall:
		plan.surface = dungeonViewSurfaceWall
		plan.icon = dungeonViewIconSpecialWall
	case references.DungeonHiddenDoor:
		plan.surface = dungeonViewSurfaceWall
		plan.icon = dungeonViewIconHiddenDoor
	case references.DungeonRevealedDoor:
		plan.icon = dungeonViewIconDoor
	}

	return plan
}

func dungeonViewLegendCellPlan(cell references.DungeonCell) dungeonViewCellPlan {
	plan := dungeonViewCellPlanFor(cell)
	plan.visible = true
	if plan.surface == dungeonViewSurfaceWall || plan.icon == dungeonViewIconDoor {
		plan.doorAxis = dungeonViewAxisHorizontal
		plan.topology = dungeonViewEast | dungeonViewWest
	}

	return plan
}

//nolint:cyclop,exhaustruct // Bounded passes collapse cells, record known edges, then derive hidden-safe topology.
func buildDungeonViewRenderPlan(model game_state.DungeonViewModel) dungeonViewRenderPlan {
	plan := dungeonViewRenderPlan{}
	for displayY := range game_state.DungeonViewSize {
		for displayX := range game_state.DungeonViewSize {
			cell := model.Cells[displayY][displayX]
			if !cell.Visible {
				continue
			}
			sourceX, sourceY := dungeonViewSourceCoordinates(cell)
			if !plan.cells[sourceY][sourceX].visible {
				plan.cells[sourceY][sourceX] = dungeonViewCellPlanFor(cell.SourceCell)
				plan.cells[sourceY][sourceX].visible = true
			}
			plan.cells[sourceY][sourceX].player = plan.cells[sourceY][sourceX].player || cell.Player
		}
	}
	dungeonViewRecordKnownEdges(model, &plan.edges)

	for sourceY := range dungeonViewFloorCells {
		for sourceX := range dungeonViewFloorCells {
			cellPlan := &plan.cells[sourceY][sourceX]
			if !cellPlan.visible {
				continue
			}
			if dungeonViewPlanIsDoor(*cellPlan) {
				cellPlan.doorAxis = dungeonViewDoorAxis(plan, sourceX, sourceY)
				cellPlan.topology = dungeonViewDoorTopology(plan, sourceX, sourceY, cellPlan.doorAxis)
			} else if cellPlan.surface == dungeonViewSurfaceWall {
				cellPlan.topology = dungeonViewWallTopology(plan, sourceX, sourceY)
			}
		}
	}

	return plan
}

func dungeonViewSourceCoordinates(cell game_state.DungeonViewCell) (int, int) {
	sourceX, sourceY := int(cell.SourcePosition.X), int(cell.SourcePosition.Y)
	if sourceX < 0 || sourceX >= dungeonViewFloorCells || sourceY < 0 || sourceY >= dungeonViewFloorCells {
		panic("visible dungeon view cell has invalid source coordinates")
	}

	return sourceX, sourceY
}

func dungeonViewRecordKnownEdges(model game_state.DungeonViewModel, edges *dungeonViewKnownEdges) {
	for displayY := range game_state.DungeonViewSize {
		for displayX := range game_state.DungeonViewSize {
			current := model.Cells[displayY][displayX]
			if !current.Visible {
				continue
			}
			sourceX, sourceY := dungeonViewSourceCoordinates(current)
			if dungeonViewNeighborIsKnown(model, displayX, displayY, 1, 0) {
				edges.east[sourceY][sourceX] = true
			}
			if dungeonViewNeighborIsKnown(model, displayX, displayY, 0, 1) {
				edges.south[sourceY][sourceX] = true
			}
		}
	}
}

type dungeonViewDirection struct {
	bit      dungeonViewTopology
	opposite dungeonViewTopology
	dx       int
	dy       int
}

func dungeonViewCardinalDirections() [4]dungeonViewDirection {
	return [...]dungeonViewDirection{
		{bit: dungeonViewNorth, opposite: dungeonViewSouth, dx: 0, dy: -1},
		{bit: dungeonViewEast, opposite: dungeonViewWest, dx: 1, dy: 0},
		{bit: dungeonViewSouth, opposite: dungeonViewNorth, dx: 0, dy: 1},
		{bit: dungeonViewWest, opposite: dungeonViewEast, dx: -1, dy: 0},
	}
}

func dungeonViewNeighborIsKnown(
	model game_state.DungeonViewModel,
	displayX int,
	displayY int,
	offsetX int,
	offsetY int,
) bool {
	neighborX, neighborY := displayX+offsetX, displayY+offsetY
	if !dungeonViewDisplayCoordinateInBounds(displayX, displayY) ||
		!dungeonViewDisplayCoordinateInBounds(neighborX, neighborY) {
		return false
	}
	current := model.Cells[displayY][displayX]
	neighbor := model.Cells[neighborY][neighborX]
	if !current.Visible || !neighbor.Visible {
		return false
	}
	expectedX := wrappedDungeonViewAxis(int(current.SourcePosition.X) + offsetX)
	expectedY := wrappedDungeonViewAxis(int(current.SourcePosition.Y) + offsetY)
	if int(neighbor.SourcePosition.X) != expectedX || int(neighbor.SourcePosition.Y) != expectedY {
		return false
	}

	return true
}

func dungeonViewDisplayCoordinateInBounds(displayX, displayY int) bool {
	return displayX >= 0 && displayX < game_state.DungeonViewSize &&
		displayY >= 0 && displayY < game_state.DungeonViewSize
}

func dungeonViewWallTopology(plan dungeonViewRenderPlan, sourceX, sourceY int) dungeonViewTopology {
	var topology dungeonViewTopology
	for _, direction := range dungeonViewCardinalDirections() {
		neighbor, ok := dungeonViewAtlasNeighbor(plan, sourceX, sourceY, direction)
		if !ok {
			continue
		}
		if neighbor.surface == dungeonViewSurfaceWall && !dungeonViewPlanIsDoor(neighbor) {
			topology |= direction.bit
		} else if dungeonViewPlanIsDoor(neighbor) {
			neighborX := wrappedDungeonViewAxis(sourceX + direction.dx)
			neighborY := wrappedDungeonViewAxis(sourceY + direction.dy)
			axis := dungeonViewDoorAxis(plan, neighborX, neighborY)
			if dungeonViewTopologyForAxis(axis)&direction.opposite != 0 {
				topology |= direction.bit
			}
		}
	}

	return topology
}

func dungeonViewDoorAxis(plan dungeonViewRenderPlan, sourceX, sourceY int) dungeonViewAxis {
	horizontalScore, verticalScore := 0, 0
	for _, direction := range dungeonViewCardinalDirections() {
		neighbor, ok := dungeonViewAtlasNeighbor(plan, sourceX, sourceY, direction)
		if !ok {
			continue
		}
		horizontalDirection := direction.bit == dungeonViewEast || direction.bit == dungeonViewWest
		if dungeonViewPlanIsWallAnchor(neighbor) {
			if horizontalDirection {
				horizontalScore += 2
			} else {
				verticalScore += 2
			}

			continue
		}
		if horizontalDirection {
			verticalScore++
		} else {
			horizontalScore++
		}
	}
	if verticalScore > horizontalScore {
		return dungeonViewAxisVertical
	}

	return dungeonViewAxisHorizontal
}

func dungeonViewDoorTopology(
	plan dungeonViewRenderPlan,
	sourceX int,
	sourceY int,
	axis dungeonViewAxis,
) dungeonViewTopology {
	axisTopology := dungeonViewTopologyForAxis(axis)
	var topology dungeonViewTopology
	for _, direction := range dungeonViewCardinalDirections() {
		if axisTopology&direction.bit == 0 {
			continue
		}
		if _, ok := dungeonViewAtlasNeighbor(plan, sourceX, sourceY, direction); ok {
			topology |= direction.bit
		}
	}

	return topology
}

func dungeonViewAtlasNeighbor(
	plan dungeonViewRenderPlan,
	sourceX int,
	sourceY int,
	direction dungeonViewDirection,
) (dungeonViewCellPlan, bool) {
	if !dungeonViewKnownEdge(plan.edges, sourceX, sourceY, direction.bit) {
		var empty dungeonViewCellPlan

		return empty, false
	}
	neighborX := wrappedDungeonViewAxis(sourceX + direction.dx)
	neighborY := wrappedDungeonViewAxis(sourceY + direction.dy)
	neighbor := plan.cells[neighborY][neighborX]

	return neighbor, neighbor.visible
}

func dungeonViewKnownEdge(edges dungeonViewKnownEdges, sourceX, sourceY int, direction dungeonViewTopology) bool {
	switch direction {
	case dungeonViewNorth:
		return edges.south[wrappedDungeonViewAxis(sourceY-1)][sourceX]
	case dungeonViewEast:
		return edges.east[sourceY][sourceX]
	case dungeonViewSouth:
		return edges.south[sourceY][sourceX]
	case dungeonViewWest:
		return edges.east[sourceY][wrappedDungeonViewAxis(sourceX-1)]
	default:
		panic("invalid dungeon view edge direction")
	}
}

func dungeonViewPlanIsDoor(plan dungeonViewCellPlan) bool {
	return plan.icon == dungeonViewIconDoor || plan.icon == dungeonViewIconHiddenDoor
}

func dungeonViewPlanIsWallAnchor(plan dungeonViewCellPlan) bool {
	return plan.surface == dungeonViewSurfaceWall || plan.icon == dungeonViewIconDoor
}

func dungeonViewContinuationRune(axis dungeonViewAxis, sourceIndex int) int {
	if sourceIndex < 0 || sourceIndex >= dungeonViewFloorCells {
		panic("invalid dungeon view continuation index")
	}
	switch axis {
	case dungeonViewAxisHorizontal:
		return dungeonViewHorizontalRuneA + sourceIndex
	case dungeonViewAxisVertical:
		return dungeonViewVerticalRuneI + sourceIndex
	case dungeonViewAxisNone:
		panic("continuation rune requires a cardinal axis")
	default:
		panic("invalid dungeon view continuation axis")
	}
}

func dungeonViewHasContinuation(plan dungeonViewRenderPlan, axis dungeonViewAxis, sourceIndex int) bool {
	if sourceIndex < 0 || sourceIndex >= dungeonViewFloorCells {
		return false
	}
	var first, second dungeonViewCellPlan
	var edgeKnown bool
	switch axis {
	case dungeonViewAxisHorizontal:
		first = plan.cells[sourceIndex][dungeonViewFloorCells-1]
		second = plan.cells[sourceIndex][0]
		edgeKnown = plan.edges.east[sourceIndex][dungeonViewFloorCells-1]
	case dungeonViewAxisVertical:
		first = plan.cells[dungeonViewFloorCells-1][sourceIndex]
		second = plan.cells[0][sourceIndex]
		edgeKnown = plan.edges.south[dungeonViewFloorCells-1][sourceIndex]
	case dungeonViewAxisNone:
		return false
	default:
		panic("invalid dungeon view continuation axis")
	}

	return edgeKnown && first.visible && second.visible &&
		first.surface != dungeonViewSurfaceWall && second.surface != dungeonViewSurfaceWall
}

//nolint:mnd // Halves center 8x8 runes within 16x16 source rows and columns.
func drawDungeonViewContinuations(dst *ebiten.Image, plan dungeonViewRenderPlan) {
	atlasBounds := dungeonViewAtlasBounds()
	for sourceRow := range dungeonViewFloorCells {
		if !dungeonViewHasContinuation(plan, dungeonViewAxisHorizontal, sourceRow) {
			continue
		}
		runeID := dungeonViewContinuationRune(dungeonViewAxisHorizontal, sourceRow)
		y := atlasBounds.Min.Y + sourceRow*dungeonViewCellSize + (dungeonViewCellSize-utext.BitmapCharHeight)/2
		drawDungeonViewRune(dst, runeID, 0, y)
		drawDungeonViewRune(dst, runeID, atlasBounds.Max.X+dungeonViewRightMarkerPad, y)
	}
	for sourceColumn := range dungeonViewFloorCells {
		if !dungeonViewHasContinuation(plan, dungeonViewAxisVertical, sourceColumn) {
			continue
		}
		runeID := dungeonViewContinuationRune(dungeonViewAxisVertical, sourceColumn)
		x := atlasBounds.Min.X + sourceColumn*dungeonViewCellSize + (dungeonViewCellSize-utext.BitmapCharWidth)/2
		drawDungeonViewRune(dst, runeID, x, 0)
		drawDungeonViewRune(dst, runeID, x, atlasBounds.Max.Y)
	}
}

//nolint:mnd // Bitmap tint channels are normalized from 8-bit EGA color values.
func drawDungeonViewRune(dst *ebiten.Image, runeID, originX, originY int) {
	markerColor := pkgcolor.EGALightCyan.ToRGBA()
	utext.GetBitmapFonts().Runic.DrawChar(
		dst,
		runeID,
		float64(originX),
		float64(originY),
		1,
		float32(markerColor.R)/255,
		float32(markerColor.G)/255,
		float32(markerColor.B)/255,
	)
}

func dungeonViewTopologyForAxis(axis dungeonViewAxis) dungeonViewTopology {
	switch axis {
	case dungeonViewAxisHorizontal:
		return dungeonViewEast | dungeonViewWest
	case dungeonViewAxisVertical:
		return dungeonViewNorth | dungeonViewSouth
	case dungeonViewAxisNone:
		return 0
	default:
		panic("invalid dungeon view axis")
	}
}

//nolint:varnamelen // x and y are native pixel coordinates.
func drawDungeonViewSurface(dst *ebiten.Image, surface dungeonViewSurface, x, y int) {
	switch surface {
	case dungeonViewSurfaceFloor:
		vector.FillRect(
			dst, float32(x), float32(y), dungeonViewCellSize, dungeonViewCellSize,
			pkgcolor.From(pkgcolor.EGABlue).Intensity(dungeonViewFloorDim).RGBA(), false,
		)
	case dungeonViewSurfaceWall:
		vector.FillRect(dst, float32(x), float32(y), dungeonViewCellSize, dungeonViewCellSize, pkgcolor.EGABlack, false)
	case dungeonViewSurfaceNone:
	}
}

//nolint:cyclop,varnamelen // Compact icon dispatch keeps the vector vocabulary visible in one place.
func drawDungeonViewDetail(dst *ebiten.Image, plan dungeonViewCellPlan, x, y int) {
	if plan.icon == dungeonViewIconDoor || plan.icon == dungeonViewIconHiddenDoor {
		drawDungeonViewDoor(dst, x, y, plan.doorAxis, plan.topology, plan.icon == dungeonViewIconHiddenDoor)

		return
	}
	if plan.surface == dungeonViewSurfaceWall {
		drawDungeonViewConnectedWall(dst, x, y, plan.topology)
	}

	switch plan.icon {
	case dungeonViewIconCeilingPit:
		drawDungeonViewCeilingPit(dst, x, y)
	case dungeonViewIconLadderUp:
		drawDungeonViewLadder(dst, x, y, true, false)
	case dungeonViewIconLadderDown:
		drawDungeonViewLadder(dst, x, y, false, true)
	case dungeonViewIconLadderBoth:
		drawDungeonViewLadder(dst, x, y, true, true)
	case dungeonViewIconChest:
		drawDungeonViewChest(dst, x, y)
	case dungeonViewIconFountain:
		drawDungeonViewFountain(dst, x, y)
	case dungeonViewIconOpenPit, dungeonViewIconHiddenPit, dungeonViewIconPitCeiling, dungeonViewIconTrap:
		drawDungeonViewPit(dst, x, y, plan.icon)
	case dungeonViewIconField:
		drawDungeonViewField(dst, x, y)
	case dungeonViewIconRoom:
		drawDungeonViewRoom(dst, x, y)
	case dungeonViewIconPlaque:
		drawDungeonViewPlaque(dst, x, y)
	case dungeonViewIconSpecialWall:
		drawDungeonViewSpecialWall(dst, x, y)
	case dungeonViewIconNone:
	case dungeonViewIconHiddenDoor, dungeonViewIconDoor:
		panic("door icon bypassed dungeon view door rendering")
	}
}

//nolint:mnd,varnamelen // Three-native-pixel cardinal runs join exactly at neighboring cell boundaries.
func drawDungeonViewConnectedWall(dst *ebiten.Image, x, y int, topology dungeonViewTopology) {
	wallColor := pkgcolor.EGALightGray
	if topology == 0 {
		vector.StrokeRect(dst, float32(x)+4.5, float32(y)+4.5, 7, 7, 3, wallColor, false)
		vector.FillRect(dst, float32(x+7), float32(y+7), 3, 3, pkgcolor.EGAWhite, false)

		return
	}
	if topology&dungeonViewNorth != 0 {
		vector.FillRect(dst, float32(x+7), float32(y), 3, 10, wallColor, false)
	}
	if topology&dungeonViewEast != 0 {
		vector.FillRect(dst, float32(x+7), float32(y+7), 9, 3, wallColor, false)
	}
	if topology&dungeonViewSouth != 0 {
		vector.FillRect(dst, float32(x+7), float32(y+7), 3, 9, wallColor, false)
	}
	if topology&dungeonViewWest != 0 {
		vector.FillRect(dst, float32(x), float32(y+7), 10, 3, wallColor, false)
	}
	vector.FillRect(dst, float32(x+7), float32(y+7), 3, 3, pkgcolor.EGAWhite, false)
}

//nolint:cyclop,mnd,nestif,varnamelen // Door stubs render only topology proven by known source edges.
func drawDungeonViewDoor(
	dst *ebiten.Image,
	x int,
	y int,
	axis dungeonViewAxis,
	topology dungeonViewTopology,
	hidden bool,
) {
	wallColor := pkgcolor.EGALightGray
	if hidden {
		if axis == dungeonViewAxisVertical {
			if topology&dungeonViewNorth != 0 {
				vector.FillRect(dst, float32(x+7), float32(y), 3, 10, wallColor, false)
			}
			if topology&dungeonViewSouth != 0 {
				vector.FillRect(dst, float32(x+7), float32(y+7), 3, 9, wallColor, false)
			}
		} else {
			if topology&dungeonViewWest != 0 {
				vector.FillRect(dst, float32(x), float32(y+7), 10, 3, wallColor, false)
			}
			if topology&dungeonViewEast != 0 {
				vector.FillRect(dst, float32(x+7), float32(y+7), 9, 3, wallColor, false)
			}
		}
		vector.StrokeLine(dst, float32(x+5), float32(y+11), float32(x+11), float32(y+5), 1, pkgcolor.EGALightCyan, false)

		return
	}
	if axis == dungeonViewAxisVertical {
		if topology&dungeonViewNorth != 0 {
			vector.FillRect(dst, float32(x+7), float32(y), 3, 6, wallColor, false)
		}
		if topology&dungeonViewSouth != 0 {
			vector.FillRect(dst, float32(x+7), float32(y+10), 3, 6, wallColor, false)
		}
		vector.FillRect(dst, float32(x+6), float32(y+5), 5, 1, wallColor, false)
		vector.FillRect(dst, float32(x+6), float32(y+10), 5, 1, wallColor, false)
		vector.StrokeLine(dst, float32(x+8), float32(y+8), float32(x+14), float32(y+5), 1, pkgcolor.EGAYellow, false)

		return
	}
	if topology&dungeonViewWest != 0 {
		vector.FillRect(dst, float32(x), float32(y+7), 6, 3, wallColor, false)
	}
	if topology&dungeonViewEast != 0 {
		vector.FillRect(dst, float32(x+10), float32(y+7), 6, 3, wallColor, false)
	}
	vector.FillRect(dst, float32(x+5), float32(y+6), 1, 5, wallColor, false)
	vector.FillRect(dst, float32(x+10), float32(y+6), 1, 5, wallColor, false)
	vector.StrokeLine(dst, float32(x+8), float32(y+8), float32(x+5), float32(y+2), 1, pkgcolor.EGAYellow, false)
}

//nolint:mnd // Native 16x16 blueprint icon.
func drawDungeonViewCeilingPit(dst *ebiten.Image, x, y int) {
	vector.StrokeRect(dst, float32(x)+3.5, float32(y)+3.5, 9, 9, 1, pkgcolor.EGALightGray, false)
	drawDungeonViewLine(dst, x+4, y+4, 3, pkgcolor.EGALightCyan)
	drawDungeonViewLine(dst, x+9, y+11, 3, pkgcolor.EGALightCyan)
}

//nolint:mnd,varnamelen // Native 16x16 blueprint icon.
func drawDungeonViewLadder(dst *ebiten.Image, x, y int, up, down bool) {
	vector.FillRect(dst, float32(x+5), float32(y+2), 1, 12, pkgcolor.EGALightGray, false)
	vector.FillRect(dst, float32(x+10), float32(y+2), 1, 12, pkgcolor.EGALightGray, false)
	for rungY := 4; rungY <= 13; rungY += 3 {
		drawDungeonViewLine(dst, x+5, y+rungY, 6, pkgcolor.EGAWhite)
	}
	if up {
		vector.StrokeLine(dst, float32(x+8), float32(y+1), float32(x+6), float32(y+3), 1, pkgcolor.EGAYellow, false)
		vector.StrokeLine(dst, float32(x+8), float32(y+1), float32(x+10), float32(y+3), 1, pkgcolor.EGAYellow, false)
	}
	if down {
		vector.StrokeLine(dst, float32(x+8), float32(y+15), float32(x+6), float32(y+13), 1, pkgcolor.EGAYellow, false)
		vector.StrokeLine(dst, float32(x+8), float32(y+15), float32(x+10), float32(y+13), 1, pkgcolor.EGAYellow, false)
	}
}

//nolint:mnd // Native 16x16 blueprint icon.
func drawDungeonViewChest(dst *ebiten.Image, x, y int) {
	vector.FillRect(dst, float32(x+3), float32(y+5), 10, 8, pkgcolor.EGABrown, false)
	vector.StrokeRect(dst, float32(x)+3.5, float32(y)+5.5, 9, 7, 1, pkgcolor.EGAYellow, false)
	drawDungeonViewLine(dst, x+3, y+8, 10, pkgcolor.EGAYellow)
	vector.FillRect(dst, float32(x+8), float32(y+8), 1, 3, pkgcolor.EGAWhite, false)
}

//nolint:mnd,varnamelen // Native 16x16 blueprint icon.
func drawDungeonViewFountain(dst *ebiten.Image, x, y int) {
	drawDungeonViewLine(dst, x+2, y+9, 12, pkgcolor.EGAWhite)
	drawDungeonViewLine(dst, x+4, y+11, 8, pkgcolor.EGAWhite)
	drawDungeonViewLine(dst, x+2, y+13, 3, pkgcolor.EGALightGray)
	drawDungeonViewLine(dst, x+11, y+13, 3, pkgcolor.EGALightGray)
	vector.StrokeLine(dst, float32(x+3), float32(y+7), float32(x+7), float32(y+3), 1, pkgcolor.EGALightCyan, false)
	vector.StrokeLine(dst, float32(x+9), float32(y+3), float32(x+13), float32(y+7), 1, pkgcolor.EGALightCyan, false)
	vector.FillRect(dst, float32(x+7), float32(y+5), 3, 4, pkgcolor.EGALightBlue, false)
}

//nolint:mnd,varnamelen // Native 16x16 blueprint icon.
func drawDungeonViewPit(dst *ebiten.Image, x, y int, icon dungeonViewIcon) {
	//nolint:exhaustive // This helper accepts only the four pit icon variants.
	switch icon {
	case dungeonViewIconOpenPit:
		vector.FillRect(dst, float32(x+4), float32(y+4), 8, 8, pkgcolor.EGABlack, false)
		vector.StrokeRect(dst, float32(x)+3.5, float32(y)+3.5, 9, 9, 1, pkgcolor.EGALightGray, false)
	case dungeonViewIconHiddenPit:
		vector.StrokeRect(dst, float32(x)+5.5, float32(y)+5.5, 5, 5, 1, pkgcolor.EGADarkGray, false)
		vector.StrokeLine(dst, float32(x+4), float32(y+12), float32(x+12), float32(y+4), 1, pkgcolor.EGALightGray, false)
	case dungeonViewIconPitCeiling:
		vector.StrokeRect(dst, float32(x)+3.5, float32(y)+3.5, 9, 9, 1, pkgcolor.EGALightGray, false)
		drawDungeonViewLine(dst, x+3, y+8, 11, pkgcolor.EGADarkGray)
	case dungeonViewIconTrap:
		vector.StrokeLine(dst, float32(x+3), float32(y+3), float32(x+13), float32(y+13), 1, pkgcolor.EGALightRed, false)
		vector.StrokeLine(dst, float32(x+13), float32(y+3), float32(x+3), float32(y+13), 1, pkgcolor.EGALightRed, false)
		vector.FillRect(dst, float32(x+7), float32(y+7), 3, 3, pkgcolor.EGAYellow, false)
	default:
		panic("invalid dungeon view pit icon")
	}
}

//nolint:mnd,varnamelen // Three sparse native energy traces preserve the floor field beneath them.
func drawDungeonViewField(dst *ebiten.Image, x, y int) {
	colors := [...]stdcolor.Color{pkgcolor.EGALightMagenta, pkgcolor.EGALightCyan, pkgcolor.EGALightMagenta}
	for index, fieldColor := range colors {
		fieldX := float32(x + 4 + index*4)
		vector.StrokeLine(dst, fieldX, float32(y+2), fieldX-2, float32(y+6), 1, fieldColor, false)
		vector.StrokeLine(dst, fieldX-2, float32(y+6), fieldX, float32(y+10), 1, fieldColor, false)
		vector.StrokeLine(dst, fieldX, float32(y+10), fieldX-2, float32(y+14), 1, fieldColor, false)
	}
}

//nolint:mnd // Native 16x16 blueprint icon.
func drawDungeonViewRoom(dst *ebiten.Image, x, y int) {
	vector.StrokeRect(dst, float32(x)+3.5, float32(y)+3.5, 9, 9, 1, pkgcolor.EGAYellow, false)
	vector.FillRect(dst, float32(x+7), float32(y+7), 3, 3, pkgcolor.EGAWhite, false)
}

//nolint:mnd // Native 16x16 blueprint icon.
func drawDungeonViewPlaque(dst *ebiten.Image, x, y int) {
	vector.FillRect(dst, float32(x+5), float32(y+5), 6, 6, pkgcolor.EGABrown, false)
	vector.StrokeRect(dst, float32(x)+5.5, float32(y)+5.5, 5, 5, 1, pkgcolor.EGAWhite, false)
}

//nolint:mnd // Native 16x16 blueprint icon.
func drawDungeonViewSpecialWall(dst *ebiten.Image, x, y int) {
	vector.StrokeRect(dst, float32(x)+3.5, float32(y)+3.5, 9, 9, 1, pkgcolor.EGALightCyan, false)
	vector.FillRect(dst, float32(x+7), float32(y+7), 3, 3, pkgcolor.EGALightBlue, false)
}

func drawDungeonViewLine(dst *ebiten.Image, x, y, width int, color stdcolor.Color) {
	vector.FillRect(dst, float32(x), float32(y), float32(width), 1, color, false)
}

// Draw implements Widget.
func (d *DungeonViewDialog) Draw(screen *ebiten.Image) {
	d.rebuildLayout(screen.Bounds().Dx(), screen.Bounds().Dy())
	screen.DrawImage(d.buffer, d.drawOptions)
	DrawClassicPixelRoundedFrame(
		screen,
		d.bounds,
		getClassicUIPixelUnit(d.screenWidth, d.screenHeight),
		DialogRuleColor,
	)
}

// Update dismisses on one fresh Enter press after the invoking key has been released.
func (d *DungeonViewDialog) Update() {
	if d.keyboard == nil {
		return
	}
	if d.waitingForRelease {
		if d.keyboard.IsWaitingForRelease() {
			return
		}
		d.waitingForRelease = false
	}
	if d.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter) || d.keyboard.IsKeyPressedAndRegister(ebiten.KeyNumpadEnter) {
		d.dismissed = true
	}
}

// IsDismissed reports whether the modal may be removed.
func (d *DungeonViewDialog) IsDismissed() bool {
	return d.dismissed
}

// Dismiss allows workflow cleanup to close the dialog safely.
func (d *DungeonViewDialog) Dismiss() {
	d.dismissed = true
}
