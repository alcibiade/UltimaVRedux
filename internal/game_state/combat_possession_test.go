package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// Every expected draw has an exact bound and value. Unexpected draws fail rather
// than cycling a sequence or silently reducing a value modulo the requested bound.
type possessionDraw struct{ bound, value int }
type possessionRNG struct {
	t     *testing.T
	draws []possessionDraw
	calls int
}

func (rng *possessionRNG) Intn(bound int) int {
	rng.t.Helper()
	require.Less(rng.t, rng.calls, len(rng.draws), "unexpected RNG draw with bound %d", bound)
	draw := rng.draws[rng.calls]
	rng.calls++
	require.Equal(rng.t, draw.bound, bound)
	require.GreaterOrEqual(rng.t, draw.value, 0)
	require.Less(rng.t, draw.value, bound)
	return draw.value
}

func setupPossessionEpisode(t *testing.T) (*GameState, *MockSystemCallbacks, *CombatMonster, int, CombatTurnIdentity) {
	t.Helper()
	state, mock, controller, index := setupHostileSpecialMonsterTest(t, references.MonsterShadowlord)
	require.NotNil(t, state)
	// A genuinely friendly second actor prevents the independent emergency policy
	// from freeing the victim before the resistance boundary under test.
	friend := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 1}
	state.CombatScene.PartyActive[1] = true
	state.CombatScene.TurnQueue.GetEntryByIdentity(friend).status = turnQueueActive
	state.CombatScene.TurnQueue.GetEntryByIdentity(friend).sourceSlotReleased = false
	require.True(t, state.CombatScene.registerCombatantObject(friend))
	state.PartyState.GetCharacter(1).SetStatus(party_state.Good)
	state.PartyState.GetCharacter(1).SetMaxHp(1000)
	state.PartyState.GetCharacter(1).SetHp(1000)
	state.CombatScene.SetPartyMemberPosition(1, references.Position{X: 1, Y: 1})
	victim := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}
	state.PartyState.GetCharacter(0).Intelligence = 1
	state.PartyState.GetCharacter(0).SetMaxHp(1000)
	state.PartyState.GetCharacter(0).SetHp(1000)
	rng := &possessionRNG{t: t, draws: []possessionDraw{{61, 60}}}
	state.resolveMonsterPossession(controller, index, victim, rng)
	require.Equal(t, 1, rng.calls)
	require.True(t, state.CombatScene.IsCombatPCPossessed(victim))
	state.PartyState.GetCharacter(0).Intelligence = 20
	mock.Reset()
	return state, mock, controller, index, victim
}

func TestPossessionProbabilityTableAndClamping(t *testing.T) {
	for _, row := range []struct{ turn, at10, at20, at30 int }{
		{1, 0, 0, 0}, {2, 0, 0, 0}, {3, 0, 0, 0},
		{4, 225, 300, 375}, {5, 225, 300, 375}, {6, 225, 300, 375},
		{7, 600, 800, 1000}, {8, 600, 800, 1000}, {9, 600, 800, 1000},
		{10, 1500, 2000, 2500}, {11, 2625, 3500, 4375}, {12, 10000, 10000, 10000},
	} {
		assert.Equal(t, row.at10, combatPossessionThresholdBP(row.turn, 10))
		assert.Equal(t, row.at20, combatPossessionThresholdBP(row.turn, 20))
		assert.Equal(t, row.at30, combatPossessionThresholdBP(row.turn, 30))
		assert.Equal(t, combatPossessionThresholdBP(row.turn, 1), combatPossessionThresholdBP(row.turn, -10))
		assert.Equal(t, row.at30, combatPossessionThresholdBP(row.turn, 255))
		for intelligence := 2; intelligence <= 30; intelligence++ {
			assert.GreaterOrEqual(t, combatPossessionThresholdBP(row.turn, intelligence), combatPossessionThresholdBP(row.turn, intelligence-1))
		}
	}
	assert.Equal(t, 157, combatPossessionThresholdBP(4, 1), "integer basis-point truncation")
	assert.Equal(t, 1837, combatPossessionThresholdBP(11, 1))
}

