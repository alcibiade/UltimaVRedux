package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/camping"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestCheckForAmbushDuringHourUsesDeterministicOddsOrForcedResult(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		forceAmbush bool
		want        bool
	}{
		{name: "deterministic miss", want: false},
		{name: "forced", forceAmbush: true, want: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld).
				WithSystemCallbacks().
				Build()
			state.SetRandomSeed(12345)

			assert.Equal(t, testCase.want, state.CheckForAmbushDuringHour(testCase.forceAmbush))
		})
	}
}

func TestCampingWorkflowNoAmbushCompletesWithoutCombat(t *testing.T) {
	state, callbacks := newCampingAmbushTestState(t)
	before := state.MapState.PlayerLocation
	installAutomaticAnyKey(state)

	require.True(t, state.ActionHoleUpLargeMapWithDebug(1, NoGuardAssigned, false, false))
	runCampingWorkflowToCompletion(t, state)

	assert.Nil(t, state.CampingScene)
	assert.Nil(t, state.CombatScene)
	assert.False(t, state.HasPendingOverworldContact())
	assert.Equal(t, before, state.MapState.PlayerLocation)
	assert.Equal(t, 1, state.GetActivePlayerIndex())
	assert.Zero(t, state.LastCamped)
	for partyIndex := range party_state.MaxActiveParty {
		assert.True(t, state.PartyState.GetCharacter(partyIndex).IsGood())
	}
	assert.Equal(t, 1, callbacks.PlaybackRequestCount(SoundWind, SoundPlaybackLoop))
	assert.Equal(t, 1, callbacks.PlaybackRequestCount(SoundFireCrackle, SoundPlaybackLoop))
	assert.Equal(t, 1, callbacks.StopRequestCount(SoundWind))
	assert.Equal(t, 1, callbacks.StopRequestCount(SoundFireCrackle))
}

func TestCampingWorkflowCompletionCleanupRunsExactlyOnce(t *testing.T) {
	state, _ := newCampingAmbushTestState(t)
	installAutomaticAnyKey(state)
	campingMap, err := references.LoadCampingMap(state.GameConfig)
	require.NoError(t, err)
	scene := camping.NewCampingSceneWithCombatRef(
		1,
		NoGuardAssigned,
		state.MapState.PlayerLocation,
		campingMap,
	)
	cleanupCalls := 0
	workflow := state.EventQueueManager.Workflow("camping_completion_cleanup").
		Priority(event_queue.PriorityNormal).
		Sequential().
		BlockInput().
		First(func() {})
	state.AddCampingSequenceToBuilder(workflow, scene, false, false, func() { cleanupCalls++ })
	workflow.Start()

	runCampingWorkflowToCompletion(t, state)

	assert.Equal(t, 1, cleanupCalls)
}

func TestOutdoorCampingUsesExactFiveMinuteCyclesWithoutIntermediatePlayerUpdates(t *testing.T) {
	state, _ := newCampingAmbushTestState(t)
	state.DateTime.Hour, state.DateTime.Minute = 10, 17
	state.lastProcessedHour = 10
	before := state.DateTime
	foodBefore := uint16(100)
	state.PartyState.Inventory.Provisions.Food.Set(foodBefore)
	poisoned := state.PartyState.GetCharacter(1)
	poisoned.SetStatus(party_state.Poisoned)
	poisoned.SetHp(20)
	poisoned.SetEquippedItem(references.SlotRing, references.RingRegeneration)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 4}
	scene := camping.NewCampingScene(2, NoGuardAssigned, state.MapState.PlayerLocation)
	campState := state.newCampingState(scene, true, false)

	assert.Equal(t, 5, scene.CycleMinutes)
	assert.Equal(t, 24, scene.TotalCycles)
	for scene.AdvanceExactCycle() {
		assert.False(t, state.processSingleCampingCycle(scene, campState))
	}

	want := before
	want.Advance(2 * 60)
	want.Turn = before.Turn + uint32(scene.TotalCycles)
	assert.Equal(t, want, state.DateTime)
	assert.Equal(t, 24, scene.CurrentCycle)
	assert.Equal(t, foodBefore, state.PartyState.Inventory.Provisions.Food.Get())
	hpAfterCamp := poisoned.GetCurrentHp()
	assert.GreaterOrEqual(t, hpAfterCamp, uint16(20))
	assert.Equal(t, 4, state.ActiveDurationEffect.TurnsRemaining)
	assert.Equal(t, uint64(24), state.GetRNG().Calls(), "one regeneration sweep must run per source loop")

	poisoned.SetEquippedItem(references.SlotRing, references.NoEquipment)
	state.CompleteSurfaceCommandCycle(references.LargeMapType)
	assert.Equal(t, hpAfterCamp-1, poisoned.GetCurrentHp(), "the owning command runs one final poison update")
	assert.Equal(t, 3, state.ActiveDurationEffect.TurnsRemaining)
}

