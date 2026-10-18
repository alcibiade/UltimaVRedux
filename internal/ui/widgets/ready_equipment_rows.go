package widgets

import (
	"fmt"
	"image"
	"math"
	"unicode/utf8"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/text"
)

const (
	readyItemPrefixRunes = 5 // Three marker cells, one bitmap icon, and a space.
	readyNoneLabel       = "None"
	readyMinSlotsShare   = 0.27
	readyMinItemsShare   = 0.38
)

// readySlotText describes the actual loadout, including the otherwise empty slot
// reserved by a two-handed weapon. It never uses a hypothetical replacement.
func readySlotText(character *party_state.PlayerCharacter, slot references.EquipmentSlot) (string, string) {
	equipment := character.GetEquippedItem(slot)
	if slot != references.SlotWeapon && slot != references.SlotShield {
		if equipment == references.NoEquipment {
			return "", readyNoneLabel
		}

		return "", equipment.String()
	}
	if equipment != references.NoEquipment {
		if equipment.IsTwoHanded() {
			return "Occupied (2H)", equipment.String()
		}

		return "Occupied", equipment.String()
	}
	for _, hand := range []references.EquipmentSlot{references.SlotWeapon, references.SlotShield} {
		other := character.GetEquippedItem(hand)
		if other != references.NoEquipment && other.IsTwoHanded() {
			return "Reserved (2H)", other.String()
		}
	}

	return "Free", ""
}

func (d *ReadyEquipmentDialog) itemRowHeight() int {
	return max(1, int(math.Ceil(max(d.itemsOutput.GetLineSpacing(), d.itemsOutput.GetFontHeight()))))
}

func (d *ReadyEquipmentDialog) itemScrollGutter() int {
	width, _ := d.font.MeasureString("v")

	return int(math.Ceil(width)) + itemDialogTextPadding
}

func (d *ReadyEquipmentDialog) itemTextRect(bounds image.Rectangle, top int) image.Rectangle {
	padding := itemDialogTextPadding + readyDetailInset

	return image.Rect(bounds.Min.X+padding, top, bounds.Max.X-padding-d.itemScrollGutter(), top+d.itemRowHeight())
}

// readyColumnWidths reserves the complete catalog identities, bitmap icons,
// markers, largest representable quantity, and scroll gutter before assigning the
// remaining width to the wrapping/paged comparison pane.
func (d *ReadyEquipmentDialog) readyColumnWidths(dialogWidth int) (int, int) {
	// Bitmap icon advances scale with Output line spacing, not just its font.
	// Use the same typography as initializeOutputs rather than a unit-height probe.
	measurement := text.NewOutput(d.font, d.font.TextFace.Size*readyLineSpacingMultiplier,
		1, d.font.CharsForWidth(dialogWidth))
	slotWidth, _ := d.font.MeasureString("Hand 2: Reserved (2H)")
	listWidth := 0.0
	for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
		nameWidth, _ := d.font.MeasureString(equipment.String())
		slotWidth = max(slotWidth, nameWidth)
		measurement.Reset()
		// Match the rendered segment boundaries: separately shaped text has
		// different advances from one merged marker/name/quantity string.
		line := measurement.AppendLine().Text("[", DialogControlHintColor).
			Text("E", readyEquippedMarkerColor).Text("]", DialogControlHintColor).
			EquipmentIconWithColor(equipment, readySlotIconColor).Text(" ", DialogItemNameColor).
			Text(equipment.String(), DialogItemNameColor).Text(" (65535)", DialogItemQuantityColor)
		listWidth = max(listWidth, measurement.MeasureLineSegments(line.GetSegments()))
		line.Done()
	}
	// GetInteriorBounds reserves a proportional inset on both sides of each
	// individual pane, not the full dialog's border width.
	interiorShare := 1 - readyCenterDivisor*float64(LayoutSpacingMedium)
	padding := readyCenterDivisor * (itemDialogTextPadding + readyDetailInset)
	slots := max(int(float64(dialogWidth)*readyMinSlotsShare),
		int(math.Ceil((slotWidth+float64(padding))/interiorShare)))
	items := max(int(float64(dialogWidth)*readyMinItemsShare),
		int(math.Ceil((listWidth+float64(padding+d.itemScrollGutter()))/interiorShare)))

	return slots, items
}

// populateItemRow preserves the full identity, marker, icon and quantity. The
// catalog-measured column budget reserves their combined width in advance.
func (d *ReadyEquipmentDialog) populateItemRow(item availableEquipment, selected bool, _ int) float64 {
	nameColor, quantityColor, iconColor := DialogItemNameColor, DialogItemQuantityColor, readySlotIconColor
	markColor, bracketColor := readyEquippedMarkerColor, DialogControlHintColor
	if item.Equipped {
		nameColor, quantityColor = readyEquippedMarkerColor, readyEquippedMarkerColor
	} else if !item.CanEquip {
		nameColor, quantityColor, iconColor = readyUnequippableColor, readyUnequippableColor, readyUnequippableColor
	}
	if selected {
		nameColor, quantityColor, iconColor = DialogSelectedText, DialogSelectedText, DialogSelectedText
		markColor, bracketColor = DialogSelectedCheckedMarkColor, DialogSelectedText
		if !item.CanEquip {
			nameColor = DialogSelectedUnavailableItemColor
		}
	}
	d.itemsOutput.Reset()
	line := d.itemsOutput.AppendLine()
	if item.Equipped {
		line.Text("[", bracketColor).Text("E", markColor).Text("]", bracketColor)
	} else {
		line.Text("   ", nameColor)
	}
	line.EquipmentIconWithColor(item.Equipment, iconColor).Text(" ", nameColor)
	suffix := fmt.Sprintf(" (%d)", item.Quantity)
	name := item.Equipment.String()
	line.Text(name, nameColor).Text(suffix, quantityColor)
	d.itemsOutput.SetMaxCharsPerLine(readyItemPrefixRunes + utf8.RuneCountInString(name+suffix))
	rowWidth := d.itemsOutput.MeasureLineSegments(line.GetSegments())
	line.Done()

	return rowWidth
}