func TestPossessionTwelveOwnBoundariesExactRNGAndMilestones(t *testing.T) {
	state, mock, _, _, victim := setupPossessionEpisode(t)
	combat := state.CombatScene
	id, _, _ := combat.CombatPossessionProgress(victim)
	rng := &possessionRNG{t: t}
	for range 8 {
		rng.draws = append(rng.draws, possessionDraw{10000, 9999})
	}
	character := state.PartyState.GetCharacter(0)
	character.SetStatus(party_state.Poisoned)
	combat.setCombatPCSleeping(victim, true)
	combat.setCombatPCUnderground(victim, true)
	character.SetEquippedItem(references.SlotWeapon, references.ChaosSword)
	before := *character
	for turn := 1; turn <= 12; turn++ {
		combat.dispatchCombatPossessionTurn(victim)
		ticket := combat.possessionTurn
		state.completeCombatPossessionTurnWithRNG(ticket, rng)
		state.completeCombatPossessionTurnWithRNG(ticket, rng)
		assert.Equal(t, min(8, max(0, turn-3)), rng.calls)
		if turn < 12 {
			gotID, progress, active := combat.CombatPossessionProgress(victim)
			require.True(t, active)
			assert.Equal(t, id, gotID)
			assert.Equal(t, turn, progress)
		}
		// Another actor's completion and detached reads cannot progress this episode.
		combat.dispatchCombatPossessionTurn(CombatTurnIdentity{IsPartyMember: true, EntityIndex: 1})
		state.completeCombatPossessionTurnWithRNG(combat.possessionTurn, rng)
		combat.CombatPossessionProgress(victim)
		state.ProjectCombatActor(victim, 0)
		combat.TurnQueue.FillForecast(make([]CombatTurnForecastEntry, 10))
	}
	assert.False(t, combat.IsCombatPCPossessed(victim))
	assert.Empty(t, combat.possessions)
	assert.Equal(t, before, *character, "natural release preserves roster, HP and equipment")
	assert.True(t, combat.IsCombatPCSleeping(victim))
	assert.True(t, combat.IsCombatPCUnderground(victim))
	require.Len(t, mock.CombatEvents, 4)
	for index, turn := range []int{3, 7, 10, 12} {
		assert.Equal(t, turn, mock.CombatEvents[index].PossessionTurns)
		assert.Equal(t, id, mock.CombatEvents[index].PossessionEpisodeID)
	}
	assert.Equal(t, CombatEventPossessionRelease, mock.CombatEvents[3].Kind)
	assert.Equal(t, CombatPossessionReleaseNatural, mock.CombatEvents[3].PossessionReleaseReason)
}

func TestPossessionRecoveryStrictThresholdAndNoSuccessMilestone(t *testing.T) {
	for _, turn := range []int{4, 7, 10, 11} {
		for _, intelligence := range []int{1, 10, 20, 30} {
			for _, succeeds := range []bool{false, true} {
				t.Run(fmt.Sprintf("turn%d/int%d/success%t", turn, intelligence, succeeds), func(t *testing.T) {
					state, mock, _, _, victim := setupPossessionEpisode(t)
					episode := state.CombatScene.possessions[victim]
					episode.turns = turn - 1
					state.CombatScene.possessions[victim] = episode
					state.PartyState.GetCharacter(0).Intelligence = byte(intelligence)
					roll := combatPossessionThresholdBP(turn, intelligence)
					if succeeds {
						roll--
					}
					rng := &possessionRNG{t: t, draws: []possessionDraw{{10000, roll}}}
					state.CombatScene.dispatchCombatPossessionTurn(victim)
					state.completeCombatPossessionTurnWithRNG(state.CombatScene.possessionTurn, rng)
					assert.Equal(t, 1, rng.calls)
					assert.Equal(t, !succeeds, state.CombatScene.IsCombatPCPossessed(victim))
					if succeeds {
						require.Len(t, mock.CombatEvents, 1)
						assert.Equal(t, CombatEventPossessionRelease, mock.CombatEvents[0].Kind)
					}
				})
			}
		}
	}
}

func TestPossessionCloneCopiesProgressAndClearsIndependently(t *testing.T) {
	state, _, controller, _, source := setupPossessionEpisode(t)
	combat := state.CombatScene
	episode := combat.possessions[source]
	episode.turns = 8
	combat.possessions[source] = episode
	clone := combat.addPCCloneFromSource(source, 0, references.Position{X: 8, Y: 8}, NewCombatPCProvider(state.PartyState.GetCharacter(0)))
	require.NotZero(t, clone.Slot)
	copyEpisode := combat.possessions[clone]
	assert.NotEqual(t, episode.id, copyEpisode.id)
	assert.Equal(t, episode.controller, copyEpisode.controller)
	assert.Equal(t, 8, copyEpisode.turns)
	combat.dispatchCombatPossessionTurn(clone)
	state.completeCombatPossessionTurnWithRNG(combat.possessionTurn, &possessionRNG{t: t, draws: []possessionDraw{{10000, 9999}}})
	assert.Equal(t, 9, combat.possessions[clone].turns)
	assert.Equal(t, 8, combat.possessions[source].turns)
	state.PartyState.GetCharacter(1).Intelligence = 255
	require.True(t, state.CastCombatAnXenEx(1, clone))
	assert.NotContains(t, combat.possessions, clone)
	assert.Equal(t, episode, combat.possessions[source])
	state.releaseCombatController(controller.CombatantID(), CombatPossessionReleaseControllerDefeated)
	assert.Empty(t, combat.possessions)
}

