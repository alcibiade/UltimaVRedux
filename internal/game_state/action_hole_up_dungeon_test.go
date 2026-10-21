package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/camping"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestDungeonCampingUninterruptedRestUsesExactFiveMinuteCycles(t *testing.T) {
	state, callbacks := newDungeonCampingTestStateWithCallbacks(t)
	var cells references.DungeonCells
	cells.SetCell(0, state.MapState.PlayerLocation.Position, 0x40) // The chest replaces the campfire.
	attachDungeonStateForTest(t, state, cells)
	dungeonBefore := state.activeDungeon
	token, start := state.BeginDungeonCommandCycle()
	require.Equal(t, 1, start.ElapsedMinutes)
	state.ActionHoleUpDungeonMap(token)
	scene, plan, err := state.PrepareDungeonCampingPlan(6, NoGuardAssigned, token)
	require.NoError(t, err)
	require.NotEqual(t, indexes.SpriteIndex(indexes.CampFire), plan.combatMap.GetTileNumber(scene.CampingMap.CampfirePosition))
	state.SetRandomSeed(dungeonCampingNoAmbushSeed(t, 5))
	installAutomaticAnyKey(state)
	workflow := state.EventQueueManager.Workflow("dungeon_camping_rest_test").
		Priority(event_queue.PriorityNormal).Sequential().BlockInput().First(func() {})
	state.AddDungeonCampingSequenceToBuilder(workflow, scene, plan, func() {
		state.CompleteDungeonCampingCommand(token)
	})
	workflow.Start()

	runCampingWorkflowToCompletion(t, state)

	assert.Equal(t, 6*60, scene.TimeElapsed)
	assert.Equal(t, 6*campingDungeonCyclesPerHourForTest, scene.CurrentCycle)
	assert.Equal(t, 14, state.LastCamped)
	assert.Same(t, dungeonBefore, state.activeDungeon)
	assert.Equal(t, references.Deceit, state.MapState.PlayerLocation.Location)
	assert.True(t, state.IsAtDungeonCommandBoundary())
	assert.Nil(t, state.pendingCampingCombat)
	assert.Nil(t, state.CombatScene)
	assert.Equal(t, 1, callbacks.PlaybackRequestCount(SoundWind, SoundPlaybackLoop))
	assert.Zero(t, callbacks.PlaybackRequestCount(SoundFireCrackle, SoundPlaybackLoop))
	assert.Equal(t, 1, callbacks.StopRequestCount(SoundWind))
	assert.Equal(t, 1, callbacks.StopRequestCount(SoundFireCrackle))
}

func TestDungeonCampingDefersMealConsumptionUntilCommandCompletion(t *testing.T) {
	state, callbacks := newDungeonCampingTestStateWithCallbacks(t)
	state.DateTime.Hour = 5
	state.DateTime.Minute = 1
	state.lastProcessedHour = 5
	state.PartyState.Inventory.Provisions.Food.Set(100)
	scene := camping.NewDungeonCampingScene(1, NoGuardAssigned, state.MapState.PlayerLocation, &references.CombatMapReference{})
	plan := &DungeonCampingPlan{}
	campState := state.newCampingState(scene, false, false)

	for range camping.DungeonCyclesPerHour {
		assert.False(t, state.processDungeonCampingCycle(scene, campState, plan))
	}

	assert.Equal(t, uint16(100), state.PartyState.Inventory.Provisions.Food.Get())
	assert.Equal(t, 5, state.lastProcessedHour)
	state.checkHourlyFoodConsumption()
	assert.Equal(t, uint16(94), state.PartyState.Inventory.Provisions.Food.Get())
	assert.Equal(t, 6, state.lastProcessedHour)
	assert.Empty(t, callbacks.TransientIconCalls)
}

