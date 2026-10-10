package map_units

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveAllEnemiesPreservesFriendliesAndVehicles(t *testing.T) {
	units := make(MapUnits, 0, MaximumNpcsPerMap)
	friendly := NewNPCFriendly(references.NPCReference{NpcType: references.NoStatedNpc}, 1)
	vehicle := NewNPCFriendlyVehiceNewRef(references.SkiffVehicle, references.Position{X: 2, Y: 2}, 0)
	enemy := NewEnemyNPC(references.EnemyReference{MonsterType: references.MonsterRat}, 2) //nolint:exhaustruct
	units = append(units, friendly, vehicle, &enemy)

	removed := units.RemoveAllEnemies()

	assert.Equal(t, 1, removed)
	require.Len(t, units, 2)
	assert.Same(t, friendly, units[0])
	assert.Same(t, vehicle, units[1])
	assert.False(t, enemy.IsVisible())
}

func TestAddEnemyHonorsCurrentMapUnitCapacity(t *testing.T) {
	units := make(MapUnits, MaximumNpcsPerMap)
	for i := range units {
		units[i] = NewNPCFriendly(references.NPCReference{NpcType: references.NoStatedNpc}, i)
	}
	enemy := NewEnemyNPC(references.EnemyReference{MonsterType: references.MonsterRat}, references.DynamicSpawnIndex) //nolint:exhaustruct

	assert.False(t, units.AddEnemy(enemy))
	assert.Len(t, units, MaximumNpcsPerMap)
}

func TestAddEnemyReusesVacantRuntimeTokenAfterEnemyClear(t *testing.T) {
	firstFriendly := NewNPCFriendly(references.NPCReference{NpcType: references.NoStatedNpc}, 0)
	removedEnemy := NewEnemyNPC(references.EnemyReference{MonsterType: references.MonsterRat}, 1) //nolint:exhaustruct
	secondFriendly := NewNPCFriendly(references.NPCReference{NpcType: references.NoStatedNpc}, 2)
	units := MapUnits{firstFriendly, &removedEnemy, secondFriendly}

	require.Equal(t, 1, units.RemoveAllEnemies())
	replacement := NewEnemyNPC(references.EnemyReference{MonsterType: references.MonsterOrc}, references.DynamicSpawnIndex) //nolint:exhaustruct
	require.True(t, units.AddEnemy(replacement))
	require.Len(t, units, 3)
	assert.Equal(t, 1, units[2].MapUnitDetails().NPCNum)
	assert.NotEqual(t, units[0].MapUnitDetails().NPCNum, units[2].MapUnitDetails().NPCNum)
	assert.NotEqual(t, units[1].MapUnitDetails().NPCNum, units[2].MapUnitDetails().NPCNum)
}
