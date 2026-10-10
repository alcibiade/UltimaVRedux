package ai

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestNewNPCAIControllerLargeMapRequiresRandomProvider(t *testing.T) {
	t.Parallel()

	assert.PanicsWithValue(t, "large-map NPC AI requires a random provider", func() {
		NewNPCAIControllerLargeMap(NewNPCAIControllerLargeMapInput{})
	})
}

// TestPirateShipCanMoveToTileRejectsShoals prevents pirate frigates from
// entering coast water that player frigates cannot traverse.
func TestPirateShipCanMoveToTileRejectsShoals(t *testing.T) {
	assert.True(t, pirateShipCanMoveToTile(&references.Tile{Index: indexes.Water1}))      //nolint:exhaustruct
	assert.True(t, pirateShipCanMoveToTile(&references.Tile{Index: indexes.Water2}))      //nolint:exhaustruct
	assert.False(t, pirateShipCanMoveToTile(&references.Tile{Index: indexes.WaterCoast})) //nolint:exhaustruct
}

func TestCalculatePirateSeekDirectionsIncludesBothPlayerwardAxes(t *testing.T) {
	controller := &NPCAIControllerLargeMap{ //nolint:exhaustruct // Focused deterministic direction fixture.
		rng: rand.New(rand.NewSource(179)), //nolint:gosec // Deterministic gameplay test.
	}

	directions := controller.calculatePirateSeekDirections(
		references.Position{X: 10, Y: 10},
		references.Position{X: 13, Y: 7},
	)

	require.Len(t, directions, 2)
	assert.ElementsMatch(t, []references.Direction{references.Right, references.Up}, directions)
}

func TestPirateWindCadenceUsesOriginalRelativeSpeeds(t *testing.T) {
	controller := &NPCAIControllerLargeMap{ //nolint:exhaustruct // Focused deterministic cadence fixture.
		rng: &fixedRandomProvider{value: 0},
	}

	assert.False(t, controller.shouldPirateShipMoveAtSpeed(1), "with-wind movement skips one turn in four")
	assert.True(t, controller.shouldPirateShipMoveAtSpeed(2), "against-wind movement succeeds on this half-speed roll")
	assert.True(t, controller.shouldPirateShipMoveAtSpeed(3), "crosswind movement is unrestricted")
	assert.False(t, controller.shouldPirateShipMoveAtSpeed(4), "calm or unknown speed remains stalled")
}

type fixedRandomProvider struct {
	value int
}

func (f *fixedRandomProvider) Intn(n int) int {
	return f.value % n
}

func TestLargeMapEnemyRandomnessIsDeterministic(t *testing.T) {
	t.Parallel()

	const seed = 179
	first := &NPCAIControllerLargeMap{rng: rand.New(rand.NewSource(seed))}  //nolint:gosec // Deterministic gameplay test.
	second := &NPCAIControllerLargeMap{rng: rand.New(rand.NewSource(seed))} //nolint:gosec // Deterministic gameplay test.

	for range 32 {
		assert.Equal(t, first.getRandomPirateDirection(), second.getRandomPirateDirection())
		assert.Equal(t, first.randomIntInRange(-9, 9), second.randomIntInRange(-9, 9))
		assert.Equal(t, first.oneInXOdds(4), second.oneInXOdds(4))
	}
}