func TestPossessionControllerAttributionStableOrderAndIdempotence(t *testing.T) {
	state, mock, controllerA, indexA, source := setupPossessionEpisode(t)
	combat := state.CombatScene
	clone := combat.addPCCloneFromSource(source, 0, references.Position{X: 8, Y: 8}, NewCombatPCProvider(state.PartyState.GetCharacter(0)))
	controllerB := NewCombatMonster(references.MonsterOrc, &(*state.GameReferences.EnemyReferences)[references.MonsterOrc], references.Position{X: 4, Y: 4}, newFixedRNG(1))
	combat.AddMonsterToQueue(controllerB)
	victimB := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 1}
	require.True(t, combat.acquireCombatPossession(victimB, controllerB.CombatantID()))
	controllerA.SetCurrentHP(0)
	state.finalizeCombatMonsterDeathWithPolicy(controllerA, indexA, nil, NoTargetPC, false, false)
	state.releaseCombatController(controllerA.CombatantID(), CombatPossessionReleaseControllerDefeated)
	assert.False(t, combat.IsCombatPCPossessed(source))
	assert.False(t, combat.IsCombatPCPossessed(clone))
	assert.True(t, combat.IsCombatPCPossessed(victimB))
	releases := []CombatEvent{}
	for _, event := range mock.CombatEvents {
		if event.Kind == CombatEventPossessionRelease {
			releases = append(releases, event)
		}
	}
	require.Len(t, releases, 2)
	assert.Equal(t, source, CombatTurnIdentity{IsPartyMember: releases[0].Target.IsPartyMember, EntityIndex: releases[0].Target.Index, Slot: releases[0].Target.Slot})
	assert.Equal(t, clone.Slot, releases[1].Target.Slot)
	assert.Zero(t, countCombatEvents(mock.CombatEvents, CombatEventRecovery))
}

func TestPossessionStaleCompletionCannotTickReplacementEpisode(t *testing.T) {
	state, _, controller, _, victim := setupPossessionEpisode(t)
	combat := state.CombatScene
	combat.dispatchCombatPossessionTurn(victim)
	stale := combat.possessionTurn
	combat.setCombatPCPossessed(victim, false)
	require.True(t, combat.acquireCombatPossession(victim, controller.CombatantID()))
	replacement := combat.possessions[victim]
	state.completeCombatPossessionTurnWithRNG(stale, &possessionRNG{t: t})
	assert.Equal(t, replacement, combat.possessions[victim])
	combat.dispatchCombatPossessionTurn(victim)
	state.completeCombatPossessionTurnWithRNG(stale, &possessionRNG{t: t})
	assert.Equal(t, replacement, combat.possessions[victim])
	state.completeCombatPossessionTurnWithRNG(combat.possessionTurn, &possessionRNG{t: t})
	assert.Equal(t, 1, combat.possessions[victim].turns)
}

func TestPossessionAcquisitionFailureAndRedundancyPreserveProgress(t *testing.T) {
	state, _, controller, controllerIndex, victim := setupPossessionEpisode(t)
	combat := state.CombatScene
	episode := combat.possessions[victim]
	episode.turns = 9
	combat.possessions[victim] = episode
	state.resolveMonsterPossession(controller, controllerIndex, victim, &possessionRNG{t: t})
	assert.Equal(t, episode, combat.possessions[victim])
	combat.setCombatPCPossessed(victim, false)
	state.PartyState.GetCharacter(0).Intelligence = 30
	state.resolveMonsterPossession(controller, controllerIndex, victim, &possessionRNG{t: t, draws: []possessionDraw{{61, 0}}})
	assert.NotContains(t, combat.possessions, victim)
	assert.False(t, combat.IsCombatPCPossessed(victim))
}

