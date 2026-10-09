package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestDeferredDungeonCastUseCancellationPreservesClockPhaseAndRNG(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	state.SetRandomSeed(42)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectQuickness, TurnsRemaining: 10}
	beforeTime := state.DateTime
	beforeCalls := state.GetRNG().Calls()

	token := state.BeginDeferredDungeonCommandCycle()
	assert.False(t, state.IsAtDungeonCommandBoundary())
	state.ClaimDungeonFullCommand(token)
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	assert.Equal(t, beforeTime, state.DateTime)
	assert.Equal(t, beforeCalls, state.GetRNG().Calls())
	assert.False(t, state.activeDungeon.quicknessSkipNext)
	assert.Equal(t, 10, state.ActiveDurationEffect.TurnsRemaining)
	assert.True(t, state.IsAtDungeonCommandBoundary())
	assert.Panics(t, func() { state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer) })

	// The next ordinary command still receives the first, charged quickness phase.
	ordinary, start := state.BeginDungeonCommandCycle()
	assert.Equal(t, 1, start.ElapsedMinutes)
	state.CompleteDungeonCommandCycle(ordinary, DungeonCommandOnly)
}

func TestDeferredDungeonCastUseCommitChargesOnceBeforeEffect(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	before := state.DateTime
	token := state.BeginDeferredDungeonCommandCycle()
	state.CommitCastUseAttempt()
	expected := before
	expected.Advance(1)
	assert.Equal(t, expected, state.DateTime)
	assert.Equal(t, uint64(1), state.CastUseCommitSequence())
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	assert.Equal(t, expected, state.DateTime)
	assert.True(t, state.IsAtDungeonCommandBoundary())
}

func TestDungeonToggleUseCommitsBeforeReplacingActorPhaseEffect(t *testing.T) {
	for _, effect := range []DurationEffectType{DurationEffectTimeStop, DurationEffectQuickness} {
		name := "Time Stop"
		if effect == DurationEffectQuickness {
			name = "Quickness"
		}
		t.Run(name, func(t *testing.T) {
			state := newDungeonStateTestGame(t, references.DungeonCells{})
			state.PartyState.Inventory.QuestItems.Set(references.Crown, 1)
			state.ActiveDurationEffect = DurationEffect{EffectType: effect, TurnsRemaining: 10}
			state.activeDungeon.quicknessSkipNext = effect == DurationEffectQuickness
			before := state.DateTime
			token := state.BeginDeferredDungeonCommandCycle()
			require.True(t, state.useQuestItem(references.Crown, func() {}))

			assert.True(t, state.IsCrownActive())
			assert.Equal(t, DurationEffectNone, state.ActiveDurationEffect.EffectType)
			assert.Equal(t, before, state.DateTime, "this action uses the phase active at commitment")
			assert.Equal(t, uint64(1), state.CastUseCommitSequence())
			state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
			assert.Equal(t, before, state.DateTime)
			assert.True(t, state.IsAtDungeonCommandBoundary())
		})
	}
}

