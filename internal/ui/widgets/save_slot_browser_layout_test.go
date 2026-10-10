package widgets

import (
	"fmt"
	"image"
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	pkgcolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestSaveSlotBrowserExplicitSelectionAndPersistentNewest(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyDown, ebiten.KeyUp} {
		keyboard := newWidgetTestKeyboard(0)
		views := makeSaveSlotViews(8)
		views[3].Newest = true
		calls := 0
		browser := newTestSaveSlotBrowser(keyboard, views, SaveSlotBrowserConfig{
			RequireExplicitSelection: true, NewestLegend: "* Newest save for this playthrough",
			OnPrimary: func(SaveSlotView) { calls++ },
		})
		require.Equal(t, -1, browser.GetSelectedIndex())
		for _, unselecting := range []ebiten.Key{ebiten.KeyEnter, ebiten.KeyNumpadEnter, ebiten.KeyPageUp, ebiten.KeyPageDown, ebiten.KeyRight} {
			pressSaveSlotBrowserKey(browser, keyboard, unselecting)
			require.Equal(t, -1, browser.GetSelectedIndex())
		}
		require.Zero(t, calls)
		require.Contains(t, strings.Join(strings.Fields(browser.detailOutput.GetOutputStr(false)), " "), "Select a destination with Up/Down.")
		pressSaveSlotBrowserKey(browser, keyboard, key)
		expected := 0
		if key == ebiten.KeyUp {
			expected = 7
		}
		require.Equal(t, expected, browser.GetSelectedIndex())
		pressSaveSlotBrowserKey(browser, keyboard, ebiten.KeyEnter)
		require.Equal(t, 1, calls)
		for i, slot := range browser.slots {
			require.Equal(t, i == 3, slot.Newest)
		}
		require.Contains(t, browser.footerOutput.GetOutputStr(false), "* Newest save for this playthrough")
	}
	browser := newTestSaveSlotBrowser(newWidgetTestKeyboard(0), makeSaveSlotViews(8), SaveSlotBrowserConfig{
		RequireExplicitSelection: true, InitialSelectionID: "f",
	})
	require.Equal(t, 5, browser.GetSelectedIndex(), "retry selection takes precedence")
}

func TestSaveSlotBrowserConnectedGeometryAtSupportedSizes(t *testing.T) {
	for _, size := range config.SupportedWindowResolutions() {
		t.Run(fmt.Sprintf("%dx%d", size.X, size.Y), func(t *testing.T) {
			setResponsiveDialogTestResolution(t, size.X, size.Y)
			views := makeSaveSlotViews(8)
			for i := range views {
				views[i].Label = fmt.Sprintf("Slot %d", i+1)
				views[i].HasSave = true
				views[i].RowSummary = "LongAvatarName"
				views[i].Description = strings.Repeat("Checkpoint ", 7)
			}
			views[3].Newest = true
			views[0].Problem = strings.Repeat("Damaged data. ", 30)
			b := newTestSaveSlotBrowser(newWidgetTestKeyboard(0), views, SaveSlotBrowserConfig{
				Title: "Select Save Game", PrimaryActionLabel: "Import", FontSize: FontSizeMainScreen,
				NewestLegend: "* Newest save for this playthrough",
			})
			require.Equal(t, 8, len(b.rowRects))
			listFrame := b.frameRects[1]
			for row, rect := range b.rowRects {
				require.True(t, rect.In(listFrame.Inset(b.pixelUnit()*2)))
				require.Equal(t, rect.Min.X-listFrame.Min.X, listFrame.Max.X-rect.Max.X)
				require.InDelta(t, b.rowRects[0].Dy(), rect.Dy(), 1)
				if row > 0 {
					require.Equal(t, b.rowRects[row-1].Max.Y, rect.Min.Y)
				}
			}
			require.Equal(t, b.rowRects[0].Min.Y-listFrame.Min.Y, listFrame.Max.Y-b.rowRects[7].Max.Y)
			require.Equal(t, GetScaledFont(FontSizeDialogTitle).TextFace.Size, b.titleOutput.Font.TextFace.Size)
			require.Same(t, b.titleOutput.Font, b.detailHeadingOutput.Font)
			require.Less(t, b.detailHeadingRect.Max.Y, b.detailTextRect.Min.Y)
			require.LessOrEqual(t, b.footerOutput.GetTotalHeightForLines(b.footerOutput.GetPanelContentLineCount()), float64(b.footerTextRect.Dy()))
			for _, line := range strings.Split(b.footerOutput.GetOutputStr(false), "\n") {
				advance, _ := b.bodyFont.MeasureString(line)
				require.LessOrEqual(t, advance, float64(b.footerTextRect.Dx()))
			}
			require.NotContains(t, b.footerOutput.GetOutputStr(false), "Page Up")
			require.Contains(t, b.footerOutput.GetOutputStr(false), "Left/Right: Details")
			require.Contains(t, strings.Join(strings.Fields(b.detailOutput.GetOutputStr(false)), " "), strings.TrimSpace(views[0].Description))
			t.Logf("secondary descriptions visible: %v; row height %d", b.showDescriptions, b.rowRects[0].Dy())
			require.True(t, b.showDescriptions, "measured eight-row layout fits at supported sizes")
			// Geometry covers every preset; pixels cover fractional and dense scaling.
			if size.X != 1366 && size.X != 2560 {
				return
			}
			screen := ebiten.NewImage(size.X, size.Y)
			t.Cleanup(screen.Dispose)
			for _, index := range []int{0, len(b.rowRects) - 1} {
				b.listState.SetIndex(index)
				b.Draw(screen)
				selected := b.rowRects[b.GetSelectedIndex()]
				require.Equal(t, pkgcolor.DialogTabActive.RGBA(), screen.At(selected.Min.X, selected.Min.Y))
				// Full-frame output must leave both gutters untouched by row fills.
				require.Equal(t, pkgcolor.DialogBackground.RGBA(), screen.At(selected.Min.X-1, selected.Min.Y))
				require.Equal(t, pkgcolor.DialogBackground.RGBA(), screen.At(selected.Max.X, selected.Min.Y))
			}
			requireImageRegionContainsColor(t, screen, b.footerTextRect, DialogControlKeyColor)
			requireImageRegionContainsColor(t, screen, b.footerTextRect, DialogControlHintColor)
			screen.Clear()
			b.drawTitle(screen)
			glyphs := renderedNontransparentBounds(screen, b.frameRects[0])
			require.False(t, glyphs.Empty())
			require.True(t, glyphs.In(b.titleRect))
			require.InDelta(t, b.titleRect.Min.Y+b.titleRect.Max.Y, glyphs.Min.Y+glyphs.Max.Y, 3,
				"actual title glyphs should balance vertical padding")
			require.InDelta(t, b.titleRect.Min.X+b.titleRect.Max.X, glyphs.Min.X+glyphs.Max.X, 3)
		})
	}
}