func TestOutdoorCampingAmbushDefersOwningUpdateUntilCombatReturns(t *testing.T) {
	state, _ := newCampingAmbushTestState(t)
	before := state.DateTime
	poisoned := state.PartyState.GetCharacter(1)
	poisoned.SetStatus(party_state.Poisoned)
	poisoned.SetHp(20)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 4}
	campingMap, err := references.LoadCampingMap(state.GameConfig)
	require.NoError(t, err)
	scene := camping.NewCampingSceneWithCombatRef(6, 0, state.MapState.PlayerLocation, campingMap)
	require.NoError(t, state.TransitionToCampingMap())
	campState := state.newCampingState(scene, false, false)
	state.initializeCampingPartyRings(campState)
	state.setupCampingScene(scene)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 4}
	_, err = state.QueueOutdoorCampingAmbush(scene, campState, 1)
	require.NoError(t, err)
	state.TransitionFromCampingMap()
	state.MapState.RestorePlayerLocation(scene.GetOriginalPlayerLocation())
	state.CampingScene = nil

	state.CompleteSurfaceCommandCycle(references.LargeMapType)
	assert.Equal(t, before, state.DateTime)
	assert.Equal(t, uint16(20), poisoned.GetCurrentHp())
	assert.Equal(t, 4, state.ActiveDurationEffect.TurnsRemaining)

	require.NoError(t, state.ResolvePendingOverworldContact())
	require.NotNil(t, state.CombatScene)
	state.EndCombat(CombatVictory)
	want := before
	want.Advance(DefaultLargeMapMinutesPerTurn)
	assert.Equal(t, want, state.DateTime)
	assert.Equal(t, uint16(19), poisoned.GetCurrentHp())
	assert.Equal(t, 3, state.ActiveDurationEffect.TurnsRemaining)
}

func TestCampingSetupClearsDurationAndOnlySleepsActiveParty(t *testing.T) {
	state, _ := newCampingAmbushTestState(t)
	before := state.MapState.PlayerLocation
	inactive := state.PartyState.GetCharacter(party_state.MaxActiveParty)
	inactive.PartyMap = party_state.PartyMapUnrecruited
	inactive.SetStatus(party_state.Sleep)
	poisoned := state.PartyState.GetCharacter(2)
	poisoned.SetStatus(party_state.Poisoned)
	state.ActiveDurationEffect = DurationEffect{
		EffectType: DurationEffectTimeStop, TurnsRemaining: 9,
	}
	campingMap, err := references.LoadCampingMap(state.GameConfig)
	require.NoError(t, err)
	scene := camping.NewCampingSceneWithCombatRef(6, 0, before, campingMap)

	state.setupCampingScene(scene)

	assert.Equal(t, DurationEffectNone, state.ActiveDurationEffect.EffectType)
	assert.Equal(t, -1, state.GetActivePlayerIndex())
	assert.True(t, state.PartyState.GetCharacter(0).IsGood(), "guard remains awake")
	assert.True(t, state.PartyState.GetCharacter(1).IsAsleep())
	assert.True(t, poisoned.IsPoisoned())
	assert.True(t, inactive.IsAsleep(), "inactive roster status must remain untouched")
}

func TestCampingHealingSkipsInactiveRoster(t *testing.T) {
	state, callbacks := newCampingAmbushTestState(t)
	active := state.PartyState.GetCharacter(0)
	active.SetHp(1)
	active.SetStatus(party_state.Good)
	inactive := state.PartyState.GetCharacter(party_state.MaxActiveParty)
	inactive.PartyMap = party_state.PartyMapUnrecruited
	inactive.SetHp(1)
	inactive.SetStatus(party_state.Good)
	scene := camping.NewCampingScene(6, NoGuardAssigned, state.MapState.PlayerLocation)

	state.applyCampingHealing(scene)

	assert.Greater(t, active.GetCurrentHp(), uint16(1))
	assert.Equal(t, uint16(1), inactive.GetCurrentHp())
	assert.Equal(t, []TransientIconKind{TransientIconHealing}, callbacks.TransientIconCalls)
}

