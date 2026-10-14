package main

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestCombatDirectionEscapeThroughUpdateAndNextFreshPass(t *testing.T) {
	for _, command := range []struct {
		key   ebiten.Key
		state InputState
	}{
		{ebiten.KeyK, KlimbDirectionInput}, {ebiten.KeyO, OpenDirectionInput},
		{ebiten.KeyJ, JimmyDoorDirectionInput}, {ebiten.KeyP, PushDirectionInput},
		{ebiten.KeyG, GetDirectionInput}, {ebiten.KeyS, SearchDirectionInput},
	} {
		t.Run(command.key.String(), func(t *testing.T) {
			assertCombatDirectionUpdateCancellation(t, command.key, command.state, "first_actor", false)
		})
	}
}

func TestCombatKlimbBesideGrateRetryEscapeThroughUpdateForFinalActor(t *testing.T) {
	for _, scenario := range []string{"last_in_sequence", "last_remaining"} {
		t.Run(scenario, func(t *testing.T) {
			assertCombatDirectionUpdateCancellation(t, ebiten.KeyK, KlimbDirectionInput, scenario, true)
		})
	}
}

func newCombatDirectionUpdateScene(t *testing.T, scenario string) (
	*GameScene, map[ebiten.Key]bool, *game_state.MockSystemCallbacks,
) {
	t.Helper()
	scene, held, _, callbacks := newRunicReferenceTestScene(t, references.Britain)
	state := scene.gameState
	require.NoError(t, state.TransitionToCombatMap(0, references.BritanniaCombatTerritory, game_state.CombatEntryNorth))
	combat := state.CombatScene
	var actors []game_state.CombatTurnIdentity
	for index, active := range combat.PartyActive {
		if !active {
			continue
		}
		identity := game_state.CombatTurnIdentity{IsPartyMember: true, EntityIndex: index}
		actors = append(actors, identity)
		character := state.PartyState.GetCharacter(index)
		character.Dexterity = 35
		character.SetStatus(party_state.Good)
		combat.PartySleeping[index], combat.PartyPossessed[index] = false, false
		entry := combat.TurnQueue.GetEntryByIdentity(identity)
		require.NotNil(t, entry)
		combat.TurnQueue.ReplaceCombatant(identity, identity, entry.Combatant)
	}
	require.GreaterOrEqual(t, len(actors), 2, "last-to-act requires multiple party actors")
	if scenario == "last_remaining" {
		for _, actor := range actors[1:] {
			combat.PartyActive[actor.EntityIndex] = false
			combat.TurnQueue.MarkFled(actor.EntityIndex, true)
		}
	}
	// Keep a live enemy so accidental completion/exit cannot be masked by an
	// empty encounter. It is not due during this first, equal-DEX party pass.
	enemies := *state.GameReferences.EnemyReferences
	rat := game_state.NewCombatMonster(references.MonsterRat, &enemies[references.MonsterRat],
		references.Position{X: 10, Y: 10}, state.AsRandomProvider())
	combat.AddMonsterToQueue(rat)
	state.BeginCombatTurns(game_state.CombatTurnsPresented)
	require.Equal(t, actors[0], combat.GetCurrentTurnIdentity())
	if scenario == "last_in_sequence" {
		for _, actor := range actors[1:] {
			state.FinishCombatAction()
			require.Equal(t, actor, combat.GetCurrentTurnIdentity(), "visit each preceding actor through the scheduler")
		}
		require.Equal(t, actors[len(actors)-1], combat.GetCurrentTurnIdentity())
	}
	if scenario == "last_remaining" {
		activeCount := 0
		for _, active := range combat.PartyActive {
			if active {
				activeCount++
			}
		}
		require.Equal(t, 1, activeCount)
	}
	position := references.Position{X: 5, Y: 5}
	combat.SetPartyMemberPosition(combat.GetCurrentTurnIdentity().EntityIndex, position)
	setProductionCombatPushTerrain(state, position, indexes.BrickFloor)
	setProductionCombatPushTerrain(state, position.Right(), indexes.Grate)
	setProductionCombatPushTerrain(state, position.Up(), indexes.BrickFloor)
	state.PartyState.Inventory.Provisions.Keys.Set(2)
	state.ActiveDurationEffect = game_state.DurationEffect{EffectType: game_state.DurationEffectProtection, TurnsRemaining: 8}
	scene.output = text.NewOutput(nil, 1, 64, 128)
	require.False(t, combat.HasScheduledCombatStep())
	callbacks.Reset()

	return scene, held, callbacks
}

