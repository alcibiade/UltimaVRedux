package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/environment"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func attachCombatSoloResetUpdateInput(t *testing.T, scene *GameScene) map[ebiten.Key]bool {
	t.Helper()
	initializeDungeonUpdateTestScene(scene, scene.gameState)
	held := make(map[ebiten.Key]bool)
	scene.keyboard = input.NewKeyboardWithKeyStateSource(-1, func(key ebiten.Key) bool { return held[key] })
	scene.keyboard.SetElapsedMillisecondsSource(scene.getElapsedMs)
	scene.gameState.SystemCallbacks.Visual.BlockAllInput = scene.blockAllInput
	scene.gameState.SystemCallbacks.Visual.IsAnyKeyPressed = scene.IsAnyKeyPressed
	t.Cleanup(func() {
		clear(held)
		scene.ResetTransientCommandInput()
		scene.gameState.EventQueueManager.Shutdown()
	})

	return held
}

func TestCombatSoloResetOwnSkippedOrPossessedTurnThroughUpdate(t *testing.T) {
	for _, condition := range []string{"sleep", "captive", "possessed"} {
		t.Run(condition, func(t *testing.T) {
			scene, held, callbacks := newCombatDirectionUpdateScene(t, "first_actor")
			state := scene.gameState
			combat := state.CombatScene
			state.SystemCallbacks.Visual.IsAnyKeyPressed = scene.IsAnyKeyPressed
			combat.SetSoloMoveCharacter(1)
			switch condition {
			case "sleep":
				combat.PartySleeping[1] = true
			case "captive":
				combat.PartyUnderground[1] = true
			case "possessed":
				combat.PartyPossessed[1] = true
			}
			state.FinishCombatAction()
			boundary, eligible := state.ProjectCombatSoloReset()
			require.True(t, eligible)
			require.Equal(t, game_state.CombatTurnIdentity{IsPartyMember: true, EntityIndex: 1}, boundary.Step.Actor)
			require.NoError(t, scene.Update(nil))
			require.True(t, scene.inputBlocked)
			before := snapshotBlockedCombatState(t, state)
			underground := combat.PartyUnderground
			callbacks.Reset()
			held[ebiten.Key0] = true
			require.NoError(t, scene.Update(nil))
			require.False(t, combat.IsSoloMoveActive())
			require.Equal(t, before, snapshotBlockedCombatState(t, state))
			require.Equal(t, underground, combat.PartyUnderground)
			require.Empty(t, callbacks.CommandPrompts)
			finishCombatSoloResetWait(t, scene)
			require.NotEqual(t, boundary.Step.Actor, combat.GetCurrentTurnIdentity(), "the skipped/hostile turn is not replayed")
			require.ErrorIs(t, state.CompleteScheduledCombatStep(boundary.Step.Token), game_state.ErrCombatPresentationDuplicate)
			clear(held)
			state.EventQueueManager.Shutdown()
		})
	}
}

func TestCombatSoloResetDuringEnemyMovementPreservesExactOwner(t *testing.T) {
	scene, held, _, boundary := newCombatSoloResetUpdateScene(t, true)
	for range 100 {
		scene.gameState.ElapsedMs += 16
		require.NoError(t, scene.Update(nil))
		if scene.combatMovement.Active {
			break
		}
	}
	require.True(t, scene.combatMovement.Active)
	movement := scene.combatMovement
	before := snapshotBlockedCombatState(t, scene.gameState)
	held[ebiten.Key0] = true
	require.NoError(t, scene.Update(nil))
	require.False(t, scene.gameState.CombatScene.IsSoloMoveActive())
	require.Equal(t, movement, scene.combatMovement)
	require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
	after, eligible := scene.gameState.ProjectCombatSoloReset()
	require.True(t, eligible)
	require.Equal(t, boundary, after)
	finishCombatSoloResetWait(t, scene)
	require.False(t, scene.combatMovement.Active)
	require.Equal(t, movement.To, scene.gameState.CombatScene.GetCombatantPosition(boundary.Step.Actor))
	require.ErrorIs(t, scene.gameState.CompleteScheduledCombatStep(boundary.Step.Token), game_state.ErrCombatPresentationDuplicate)
}

