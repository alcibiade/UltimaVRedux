package party_state

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuantityActivityRecordsOnlyActualCommittedMovement(t *testing.T) {
	for _, test := range []struct {
		name     string
		quantity interface {
			ItemQuantity
			Activity() QuantityActivity
		}
		maximum uint16
	}{
		{"large", &ItemQuantityLarge{}, ItemQuantityLargeMax},
		{"small", &ItemQuantitySmall{}, ItemQuantitySmallMax},
	} {
		t.Run(test.name, func(t *testing.T) {
			quantity := test.quantity
			quantity.Set(test.maximum - 1)
			before := quantity.Activity()
			quantity.IncrementBy(20)  // Only one fits.
			quantity.IncrementByOne() // Saturated; not a gain.
			require.False(t, quantity.DecrementBy(test.maximum+1))
			quantity.IncrementBy(0)
			require.True(t, quantity.DecrementBy(0))
			require.True(t, quantity.DecrementByOne())
			quantity.IncrementByOne() // Opposing changes must both survive.
			require.True(t, quantity.DecrementBy(2))
			quantity.SetHasOne()
			quantity.Set(1) // Setting the same value is also a no-op.
			after := quantity.Activity()
			require.Equal(t, uint64(2), after.Gained-before.Gained)
			require.Equal(t, uint64(test.maximum), after.Lost-before.Lost)
			require.Equal(t, uint16(1), quantity.Get())
			quantity.Set(0)
			before = quantity.Activity()
			require.False(t, quantity.DecrementByOne())
			require.Equal(t, before, quantity.Activity())
			quantity.Set(65535)
			require.Equal(t, uint64(test.maximum), quantity.Activity().Gained-before.Gained)
			require.Equal(t, test.maximum, quantity.Get())
		})
	}
}