func assertCombatDirectionUpdateCancellation(t *testing.T, key ebiten.Key, prompt InputState, scenario string, retry bool) {
	t.Helper()
	scene, held, callbacks := newCombatDirectionUpdateScene(t, scenario)
	expected, _, expectedCallbacks := newCombatDirectionUpdateScene(t, scenario)
	state := scene.gameState
	combat := state.CombatScene
	// Keep the injected frame clock fixed; only the command may advance time.
	update := func() {
		t.Helper()
		require.NoError(t, scene.Update(nil))
	}
	before := snapshotBlockedCombatState(t, state)
	beforeLocation := state.MapState.PlayerLocation
	var forecast [8]game_state.CombatTurnForecastEntry
	require.Equal(t, len(forecast), combat.TurnQueue.FillForecast(forecast[:]))
	held[key] = true
	update()
	require.Equal(t, prompt, scene.secondaryKeyState)
	delete(held, key)
	if retry {
		held[ebiten.KeyUp] = true
		update()
		require.Equal(t, KlimbDirectionInput, scene.secondaryKeyState)
		require.Contains(t, callbacks.Messages, "What?")
		require.Equal(t, before, snapshotBlockedCombatState(t, state))
		delete(held, ebiten.KeyUp)
	}
	held[ebiten.KeyEscape] = true
	for range 4 {
		update()
		require.Equal(t, PrimaryInput, scene.secondaryKeyState)
		require.Same(t, combat, state.CombatScene, "held Escape must not leave combat")
		require.Zero(t, scene.dialogStack.Count(), "held Escape must not open battlefield-exit confirmation")
		require.Equal(t, before, snapshotBlockedCombatState(t, state))
		require.Equal(t, beforeLocation, state.MapState.PlayerLocation)
		require.False(t, combat.HasScheduledCombatStep())
		require.False(t, state.EventQueueManager.HasActiveWorkflows())
		require.NotContains(t, callbacks.CurrentRowContent, "Escape...", "held cancellation cannot reach primary exit handling")
		require.NotContains(t, callbacks.Messages, "Not yet!")
	}
	require.Contains(t, scene.output.GetOutputStr(true), "Cancelled")
	var afterForecast [8]game_state.CombatTurnForecastEntry
	require.Equal(t, len(afterForecast), combat.TurnQueue.FillForecast(afterForecast[:]))
	require.Equal(t, forecast, afterForecast)
	require.Equal(t, 8, state.ActiveDurationEffect.TurnsRemaining)
	require.Empty(t, callbacks.TimeAdvanced())
	require.Empty(t, callbacks.CombatEvents)
	require.Empty(t, callbacks.CombatConclusions)

	// No idle/release frame: a fresh Pass is accepted immediately, then its
	// held key must not spend another action (even for the sole surviving PC).
	callbacks.Reset()
	delete(held, ebiten.KeyEscape)
	held[ebiten.KeySpace] = true
	update()
	expected.gameState.FinishCombatAction()
	require.Equal(t, snapshotBlockedCombatState(t, expected.gameState), snapshotBlockedCombatState(t, state))
	require.Equal(t, expected.gameState.ActiveDurationEffect, state.ActiveDurationEffect)
	require.Equal(t, expectedCallbacks.CommandPrompts, callbacks.CommandPrompts)
	require.NotEqual(t, 8, state.ActiveDurationEffect.TurnsRemaining, "fresh Pass must actually complete")
	afterPass := snapshotBlockedCombatState(t, state)
	prompts := len(callbacks.CommandPrompts)
	for range 3 {
		update()
		require.Equal(t, afterPass, snapshotBlockedCombatState(t, state))
		require.Equal(t, expected.gameState.ActiveDurationEffect, state.ActiveDurationEffect)
		require.Len(t, callbacks.CommandPrompts, prompts, "held fresh command executes exactly once")
	}
}

