package game_state

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestClassifyOverworldEnemyContact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		enemy        *map_units.NPCEnemy
		expectedType OverworldContactType
	}{
		{
			name:         "nil enemy",
			enemy:        nil,
			expectedType: OverworldContactNone,
		},
		{
			name:         "empty enemy",
			enemy:        newContactTestEnemy(references.MonsterOrc, 0, true, false, false),
			expectedType: OverworldContactNone,
		},
		{
			name:         "land monster",
			enemy:        newContactTestEnemy(references.MonsterOrc, indexes.OrcKeyIndex, true, false, false),
			expectedType: OverworldContactLandMonster,
		},
		{
			name:         "water monster",
			enemy:        newContactTestEnemy(references.MonsterShark, indexes.Shark1, true, true, false),
			expectedType: OverworldContactWaterMonster,
		},
		{
			name:         "pirate ship identified by sprite",
			enemy:        newContactTestEnemy(references.MonsterStocks, indexes.PirateShipUp, true, true, false),
			expectedType: OverworldContactPirateShip,
		},
		{
			name:         "whirlpool",
			enemy:        newContactTestEnemy(references.MonsterWhirlpool, indexes.WhirlpoolKeyIndex, true, true, false),
			expectedType: OverworldContactWhirlpool,
		},
		{
			name:         "sandtrap",
			enemy:        newContactTestEnemy(references.MonsterSandtrap, indexes.SandtrapKeyIndex, true, false, true),
			expectedType: OverworldContactSandtrap,
		},
		{
			name:         "force field",
			enemy:        newContactTestEnemy(references.MonsterForceField, indexes.MagicField, true, false, false),
			expectedType: OverworldContactNone,
		},
		{
			name:         "non-hostile enemy reference",
			enemy:        newContactTestEnemy(references.MonsterBard, indexes.BardKeyIndex, false, false, false),
			expectedType: OverworldContactNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedType, ClassifyOverworldEnemyContact(tt.enemy))
		})
	}
}

func TestSelectOverworldCombatMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		contactType  OverworldContactType
		vehicleType  references.VehicleType
		terrainIndex indexes.SpriteIndex
		expectedMap  references.BritanniaCombatMap
	}{
		{"frigate versus pirate", OverworldContactPirateShip, references.FrigateVehicle, indexes.Water1, references.ShipShipCombatMap},
		{"land versus pirate", OverworldContactPirateShip, references.NoPartyVehicle, indexes.Water1, references.LandShipCombatMap},
		{"frigate versus sea monster", OverworldContactWaterMonster, references.FrigateVehicle, indexes.Water1, references.ShipSeaCombatMap},
		{"land versus sea monster", OverworldContactWaterMonster, references.NoPartyVehicle, indexes.Water1, references.LandSeaCombatMap},
		{"frigate versus shore monster", OverworldContactLandMonster, references.FrigateVehicle, indexes.Grass, references.ShipShoreCombatMap},
		{"frigate versus monster on water", OverworldContactLandMonster, references.FrigateVehicle, indexes.Water1, references.ShipSeaCombatMap},
		{"land versus monster on water", OverworldContactLandMonster, references.NoPartyVehicle, indexes.Water1, references.LandSeaCombatMap},
		{"swamp", OverworldContactLandMonster, references.NoPartyVehicle, indexes.Swamp, references.SwampCombatMap},
		{"grass", OverworldContactLandMonster, references.NoPartyVehicle, indexes.Grass, references.GladeCombatMap},
		{"brush", OverworldContactLandMonster, references.NoPartyVehicle, indexes.Forest1, references.TreedCombatMap},
		{"thick brush", OverworldContactLandMonster, references.NoPartyVehicle, indexes.Forest2, references.TreedCombatMap},
		{"desert", OverworldContactLandMonster, references.NoPartyVehicle, indexes.Desert1, references.DesertCombatMap},
		{"waste", OverworldContactLandMonster, references.NoPartyVehicle, indexes.LeftDesert2, references.DesertCombatMap},
		{"forest", OverworldContactLandMonster, references.NoPartyVehicle, indexes.Forest3, references.CleanTreeCombatMap},
		{"tropical", OverworldContactLandMonster, references.NoPartyVehicle, indexes.Forest4, references.CleanTreeCombatMap},
		{"hills", OverworldContactLandMonster, references.NoPartyVehicle, indexes.Hills1, references.MountainsCombatMap},
		{"mountains", OverworldContactLandMonster, references.NoPartyVehicle, indexes.SmallMountains, references.MountainsCombatMap},
		{"bridge", OverworldContactLandMonster, references.NoPartyVehicle, indexes.Bridge, references.BigBridgeCombatMap},
		{"river bridge is bridge", OverworldContactLandMonster, references.NoPartyVehicle, indexes.TrollBridgeHoriz, references.BigBridgeCombatMap},
		{"river", OverworldContactLandMonster, references.NoPartyVehicle, indexes.WaterStream1, references.LandSeaCombatMap},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			terrain := &references.Tile{
				Index: tt.terrainIndex,
			}
			combatMap, err := SelectOverworldCombatMap(tt.contactType, tt.vehicleType, terrain)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedMap, combatMap)
		})
	}
}

