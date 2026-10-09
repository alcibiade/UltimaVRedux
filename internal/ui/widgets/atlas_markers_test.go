package widgets

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestAtlasMarkersClassicBracketsStableAtOddAndEvenUnitsAndZoom(t *testing.T) {
	for _, size := range []image.Point{{X: 960, Y: 600}, {X: 1280, Y: 720}, {X: 1920, Y: 1080}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			dialog := atlasRenderingDialog()
			dialog.width, dialog.height = size.X, size.Y
			entry := journal.AtlasEntry{Kind: "castle", Charted: true, Survey: atlasMeasuredSite(128, 128)}
			for _, zoom := range []int{1, 2, 4} {
				dialog.view.Zoom = zoom
				center, visible := dialog.atlasMarkerCenter(entry)
				require.True(t, visible)
				bounds := dialog.selectionBounds(center)
				require.Equal(t, center.X-bounds.Min.X, bounds.Max.X-1-center.X)
				require.Equal(t, center.Y-bounds.Min.Y, bounds.Max.Y-1-center.Y)
				screen := ebiten.NewImage(528, 528)
				dialog.drawSelectionBrackets(screen, center)
				pixels := atlasMarkerPixels(screen)
				screen.Deallocate()
				yellow := color.RGBAModel.Convert(DialogLabelColor).(color.RGBA)
				unit := ClassicUIPixelUnit(size.X, size.Y)
				for y := bounds.Min.Y; y < bounds.Min.Y+unit; y++ {
					for x := bounds.Min.X; x < bounds.Min.X+2*unit; x++ {
						require.Equal(t, yellow, pixels.RGBAAt(x, y), "corner has complete unit-thick arms")
					}
				}
				require.Zero(t, pixels.RGBAAt(center.X, center.Y).A, "brackets must not obscure the actual symbol")
				require.Zero(t, pixels.RGBAAt(center.X, bounds.Min.Y).A, "open middle rules out a ring")
				require.Equal(t, ucolor.EGABlack.ToRGBA(), pixels.RGBAAt(bounds.Min.X-1, bounds.Min.Y))
				for _, pixel := range []color.RGBA{pixels.RGBAAt(bounds.Min.X, bounds.Min.Y),
					pixels.RGBAAt(bounds.Max.X-1, bounds.Max.Y-1)} {
					require.Equal(t, yellow, pixel, "all corners retain their yellow foreground over the underlay")
				}
			}
		})
	}
}

func atlasMarkerPixels(screen *ebiten.Image) *image.RGBA {
	pixels := image.NewRGBA(screen.Bounds())
	screen.ReadPixels(pixels.Pix)

	return pixels
}

func TestAtlasMarkersOutlineMatchesArtCellWithoutResizingBody(t *testing.T) {
	setQuestJournalTestResolution(t, 1280, 720)
	view := NewAtlasViewState()
	dialog := NewAtlasDialog(atlasCleanupDossier(), nil, newWidgetTestKeyboard(0), &view, func() {})
	unit := getClassicUIPixelUnit(1280, 720)
	require.Equal(t, 4, unit, "same logical pixel as standard message-box borders")
	require.Equal(t, 12, dialog.atlasSymbolCell(), "same 8px body plus two 2px artwork-cell outline edges")
	require.Equal(t, 25, dialog.selectionBounds(image.Pt(100, 100)).Dx(), "corners follow the reduced outline")
	require.LessOrEqual(t, dialog.directoryRowHeight(), 32, "compact rows must not retain the old 40px glyph reservation")
	screen := ebiten.NewImage(128, 128)
	t.Cleanup(screen.Deallocate)
	DrawClassicPixelRoundedFrame(screen, image.Rect(8, 8, 120, 40), unit, DialogRuleColor)
	dialog.drawAtlasMarker(screen, journal.AtlasEntry{Kind: "towne"}, image.Pt(64, 80), DialogRuleColor, true)
	pixels := atlasMarkerPixels(screen)
	ink := color.RGBAModel.Convert(DialogRuleColor)
	// The roof stays eight pixels across. Its outline is one artwork cell,
	// deliberately smaller than the four-pixel standard message-box border.
	for offset := range unit {
		require.Equal(t, ink, pixels.At(64, 8+offset))
	}
	for offset := range 2 {
		require.Equal(t, ucolor.EGABlack.ToRGBA(), pixels.RGBAAt(58+offset, 79))
		require.Equal(t, ucolor.EGABlack.ToRGBA(), pixels.RGBAAt(68+offset, 79))
	}
	for x := 60; x < 68; x++ {
		require.Equal(t, ink, pixels.At(x, 79), "fine-grid roof fits inside its 8px budget")
	}
	require.Zero(t, pixels.RGBAAt(57, 79).A, "outline is included in the total 12px budget")
	require.Zero(t, pixels.RGBAAt(70, 79).A)
}

