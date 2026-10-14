package main

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestSurfaceDirectionEscapeProductionRouteAndNextFreshCommand(t *testing.T) {
	commands := []struct {
		key   ebiten.Key
		state InputState
	}{
		{ebiten.KeyA, AttackDirectionInput}, {ebiten.KeyG, GetDirectionInput},
		{ebiten.KeyP, PushDirectionInput}, {ebiten.KeyO, OpenDirectionInput},
		{ebiten.KeyJ, JimmyDoorDirectionInput}, {ebiten.KeyK, KlimbDirectionInput},
		{ebiten.KeyL, LookDirectionInput}, {ebiten.KeyT, TalkDirectionInput},
		{ebiten.KeyS, SearchDirectionInput}, {ebiten.KeyF, FireDirectionInput},
	}
	for _, location := range []references.Location{references.Britain, references.BritanniaUnderworld} {
		for _, command := range commands {
			if location == references.Britain && command.key == ebiten.KeyF {
				continue // Town Fire resolves its cannon without a direction prompt.
			}
			t.Run(location.String()+"/"+command.key.String(), func(t *testing.T) {
				scene, held, _, callbacks := newRunicReferenceTestScene(t, location)
				// Hold the frame clock fixed so autonomous wind ticks are not
				// mistaken for a command-owned RNG draw.
				update := func() { require.NoError(t, scene.Update(nil)) }
				state := scene.gameState
				scene.output = text.NewOutput(nil, 1, 64, 128)
				state.CurrentNPCAIController.GetNpcs().ClearAllMapUnits()
				var townController *surfaceCadenceController
				if location.GetMapType() == references.SmallMapType {
					townController = &surfaceCadenceController{NPCAIController: state.CurrentNPCAIController}
					state.CurrentNPCAIController = townController
				}
				// Outdoor phases use the concrete large-map controller's
				// AdvanceNextTurnWithActorCheck, not the town controller method.
				position := state.MapState.PlayerLocation.Position
				state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Grass)
				state.PartyState.Inventory.Provisions.Keys.Set(2)
				state.SystemCallbacks.Message.ShowCommandPrompt = scene.showCommandPrompt
				state.ActiveDurationEffect = game_state.DurationEffect{EffectType: game_state.DurationEffectProtection, TurnsRemaining: 5}
				beforeTime, beforeCalls := state.DateTime, state.GetRNG().Calls()
				beforeFood := state.PartyState.Inventory.Provisions.Food.Get()
				callbacks.Reset()
				held[command.key] = true
				update()
				require.Equal(t, command.state, scene.secondaryKeyState)
				for _, legend := range []string{"Arrows", "Space:", "Esc:", "[", "]"} {
					require.NotContains(t, scene.output.GetOutputStr(true), legend)
				}
				delete(held, command.key)
				held[ebiten.KeyEscape] = true
				for range 3 {
					update()
				}
				require.Equal(t, PrimaryInput, scene.secondaryKeyState)
				require.Zero(t, scene.dialogStack.Count(), "held cancellation cannot open Escape menu")
				require.Equal(t, beforeTime, state.DateTime)
				require.Equal(t, beforeCalls, state.GetRNG().Calls())
				require.Equal(t, beforeFood, state.PartyState.Inventory.Provisions.Food.Get())
				require.EqualValues(t, 2, state.PartyState.Inventory.Provisions.Keys.Get())
				require.Equal(t, 5, state.ActiveDurationEffect.TurnsRemaining)
				if townController != nil {
					require.Zero(t, townController.turns)
				}
				require.Empty(t, callbacks.TimeAdvanced())
				require.Nil(t, state.CombatScene)
				require.False(t, state.HasPendingOverworldContact())
				require.False(t, state.EventQueueManager.HasActiveWorkflows())
				// No idle frame between releasing Escape and the alternative command.
				delete(held, ebiten.KeyEscape)
				held[ebiten.KeySpace] = true
				update()
				minutes := game_state.DefaultLargeMapMinutesPerTurn
				if location.GetMapType() == references.SmallMapType {
					minutes = game_state.DefaultSmallMapMinutesPerTurn
				}
				expectedTime := beforeTime
				expectedTime.Advance(minutes)
				require.Equal(t, expectedTime, state.DateTime, "only the fresh Pass owns a normal cycle")
				callbacks.AssertTimeAdvanced(minutes)
				require.Equal(t, 4, state.ActiveDurationEffect.TurnsRemaining)
				if townController != nil {
					require.Equal(t, 1, townController.turns)
				}
				afterPassCalls := state.GetRNG().Calls()
				for range 3 {
					update()
				}
				require.Equal(t, expectedTime, state.DateTime, "held next key executes exactly once")
				require.Equal(t, afterPassCalls, state.GetRNG().Calls())
				require.Equal(t, 4, state.ActiveDurationEffect.TurnsRemaining)
				if townController != nil {
					require.Equal(t, 1, townController.turns)
				}
			})
		}
	}
}

func TestSurfaceCancellationDoesNotClaimLegacyUseOrCombatDirections(t *testing.T) {
	for _, location := range []references.Location{references.Britain, references.BritanniaUnderworld} {
		scene, held, _, callbacks := newRunicReferenceTestScene(t, location)
		scene.secondaryKeyState = UseDirectionInput
		held[ebiten.KeyEscape] = true
		require.False(t, scene.cancelSurfaceDirectionInput())
		require.Equal(t, UseDirectionInput, scene.secondaryKeyState)
		require.Empty(t, callbacks.TimeAdvanced())
	}
	scene, _ := newCombatUseTestScene(t)
	scene.secondaryKeyState = OpenDirectionInput
	scene.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	require.False(t, scene.cancelSurfaceDirectionInput())
	require.Equal(t, OpenDirectionInput, scene.secondaryKeyState)
}
