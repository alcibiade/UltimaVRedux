package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestSmallMapShadowlordEncounterPreservesIdentityAndReclaimsSceptre(t *testing.T) {
	lords := []references.Shadowlord{references.Faulinei, references.Astaroth, references.Nosfentor}
	sourceKinds := []SmallMapEncounterSourceKind{
		SmallMapEncounterAutomaticShadowlord,
		SmallMapEncounterYellShadowlord,
	}

	for _, lord := range lords {
		for _, sourceKind := range sourceKinds {
			for _, ownsSceptre := range []bool{false, true} {
				name := lord.String() + "/" + smallMapShadowlordSourceName(sourceKind)
				if ownsSceptre {
					name += "/with_Sceptre"
				} else {
					name += "/without_Sceptre"
				}
				t.Run(name, func(t *testing.T) {
					state, callbacks, source := newSmallMapShadowlordEncounterFixture(
						t,
						lord,
						sourceKind,
						references.Position{X: 10, Y: 10},
						references.Position{X: 10, Y: 9},
					)
					if ownsSceptre {
						state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
					} else {
						state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 0)
					}
					shard := references.QuestItem(int(references.ShardOfFalsehood) + int(lord))
					state.PartyState.Inventory.QuestItems.Set(shard, 1)

					result, err := state.StartSmallMapCombatEncounter(
						source,
						SmallMapEncounterShadowlordAdjacency,
					)

					require.NoError(t, err)
					require.NotNil(t, result)
					assert.Equal(t, 0x0a, result.MapIndex)
					assert.Equal(t, 1, result.PrimaryCount)
					assert.Zero(t, result.FriendCount)
					require.NotNil(t, state.CombatScene)
					assert.Equal(t, 1,
						state.CombatScene.MonsterManager.GetMonsterTypeCount(references.MonsterShadowlord))
					require.NotNil(t, state.CombatScene.SmallMapOrigin)
					assert.Same(t, source, state.CombatScene.SmallMapOrigin.SourceEnemy)
					assert.Equal(t, lord, state.CombatScene.SmallMapOrigin.Shadowlord)
					assert.Equal(t, sourceKind, state.CombatScene.SmallMapOrigin.SourceKind)
					assert.Nil(t, state.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(
						references.Position{X: 10, Y: 9}, 0,
					), "physical source must be gone before the first tactical turn")
					assert.Equal(t, uint16(0), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
					assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(shard))
					assert.True(t, state.ShadowlordState.IsAlive(lord))

					if ownsSceptre {
						assert.Equal(t, 1, countStrings(callbacks.Messages, "The Sceptre is reclaimed!"))
						assert.Equal(t, []SoundEffect{SoundShardPulse}, callbacks.SoundEffectsPlayed)
						assert.True(t, state.CombatScene.SmallMapOrigin.Consequences.SceptreReclaimed)
					} else {
						assert.NotContains(t, callbacks.Messages, "The Sceptre is reclaimed!")
						assert.Empty(t, callbacks.SoundEffectsPlayed)
						assert.False(t, state.CombatScene.SmallMapOrigin.Consequences.SceptreReclaimed)
					}
				})
			}
		}
	}
}