func TestAtlasMarkersBoundedSilhouettesAndSharedMapSymbol(t *testing.T) {
	for _, unit := range []int{1, 3, 4, 5} {
		t.Run(fmt.Sprint(unit), func(t *testing.T) {
			dialog := atlasRenderingDialog()
			dialog.width, dialog.height = 320*unit, 200*unit
			center := image.Pt(264, 264)
			bodySize := max(5, 2*unit)
			outline := map[int]int{1: 1, 3: 1, 4: 2, 5: 2}[unit]
			cell := image.Rect(center.X-bodySize/2, center.Y-bodySize/2,
				center.X-bodySize/2+bodySize, center.Y-bodySize/2+bodySize)
			renders := make(map[string]*image.RGBA)
			ink := color.RGBAModel.Convert(DialogAtlasUndiscoveredColor).(color.RGBA)
			for _, kind := range []string{"castle", "stronghold", "keep", "towne", "village", "hut", "shrine", "ruins", "lighthouse", "dungeon"} {
				entry := journal.AtlasEntry{Kind: kind}
				screen := ebiten.NewImage(528, 528)
				dialog.drawAtlasSymbol(screen, entry, center, atlasPlaceColor(entry))
				pixels := atlasMarkerPixels(screen)
				renders[kind] = pixels
				require.Positive(t, atlasExactInkCount(pixels, ink))
				for y := range 528 {
					for x := range 528 {
						pixel := pixels.RGBAAt(x, y)
						if pixel.A != 0 {
							require.True(t, image.Pt(x, y).In(cell), "symbol leaked its cell: %s at %d,%d", kind, x, y)
							require.Equal(t, ink, pixel, "bare row symbol has no smoothing or black backing")
							require.Equal(t, pixel, pixels.RGBAAt(cell.Max.X-1-(x-cell.Min.X), y), "symmetric source art stays symmetric after compact rasterization")
						}
					}
				}
				for _, zoom := range []int{1, 2, 4} {
					dialog.view.Zoom = zoom
					screen.Clear()
					dialog.drawAtlasMarker(screen, entry, center, atlasPlaceColor(entry), true)
					outlined := atlasMarkerPixels(screen)
					require.Positive(t, atlasExactInkCount(outlined, ucolor.EGABlack), "map symbol gets dark contrast")
					for y := cell.Min.Y - unit; y < cell.Max.Y+unit; y++ {
						for x := cell.Min.X - unit; x < cell.Max.X+unit; x++ {
							pixel := outlined.RGBAAt(x, y)
							require.True(t, pixel.A == 0 || pixel == ink || pixel == ucolor.EGABlack.ToRGBA())
							if pixel.A != 0 {
								require.True(t, image.Pt(x, y).In(cell.Inset(-outline)), "outline stays within one artwork cell, not a full message-box border unit")
							}
							require.Equal(t, pixels.RGBAAt(x, y) == ink, pixel == ink, "outline never changes the shared silhouette or zoom size")
						}
					}
				}
				screen.Deallocate()
			}
			require.Equal(t, renders["towne"].Pix, renders["village"].Pix)
			require.Equal(t, renders["shrine"].Pix, renders["ruins"].Pix)
			kinds := []string{"castle", "stronghold", "keep", "towne", "hut", "shrine", "lighthouse", "dungeon"}
			for index, kind := range kinds {
				for _, other := range kinds[index+1:] {
					require.NotEqual(t, renders[kind].Pix, renders[other].Pix, "%s and %s have distinct silhouettes", kind, other)
				}
			}
		})
	}
}

