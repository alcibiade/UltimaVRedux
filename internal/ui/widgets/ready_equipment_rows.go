package widgets

import (
	"fmt"
	"image"
	"math"
	"unicode/utf8"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	readyItemPrefixRunes = 5 // Three marker cells, one bitmap icon, and a space.
	readyNoneLabel       = "None"
)

// readySlotText describes the actual loadout, including the otherwise empty slot
// reserved by a two-handed weapon. It never uses a hypothetical replacement.
func readySlotText(character *party_state.PlayerCharacter, slot references.EquipmentSlot) (string, string) {
	equipment := character.GetEquippedItem(slot)
	if slot != references.SlotWeapon && slot != references.SlotShield {
		if equipment == references.NoEquipment {
			return "", readyNoneLabel
		}

		return "", equipment.ShortString()
	}
	if equipment != references.NoEquipment {
		name := equipment.ShortString()
		if equipment.IsTwoHanded() {
			name += " (2H)"
		}

		return "Occupied", name
	}
	for _, hand := range []references.EquipmentSlot{references.SlotWeapon, references.SlotShield} {
		other := character.GetEquippedItem(hand)
		if other != references.NoEquipment && other.IsTwoHanded() {
			return "Occupied", "By " + other.ShortString() + " (2H)"
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

// populateItemRow preserves the marker, bitmap icon and complete quantity. Names
// use the catalog's established short form only when the full name will not fit.
func (d *ReadyEquipmentDialog) populateItemRow(item availableEquipment, selected bool, width int) float64 {
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
	suffixWidth, _ := d.font.MeasureString(suffix)
	nameWidth := max(0, width-int(math.Ceil(d.itemsOutput.MeasureLineSegments(line.GetSegments())+suffixWidth)))
	name := item.Equipment.String()
	measured, _ := d.font.MeasureString(name)
	if measured > float64(nameWidth) {
		name = item.Equipment.ShortString()
	}
	name = FitDialogLine(name, d.font, nameWidth)
	line.Text(name, nameColor).Text(suffix, quantityColor)
	d.itemsOutput.SetMaxCharsPerLine(readyItemPrefixRunes + utf8.RuneCountInString(name+suffix))
	rowWidth := d.itemsOutput.MeasureLineSegments(line.GetSegments())
	line.Done()

	return rowWidth
}