func TestSmallMapShadowlordEncounterConclusionsKeepProgressionAndSceptreLoss(t *testing.T) {
	conclusions := []struct {
		name           string
		reason         CombatEndReason
		totalPartyKill bool
	}{
		{name: "victory", reason: CombatVictory},
		{name: "defeat_or_flight", reason: CombatDefeat},
		{name: "evacuated", reason: CombatEvacuated},
		{name: "total_party_kill", reason: CombatDefeat, totalPartyKill: true},
	}
	sourceKinds := []SmallMapEncounterSourceKind{
		SmallMapEncounterAutomaticShadowlord,
		SmallMapEncounterYellShadowlord,
	}

	for _, sourceKind := range sourceKinds {
		for _, conclusion := range conclusions {
			t.Run(smallMapShadowlordSourceName(sourceKind)+"/"+conclusion.name, func(t *testing.T) {
				state, callbacks, source := newSmallMapShadowlordEncounterFixture(
					t,
					references.Astaroth,
					sourceKind,
					references.Position{X: 10, Y: 10},
					references.Position{X: 10, Y: 9},
				)
				state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
				state.PartyState.Inventory.QuestItems.Set(references.ShardOfHatred, 1)
				deadBefore := state.PartyState.IsNPCDead(
					references.Stonegate,
					shadowlordDeadNPCIndex(references.Astaroth),
				)

				_, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterShadowlordAdjacency)
				require.NoError(t, err)
				if conclusion.totalPartyKill {
					state.PartyState.ForEachCharacter(func(_ int, character *party_state.PlayerCharacter) {
						character.SetHp(0)
					})
				}
				state.EndCombat(conclusion.reason)

				assert.Nil(t, state.CombatScene)
				assert.Equal(t, uint16(0), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
				assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.ShardOfHatred))
				assert.True(t, state.ShadowlordState.IsAlive(references.Astaroth))
				assert.Equal(t, deadBefore, state.PartyState.IsNPCDead(
					references.Stonegate,
					shadowlordDeadNPCIndex(references.Astaroth),
				))
				assert.Empty(t, callbacks.TimeAdvanced(),
					"automatic adjacency and Yell handoffs do not own a second source-map turn")
				if sourceKind == SmallMapEncounterAutomaticShadowlord {
					assert.Equal(t, references.Astaroth, state.ShadowlordState.AutomaticPresence)
					assert.Equal(t, 1, countMatchingShadowlords(
						state.CurrentNPCAIController.GetNpcs(),
						references.Astaroth,
						references.MainFloor,
					))
					assert.NotNil(t, matchingShadowlordAt(
						state.CurrentNPCAIController.GetNpcs(),
						GetShadowlordSpawnPosition(references.Britain),
						references.MainFloor,
						references.Astaroth,
					))
				} else {
					assert.False(t, state.ShadowlordState.Summoned)
					assert.Equal(t, references.NoShadowlord, state.ShadowlordState.LastCalled)
					assert.Zero(t, countShadowlordNPCs(state.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)))
				}
			})
		}
	}
}

func TestSmallMapShadowlordActionSixRangeAndDelayedHandoff(t *testing.T) {
	for _, test := range []struct {
		name            string
		distance        int
		expectPending   bool
		expectPositionY references.Coordinate
	}{
		{name: "adjacent_hands_off_before_movement", distance: 1, expectPending: true, expectPositionY: 9},
		{name: "distance_two_approaches", distance: 2, expectPositionY: 9},
		{name: "distance_three_approaches", distance: 3, expectPositionY: 8},
		{name: "distance_four_does_not_approach", distance: 4, expectPositionY: 6},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, _, source := newSmallMapShadowlordEncounterFixture(
				t,
				references.Faulinei,
				SmallMapEncounterAutomaticShadowlord,
				references.Position{X: 10, Y: 10},
				references.Position{X: 10, Y: references.Coordinate(10 - test.distance)},
			)

			state.smallMapProcessNPCs()

			assert.Equal(t, test.expectPending, state.pendingSmallMapEncounter != nil)
			assert.Nil(t, state.CombatScene, "the NPC phase queues through the shared handoff boundary")
			assert.Equal(t, test.expectPositionY, source.Pos().Y)
			if test.expectPending {
				pending := state.pendingSmallMapEncounter
				state.smallMapProcessNPCs()
				assert.Same(t, pending, state.pendingSmallMapEncounter,
					"an already-owned handoff must not duplicate the encounter")
			}
		})
	}
}

func TestAutomaticShadowlordEncounterSurvivesMidnightAssignmentMove(t *testing.T) {
	state, _, source := newSmallMapShadowlordEncounterFixture(
		t,
		references.Faulinei,
		SmallMapEncounterAutomaticShadowlord,
		references.Position{X: 10, Y: 10},
		references.Position{X: 10, Y: 9},
	)
	state.ShadowlordState.TownAssignments[references.Faulinei] = references.Yew

	_, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterShadowlordAdjacency)

	require.NoError(t, err)
	require.NotNil(t, state.CombatScene)
	assert.Equal(t, SmallMapEncounterAutomaticShadowlord, state.CombatScene.SmallMapOrigin.SourceKind)
}