func TestPossessionLifecycleCleanup(t *testing.T) {
	for _, removal := range []string{"death", "departure", "clone", "replacement", "teardown", "emergency"} {
		t.Run(removal, func(t *testing.T) {
			state, _, _, _, source := setupPossessionEpisode(t)
			combat := state.CombatScene
			clone := combat.addPCCloneFromSource(source, 0, references.Position{X: 8, Y: 8}, NewCombatPCProvider(state.PartyState.GetCharacter(0)))
			combat.dispatchCombatPossessionTurn(source)
			ticket := combat.possessionTurn
			switch removal {
			case "death":
				state.PartyState.GetCharacter(0).SetHp(0)
				state.markCombatPCDead(0)
				assert.Empty(t, combat.possessions)
			case "departure":
				state.departPCFromCombat(source)
				assert.NotContains(t, combat.possessions, source)
				assert.Contains(t, combat.possessions, clone)
			case "clone":
				combat.removePCClone(clone)
				assert.NotContains(t, combat.possessions, clone)
				assert.Contains(t, combat.possessions, source)
			case "replacement":
				require.True(t, state.replaceCombatTargetWithRat(source, combat.GetCombatantPosition(source)))
				assert.NotContains(t, combat.possessions, source)
				assert.Contains(t, combat.possessions, clone)
			case "teardown":
				state.TransitionFromCombatMap()
				assert.Empty(t, combat.possessions)
			case "emergency":
				require.True(t, state.recoverOnePossessedPC())
				assert.NotContains(t, combat.possessions, source)
				assert.True(t, combat.IsCombatPCSleeping(source))
				assert.Contains(t, combat.possessions, clone)
			}
			if removal != "clone" {
				state.completeCombatPossessionTurnWithRNG(ticket, &possessionRNG{t: t})
				assert.NotContains(t, combat.possessions, source)
			}
		})
	}
}

// Unlike the older helper, this fixture schedules exactly one full identity,
// including Slot, and then a friendly source so an immediate scan terminates.
func forcePossessionActor(combat *CombatState, actor CombatTurnIdentity) {
	for _, entry := range combat.TurnQueue.entries {
		entry.turnCountdown = turnCountdownBase
		if turnQueueIdentity(entry) == actor {
			entry.turnCountdown = 1
		}
	}
	combat.TurnQueue.GetEntry(1, true).turnCountdown = 2
	combat.TurnQueue.currentIndex = 0
}

func TestPossessionImmediatePresentedBoundaryParity(t *testing.T) {
	for _, condition := range []string{"awake", "sleep", "captivity", "time-stop", "quickness"} {
		for _, cloneActor := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/clone%t", condition, cloneActor), func(t *testing.T) {
				setup := func() (*GameState, *MockSystemCallbacks, CombatTurnIdentity) {
					state, mock, _, _, actor := setupPossessionEpisode(t)
					if cloneActor {
						actor = state.CombatScene.addPCCloneFromSource(actor, 0, references.Position{X: 8, Y: 8}, NewCombatPCProvider(state.PartyState.GetCharacter(0)))
					}
					episode := state.CombatScene.possessions[actor]
					episode.turns = 6
					state.CombatScene.possessions[actor] = episode
					switch condition {
					case "sleep":
						state.setCombatPCSleeping(actor, true)
					case "captivity":
						state.CombatScene.setCombatPCUnderground(actor, true)
					case "time-stop":
						state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectTimeStop, TurnsRemaining: 20}
					case "quickness":
						state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectQuickness, TurnsRemaining: 20}
					}
					forcePossessionActor(state.CombatScene, actor)
					state.SetRandomSeed(42)
					if condition == "quickness" {
						seed := findCombatWakeSeed(t, state, 1, func(roll int) bool { return roll == 0 })
						state.SetRandomSeed(seed)
					}
					mock.Reset()
					return state, mock, actor
				}
				immediate, immediateMock, actor := setup()
				presented, presentedMock, _ := setup()
				immediate.BeginCombatTurns(CombatTurnsImmediate)
				presented.BeginCombatTurns(CombatTurnsPresented)
				step, ok := presented.ReviewScheduledCombatStep()
				require.True(t, ok)
				assert.Equal(t, actor, step.Actor)
				calls := presented.GetRNG().Calls()
				for range 5 {
					presented.AdvanceCombatTurn()
					presented.ProjectCombatActor(actor, 0)
					presented.CombatScene.TurnQueue.FillForecast(make([]CombatTurnForecastEntry, 8))
				}
				assert.Equal(t, calls, presented.GetRNG().Calls())
				assert.Equal(t, 6, presented.CombatScene.possessions[actor].turns)
				_, err := presented.ResolveScheduledCombatStep(step.Token)
				require.NoError(t, err)
				assert.Equal(t, 6, presented.CombatScene.possessions[actor].turns)
				require.NoError(t, presented.CompleteScheduledCombatStep(step.Token))
				assert.ErrorIs(t, presented.CompleteScheduledCombatStep(step.Token), ErrCombatPresentationDuplicate)
				assert.Equal(t, immediate.CombatScene.possessions, presented.CombatScene.possessions)
				assert.Equal(t, immediateMock.CombatEvents, presentedMock.CombatEvents)
				assert.Equal(t, immediate.CombatScene.PartyPossessed, presented.CombatScene.PartyPossessed)
				assert.Equal(t, immediate.CombatScene.GetCurrentTurnIdentity(), presented.CombatScene.GetCurrentTurnIdentity())
				assert.Equal(t, immediate.GetRNG().Calls(), presented.GetRNG().Calls())
				assert.Equal(t, immediate.RandomIntInRange(0, 9999), presented.RandomIntInRange(0, 9999))
			})
		}
	}
}