func TestCloneTargetingDefersAllPlacementRNGUntilPaidResolution(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		name := "success"
		if blocked {
			name = "bounded placement failure"
		}
		t.Run(name, func(t *testing.T) {
			state, callbacks, target := setupVasFlamTargeting(t)
			casterIdx := state.CombatScene.GetActiveCombatantIndex()
			caster := state.PartyState.GetCharacter(casterIdx)
			caster.Level = 8
			state.PartyState.Inventory.Spells.Set(references.InQuasXen, 2)
			if blocked {
				for x := range references.XCombatMapTiles {
					for y := range references.YCombatMapTiles {
						state.setCombatTerrain(references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}, indexes.StoneBrickWall)
					}
				}
			}
			request := CastRequest{Spell: references.InQuasXen, CasterIdx: casterIdx}
			beforeCalls := state.GetRNG().Calls()
			beforeMP := caster.GetCurrentMp()
			beforeEntries := state.CombatScene.TurnQueue.TotalEntryCount()
			require.True(t, state.BeginCombatSpellTargeting(request).Valid)
			state.CombatScene.SpellTargeting.Cursor = target.Position()
			require.Equal(t, CombatSpellTargetCancelled, state.CancelCombatSpellTargeting())
			assert.Equal(t, beforeCalls, state.GetRNG().Calls())
			assert.Equal(t, beforeMP, caster.GetCurrentMp())
			require.True(t, state.BeginCombatSpellTargeting(request).Valid)
			state.CombatScene.SpellTargeting.Cursor = target.Position()
			require.Equal(t, CombatSpellTargetResolved, state.ConfirmCombatSpellTarget().Outcome)
			assert.Equal(t, beforeCalls, state.GetRNG().Calls(), "confirmation does not search for a free reroll")
			assert.Equal(t, beforeMP-byte(references.GetSpellCircle(references.InQuasXen)), caster.GetCurrentMp())
			assert.Equal(t, uint16(1), state.PartyState.Inventory.Spells.Get(references.InQuasXen))
			assert.Nil(t, state.CombatScene.SpellTargeting, "placement failure cannot return to free targeting")
			require.True(t, state.ResolvePendingCombatSpell())
			afterCalls := state.GetRNG().Calls()
			if blocked {
				assert.Equal(t, beforeCalls+512, afterCalls, "256 attempts, two coordinates each")
				assert.Equal(t, beforeEntries, state.CombatScene.TurnQueue.TotalEntryCount())
				assert.Contains(t, callbacks.Messages, "Failed! No room for a clone.")
			} else {
				assert.Greater(t, afterCalls, beforeCalls)
				assert.Equal(t, beforeEntries+1, state.CombatScene.TurnQueue.TotalEntryCount())
			}
			assert.False(t, state.ResolvePendingCombatSpell())
			assert.Equal(t, afterCalls, state.GetRNG().Calls())
			assert.Equal(t, uint64(1), state.CastUseCommitSequence())
		})
	}
}

func TestPotionFinalTargetRejectsInvalidOrStaleWithoutCommitment(t *testing.T) {
	for _, stale := range []bool{false, true} {
		state, _ := newUseCommitTestState(t)
		state.PartyState.Inventory.Potions.Set(references.Red, 1)
		commit, _ := state.BeginPotionUse(references.Red, UsageField)
		beforeCalls := state.GetRNG().Calls()
		target := -1
		if stale {
			target = 0
			state.MapState.PlayerLocation.Position.X++
		}
		_, ok := state.CommitPotionUseForTarget(commit, target)
		assert.False(t, ok)
		assert.Equal(t, uint16(1), state.PartyState.Inventory.Potions.Get(references.Red))
		assert.Zero(t, state.CastUseCommitSequence())
		assert.Equal(t, beforeCalls, state.GetRNG().Calls())
	}
}

func TestCloneConfirmationRejectsFullCapacityWithoutResourcesOrRNG(t *testing.T) {
	state, _, target := setupVasFlamTargeting(t)
	casterIdx := state.CombatScene.GetActiveCombatantIndex()
	caster := state.PartyState.GetCharacter(casterIdx)
	caster.Level = 8
	state.PartyState.Inventory.Spells.Set(references.InQuasXen, 2)
	request := CastRequest{Spell: references.InQuasXen, CasterIdx: casterIdx}
	require.True(t, state.BeginCombatSpellTargeting(request).Valid)
	identity := state.CombatScene.GetCurrentTurnIdentity()
	for state.CombatScene.TurnQueue.hasFreeSourceSlot(0) {
		state.CombatScene.addPCCloneFromSource(identity, casterIdx,
			references.Position{}, NewCombatPCProvider(caster))
	}
	beforeCalls := state.GetRNG().Calls()
	beforeMP := caster.GetCurrentMp()
	beforeEntries := state.CombatScene.TurnQueue.TotalEntryCount()
	state.CombatScene.SpellTargeting.Cursor = target.Position()

	assert.Equal(t, CombatSpellTargetInvalid, state.ConfirmCombatSpellTarget().Outcome)
	assert.Equal(t, beforeCalls, state.GetRNG().Calls())
	assert.Equal(t, beforeMP, caster.GetCurrentMp())
	assert.Equal(t, uint16(2), state.PartyState.Inventory.Spells.Get(references.InQuasXen))
	assert.Equal(t, beforeEntries, state.CombatScene.TurnQueue.TotalEntryCount())
	assert.Zero(t, state.CastUseCommitSequence())
	assert.Nil(t, state.CombatScene.pendingSpell)
	assert.Equal(t, identity, state.CombatScene.GetCurrentTurnIdentity())
	assert.Equal(t, CombatSpellTargetCancelled, state.CancelCombatSpellTargeting())
}