func TestAutomaticShadowlordReturnReevaluatesCurrentAssignmentAndMapEffects(t *testing.T) {
	for _, test := range []struct {
		name     string
		assigned references.Shadowlord
		change   bool
	}{
		{name: "assignment unchanged", assigned: references.Faulinei},
		{name: "assignment changed during combat", assigned: references.Astaroth, change: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, _, source := newSmallMapShadowlordEncounterFixture(
				t,
				references.Faulinei,
				SmallMapEncounterAutomaticShadowlord,
				references.Position{X: 10, Y: 10},
				references.Position{X: 10, Y: 9},
			)
			state.DateTime.Day = 13
			state.ApplyShadowlordMapEffects(references.Faulinei)
			beforeTerrain := currentSmallMapTerrain(state)
			_, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterShadowlordAdjacency)
			require.NoError(t, err)
			if test.change {
				state.ShadowlordState.TownAssignments[references.Faulinei] = references.Yew
				state.ShadowlordState.TownAssignments[references.Astaroth] = references.Britain
			}
			rngCallsBefore := state.GetRNG().Calls()

			state.EndCombat(CombatVictory)

			assert.Equal(t, test.assigned, state.ShadowlordState.AutomaticPresence)
			assert.Equal(t, 1, countShadowlordNPCs(state.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)))
			assert.Equal(t, 1, countMatchingShadowlords(
				state.CurrentNPCAIController.GetNpcs(),
				test.assigned,
				references.MainFloor,
			))
			assert.NotNil(t, matchingShadowlordAt(
				state.CurrentNPCAIController.GetNpcs(),
				GetShadowlordSpawnPosition(references.Britain),
				references.MainFloor,
				test.assigned,
			))
			assert.Equal(t, rngCallsBefore, state.GetRNG().Calls(),
				"combat-return map effects must not consume shared gameplay RNG")
			assert.Equal(t, beforeTerrain, currentSmallMapTerrain(state),
				"combat return must rebuild source terrain before the same day's effect pass")
		})
	}
}

func TestAutomaticShadowlordReturnClearsPresenceWhenAssignmentAbsentOrSpawnFails(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(*GameState)
	}{
		{
			name: "assignment absent",
			setup: func(state *GameState) {
				state.ShadowlordState.TownAssignments[references.Faulinei] = references.Yew
			},
		},
		{
			name:  "spawn fails",
			setup: removeShadowlordEnemyReference,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, _, source := newSmallMapShadowlordEncounterFixture(
				t,
				references.Faulinei,
				SmallMapEncounterAutomaticShadowlord,
				references.Position{X: 10, Y: 10},
				references.Position{X: 10, Y: 9},
			)
			_, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterShadowlordAdjacency)
			require.NoError(t, err)
			test.setup(state)

			state.EndCombat(CombatVictory)

			assert.False(t, state.ShadowlordState.HasAutomaticPresence())
			assert.Zero(t, countShadowlordNPCs(state.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)))
		})
	}
}

func TestQueuedAdjacentShadowlordProcessesEveryOtherNPCExactlyOnce(t *testing.T) {
	state, _, source := newSmallMapShadowlordEncounterFixture(
		t,
		references.Faulinei,
		SmallMapEncounterAutomaticShadowlord,
		references.Position{X: 10, Y: 10},
		references.Position{X: 10, Y: 9},
	)
	enemyReference := state.enemyReference(references.MonsterShadowlord)
	require.NotNil(t, enemyReference)
	for _, position := range []references.Position{{X: 7, Y: 10}, {X: 13, Y: 10}} {
		enemy := map_units.NewEnemyNPC(*enemyReference, references.DynamicSpawnIndex)
		enemy.SetPos(position)
		enemy.SetFloor(references.MainFloor)
		enemy.SetVisible(true)
		require.True(t, state.CurrentNPCAIController.GetNpcs().AddEnemy(enemy))
	}
	left := state.CurrentNPCAIController.GetNpcs().GetVisibleEnemyAtPositionAndFloorOrNil(
		references.Position{X: 7, Y: 10}, references.MainFloor,
	)
	right := state.CurrentNPCAIController.GetNpcs().GetVisibleEnemyAtPositionAndFloorOrNil(
		references.Position{X: 13, Y: 10}, references.MainFloor,
	)
	require.NotNil(t, left)
	require.NotNil(t, right)

	state.smallMapProcessNPCs()

	require.NotNil(t, state.pendingSmallMapEncounter)
	assert.Same(t, source, state.pendingSmallMapEncounter.SourceEnemy)
	assert.Equal(t, references.Position{X: 10, Y: 9}, source.Pos())
	assert.Equal(t, references.Position{X: 8, Y: 10}, left.Pos())
	assert.Equal(t, references.Position{X: 12, Y: 10}, right.Pos())
}

