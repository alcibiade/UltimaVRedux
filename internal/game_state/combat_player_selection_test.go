package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func newCombatPlayerSelectionTestState(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(42).
		Build()
	for index := range party_state.MaxActiveParty {
		character := state.PartyState.GetCharacter(index)
		character.PartyMap = party_state.PartyMapActive
		character.SetStatus(party_state.Good)
		character.SetHp(character.GetMaxHp())
		character.Dexterity = 35
		character.SetEquippedItem(references.SlotWeapon, references.NoEquipment)
		character.SetEquippedItem(references.SlotShield, references.NoEquipment)
		character.SetEquippedItem(references.SlotRing, references.NoEquipment)
		require.True(t, character.IsAlive())
		require.NotEmpty(t, character.GetNameAsString())
	}
	require.NoError(t, state.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth))
	state.CombatScene.MonsterManager.Clear()
	state.CombatScene.TurnQueue.RemoveAllMonsters()
	state.ActiveDurationEffect = DurationEffect{}
	for x := range references.XCombatMapTiles {
		for y := range references.YCombatMapTiles {
			state.CombatScene.CombatMapRef.SetTileNumber(references.Position{
				X: references.Coordinate(x), Y: references.Coordinate(y),
			}, indexes.BrickFloor)
		}
	}
	state.BeginCombatTurns(CombatTurnsPresented)
	require.False(t, state.CombatScene.HasScheduledCombatStep())
	require.Equal(t, 0, state.CombatScene.GetCurrentTurnIdentity().EntityIndex)
	callbacks.Reset()

	return state, callbacks
}

func TestCombatPlayerSelectionRejectsUnavailableEncounterEntries(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*GameState, CombatTurnIdentity)
	}{
		{name: "departed", mutate: func(gs *GameState, identity CombatTurnIdentity) {
			gs.departPCFromCombat(identity)
		}},
		{name: "absorbed", mutate: func(gs *GameState, identity CombatTurnIdentity) {
			gs.CombatScene.TurnQueue.GetEntryByIdentity(identity).status = turnQueueAbsorbed
			gs.CombatScene.PartyActive[identity.EntityIndex] = false
		}},
		{name: "dead", mutate: func(gs *GameState, identity CombatTurnIdentity) {
			gs.PartyState.GetCharacter(identity.EntityIndex).MarkAsDead()
		}},
		{name: "zero hp", mutate: func(gs *GameState, identity CombatTurnIdentity) {
			gs.PartyState.GetCharacter(identity.EntityIndex).SetHp(0)
		}},
		{name: "sleeping entry", mutate: func(gs *GameState, identity CombatTurnIdentity) {
			gs.CombatScene.setCombatPCSleeping(identity, true)
		}},
		{name: "underground", mutate: func(gs *GameState, identity CombatTurnIdentity) {
			gs.CombatScene.setCombatPCUnderground(identity, true)
		}},
		{name: "possessed", mutate: func(gs *GameState, identity CombatTurnIdentity) {
			gs.CombatScene.setCombatPCPossessed(identity, true)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, _ := newCombatPlayerSelectionTestState(t)
			require.True(t, state.SelectCombatPlayer(0))
			test.mutate(state, CombatTurnIdentity{IsPartyMember: true, EntityIndex: 4})
			before := state.CombatScene.TurnQueue.actionCount
			randomCalls := state.GetRNG().Calls()

			assert.False(t, state.SelectCombatPlayer(4))

			assert.Equal(t, 0, state.GetActivePlayerIndex())
			assert.Equal(t, 0, state.CombatScene.GetSoloMoveCharacter())
			assert.Equal(t, before, state.CombatScene.TurnQueue.actionCount)
			assert.Equal(t, randomCalls, state.GetRNG().Calls())
			assert.False(t, state.CombatScene.HasScheduledCombatStep())
		})
	}
}

func TestCombatPlayerSelectionRapidSwitchAndClear(t *testing.T) {
	state, _ := newCombatPlayerSelectionTestState(t)
	// No elapsed-time advancement between selections: valid rapid input must
	// retain DEX ordering and terminate without relying on keyboard debounce.
	for range 10 {
		for _, selected := range []int{4, 5} {
			require.True(t, state.SelectCombatPlayer(selected))
			assert.Equal(t, selected, state.CombatScene.GetCurrentTurnIdentity().EntityIndex)
			assert.Equal(t, selected, state.CombatScene.GetSoloMoveCharacter())
			assert.False(t, state.CombatScene.HasScheduledCombatStep())
			before := state.CombatScene.TurnQueue.actionCount
			require.True(t, state.SelectCombatPlayer(selected))
			assert.Equal(t, before, state.CombatScene.TurnQueue.actionCount)
		}
	}
	before := state.CombatScene.TurnQueue.actionCount
	require.True(t, state.SelectCombatPlayer(-1))
	assert.False(t, state.CombatScene.IsSoloMoveActive())
	assert.Equal(t, -1, state.GetActivePlayerIndex())
	assert.Equal(t, before, state.CombatScene.TurnQueue.actionCount)
}