func TestPossessionDaemonRetirementAndAdministrativeRemovalAreNotDefeat(t *testing.T) {
	state, mock, _, _, victim := setupPossessionEpisode(t)
	combat := state.CombatScene
	combat.setCombatPCPossessed(victim, false)
	daemon := NewCombatMonster(references.MonsterDevil, &(*state.GameReferences.EnemyReferences)[references.MonsterDevil], references.Position{X: 3, Y: 3}, newFixedRNG(1))
	index := combat.AddMonsterToQueue(daemon)
	state.resolveMonsterPossession(daemon, index, victim, &possessionRNG{t: t, draws: []possessionDraw{{61, 60}}})
	require.False(t, daemon.IsActive())
	episode := combat.possessions[victim]
	assert.Equal(t, daemon.CombatantID(), episode.controller)
	assert.Equal(t, []CombatEventKind{CombatEventPossession, CombatEventRetired}, combatEventKinds(mock.CombatEvents))
	_, err := state.ClearDebugCombatMonsters(DebugCombatBatchRequest{EncounterID: combat.EncounterID, Expected: state.debugCombatSnapshot()})
	require.NoError(t, err)
	assert.Equal(t, episode, combat.possessions[victim])
	fresh := NewCombatMonster(references.MonsterOrc, &(*state.GameReferences.EnemyReferences)[references.MonsterOrc], references.Position{X: 3, Y: 3}, newFixedRNG(1))
	freshIndex := combat.AddMonsterToQueue(fresh)
	assert.NotEqual(t, episode.controller, fresh.CombatantID())
	fresh.SetCurrentHP(0)
	state.finalizeCombatMonsterDeathWithPolicy(fresh, freshIndex, nil, NoTargetPC, false, false)
	assert.Equal(t, episode, combat.possessions[victim], "reused manager indices cannot redirect attribution")
}

func TestPossessionMonsterCloneDoesNotInheritVictims(t *testing.T) {
	state, _, original, originalIndex, victim := setupPossessionEpisode(t)
	before := state.CombatScene.possessions[victim]
	require.True(t, state.castCombatInQuasXenAt(1, CombatTurnIdentity{EntityIndex: originalIndex}, references.Position{X: 4, Y: 4}))
	var cloned *CombatMonster
	cloneIndex := -1
	state.CombatScene.MonsterManager.ForEachActive(func(index int, monster *CombatMonster) {
		if monster.CombatantID() != original.CombatantID() {
			cloned, cloneIndex = monster, index
		}
	})
	require.NotNil(t, cloned)
	assert.NotEqual(t, original.CombatantID(), cloned.CombatantID())
	cloned.SetCurrentHP(0)
	state.finalizeCombatMonsterDeathWithPolicy(cloned, cloneIndex, nil, NoTargetPC, false, false)
	assert.Equal(t, before, state.CombatScene.possessions[victim])
}