func TestYellShadowlordMovesAdjacentBeforeLaterCombatHandoff(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Lycaeum).
		WithFloor(0).
		WithPlayerAt(10, 10).
		WithSystemCallbacks().
		Build()
	state.ShadowlordState.Reset()
	state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
	setSmallMapShadowlordTestPath(state)

	require.True(t, state.ActionYellSmallMap("FAULINEI"))
	source := state.CurrentNPCAIController.GetNpcs().GetVisibleEnemyAtPositionAndFloorOrNil(
		references.Position{X: 10, Y: 8}, 0,
	)
	require.NotNil(t, source)
	assert.Nil(t, state.pendingSmallMapEncounter)
	assert.Nil(t, state.CombatScene)
	assert.Empty(t, callbacks.TimeAdvanced())

	state.smallMapProcessNPCs()
	assert.Equal(t, references.Position{X: 10, Y: 9}, source.Pos())
	assert.Nil(t, state.pendingSmallMapEncounter,
		"moving adjacent must not begin combat during the same NPC phase")

	state.smallMapProcessNPCs()
	require.NotNil(t, state.pendingSmallMapEncounter)
	assert.Equal(t, SmallMapEncounterYellShadowlord, state.pendingSmallMapEncounter.SourceKind)
	require.NoError(t, state.ResolvePendingOverworldContact())
	assert.Equal(t, 0x0a, state.CombatScene.CombatMapRef.MapIndex)
}

func TestActionAttackSmallMapStartsEligibleShadowlordEncounterOnce(t *testing.T) {
	for _, sourceKind := range []SmallMapEncounterSourceKind{
		SmallMapEncounterAutomaticShadowlord,
		SmallMapEncounterYellShadowlord,
	} {
		t.Run(smallMapShadowlordSourceName(sourceKind), func(t *testing.T) {
			state, callbacks, _ := newSmallMapShadowlordEncounterFixture(
				t,
				references.Nosfentor,
				sourceKind,
				references.Position{X: 10, Y: 10},
				references.Position{X: 10, Y: 9},
			)
			state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
			beforeTurn := state.DateTime
			state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 2}

			assert.True(t, state.ActionAttackSmallMap(references.Up))
			require.NotNil(t, state.CombatScene)
			assert.Equal(t, SmallMapEncounterAttack, state.CombatScene.SmallMapOrigin.Cause)
			assert.Equal(t, references.Nosfentor, state.CombatScene.SmallMapOrigin.Shadowlord)
			assert.Equal(t, 1, countStrings(callbacks.Messages, "The Sceptre is reclaimed!"))
			assert.Equal(t, []SoundEffect{SoundShardPulse}, callbacks.SoundEffectsPlayed)
			assert.Empty(t, callbacks.TimeAdvanced(),
				"Attack completes its source-map turn only after combat")

			state.EndCombat(CombatDefeat)
			expectedDate := beforeTurn
			expectedDate.Advance(DefaultSmallMapMinutesPerTurn)
			assert.Equal(t, expectedDate, state.DateTime)
			assert.Equal(t, 1, state.ActiveDurationEffect.TurnsRemaining)
		})
	}
}

