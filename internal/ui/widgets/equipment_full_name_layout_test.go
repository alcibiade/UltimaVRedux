package widgets

import (
	"image"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestReadyFullNamesRetainStrengthBudgetAndSelectedDetails(t *testing.T) {
	state, dialog, _ := newReadyReviewFixture(t)
	for _, equipment := range []references.Equipment{references.TwoHHammer, references.RingRegeneration, references.ChaosSword} {
		state.PartyState.Inventory.Equipment.Set(equipment, 99)
	}
	dialog.updateAvailableItems()
	for _, size := range []image.Point{{1280, 720}, {1920, 1080}, {3840, 2160}} {
		setResponsiveDialogTestResolution(t, size.X, size.Y)
		dialog.ensureLayout()
		for _, equipment := range []references.Equipment{references.TwoHHammer, references.RingRegeneration, references.ChaosSword} {
			selectReadyReviewItem(t, dialog, equipment)
			dialog.populateDetailOutput()
			content := strings.Join(strings.Fields(dialog.detailOutput.GetOutputStr(false)), " ")
			require.Contains(t, content, equipment.String())
			require.Equal(t, 2, strings.Count(content, equipment.String()), "heading and Strength budget retain complete identity")
			require.NotContains(t, content, "...")
			for _, line := range strings.Split(dialog.detailOutput.GetOutputStr(false), "\n") {
				width, _ := dialog.font.MeasureString(strings.TrimSpace(line))
				require.LessOrEqual(t, width, float64(dialog.detailContentRect().Dx()))
			}
			dialog.itemListNav.SetIndex(len(dialog.availableItems) - 1)
			start, end := dialog.itemListNav.GetVisibleRange()
			require.Greater(t, end, start)
			require.Equal(t, len(dialog.availableItems), end)
		}
	}
}

func TestArmsFullCatalogRowsFitWithIconsCountsAndScrolling(t *testing.T) {
	for _, size := range []image.Point{{1280, 720}, {1920, 1080}, {3840, 2160}} {
		t.Run(size.String(), func(t *testing.T) {
			setMerchantShellTestResolution(t, size.X, size.Y)
			state := testArmsShopState()
			state.SellQuotes = nil
			for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
				price := equipment.GetBasePrice()
				sellable := price > 0 && !equipment.IsAmmo()
				quote := game_state.ArmsSellQuote{Equipment: equipment, BasePrice: price, Gold: state.Gold,
					QuantityBefore: 99, ReadiedQuantity: 6, Sellable: sellable}
				if sellable {
					quote.UnitOffer, quote.MaxQuantity = 1, 99
				}
				state.SellQuotes = append(state.SellQuotes, quote)
			}
			dialog := newTestArmsShopDialog(newWidgetTestKeyboard(0), state)
			pressArmsShopKey(dialog, dialog.keyboard, ebiten.KeyS)
			screen := ebiten.NewImage(size.X, size.Y)
			t.Cleanup(screen.Dispose)
			for index, row := range dialog.GetRows() {
				require.Equal(t, row.Equipment.String(), row.Name)
				require.Equal(t, uint16(99), row.Quantity)
				require.Equal(t, uint16(6), row.Readied)
				dialog.activeMode().list.SetIndex(index)
				dialog.populateWorkspaceOutput()
				require.Contains(t, strings.Join(strings.Fields(dialog.GetDetailText()), " "), row.Name)
				bounds := dialog.rowBounds[0]
				cell := dialog.cellContentBounds(bounds, dialog.columns[0], row.Equipment)
				screen.Clear()
				dialog.drawCell(screen, bounds, dialog.columns[0], " "+row.Name, DialogItemNameColor, etext.AlignStart, row.Equipment)
				content := strings.Join(strings.Fields(dialog.rowOutput.GetOutputStr(false)), " ")
				require.Contains(t, content, row.Name)
				require.NotContains(t, content, "...")
				require.LessOrEqual(t, dialog.rowOutput.GetTotalHeightForLines(dialog.rowOutput.GetPanelContentLineCount()), float64(cell.Dy()))
				require.True(t, renderedContentBounds(t, screen).In(cell), row.Name)
			}
			require.Positive(t, dialog.GetScrollOffset())
			require.Equal(t, references.TwoHSword, dialog.GetSelectedEquipment())
			require.Equal(t, GetScaledFont(FontSizeMainScreen).TextFace.Size, dialog.rowFont.TextFace.Size)
		})
	}
}
