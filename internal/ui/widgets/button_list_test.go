package widgets

import (
	"fmt"
	stdcolor "image/color"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	gametext "github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestButtonListModalUsesBalancedContentMeasuredCard(t *testing.T) {
	modal := newTestButtonListModal("Ultima V Redux")
	modal.AddButton("Back", func() {})
	ensureTestButtonListLayout(modal)

	require.Len(t, modal.layout.rowRects, 1)
	topPadding := modal.layout.titleRect.Min.Y - modal.layout.cardRect.Min.Y
	bottomPadding := modal.layout.cardRect.Max.Y - modal.layout.footerRect.Max.Y
	assert.Equal(t, topPadding, bottomPadding)
	assert.Equal(t, modal.layout.cardRect.Min.X+modal.layout.cardRect.Dx()/2,
		modal.layout.titleRect.Min.X+modal.layout.titleRect.Dx()/2)
	assert.Less(t, modal.layout.cardRect.Dy(), modal.layout.cardRect.Dx(),
		"a one-row menu should remain compact rather than reserving fixed vertical space")
	assert.Equal(t,
		gametext.GetScaledNumberToWidth(modal.lastWidth, float64(FontSizeBody)),
		modal.rowOutput.Font.TextFace.Size,
	)
	assert.Equal(t, modal.rowOutput.Font.TextFace.Size, modal.descriptionOutput.Font.TextFace.Size)
	assert.Equal(t, modal.rowOutput.Font.TextFace.Size, modal.footerOutput.Font.TextFace.Size)
	assert.Equal(t,
		gametext.GetScaledNumberToWidth(modal.lastWidth, float64(FontSizeTitle)),
		modal.titleOutput.Font.TextFace.Size,
	)
}

func TestButtonListModalCustomWidthPreservesMeasuredRowHeight(t *testing.T) {
	const customWidth = 0.22
	modal := newTestButtonListModal("Ultima V Redux")
	modal.SetButtonWidthPercent(customWidth)
	modal.AddButton("Back to Main Menu", func() {})
	ensureTestButtonListLayout(modal)

	screenWidth, _ := display.GetManager().GetCurrentSize()
	row := modal.layout.rowRects[0]
	require.InDelta(t, customWidth, float64(row.Dx())/float64(screenWidth), 0.001)
	require.Greater(t, row.Dy(), int(modal.rowOutput.GetFontHeight()))
	require.InDelta(t, customWidth, modal.ButtonWidthPercent(), 0.0001)
}

func TestButtonListModalDescriptionReservesOnlyWrappedTextHeight(t *testing.T) {
	modal := newTestButtonListModal("Exit to DOS")
	modal.SetDescription("Unsaved progress will be lost. Save before leaving Ultima V Redux?")
	modal.AddButton("Quick Save", func() {})
	ensureTestButtonListLayout(modal)

	require.False(t, modal.layout.descriptionRect.Empty())
	require.Contains(t, modal.descriptionOutput.GetOutputStr(false), "Unsaved progress")
	require.Less(t, modal.layout.descriptionRect.Max.Y, modal.layout.rowRects[0].Min.Y)
	expectedHeight := int(modal.descriptionOutput.GetTotalHeightForLines(
		modal.descriptionOutput.GetPanelContentLineCount(),
	))
	require.InDelta(t, expectedHeight, modal.layout.descriptionRect.Dy(), 1)
}

func TestButtonListModalSupportsLongDescriptionsAndExplicitEscapeAction(t *testing.T) {
	modal := newTestButtonListModal("Set the Game-File Location")
	modal.SetBorderWidthPercent(0.82)
	modal.SetDescriptionRowCapacity(12)
	modal.SetEscapeLabel("Quit")
	modal.SetDescription("Settings file: /Users/avatar/.ultima_v_redux/config.yaml\n" +
		"1. Find DataFilePath.\n" +
		"2. Replace its value with your Ultima V folder.\n" +
		"3. Save the file, then return here.\n" +
		"Example: DataFilePath: '/Users/avatar/Documents/UltimaV'\n" +
		"Choose the folder containing DATA.OVL, not the DATA.OVL file itself.")
	modal.AddButton("Check Saved Settings", func() {})
	modal.ensureLayout(1280, 720)

	assert.Contains(t, modal.descriptionOutput.GetOutputStr(false), "Settings file:")
	assert.Contains(t, modal.descriptionOutput.GetOutputStr(false), "DATA.OVL")
	assert.Contains(t, modal.footerOutput.GetOutputStr(false), "[Esc] Quit")
	assert.GreaterOrEqual(t, modal.layout.cardRect.Min.Y, 0)
	assert.LessOrEqual(t, modal.layout.cardRect.Max.Y, 720)
}

