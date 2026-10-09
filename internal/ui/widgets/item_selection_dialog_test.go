package widgets

import (
	"image"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestItemSelectionDialogMandatoryConfigPanics(t *testing.T) {
	valid := testItemSelectionConfig(newWidgetTestKeyboard(0), testUseItems())
	tests := []struct {
		name   string
		mutate func(*ItemSelectionDialogConfig)
		panic  string
	}{
		{
			name:   "keyboard",
			mutate: func(config *ItemSelectionDialogConfig) { config.Keyboard = nil },
			panic:  "ItemSelectionDialog: keyboard is required",
		},
		{
			name:   "select callback",
			mutate: func(config *ItemSelectionDialogConfig) { config.OnSelect = nil },
			panic:  "ItemSelectionDialog: OnSelect is required",
		},
		{
			name:   "cancel callback",
			mutate: func(config *ItemSelectionDialogConfig) { config.OnCancel = nil },
			panic:  "ItemSelectionDialog: OnCancel is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			test.mutate(&config)
			require.PanicsWithValue(t, test.panic, func() {
				NewItemSelectionDialog(config)
			})
		})
	}
}

func TestItemSelectionDialogOnlyBuildsNonEmptyTabsIncludingKeys(t *testing.T) {
	setItemDialogTestResolution(t, 320, 200)
	items := []game_state.UseableItemInfo{
		testUseItem("Crown", game_state.UseItemCategoryQuest, references.Crown),
		testUseItem("Yellow", game_state.UseItemCategoryPotion, references.Yellow),
		testUseItem("Skull Keys", game_state.UseItemCategoryKeys, references.SkullKeys),
	}
	dialog := NewItemSelectionDialog(testItemSelectionConfig(newWidgetTestKeyboard(0), items))

	require.Equal(t, []ItemCategory{
		ItemCategoryAll,
		ItemCategoryQuest,
		ItemCategoryPotion,
		ItemCategoryKeys,
	}, dialog.categories)
	require.Len(t, dialog.layout.tabRects, len(dialog.categories))
	for _, rect := range dialog.layout.tabRects {
		require.False(t, rect.Empty())
		require.True(t, rect.In(dialog.layout.headerRect))
	}
	assert.NotContains(t, dialog.categories, ItemCategoryScroll)
	assert.NotContains(t, dialog.categories, ItemCategoryMoonstone)
}

func TestItemSelectionDialogNavigationPreservesPerTabSelectionAndScroll(t *testing.T) {
	setItemDialogTestResolution(t, 320, 200)
	items := make([]game_state.UseableItemInfo, 0, 42)
	for index := range 40 {
		item := testUseItem("Quest "+string(rune('A'+index)), game_state.UseItemCategoryQuest, references.Crown)
		items = append(items, item)
	}
	items = append(items,
		testUseItem("Yellow", game_state.UseItemCategoryPotion, references.Yellow),
		testUseItem("Red", game_state.UseItemCategoryPotion, references.Red),
	)
	dialog := NewItemSelectionDialog(testItemSelectionConfig(newWidgetTestKeyboard(0), items))

	for range 35 {
		dialog.moveSelection(1)
	}
	allSelection := dialog.GetSelectedItemIndex()
	allScroll := dialog.scrollStarts[ItemCategoryAll]
	require.Equal(t, 35, allSelection)
	require.Positive(t, allScroll)

	dialog.switchCategoryBy(1)
	require.Equal(t, ItemCategoryQuest, dialog.currentCategory())
	dialog.moveSelection(2)
	require.Equal(t, 2, dialog.GetSelectedItemIndex())
	dialog.switchCategoryBy(-1)
	require.Equal(t, ItemCategoryAll, dialog.currentCategory())
	assert.Equal(t, allSelection, dialog.GetSelectedItemIndex())
	assert.Equal(t, allScroll, dialog.scrollStarts[ItemCategoryAll])
	assert.Equal(t, -1, itemDialogCategoryDelta(true))
	assert.Equal(t, 1, itemDialogCategoryDelta(false))
}

