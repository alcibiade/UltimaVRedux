package widgets

import (
	"fmt"
	"image"
	"math"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestZStatsCanonicalSpellDetailsFitOnePage(t *testing.T) {
	dossier := testZStatsDossier()
	dossier.Magic.Spells = nil
	items := references.NewInventoryItemsReferences()
	for spell := references.InLor; spell <= references.AnTym; spell++ {
		info := references.GetSpellInfo(spell)
		var reagents []string
		for reagent := references.SulfurAsh; reagent <= references.MandrakeRoot; reagent++ {
			if references.GetSpellReagents(spell)&(1<<reagent) != 0 {
				reagents = append(reagents, items.GetReferenceByItem(reagent).ItemName)
			}
		}
		dossier.Magic.Spells = append(dossier.Magic.Spells, game_state.ZStatsSpell{
			ID: spell.ID(), Name: info.Name, Syllables: info.Name, Circle: info.Circle,
			MPCost: info.Circle, Prepared: 99, FormulaState: "confirmed",
			Reagents: reagents, Targeting: info.Targeting.String(), Contexts: info.Context.String(),
			Description: info.Description,
		})
	}
	for _, size := range config.SupportedWindowResolutions() {
		t.Run(fmt.Sprintf("%dx%d", size.X, size.Y), func(t *testing.T) {
			setZStatsTestResolution(t, size.X, size.Y)
			dialog := newTestZStatsDossierDialog(dossier, newWidgetTestKeyboard(0))
			dialog.switchTab(ZStatsDossierSpellsTab)
			for index, spell := range dossier.Magic.Spells {
				dialog.tabSelections[ZStatsDossierSpellsTab] = index
				dialog.resetDetailPage()
				dialog.prepareSpellDetail(dialog.layout.globalDetailRect)
				require.Equal(t, 1, dialog.detailPageCount(), spell.Name)
			}
		})
	}
}

func TestZStatsSpellSummaryStaysFixedWhileOnlyOverflowBodyPages(t *testing.T) {
	for _, size := range []image.Point{{1280, 720}, {1920, 1080}} {
		t.Run(fmt.Sprintf("%dx%d", size.X, size.Y), func(t *testing.T) {
			setZStatsTestResolution(t, size.X, size.Y)
			keyboard := newWidgetTestKeyboard(0)
			dialog := newRenderedTestZStatsDossierDialog(testZStatsDossier(), keyboard)
			dialog.switchTab(ZStatsDossierSpellsTab)
			screen := ebiten.NewImageWithOptions(dialog.layout.globalDetailRect, nil)
			t.Cleanup(screen.Dispose)
			dialog.drawSpellDetail(screen, dialog.layout.globalDetailRect)
			dialog.prepareFooter()
			require.Equal(t, 1, dialog.detailPageCount())
			require.NotContains(t, dialog.footerOutput.GetOutputStr(false), "Space")
			summary := dialog.summaryOutput.GetOutputStr(false)
			require.Contains(t, summary, "Circle: 1")
			require.Contains(t, summary, "MP cost: 1")
			require.Contains(t, summary, "Prepared: 2")
			require.Contains(t, dialog.detailOutput.GetOutputStr(false), "Formula: candidate")
			_, top, _ := dialog.catalogDetailRects(dialog.layout.globalDetailRect)
			before := make([]byte, top.Dx()*top.Dy()*4)
			screen.SubImage(top).(*ebiten.Image).ReadPixels(before)

			dialog.dossier.Magic.Spells[0].Description = strings.Repeat("A lengthy spell description. ", 100)
			screen.Clear()
			dialog.drawSpellDetail(screen, dialog.layout.globalDetailRect)
			require.Greater(t, dialog.detailPageCount(), 1)
			pressZStatsDossierKey(dialog, keyboard, ebiten.KeySpace)
			screen.Clear()
			dialog.drawSpellDetail(screen, dialog.layout.globalDetailRect)
			dialog.prepareFooter()
			require.Equal(t, 1, dialog.detailPage)
			after := make([]byte, len(before))
			screen.SubImage(top).(*ebiten.Image).ReadPixels(after)
			require.Equal(t, before, after, "paging the body must not replace the summary")
			require.Equal(t, summary, dialog.summaryOutput.GetOutputStr(false))
			require.Contains(t, dialog.footerOutput.GetOutputStr(false), "Space")
		})
	}
}

func TestZStatsCatalogQuantitiesAndUncountedEquipment(t *testing.T) {
	setZStatsTestResolution(t, 1280, 720)
	for _, quantity := range []uint16{0, 12} {
		dossier := testZStatsDossier()
		dossier.Provisions[0].Quantity = quantity
		dossier.Armory[0].Available = quantity
		dossier.Magic.Spells[0].Prepared = quantity
		dossier.Magic.Reagents[0].Quantity = quantity
		dossier.Magic.Potions[0].Quantity = quantity
		dossier.Magic.Scrolls[0].Quantity = quantity
		dossier.Items[0].Quantity = max(1, quantity)
		dialog := newTestZStatsDossierDialog(dossier, newWidgetTestKeyboard(0))
		assertZStatsCatalogRows(t, dialog, quantity)
	}
	dialog := newTestZStatsDossierDialog(testZStatsDossier(), newWidgetTestKeyboard(0))
	dialog.switchTab(ZStatsDossierCharacterTab)
	dialog.prepareListRow(image.Rect(100, 100, 1000, 140), 0, false)
	require.NotContains(t, dialog.rowOutput.GetOutputStr(false), "(")
	dialog.switchTab(ZStatsDossierItemsTab)
	require.Equal(t, 2, dialog.currentItemCount(), "unowned possessions remain filtered")
	dialog.prepareListRow(image.Rect(100, 100, 1000, 140), 1, false)
	require.Contains(t, dialog.rowOutput.GetOutputStr(false), "Blue Moonstone (1)")
}

func assertZStatsCatalogRows(t *testing.T, dialog *ZStatsDossierDialog, quantity uint16) {
	t.Helper()
	for _, test := range []struct {
		tab  ZStatsDossierTab
		want string
	}{
		{ZStatsDossierProvisionsTab, fmt.Sprintf("Food (%d)", quantity)},
		{ZStatsDossierArmoryTab, "Long Sword"},
		{ZStatsDossierSpellsTab, "In Lor"},
		{ZStatsDossierReagentsTab, fmt.Sprintf("Garlic (%d)", quantity)},
		{ZStatsDossierPotionsTab, fmt.Sprintf("Red Potion (%d)", quantity)},
		{ZStatsDossierScrollsTab, fmt.Sprintf("In Lor Scroll (%d)", quantity)},
		{ZStatsDossierItemsTab, fmt.Sprintf("Crown (%d)", max(1, quantity))},
	} {
		dialog.switchTab(test.tab)
		for _, selected := range []bool{false, true} {
			dialog.prepareListRow(image.Rect(100, 100, 1000, 140), 0, selected)
			require.Contains(t, dialog.rowOutput.GetOutputStr(false), test.want)
			if test.tab == ZStatsDossierSpellsTab {
				require.Equal(t, fmt.Sprint(quantity), dialog.spellView.count.GetOutputStr(false))
				require.Equal(t, 2, dialog.rowOutput.GetPanelContentLineCount())
			} else {
				require.Equal(t, 1, dialog.rowOutput.GetPanelContentLineCount())
			}
		}
	}
}

func TestZStatsLongCatalogNameRetainsQuantityWithoutWrapping(t *testing.T) {
	setZStatsTestResolution(t, 1280, 720)
	dossier := testZStatsDossier()
	dossier.Provisions[0].Name = strings.Repeat("Long provision name ", 20)
	dialog := newTestZStatsDossierDialog(dossier, newWidgetTestKeyboard(0))
	dialog.switchTab(ZStatsDossierProvisionsTab)
	dialog.prepareListRow(dialog.panelBodyRect(dialog.layout.globalListRect), 0, false)
	row := dialog.rowOutput.GetOutputStr(false)
	require.Contains(t, row, "Long")
	require.Contains(t, row, "(123)")
	require.Equal(t, 1, dialog.rowOutput.GetPanelContentLineCount())
}

func TestZStatsArmoryNamesUseNormalFontAndFitActualCatalogWidth(t *testing.T) {
	for _, size := range config.SupportedWindowResolutions() {
		t.Run(fmt.Sprintf("%dx%d", size.X, size.Y), func(t *testing.T) {
			setZStatsTestResolution(t, size.X, size.Y)
			dossier := testZStatsDossier()
			dossier.Armory[0].Available = 65535
			dossier.Armory[0].Readied = 6
			dialog := newRenderedTestZStatsDossierDialog(dossier, newWidgetTestKeyboard(0))
			dialog.switchTab(ZStatsDossierArmoryTab)
			row := dialog.panelBodyRect(dialog.layout.globalListRect)
			row.Max.Y = row.Min.Y + int(math.Ceil(dialog.rowOutput.GetLineSpacing()))
			screen := ebiten.NewImageWithOptions(row, nil)
			t.Cleanup(screen.Dispose)
			dialog.drawListRow(screen, row, 0, false)
			require.True(t, strings.HasSuffix(dialog.rowOutput.GetOutputStr(false), " Long Sword"))
			require.NotContains(t, dialog.rowOutput.GetOutputStr(false), "(")
			require.Same(t, dialog.bodyFont, dialog.rowOutput.Font)
			require.Equal(t, 1, dialog.rowOutput.GetPanelContentLineCount())
			rendered := renderedContentBounds(t, screen)
			require.LessOrEqual(t, rendered.Max.X, row.Max.X-dialog.scrollIndicatorWidth())

			// The same equipment in Character uses the standard list font.
			screen.Clear()
			dialog.switchTab(ZStatsDossierCharacterTab)
			dialog.drawListRow(screen, row, 2, false)
			require.Equal(t, renderedContentBounds(t, screen).Size(), rendered.Size())

			dialog.switchTab(ZStatsDossierArmoryTab)
			dialog.prepareArmoryDetail(dialog.layout.globalDetailRect)
			require.Contains(t, dialog.summaryOutput.GetOutputStr(false), "65535 / 6")
		})
	}
}

func TestZStatsCompactSpellDetailsRetainUsableOverflowArea(t *testing.T) {
	for _, size := range []image.Point{{320, 200}, {640, 360}} {
		t.Run(size.String(), func(t *testing.T) {
			setZStatsTestResolution(t, size.X, size.Y)
			dossier := testZStatsDossier()
			dossier.Magic.Spells[0].Name = "In Vas Grav Corp"
			dossier.Magic.Spells[0].Prepared = 65535
			dossier.Magic.Spells[0].Description = strings.Repeat("The spell description remains accessible. ", 40)
			dialog := newTestZStatsDossierDialog(dossier, newWidgetTestKeyboard(0))
			dialog.switchTab(ZStatsDossierSpellsTab)
			dialog.prepareSpellDetail(dialog.layout.globalDetailRect)
			portrait, summary, body := dialog.catalogDetailRects(dialog.layout.globalDetailRect)
			summary.Max.Y = summary.Min.Y + int(math.Ceil(dialog.summaryOutput.GetTotalHeightForLines(dialog.summaryOutput.GetPanelContentLineCount())))
			body.Min.Y = max(portrait.Max.Y, summary.Max.Y) + dialog.layout.padding
			require.GreaterOrEqual(t, body.Dy(), int(math.Ceil(dialog.detailOutput.GetFontHeight())))
			require.Greater(t, dialog.detailPageCount(), 1)
		})
	}
}