func TestButtonListModalPagesDescriptionsWithoutCircularReordering(t *testing.T) {
	modal := newTestButtonListModal("Recovery")
	modal.SetDescriptionRowCapacity(2)
	modal.SetDescription("first\nsecond\nthird\nactionable suffix")
	modal.AddButton("Back", func() {})
	modal.ensureLayout(1280, 720)

	assert.Equal(t, 2, modal.descriptionPageCount())
	assert.Contains(t, modal.descriptionOutput.GetOutputStr(false), "first")
	assert.Contains(t, modal.descriptionOutput.GetOutputStr(false), "actionable suffix")
	assert.Contains(t, modal.footerOutput.GetOutputStr(false), "[Shift+Up/Down] Read")

	modal.keyboard.SimulateKeyPress(ebiten.KeyPageDown)
	modal.Update()
	assert.Equal(t, 1, modal.descriptionPage)
}

func TestButtonListModalRetainsDescriptionsBeyondFormerCircularCapacity(t *testing.T) {
	lines := make([]string, 140)
	for index := range lines {
		lines[index] = fmt.Sprintf("detail-%03d", index)
	}
	modal := newTestButtonListModal("Search Details")
	modal.SetDescriptionRowCapacity(2)
	modal.SetDescription(strings.Join(lines, "\n"))
	modal.AddButton("Back", func() {})
	modal.ensureLayout(1280, 720)

	output := modal.descriptionOutput.GetOutputStr(false)
	assert.Contains(t, output, "detail-000")
	assert.Contains(t, output, "detail-139")
	assert.Equal(t, 70, modal.descriptionPageCount())
}

func TestButtonListModalCardCanExpandWithoutResizingRows(t *testing.T) {
	const (
		buttonWidth = 0.28
		borderWidth = 0.45
	)
	modal := newTestButtonListModal("Exit to DOS")
	modal.SetButtonWidthPercent(buttonWidth)
	modal.SetBorderWidthPercent(borderWidth)
	modal.SetDescription("Unsaved progress will be lost. Save before leaving Ultima V Redux?")
	modal.AddButton("Quick Save", func() {})
	ensureTestButtonListLayout(modal)

	screenWidth, _ := display.GetManager().GetCurrentSize()
	require.InDelta(t, buttonWidth, float64(modal.layout.rowRects[0].Dx())/float64(screenWidth), 0.001)
	require.InDelta(t, borderWidth, float64(modal.layout.cardRect.Dx())/float64(screenWidth), 0.001)
}

func TestButtonListModalDrawsOpaqueClassicCardAndInvertedSelection(t *testing.T) {
	modal := newTestButtonListModal("Choose")
	modal.AddButton("One", func() {})
	modal.AddButton("Two", func() {})
	width, height := display.GetManager().GetCurrentSize()
	screen := ebiten.NewImage(width, height)
	t.Cleanup(screen.Dispose)
	screen.Fill(ucolor.EGAMagenta)
	modal.Draw(screen)

	card := modal.layout.cardRect
	selected := modal.layout.rowRects[0]
	assert.Equal(t, rgba(DialogRuleColor), rgba(screen.At(card.Min.X+card.Dx()/2, card.Min.Y)))
	assert.Equal(t, rgba(ucolor.DialogBackground.RGBA()), rgba(screen.At(
		card.Min.X+modal.layout.pixelUnit*3,
		modal.layout.titleRect.Min.Y+modal.layout.titleRect.Dy()/2,
	)))
	assert.Equal(t, rgba(DialogSelectedBackground), rgba(screen.At(
		selected.Min.X+modal.layout.pixelUnit*3,
		selected.Min.Y+selected.Dy()/2,
	)))
}

func TestButtonListModalKeyboardNavigationWraps(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	modal := NewButtonListModal(
		"Choose",
		func() {},
		keyboard,
		&sprites.PercentBasedPlacement{EndPercentX: 1, EndPercentY: 1},
	)
	for _, label := range []string{"One", "Two", "Three"} {
		modal.AddButton(label, func() {})
	}

	assertButtonListSelection(t, modal, 0)
	pressButtonListKey(keyboard, modal, ebiten.KeyUp)
	assertButtonListSelection(t, modal, 2)
	pressButtonListKey(keyboard, modal, ebiten.KeyRight)
	assertButtonListSelection(t, modal, 0)
	pressButtonListKey(keyboard, modal, ebiten.KeyLeft)
	assertButtonListSelection(t, modal, 2)
	pressButtonListKey(keyboard, modal, ebiten.KeyDown)
	assertButtonListSelection(t, modal, 0)
}