func TestExplicitAttackAutomaticShadowlordAtCanonicalSpawnStartsExactlyOneReturnEncounter(t *testing.T) {
	spawn := GetShadowlordSpawnPosition(references.Britain)
	state, callbacks, firstSource := newSmallMapShadowlordEncounterFixture(
		t,
		references.Faulinei,
		SmallMapEncounterAutomaticShadowlord,
		spawn.Down(),
		spawn,
	)
	beforeTurn := state.DateTime
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 2}

	require.True(t, state.ActionAttackSmallMap(references.Up))
	require.NotNil(t, state.CombatScene)
	assert.Equal(t, SmallMapEncounterAttack, state.CombatScene.SmallMapOrigin.Cause)
	assert.Same(t, firstSource, state.CombatScene.SmallMapOrigin.SourceEnemy)

	state.EndCombat(CombatVictory)

	// Original town attack resolution completes its source action after gocombat
	// reloads the town. That action's one NPC phase pass sees the rematerialized
	// canonical Shadowlord adjacent and enters combat once more.
	// See ref:CRIME-0004.
	require.NotNil(t, state.CombatScene)
	secondOrigin := state.CombatScene.SmallMapOrigin
	require.NotNil(t, secondOrigin)
	assert.Equal(t, SmallMapEncounterShadowlordAdjacency, secondOrigin.Cause)
	assert.Equal(t, SmallMapEncounterAutomaticShadowlord, secondOrigin.SourceKind)
	assert.Equal(t, spawn, secondOrigin.Position)
	assert.NotSame(t, firstSource, secondOrigin.SourceEnemy)
	assert.False(t, secondOrigin.Consequences.CompleteSourceTurn)
	expectedDate := beforeTurn
	expectedDate.Advance(DefaultSmallMapMinutesPerTurn)
	assert.Equal(t, expectedDate, state.DateTime)
	assert.Equal(t, 1, state.ActiveDurationEffect.TurnsRemaining)
	assert.False(t, state.HasPendingOverworldContact())

	state.EndCombat(CombatVictory)

	assert.Nil(t, state.CombatScene, "the nested return does not run npcdriver again")
	assert.False(t, state.HasPendingOverworldContact())
	assert.Equal(t, expectedDate, state.DateTime)
	assert.Equal(t, 1, state.ActiveDurationEffect.TurnsRemaining)
	assert.NotNil(t, matchingShadowlordAt(
		state.CurrentNPCAIController.GetNpcs(),
		spawn,
		references.MainFloor,
		references.Faulinei,
	))
	require.Len(t, callbacks.CombatConclusions, 2)
}

func TestSmallMapShadowlordEncounterRejectsInvalidTargetsWithoutMutation(t *testing.T) {
	t.Run("non-adjacent", func(t *testing.T) {
		state, callbacks, source := newSmallMapShadowlordEncounterFixture(
			t,
			references.Faulinei,
			SmallMapEncounterAutomaticShadowlord,
			references.Position{X: 10, Y: 10},
			references.Position{X: 10, Y: 8},
		)
		state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)

		result, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterAttack)

		assert.ErrorIs(t, err, ErrInvalidSmallMapEncounter)
		assert.Nil(t, result)
		assert.Nil(t, state.CombatScene)
		assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
		assert.True(t, source.IsVisible())
		assert.Empty(t, callbacks.Messages)
		assert.Empty(t, callbacks.SoundEffectsPlayed)
	})

	t.Run("different-floor", func(t *testing.T) {
		state, callbacks, source := newSmallMapShadowlordEncounterFixture(
			t,
			references.Faulinei,
			SmallMapEncounterAutomaticShadowlord,
			references.Position{X: 10, Y: 10},
			references.Position{X: 10, Y: 9},
		)
		state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
		source.SetFloor(1)

		result, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterAttack)

		assert.ErrorIs(t, err, ErrInvalidSmallMapEncounter)
		assert.Nil(t, result)
		assert.Nil(t, state.CombatScene)
		assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
		assert.Empty(t, callbacks.Messages)
		assert.Empty(t, callbacks.SoundEffectsPlayed)
	})

	t.Run("stale-queued-source", func(t *testing.T) {
		state, callbacks, source := newSmallMapShadowlordEncounterFixture(
			t,
			references.Faulinei,
			SmallMapEncounterAutomaticShadowlord,
			references.Position{X: 10, Y: 10},
			references.Position{X: 10, Y: 9},
		)
		state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
		require.NoError(t, state.QueueSmallMapCombatEncounter(
			source,
			SmallMapEncounterShadowlordAdjacency,
		))
		require.True(t, state.CurrentNPCAIController.GetNpcs().RemoveEnemy(source))

		err := state.ResolvePendingOverworldContact()

		assert.ErrorIs(t, err, ErrInvalidSmallMapEncounter)
		assert.Nil(t, state.CombatScene)
		assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
		assert.Empty(t, callbacks.Messages)
		assert.Empty(t, callbacks.SoundEffectsPlayed)
	})

	t.Run("composer-not-ready", func(t *testing.T) {
		state, callbacks, source := newSmallMapShadowlordEncounterFixture(
			t,
			references.Faulinei,
			SmallMapEncounterAutomaticShadowlord,
			references.Position{X: 10, Y: 10},
			references.Position{X: 10, Y: 9},
		)
		state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
		state.EventQueueManager.Workflow("blocking_test").
			Priority(event_queue.PriorityNormal).
			Sequential().
			First(func() {}).
			Finally(func() {}).
			Start()

		result, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterShadowlordAdjacency)

		assert.ErrorIs(t, err, ErrInvalidCombatEncounter)
		assert.Nil(t, result)
		assert.Nil(t, state.CombatScene)
		assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
		assert.True(t, source.IsVisible())
		assert.Empty(t, callbacks.Messages)
		assert.Empty(t, callbacks.SoundEffectsPlayed)
	})

	t.Run("ordinary-enemy", func(t *testing.T) {
		state, callbacks, _ := newSmallMapShadowlordEncounterFixture(
			t,
			references.Faulinei,
			SmallMapEncounterAutomaticShadowlord,
			references.Position{X: 10, Y: 10},
			references.Position{X: 10, Y: 9},
		)
		state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
		enemyRef, err := state.enemyReferenceForCombat(references.MonsterOrc)
		require.NoError(t, err)
		enemy := map_units.NewEnemyNPC(*enemyRef, 0)
		enemy.SetPos(references.Position{X: 10, Y: 9})
		enemy.SetFloor(0)
		require.True(t, state.CurrentNPCAIController.GetNpcs().AddEnemy(enemy))
		state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)

		assert.False(t, state.ActionAttackSmallMap(references.Up))
		assert.Nil(t, state.CombatScene)
		assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
		assert.Equal(t, []string{"Nothing to attack!"}, callbacks.Messages)
		assert.Empty(t, callbacks.SoundEffectsPlayed)
	})
}

