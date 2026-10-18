package main

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gametext "github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/mainscreen"
)

func TestRightSideOutputRectangleReservesFourSuppliesRows(t *testing.T) {
	for _, test := range []struct {
		size      image.Point
		output    image.Rectangle
		provision image.Rectangle
	}{
		{image.Pt(1280, 720), image.Rect(985, 417, 1267, 601), image.Rect(985, 622, 1267, 702)},
		{image.Pt(1366, 768), image.Rect(1051, 445, 1351, 642), image.Rect(1051, 664, 1351, 748)},
		{image.Pt(1920, 1080), image.Rect(1478, 626, 1900, 905), image.Rect(1478, 937, 1900, 1053)},
		{image.Pt(2560, 1440), image.Rect(1971, 835, 2534, 1206), image.Rect(1971, 1248, 2534, 1404)},
		{image.Pt(3840, 2160), image.Rect(2956, 1252, 3801, 1811), image.Rect(2956, 1874, 3801, 2106)},
	} {
		t.Run(test.size.String(), func(t *testing.T) {
			bounds := logicalGameBounds(test.size.X, test.size.Y)
			layout := calculateRightRailLayout(bounds)
			assert.Equal(t, test.output, layout.outputContent)
			assert.Equal(t, test.provision, layout.provisionContent)
			_, output := newMainOutput(bounds.Dx())
			assert.Equal(t, 11, output.GetLineCapacity(layout.outputContent.Dy()))
			assert.LessOrEqual(t, output.GetTotalHeightForLines(11), float64(layout.outputContent.Dy()))
			assert.Greater(t, output.GetTotalHeightForLines(12), float64(layout.outputContent.Dy()))
			scene := &GameScene{output: output}
			scene.syncRightRailLayout(bounds)
			t.Cleanup(scene.outputClipImage.Deallocate)
			assert.Equal(t, test.output.Size(), scene.outputClipImage.Bounds().Size(), "the actual transcript clip must shrink too")
			assert.Equal(t, 11, scene.output.GetLineCapacity(scene.outputClipImage.Bounds().Dy()))
		})
	}
}

func TestStatusBandDoesNotMoveOtherRailOrMapRegions(t *testing.T) {
	for _, size := range []image.Point{{1280, 720}, {1366, 768}, {1920, 1080}, {2560, 1440}, {3840, 2160}} {
		bounds := logicalGameBounds(size.X, size.Y)
		layout := calculateRightRailLayout(bounds)
		assert.Equal(t, percentRect(bounds, 0.751, 0, 1, 0.50), layout.characterBorder)
		assert.Equal(t, percentRect(bounds, 0.77, 0.025, 0.99, 0.48), layout.characterContent)
		assert.Equal(t, percentRect(bounds, 0.751, 0.485, 1, 0.575), layout.activeBorder)
		oldOutput := percentRect(bounds, 0.77, 0.580, 0.99, 0.886)
		oldProvision := percentRect(bounds, 0.77, 0.915, 0.99, provisionContentEnd)
		growth := mainscreen.ProvisionSummaryHeight(bounds.Dx()) - oldProvision.Dy()
		assert.Equal(t, oldOutput.Dy()-growth, layout.outputContent.Dy())
		assert.Equal(t, mainscreen.ProvisionSummaryHeight(bounds.Dx()), layout.provisionContent.Dy())
		assert.Equal(t, oldProvision.Max.Y, layout.provisionContent.Max.Y, "supplies only grow upward")
		assert.Equal(t, oldOutput.Min, layout.outputContent.Min, "message top and horizontal geometry stay fixed")
		assert.Equal(t, int(float64(size.Y)*outputPanelEndY)-growth, layout.outputBorder.Max.Y)
		assert.Equal(t, int(float64(size.Y)*provisionPanelY)-growth, layout.provisionBorder.Min.Y)
		_, output := newMainOutput(bounds.Dx())
		assert.Equal(t, 11, output.GetLineCapacity(layout.outputContent.Dy()))
	}
	assert.Equal(t, 0.0, gameScreenPercents.StartPercentX)
	assert.Equal(t, 0.76, gameScreenPercents.EndPercentX)
	assert.Equal(t, 0.0, gameScreenPercents.StartPercentY)
	assert.Equal(t, 1.0, gameScreenPercents.EndPercentY)
	assert.Equal(t, 16, maxCharsPerLine)
}

func TestRightRailLayoutUsesLogicalBoundsOnNonWidescreenDisplays(t *testing.T) {
	ultrawide := logicalGameBounds(3440, 1440)
	assert.Equal(t, image.Rect(0, 0, 2560, 1440), ultrawide)
	standard := logicalGameBounds(1600, 1200)
	assert.Equal(t, image.Rect(0, 0, 1600, 900), standard)

	layout := calculateRightRailLayout(ultrawide)

	assert.True(t, layout.railBorder.In(ultrawide))
	assert.True(t, layout.characterContent.In(ultrawide))
	assert.True(t, layout.activeBorder.In(ultrawide))
	assert.True(t, layout.outputContent.In(ultrawide))
	assert.True(t, layout.provisionContent.In(ultrawide))
	font, output := newMainOutput(ultrawide.Dx())
	assert.NotNil(t, font)
	assert.Equal(t, visibleLinesForOutput, output.GetLineCapacity(layout.outputContent.Dy()))
}

