package mainscreen

import (
	"image"
	"image/color"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/stretchr/testify/require"
)

func TestAmmunitionClassificationAndReadOnlySnapshot(t *testing.T) {
	var inventory party_state.Inventory
	require.Equal(t, AmmunitionCounts{}, SnapshotAmmunition(&inventory))
	require.Equal(t, party_state.Inventory{}, inventory, "rendering must not allocate mutable quantity storage")
	for _, test := range []struct {
		count uint16
		value string
		color color.Color
	}{
		{0, "0 OUT", ucolor.SupplyEmpty}, {1, "1 LOW", ucolor.SupplyLow},
		{19, "19 LOW", ucolor.SupplyLow}, {20, "20 LOW", ucolor.SupplyLow},
		{21, "21", ucolor.UITextPrimary.RGBA()}, {99, "99", ucolor.UITextPrimary.RGBA()},
	} {
		value, valueColor := ammunitionValue(test.count)
		require.Equal(t, test.value, value)
		require.Equal(t, test.color, valueColor)
	}
	inventory.Equipment.Set(references.Arrows, 20)
	inventory.Equipment.Set(references.Quarrels, 99)
	require.Equal(t, AmmunitionCounts{20, 99}, SnapshotAmmunition(&inventory))
}

func TestAmmunitionShortageTransitions(t *testing.T) {
	for _, test := range []struct {
		previous, current uint16
		notice            string
	}{
		{21, 20, "Arrows running low: 20 left."}, {35, 18, "Arrows running low: 18 left."},
		{20, 19, ""}, {19, 1, ""}, {20, 20, ""}, {1, 0, "Out of arrows!"},
		{99, 0, "Out of arrows!"}, {0, 0, ""}, {0, 5, ""}, {5, 10, ""},
		{18, 99, ""}, {0, 99, ""}, {5, 0, "Out of arrows!"}, {99, 20, "Arrows running low: 20 left."},
	} {
		notices := (AmmunitionCounts{test.current, 99}).ShortageNotices(AmmunitionCounts{test.previous, 99})
		if test.notice == "" {
			require.Empty(t, notices, "%d -> %d", test.previous, test.current)
		} else {
			require.Equal(t, []string{test.notice}, notices)
		}
	}
	require.Equal(t, []string{"Arrows running low: 20 left.", "Quarrels running low: 18 left."},
		(AmmunitionCounts{20, 18}).ShortageNotices(AmmunitionCounts{21, 35}))
	require.Equal(t, []string{"Out of arrows!", "Out of quarrels!"},
		(AmmunitionCounts{}).ShortageNotices(AmmunitionCounts{1, 99}))
}

func TestAmmunitionCellsFitNormalTypographyAndKeepLabelsStable(t *testing.T) {
	for _, size := range []image.Point{{1280, 720}, {1365, 768}, {1920, 1080}, {2560, 1440}, {3840, 2160}} {
		t.Run(size.String(), func(t *testing.T) {
			summary := provisionTestSummary(size)
			cells := summary.ammunitionCells(summary.rows()[2])
			state := maximumProvisionTestState()
			for _, count := range []uint16{0, 1, 19, 20, 21, 99} {
				state.PartyState.Inventory.Equipment.Set(references.Arrows, count)
				state.PartyState.Inventory.Equipment.Set(references.Quarrels, count)
				resources := provisionAmmunitionResources(state)
				for index, resource := range resources {
					cell := cells[index]
					value, statusColor := ammunitionValue(count)
					require.Equal(t, [2]string{"Ar", "Qr"}[index], resource.label)
					require.Equal(t, ucolor.EGALightCyan, resource.labelColor, "both ammo labels stay cyan at every severity")
					require.Equal(t, ucolor.UITextPrimary.RGBA(), resource.valueColor)
					require.Equal(t, statusColor, resource.statusColor)
					if resource.status == "" {
						require.Equal(t, value, resource.value)
					} else {
						require.Equal(t, value, resource.value+" "+resource.status)
					}
					require.LessOrEqual(t, measuredProvisionWidth(summary.ultimaFont, resource.value), cell.value.Dx())
					for _, status := range []string{"LOW", "OUT"} {
						require.LessOrEqual(t, measuredProvisionWidth(summary.ultimaFont, status), cell.status.Dx())
					}
				}
			}
		})
	}
}