func TestPossessionControllerRatReplacementAndControlledEscape(t *testing.T) {
	for _, outcome := range []string{"rat", "controlled-escape", "debug-kill"} {
		t.Run(outcome, func(t *testing.T) {
			state, mock, controller, index, victim := setupPossessionEpisode(t)
			experience := state.PartyState.GetCharacter(0).Exp
			objects := state.CombatScene.ObjectCount()
			want := CombatPossessionReleaseControllerReplaced
			switch outcome {
			case "rat":
				// Invoke the successful spell mutation; rat construction consumes its
				// normal DEX draw, while release adds no roll or death rewards.
				require.True(t, state.replaceCombatTargetWithRat(CombatTurnIdentity{EntityIndex: index}, controller.Position()))
			case "controlled-escape":
				controller.SetLoyalty(CombatMonsterLoyaltyParty)
				controller.SetPosition(references.Position{X: 0, Y: 5})
				assert.Contains(t, state.CombatScene.possessions, victim, "allegiance change is not release")
				require.Equal(t, CombatMoveFled, state.moveControlledCombatMonster(controller, index, references.Left))
				want = CombatPossessionReleaseControllerEscaped
			case "debug-kill":
				_, err := state.KillDebugCombatMonster(DebugCombatKillRequest{EncounterID: state.CombatScene.EncounterID, CombatantID: controller.CombatantID(), ExpectedHP: controller.CurrentHP()})
				require.NoError(t, err)
				want = CombatPossessionReleaseControllerDefeated
			}
			assert.NotContains(t, state.CombatScene.possessions, victim)
			assert.False(t, state.CombatScene.IsCombatPCPossessed(victim))
			assert.False(t, state.CombatScene.IsCombatPCSleeping(victim))
			assert.Equal(t, experience, state.PartyState.GetCharacter(0).Exp)
			assert.Equal(t, objects, state.CombatScene.ObjectCount())
			require.Equal(t, 1, countCombatEvents(mock.CombatEvents, CombatEventPossessionRelease))
			for _, event := range mock.CombatEvents {
				if event.Kind == CombatEventPossessionRelease {
					assert.Equal(t, want, event.PossessionReleaseReason)
				}
			}
			assert.Zero(t, countCombatEvents(mock.CombatEvents, CombatEventRecovery))
		})
	}
}

func TestPossessionFailedCurePreservesAndSuccessfulCureStartsFreshEpisode(t *testing.T) {
	state, _, controller, _, victim := setupPossessionEpisode(t)
	episode := state.CombatScene.possessions[victim]
	episode.turns = 8
	state.CombatScene.possessions[victim] = episode
	state.PartyState.GetCharacter(0).Intelligence = 255
	state.PartyState.GetCharacter(1).Intelligence = 1
	require.False(t, state.CastCombatAnXenEx(1, victim))
	assert.Equal(t, episode, state.CombatScene.possessions[victim])
	state.PartyState.GetCharacter(1).Intelligence = 255
	state.PartyState.GetCharacter(0).Intelligence = 1
	require.True(t, state.CastCombatAnXenEx(1, victim))
	assert.NotContains(t, state.CombatScene.possessions, victim)
	require.True(t, state.CombatScene.acquireCombatPossession(victim, controller.CombatantID()))
	assert.NotEqual(t, episode.id, state.CombatScene.possessions[victim].id)
	assert.Zero(t, state.CombatScene.possessions[victim].turns)
}

func TestPossessionRawChaosAndIntrinsicHostilityDoNotInventEpisodes(t *testing.T) {
	state, _, controller, _, victim := setupPossessionEpisode(t)
	state.releaseCombatController(controller.CombatantID(), CombatPossessionReleaseControllerDefeated)
	state.PartyState.GetCharacter(0).SetEquippedItem(references.SlotWeapon, references.ChaosSword)
	require.True(t, state.activateChaosSwordControl(victim))
	_, _, timed := state.CombatScene.CombatPossessionProgress(victim)
	assert.False(t, timed)
	state.CombatScene.dispatchCombatPossessionTurn(victim)
	state.completeCombatPossessionTurnWithRNG(state.CombatScene.possessionTurn, &possessionRNG{t: t})
	assert.True(t, state.CombatScene.IsCombatPCPossessed(victim))
	state.CombatScene.setCombatPCPossessed(victim, false)
	// The Avatar is deliberately exempt from intrinsic Saduj hostility, even
	// when named Saduj. Use the active non-Avatar fixture for this independent rule.
	saduj := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 1}
	character := state.PartyState.GetCharacter(saduj.EntityIndex)
	clear(character.Name[:])
	copy(character.Name[:], "Saduj")
	character.SetEquippedItem(references.SlotWeapon, references.ChaosSword)
	require.True(t, state.isIntrinsicHostilePC(saduj))
	require.True(t, state.CombatScene.acquireCombatPossession(saduj, controller.CombatantID()))
	state.releaseCombatController(controller.CombatantID(), CombatPossessionReleaseControllerDefeated)
	assert.False(t, state.CombatScene.IsCombatPCPossessed(saduj))
	assert.Equal(t, CombatMonsterLoyaltyMonster, state.combatIdentityLoyalty(saduj))
	assert.Equal(t, references.ChaosSword, character.GetEquippedItem(references.SlotWeapon))
	assert.Empty(t, state.CombatScene.possessions)
}