func TestCombatSoloResetDuringEnemyProjectileMatchesSeededCurrentAction(t *testing.T) {
	scene, callbacks, step := newScheduledProjectileTestScene(t, references.MonsterEttin, 4)
	held := attachCombatSoloResetUpdateInput(t, scene)
	control, controlCallbacks, controlStep := newScheduledProjectileTestScene(t, references.MonsterEttin, 4)
	attachCombatSoloResetUpdateInput(t, control)
	state := scene.gameState
	state.CombatScene.SetSoloMoveCharacter(0)
	// The queue-only projectile fixture's seed starts directly at the AI roll.
	// Production Update first advances the shared wind stream. Keep that work
	// real in both scenes and select a seed for this exact elapsed-time history.
	const frameMs int64 = 16
	launchMs := ((int64(game_state.MonsterMoveDelayMs) + frameMs - 1) / frameMs) * frameMs
	seed := combatSoloResetProjectileSeed(t, launchMs)
	require.Zero(t, state.ElapsedMs)
	require.Zero(t, control.gameState.ElapsedMs)
	state.SetRandomSeed(seed)
	control.gameState.SetRandomSeed(seed)
	scene.startScheduledMonsterTurn(step)
	control.startScheduledMonsterTurn(controlStep)
	require.NoError(t, scene.Update(nil))
	require.NoError(t, control.Update(nil))
	for state.ElapsedMs < launchMs {
		state.ElapsedMs += frameMs
		control.gameState.ElapsedMs += frameMs
		require.NoError(t, scene.Update(nil))
		require.NoError(t, control.Update(nil))
	}
	require.NotNil(t, scene.activeProjectile)
	require.NotNil(t, control.activeProjectile)
	require.Equal(t, step.Actor, state.CombatScene.GetCurrentTurnIdentity())
	require.Equal(t, control.gameState.GetRNG().Calls(), state.GetRNG().Calls())
	flight := scene.activeProjectile
	before := snapshotBlockedCombatState(t, state)
	held[ebiten.Key0] = true
	require.NoError(t, scene.Update(nil))
	require.NoError(t, control.Update(nil))
	require.False(t, state.CombatScene.IsSoloMoveActive())
	require.Same(t, flight, scene.activeProjectile)
	require.Equal(t, before, snapshotBlockedCombatState(t, state))
	boundary, eligible := state.ProjectCombatSoloReset()
	require.True(t, eligible)
	require.Equal(t, step, boundary.Step)
	for range 200 {
		state.ElapsedMs += 16
		control.gameState.ElapsedMs += 16
		require.NoError(t, scene.Update(nil))
		require.NoError(t, control.Update(nil))
		if !scene.inputBlocked && !state.CombatScene.HasScheduledCombatStep() {
			break
		}
	}
	require.False(t, scene.inputBlocked, "held 0 cannot stall projectile cleanup")
	require.Nil(t, scene.activeProjectile)
	require.Equal(t, snapshotBlockedCombatState(t, control.gameState), snapshotBlockedCombatState(t, state))
	require.Equal(t, controlCallbacks.CombatEvents, callbacks.CombatEvents)
	require.Len(t, callbacks.CombatEvents, 1)
	require.Len(t, callbacks.CombatImpactCalls, 1)
	require.False(t, state.ResolvePendingMonsterMissile())
	require.ErrorIs(t, state.CompleteScheduledCombatStep(step.Token), game_state.ErrCombatPresentationDuplicate)
}

type combatSoloResetWindCallbacks struct{}

func (combatSoloResetWindCallbacks) ShowWindUI(environment.WindDirection) {}
func (combatSoloResetWindCallbacks) AddRowStr(string)                     {}

func combatSoloResetProjectileSeed(t *testing.T, launchMs int64) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 10000; seed++ {
		rng := beradrng.New(seed)
		wind := environment.NewWindSystem(rng, combatSoloResetWindCallbacks{})
		wind.AdvanceThrough(launchMs)
		// Same successful Ettin throw as projectileHitSeed, after real wind
		// draws rather than assuming queue-only and Update histories coincide.
		if rng.Intn(256) < 128 && rng.Intn(61) >= 58 {
			return seed
		}
	}
	t.Fatal("no deterministic projectile seed after production wind updates")

	return 0
}

func TestCombatSoloResetRejectsPlayerMovementAndProjectileThroughUpdate(t *testing.T) {
	for _, action := range []string{"movement", "projectile"} {
		t.Run(action, func(t *testing.T) {
			scene, actor := setupCombatAutoAttackInputTest(t)
			held := attachCombatSoloResetUpdateInput(t, scene)
			state := scene.gameState
			require.True(t, state.SelectCombatPlayer(actor.EntityIndex))
			workflow := "player_combat_movement"
			if action == "movement" {
				held[ebiten.KeyRight] = true
			} else {
				workflow = combatAutoAttackWorkflowName
				character := state.PartyState.GetCharacter(actor.EntityIndex)
				character.SetEquippedItem(references.SlotWeapon, references.Bow)
				character.SetEquippedItem(references.SlotShield, references.NoEquipment)
				state.PartyState.Inventory.Equipment.Set(references.Arrows, 2)
				state.CombatScene.MonsterManager.GetMonster(0).SetPosition(references.Position{X: 5, Y: 2})
				state.SystemCallbacks.Visual.StartProjectile = scene.startProjectile
				state.SystemCallbacks.Visual.UpdateProjectileProgress = scene.updateProjectileProgress
				state.SystemCallbacks.Visual.ClearProjectile = scene.clearProjectile
				held[ebiten.KeyTab] = true
			}
			require.NoError(t, scene.Update(nil))
			clear(held)
			require.NoError(t, scene.Update(nil))
			require.True(t, state.EventQueueManager.HasWorkflow(workflow))
			held[ebiten.Key0] = true
			require.NoError(t, scene.Update(nil))
			require.True(t, state.CombatScene.IsSoloMoveActive())
			require.True(t, scene.keyboard.IsSuppressedUntilRelease(ebiten.Key0))
			for range 200 {
				state.ElapsedMs += 16
				require.NoError(t, scene.Update(nil))
				require.True(t, state.CombatScene.IsSoloMoveActive(), "rejected player-action input cannot become an enemy-turn reset")
				if !state.EventQueueManager.HasWorkflow(workflow) {
					break
				}
			}
			// Player workflows retain their own raw release policy; release the
			// rejected key before testing that ordinary ownership completes.
			clear(held)
			finishCombatSoloResetWait(t, scene)
			require.True(t, state.CombatScene.IsSoloMoveActive())
		})
	}
}
