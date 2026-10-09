package map_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

func TestDoorVisitDiscardOnlyDepartedPhysicalSmallMap(t *testing.T) {
	for _, destination := range []references.Location{
		references.Britain, references.BlackthornPrison, references.CombatLocation,
		references.BritanniaUnderworld, references.Yew, references.Deceit,
	} {
		small, large, combat := &LayeredMap{}, &LayeredMap{}, &LayeredMap{}
		maps := LayeredMaps{
			smallMapLocation: references.Britain,
			layeredMaps: map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap{
				references.SmallMapType:  {0: small},
				references.LargeMapType:  {0: large},
				references.CombatMapType: {0: combat},
			},
		}
		maps.DiscardDepartedSmallMap(destination)
		if destination == references.Britain || destination.GetMapType() == references.CombatMapType {
			assert.Same(t, small, maps.GetLayeredMap(references.SmallMapType, 0))
			assert.Equal(t, references.Britain, maps.smallMapLocation)
		} else {
			assert.Nil(t, maps.GetLayeredMap(references.SmallMapType, 0))
			assert.Equal(t, references.BritanniaUnderworld, maps.smallMapLocation)
		}
		assert.Same(t, large, maps.GetLayeredMap(references.LargeMapType, 0))
		assert.Same(t, combat, maps.GetLayeredMap(references.CombatMapType, 0))
	}
}
