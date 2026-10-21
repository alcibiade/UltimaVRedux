package widgets

import (
	"fmt"
	"image"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/text"
)

// Captured from the unmodified dossier before the two-row tabs/strip change.
// Values measure usable content, not the outer frame; no legacy renderer is kept.
func TestZStatsContentBudgetPreservesBaseline(t *testing.T) {
	baselines := []struct {
		list, rows, body, spellList, spellRows, spellBody, portrait, stats, item int
	}{
		{339, 17, 265, 301, 7, 263, 87, 195, 61},
		{358, 17, 278, 318, 7, 277, 94, 206, 63},
		{507, 18, 397, 451, 8, 393, 130, 292, 90},
		{676, 18, 528, 602, 8, 524, 174, 390, 120},
		{1008, 18, 784, 896, 8, 779, 262, 579, 177},
	}
	for index, size := range config.SupportedWindowResolutions() {
		baseline := baselines[index]
		setZStatsTestResolution(t, size.X, size.Y)
		d := newTestZStatsDossierDialog(testZStatsDossier(), newWidgetTestKeyboard(0))
		t.Cleanup(d.outerBorder.Dispose)
		d.switchTab(ZStatsDossierCharacterTab)
		character := d.panelBodyRect(d.layout.characterListRect)
		stats := insetZStatsRect(d.layout.characterStatsRect, d.layout.padding*2)
		_, _, item := d.catalogDetailRects(d.layout.characterItemRect)
		d.switchTab(ZStatsDossierArmoryTab)
		list := d.panelBodyRect(d.layout.globalListRect)
		portrait, summary, body := d.catalogDetailRects(d.layout.globalDetailRect)
		require.GreaterOrEqual(t, list.Dy(), baseline.list, size)
		require.GreaterOrEqual(t, character.Dy(), baseline.list, size)
		require.GreaterOrEqual(t, stats.Dy(), baseline.stats, size)
		require.GreaterOrEqual(t, item.Dy(), baseline.item, size)
		require.GreaterOrEqual(t, d.visibleListRows(d.layout.globalListRect), baseline.rows, size)
		require.GreaterOrEqual(t, body.Dy(), baseline.body, size)
		require.GreaterOrEqual(t, d.detailOutput.GetLineCapacity(body.Dy()), 14, size)
		require.GreaterOrEqual(t, portrait.Dy(), baseline.portrait, size)
		require.GreaterOrEqual(t, summary.Dy(), baseline.portrait, size)
		outer := d.outerBorder.GetBounds()
		for tab := ZStatsDossierCharacterTab; tab < zStatsDossierTabCount; tab++ {
			d.switchTab(tab)
			d.ensureLayout()
			d.prepareFooter()
			require.Equal(t, outer, d.outerBorder.GetBounds())
			for _, frame := range d.frameRects() {
				require.True(t, frame.In(outer), "%s: %v outside %v", d.tabTitle(tab), frame, outer)
			}
			require.LessOrEqual(t, d.footerOutput.GetTotalHeightForLines(d.footerOutput.GetPanelContentLineCount()),
				float64(d.layout.footerRect.Inset(d.layout.padding).Dy()))
			require.Contains(t, d.footerOutput.GetOutputStr(false), "Tab/Shift+Tab")
			require.Contains(t, d.footerOutput.GetOutputStr(false), "Close")
		}
		d.switchTab(ZStatsDossierSpellsTab)
		d.prepareSpellDetail(d.layout.globalDetailRect)
		list = d.panelBodyRect(d.layout.globalListRect)
		portrait, summary, body = d.catalogDetailRects(d.layout.globalDetailRect)
		summary.Max.Y = summary.Min.Y + int(math.Ceil(d.summaryOutput.GetTotalHeightForLines(d.summaryOutput.GetPanelContentLineCount())))
		body.Min.Y = max(portrait.Max.Y, summary.Max.Y) + d.layout.padding
		require.Equal(t, outer, d.outerBorder.GetBounds())
		require.GreaterOrEqual(t, list.Dy(), baseline.spellList, size)
		require.GreaterOrEqual(t, d.visibleListRows(d.layout.globalListRect), baseline.spellRows, size)
		require.GreaterOrEqual(t, d.layout.globalDetailRect.Dy(), baseline.list+int(math.Ceil(d.titleOutput.GetLineSpacing()))+d.layout.padding*4, size)
		require.GreaterOrEqual(t, body.Dy(), baseline.spellBody, size)
		require.GreaterOrEqual(t, d.detailOutput.GetLineCapacity(body.Dy()), 14, size)
		require.GreaterOrEqual(t, portrait.Dy(), baseline.portrait, size)
		require.GreaterOrEqual(t, summary.Dy(), baseline.portrait, size)
		t.Logf("%v usable list=%v body=%v rows=%d detail capacity=%d", size, list.Size(), body.Size(), d.visibleListRows(d.layout.globalListRect), d.detailOutput.GetLineCapacity(body.Dy()))
	}
}