func TestPossessionTwoRemainingVictimsEmergencyReturnsControl(t *testing.T) {
	state, mock, controller, controllerIndex, victim := setupPossessionEpisode(t)
	friend := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 1}
	require.True(t, state.CombatScene.acquireCombatPossession(friend, controller.CombatantID()))
	controller.SetActive(false) // Administrative removal is not controller defeat.
	state.CombatScene.TurnQueue.MarkFled(controllerIndex, false)
	require.Equal(t, CombatNotEnded, state.checkCombatEndConditions())
	assert.NotContains(t, state.CombatScene.possessions, victim)
	assert.Contains(t, state.CombatScene.possessions, friend)
	assert.True(t, state.CombatScene.IsCombatPCSleeping(victim))
	assert.Equal(t, 1, countCombatEvents(mock.CombatEvents, CombatEventRecovery))
	// Once the retained emergency sleeper wakes, the ordinary scheduler grants
	// input; no phantom presented completion or lost-control state remains.
	state.setCombatPCSleeping(victim, false)
	for _, entry := range state.CombatScene.TurnQueue.entries {
		entry.turnCountdown = turnCountdownBase
		if turnQueueIdentity(entry) == victim {
			entry.turnCountdown = 1
		}
	}
	state.CombatScene.TurnQueue.currentIndex = 0
	state.BeginCombatTurns(CombatTurnsPresented)
	assert.Equal(t, victim, state.CombatScene.GetCurrentTurnIdentity())
	assert.False(t, state.CombatScene.HasScheduledCombatStep())
	assert.Equal(t, 0, state.GetActivePlayerIndex())
}

func TestPossessionTwelfthSuppressedBoundaryReleasesAfterStandingHazards(t *testing.T) {
	for _, lethal := range []bool{false, true} {
		t.Run(fmt.Sprintf("lethal%t", lethal), func(t *testing.T) {
			state, mock, _, _, actor := setupPossessionEpisode(t)
			combat := state.CombatScene
			episode := combat.possessions[actor]
			episode.turns = 11
			combat.possessions[actor] = episode
			state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectTimeStop, TurnsRemaining: 20}
			if lethal {
				state.PartyState.GetCharacter(0).SetHp(1)
			}
			require.True(t, combat.AddField(NewCombatField(CombatFieldFire, combat.GetCombatantPosition(actor))))
			forcePossessionActor(combat, actor)
			seed := findCombatWakeSeed(t, state, 10, func(roll int) bool { return roll > 0 })
			state.SetRandomSeed(seed)
			state.BeginCombatTurns(CombatTurnsPresented)
			step, ok := state.ReviewScheduledCombatStep()
			require.True(t, ok)
			_, err := state.ResolveScheduledCombatStep(step.Token)
			require.NoError(t, err)
			require.Equal(t, 1, countCombatEvents(mock.CombatEvents, CombatEventFieldTriggered))
			assert.Zero(t, countCombatEvents(mock.CombatEvents, CombatEventPossessionRelease), "presentation resolution is not completed boundary")
			calls := state.GetRNG().Calls()
			require.NoError(t, state.CompleteScheduledCombatStep(step.Token))
			assert.Equal(t, calls, state.GetRNG().Calls(), "death and guaranteed turn twelve both avoid recovery RNG")
			assert.NotContains(t, combat.possessions, actor)
			if lethal {
				assert.Zero(t, countCombatEvents(mock.CombatEvents, CombatEventPossessionRelease))
				assert.True(t, state.PartyState.GetCharacter(0).IsDead())
			} else {
				require.Equal(t, 1, countCombatEvents(mock.CombatEvents, CombatEventPossessionRelease))
				assert.Equal(t, CombatEventFieldTriggered, mock.CombatEvents[0].Kind)
				assert.Equal(t, CombatEventPossessionRelease, mock.CombatEvents[1].Kind)
				assert.Equal(t, 12, mock.CombatEvents[1].PossessionTurns)
			}
		})
	}
}