func TestAtlasMarkersPersonalDiscoveryColorSurvivesSelection(t *testing.T) {
	dialog := atlasRenderingDialog()
	screen := ebiten.NewImage(528, 528)
	t.Cleanup(screen.Deallocate)
	require.Equal(t, ucolor.EGALightGreen, DialogAtlasDiscoveredColor)
	require.Equal(t, ucolor.EGALightGray, DialogAtlasUndiscoveredColor)
	for _, discovered := range []bool{false, true} {
		for _, nameKnown := range []bool{false, true} {
			entry := journal.AtlasEntry{Kind: "castle", Charted: true, NameKnown: nameKnown,
				Discovered: discovered, Survey: atlasMeasuredSite(128, 128)}
			want := DialogAtlasUndiscoveredColor
			if discovered {
				want = DialogAtlasDiscoveredColor
			}
			require.Equal(t, want, atlasPlaceColor(entry), "names and chart coordinates do not confer discovery")
			for _, selected := range []bool{false, true} {
				screen.Clear()
				dialog.drawPoint(screen, entry, selected)
				pixels := atlasMarkerPixels(screen)
				center, visible := dialog.atlasMarkerCenter(entry)
				require.True(t, visible)
				ink := color.RGBAModel.Convert(want).(color.RGBA)
				for y := center.Y - 1; y < center.Y+1; y++ {
					for x := center.X - 4; x < center.X+4; x++ {
						require.Equal(t, ink, pixels.RGBAAt(x, y), "selected castle retains its discovery colour")
					}
				}
				if selected {
					yellow := color.RGBAModel.Convert(DialogLabelColor).(color.RGBA)
					bounds := dialog.selectionBounds(center)
					require.Equal(t, yellow, pixels.RGBAAt(bounds.Min.X, bounds.Min.Y), "corner brackets use the common highlight")
					require.Zero(t, pixels.RGBAAt(center.X, bounds.Min.Y).A, "selection has open sides, not a circle")
				}
			}
			entry.Survey = journal.SurveyedLocation{}
			require.Equal(t, want, atlasPlaceColor(entry), "measurement availability does not change personal discovery")
		}
	}
}

func TestAtlasMarkersUnknownAndUnmeasuredDoNotInventMapPoints(t *testing.T) {
	view := NewAtlasViewState()
	dialog := &AtlasDialog{view: &view, mapRect: image.Rect(8, 8, 40, 40)}
	screen := ebiten.NewImage(48, 48)
	t.Cleanup(screen.Deallocate)
	screen.Fill(ucolor.EGABlue)
	before := atlasMarkerPixels(screen)
	dialog.drawAtlasSymbol(screen, journal.AtlasEntry{Kind: "unknown"}, image.Pt(24, 24), DialogLabelColor)
	dialog.drawPoint(screen, journal.AtlasEntry{Kind: "hut"}, true)
	require.Equal(t, before.Pix, atlasMarkerPixels(screen).Pix)
}

func TestAtlasMarkersMoonPhasesFitCompactBody(t *testing.T) {
	for _, unit := range []int{1, 3, 4, 5} {
		spacing := max(5, 2*unit)
		dialog := &AtlasDialog{width: 320 * unit, height: 200 * unit}
		var phases [][]byte
		for phase := range 8 {
			screen := ebiten.NewImage(64, 64)
			dialog.drawAtlasSymbol(screen, journal.AtlasEntry{Kind: "moonstone", Phase: phase}, image.Pt(32, 32), DialogValueColor)
			pixels := atlasMarkerPixels(screen)
			screen.Deallocate()
			phases = append(phases, pixels.Pix)
			cell := image.Rect(32-spacing/2, 32-spacing/2, 32-spacing/2+spacing, 32-spacing/2+spacing)
			painted := 0
			for y := range 64 {
				for x := range 64 {
					pixel := pixels.RGBAAt(x, y)
					if pixel.A != 0 {
						painted++
						require.True(t, image.Pt(x, y).In(cell), "phase %d is offset from its measured cell", phase)
						require.Equal(t, color.RGBAModel.Convert(DialogValueColor), pixel)
					}
				}
			}
			require.Positive(t, painted)
		}
		for phase := range len(phases) {
			for other := phase + 1; other < len(phases); other++ {
				require.NotEqual(t, phases[phase], phases[other], "every compact phase must remain distinct")
			}
		}
	}
}