func TestLogicalGameBoundsAlwaysReturnsPositiveSixteenByNineSurface(t *testing.T) {
	for _, test := range []struct {
		name     string
		width    int
		height   int
		expected image.Rectangle
	}{
		{name: "exact", width: 1280, height: 720, expected: image.Rect(0, 0, 1280, 720)},
		{name: "sixteen ten", width: 2560, height: 1600, expected: image.Rect(0, 0, 2560, 1440)},
		{name: "four three", width: 1600, height: 1200, expected: image.Rect(0, 0, 1600, 900)},
		{name: "ultrawide", width: 3440, height: 1440, expected: image.Rect(0, 0, 2560, 1440)},
		{name: "odd", width: 1366, height: 768, expected: image.Rect(0, 0, 1365, 768)},
		{name: "tiny", width: 1, height: 1, expected: image.Rect(0, 0, 1280, 720)},
		{name: "invalid", width: 0, height: -1, expected: image.Rect(0, 0, 1280, 720)},
	} {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, logicalGameBounds(test.width, test.height))
		})
	}
}

func TestFinalScreenGeometryCentersUniformAspectFit(t *testing.T) {
	for _, test := range []struct {
		name        string
		destination image.Rectangle
		source      image.Rectangle
		expectedX   float64
		expectedY   float64
		expectedMax image.Point
	}{
		{
			name: "ultrawide pillarbox", destination: image.Rect(0, 0, 3440, 1440),
			source: image.Rect(0, 0, 2560, 1440), expectedX: 440, expectedMax: image.Pt(3000, 1440),
		},
		{
			name: "four three letterbox", destination: image.Rect(0, 0, 1600, 1200),
			source: image.Rect(0, 0, 1600, 900), expectedY: 150, expectedMax: image.Pt(1600, 1050),
		},
		{
			name: "odd remainder goes right", destination: image.Rect(0, 0, 1366, 768),
			source: image.Rect(0, 0, 1365, 768), expectedMax: image.Pt(1365, 768),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			geometry := finalScreenGeometry(test.destination, test.source)
			minimumX, minimumY := geometry.Apply(float64(test.source.Min.X), float64(test.source.Min.Y))
			maximumX, maximumY := geometry.Apply(float64(test.source.Max.X), float64(test.source.Max.Y))

			assert.InDelta(t, test.expectedX, minimumX, 0.001)
			assert.InDelta(t, test.expectedY, minimumY, 0.001)
			assert.InDelta(t, float64(test.expectedMax.X), maximumX, 0.001)
			assert.InDelta(t, float64(test.expectedMax.Y), maximumY, 0.001)
		})
	}
}

func TestMainOutputClipsLongTextToItsPanel(t *testing.T) {
	font := gametext.NewUltimaFont(12)
	output := gametext.NewOutput(font, 18, maxLinesForOutput, 100)
	output.AddText("This line is deliberately much wider than the output panel.")
	outputBounds := image.Rect(40, 20, 100, 60)
	scene := &GameScene{
		output:          output,
		outputClipImage: ebiten.NewImage(outputBounds.Dx(), outputBounds.Dy()),
		rightRail:       rightRailLayout{outputContent: outputBounds},
	}
	screen := ebiten.NewImage(160, 80)

	scene.drawMainOutput(screen)

	visibleInside := false
	for y := outputBounds.Min.Y; y < outputBounds.Max.Y; y++ {
		for x := outputBounds.Min.X; x < outputBounds.Max.X; x++ {
			_, _, _, alpha := screen.At(x, y).RGBA()
			visibleInside = visibleInside || alpha != 0
		}
	}
	assert.True(t, visibleInside)
	for y := range screen.Bounds().Dy() {
		for x := outputBounds.Max.X; x < screen.Bounds().Max.X; x++ {
			_, _, _, alpha := screen.At(x, y).RGBA()
			assert.Zero(t, alpha)
		}
	}
}

func TestRightSideOutputMatchesOriginalCharacterWidth(t *testing.T) {
	assert.Equal(t, 16, maxCharsPerLine)
}

func TestRightSideOutputKeepsExactWidthSearchPhraseTogether(t *testing.T) {
	_, output := newMainOutput(1280)

	output.AddText("Thou dost find nothing of note.")

	assert.Contains(t, output.GetOutputStr(false), "nothing of note.")
}

func TestRightRailLayoutContainsNonOverlappingContentAtSupportedResolutions(t *testing.T) {
	resolutions := []image.Point{
		{X: 1280, Y: 720},
		{X: 1366, Y: 768},
		{X: 1920, Y: 1080},
		{X: 2560, Y: 1440},
		{X: 3840, Y: 2160},
	}

	for _, resolution := range resolutions {
		t.Run(resolution.String(), func(t *testing.T) {
			bounds := logicalGameBounds(resolution.X, resolution.Y)
			layout := calculateRightRailLayout(bounds)
			require.True(t, layout.railBorder.In(bounds))
			status := calculateStatusBandLayout(layout.activeBorder)
			require.True(t, status.content.In(layout.activeBorder))
			assert.Greater(t, status.content.Min.X, layout.activeBorder.Min.X)
			assert.Greater(t, status.content.Min.Y, layout.activeBorder.Min.Y)
			assert.Less(t, status.content.Max.X, layout.activeBorder.Max.X)
			assert.Less(t, status.content.Max.Y, layout.activeBorder.Max.Y)
			require.True(t, layout.outputContent.In(layout.outputBorder))
			require.True(t, layout.provisionContent.In(layout.provisionBorder))
			assert.True(t, layout.activeBorder.Max.Y <= layout.outputContent.Min.Y)
			assert.True(t, layout.outputContent.Max.Y <= layout.provisionContent.Min.Y)

			scale := float64(bounds.Dx()) / 1920
			font := gametext.NewUltimaFont(float64(defaultOutputFontSize) * scale)
			output := gametext.NewOutput(
				font,
				float64(defaultOutputFontSize)*defaultOutputLineSpacingMult*scale,
				maxLinesForOutput,
				maxCharsPerLine,
			)
			assert.Equal(t, visibleLinesForOutput, output.GetLineCapacity(layout.outputContent.Dy()))
		})
	}
}
