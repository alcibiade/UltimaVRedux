package debugging

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestInventoryLockerLabelsKeepStableChoiceValues(t *testing.T) {
	items := references.NewInventoryItemsReferences()
	for index, choice := range inventoryPotionChoices() {
		require.Equal(t, strconv.Itoa(index), choice.Value)
		require.Equal(t, items.Potion[references.Potion(index)].FullDisplayName(), choice.Label)
	}
	for index, choice := range inventoryScrollChoices() {
		require.Equal(t, strconv.Itoa(index), choice.Value)
		require.Equal(t, items.Scroll[references.Scroll(index)].FullDisplayName(), choice.Label)
	}
	for index, choice := range inventoryEquipmentChoices() {
		require.Equal(t, strconv.Itoa(index), choice.Value)
		require.Equal(t, references.Equipment(index).String(), choice.Label)
	}
	wanted := map[string]string{"wooden_box": "Sandalwood Box", "carpet": "Magic Carpet", "hms_cape": "H.M.S. Cape Plans"}
	for _, choice := range questItemChoices() {
		if label, selected := wanted[choice.Value]; selected {
			require.Equal(t, label, choice.Label)
			delete(wanted, choice.Value)
		}
	}
	require.Empty(t, wanted)
}
