package widgets

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

func TestDungeonViewCellPlansUseCartographicContracts(t *testing.T) {
	tests := []struct {
		name    string
		cell    references.DungeonCell
		surface dungeonViewSurface
		icon    dungeonViewIcon
	}{
		{name: "passage", cell: 0x00, surface: dungeonViewSurfaceFloor, icon: dungeonViewIconNone},
		{name: "ceiling pit", cell: 0x08, surface: dungeonViewSurfaceFloor, icon: dungeonViewIconCeilingPit},
		{name: "ladder up", cell: 0x10, surface: dungeonViewSurfaceFloor, icon: dungeonViewIconLadderUp},
		{name: "ladder down", cell: 0x20, surface: dungeonViewSurfaceFloor, icon: dungeonViewIconLadderDown},
		{name: "ladder both", cell: 0x30, surface: dungeonViewSurfaceFloor, icon: dungeonViewIconLadderBoth},
		{name: "chest", cell: 0x4f, surface: dungeonViewSurfaceFloor, icon: dungeonViewIconChest},
		{name: "fountain", cell: 0x50, surface: dungeonViewSurfaceFloor, icon: dungeonViewIconFountain},
		{name: "trap", cell: 0x62, surface: dungeonViewSurfaceFloor, icon: dungeonViewIconTrap},
		{name: "field", cell: 0x83, surface: dungeonViewSurfaceFloor, icon: dungeonViewIconField},
		{name: "room", cell: 0xa2, surface: dungeonViewSurfaceFloor, icon: dungeonViewIconRoom},
		{name: "live room", cell: 0xff, surface: dungeonViewSurfaceFloor, icon: dungeonViewIconRoom},
		{name: "wall", cell: 0xb0, surface: dungeonViewSurfaceWall, icon: dungeonViewIconNone},
		{name: "special wall", cell: 0xc0, surface: dungeonViewSurfaceWall, icon: dungeonViewIconSpecialWall},
		{name: "hidden door", cell: 0xd0, surface: dungeonViewSurfaceWall, icon: dungeonViewIconHiddenDoor},
		{name: "door", cell: 0xe0, surface: dungeonViewSurfaceFloor, icon: dungeonViewIconDoor},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := dungeonViewCellPlanFor(test.cell)
			assert.Equal(t, test.surface, plan.surface)
			assert.Equal(t, test.icon, plan.icon)
		})
	}
}

func TestDungeonViewAtlasCollapsesVisibleOccurrencesUniquely(t *testing.T) {
	model := game_state.DungeonViewModel{}
	setDungeonViewDisplayCell(&model, 4, 5, 2, 3, 0x4f, false)
	setDungeonViewDisplayCell(&model, 12, 13, 2, 3, 0x4f, true)
	setDungeonViewDisplayCell(&model, 6, 7, 6, 1, 0x50, false)

	plan := buildDungeonViewRenderPlan(model)

	assert.True(t, plan.cells[3][2].visible)
	assert.True(t, plan.cells[3][2].player)
	assert.Equal(t, dungeonViewIconChest, plan.cells[3][2].icon)
	assert.Equal(t, dungeonViewIconFountain, plan.cells[1][6].icon)
	assert.Equal(t, 2, visibleDungeonViewAtlasCells(plan))
}

func TestDungeonViewAtlasPlacementIsStableBySourceCoordinate(t *testing.T) {
	first := game_state.DungeonViewModel{}
	second := game_state.DungeonViewModel{}
	setDungeonViewDisplayCell(&first, 3, 4, 5, 2, 0x30, false)
	setDungeonViewDisplayCell(&second, 17, 18, 5, 2, 0x30, false)

	firstPlan := buildDungeonViewRenderPlan(first)
	secondPlan := buildDungeonViewRenderPlan(second)

	assert.Equal(t, firstPlan.cells, secondPlan.cells)
	originX, originY := dungeonViewCellOrigin(5, 2)
	assert.Equal(t, dungeonViewGridOrigin+5*dungeonViewCellSize, originX)
	assert.Equal(t, dungeonViewGridOrigin+2*dungeonViewCellSize, originY)
}

func TestDungeonViewAtlasIgnoresInvisibleRawCells(t *testing.T) {
	model := game_state.DungeonViewModel{}
	setDungeonViewDisplayCell(&model, 10, 10, 1, 1, 0x00, false)
	model.Cells[10][11] = game_state.DungeonViewCell{
		Visible: false, SourcePosition: references.Position{X: 2, Y: 1}, SourceCell: 0xb0,
	}

	plan := buildDungeonViewRenderPlan(model)

	assert.True(t, plan.cells[1][1].visible)
	assert.False(t, plan.cells[1][2].visible)
	assert.False(t, plan.edges.east[1][1])
}