func TestSelectOverworldCombatMapRejectsSpecialContacts(t *testing.T) {
	t.Parallel()

	for _, contactType := range []OverworldContactType{
		OverworldContactNone,
		OverworldContactWhirlpool,
		OverworldContactSandtrap,
	} {
		combatMap, err := SelectOverworldCombatMap(contactType, references.NoPartyVehicle, nil)

		assert.Zero(t, combatMap)
		assert.True(t, errors.Is(err, ErrUnsupportedOverworldCombatContact))
	}
}

func TestStartOverworldCombatEncounterUsesOriginalFixedGroups(t *testing.T) {
	tests := []struct {
		monsterType   references.MonsterType
		expectedCount int
	}{
		{references.MonsterSeaSerpent, references.SoloMonsterCount},
		{references.MonsterSkeleton, references.GuardGroupCount},
		{references.MonsterBat, references.SwarmMonsterCount},
	}

	for _, tt := range tests {
		t.Run(tt.monsterType.Name(), func(t *testing.T) {
			gs, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
			enemy := addOverworldTestEnemy(t, gs, tt.monsterType, references.Position{X: 51, Y: 50})
			require.NoError(t, gs.StartOverworldCombatEncounter(enemy))

			assert.Equal(t, tt.expectedCount, gs.CombatScene.MonsterManager.Count())
			assert.Equal(t, tt.monsterType, gs.CombatScene.MonsterManager.GetMonster(0).MonsterType())
		})
	}
}

func TestStartOverworldCombatEncounterRollsDragonGroupFromMonsterData(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	gs.DateTime.Year = 2
	gs.SetRandomSeed(179)
	enemy := addOverworldTestEnemy(t, gs, references.MonsterDragon, references.Position{X: 51, Y: 50})
	enemyRef, err := gs.enemyReferenceForCombat(references.MonsterDragon)
	require.NoError(t, err)
	require.Equal(t, 2, enemyRef.MaxPerMap, "original dragon group maximum")

	require.NoError(t, gs.StartOverworldCombatEncounter(enemy))

	assert.GreaterOrEqual(t, gs.CombatScene.MonsterManager.Count(), 1)
	assert.LessOrEqual(t, gs.CombatScene.MonsterManager.Count(), enemyRef.MaxPerMap)
}

func TestStartOverworldCombatEncounterUsesPirateBards(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	gs.DateTime.Year = 2
	gs.SetRandomSeed(179)

	pirate := addPirateTestEnemy(t, gs, references.Position{X: 51, Y: 50}, indexes.PirateShipUp)
	require.NoError(t, gs.StartOverworldCombatEncounter(pirate))

	assert.Equal(t, references.MonsterBard, gs.CombatScene.MonsterManager.GetMonster(0).MonsterType())
}

func TestOverworldEncounterMonsterCountAppliesOpeningCampaignMonthReduction(t *testing.T) {
	actualGS, _ := NewIntegrationTestBuilder(t).Build()
	expectedGS, _ := NewIntegrationTestBuilder(t).Build()
	actualGS.DateTime.Year, actualGS.DateTime.Month, actualGS.DateTime.Day = 139, 4, 5
	actualGS.openingCampaignEncounterReduction = true
	expectedGS.DateTime = actualGS.DateTime
	actualGS.SetRandomSeed(179)
	expectedGS.SetRandomSeed(179)

	enemyRef, err := actualGS.enemyReferenceForCombat(references.MonsterOrc)
	assert.NoError(t, err)
	maximum := min(enemyRef.MaxPerMap, references.MaxMonstersPerMap)
	expectedCount := expectedGS.RollDice(maximum)
	expectedCount = expectedGS.RollDice(expectedCount)

	actualCount, err := actualGS.overworldEncounterMonsterCount(references.MonsterOrc, enemyRef)

	assert.NoError(t, err)
	assert.Equal(t, expectedCount, actualCount)
}

func TestOpeningCampaignEncounterReductionIsIndependentOfCalendar(t *testing.T) {
	state := &GameState{openingCampaignEncounterReduction: true}
	state.DateTime.Year, state.DateTime.Month = 140, 4
	assert.True(t, state.hasOpeningCampaignEncounterReduction())

	state.openingCampaignEncounterReduction = false
	state.DateTime.Year, state.DateTime.Month = 139, 4
	assert.False(t, state.hasOpeningCampaignEncounterReduction())
}