func TestItemSelectionDialogActionPagesPrecedeMeasuredLorePages(t *testing.T) {
	setItemDialogTestResolution(t, 320, 200)
	item := testUseItem("Shard", game_state.UseItemCategoryQuest, references.ShardOfFalsehood)
	item.FullName = "Shard of Falsehood"
	item.Warning = "Destruction is permanent once the ritual commits."
	item.Description = strings.Repeat("Long lore remains readable at the standard font size. ", 60)
	dialog := NewItemSelectionDialog(testItemSelectionConfig(newWidgetTestKeyboard(0), []game_state.UseableItemInfo{item}))

	pages := dialog.detailPages()
	require.Greater(t, len(pages), 2)
	require.Equal(t, "Action", pages[0][0].value)
	lorePage := -1
	for pageIndex, page := range pages {
		for _, line := range page {
			if line.header && line.value == "Lore" {
				lorePage = pageIndex
			}
		}
	}
	require.Positive(t, lorePage)
	capacity := dialog.detailOutput.GetLineCapacity(dialog.detailTextRect().Dy())
	for _, page := range pages {
		require.LessOrEqual(t, dialog.detailLineCount(page), capacity)
	}

	keyboard := dialog.keyboard
	pressItemDialogKey(dialog, keyboard, ebiten.KeySpace)
	require.Equal(t, 1, dialog.GetDetailPage())
	dialog.moveSelection(0)
	require.Equal(t, 1, dialog.GetDetailPage(), "a bounded no-op does not discard the page")
}

func TestItemSelectionDialogAllowsUnavailableSelectionAndCancelsExactlyOnce(t *testing.T) {
	item := testUseItem("Purple", game_state.UseItemCategoryPotion, references.Purple)
	item.AvailableNow = false
	item.Warning = "No noticeable effect outside combat; it is still consumed."
	keyboard := newWidgetTestKeyboard(0)
	selected := make([]game_state.UseableItemInfo, 0, 1)
	cancelCalls := 0
	config := testItemSelectionConfig(keyboard, []game_state.UseableItemInfo{item})
	config.OnSelect = func(result game_state.UseableItemInfo) { selected = append(selected, result) }
	config.OnCancel = func() { cancelCalls++ }
	dialog := NewItemSelectionDialog(config)

	pressItemDialogKey(dialog, keyboard, ebiten.KeyNumpadEnter)
	require.True(t, dialog.IsDismissed())
	require.Len(t, selected, 1)
	assert.Equal(t, references.Item(references.Purple), selected[0].ItemType)
	assert.Zero(t, cancelCalls)
	dialog.HandleEnter()
	assert.Len(t, selected, 1)

	cancelDialog := NewItemSelectionDialog(config)
	pressItemDialogKey(cancelDialog, keyboard, ebiten.KeyEscape)
	cancelDialog.Dismiss()
	assert.Equal(t, 1, cancelCalls)
}