func TestDungeonViewKnownEdgesRequireVisibleExpectedDisplayPairs(t *testing.T) {
	model := game_state.DungeonViewModel{}
	setDungeonViewDisplayCell(&model, 10, 10, 7, 3, 0x00, false)
	setDungeonViewDisplayCell(&model, 11, 10, 0, 3, 0x00, false)
	setDungeonViewDisplayCell(&model, 14, 10, 4, 7, 0x00, false)
	setDungeonViewDisplayCell(&model, 14, 11, 4, 0, 0x00, false)
	setDungeonViewDisplayCell(&model, 17, 10, 2, 5, 0x00, false)
	setDungeonViewDisplayCell(&model, 18, 10, 4, 5, 0x00, false)

	plan := buildDungeonViewRenderPlan(model)

	assert.True(t, plan.edges.east[3][7])
	assert.True(t, plan.edges.south[7][4])
	assert.False(t, plan.edges.east[5][2])
}

func TestDungeonViewTopologyUsesOnlyKnownCollapsedEdges(t *testing.T) {
	unknown := game_state.DungeonViewModel{}
	setDungeonViewDisplayCell(&unknown, 3, 3, 7, 4, 0xb0, false)
	setDungeonViewDisplayCell(&unknown, 15, 15, 0, 4, 0xb0, false)
	unknownPlan := buildDungeonViewRenderPlan(unknown)

	known := game_state.DungeonViewModel{}
	setDungeonViewDisplayCell(&known, 10, 10, 7, 4, 0xb0, false)
	setDungeonViewDisplayCell(&known, 11, 10, 0, 4, 0xb0, false)
	knownPlan := buildDungeonViewRenderPlan(known)

	assert.Zero(t, unknownPlan.cells[4][7].topology)
	assert.Zero(t, unknownPlan.cells[4][0].topology)
	assert.Equal(t, dungeonViewEast, knownPlan.cells[4][7].topology)
	assert.Equal(t, dungeonViewWest, knownPlan.cells[4][0].topology)
}

func TestDungeonViewDoorDoesNotRenderUnknownConnections(t *testing.T) {
	model := game_state.DungeonViewModel{}
	setDungeonViewDisplayCell(&model, 10, 10, 3, 3, 0xe0, false)

	plan := buildDungeonViewRenderPlan(model)

	assert.Equal(t, dungeonViewAxisHorizontal, plan.cells[3][3].doorAxis)
	assert.Zero(t, plan.cells[3][3].topology)
}

func TestDungeonViewContinuationRunesHaveStableSourceIdentities(t *testing.T) {
	assert.Equal(t, int('A'), dungeonViewContinuationRune(dungeonViewAxisHorizontal, 0))
	assert.Equal(t, int('H'), dungeonViewContinuationRune(dungeonViewAxisHorizontal, 7))
	assert.Equal(t, int('I'), dungeonViewContinuationRune(dungeonViewAxisVertical, 0))
	assert.Equal(t, int('P'), dungeonViewContinuationRune(dungeonViewAxisVertical, 7))
}

func TestDungeonViewContinuationRequiresKnownTraversableEndpoints(t *testing.T) {
	known := game_state.DungeonViewModel{}
	setDungeonViewDisplayCell(&known, 10, 10, 7, 3, 0x00, false)
	setDungeonViewDisplayCell(&known, 11, 10, 0, 3, 0xe0, false)
	assert.True(t, dungeonViewHasContinuation(buildDungeonViewRenderPlan(known), dungeonViewAxisHorizontal, 3))

	unknown := game_state.DungeonViewModel{}
	setDungeonViewDisplayCell(&unknown, 3, 3, 7, 3, 0x00, false)
	setDungeonViewDisplayCell(&unknown, 15, 15, 0, 3, 0x00, false)
	assert.False(t, dungeonViewHasContinuation(buildDungeonViewRenderPlan(unknown), dungeonViewAxisHorizontal, 3))

	blocked := game_state.DungeonViewModel{}
	setDungeonViewDisplayCell(&blocked, 10, 10, 7, 3, 0xb0, false)
	setDungeonViewDisplayCell(&blocked, 11, 10, 0, 3, 0x00, false)
	assert.False(t, dungeonViewHasContinuation(buildDungeonViewRenderPlan(blocked), dungeonViewAxisHorizontal, 3))
}