func TestCampingEncounterCountAppliesOpeningCampaignMonthReduction(t *testing.T) {
	actual := &GameState{}
	expected := &GameState{}
	actual.DateTime.Year, actual.DateTime.Month = 139, 4
	actual.openingCampaignEncounterReduction = true
	expected.DateTime = actual.DateTime
	actual.SetRandomSeed(179)
	expected.SetRandomSeed(179)
	const maximum = 6
	want := expected.RollDice(maximum)
	want = expected.RollDice(want)

	got := actual.resolveCampingAmbushCount(maximum)

	assert.Equal(t, want, got)
	assert.Equal(t, expected.GetRNG().Calls(), actual.GetRNG().Calls())
}

func TestStartOverworldCombatEncounterIsDeterministic(t *testing.T) {
	firstGS, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	secondGS, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	firstGS.DateTime.Year = 2
	secondGS.DateTime.Year = 2
	firstGS.SetRandomSeed(179)
	secondGS.SetRandomSeed(179)

	firstEnemy := addOverworldTestEnemy(t, firstGS, references.MonsterOrc, references.Position{X: 51, Y: 50})
	secondEnemy := addOverworldTestEnemy(t, secondGS, references.MonsterOrc, references.Position{X: 51, Y: 50})
	require.NoError(t, firstGS.StartOverworldCombatEncounter(firstEnemy))
	require.NoError(t, secondGS.StartOverworldCombatEncounter(secondEnemy))

	assert.Equal(t, combatMonsterSignatures(firstGS), combatMonsterSignatures(secondGS))
}

func TestStartOverworldCombatEncounterRejectsInvalidFriendAtomically(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).Build()
	controlGS, _ := NewIntegrationTestBuilder(t).Build()
	enemy := addOverworldTestEnemy(t, gs, references.MonsterOrc, references.Position{X: 51, Y: 50})
	refs := append(references.EnemyReferences(nil), (*gs.GameReferences.EnemyReferences)...)
	refs[references.MonsterOrc].Friend = &references.EnemyReference{MonsterType: references.MonsterType(999)}
	gameRefs := *gs.GameReferences
	gameRefs.EnemyReferences = &refs
	gs.GameReferences = &gameRefs
	before := gs.MapState.PlayerLocation
	gs.SetRandomSeed(179)
	controlGS.SetRandomSeed(179)

	err := gs.StartOverworldCombatEncounter(enemy)

	assert.ErrorIs(t, err, ErrInvalidOverworldEncounter)
	assert.Nil(t, gs.CombatScene)
	assert.Equal(t, before, gs.MapState.PlayerLocation)
	assert.Equal(t, controlGS.RandomIntInRange(1, 1000), gs.RandomIntInRange(1, 1000),
		"failed preflight must not advance deterministic combat RNG")
}

func TestStartOverworldCombatEncounterTransitionsWithOrigin(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithPlayerAt(50, 50).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()

	sourcePosition := references.Position{X: 51, Y: 50}
	enemyRef, err := gs.enemyReferenceForCombat(references.MonsterOrc)
	require.NoError(t, err)
	enemy := map_units.NewEnemyNPC(*enemyRef, 0)
	enemy.SetPos(sourcePosition)
	enemy.SetFloor(gs.MapState.PlayerLocation.Floor)
	require.True(t, gs.GetCurrentLargeMapNPCAIController().GetNpcs().AddEnemy(enemy))

	sourceEnemy := gs.GetCurrentLargeMapNPCAIController().GetNpcs().
		GetVisibleEnemyAtPositionAndFloorOrNil(sourcePosition, gs.MapState.PlayerLocation.Floor)
	require.NotNil(t, sourceEnemy)
	preCombatLocation := gs.MapState.PlayerLocation

	require.NoError(t, gs.StartOverworldCombatEncounter(sourceEnemy))

	assert.Equal(t, references.CombatLocation, gs.MapState.PlayerLocation.Location)
	require.NotNil(t, gs.CombatScene)
	require.NotNil(t, gs.CombatScene.OverworldOrigin)
	assert.Same(t, sourceEnemy, gs.CombatScene.OverworldOrigin.SourceEnemy)
	assert.Equal(t, preCombatLocation, gs.CombatScene.PreCombatPlayerLocation)
	assert.Equal(t, OverworldContactLandMonster, gs.CombatScene.OverworldOrigin.ContactType)
	assert.Positive(t, gs.CombatScene.MonsterManager.Count())
	_, queuedMonsters := gs.CombatScene.TurnQueue.CountRemaining()
	assert.Equal(t, gs.CombatScene.MonsterManager.Count(), queuedMonsters)
}

