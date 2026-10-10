package party_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsumeMealTimeFoodRejectsImpossibleDinerCountBeforeMutation(t *testing.T) {
	inventory := &Inventory{}
	inventory.Provisions.Food.Set(9)
	for _, diners := range []int{-1, MaxActiveParty + 1, 1 << 16} {
		assert.Panics(t, func() { inventory.ConsumeMealTimeFood(6, diners) })
		assert.Equal(t, uint16(9), inventory.Provisions.Food.Get())
	}
	starving, _ := inventory.ConsumeMealTimeFood(6, MaxActiveParty)
	assert.False(t, starving)
	assert.Equal(t, uint16(3), inventory.Provisions.Food.Get())
}
