package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestLargeMapMonsterMovementUsesDestinationTerrainBelowUnitLayer(t *testing.T) {
	for _, test := range []struct {
		name      string
		tile      indexes.SpriteIndex
		roll      int
		rollBound int
		want      bool
	}{
		{name: "swamp moves on zero", tile: indexes.Swamp, roll: 0, rollBound: 2, want: true},
		{name: "swamp waits on one", tile: indexes.Swamp, roll: 1, rollBound: 2, want: false},
		{name: "forest moves on two", tile: indexes.Forest3, roll: 2, rollBound: 3, want: true},
		{name: "forest waits on one", tile: indexes.Forest3, roll: 1, rollBound: 3, want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			rng := &scriptedGenerationRNG{t: t, values: []int{test.roll}}
			controller, layeredMap := newLargeMapGenerationTestController(
				t, references.FloorNumber(references.OVERWORLD), 12, rng,
			)
			destination := references.Position{X: 101, Y: 100}
			layeredMap.SetTileByLayer(map_state.MapLayer, &destination, test.tile)

			enemyReference := (*controller.enemyReferences)[0]
			enemy := map_units.NewEnemyNPC(enemyReference, 0)
			enemy.SetPos(destination)
			enemy.SetFloor(references.FloorNumber(references.OVERWORLD))
			requireAddedEnemy(t, &controller.mapUnits, enemy)
			controller.FreshenExistingNPCsOnMap()

			assert.Equal(t, test.want, controller.shouldEnemyMoveTo(&enemy, destination))
			assert.Equal(t, []int{test.rollBound}, rng.calls)
		})
	}
}

func TestLargeMapFlyingMonsterBypassesDestinationTerrainSlowdown(t *testing.T) {
	rng := &scriptedGenerationRNG{t: t}
	controller, layeredMap := newLargeMapGenerationTestController(
		t, references.FloorNumber(references.OVERWORLD), 12, rng,
	)
	destination := references.Position{X: 101, Y: 100}
	layeredMap.SetTileByLayer(map_state.MapLayer, &destination, indexes.Forest3)

	enemyReference := findEnemyReferenceForMovementTest(t, controller.enemyReferences, references.MonsterDragon)
	enemy := map_units.NewEnemyNPC(*enemyReference, 0)
	enemy.SetPos(destination)
	enemy.SetFloor(references.FloorNumber(references.OVERWORLD))
	requireAddedEnemy(t, &controller.mapUnits, enemy)
	controller.FreshenExistingNPCsOnMap()

	assert.True(t, controller.shouldEnemyMoveTo(&enemy, destination))
	assert.Empty(t, rng.calls)
}

func TestRandomMovementDoesNotRetryAfterTerrainSlowdown(t *testing.T) {
	rng := &scriptedGenerationRNG{t: t, values: []int{0, 1}}
	controller, layeredMap := newLargeMapGenerationTestController(
		t, references.FloorNumber(references.OVERWORLD), 12, rng,
	)
	// Keep the selected left-hand destination clear of the party at (100,100),
	// so this tests slowdown rather than occupancy rejection.
	start := references.Position{X: 103, Y: 100}
	for _, position := range start.GetFourDirectionsWrapped(references.XLargeMapTiles, references.YLargeMapTiles) {
		layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.Swamp)
	}

	enemyReference := findEnemyReferenceForMovementTest(t, controller.enemyReferences, references.MonsterOrc)
	enemy := map_units.NewEnemyNPC(*enemyReference, 0)
	enemy.SetPos(start)
	enemy.SetFloor(references.FloorNumber(references.OVERWORLD))
	requireAddedEnemy(t, &controller.mapUnits, enemy)
	controller.FreshenExistingNPCsOnMap()

	controller.moveRandomDirection(&enemy)

	assert.Equal(t, start, enemy.Pos())
	assert.Equal(t, []int{4, 2}, rng.calls)
}

func TestDirectedAStarPursuitWaitsOneInFourOpportunities(t *testing.T) {
	rng := &scriptedGenerationRNG{t: t, values: []int{1, 0}}
	controller, layeredMap := newLargeMapGenerationTestController(
		t, references.FloorNumber(references.OVERWORLD), 12, rng,
	)
	start := references.Position{X: 92, Y: 100}
	for x := start.X; x <= controller.mapState.PlayerLocation.Position.X; x++ {
		position := references.Position{X: x, Y: start.Y}
		layeredMap.SetTileByLayer(map_state.MapLayer, &position, indexes.Grass)
	}

	enemyReference := findEnemyReferenceForMovementTest(t, controller.enemyReferences, references.MonsterOrc)
	enemy := map_units.NewEnemyNPC(*enemyReference, 0)
	enemy.SetPos(start)
	enemy.SetFloor(references.FloorNumber(references.OVERWORLD))
	requireAddedEnemy(t, &controller.mapUnits, enemy)
	controller.FreshenExistingNPCsOnMap()

	controller.calculateNextNPCPosition(&enemy)

	assert.Equal(t, start, enemy.Pos())
	assert.Equal(t, []int{randomMovementOdds, directedPursuitWaitOdds}, rng.calls)
}

func requireAddedEnemy(t *testing.T, units *map_units.MapUnits, enemy map_units.NPCEnemy) {
	t.Helper()
	if !units.AddEnemy(enemy) {
		t.Fatal("failed to add test enemy")
	}
}

func findEnemyReferenceForMovementTest(
	t *testing.T,
	enemies *references.EnemyReferences,
	monsterType references.MonsterType,
) *references.EnemyReference {
	t.Helper()
	for index := range *enemies {
		if (*enemies)[index].MonsterType == monsterType {
			return &(*enemies)[index]
		}
	}
	t.Fatalf("missing enemy reference for %v", monsterType)

	return nil
}
