package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestAmbientTintDarknessOnlyOverridesUnderworldExploration(t *testing.T) {
	for _, context := range []struct {
		name     string
		location references.Location
		floor    references.FloorNumber
		night    bool
	}{
		{"underworld", references.BritanniaUnderworld, -1, true},
		{"overworld", references.BritanniaUnderworld, 0, false},
		{"town", references.LordBritishsCastle, 0, false},
		{"basement", references.LordBritishsCastle, -1, false},
		{"dungeon", references.Deceit, -1, false},
		{"combat", references.CombatLocation, -1, false},
	} {
		t.Run(context.name, func(t *testing.T) {
			state := &GameState{}
			state.MapState.PlayerLocation = references.PlayerLocation{Location: context.location, Floor: context.floor}
			for _, clock := range []struct{ hour, minute byte }{{0, 0}, {5, 0}, {5, 15}, {5, 30}, {12, 0}, {19, 30}, {19, 45}, {20, 0}} {
				state.DateTime.Hour, state.DateTime.Minute = clock.hour, clock.minute
				want := state.DateTime.GetAmbientTintDarkness()
				if context.night {
					want = 1
				}
				assert.Equal(t, want, state.AmbientTintDarkness())
			}
		})
	}
}