func TestPossessionPresentedOldEpisodeCompletionDoesNotTickReacquisition(t *testing.T) {
	state, _, controller, _, actor := setupPossessionEpisode(t)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectTimeStop, TurnsRemaining: 20}
	forcePossessionActor(state.CombatScene, actor)
	state.BeginCombatTurns(CombatTurnsPresented)
	step, ok := state.ReviewScheduledCombatStep()
	require.True(t, ok)
	_, err := state.ResolveScheduledCombatStep(step.Token)
	require.NoError(t, err)
	state.CombatScene.setCombatPCPossessed(actor, false)
	require.True(t, state.CombatScene.acquireCombatPossession(actor, controller.CombatantID()))
	episode := state.CombatScene.possessions[actor]
	require.NoError(t, state.CompleteScheduledCombatStep(step.Token))
	assert.Equal(t, episode, state.CombatScene.possessions[actor])
	assert.ErrorIs(t, state.CompleteScheduledCombatStep(step.Token), ErrCombatPresentationDuplicate)
	assert.Equal(t, episode, state.CombatScene.possessions[actor])
}

func TestPossessionAppearanceAndControllerNonRemovalPreserveEpisode(t *testing.T) {
	state, _, controller, _, actor := setupPossessionEpisode(t)
	combat := state.CombatScene
	before := combat.possessions[actor]
	combat.CurrentTurnIdentity = actor
	combat.currentTurnIdentitySet = true
	require.True(t, state.TransformActivePartyMemberIntoRat(actor.EntityIndex))
	controller.SetInvisible(true)
	controller.SetPosition(references.Position{X: 5, Y: 5})
	controller.SetFleeing(true)
	controller.SetLoyalty(CombatMonsterLoyaltyParty)
	assert.Equal(t, before, combat.possessions[actor])
	assert.True(t, combat.IsCombatPCPossessed(actor))
}

func TestPossessionOldEncounterTicketIsRejected(t *testing.T) {
	state, _, _, _, actor := setupPossessionEpisode(t)
	state.CombatScene.dispatchCombatPossessionTurn(actor)
	stale := state.CombatScene.possessionTurn
	state.TransitionFromCombatMap()
	require.NoError(t, state.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth))
	combat := state.CombatScene
	controller := NewCombatMonster(references.MonsterOrc, &(*state.GameReferences.EnemyReferences)[references.MonsterOrc], references.Position{X: 0, Y: 0}, newFixedRNG(1))
	combat.AddMonsterToQueue(controller)
	require.True(t, combat.acquireCombatPossession(actor, controller.CombatantID()))
	combat.dispatchCombatPossessionTurn(actor)
	before := combat.possessions[actor]
	state.completeCombatPossessionTurnWithRNG(stale, &possessionRNG{t: t})
	assert.Equal(t, before, combat.possessions[actor])
}

func TestPossessionProductionSchedulerCountsTwelveSuppressedOwnTurns(t *testing.T) {
	state, mock, _, _, actor := setupPossessionEpisode(t)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectTimeStop, TurnsRemaining: 20}
	seed := findCombatWakeSeed(t, state, 9999, func(roll int) bool { return roll >= 9000 })
	for turn := 1; turn <= 12; turn++ {
		forcePossessionActor(state.CombatScene, actor)
		state.SetRandomSeed(seed)
		beforeCalls := state.GetRNG().Calls()
		state.AdvanceCombatTurn()
		expectedDraws := uint64(0)
		if turn >= 4 && turn <= 11 {
			expectedDraws = 1
		}
		assert.EqualValues(t, expectedDraws, state.GetRNG().Calls()-beforeCalls, "turn %d", turn)
		assert.Equal(t, CombatTurnIdentity{IsPartyMember: true, EntityIndex: 1}, state.CombatScene.GetCurrentTurnIdentity())
		_, progress, active := state.CombatScene.CombatPossessionProgress(actor)
		assert.Equal(t, turn < 12, active)
		if active {
			assert.Equal(t, turn, progress)
		}
	}
	assert.False(t, state.CombatScene.IsCombatPCPossessed(actor))
	assert.Equal(t, 3, countCombatEvents(mock.CombatEvents, CombatEventPossessionResistance))
	assert.Equal(t, 1, countCombatEvents(mock.CombatEvents, CombatEventPossessionRelease))
	assert.Zero(t, countCombatEvents(mock.CombatEvents, CombatEventRecovery))
}