func TestDungeonViewWideLayoutCentersMapAndTwoColumnLegend(t *testing.T) {
	bounds := image.Rect(0, 0, 1200, 622)
	layout := calculateDungeonViewLayout(bounds, 1280, 720)
	groupCenter := float64(layout.mapArea.Min.X+layout.legend.Max.X) / 2

	assert.False(t, layout.legendBelow)
	assert.Equal(t, 4, layout.mapScale)
	assert.Equal(t, dungeonViewNativeSize*layout.mapScale, layout.mapArea.Dx())
	assert.Equal(t, dungeonViewWideLegendWidth(layout.mapScale, 1280), layout.legend.Dx())
	assert.Equal(t, dungeonViewLegendRowCount*dungeonViewLegendPhysicalIconSize(layout), layout.legend.Dy())
	assert.InDelta(t, float64(bounds.Dx())/2, groupCenter, 1)
	assert.False(t, layout.mapArea.Overlaps(layout.legend))
}

func TestDungeonViewWideLegendLabelWidthScalesAtHighResolution(t *testing.T) {
	assert.Equal(t, 508, dungeonViewWideLegendWidth(4, 1280))
	assert.Equal(t, 612, dungeonViewWideLegendWidth(5, 1920))
	assert.Equal(t, 820, dungeonViewWideLegendWidth(7, 2560))
}

func TestDungeonViewWideHeaderAndFooterScaleAt4K(t *testing.T) {
	layout := calculateDungeonViewLayout(image.Rect(0, 0, 3600, 1980), 3840, 2160)

	assert.Equal(t, 11, layout.mapScale)
	assert.GreaterOrEqual(t, layout.header.Dy(), 44)
	assert.GreaterOrEqual(t, layout.footer.Dy(), 44)
	assert.False(t, layout.header.Overlaps(layout.mapArea))
	assert.False(t, layout.footer.Overlaps(layout.mapArea))
	assert.False(t, layout.mapArea.Overlaps(layout.legend))
}

func TestDungeonViewNarrowLayoutKeepsTwoColumnLegendBelowMap(t *testing.T) {
	bounds := image.Rect(0, 0, 300, 500)
	layout := calculateDungeonViewLayout(bounds, 320, 560)

	assert.True(t, layout.legendBelow)
	assert.Equal(t, 1, layout.mapScale)
	assert.Equal(t, layout.mapArea.Max.Y+dungeonViewNarrowGap, layout.legend.Min.Y)
	assert.Equal(t, dungeonViewLegendRowCount*dungeonViewLegendPhysicalIconSize(layout), layout.legend.Dy())
	assert.False(t, layout.mapArea.Overlaps(layout.legend))
}

func TestDungeonViewIntegerScaleMatchesSupportedResolutions(t *testing.T) {
	tests := []struct {
		width     int
		height    int
		area      image.Rectangle
		wantScale int
	}{
		{width: 1280, height: 720, area: image.Rect(0, 0, 580, 580), wantScale: 4},
		{width: 1920, height: 1080, area: image.Rect(0, 0, 725, 725), wantScale: 5},
		{width: 2560, height: 1440, area: image.Rect(0, 0, 1015, 1015), wantScale: 7},
	}
	for _, test := range tests {
		assert.Equal(t, test.wantScale, dungeonViewMapScaleForArea(test.area, test.width, test.height))
	}
}

func TestDungeonViewLegendIconsAreSeventyFivePercentOfMapCells(t *testing.T) {
	layout := dungeonViewLayout{mapScale: 5}

	assert.Equal(t, 60, dungeonViewLegendPhysicalIconSize(layout))
	assert.Equal(t, dungeonViewLegendIconSize, dungeonViewLegendNativeIcon(dungeonViewLegend[0]).Bounds().Dx())
	assert.Equal(t, dungeonViewLegendIconSize, dungeonViewLegendNativeIcon(dungeonViewLegend[0]).Bounds().Dy())
}

func TestDungeonViewLegendUsesNativeTwelvePixelLadders(t *testing.T) {
	for _, entry := range dungeonViewLegend {
		icon := dungeonViewLegendNativeIcon(entry)
		assert.Equal(t, image.Pt(dungeonViewLegendIconSize, dungeonViewLegendIconSize), icon.Bounds().Size())
	}
	ladder := dungeonViewLegendNativeIcon(dungeonViewLegend[9])
	pixels := dungeonViewImagePixels(ladder)

	assert.NotZero(t, dungeonViewPixelAlpha(pixels, dungeonViewLegendIconSize, 3, 2))
	assert.NotZero(t, dungeonViewPixelAlpha(pixels, dungeonViewLegendIconSize, 8, 2))
	assert.NotZero(t, dungeonViewPixelAlpha(pixels, dungeonViewLegendIconSize, 3, 3))
	assert.NotZero(t, dungeonViewPixelAlpha(pixels, dungeonViewLegendIconSize, 8, 9))
	assert.NotZero(t, dungeonViewPixelAlpha(pixels, dungeonViewLegendIconSize, 6, 10))
}