func TestDungeonCommandPrechargeDefersHourlyFoodUntilFullCompletion(t *testing.T) {
	state, callbacks := newDungeonCampingTestStateWithCallbacks(t)
	state.DateTime.Hour = 5
	state.DateTime.Minute = 59
	state.lastProcessedHour = 5
	state.PartyState.Inventory.Provisions.Food.Set(100)

	token, start := state.BeginDungeonCommandCycle()
	require.Equal(t, 1, start.ElapsedMinutes)
	assert.Equal(t, uint16(100), state.PartyState.Inventory.Provisions.Food.Get())
	assert.Equal(t, 5, state.lastProcessedHour)
	state.ClaimDungeonFullCommand(token)
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)

	assert.Equal(t, uint16(94), state.PartyState.Inventory.Provisions.Food.Get())
	assert.Equal(t, 6, state.lastProcessedHour)
	assert.True(t, state.IsAtDungeonCommandBoundary())
	assert.Empty(t, callbacks.TransientIconCalls)
}

func TestDungeonCampingBardCompletionEnablesFirstGuardWanderRoll(t *testing.T) {
	state, callbacks := newDungeonCampingTestStateWithCallbacks(t)
	state.PartyState.GetCharacter(0).Class = party_state.Bard
	scene := camping.NewDungeonCampingScene(1, 0, state.MapState.PlayerLocation, &references.CombatMapReference{})
	campState := state.newCampingState(scene, false, false)
	state.setupBardGuard(scene, campState)
	require.True(t, campState.BardMusicPlaying)
	seed := uint64(1701)
	state.SetRandomSeed(seed)
	expected := beradrng.New(seed)
	_ = expected.Intn(GuardMoveOdds)
	state.finishBardGuardPerformance(campState)

	assert.False(t, state.processDungeonCampingCycle(scene, campState, &DungeonCampingPlan{}))

	assert.False(t, campState.BardMusicPlaying)
	assert.Equal(t, expected.Intn(1000), state.GetRNG().Intn(1000))
	assert.Equal(t, 1, callbacks.PlaybackRequestCount(SoundBardLute, SoundPlaybackLoop))
	assert.Equal(t, 1, callbacks.StopRequestCount(SoundBardLute))
}

func TestLastCampedDecrementsAtHourCrossing(t *testing.T) {
	state := newDungeonCampingTestState(t)
	state.LastCamped = 2
	state.DateTime.Minute = 59

	state.advanceGameplayMinutes(1, false)
	assert.Equal(t, 1, state.LastCamped)
	state.DateTime.Minute = 59
	state.advanceGameplayMinutes(1, false)
	assert.Zero(t, state.LastCamped)
	state.DateTime.Minute = 59
	state.advanceGameplayMinutes(1, false)
	assert.Zero(t, state.LastCamped)
}

func TestDungeonCampingCombatRetainsCommandAndReturnsEveryOutcome(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		reason CombatEndReason
	}{
		{name: "victory", reason: CombatVictory},
		{name: "defeat", reason: CombatDefeat},
		{name: "evacuated", reason: CombatEvacuated},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state := newDungeonCampingTestState(t)
			dungeonBefore := state.activeDungeon
			locationBefore := state.MapState.PlayerLocation
			state.ActiveDurationEffect = DurationEffect{
				EffectType: DurationEffectTimeStop, TurnsRemaining: 8,
			}
			token, _ := state.BeginDungeonCommandCycle()
			state.ActionHoleUpDungeonMap(token)
			scene, plan, err := state.PrepareDungeonCampingPlan(6, 0, token)
			require.NoError(t, err)
			campState := state.newCampingState(scene, false, false)
			require.NoError(t, state.transitionToCampingMap(plan.combatMap))
			state.initializeCampingPartyRings(campState)
			state.setupCampingScene(scene)
			require.NoError(t, state.QueueDungeonCampingAmbush(scene, plan, campState, campState.ActivePlayerIndex))
			campState.AmbushedAt = 1
			campState.Interrupted = true
			state.finalizeCampingSequence(scene, campState, func() {
				state.CompleteDungeonCampingCommand(token)
			})

			assert.False(t, state.IsAtDungeonCommandBoundary())
			require.NotNil(t, state.pendingCampingCombat)
			assert.Equal(t, references.DungeonCombatTerritory, state.pendingCampingCombat.territory)
			require.NoError(t, state.ResolvePendingOverworldContact())
			require.NotNil(t, state.CombatScene)
			assert.False(t, state.IsAtDungeonCommandBoundary())

			state.EndCombat(testCase.reason)

			assert.Equal(t, locationBefore, state.MapState.PlayerLocation)
			assert.Same(t, dungeonBefore, state.activeDungeon)
			assert.Nil(t, state.CombatScene)
			assert.True(t, state.IsAtDungeonCommandBoundary())
		})
	}
}

