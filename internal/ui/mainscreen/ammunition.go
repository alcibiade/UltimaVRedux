package mainscreen

import (
	"fmt"
	"image/color"
	"strconv"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// AmmunitionLowThreshold is the inclusive Redux presentation warning threshold.
const AmmunitionLowThreshold uint16 = 20

// AmmunitionCounts is a detached, read-only snapshot in stable Arrows/Quarrels order.
type AmmunitionCounts [2]uint16

// SnapshotAmmunition reads shared ammunition balances without allocating inventory entries.
func SnapshotAmmunition(inventory *party_state.Inventory) AmmunitionCounts {
	return AmmunitionCounts{inventory.Equipment.Get(references.Arrows), inventory.Equipment.Get(references.Quarrels)}
}

func ammunitionValue(count uint16) (string, color.Color) {
	switch {
	case count == 0:
		return "0 OUT", ucolor.SupplyEmpty
	case count <= AmmunitionLowThreshold:
		return fmt.Sprintf("%d LOW", count), ucolor.SupplyLow
	default:
		return strconv.FormatUint(uint64(count), 10), ucolor.UITextPrimary.RGBA()
	}
}

// ShortageNotices compares only final balances. Increases within low stock do not
// re-arm a low warning; every positive balance permits the next depletion notice.
func (current AmmunitionCounts) ShortageNotices(previous AmmunitionCounts) []string {
	var notices []string
	for index, name := range [2]string{"arrows", "quarrels"} {
		if previous[index] > 0 && current[index] == 0 {
			notices = append(notices, "Out of "+name+"!")
		} else if previous[index] > AmmunitionLowThreshold && current[index] <= AmmunitionLowThreshold && current[index] > 0 {
			fullName := [2]string{"Arrows", "Quarrels"}[index]
			notices = append(notices, fmt.Sprintf("%s running low: %d left.", fullName, current[index]))
		}
	}

	return notices
}