func TestDungeonViewLegendContinuationRuneFitsNativeCanvas(t *testing.T) {
	icon := dungeonViewLegendNativeIcon(dungeonViewLegend[len(dungeonViewLegend)-1])
	pixels := dungeonViewImagePixels(icon)

	assert.NotZero(t, dungeonViewPixelAlpha(pixels, dungeonViewLegendIconSize, 5, 5))
}

func TestDungeonViewLegendSpecialWallAndHiddenPitUseDistinctNativeSymbols(t *testing.T) {
	specialWall := dungeonViewLegendNativeIcon(dungeonViewLegend[4])
	hiddenPit := dungeonViewLegendNativeIcon(dungeonViewLegend[14])
	specialPixels := dungeonViewImagePixels(specialWall)
	hiddenPitPixels := dungeonViewImagePixels(hiddenPit)

	assert.NotZero(t, dungeonViewPixelAlpha(specialPixels, dungeonViewLegendIconSize, 5, 2))
	assert.NotZero(t, dungeonViewPixelAlpha(hiddenPitPixels, dungeonViewLegendIconSize, 3, 9))
	assert.NotEqual(t, specialPixels, hiddenPitPixels)
}

func TestDungeonViewLegendIncludesSpecialWallAndPitVariants(t *testing.T) {
	require.Len(t, dungeonViewLegend, 18)
	assert.Equal(t, dungeonViewLegendRowCount*dungeonViewLegendColumnCount, len(dungeonViewLegend))
	assert.Equal(t, "Wall writing", dungeonViewLegend[3].label)
	assert.Equal(t, references.DungeonCell(0xb1), dungeonViewLegend[3].cell)
	assert.Equal(t, "Special wall", dungeonViewLegend[4].label)
	assert.Equal(t, references.DungeonCell(0xc0), dungeonViewLegend[4].cell)
	assert.Equal(t, "Pit / hole", dungeonViewLegend[13].label)
	assert.Equal(t, references.DungeonCell(0x60), dungeonViewLegend[13].cell)
	assert.Equal(t, "Hidden pit", dungeonViewLegend[14].label)
	assert.Equal(t, references.DungeonCell(0x61), dungeonViewLegend[14].cell)
	assert.Equal(t, "Trap / bomb", dungeonViewLegend[15].label)
	assert.Equal(t, references.DungeonCell(0x62), dungeonViewLegend[15].cell)
	assert.Equal(t, "Continuation", dungeonViewLegend[17].label)
	assert.True(t, dungeonViewLegend[17].continuation)
}

func TestDungeonViewLegendUsesConciseNarrowLabelsWithoutChangingTypography(t *testing.T) {
	assert.Equal(t, "Wall writing", dungeonViewLegendLabel(dungeonViewLegend[3], false))
	assert.Equal(t, "Writing", dungeonViewLegendLabel(dungeonViewLegend[3], true))
	assert.Equal(t, "Special", dungeonViewLegendLabel(dungeonViewLegend[4], true))
	assert.Equal(t, "Secret", dungeonViewLegendLabel(dungeonViewLegend[6], true))
}

func TestDungeonViewFooterExplainsMatchingMarksWithoutCellCount(t *testing.T) {
	assert.Equal(t, []string{"MATCHING EDGE MARKS CONNECT     PRESS ENTER TO RETURN"}, dungeonViewFooterLines(false))
	assert.Equal(t, []string{"MATCHING EDGE MARKS CONNECT", "PRESS ENTER TO RETURN"}, dungeonViewFooterLines(true))
}

func TestDungeonViewConnectedWallUsesThreeNativePixelStroke(t *testing.T) {
	icon := ebiten.NewImage(dungeonViewCellSize, dungeonViewCellSize)
	drawDungeonViewConnectedWall(icon, 0, 0, dungeonViewEast|dungeonViewWest)
	pixels := dungeonViewImagePixels(icon)

	assert.Zero(t, dungeonViewPixelAlpha(pixels, dungeonViewCellSize, 0, 6))
	assert.NotZero(t, dungeonViewPixelAlpha(pixels, dungeonViewCellSize, 0, 7))
	assert.NotZero(t, dungeonViewPixelAlpha(pixels, dungeonViewCellSize, 0, 8))
	assert.NotZero(t, dungeonViewPixelAlpha(pixels, dungeonViewCellSize, 0, 9))
	assert.Zero(t, dungeonViewPixelAlpha(pixels, dungeonViewCellSize, 0, 10))
}