func newSmallMapShadowlordEncounterFixture(
	t *testing.T,
	lord references.Shadowlord,
	sourceKind SmallMapEncounterSourceKind,
	playerPosition references.Position,
	sourcePosition references.Position,
) (*GameState, *MockSystemCallbacks, *map_units.NPCEnemy) {
	t.Helper()
	location := references.Britain
	if sourceKind == SmallMapEncounterYellShadowlord {
		location = references.Lycaeum
	}
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(location).
		WithFloor(0).
		WithPlayerAt(int(playerPosition.X), int(playerPosition.Y)).
		WithRandomSeed(317).
		WithSystemCallbacks().
		Build()
	state.ShadowlordState.Reset()
	state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
	setSmallMapShadowlordTestPath(state)

	switch sourceKind {
	case SmallMapEncounterAutomaticShadowlord:
		state.ShadowlordState.TownAssignments[lord] = location
		state.ShadowlordState.AutomaticPresence = lord
	case SmallMapEncounterYellShadowlord:
		require.True(t, state.ShadowlordState.Summon(lord))
	case SmallMapEncounterOrdinaryNPC:
		t.Fatal("Shadowlord fixture requires Shadowlord source provenance")
	}

	source := state.insertShadowlordNPCAtPosition(lord, sourcePosition, true)
	require.NotNil(t, source)

	return state, callbacks, source
}

func setSmallMapShadowlordTestPath(state *GameState) {
	for y := references.Coordinate(6); y <= 10; y++ {
		position := references.Position{X: 10, Y: y}
		state.GetLayeredMapByCurrentLocation().SetTileByLayer(
			map_state.MapOverrideLayer,
			&position,
			indexes.Grass,
		)
	}
}

func smallMapShadowlordSourceName(sourceKind SmallMapEncounterSourceKind) string {
	if sourceKind == SmallMapEncounterAutomaticShadowlord {
		return "automatic"
	}
	return "yell"
}

func countStrings(values []string, target string) int {
	count := 0
	for _, value := range values {
		if value == target {
			count++
		}
	}

	return count
}
