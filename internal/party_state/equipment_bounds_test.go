package party_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

func TestSetEquippedItemRejectsValuesOutsideLegacyByteWithoutMutation(t *testing.T) {
	character := &PlayerCharacter{}
	character.SetEquippedItem(references.SlotWeapon, references.Dagger)
	for _, invalid := range []references.Equipment{-1, 256} {
		assert.Panics(t, func() { character.SetEquippedItem(references.SlotWeapon, invalid) })
		assert.Equal(t, references.Dagger, character.GetEquippedItem(references.SlotWeapon))
	}
	character.SetEquippedItem(references.SlotWeapon, references.NoEquipment)
	assert.Equal(t, references.NoEquipment, character.GetEquippedItem(references.SlotWeapon))
}