// TestOverworldEncounterSupportsCombatAttackTargeting prevents production
// encounters from entering combat without tactical A-command state.
func TestOverworldEncounterSupportsCombatAttackTargeting(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithPlayerAt(50, 50).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()
	sourceEnemy := addOverworldTestEnemy(
		t,
		gs,
		references.MonsterOrc,
		references.Position{X: 51, Y: 50},
	)

	require.NoError(t, gs.StartOverworldCombatEncounter(sourceEnemy))
	require.True(t, gs.BeginCombatAttack(), "A command must begin tactical targeting")
	require.NotNil(t, gs.CombatScene.AttackTargeting)

	targeting := gs.CombatScene.AttackTargeting
	assert.Equal(t, gs.CombatScene.GetActiveCombatantIndex(), targeting.AttackerIndex)
	assert.Equal(t, targeting.Origin, targeting.Cursor)
	assert.Equal(t, CombatAimFriendly, gs.GetCombatAimDisposition(targeting.Cursor))
}

func TestStartOverworldCombatEncounterRejectsSpecialContactWithoutTransition(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).WithPlayerAt(50, 50).Build()

	sourcePosition := references.Position{X: 51, Y: 50}
	enemyRef, err := gs.enemyReferenceForCombat(references.MonsterWhirlpool)
	require.NoError(t, err)
	enemy := map_units.NewEnemyNPC(*enemyRef, 0)
	enemy.SetPos(sourcePosition)
	enemy.SetFloor(gs.MapState.PlayerLocation.Floor)
	require.True(t, gs.GetCurrentLargeMapNPCAIController().GetNpcs().AddEnemy(enemy))
	sourceEnemy := gs.GetCurrentLargeMapNPCAIController().GetNpcs().
		GetVisibleEnemyAtPositionAndFloorOrNil(sourcePosition, gs.MapState.PlayerLocation.Floor)

	err = gs.StartOverworldCombatEncounter(sourceEnemy)

	assert.ErrorIs(t, err, ErrUnsupportedOverworldCombatContact)
	assert.Equal(t, references.BritanniaUnderworld, gs.MapState.PlayerLocation.Location)
	assert.Nil(t, gs.CombatScene)
}

func TestFinishTurnStartsAdjacentOverworldEncounterBeforeEnemyMovement(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithPlayerAt(50, 50).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()
	gs.DebugOptions.MonsterGen = false
	sourceEnemy := addOverworldTestEnemy(t, gs, references.MonsterOrc, references.Position{X: 51, Y: 50})

	gs.FinishTurn()

	require.NotNil(t, gs.CombatScene)
	require.NotNil(t, gs.CombatScene.OverworldOrigin)
	assert.Same(t, sourceEnemy, gs.CombatScene.OverworldOrigin.SourceEnemy)
}

func TestFinishTurnDoesNotStartDiagonalOverworldEncounter(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithPlayerAt(50, 50).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()
	gs.DebugOptions.MonsterGen = false
	addOverworldTestEnemy(t, gs, references.MonsterOrc, references.Position{X: 51, Y: 51})

	gs.FinishTurn()

	assert.Nil(t, gs.CombatScene)
}

func TestFinishTurnStartsWrappedEdgeOverworldEncounter(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithPlayerAt(0, 50).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()
	gs.DebugOptions.MonsterGen = false
	sourceEnemy := addOverworldTestEnemy(t, gs, references.MonsterOrc, references.Position{X: 255, Y: 50})

	gs.FinishTurn()

	require.NotNil(t, gs.CombatScene)
	assert.Same(t, sourceEnemy, gs.CombatScene.OverworldOrigin.SourceEnemy)
}

func TestUnderworldEnemyContactStartsTacticalCombat(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithFloor(references.FloorNumber(references.UNDERWORLD)).
		WithPlayerAt(50, 50).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()
	gs.DebugOptions.MonsterGen = false
	addOverworldTestEnemy(t, gs, references.MonsterBat, references.Position{X: 51, Y: 50})

	gs.FinishTurn()

	require.NotNil(t, gs.CombatScene)
	require.NotNil(t, gs.CombatScene.OverworldOrigin)
	assert.Equal(t, references.UNDERWORLD, gs.CombatScene.OverworldOrigin.World)
}