func TestCampingAmbushChecksOnlyAfterLiveHourTransition(t *testing.T) {
	state, _ := newCampingAmbushTestState(t)
	state.DateTime.Hour = 10
	state.DateTime.Minute = 45
	scene := camping.NewCampingScene(2, NoGuardAssigned, state.MapState.PlayerLocation)
	campingState := &CampingState{
		AmbushedAt: NoAmbush, AmbushCheckHour: state.DateTime.Hour, ForceWanderer: true,
	}

	scene.CurrentCycle = 1
	assert.False(t, state.processSingleCampingCycle(scene, campingState))
	assert.Equal(t, byte(10), campingState.AmbushCheckHour)
	scene.CurrentCycle = 2
	assert.False(t, state.processSingleCampingCycle(scene, campingState))
	assert.Equal(t, byte(10), campingState.AmbushCheckHour)
	scene.CurrentCycle = 3
	assert.False(t, state.processSingleCampingCycle(scene, campingState))
	assert.Equal(t, byte(10), campingState.AmbushCheckHour)
	scene.CurrentCycle = 4
	assert.False(t, state.processSingleCampingCycle(scene, campingState))
	assert.Equal(t, byte(11), campingState.AmbushCheckHour)
}

func TestCampingGuardMovesBeforeNextHourAmbushUsesPosition(t *testing.T) {
	state, _ := newCampingAmbushTestState(t)
	state.DateTime.Hour = 10
	state.DateTime.Minute = 50
	state.SetRandomSeed(campingBoundaryGuardMoveSeed(t))
	before := state.MapState.PlayerLocation
	campingMap, err := references.LoadCampingMap(state.GameConfig)
	require.NoError(t, err)
	scene := camping.NewCampingSceneWithCombatRef(2, 0, before, campingMap)
	require.NoError(t, state.TransitionToCampingMap())
	state.setupCampingScene(scene)
	campingState := &CampingState{
		GuardIndex: 0, GuardPosition: scene.GetPartyMemberPosition(0), AmbushedAt: NoAmbush,
		AmbushCheckHour: state.DateTime.Hour, ActivePlayerIndex: 1,
	}
	state.initializeCampingPartyRings(campingState)
	initialGuardPosition := campingState.GuardPosition

	scene.CurrentCycle = 1
	assert.False(t, state.processSingleCampingCycle(scene, campingState))
	assert.Equal(t, initialGuardPosition, campingState.GuardPosition)
	scene.CurrentCycle = 2
	assert.False(t, state.processSingleCampingCycle(scene, campingState))
	movedGuardPosition := campingState.GuardPosition
	require.NotEqual(t, initialGuardPosition, movedGuardPosition)
	scene.CurrentCycle = 3
	assert.True(t, state.processSingleCampingCycle(scene, campingState))
	require.NotNil(t, state.pendingCampingCombat)
	assert.Equal(t, movedGuardPosition, state.pendingCampingCombat.combat.PartyPositions[0])
}

func TestCampingAmbushPreparationFailureInterruptsWithoutLeavingSleep(t *testing.T) {
	state, _ := newCampingAmbushTestState(t)
	before := state.MapState.PlayerLocation
	campingMap, err := references.LoadCampingMap(state.GameConfig)
	require.NoError(t, err)
	scene := camping.NewCampingSceneWithCombatRef(6, NoGuardAssigned, before, campingMap)
	require.NoError(t, state.TransitionToCampingMap())
	activePlayerIndex := state.GetActivePlayerIndex()
	state.setupCampingScene(scene)
	emptyEnemyReferences := references.EnemyReferences{}
	gameReferences := *state.GameReferences
	gameReferences.EnemyReferences = &emptyEnemyReferences
	state.GameReferences = &gameReferences
	scene.CurrentCycle = DebugAmbushCycle
	campingState := &CampingState{
		AmbushedAt: NoAmbush, ForceAmbush: true, AmbushCheckHour: state.DateTime.Hour,
		ActivePlayerIndex: activePlayerIndex,
	}
	state.initializeCampingPartyRings(campingState)

	assert.True(t, state.processSingleCampingCycle(scene, campingState))
	assert.True(t, campingState.Interrupted)
	assert.Nil(t, state.pendingCampingCombat)
	state.cleanupCampingScene(scene, campingState)
	for partyIndex := range party_state.MaxActiveParty {
		assert.False(t, state.PartyState.GetCharacter(partyIndex).IsAsleep())
	}
}