func TestZStatsMoonstoneHeaderIsBoundedAndReclaimsPortraitBudgetAfterResize(t *testing.T) {
	setZStatsTestResolution(t, 320, 200)
	dossier := testZStatsDossier()
	items := references.NewInventoryItemsReferences()
	dossier.Items = nil
	for _, stone := range references.MoonstoneCatalog() {
		item := items.GetReferenceByItem(stone)
		dossier.Items = append(dossier.Items, game_state.ZStatsOwnedItem{
			ID: stone.ID(), Name: item.FullDisplayName(), Category: "Moonstone", Quantity: 1,
			Description: strings.Repeat("The existing moonstone lore remains readable. ", 80),
			MoonPhase:   game_state.ZStatsMoonPhase{Present: true, Phase: datetime.MoonPhase(stone.ID())},
		})
	}
	dialog := newTestZStatsDossierDialog(dossier, newWidgetTestKeyboard(0))
	t.Cleanup(func() { dialog.outerBorder.Dispose() })
	dialog.switchTab(ZStatsDossierItemsTab)
	for _, size := range append([]config.ScreenResolution{{X: 320, Y: 200}}, config.SupportedWindowResolutions()...) {
		t.Run(fmt.Sprintf("%dx%d", size.X, size.Y), func(t *testing.T) {
			setZStatsTestResolution(t, size.X, size.Y)
			dialog.ensureLayout()
			outer := dialog.outerBorder.GetBounds()
			for index, item := range dossier.Items {
				dialog.setCurrentSelection(index)
				dialog.resetDetailPage()
				prepared := dialog.prepareItemDetail(dialog.layout.globalDetailRect)
				require.True(t, prepared.portrait.Empty(), "moonstones must not allocate a generic portrait")
				require.False(t, prepared.phaseIcon.Empty())
				require.True(t, prepared.phaseIcon.In(dialog.layout.globalDetailRect))
				require.True(t, prepared.summary.In(dialog.layout.globalDetailRect))
				require.True(t, prepared.body.In(dialog.layout.globalDetailRect))
				require.GreaterOrEqual(t, prepared.summary.Min.X, prepared.phaseIcon.Max.X+dialog.layout.padding)
				require.GreaterOrEqual(t, prepared.body.Min.Y, max(prepared.phaseIcon.Max.Y, prepared.summary.Max.Y)+dialog.layout.padding)
				unit := dialog.pixelUnit()
				require.Zero(t, prepared.phaseIcon.Min.X%unit)
				require.Zero(t, prepared.phaseIcon.Min.Y%unit)
				headerBound := max(int(math.Ceil(dialog.rowOutput.GetTotalHeightForLines(2))), text.BitmapCharHeight+4*unit)
				require.LessOrEqual(t, prepared.phaseIcon.Dy(), headerBound)
				require.Equal(t, prepared.phaseIcon.Dx(), prepared.phaseIcon.Dy())
				require.Greater(t, dialog.detailPageCount(), 1)
				require.Equal(t, outer, dialog.outerBorder.GetBounds())
				heading := strings.Join(strings.Fields(dialog.summaryOutput.GetOutputStr(false)), " ")
				require.Contains(t, heading, item.Name)
				require.Contains(t, heading, "Moonstone · Quantity: 1")
				require.Equal(t, 2, dialog.summaryOutput.GetPanelContentLineCount(), "canonical phase heading and category/quantity form two lines")
				// A much taller pane never grows the art, and leaves more room for lore.
				taller := dialog.layout.globalDetailRect
				taller.Max.Y += size.Y
				expanded := dialog.prepareItemDetail(taller)
				require.Equal(t, prepared.phaseIcon, expanded.phaseIcon)
				require.Equal(t, prepared.summary, expanded.summary)
				require.Equal(t, prepared.body.Min, expanded.body.Min)
				require.Equal(t, prepared.body.Dy()+size.Y, expanded.body.Dy())
				_, _, ordinaryBody := dialog.catalogDetailRects(dialog.layout.globalDetailRect)
				require.Greater(t, prepared.body.Dy(), ordinaryBody.Dy(), "unused portrait space belongs to lore")
			}
		})
	}
}

func TestZStatsOtherCatalogsKeepPortraitAllocation(t *testing.T) {
	setZStatsTestResolution(t, 1280, 720)
	dialog := newTestZStatsDossierDialog(testZStatsDossier(), newWidgetTestKeyboard(0))
	t.Cleanup(dialog.outerBorder.Dispose)
	for _, test := range []struct {
		tab     ZStatsDossierTab
		prepare func(image.Rectangle) zStatsCatalogDetailLayout
	}{
		{ZStatsDossierProvisionsTab, dialog.prepareProvisionDetail},
		{ZStatsDossierArmoryTab, dialog.prepareArmoryDetail},
		{ZStatsDossierSpellsTab, dialog.prepareSpellDetail},
		{ZStatsDossierReagentsTab, dialog.prepareMagicInventoryDetail},
		{ZStatsDossierPotionsTab, dialog.prepareMagicInventoryDetail},
		{ZStatsDossierScrollsTab, dialog.prepareMagicInventoryDetail},
		{ZStatsDossierItemsTab, dialog.prepareItemDetail},
	} {
		dialog.switchTab(test.tab)
		prepared := test.prepare(dialog.layout.globalDetailRect)
		portrait, _, _ := dialog.catalogDetailRects(dialog.layout.globalDetailRect)
		require.Equal(t, portrait, prepared.portrait, dialog.tabTitle(test.tab))
		require.True(t, prepared.phaseIcon.Empty())
	}
	dialog.switchTab(ZStatsDossierCharacterTab)
	prepared := dialog.prepareCharacterItemDetail()
	portrait, _, _ := dialog.catalogDetailRects(dialog.layout.characterItemRect)
	require.Equal(t, portrait, prepared.portrait)
	require.True(t, prepared.phaseIcon.Empty())
}