func TestButtonListModalOnlyRepeatsDirectionalNavigation(t *testing.T) {
	held := map[ebiten.Key]bool{}
	keyboard := input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return held[key] })
	calls := 0
	modal := NewButtonListModal("Choose", func() {}, keyboard,
		&sprites.PercentBasedPlacement{EndPercentX: 1, EndPercentY: 1})
	for range 3 {
		modal.AddButton("Choice", func() { calls++ })
	}
	held[ebiten.KeyDown] = true
	modal.Update()
	require.Equal(t, 1, modal.currentSelectionIndex)
	for range 18 {
		modal.Update()
	}
	require.Equal(t, 2, modal.currentSelectionIndex)
	clear(held)
	keyboard.ClearFrameConsumption()
	modal.Update()
	held[ebiten.KeyEnter] = true
	for range 60 {
		modal.Update()
	}
	require.Equal(t, 1, calls)
}

func TestButtonListModalEmptyListIgnoresNavigationAndActivation(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	modal := NewButtonListModal(
		"Choose",
		func() {},
		keyboard,
		&sprites.PercentBasedPlacement{EndPercentX: 1, EndPercentY: 1},
	)

	require.NotPanics(t, func() {
		pressButtonListKey(keyboard, modal, ebiten.KeyDown)
		pressButtonListKey(keyboard, modal, ebiten.KeyUp)
		pressButtonListKey(keyboard, modal, ebiten.KeyEnter)
	})
	assert.Equal(t, -1, modal.currentSelectionIndex)
}

func TestButtonListModalUpdatePreservesLogicalDrawLayout(t *testing.T) {
	modal := newTestButtonListModal("Choose")
	modal.AddButton("One", func() {})
	modal.ensureLayout(1365, 768)
	card := modal.layout.cardRect

	modal.Update()

	assert.Equal(t, 1365, modal.lastWidth)
	assert.Equal(t, 768, modal.lastHeight)
	assert.Equal(t, card, modal.layout.cardRect)
}

func TestButtonListModalKeyboardCallbacksDispatchOnce(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	escapeCount := 0
	firstCount := 0
	secondCount := 0
	modal := NewButtonListModal(
		"Choose",
		func() { escapeCount++ },
		keyboard,
		&sprites.PercentBasedPlacement{EndPercentX: 1, EndPercentY: 1},
	)
	modal.AddButton("One", func() { firstCount++ })
	modal.AddButton("Two", func() { secondCount++ })

	pressButtonListKey(keyboard, modal, ebiten.KeyDown)
	pressButtonListKey(keyboard, modal, ebiten.KeyEnter)
	modal.Update()
	assert.Equal(t, 0, firstCount)
	assert.Equal(t, 1, secondCount)

	pressButtonListKey(keyboard, modal, ebiten.KeyEscape)
	modal.Update()
	assert.Equal(t, 1, escapeCount)
}

func newTestButtonListModal(title string) *ButtonListModal {
	return NewButtonListModal(
		title,
		func() {},
		newWidgetTestKeyboard(0),
		&sprites.PercentBasedPlacement{EndPercentX: 1, EndPercentY: 1},
	)
}

func ensureTestButtonListLayout(modal *ButtonListModal) {
	width, height := display.GetManager().GetCurrentSize()
	modal.ensureLayout(width, height)
}

func pressButtonListKey(keyboard *input.Keyboard, modal *ButtonListModal, key ebiten.Key) {
	keyboard.SimulateKeyPress(key)
	modal.Update()
}

func assertButtonListSelection(t *testing.T, modal *ButtonListModal, selectedIndex int) {
	t.Helper()
	assert.Equal(t, selectedIndex, modal.currentSelectionIndex)
	assert.GreaterOrEqual(t, selectedIndex, 0)
	assert.Less(t, selectedIndex, len(modal.items))
}

func rgba(value stdcolor.Color) stdcolor.RGBA {
	return stdcolor.RGBAModel.Convert(value).(stdcolor.RGBA)
}