func TestSkiffWaterContactUsesOriginalDirectPartyDamage(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()
	gs.DebugOptions.MonsterGen = false

	playerPosition, enemyPosition := findAdjacentDeepWaterPositions(t, gs)
	gs.MapState.PlayerLocation.Position = playerPosition
	skiff := map_units.NewNPCFriendlyVehiceNewRef(
		references.SkiffVehicle,
		playerPosition,
		gs.MapState.PlayerLocation.Floor,
	)
	gs.PartyVehicle = *skiff
	sourceEnemy := addOverworldTestEnemy(t, gs, references.MonsterShark, enemyPosition)
	oldHP := gs.PartyState.GetCharacter(0).GetCurrentHp()

	gs.FinishTurn()

	assert.Nil(t, gs.CombatScene)
	newHP := gs.PartyState.GetCharacter(0).GetCurrentHp()
	assert.Less(t, newHP, oldHP)
	assert.GreaterOrEqual(t, newHP, oldHP-8)
	assert.Contains(t, mockCallbacks.Messages, "Attacked!")
	assert.Same(t, sourceEnemy, gs.GetCurrentLargeMapNPCAIController().GetNpcs().
		GetVisibleEnemyAtPositionAndFloorOrNil(enemyPosition, gs.MapState.PlayerLocation.Floor))
}

func TestSandtrapContactUsesDirectDamageWithoutTacticalCombat(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithPlayerAt(50, 50).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()
	gs.DebugOptions.MonsterGen = false
	sourcePosition := references.Position{X: 51, Y: 50}
	sourceEnemy := addOverworldTestEnemy(t, gs, references.MonsterSandtrap, sourcePosition)
	oldHP := gs.PartyState.GetCharacter(0).GetCurrentHp()

	gs.FinishTurn()

	assert.Nil(t, gs.CombatScene)
	newHP := gs.PartyState.GetCharacter(0).GetCurrentHp()
	assert.Less(t, newHP, oldHP)
	assert.NotContains(t, mockCallbacks.Messages, "Attacked!")
	assert.Same(t, sourceEnemy, gs.GetCurrentLargeMapNPCAIController().GetNpcs().
		GetVisibleEnemyAtPositionAndFloorOrNil(sourcePosition, gs.MapState.PlayerLocation.Floor))
}

func TestOutdoorActorContactUsesHighestNPCSlot(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).WithPlayerAt(50, 50).Build()
	gs.DebugOptions.MonsterGen = false
	higherSlotEnemy := addOverworldTestEnemy(t, gs, references.MonsterOrc, references.Position{X: 49, Y: 50})
	lowerSlotEnemy := addOverworldTestEnemy(t, gs, references.MonsterDragon, references.Position{X: 51, Y: 50})
	higherSlotEnemy.MapUnitDetails().NPCNum = 31
	lowerSlotEnemy.MapUnitDetails().NPCNum = 2

	gs.largeMapProcessTurnEffects()
	assert.Same(t, higherSlotEnemy, gs.pendingOverworldEncounter)
}

func TestPendingEncounterWaitsForActiveTurnEffectQueue(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithPlayerAt(50, 50).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()
	gs.DebugOptions.MonsterGen = false
	addOverworldTestEnemy(t, gs, references.MonsterOrc, references.Position{X: 51, Y: 50})
	gs.EventQueueManager.Workflow("turn_hazard_test").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {}).Finally(

		func() {}).
		Start()

	gs.FinishTurn()

	assert.Nil(t, gs.CombatScene)
	assert.True(t, gs.HasPendingOverworldContact())
	for gs.EventQueueManager.HasActiveWorkflows() {
		gs.EventQueueManager.Update()
	}
	require.NoError(t, gs.ResolvePendingOverworldContact())
	require.NotNil(t, gs.CombatScene)
}

// TestPendingEncounterDoesNotStartAfterContactMovesOutOfRange prevents a
// deferred tactical transition from using stale adjacency.
func TestPendingEncounterDoesNotStartAfterContactMovesOutOfRange(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithPlayerAt(50, 50).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()
	gs.DebugOptions.MonsterGen = false
	enemy := addOverworldTestEnemy(t, gs, references.MonsterOrc, references.Position{X: 51, Y: 50})
	gs.largeMapProcessTurnEffects()
	require.Same(t, enemy, gs.pendingOverworldEncounter)
	enemy.SetPos(references.Position{X: 52, Y: 50})

	require.NoError(t, gs.ResolvePendingOverworldContact())

	assert.Nil(t, gs.CombatScene)
	assert.False(t, gs.HasPendingOverworldContact())
	assert.True(t, enemy.IsVisible())
}

func TestQueueOverworldEncounterAtRejectsOneBlankTileGap(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithPlayerAt(50, 50).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()
	gs.DebugOptions.MonsterGen = false
	addOverworldTestEnemy(t, gs, references.MonsterOrc, references.Position{X: 52, Y: 50})

	assert.False(t, gs.QueueOverworldEncounterAt(references.Position{X: 52, Y: 50}))
	assert.False(t, gs.HasPendingOverworldContact())
}

