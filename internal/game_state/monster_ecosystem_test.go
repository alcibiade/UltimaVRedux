package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ecosystemTestController struct {
	units map_units.MapUnits
}

func (c *ecosystemTestController) PopulateMapFirstLoad()           {}
func (c *ecosystemTestController) AdvanceNextTurnCalcAndMoveNPCs() {}
func (c *ecosystemTestController) FreshenExistingNPCsOnMap()       {}
func (c *ecosystemTestController) RemoveAllEnemies()               { c.units.RemoveAllEnemies() }
func (c *ecosystemTestController) HardResetNPCs(_ bool)            {}
func (c *ecosystemTestController) GetNpcs() *map_units.MapUnits    { return &c.units }
func (c *ecosystemTestController) ClearEnemies() int               { return c.units.RemoveAllEnemies() }

func TestMonsterGenerationSettersValidateAndAreIdempotent(t *testing.T) {
	state := &GameState{TheOdds: references.NewDefaultTheOdds()} //nolint:exhaustruct

	state.SetMonsterGenerationEnabled(true)
	state.SetMonsterGenerationEnabled(true)
	assert.True(t, state.DebugOptions.MonsterGen)

	require.NoError(t, state.SetMonsterGenerationBaseOdds(64))
	require.NoError(t, state.SetMonsterGenerationBaseOdds(64))
	assert.Equal(t, 64, state.TheOdds.GetMonsterGenerationRollMaximum())

	assert.Error(t, state.SetMonsterGenerationBaseOdds(0))
	assert.Error(t, state.SetMonsterGenerationBaseOdds(1001))
	assert.Equal(t, 64, state.TheOdds.GetMonsterGenerationRollMaximum(), "invalid odds must not mutate state")
}

func TestCurrentMonsterEcosystemSnapshotIsDetachedAndCategorized(t *testing.T) {
	enemy := map_units.NewEnemyNPC(references.EnemyReference{
		MonsterType:     references.MonsterDragon,
		Name:            "Dragon",
		ActivelyAttacks: true,
		CanFlyOverWater: true,
	}, 7) //nolint:exhaustruct
	enemy.SetPos(references.Position{X: 10, Y: 11})
	enemy.SetFloor(2)
	enemy.SetVisible(true)
	friendly := map_units.NewNPCFriendly(references.NPCReference{NpcType: references.NoStatedNpc}, 8)
	vehicle := map_units.NewNPCFriendlyVehiceNewRef(references.SkiffVehicle, references.Position{X: 12, Y: 13}, 2)
	controller := &ecosystemTestController{units: map_units.MapUnits{&enemy, friendly, vehicle}}
	state := &GameState{
		CurrentNPCAIController: controller,
		TheOdds:                references.NewDefaultTheOdds(),
	} //nolint:exhaustruct
	state.MapState.PlayerLocation = references.PlayerLocation{Location: references.BritanniaUnderworld, Floor: 2}
	state.SetMonsterGenerationEnabled(true)

	snapshot := state.CurrentMonsterEcosystemSnapshot()

	assert.Equal(t, map_units.MaximumNpcsPerMap, snapshot.Capacity)
	assert.Equal(t, 3, snapshot.CurrentUnitCount)
	assert.Equal(t, 1, snapshot.EnemyCount)
	assert.Equal(t, 1, snapshot.FriendlyCount)
	assert.Equal(t, 1, snapshot.VehicleCount)
	require.Len(t, snapshot.Enemies, 1)
	assert.Equal(t, 7, snapshot.Enemies[0].RuntimeToken)
	assert.Equal(t, references.MonsterDragon, snapshot.Enemies[0].MonsterType)
	assert.True(t, snapshot.Enemies[0].CanFlyOverWater)

	snapshot.Enemies[0].Name = "changed"
	snapshot.Enemies[0].Position.X = 99
	assert.Equal(t, "Dragon", enemy.EnemyReference.Name)
	assert.Equal(t, references.Coordinate(10), enemy.Pos().X)
}

func TestClearCurrentMapEnemiesPreservesNonEnemiesAndReportsUnsupported(t *testing.T) {
	enemy := map_units.NewEnemyNPC(references.EnemyReference{MonsterType: references.MonsterRat}, 1) //nolint:exhaustruct
	friendly := map_units.NewNPCFriendly(references.NPCReference{NpcType: references.NoStatedNpc}, 2)
	vehicle := map_units.NewNPCFriendlyVehiceNewRef(references.SkiffVehicle, references.Position{X: 2, Y: 2}, 0)
	controller := &ecosystemTestController{units: map_units.MapUnits{&enemy, friendly, vehicle}}
	state := &GameState{CurrentNPCAIController: controller} //nolint:exhaustruct

	removed, supported := state.ClearCurrentMapEnemies()

	assert.True(t, supported)
	assert.Equal(t, 1, removed)
	require.Len(t, controller.units, 2)
	assert.Same(t, friendly, controller.units[0])
	assert.Same(t, vehicle, controller.units[1])

	state.CurrentNPCAIController = &SimpleTestController{mapUnits: make(map_units.MapUnits, 0)}
	removed, supported = state.ClearCurrentMapEnemies()
	assert.False(t, supported)
	assert.Zero(t, removed)
}

func TestClearCurrentMapEnemiesRejectsCombatWithoutMutation(t *testing.T) {
	enemy := map_units.NewEnemyNPC(references.EnemyReference{MonsterType: references.MonsterRat}, 1) //nolint:exhaustruct
	controller := &ecosystemTestController{units: map_units.MapUnits{&enemy}}
	state := &GameState{CurrentNPCAIController: controller} //nolint:exhaustruct
	state.MapState.PlayerLocation.Location = references.CombatLocation

	removed, supported := state.ClearCurrentMapEnemies()

	assert.False(t, supported)
	assert.Zero(t, removed)
	require.Len(t, controller.units, 1)
	assert.Same(t, &enemy, controller.units[0])
}

func TestForcedEnemyInsertionReportsCapacityFailure(t *testing.T) {
	tiles := references.NewTileReferences()
	controller := &ecosystemTestController{units: make(map_units.MapUnits, map_units.MaximumNpcsPerMap)}
	for i := range controller.units {
		controller.units[i] = map_units.NewNPCFriendly(references.NPCReference{NpcType: references.NoStatedNpc}, i)
	}
	state := &GameState{
		CurrentNPCAIController: controller,
		GameReferences: &references.GameReferences{
			TileReferences: tiles,
		},
	} //nolint:exhaustruct

	assert.False(t, state.spawnWhirlpoolAt(references.Position{X: 1, Y: 1}))
	assert.False(t, state.spawnPirateAt(references.Position{X: 2, Y: 2}, references.Position{X: 3, Y: 3}))
	assert.Len(t, controller.units, map_units.MaximumNpcsPerMap)
}