func TestSaveSlotBrowserEnterHintSegmentsAndNewestGlyph(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	b := newTestSaveSlotBrowser(newWidgetTestKeyboard(0), []SaveSlotView{{
		ID: "slot-01", Label: "Slot 1", RowSummary: "Avatar", HasSave: true, PrimaryEnabled: true, Newest: true,
	}}, SaveSlotBrowserConfig{PrimaryActionLabel: "Import"})
	line := strings.Split(b.footerOutput.GetOutputStr(false), "\n")[1]
	lineWidth, _ := b.bodyFont.MeasureString(line)
	// A row-sized fixture also fits the isolated footer segment at the origin.
	screen := ebiten.NewImageWithOptions(image.Rect(0, 0, b.rowRects[0].Max.X, b.rowRects[0].Max.Y), nil)
	t.Cleanup(screen.Dispose)
	keyWidth, _ := b.bodyFont.MeasureString("Enter")
	hintWidth, _ := b.bodyFont.MeasureString(": Import")
	b.footerOutput.DrawPanelContentRange(screen, image.Pt(0, 0), int(math.Ceil(lineWidth)), etext.AlignCenter, 1, 1)
	requireImageRegionContainsColor(t, screen, image.Rect(0, 0, int(keyWidth), 40), DialogControlKeyColor)
	requireImageRegionContainsColor(t, screen, image.Rect(int(math.Ceil(keyWidth)), 0, int(keyWidth+hintWidth), 40), DialogControlHintColor)
	screen.Clear()
	b.drawSlotRow(screen, b.rowRects[0], b.slots[0], false)
	prefixWidth, _ := b.bodyFont.MeasureString("Slot 1 ")
	markerWidth, _ := b.bodyFont.MeasureString("*")
	marker := b.rowRects[0]
	marker.Min.X += saveSlotBrowserRowHorizontalPadding + int(math.Ceil(prefixWidth))
	marker.Max.X = marker.Min.X + int(markerWidth)
	require.False(t, renderedNontransparentBounds(screen, marker).Empty(), "newest star is visible")
	screen.Clear()
	b.slots[0].Newest = false
	b.drawSlotRow(screen, b.rowRects[0], b.slots[0], false)
	require.True(t, renderedNontransparentBounds(screen, marker).Empty(), "absent marker reserves a blank column")
}