func TestCombatDirectionCancellationPreservesExactAction(t *testing.T) {
	commands := []struct {
		key   ebiten.Key
		state InputState
	}{
		{ebiten.KeyK, KlimbDirectionInput}, {ebiten.KeyO, OpenDirectionInput},
		{ebiten.KeyJ, JimmyDoorDirectionInput}, {ebiten.KeyP, PushDirectionInput},
		{ebiten.KeyG, GetDirectionInput}, {ebiten.KeyS, SearchDirectionInput},
	}
	for _, command := range commands {
		t.Run(command.key.String(), func(t *testing.T) {
			scene, callbacks := newStartedCombatPushScene(t)
			held := map[ebiten.Key]bool{}
			scene.keyboard = input.NewKeyboardWithKeyStateSource(-1, func(key ebiten.Key) bool { return held[key] })
			scene.gameState.PartyState.Inventory.Provisions.Keys.Set(2)
			before := snapshotBlockedCombatState(t, scene.gameState)
			scene.combatMapInputHandler(command.key)
			require.Equal(t, command.state, scene.secondaryKeyState)
			held[ebiten.KeyEscape] = true
			scene.combatMapHandleSecondaryInput()
			require.Equal(t, PrimaryInput, scene.secondaryKeyState)
			require.Contains(t, scene.output.GetOutputStr(true), "Cancelled")
			for range 3 {
				scene.keyboard.ClearFrameConsumption()
				require.False(t, scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyEscape), "held Escape cannot exit the battlefield")
			}
			require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
			require.Empty(t, callbacks.TimeAdvanced())
			delete(held, ebiten.KeyEscape)
			held[ebiten.KeyP] = true
			scene.keyboard.ClearFrameConsumption()
			require.True(t, scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyP))
			scene.combatMapInputHandler(ebiten.KeyP)
			require.Equal(t, PushDirectionInput, scene.secondaryKeyState)
			require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
		})
	}
}

func TestCombatDirectionPassStillCompletesExactlyOnce(t *testing.T) {
	for _, state := range []InputState{KlimbDirectionInput, OpenDirectionInput, JimmyDoorDirectionInput, PushDirectionInput, GetDirectionInput, SearchDirectionInput} {
		actual, callbacks := newStartedCombatPushScene(t)
		expected, expectedCallbacks := newStartedCombatPushScene(t)
		actual.secondaryKeyState = state
		actual.keyboard.SimulateKeyPress(ebiten.KeySpace)
		actual.combatMapHandleSecondaryInput()
		expected.gameState.FinishCombatAction()
		assertProductionCombatPushCompletion(t, actual, expected.gameState, callbacks, expectedCallbacks)
	}
}

func TestControlledMonsterPushCancellationPreservesExactAction(t *testing.T) {
	state, _, _ := setupControlledMonsterInputTest(t)
	scene := &GameScene{gameState: state, keyboard: input.NewKeyboard(0), output: text.NewOutput(nil, 1, 10, 80)}
	before := snapshotBlockedCombatState(t, state)
	scene.combatMapInputHandler(ebiten.KeyP)
	require.Equal(t, PushDirectionInput, scene.secondaryKeyState)
	scene.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	scene.combatMapHandleSecondaryInput()
	require.Equal(t, PrimaryInput, scene.secondaryKeyState)
	require.Equal(t, before, snapshotBlockedCombatState(t, state))
}

func TestCombatKlimbRetryCanCancelWithOnlyOnePartyActor(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	combat := scene.gameState.CombatScene
	for index := range combat.PartyActive {
		if index != 0 {
			combat.PartyActive[index] = false
			combat.TurnQueue.MarkFled(index, true)
		}
	}
	scene.gameState.BeginCombatTurns(game_state.CombatTurnsPresented)
	before := snapshotBlockedCombatState(t, scene.gameState)
	scene.combatMapInputHandler(ebiten.KeyK)
	scene.keyboard.SimulateKeyPress(ebiten.KeyUp)
	scene.combatMapHandleSecondaryInput()
	require.Equal(t, KlimbDirectionInput, scene.secondaryKeyState)
	scene.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	scene.combatMapHandleSecondaryInput()
	require.Equal(t, PrimaryInput, scene.secondaryKeyState)
	require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
}
