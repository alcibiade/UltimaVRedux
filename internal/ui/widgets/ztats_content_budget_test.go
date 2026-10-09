package widgets

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
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
			require.Contains(t, d.footerOutput.GetOutputStr(false), "Tab/Shift-Tab")
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