func TestEnemyThatMovesAdjacentWaitsUntilNextTurnToAttack(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()
	gs.DebugOptions.MonsterGen = false

	playerPosition, middlePosition, enemyPosition := findThreeWalkableLandPositions(t, gs)
	gs.MapState.PlayerLocation.Position = playerPosition
	sourceEnemy := addOverworldTestEnemy(t, gs, references.MonsterOrc, enemyPosition)
	const randomMoveOdds = 4
	for seed := uint64(1); ; seed++ {
		gs.SetRandomSeed(seed)
		if gs.GetRNG().Intn(randomMoveOdds) != 0 {
			gs.SetRandomSeed(seed)

			break
		}
	}

	gs.largeMapProcessTurnEffects()

	assert.Equal(t, middlePosition, sourceEnemy.Pos())
	assert.Nil(t, gs.pendingOverworldEncounter)
	assert.Nil(t, gs.CombatScene)

	gs.FinishTurn()
	require.NotNil(t, gs.CombatScene)
}

func TestCombatPassabilityIgnoresBoardedFrigate(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).Build()
	require.NoError(t, gs.TransitionToCombatMap(
		int(references.GladeCombatMap),
		references.BritanniaCombatTerritory,
		CombatEntryNorth,
	))

	activeIndex := gs.CombatScene.GetActiveCombatantIndex()
	var candidate *references.Position
	for y := references.Coordinate(0); y < references.YCombatMapTiles && candidate == nil; y++ {
		for x := references.Coordinate(0); x < references.XCombatMapTiles; x++ {
			position := references.Position{X: x, Y: y}
			tile := gs.GetCurrentLayeredMap().GetTopTile(&position)
			if tile.IsWalkingPassable() && !tile.IsBoatPassable() &&
				!gs.CombatScene.isPartyMemberAtPosition(position, activeIndex) {
				candidate = &position

				break
			}
		}
	}
	require.NotNil(t, candidate, "glade combat map should contain walkable non-water terrain")

	gs.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		gs.CombatScene.PreCombatPlayerLocation.Position,
		gs.CombatScene.PreCombatPlayerLocation.Floor,
	)

	assert.True(t, gs.isCombatPositionValid(candidate, activeIndex))
}

func TestShipCombatMapsPlacePartyOnWalkableTerrain(t *testing.T) {
	for _, combatMap := range []references.BritanniaCombatMap{
		references.ShipSeaCombatMap,
		references.ShipShoreCombatMap,
		references.ShipShipCombatMap,
	} {
		t.Run(fmt.Sprintf("map_%d", combatMap), func(t *testing.T) {
			gs, _ := NewIntegrationTestBuilder(t).Build()
			require.NoError(t, gs.TransitionToCombatMap(
				int(combatMap),
				references.BritanniaCombatTerritory,
				CombatEntryNorth,
			))

			for partyIndex, active := range gs.CombatScene.PartyActive {
				if !active {
					continue
				}
				position := gs.CombatScene.PartyPositions[partyIndex]
				tile := gs.GetCurrentLayeredMap().GetTopTile(&position)
				assert.True(t, tile.IsWalkingPassable(),
					"map %d party member %d position %v must be walkable",
					combatMap, partyIndex, position)
			}
		})
	}
}

func TestEndCombatRemovesOverworldSourceForNonVictoryOutcome(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithPlayerAt(50, 50).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()
	require.True(t, gs.SetActivePlayer(1))
	sourcePosition := references.Position{X: 51, Y: 50}
	sourceEnemy := addOverworldTestEnemy(t, gs, references.MonsterOrc, sourcePosition)
	require.NoError(t, gs.StartOverworldCombatEncounter(sourceEnemy))
	gs.CombatScene.AddObject(NewCombatChest(references.Position{X: 5, Y: 5}, 10, false))

	gs.EndCombat(CombatDefeat)

	assert.Equal(t, references.BritanniaUnderworld, gs.MapState.PlayerLocation.Location)
	assert.Nil(t, gs.CombatScene)
	assert.Equal(t, 1, gs.GetActivePlayerIndex())
	assert.Nil(t, gs.GetCurrentLargeMapNPCAIController().GetNpcs().
		GetVisibleEnemyAtPositionAndFloorOrNil(sourcePosition, gs.MapState.PlayerLocation.Floor))

	require.NoError(t, gs.TransitionToCombatMap(
		int(references.GladeCombatMap),
		references.BritanniaCombatTerritory,
		CombatEntryNorth,
	))
	assert.Zero(t, gs.CombatScene.ObjectCount(), "combat objects must not leak into the next encounter")
}

