package widgets

import (
	"fmt"
	"image"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/stretchr/testify/require"
)

func TestReadyCompactTypographyAndThreePaneColumnBudget(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	state, _ := game_state.NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
	dialog := NewReadyEquipmentDialog(newWidgetTestKeyboard(-1), &sprites.SpriteSheet{}, state, game_state.ReadyEquipmentRequest{})
	require.Equal(t, GetScaledFont(FontSizeLarge).TextFace.Size, dialog.font.TextFace.Size)
	require.Equal(t, GetScaledFont(FontSizeDialogTitle).TextFace.Size, dialog.titleFont.TextFace.Size)
	require.InDelta(t, 1280*0.9*0.38, dialog.itemsBorder.GetBounds().Dx(), 8)
	require.InDelta(t, 1280*0.9*0.35, dialog.detailBorder.GetBounds().Dx(), 8)
}

func TestReadyEveryGearRowFitsWithMarkersIconsAndFullQuantity(t *testing.T) {
	state, _ := game_state.NewIntegrationTestBuilder(t).WithLocation(references.Britain).WithSystemCallbacks().Build()
	for _, size := range []image.Point{{1280, 720}, {1366, 768}, {1920, 1080}, {2560, 1440}, {3840, 2160}} {
		t.Run(size.String(), func(t *testing.T) {
			setResponsiveDialogTestResolution(t, size.X, size.Y)
			dialog := NewReadyEquipmentDialog(newWidgetTestKeyboard(-1), &sprites.SpriteSheet{}, state, game_state.ReadyEquipmentRequest{})
			bounds := dialog.itemsBorder.GetInteriorBounds()
			row := dialog.itemTextRect(bounds, bounds.Min.Y)
			require.Less(t, row.Max.X, bounds.Max.X-dialog.itemScrollGutter())
			for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
				if equipment.GetSlotType() != references.SlotNone {
					equippedWidth := dialog.slotsBorder.GetInteriorBounds().Dx() - 2*(itemDialogTextPadding+readyDetailInset)
					name := FitDialogLine("  "+equipment.ShortString(), dialog.font, equippedWidth)
					require.NotContains(t, name, "...", equipment.ShortString())
					measured, _ := dialog.font.MeasureString(name)
					require.LessOrEqual(t, measured, float64(equippedWidth))
				}
				for _, quantity := range []uint16{99, 65535} {
					for _, selected := range []bool{false, true} {
						item := availableEquipment{Equipment: equipment, Quantity: quantity, Equipped: true, CanEquip: true}
						width := dialog.populateItemRow(item, selected, row.Dx())
						content := dialog.itemsOutput.GetOutputStr(false)
						require.LessOrEqual(t, width, float64(row.Dx()), equipment.String())
						require.Contains(t, content, "[E]")
						require.Contains(t, content, fmt.Sprintf("(%d)", quantity))
						require.NotContains(t, content, "...")
						require.NotContains(t, content, "\n")
						require.Equal(t, 1, dialog.itemsOutput.GetPanelContentLineCount())
					}
				}
			}
		})
	}
}

func TestReadyHandAvailabilityAccountsForTwoHandedReservation(t *testing.T) {
	state, dialog, _ := newReadyReviewFixture(t)
	character := state.PartyState.GetAvatar()
	for _, test := range []struct {
		name                                             string
		first, second                                    references.Equipment
		firstStatus, firstText, secondStatus, secondText string
	}{
		{"empty", references.NoEquipment, references.NoEquipment, "Free", "", "Free", ""},
		{"one weapon", references.SilverSword, references.NoEquipment, "Occupied", "Silver Swd", "Free", ""},
		{"shield only", references.NoEquipment, references.SmallShield, "Free", "", "Occupied", "Sm. Shield"},
		{"weapon and shield", references.SilverSword, references.SmallShield, "Occupied", "Silver Swd", "Occupied", "Sm. Shield"},
		{"two weapons", references.SilverSword, references.LongSword, "Occupied", "Silver Swd", "Occupied", "Long Sword"},
		{"two handed", references.Halberd, references.NoEquipment, "Occupied", "Halberd (2H)", "Occupied", "By Halberd (2H)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			character.SetEquippedItem(references.SlotWeapon, test.first)
			character.SetEquippedItem(references.SlotShield, test.second)
			status, label := readySlotText(character, references.SlotWeapon)
			require.Equal(t, test.firstStatus, status)
			require.Equal(t, test.firstText, label)
			status, label = readySlotText(character, references.SlotShield)
			require.Equal(t, test.secondStatus, status)
			require.Equal(t, test.secondText, label)
		})
	}
	// Removing the actual two-handed weapon frees both hands immediately.
	dialog.updateAvailableItems()
	selectReadyReviewItem(t, dialog, references.Halberd)
	dialog.toggleEquipSelected()
	commitReadyDialogIntent(t, dialog, state)
	for _, hand := range []references.EquipmentSlot{references.SlotWeapon, references.SlotShield} {
		status, _ := readySlotText(character, hand)
		require.Equal(t, "Free", status)
	}
}

func TestReadyOccupiedHandLabelsFitCompactPane(t *testing.T) {
	state, dialog, _ := newReadyReviewFixture(t)
	character := state.PartyState.GetAvatar()
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)
	width := dialog.slotsBorder.GetInteriorBounds().Dx() - 2*(itemDialogTextPadding+readyDetailInset)
	for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
		if !equipment.IsTwoHanded() {
			continue
		}
		character.SetEquippedItem(references.SlotWeapon, equipment)
		for _, hand := range []references.EquipmentSlot{references.SlotWeapon, references.SlotShield} {
			status, label := readySlotText(character, hand)
			for _, line := range []string{hand.DisplayName() + ": " + status, label} {
				measured, _ := dialog.font.MeasureString(line)
				require.LessOrEqual(t, measured, float64(width), line)
			}
		}
	}
}
