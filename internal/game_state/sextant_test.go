package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func TestSextantEligibilityExplicitContexts(t *testing.T) {
	t.Parallel()
	contexts := []struct {
		name     string
		location references.Location
		floor    references.FloorNumber
	}{
		{"Britannia", references.BritanniaUnderworld, 0},
		{"Underworld", references.BritanniaUnderworld, -1},
		{"invalid world floor", references.BritanniaUnderworld, 1},
		{"town outdoors", references.Britain, 0},
		{"town basement", references.Britain, -1},
		{"dungeon", references.Deceit, 0},
		{"combat", references.CombatLocation, 0},
		{"camping", references.CampingLocation, 0},
		{"shrine", references.ShrineLocation, 0},
		{"codex", references.CodexLocation, 0},
		{"prison", references.BlackthornPrison, 0},
		{"finale", references.FinaleLocation, 0},
		{"empty", references.EmptyLocation, 0},
	}
	for _, context := range contexts {
		t.Run(context.name, func(t *testing.T) {
			for hour := range uint8(24) {
				for _, owned := range []bool{false, true} {
					input := SextantContext{HasSextant: owned, Location: context.location, Floor: context.floor, Hour: hour}
					want := SextantReady
					switch {
					case !owned:
						want = SextantMissing
					case context.name != "Britannia":
						want = SextantNotOverworld
					case hour >= 6 && hour <= 18:
						want = SextantDaylight
					}
					require.Equal(t, want, EvaluateSextantEligibility(input), "hour %d, owned %t", hour, owned)
				}
			}
		})
	}
}

func TestSextantCurrentAdapterBoundariesAreReadOnly(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(71, 93).WithRandomSeed(42).WithSystemCallbacks().Build()
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	for _, boundary := range []struct {
		hour, minute uint8
		ready        bool
	}{{5, 59, true}, {6, 0, false}, {18, 59, false}, {19, 0, true}} {
		state.DateTime.Hour, state.DateTime.Minute = boundary.hour, boundary.minute
		before, calls, observations := state.DateTime, state.GetRNG().Calls(), state.dungeonJournal
		callbacks.Reset()
		fix, ready := state.CurrentSextantFix()
		require.Equal(t, boundary.ready, ready)
		require.Equal(t, ready, state.CurrentSextantEligibility() == SextantReady)
		if ready {
			require.Equal(t, state.MapState.PlayerLocation, fix)
		} else {
			require.Equal(t, references.PlayerLocation{}, fix, "no last-known fix")
		}
		item := requireUsableItem(t, state.GetUsableItemsForContext(UsageField), references.Sextant)
		require.Equal(t, ready, item.AvailableNow)
		require.Equal(t, before, state.DateTime)
		require.Equal(t, calls, state.GetRNG().Calls())
		require.Equal(t, observations, state.dungeonJournal)
		require.Empty(t, callbacks.Messages)
		require.Empty(t, callbacks.TimeAdvanced())
	}
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
	fix, ready := state.CurrentSextantFix()
	require.False(t, ready)
	require.Equal(t, references.PlayerLocation{}, fix)
	require.False(t, state.useSextant())
	require.Equal(t, []string{"You have none!"}, callbacks.Messages)
}