func TestPirateVictoryCreatesOriginalSalvageFrigate(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithPlayerAt(50, 50).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()
	sourcePosition := references.Position{X: 51, Y: 50}
	sourceEnemy := addPirateTestEnemy(t, gs, sourcePosition, indexes.PirateShipUp)
	require.NoError(t, gs.StartOverworldCombatEncounter(sourceEnemy))
	gs.CombatScene.Victory = true

	assert.True(t, gs.ActionEscapeCombatMap())

	assert.Nil(t, gs.CombatScene)
	assert.Nil(t, gs.GetCurrentLargeMapNPCAIController().GetNpcs().
		GetVisibleEnemyAtPositionAndFloorOrNil(sourcePosition, gs.MapState.PlayerLocation.Floor))
	frigate := gs.GetCurrentLargeMapNPCAIController().GetNpcs().
		GetVehicleAtPositionAndFloorOrNil(sourcePosition, gs.MapState.PlayerLocation.Floor)
	require.NotNil(t, frigate)
	assert.Equal(t, references.FrigateVehicle, frigate.GetVehicleDetails().VehicleType)
	assert.Equal(t, 99, frigate.GetVehicleDetails().GetHullHP())
	assert.Equal(t, 2, frigate.GetVehicleDetails().GetSkiffQuantity())
	assert.Equal(t, references.Up, frigate.GetVehicleDetails().CurrentDirection())
}

func TestPirateDefeatDoesNotCreateSalvageFrigate(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithPlayerAt(50, 50).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()
	sourcePosition := references.Position{X: 51, Y: 50}
	sourceEnemy := addPirateTestEnemy(t, gs, sourcePosition, indexes.PirateShipUp)
	require.NoError(t, gs.StartOverworldCombatEncounter(sourceEnemy))

	gs.EndCombat(CombatDefeat)

	assert.Nil(t, gs.CombatScene)
	assert.Nil(t, gs.GetCurrentLargeMapNPCAIController().GetNpcs().
		GetVisibleEnemyAtPositionAndFloorOrNil(sourcePosition, gs.MapState.PlayerLocation.Floor))
	assert.Nil(t, gs.GetCurrentLargeMapNPCAIController().GetNpcs().
		GetVehicleAtPositionAndFloorOrNil(sourcePosition, gs.MapState.PlayerLocation.Floor))
}

func TestFrigateEncounterFurlsSailsAndPreservesRemainingPartyVehicleState(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithRandomSeed(179).
		WithSystemCallbacks().
		Build()
	playerPosition, enemyPosition := findAdjacentDeepWaterPositions(t, gs)
	gs.MapState.PlayerLocation.Position = playerPosition
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		playerPosition,
		gs.MapState.PlayerLocation.Floor,
	)
	details := frigate.GetVehicleDetails()
	details.SetHullHP(77)
	details.SetSkiffQuantity(1)
	details.SetSailState(map_units.SailRaised)
	details.SetPartyVehicleDirection(references.Left)
	details.HasHMSCape = true
	gs.PartyVehicle = *frigate
	sourceEnemy := addOverworldTestEnemy(t, gs, references.MonsterShark, enemyPosition)

	require.NoError(t, gs.StartOverworldCombatEncounter(sourceEnemy))
	assert.Equal(t, map_units.SailFurled, gs.PartyVehicle.GetVehicleDetails().GetSailState())
	assert.False(t, gs.IsSailing())
	gs.EndCombat(CombatDefeat)

	restoredDetails := gs.PartyVehicle.GetVehicleDetails()
	assert.Equal(t, references.FrigateVehicle, restoredDetails.VehicleType)
	assert.Equal(t, 77, restoredDetails.GetHullHP())
	assert.Equal(t, 1, restoredDetails.GetSkiffQuantity())
	assert.Equal(t, map_units.SailFurled, restoredDetails.GetSailState())
	assert.False(t, gs.IsSailing(), "combat return leaves movement under manual control")
	assert.Equal(t, references.Left, restoredDetails.CurrentDirection())
	assert.True(t, restoredDetails.HasHMSCape)
}

func TestEndCombatKeepsDebugTransitionIndependentOfOverworldOrigin(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	require.NoError(t, gs.TransitionToCombatMap(
		int(references.GladeCombatMap),
		references.BritanniaCombatTerritory,
		CombatEntryNorth,
	))
	require.Nil(t, gs.CombatScene.OverworldOrigin)

	assert.NotPanics(t, func() {
		gs.EndCombat(CombatVictory)
	})
	assert.Nil(t, gs.CombatScene)
}