func TestCampingWorkflowQueuesRealAmbushAndPreservesGuardSleepPolicy(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		guardIndex int
		wantSleep  bool
	}{
		{name: "guard wakes party", guardIndex: 0, wantSleep: false},
		{name: "no guard carries sleep into combat", guardIndex: NoGuardAssigned, wantSleep: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state, callbacks := newCampingAmbushTestState(t)
			before := state.MapState.PlayerLocation
			beforeHP := state.PartyState.GetCharacter(1).GetCurrentHp()
			installAutomaticAnyKey(state)

			require.True(t, state.ActionHoleUpLargeMapWithDebug(
				6,
				testCase.guardIndex,
				false,
				true,
			))
			runCampingWorkflowToCompletion(t, state)

			assert.Nil(t, state.CampingScene)
			assert.Nil(t, state.CombatScene, "combat must wait for workflow ownership release")
			assert.NotNil(t, state.pendingCampingCombat)
			assert.Equal(t, before, state.MapState.PlayerLocation)
			assert.Equal(t, beforeHP, state.PartyState.GetCharacter(1).GetCurrentHp())
			assert.Zero(t, state.LastCamped, "ambush must skip normal rest benefits")
			assert.True(t, callbacks.MessagesContain("Ambushed!"))
			assert.False(t, callbacks.MessagesContain("Party rested!"))
			assert.False(t, callbacks.MessagesContain("Combat would start here"))
			assert.Equal(t, 1, callbacks.PlaybackRequestCount(SoundFireCrackle, SoundPlaybackLoop))
			assert.Equal(t, 1, callbacks.StopRequestCount(SoundFireCrackle), "fire stops before the combat handoff")

			require.NoError(t, state.ResolvePendingOverworldContact())
			require.NotNil(t, state.CombatScene)
			require.NotNil(t, state.CombatScene.CampingOrigin)
			assert.Equal(t, int(references.CampFireCombatMap), state.CombatScene.CombatMapRef.MapIndex)
			assert.Equal(t,
				state.CombatScene.CampingOrigin.Ambusher,
				state.CombatScene.MonsterManager.GetMonster(0).MonsterType(),
			)
			for partyIndex := range party_state.MaxActiveParty {
				character := state.PartyState.GetCharacter(partyIndex)
				assert.Equal(t, testCase.wantSleep, character.IsAsleep())
				assert.True(t, state.CombatScene.PartyActive[partyIndex])
				assert.Equal(t, testCase.wantSleep, state.CombatScene.PartySleeping[partyIndex])
			}
		})
	}
}

func TestCampingCombatReturnsToExactOriginForEveryOutcome(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		reason CombatEndReason
	}{
		{name: "victory", reason: CombatVictory},
		{name: "defeat", reason: CombatDefeat},
		{name: "evacuated", reason: CombatEvacuated},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state, before := startCampingCombatForTest(t, 0)
			deathSequences := 0
			state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() { deathSequences++ }

			state.EndCombat(testCase.reason)

			assert.Nil(t, state.CombatScene)
			assert.Equal(t, before, state.MapState.PlayerLocation)
			assert.Equal(t, 1, state.GetActivePlayerIndex())
			assert.Zero(t, deathSequences, "defeat by fleeing is not a party wipe")
			assert.False(t, state.HasPendingOverworldContact())
		})
	}
}

func TestCampingCombatTruePartyLossRestoresOriginBeforeDeath(t *testing.T) {
	state, before := startCampingCombatForTest(t, 0)
	deathSequences := 0
	state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() {
		assert.Equal(t, before, state.MapState.PlayerLocation)
		deathSequences++
	}
	for partyIndex := range party_state.MaxActiveParty {
		character := state.PartyState.GetCharacter(partyIndex)
		character.SetHp(0)
		character.MarkAsDead()
	}

	state.EndCombat(CombatDefeat)

	assert.Nil(t, state.CombatScene)
	assert.Equal(t, before, state.MapState.PlayerLocation)
	assert.Equal(t, -1, state.GetActivePlayerIndex())
	assert.Equal(t, 1, deathSequences)
}

