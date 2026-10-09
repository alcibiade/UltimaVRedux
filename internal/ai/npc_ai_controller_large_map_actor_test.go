package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestLargeMapActorAttackChecksInterleaveDescendingMovement(t *testing.T) {
	rng := &scriptedGenerationRNG{t: t, values: []int{1, 1}}
	controller, layer := newLargeMapGenerationTestController(t, 0, 12, rng)
	controller.debugOptions.MonsterGen = false
	for y := references.Coordinate(80); y <= 120; y++ {
		for x := references.Coordinate(80); x <= 120; x++ {
			pos := references.Position{X: x, Y: y}
			layer.SetTileByLayer(map_state.MapLayer, &pos, indexes.Grass)
		}
	}
	var actors []*map_units.NPCEnemy
	for _, slot := range []int{10, 31, 20} {
		enemy := map_units.NewEnemyNPC(*controller.lookupEnemyByMonsterType(references.MonsterDragon), slot)
		enemy.SetPos(references.Position{X: 105, Y: 100 + references.Coordinate(len(actors))})
		enemy.SetFloor(0)
		require.True(t, controller.mapUnits.AddEnemy(enemy))
		actors = append(actors, controller.mapUnits[len(controller.mapUnits)-1].(*map_units.NPCEnemy))
		actors[len(actors)-1].MapUnitDetails().NPCNum = slot
	}
	lowerBefore, higherBefore := actors[0].Pos(), actors[1].Pos()
	var order []int
	controller.AdvanceNextTurnWithActorCheck(func(enemy *map_units.NPCEnemy) OutdoorActorAction {
		slot := enemy.MapUnitDetails().NPCNum
		order = append(order, slot)
		if slot == 31 {
			assert.Equal(t, higherBefore, enemy.Pos(), "attack evaluated before its own movement")
			assert.Empty(t, rng.calls)
			return OutdoorActorNoAttack
		}
		assert.NotEqual(t, higherBefore, actors[1].Pos(), "earlier nonattacker moves before the next check")
		assert.Equal(t, []int{4, 4}, rng.calls)
		if slot == 20 {
			return OutdoorActorAttacked
		}
		return OutdoorActorNoAttack
	})
	assert.Equal(t, []int{31, 20, 10}, order, "later attacks are evaluated after an attack")
	assert.Equal(t, lowerBefore, actors[0].Pos(), "earlier attack suppresses later movement")
	assert.Equal(t, []int{4, 4}, rng.calls)
}

func TestLargeMapDeferredActorResumesWithoutGenerationOrMovement(t *testing.T) {
	rng := &scriptedGenerationRNG{t: t}
	controller, _ := newLargeMapGenerationTestController(t, 0, 12, rng)
	controller.debugOptions.MonsterGen = false
	for _, slot := range []int{1, 20, 31} {
		enemy := map_units.NewEnemyNPC(*controller.lookupEnemyByMonsterType(references.MonsterDragon), slot)
		enemy.SetPos(references.Position{X: 102, Y: 102})
		enemy.SetFloor(0)
		require.True(t, controller.mapUnits.AddEnemy(enemy))
		controller.mapUnits[len(controller.mapUnits)-1].MapUnitDetails().NPCNum = slot
	}
	var slots []int
	controller.AdvanceNextTurnWithActorCheck(func(enemy *map_units.NPCEnemy) OutdoorActorAction {
		slot := enemy.MapUnitDetails().NPCNum
		slots = append(slots, slot)
		if slot == 20 {
			return OutdoorActorDeferred
		}
		return OutdoorActorAttacked
	})
	assert.Equal(t, []int{31, 20}, slots)
	controller.debugOptions.MonsterGen = true
	controller.ContinueTurnAfterAttack(20, func(enemy *map_units.NPCEnemy) OutdoorActorAction {
		slots = append(slots, enemy.MapUnitDetails().NPCNum)
		return OutdoorActorNoAttack
	})
	assert.Equal(t, []int{31, 20, 1}, slots)
	assert.Empty(t, rng.calls, "continuation repeats neither generation nor movement draws")
	for _, unit := range controller.mapUnits {
		assert.Equal(t, references.Position{X: 102, Y: 102}, unit.Pos())
	}
}