func addOverworldTestEnemy(
	t *testing.T,
	gs *GameState,
	monsterType references.MonsterType,
	position references.Position,
) *map_units.NPCEnemy {
	t.Helper()

	enemyRef, err := gs.enemyReferenceForCombat(monsterType)
	require.NoError(t, err)
	enemy := map_units.NewEnemyNPC(*enemyRef, 0)
	enemy.SetPos(position)
	enemy.SetFloor(gs.MapState.PlayerLocation.Floor)
	require.True(t, gs.GetCurrentLargeMapNPCAIController().GetNpcs().AddEnemy(enemy))

	result := gs.GetCurrentLargeMapNPCAIController().GetNpcs().
		GetVisibleEnemyAtPositionAndFloorOrNil(position, gs.MapState.PlayerLocation.Floor)
	require.NotNil(t, result)

	return result
}

func addPirateTestEnemy(
	t *testing.T,
	gs *GameState,
	position references.Position,
	spriteIndex indexes.SpriteIndex,
) *map_units.NPCEnemy {
	t.Helper()

	enemyRef, err := gs.enemyReferenceForCombat(references.MonsterStocks)
	require.NoError(t, err)
	pirateRef := *enemyRef
	pirateRef.KeyFrameTile = gs.GameReferences.TileReferences.GetTile(spriteIndex)
	pirateRef.ActivelyAttacks = true
	pirateRef.IsWaterEnemy = true
	enemy := map_units.NewEnemyNPC(pirateRef, 0)
	enemy.SetPos(position)
	enemy.SetFloor(gs.MapState.PlayerLocation.Floor)
	require.True(t, gs.GetCurrentLargeMapNPCAIController().GetNpcs().AddEnemy(enemy))

	result := gs.GetCurrentLargeMapNPCAIController().GetNpcs().
		GetVisibleEnemyAtPositionAndFloorOrNil(position, gs.MapState.PlayerLocation.Floor)
	require.NotNil(t, result)

	return result
}

func findAdjacentDeepWaterPositions(t *testing.T, gs *GameState) (references.Position, references.Position) {
	t.Helper()

	for y := references.Coordinate(0); y < references.YLargeMapTiles; y++ {
		for x := references.Coordinate(0); x < references.XLargeMapTiles; x++ {
			first := references.Position{X: x, Y: y}
			if !gs.GetCurrentLayeredMap().GetTopTile(&first).Is(indexes.Water1) {
				continue
			}
			for _, direction := range []references.Direction{
				references.Up,
				references.Right,
				references.Down,
				references.Left,
			} {
				second := direction.GetNewPositionInDirection(&first).
					GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
				if gs.GetCurrentLayeredMap().GetTopTile(second).Is(indexes.Water1) {
					return first, *second
				}
			}
		}
	}

	t.Fatal("expected adjacent deep-water positions")

	return references.Position{}, references.Position{}
}

func findThreeWalkableLandPositions(
	t *testing.T,
	gs *GameState,
) (references.Position, references.Position, references.Position) {
	t.Helper()

	for y := references.Coordinate(1); y < references.YLargeMapTiles-1; y++ {
		for x := references.Coordinate(1); x < references.XLargeMapTiles-2; x++ {
			first := references.Position{X: x, Y: y}
			second := references.Position{X: x + 1, Y: y}
			third := references.Position{X: x + 2, Y: y}
			if gs.GetCurrentLayeredMap().GetTopTile(&first).Is(indexes.Grass) &&
				gs.GetCurrentLayeredMap().GetTopTile(&second).Is(indexes.Grass) &&
				gs.GetCurrentLayeredMap().GetTopTile(&third).Is(indexes.Grass) {
				return first, second, third
			}
		}
	}

	t.Fatal("expected three consecutive walkable land positions")

	return references.Position{}, references.Position{}, references.Position{}
}

func newContactTestEnemy(
	monsterType references.MonsterType,
	spriteIndex indexes.SpriteIndex,
	activelyAttacks bool,
	isWaterEnemy bool,
	isSandEnemy bool,
) *map_units.NPCEnemy {
	var keyFrameTile *references.Tile
	if spriteIndex != 0 {
		keyFrameTile = &references.Tile{
			Index: spriteIndex,
		}
	}

	enemyReference := references.EnemyReference{
		MonsterType:     monsterType,
		KeyFrameTile:    keyFrameTile,
		ActivelyAttacks: activelyAttacks,
		IsWaterEnemy:    isWaterEnemy,
		IsSandEnemy:     isSandEnemy,
		EnemyAbilities:  make(map[references.EnemyAbility]bool),
	}
	enemy := map_units.NewEnemyNPC(enemyReference, 0)
	enemy.SetVisible(true)

	return &enemy
}