func TestCampingCombatPendingAndActiveContinuationsRejectSaveAndDebugExit(t *testing.T) {
	state, _ := newCampingAmbushTestState(t)
	before := state.MapState.PlayerLocation
	campingMap, err := references.LoadCampingMap(state.GameConfig)
	require.NoError(t, err)
	scene := camping.NewCampingSceneWithCombatRef(6, 0, before, campingMap)
	require.NoError(t, state.TransitionToCampingMap())
	activePlayerIndex := state.GetActivePlayerIndex()
	campState := state.newCampingState(scene, false, false)
	state.initializeCampingPartyRings(campState)
	state.setupCampingScene(scene)

	_, err = state.QueueOutdoorCampingAmbush(scene, campState, activePlayerIndex)
	require.NoError(t, err)
	assert.ErrorIs(t, state.ValidateSaveSnapshotEligibility(), ErrSaveSnapshotNotRepresentable)

	state.TransitionFromCampingMap()
	state.MapState.RestorePlayerLocation(before)
	state.CampingScene = nil
	require.NoError(t, state.ResolvePendingOverworldContact())
	result, err := state.ExitDebugCombat(state.CombatScene.EncounterID)
	assert.ErrorIs(t, err, ErrDebugCombatBusy)
	assert.Equal(t, DebugCombatExitResult{}, result)
	assert.NotNil(t, state.CombatScene)
}

func TestCampingAmbushFixedEightMemberGroupsRemainFixed(t *testing.T) {
	state, _ := newCampingAmbushTestState(t)

	assert.Equal(t, 8, state.resolveCampingAmbushCount(8))
}

func TestCampingAmbushUsesWeightedPoolAndInterleavedSourceRNGOrder(t *testing.T) {
	state, _ := newCampingAmbushTestState(t)
	state.DateTime.Year = 2
	state.DateTime.Month = 1
	state.DateTime.Day = 1
	state.openingCampaignEncounterReduction = false
	wantPool := [8]references.MonsterType{
		references.MonsterTroll,
		references.MonsterRat,
		references.MonsterBat,
		references.MonsterSlime,
		references.MonsterSpider,
		references.MonsterGremlin,
		references.MonsterHeadless,
		references.MonsterRat,
	}
	assert.Equal(t, wantPool, references.CampingAmbushMonsters)

	var ambusher references.MonsterType
	var enemyRef *references.EnemyReference
	for _, candidate := range references.CampingAmbushMonsters {
		candidateRef, err := state.enemyReferenceForCombat(candidate)
		require.NoError(t, err)
		if candidateRef.MaxPerMap != 1 && candidateRef.MaxPerMap != 8 &&
			candidateRef.MaxPerMap != references.MaxMonstersInCombat && candidateRef.Friend != nil {
			ambusher = candidate
			enemyRef = candidateRef

			break
		}
	}
	require.NotNil(t, enemyRef, "fixture requires a variable-size camping monster with a friend")
	combatMap, err := references.LoadCampingMap(state.GameConfig)
	require.NoError(t, err)
	scene := camping.NewCampingSceneWithCombatRef(6, NoGuardAssigned, state.MapState.PlayerLocation, combatMap)
	campState := state.newCampingState(scene, false, false)
	state.initializeCampingPartyRings(campState)
	combat, err := state.prepareCampingParty(scene, campState, combatMap)
	require.NoError(t, err)
	seed := uint64(8743)
	state.SetRandomSeed(seed)
	expected := beradrng.New(seed)
	expectedPositions := combatMap.MonsterPositions
	for positionIndex := 0; positionIndex < len(expectedPositions)-1; positionIndex++ {
		randomIndex := expected.Intn(len(expectedPositions))
		expectedPositions[positionIndex], expectedPositions[randomIndex] =
			expectedPositions[randomIndex], expectedPositions[positionIndex]
	}
	expectedCount := expected.Intn(enemyRef.MaxPerMap) + 1
	expectedTypes := make([]references.MonsterType, expectedCount)
	expectedDexterity := make([]int, expectedCount)
	for monsterIndex := range expectedCount {
		expectedTypes[monsterIndex] = ambusher
		if monsterIndex > 0 && enemyRef.Friend != nil &&
			monsterIndex < expectedCount/references.FriendDivisor+1 && expected.Intn(9) == 0 {
			expectedTypes[monsterIndex] = enemyRef.Friend.MonsterType
		}
		monsterRef, refErr := state.enemyReferenceForCombat(expectedTypes[monsterIndex])
		require.NoError(t, refErr)
		dexterity := monsterRef.Dexterity + expected.Intn(8) - 4
		if dexterity < 0 || dexterity > maxDexterity {
			dexterity = monsterRef.Dexterity
		}
		expectedDexterity[monsterIndex] = dexterity
	}

	result, err := state.populateOutdoorCampingCombat(combat, ambusher)
	require.NoError(t, err)

	assert.Equal(t, expected.Calls(), state.GetRNG().Calls())
	assert.Equal(t, expectedCount, result.PrimaryCount+result.FriendCount)
	for monsterIndex := range expectedCount {
		monster := combat.MonsterManager.GetMonster(monsterIndex)
		require.NotNil(t, monster)
		assert.Equal(t, expectedTypes[monsterIndex], monster.MonsterType())
		assert.Equal(t, expectedPositions[monsterIndex], monster.Position())
		assert.Equal(t, expectedDexterity[monsterIndex], monster.CurrentDex())
	}
}