func TestDungeonViewRightContinuationHasOneNativePixelOfPadding(t *testing.T) {
	atlasBounds := dungeonViewAtlasBounds()
	runeRight := atlasBounds.Max.X + dungeonViewRightMarkerPad + 8

	assert.Equal(t, 1, dungeonViewRightMarkerPad)
	assert.LessOrEqual(t, runeRight, dungeonViewNativeSize)
}

func TestDungeonViewChestUsesTrueSixteenPixelCanvas(t *testing.T) {
	icon := ebiten.NewImage(dungeonViewCellSize, dungeonViewCellSize)
	drawDungeonViewChest(icon, 0, 0)
	pixels := dungeonViewImagePixels(icon)

	assert.NotZero(t, dungeonViewPixelAlpha(pixels, dungeonViewCellSize, 12, 8))
	assert.Zero(t, dungeonViewPixelAlpha(pixels, dungeonViewCellSize, 14, 8))
}

func TestDungeonViewPlayerSegmentsFollowFacing(t *testing.T) {
	tests := []struct {
		facing references.DungeonFacing
		tipX   float32
		tipY   float32
	}{
		{facing: references.DungeonFacingNorth, tipX: 8, tipY: 2},
		{facing: references.DungeonFacingEast, tipX: 14, tipY: 8},
		{facing: references.DungeonFacingSouth, tipX: 8, tipY: 14},
		{facing: references.DungeonFacingWest, tipX: 2, tipY: 8},
	}
	for _, test := range tests {
		segments := dungeonViewPlayerSegments(test.facing)
		assert.Len(t, segments, 3)
		assert.Equal(t, test.tipX, segments[0].toX)
		assert.Equal(t, test.tipY, segments[0].toY)
	}
}

func TestDungeonViewDialogRebuildsWithReadableNarrowTypography(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	dialog := NewDungeonViewDialog(DungeonViewDialogConfig{
		Placement: sprites.PercentBasedPlacement{
			StartPercentX: 0.03, EndPercentX: 0.97, StartPercentY: 0.04, EndPercentY: 0.96,
		},
		Keyboard: nil,
		Model:    game_state.DungeonViewModel{},
	})

	setResponsiveDialogTestResolution(t, 320, 560)
	dialog.rebuildLayout(320, 560)

	assert.True(t, dialog.layout.legendBelow)
	assert.GreaterOrEqual(t, dialog.titleOutput.Font.TextFace.Size, float64(dungeonViewMinimumTextSize))
	assert.Equal(t, dialog.titleOutput.Font.TextFace.Size, dialog.legendOutputs[0].Font.TextFace.Size)
	assert.GreaterOrEqual(t, dialog.footerOutput.Font.TextFace.Size, float64(dungeonViewMinimumTextSize))
}

func TestDungeonViewDialogDismissesOnlyOnEnter(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	dialog := &DungeonViewDialog{keyboard: keyboard}
	keyboard.SimulateKeyPress(ebiten.KeyPeriod)
	dialog.Update()
	assert.False(t, dialog.IsDismissed())

	keyboard.SimulateKeyPress(ebiten.KeyEnter)
	dialog.Update()
	assert.True(t, dialog.IsDismissed())
	dialog.Dismiss()
	assert.True(t, dialog.IsDismissed())
}

func setDungeonViewDisplayCell(
	model *game_state.DungeonViewModel,
	displayX int,
	displayY int,
	sourceX int,
	sourceY int,
	cell references.DungeonCell,
	player bool,
) {
	model.Cells[displayY][displayX] = game_state.DungeonViewCell{
		Visible: true,
		Player:  player,
		SourcePosition: references.Position{
			X: references.Coordinate(sourceX), Y: references.Coordinate(sourceY),
		},
		SourceCell: cell,
	}
}

func visibleDungeonViewAtlasCells(plan dungeonViewRenderPlan) int {
	count := 0
	for sourceY := range dungeonViewFloorCells {
		for sourceX := range dungeonViewFloorCells {
			if plan.cells[sourceY][sourceX].visible {
				count++
			}
		}
	}

	return count
}

func dungeonViewImagePixels(img *ebiten.Image) []byte {
	pixels := make([]byte, img.Bounds().Dx()*img.Bounds().Dy()*4)
	img.ReadPixels(pixels)

	return pixels
}

func dungeonViewPixelAlpha(pixels []byte, width, x, y int) byte {
	return pixels[(y*width+x)*4+3]
}