func TestDungeonCampingCenterChestDisplaysBlocksGuardAndSurvivesAmbushHandoff(t *testing.T) {
	var cells references.DungeonCells
	position := references.Position{X: 1, Y: 1}
	cells.SetCell(7, position, 0x40)
	state := newDungeonStateTestGame(t, cells)
	state.MapState.PlayerLocation.Floor = 7
	enemyReferences := make(references.EnemyReferences, int(references.MonsterShadowlord)+1)
	tileReferences := references.NewTileReferences()
	state.GameReferences = &references.GameReferences{ //nolint:exhaustruct // Asset-free generated-camp fixture.
		EnemyReferences: &enemyReferences,
		TileReferences:  tileReferences,
	}
	command := DungeonCommandCycle{id: 1}
	scene, plan, err := state.PrepareDungeonCampingPlan(1, 0, command)
	require.NoError(t, err)
	enemyReferences[plan.monster] = references.EnemyReference{ //nolint:exhaustruct // Selected homogeneous ambusher.
		MonsterType: plan.monster, Dexterity: 10, HitPoints: 10, MaxPerMap: plan.monsterCount,
	}
	center := references.Position{X: 5, Y: 5}
	assert.Equal(t, indexes.SpriteIndex(0xdc), plan.combatMap.GetTileNumber(center))
	assert.Equal(t, references.FloorNumber(7), scene.GetOriginalPlayerLocation().Floor)

	require.NoError(t, state.transitionToCampingMap(plan.combatMap))
	assert.Equal(t, references.FloorNumber(0), state.MapState.PlayerLocation.Floor)
	campingState := state.newCampingState(scene, false, false)
	state.initializeCampingPartyRings(campingState)
	state.setupCampingScene(scene)
	assert.False(t, state.isValidCampingPosition(&center), "the displayed chest blocks guard movement")

	require.NoError(t, state.QueueDungeonCampingAmbush(scene, plan, campingState, campingState.ActivePlayerIndex))
	require.NotNil(t, state.pendingCampingCombat)
	combat := state.pendingCampingCombat.combat
	chest, found := combat.objectAtMatching(center, CombatObject.IsChest)
	require.True(t, found)
	assert.Equal(t, 28, chest.TreasureValue(), "treasure value uses source dungeon level eight")
	assert.False(t, chest.IsLockedAndTrapped())
	assert.Equal(t, indexes.SpriteIndex(0x45), combat.CombatMapRef.GetTileNumber(center))
	assert.Equal(t, uint8(2), chest.originalObjectSlot, "the retained chest follows the one living PC")
	for _, entry := range combat.TurnQueue.entries {
		if entry.Combatant.IsPC() {
			continue
		}
		assert.Greater(t, entry.originalObjectSlot, chest.originalObjectSlot,
			"ambushers are admitted after the retained center chest")
	}
}

func newDungeonCampingTestState(t *testing.T) *GameState {
	t.Helper()
	state, _ := newDungeonCampingTestStateWithCallbacks(t)

	return state
}

func newDungeonCampingTestStateWithCallbacks(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithPlayerAt(1, 1).
		WithSystemCallbacks().
		WithRandomSeed(4815).
		Build()
	var cells references.DungeonCells
	attachDungeonStateForTest(t, state, cells)
	for index := range party_state.MaxActiveParty {
		character := state.PartyState.GetCharacter(index)
		character.PartyMap = party_state.PartyMapActive
		character.SetStatus(party_state.Good)
		character.SetHp(character.GetMaxHp())
	}
	require.True(t, state.SetActivePlayer(1))
	state.LastCamped = 0

	return state, callbacks
}

const campingDungeonCyclesPerHourForTest = 12

func dungeonCampingNoAmbushSeed(t *testing.T, checks int) uint64 {
	t.Helper()
	for seed := uint64(1); seed < 100_000; seed++ {
		candidate := beradrng.New(seed)
		valid := true
		for range checks {
			if candidate.Intn(64) == 0 {
				valid = false
				break
			}
		}
		if valid {
			return seed
		}
	}
	t.Fatal("unable to find no-ambush dungeon camping seed")

	return 0
}
