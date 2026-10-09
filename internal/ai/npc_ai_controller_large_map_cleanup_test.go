package ai

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLargeMapDistanceCleanupPreservesFriendlyVehicles(t *testing.T) {
	t.Parallel()

	playerPosition := references.Position{X: 1, Y: 1}
	farPosition := references.Position{X: 100, Y: 100}
	vehicle := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		farPosition,
		references.MainFloor,
	)
	enemyReference := references.EnemyReference{ //nolint:exhaustruct // Cleanup needs only nonempty hostile identity.
		KeyFrameTile: &references.Tile{Index: indexes.Rat_KeyIndex}, //nolint:exhaustruct
	}
	farEnemy := map_units.NewEnemyNPC(enemyReference, 1)
	farEnemy.SetPos(farPosition)
	farEnemy.SetVisible(true)
	nearEnemy := map_units.NewEnemyNPC(enemyReference, 2)
	nearEnemy.SetPos(references.Position{X: 2, Y: 2})
	nearEnemy.SetVisible(true)

	controller := &NPCAIControllerLargeMap{ //nolint:exhaustruct // Focused cleanup fixture.
		mapState: &map_state.MapState{ //nolint:exhaustruct
			PlayerLocation: references.PlayerLocation{Position: playerPosition},
		},
		mapUnits: map_units.MapUnits{vehicle, &farEnemy, &nearEnemy},
	}

	controller.cleanupDistantEnemies()

	require.Len(t, controller.mapUnits, 2)
	assert.Same(t, vehicle, controller.mapUnits[0])
	assert.Same(t, &nearEnemy, controller.mapUnits[1])
}