func TestAtlasMarkersAvatarPlainRedCrossAndClipping(t *testing.T) {
	view := NewAtlasViewState()
	dialog := &AtlasDialog{view: &view, mapRect: image.Rect(16, 16, 80, 80)}
	screen := ebiten.NewImage(96, 96)
	t.Cleanup(screen.Deallocate)
	for _, sample := range []struct{ unit, zoom, body, start, end, outline int }{
		{1, 1, 5, 2, 3, 1}, {1, 2, 8, 3, 5, 1}, {1, 4, 10, 4, 7, 1},
		{3, 1, 6, 2, 4, 1}, {3, 2, 9, 3, 5, 1}, {3, 4, 12, 5, 8, 1},
		{4, 1, 8, 3, 5, 2}, {4, 2, 12, 5, 8, 2}, {4, 4, 16, 6, 10, 2},
		{5, 1, 10, 4, 7, 2}, {5, 2, 15, 5, 9, 2}, {5, 4, 20, 8, 13, 2},
	} {
		unit := sample.unit
		dialog.width, dialog.height = 320*unit, 200*unit
		view.Zoom = sample.zoom
		for _, location := range []journal.SiteLocation{{X: 128, Y: 128}, {X: 0, Y: 0}, {X: 255, Y: 255}} {
			view.CenterX, view.CenterY = 128, 128
			if location.X == 0 {
				view.CenterX, view.CenterY = view.span()/2, view.span()/2
			} else if location.X == 255 {
				view.CenterX, view.CenterY = 256-view.span()/2, 256-view.span()/2
			}
			screen.Fill(ucolor.EGABlue)
			dialog.dossier.CurrentFix = journal.SurveyedLocation{Measured: true, Location: location}
			dialog.drawAvatarFix(screen)
			pixels := atlasMarkerPixels(screen)
			x, y, visible := view.project(float64(location.X)+.5, float64(location.Y)+.5, dialog.mapRect)
			require.True(t, visible)
			origin := image.Pt(int(x)-sample.body/2, int(y)-sample.body/2)
			horizontal := image.Rect(origin.X, origin.Y+sample.start, origin.X+sample.body, origin.Y+sample.end)
			vertical := image.Rect(origin.X+sample.start, origin.Y, origin.X+sample.end, origin.Y+sample.body)
			require.Equal(t, image.Rectangle{Min: origin, Max: origin.Add(image.Pt(sample.body, sample.body))}.Inset(-sample.outline),
				dialog.avatarMarkerBounds(image.Pt(int(x), int(y))), "callout clearance includes the zoom-scaled cross and outline")
			for py := range 96 {
				for px := range 96 {
					point := image.Pt(px, py)
					want := ucolor.EGABlue.ToRGBA()
					if image.Pt(px, py).In(dialog.mapRect) {
						if point.In(horizontal.Inset(-sample.outline)) || point.In(vertical.Inset(-sample.outline)) {
							want = ucolor.EGABlack.ToRGBA()
						}
						if point.In(horizontal) || point.In(vertical) {
							want = ucolor.EGARed.ToRGBA()
						}
					}
					require.Equal(t, want, pixels.RGBAAt(px, py), "plain cross / map clip at zoom %d, pixel %d,%d", sample.zoom, px, py)
				}
			}
		}
	}
	dialog.dossier.CurrentFix.Measured = false
	screen.Fill(ucolor.EGABlue)
	before := atlasMarkerPixels(screen)
	dialog.drawAvatarFix(screen)
	require.Equal(t, before.Pix, atlasMarkerPixels(screen).Pix, "a stale location without a current fix must vanish")
}

func TestAtlasMarkersAvatarAboveLabelsInTopLevelDraw(t *testing.T) {
	setQuestJournalTestResolution(t, 1280, 720)
	view := NewAtlasViewState()
	dossier := journal.AtlasDossier{Entries: []journal.AtlasEntry{
		{Key: "castle", Name: "Castle", Kind: "castle", Charted: true, Survey: atlasMeasuredSite(240, 128)},
	}}
	dialog := NewAtlasDialog(dossier, nil, newWidgetTestKeyboard(0), &view, func() {})
	// At the right edge, the selected label clamps back across the point.
	dossier.CurrentFix = dossier.Entries[0].Survey
	dialog.Refresh(dossier)
	beforeView, beforeDossier := view, dialog.dossier
	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Deallocate)
	dialog.Draw(screen)
	x, y, visible := view.project(240.5, 128.5, dialog.mapRect)
	require.True(t, visible)
	first := atlasMarkerPixels(screen)
	require.Equal(t, ucolor.EGARed.ToRGBA(), first.RGBAAt(int(x), int(y)), "label must not cover the current fix")
	for py := range 720 {
		for px := range 1280 {
			if first.RGBAAt(px, py) == ucolor.EGARed.ToRGBA() {
				require.True(t, image.Pt(px, py).In(dialog.mapRect))
			}
		}
	}
	dialog.Draw(screen)
	require.Equal(t, first.Pix, atlasMarkerPixels(screen).Pix, "repeated rendering is deterministic")
	require.Equal(t, beforeView, view)
	require.Equal(t, beforeDossier, dialog.dossier)
	dossier.CurrentFix = journal.SurveyedLocation{}
	dialog.Refresh(dossier)
	dialog.Draw(screen)
	require.NotEqual(t, ucolor.EGARed.ToRGBA(), atlasMarkerPixels(screen).RGBAAt(int(x), int(y)))
}
