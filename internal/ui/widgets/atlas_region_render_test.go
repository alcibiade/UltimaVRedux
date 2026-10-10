package widgets

import (
	"fmt"
	"image"
	"math"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestAtlasRenderingRegionBalancedNamesAndCompleteOutput(t *testing.T) {
	dialog := atlasRenderingDialog()
	screen := ebiten.NewImage(528, 528)
	t.Cleanup(screen.Deallocate)
	for _, test := range []struct {
		name  string
		lines []string
	}{
		{"Spiritwood", []string{"Spiritwood"}},
		{"Valorian Isles", []string{"Valorian", "Isles"}},
		{"Dagger Isle", []string{"Dagger", "Isle"}},
		{"Verity Isle", []string{"Verity", "Isle"}},
		{"Isle of the Avatar", []string{"Isle of", "the Avatar"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			lines, rect := dialog.regionLabelLayout(test.name, image.Pt(264, 264), image.Rectangle{})
			require.Equal(t, test.lines, lines)
			dialog.drawRegionLabel(screen, lines, rect)
			require.Equal(t, len(lines), dialog.label.GetPanelContentLineCount(), "no implicit wrapping or truncation")
			require.Equal(t, strings.Join(lines, " \n"), dialog.label.GetOutputStr(false))
			require.NotContains(t, dialog.label.GetOutputStr(false), "...")
		})
	}
}

func TestAtlasRenderingRegionHeightUsesOutputPitchAndSkipsUnavailableSpace(t *testing.T) {
	dialog := atlasRenderingDialog()
	dialog.label = text.NewOutput(dialog.font, 37.5, 4, 128)
	lines, rect := dialog.regionLabelLayout("Isle of the Avatar", image.Pt(264, 264), image.Rectangle{})
	require.Len(t, lines, 2)
	require.Equal(t, int(math.Ceil(dialog.label.GetTotalHeightForLines(2))), rect.Dy())
	require.Greater(t, rect.Dy(), int(math.Ceil(dialog.label.GetFontHeight()*2)), "line pitch belongs to Output")
	for _, pane := range []image.Rectangle{
		image.Rect(0, 0, rect.Dx()+3, 300),
		image.Rect(0, 0, 300, rect.Dy()+3),
	} {
		dialog.mapRect = pane
		absentLines, absentRect := dialog.regionLabelLayout("Isle of the Avatar", pane.Min, image.Rectangle{})
		require.Empty(t, absentLines)
		require.True(t, absentRect.Empty(), "skip a whole block rather than clipping any portion")
	}
	dialog.mapRect = image.Rect(8, 8, 520, 520)
	_, absent := dialog.regionLabelLayout("Isle of the Avatar", image.Pt(264, 264), rect)
	require.True(t, absent.Empty(), "whole block yields to the selected plate")
	_, absent = dialog.regionLabelLayout("Isle of the Avatar", image.Pt(264, 264), rect.Add(image.Pt(0, rect.Dy()-1)))
	require.True(t, absent.Empty(), "second-line overlap also yields to the selected plate")
}

// All real anchors are exercised at every zoom. Zoomed edge cameras deliberately
// cross the world seam: decoration may move inward, but camera and anchors cannot.
func TestAtlasRenderingRegionBlocksInsidePaneAtEverySizeZoomAndEdge(t *testing.T) {
	for _, resolution := range config.SupportedWindowResolutions() {
		t.Run(fmt.Sprintf("%dx%d", resolution.X, resolution.Y), func(t *testing.T) {
			setQuestJournalTestResolution(t, resolution.X, resolution.Y)
			dialog := atlasRenderingDialog()
			dialog.ensureLayout()
			for _, zoom := range []int{1, 2, 4} {
				dialog.view.Zoom = zoom
				for _, region := range references.AtlasRegionLabels() {
					for _, fraction := range []image.Point{
						{X: 50, Y: 50}, {X: 0, Y: 50}, {X: 99, Y: 50},
						{X: 50, Y: 0}, {X: 50, Y: 99}, {X: 0, Y: 0}, {X: 99, Y: 99},
					} {
						dialog.view.CenterX = atlasModulo(float64(region.X) + dialog.view.span()*(.5-float64(fraction.X)/100))
						dialog.view.CenterY = atlasModulo(float64(region.Y) + dialog.view.span()*(.5-float64(fraction.Y)/100))
						beforeView := *dialog.view
						x, y, visible := dialog.view.project(float64(region.X), float64(region.Y), dialog.mapRect)
						require.True(t, visible)
						anchor := image.Pt(int(x), int(y))
						lines, rect := dialog.regionLabelLayout(region.Name, anchor, image.Rectangle{})
						require.False(t, rect.Empty(), "%s zoom %d edge %v", region.Name, zoom, fraction)
						require.True(t, rect.In(dialog.mapRect.Inset(2)), "complete block must fit inside pane margins")
						require.LessOrEqual(t, len(lines), 2)
						require.Equal(t, region.Name, strings.Join(lines, " "), "proper name is preserved, including singular Isle")
						require.Equal(t, int(math.Ceil(dialog.label.GetTotalHeightForLines(len(lines)))), rect.Dy())
						widest := 0.0
						for _, line := range lines {
							width, _ := dialog.font.MeasureString(line)
							widest = max(widest, width)
						}
						require.Equal(t, int(math.Ceil(widest)), rect.Dx())
						secondLines, secondRect := dialog.regionLabelLayout(region.Name, anchor, image.Rectangle{})
						require.Equal(t, lines, secondLines)
						require.Equal(t, rect, secondRect)
						require.Equal(t, beforeView, *dialog.view, "decorative layout cannot move the actual map")
						_, blocked := dialog.regionLabelLayout(region.Name, anchor, rect)
						require.True(t, blocked.Empty())
					}
				}
			}
		})
	}
}

func TestAtlasRenderingRegionRasterCompleteUnboxedAtPaneEdges(t *testing.T) {
	for _, resolution := range config.SupportedWindowResolutions() {
		t.Run(fmt.Sprintf("%dx%d", resolution.X, resolution.Y), func(t *testing.T) {
			setQuestJournalTestResolution(t, resolution.X, resolution.Y)
			dialog := atlasRenderingDialog()
			dialog.ensureLayout()
			for _, region := range references.AtlasRegionLabels() {
				for _, anchor := range []image.Point{dialog.mapRect.Min, dialog.mapRect.Max.Sub(image.Pt(1, 1))} {
					lines, rect := dialog.regionLabelLayout(region.Name, anchor, image.Rectangle{})
					// Absolute coordinates preserve the real pane-edge clip while
					// allocating only the measured label and its outside guard pixels.
					screen := ebiten.NewImageWithOptions(rect.Inset(-2), nil)
					capture := atlasSubImage(screen, rect.Inset(-2))
					capture.Clear()
					dialog.drawRegionLabel(screen, lines, rect)
					unclipped := atlasMarkerPixels(capture)
					capture.Clear()
					dialog.drawRegionLabel(atlasSubImage(screen, dialog.mapRect), lines, rect)
					clipped := atlasMarkerPixels(capture)
					require.Equal(t, unclipped.Pix, clipped.Pix, "%s: clipping must never remove letter pixels", region.Name)
					require.Equal(t, len(lines), dialog.label.GetPanelContentLineCount())
					require.Equal(t, strings.Join(lines, " \n"), dialog.label.GetOutputStr(false))
					painted := 0
					for y := unclipped.Rect.Min.Y; y < unclipped.Rect.Max.Y; y++ {
						for x := unclipped.Rect.Min.X; x < unclipped.Rect.Max.X; x++ {
							pixel := unclipped.RGBAAt(x, y)
							if pixel.A == 0 {
								continue
							}
							painted++
							require.True(t, image.Pt(x, y).In(rect), "%s raster outside measured block at %d,%d", region.Name, x, y)
							require.Equal(t, pixel.R, pixel.G)
							require.Equal(t, pixel.G, pixel.B)
							require.NotEqual(t, ucolor.EGABlack.ToRGBA(), pixel, "no backing box, leader, point or outline")
						}
					}
					require.Positive(t, painted)
					require.Less(t, painted, rect.Dx()*rect.Dy()/2)
					for index := range lines {
						lineTop := rect.Min.Y + int(math.Floor(float64(index)*dialog.label.GetLineSpacing()))
						lineBottom := min(rect.Max.Y, lineTop+int(math.Ceil(dialog.label.GetFontHeight())))
						lineInk := 0
						for y := lineTop; y < lineBottom; y++ {
							for x := rect.Min.X; x < rect.Max.X; x++ {
								if unclipped.RGBAAt(x, y).A > 0 {
									lineInk++
								}
							}
						}
						require.Positive(t, lineInk, "every complete line must be visible")
					}
					screen.Dispose()
				}
			}
		})
	}
}