func TestItemSelectionDialogIgnoresNumberShortcutsAndUsesResponsiveMainFont(t *testing.T) {
	setItemDialogTestResolution(t, 320, 200)
	keyboard := newWidgetTestKeyboard(0)
	dialog := NewItemSelectionDialog(testItemSelectionConfig(keyboard, testUseItems()))
	dialog.ensureLayout()
	smallSize := dialog.font.TextFace.Size

	pressItemDialogKey(dialog, keyboard, ebiten.Key1)
	assert.Zero(t, dialog.GetSelectedCategory(), "number keys do not select categories")
	require.Equal(t, FontSizeMainScreen, dialog.GetFontSize())
	for _, size := range []float64{
		dialog.titleOutput.Font.TextFace.Size,
		dialog.tabOutput.Font.TextFace.Size,
		dialog.rowOutput.Font.TextFace.Size,
		dialog.detailOutput.Font.TextFace.Size,
		dialog.pageOutput.Font.TextFace.Size,
		dialog.footerOutput.Font.TextFace.Size,
	} {
		require.Equal(t, smallSize, size)
	}

	dialog.moveSelection(2)
	dialog.detailPage = min(1, dialog.GetDetailPageCount()-1)
	setResponsiveDialogTestResolution(t, 1280, 720)
	dialog.ensureLayout()
	require.Equal(t, 2, dialog.GetSelectedItemIndex())
	require.Less(t, dialog.GetDetailPage(), dialog.GetDetailPageCount())
	require.Greater(t, dialog.font.TextFace.Size, smallSize)
	largeBounds := dialog.outerBorder.GetBounds()
	require.InDelta(t, 1280*itemDialogMarginX, largeBounds.Min.X, 1)
	require.InDelta(t, 720*itemDialogMarginY, largeBounds.Min.Y, 1)
	require.InDelta(t, 1280*(1-itemDialogMarginX), largeBounds.Max.X, 1)
	require.InDelta(t, 720*(1-itemDialogMarginY), largeBounds.Max.Y, 1)
	drawn := dialogImage(1280, 720, dialog)
	t.Cleanup(drawn.Dispose)
	assert.NotEqual(t, uint32(0), alphaAt(drawn, dialog.layout.panelRect.Min.Add(image.Pt(4, 4))))
	overlap := dialog.pixelUnit()
	require.Equal(t, dialog.layout.headerRect.Max.Y-overlap, dialog.layout.bodyRect.Min.Y)
	require.Equal(t, dialog.layout.bodyRect.Max.Y-overlap, dialog.layout.footerRect.Min.Y)
	require.Equal(t, dialog.layout.listRect.Max.X-overlap, dialog.layout.detailRect.Min.X)
	panePerimeterPoint := image.Pt(
		dialog.layout.headerRect.Min.X+dialog.layout.headerRect.Dx()/2,
		dialog.layout.headerRect.Min.Y,
	)
	requireColorRGBA(t, DialogRuleColor, drawn.At(panePerimeterPoint.X, panePerimeterPoint.Y))
	paneCorner := dialog.layout.headerRect.Min
	requireColorNotRGBA(t, MerchantDialogBackgroundColor, drawn.At(paneCorner.X, paneCorner.Y))
	legacyOuterPoint := image.Pt(largeBounds.Min.X+largeBounds.Dx()/2, largeBounds.Min.Y)
	requireColorNotRGBA(t, DialogRuleColor, drawn.At(legacyOuterPoint.X, legacyOuterPoint.Y))
}

func testItemSelectionConfig(
	keyboard *input.Keyboard,
	items []game_state.UseableItemInfo,
) ItemSelectionDialogConfig {
	return ItemSelectionDialogConfig{
		Items: items, Keyboard: keyboard, OnSelect: func(game_state.UseableItemInfo) {}, OnCancel: func() {},
	}
}

func testUseItems() []game_state.UseableItemInfo {
	return []game_state.UseableItemInfo{
		testUseItem("Crown", game_state.UseItemCategoryQuest, references.Crown),
		testUseItem("Sceptre", game_state.UseItemCategoryQuest, references.Sceptre),
		testUseItem("Shard", game_state.UseItemCategoryQuest, references.ShardOfFalsehood),
		testUseItem("Yellow", game_state.UseItemCategoryPotion, references.Yellow),
		testUseItem("Vas Lor", game_state.UseItemCategoryScroll, references.ScrollVasLor),
		testUseItem("Watch", game_state.UseItemCategoryTools, references.PocketWatch),
		testUseItem("New Moon", game_state.UseItemCategoryMoonstone, references.NewMoon),
		testUseItem("Skull Keys", game_state.UseItemCategoryKeys, references.SkullKeys),
	}
}

func testUseItem(
	name string,
	category game_state.UseItemCategory,
	item references.Item,
) game_state.UseableItemInfo {
	return game_state.UseableItemInfo{
		Name:            name,
		FullName:        name,
		Description:     "Lore for " + name + ".",
		EffectSummary:   "Produces the expected effect.",
		NextInteraction: "The effect resolves immediately.",
		Requirement:     "Ready to use.",
		Consumption:     "Not consumed.",
		Warning:         "",
		AvailableNow:    true,
		Category:        category,
		UsageContext:    game_state.UsageField,
		Quantity:        1,
		SpriteIndex:     indexes.ItemKey,
		ItemRef:         references.InventoryItem{},
		ItemType:        item,
	}
}

func pressItemDialogKey(dialog *ItemSelectionDialog, keyboard *input.Keyboard, key ebiten.Key) {
	keyboard.SimulateKeyPress(key)
	dialog.Update()
}

func setItemDialogTestResolution(t *testing.T, width, height int) {
	t.Helper()
	setResponsiveDialogTestResolution(t, width, height)
}

func dialogImage(width, height int, dialog *ItemSelectionDialog) *ebiten.Image {
	screen := ebiten.NewImage(width, height)
	dialog.Draw(screen)

	return screen
}