func newCampingAmbushTestState(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(42, 73).
		WithSystemCallbacks().
		WithRandomSeed(12345).
		Build()
	for partyIndex := range party_state.MaxActiveParty {
		character := state.PartyState.GetCharacter(partyIndex)
		character.PartyMap = party_state.PartyMapActive
		character.SetStatus(party_state.Good)
		character.SetHp(character.GetMaxHp())
	}
	state.PartyState.GetCharacter(1).SubtractHp(10)
	require.True(t, state.SetActivePlayer(1))
	state.LastCamped = 0

	return state, callbacks
}

func startCampingCombatForTest(
	t *testing.T,
	guardIndex int,
) (*GameState, references.PlayerLocation) {
	t.Helper()
	state, _ := newCampingAmbushTestState(t)
	before := state.MapState.PlayerLocation
	campingMap, err := references.LoadCampingMap(state.GameConfig)
	require.NoError(t, err)
	scene := camping.NewCampingSceneWithCombatRef(6, guardIndex, before, campingMap)
	require.NoError(t, state.TransitionToCampingMap())
	activePlayerIndex := state.GetActivePlayerIndex()
	campState := state.newCampingState(scene, false, false)
	state.initializeCampingPartyRings(campState)
	state.setupCampingScene(scene)
	_, err = state.QueueOutdoorCampingAmbush(scene, campState, activePlayerIndex)
	require.NoError(t, err)
	state.TransitionFromCampingMap()
	state.MapState.RestorePlayerLocation(before)
	state.CampingScene = nil
	require.NoError(t, state.ResolvePendingOverworldContact())
	require.NotNil(t, state.CombatScene)

	return state, before
}

func runCampingWorkflowToCompletion(t *testing.T, state *GameState) {
	t.Helper()
	for update := 0; update < 2000 && state.EventQueueManager.HasActiveWorkflows(); update++ {
		state.ElapsedMs += 500
		state.EventQueueManager.Update()
	}
	require.False(t, state.EventQueueManager.HasActiveWorkflows(), "camping workflow did not complete")
}

func installAutomaticAnyKey(state *GameState) {
	checks := 0
	state.SystemCallbacks.Visual.IsAnyKeyPressed = func() bool {
		checks++

		return checks%2 == 0
	}
}

func campingBoundaryGuardMoveSeed(t *testing.T) uint64 {
	t.Helper()
	for seed := uint64(1); seed < 100_000; seed++ {
		rng := beradrng.New(seed)
		if rng.Intn(GuardMoveOdds) == 0 || rng.Intn(GuardMoveOdds) != 0 ||
			rng.Intn(DirectionCount) != 0 {
			continue
		}
		if rng.Intn(64) == 0 {
			return seed
		}
	}

	t.Fatal("unable to find camping boundary guard-move seed")

	return 0
}