func TestCombatPlayerSelectionRejectsOverlappingPresentation(t *testing.T) {
	for _, blockingWorkflow := range []bool{false, true} {
		t.Run(map[bool]string{false: "scheduled step", true: "workflow"}[blockingWorkflow], func(t *testing.T) {
			state, _ := newCombatPlayerSelectionTestState(t)
			require.True(t, state.SelectCombatPlayer(0))
			if blockingWorkflow {
				state.EventQueueManager.Workflow("selection_test").Priority(event_queue.PriorityHigh).
					Sequential().BlockInput().First(func() {}).Wait(100).Then(func() {}).Start()
			} else {
				state.FinishCombatTurnDeferred()
			}
			before := state.CombatScene.TurnQueue.actionCount
			for _, selected := range []int{4, 5, -1} {
				assert.False(t, state.SelectCombatPlayer(selected))
			}
			assert.Equal(t, before, state.CombatScene.TurnQueue.actionCount)
			assert.Equal(t, 0, state.CombatScene.GetSoloMoveCharacter())
			assert.Equal(t, 0, state.GetActivePlayerIndex())
		})
	}
}

func TestCombatAdvanceRecoversUnreachableSoloSelection(t *testing.T) {
	for _, presented := range []bool{false, true} {
		t.Run(map[bool]string{false: "immediate", true: "presented"}[presented], func(t *testing.T) {
			state, _ := newCombatPlayerSelectionTestState(t)
			state.CombatScene.presentTurns = presented
			state.departPCFromCombat(CombatTurnIdentity{IsPartyMember: true, EntityIndex: 4})
			// Bypass command validation to exercise the turn-loop invariant itself.
			require.True(t, state.SetActivePlayer(4), "departed character remains in the persistent party")
			state.CombatScene.SetSoloMoveCharacter(4)
			before := state.CombatScene.TurnQueue.actionCount

			state.AdvanceCombatTurn()

			assert.False(t, state.CombatScene.IsSoloMoveActive())
			assert.Equal(t, before+1, state.CombatScene.TurnQueue.actionCount)
			assert.NotEqual(t, 4, state.GetActivePlayerIndex())
			assert.False(t, state.CombatScene.HasScheduledCombatStep())
		})
	}
}

func TestCombatPlayerSelectionAllowsSurvivingClone(t *testing.T) {
	state, _ := newCombatPlayerSelectionTestState(t)
	source := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 4}
	clone := state.CombatScene.addPCCloneFromSource(source, 4, references.Position{X: 5, Y: 5},
		NewCombatPCProvider(state.PartyState.GetCharacter(4)))
	state.departPCFromCombat(source)

	require.True(t, state.SelectCombatPlayer(4))

	assert.Equal(t, clone, state.CombatScene.GetCurrentTurnIdentity())
	assert.True(t, state.CombatScene.IsSoloMoveActive())
}

func TestCombatPlayerSelectionAdvancesControlledMonsterWithMatchingRosterIndex(t *testing.T) {
	state, _ := newCombatPlayerSelectionTestState(t)
	_, monsterIndex := addControlledTestMonster(state, references.MonsterOrc,
		references.Position{X: 5, Y: 5}, CombatMonsterLoyaltyParty)
	authorizeControlledMonster(state, monsterIndex)
	// CurrentTurnIndex retains the previous PC; it is not the current actor.
	state.CombatScene.CurrentTurnIndex = 4
	before := state.CombatScene.TurnQueue.actionCount

	require.True(t, state.SelectCombatPlayer(4))

	assert.NotEqual(t, before, state.CombatScene.TurnQueue.actionCount)
	assert.Equal(t, CombatTurnIdentity{IsPartyMember: true, EntityIndex: 4}, state.CombatScene.GetCurrentTurnIdentity())
}

func TestCombatPlayerSelectionWaitsForInterveningHostileTurn(t *testing.T) {
	state, _ := newCombatPlayerSelectionTestState(t)
	_, monsterIndex := addControlledTestMonster(state, references.MonsterOrc,
		references.Position{X: 5, Y: 5}, CombatMonsterLoyaltyMonster)
	for _, entry := range state.CombatScene.TurnQueue.entries {
		entry.turnCountdown = turnCountdownBase
	}
	state.CombatScene.TurnQueue.GetEntry(monsterIndex, false).turnCountdown = 1

	require.True(t, state.SelectCombatPlayer(4))
	step, ok := state.ReviewScheduledCombatStep()
	require.True(t, ok)
	require.Equal(t, CombatPresentationMonsterTurn, step.Kind)
	require.Equal(t, monsterIndex, step.MonsterIndex)
	before := state.CombatScene.TurnQueue.actionCount

	assert.False(t, state.SelectCombatPlayer(5))
	assert.Equal(t, 4, state.CombatScene.GetSoloMoveCharacter())
	assert.Equal(t, before, state.CombatScene.TurnQueue.actionCount)
	assert.Equal(t, step.Token, state.CombatScene.pendingPresentationStep.step.Token)
}