func TestSaveSlotBrowserSelectedFolderReviewFitsSupportedSizes(t *testing.T) {
	path := "/Users/avatar/Documents/InstalledGames/UltimaV"
	for _, size := range config.SupportedWindowResolutions() {
		t.Run(fmt.Sprintf("%dx%d", size.X, size.Y), func(t *testing.T) {
			setResponsiveDialogTestResolution(t, size.X, size.Y)
			browser := newTestSaveSlotBrowser(newWidgetTestKeyboard(0), []SaveSlotView{{
				ID: "gog", Label: "GOG installation", ReviewValue: path, HasSave: true, PrimaryEnabled: true,
				Details: []SaveSlotDetail{{Label: "Folder", Value: path}},
			}}, SaveSlotBrowserConfig{
				Title: "Choose Game Files", PrimaryActionLabel: "Use Selected", CancelActionLabel: "Back",
				FontSize: FontSizeMainScreen, ReviewLabel: "Selected Folder",
			})

			require.True(t, browser.reviewFrameRect.In(browser.detailRect))
			require.Less(t, browser.reviewFrameRect.Max.Y, browser.detailTextRect.Min.Y)
			require.Contains(t, browser.footerOutput.GetOutputStr(false), "Esc: Back")
			screen := ebiten.NewImageWithOptions(browser.reviewTextRect, nil)
			browser.drawReview(screen)
			screen.Dispose()
			reviewLines := browser.reviewOutput.GetPanelContentLineCount()
			require.LessOrEqual(t, reviewLines, saveSlotBrowserReviewLineCount)
			require.LessOrEqual(t,
				browser.reviewTextY+int(math.Ceil(browser.reviewOutput.GetTotalHeightForLines(reviewLines))),
				browser.reviewTextRect.Max.Y,
			)
			require.Contains(t, strings.Join(strings.Fields(browser.detailOutput.GetOutputStr(false)), ""), path)
			require.Contains(t, strings.Join(strings.Fields(browser.reviewOutput.GetOutputStr(false)), ""), path)
		})
	}
}

func TestSaveSlotBrowserCompactFallbackKeepsFullDescriptionPaged(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	description := strings.Repeat("Checkpoint details remain available. ", 20)
	b := newTestSaveSlotBrowser(newWidgetTestKeyboard(0), []SaveSlotView{{
		ID: "slot-01", Label: "Slot 1", Description: description,
	}}, SaveSlotBrowserConfig{FontSize: FontSizeMainScreen, Placement: sprites.PercentBasedPlacement{
		StartPercentX: 0.1, EndPercentX: 0.9, StartPercentY: 0.25, EndPercentY: 0.75,
	}})
	require.False(t, b.showDescriptions)
	require.Contains(t, strings.Join(strings.Fields(b.detailOutput.GetOutputStr(false)), " "), strings.TrimSpace(description))
	require.Greater(t, b.detailPageCount(), 1)
	for page := range b.detailPageCount() {
		require.Equal(t, page, b.detailPage)
		b.moveDetailPage(1)
	}
}

func TestSaveSlotBrowserDescriptionPreviewRetainsPrefix(t *testing.T) {
	// Non-repeating text distinguishes the preview's prefix from a wrapped tail.
	description := "Before Blackthorn: carry keys, rescue friends, then prepare for Doom. Final note."
	for _, size := range config.SupportedWindowResolutions() {
		t.Run(fmt.Sprintf("%dx%d", size.X, size.Y), func(t *testing.T) {
			setResponsiveDialogTestResolution(t, size.X, size.Y)
			view := SaveSlotView{ID: "slot-01", Label: "Slot 1", RowSummary: "Avatar", HasSave: true, Description: description}
			b := newTestSaveSlotBrowser(newWidgetTestKeyboard(0), []SaveSlotView{view}, SaveSlotBrowserConfig{})
			require.True(t, b.showDescriptions)
			textWidth := b.rowRects[0].Dx() - saveSlotBrowserRowHorizontalPadding*saveSlotBrowserCenterDivisor
			capacity := b.descriptionOutput.Font.CharsForWidth(textWidth)
			expected := truncateRunes(description, capacity)
			require.Greater(t, utf8.RuneCountInString(expected), modalDialogTitleMaxChars)
			screen := ebiten.NewImageWithOptions(b.rowRects[0], nil)
			b.drawSlotRow(screen, b.rowRects[0], view, false)
			screen.Dispose()
			require.Equal(t, expected, strings.TrimSpace(b.descriptionOutput.GetOutputStr(false)))
			require.True(t, strings.HasPrefix(b.descriptionOutput.GetOutputStr(false), "Before Blackthorn:"))
			require.Equal(t, 1, b.descriptionOutput.GetPanelContentLineCount())
		})
	}
}

func renderedNontransparentBounds(screen *ebiten.Image, region image.Rectangle) image.Rectangle {
	pixels := make([]byte, region.Dx()*region.Dy()*4)
	screen.SubImage(region).(*ebiten.Image).ReadPixels(pixels)
	var bounds image.Rectangle
	for y := range region.Dy() {
		for x := range region.Dx() {
			if pixels[(y*region.Dx()+x)*4+3] != 0 {
				bounds = bounds.Union(image.Rect(region.Min.X+x, region.Min.Y+y, region.Min.X+x+1, region.Min.Y+y+1))
			}
		}
	}
	return bounds
}
