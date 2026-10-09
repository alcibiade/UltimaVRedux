package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatAutoAttackTabRangedLaunchImpactAndCompletion(t *testing.T) {
	for _, adjacent := range []bool{false, true} {
		t.Run(map[bool]string{false: "remote", true: "adjacent"}[adjacent], func(t *testing.T) {
			scene, actor := setupCombatAutoAttackInputTest(t)
			state := scene.gameState
			character := state.PartyState.GetCharacter(actor.EntityIndex)
			character.SetEquippedItem(references.SlotWeapon, references.Bow)
			character.SetEquippedItem(references.SlotShield, references.NoEquipment)
			character.Dexterity = 99 // Successful hit for every original d30 result against this rat.
			state.PartyState.Inventory.Equipment.Set(references.Arrows, 2)
			target := state.CombatScene.MonsterManager.GetMonster(0)
			originalArmour := target.EnemyRef().Armour
			t.Cleanup(func() { target.EnemyRef().Armour = originalArmour })
			target.EnemyRef().Armour = 0 // Bow damage is at least one, independently of the fixed seed.
			state.SetRandomSeed(539)
			state.ActiveDurationEffect = game_state.DurationEffect{EffectType: game_state.DurationEffectProtection, TurnsRemaining: 10}
			var forecast [1]game_state.CombatTurnForecastEntry
			require.Equal(t, 1, state.CombatScene.TurnQueue.FillForecast(forecast[:]))
			if !adjacent {
				target.SetPosition(references.Position{X: 5, Y: 2})
			}
			launches, clears := 0, 0
			state.SystemCallbacks.Visual.StartProjectile = func(from, to references.Position, missile references.MissileType) uint64 {
				launches++
				assert.Equal(t, state.CombatScene.GetCombatantPosition(actor), from)
				assert.Equal(t, references.MissileArrow, missile)
				return 100
			}
			state.SystemCallbacks.Visual.ClearProjectile = func() { clears++ }
			var events []game_state.CombatEvent
			state.SystemCallbacks.Combat.PublishEvent = func(event game_state.CombatEvent) { events = append(events, event) }
			hp := target.CurrentHP()
			position := state.CombatScene.GetCombatantPosition(actor)
			scene.combatMapInputHandler(ebiten.KeyTab)
			assert.True(t, scene.inputBlocked)
			assert.Equal(t, uint16(2), state.PartyState.Inventory.Equipment.Get(references.Arrows))
			state.EventQueueManager.Update()
			require.Equal(t, 1, launches)
			assert.Equal(t, uint16(1), state.PartyState.Inventory.Equipment.Get(references.Arrows))
			assert.Equal(t, hp, target.CurrentHP())
			assert.Empty(t, events)
			assert.Equal(t, actor, state.CombatScene.GetCurrentTurnIdentity())
			assert.Equal(t, 10, state.ActiveDurationEffect.TurnsRemaining, "no action completion during travel")
			assert.Equal(t, position, state.CombatScene.GetCombatantPosition(actor))
			state.ElapsedMs += 100
			for range 8 {
				state.EventQueueManager.Update()
			}
			require.NotEmpty(t, events)
			assert.Equal(t, game_state.CombatEventRangedAttack, events[0].Kind)
			assert.Equal(t, references.Bow, events[0].Diagnostics.Source)
			assert.Equal(t, game_state.CombatHitNormal, events[0].Diagnostics.Hit.Outcome)
			assert.Less(t, target.CurrentHP(), hp, "the production Tab path must actually damage the target")
			assert.Equal(t, hp-target.CurrentHP(), events[0].HPLost)
			assert.Equal(t, 9, state.ActiveDurationEffect.TurnsRemaining, "exactly one action completion")
			assert.Equal(t, forecast[0].CombatTurnIdentity, state.CombatScene.GetCurrentTurnIdentity(), "exactly one queue advancement")
			assert.Greater(t, clears, 0)
			assert.Nil(t, state.CombatScene.AttackTargeting)
			assert.False(t, state.EventQueueManager.HasWorkflow(combatAutoAttackWorkflowName))
			assert.False(t, scene.inputBlocked)
			assert.Equal(t, PrimaryInput, scene.secondaryKeyState)
			after := snapshotBlockedCombatState(t, state)
			for range 4 {
				state.EventQueueManager.Update()
			}
			assert.Equal(t, after, snapshotBlockedCombatState(t, state), "completion is exact once")
		})
	}
}

func TestCombatAutoAttackShutdownBoundaries(t *testing.T) {
	for _, boundary := range []string{"before first update", "during return", "after final impact before cleanup"} {
		t.Run(boundary, func(t *testing.T) {
			scene, actor := setupCombatAutoAttackInputTest(t)
			state := scene.gameState
			weapon := references.Bow
			if boundary == "during return" {
				weapon = references.MagicAxe
			}
			character := state.PartyState.GetCharacter(actor.EntityIndex)
			character.SetEquippedItem(references.SlotWeapon, weapon)
			character.SetEquippedItem(references.SlotShield, references.NoEquipment)
			character.Dexterity = 99
			state.SetRandomSeed(539)
			state.PartyState.Inventory.Equipment.Set(references.Arrows, 2)
			state.ActiveDurationEffect = game_state.DurationEffect{EffectType: game_state.DurationEffectProtection, TurnsRemaining: 10}
			var forecast [1]game_state.CombatTurnForecastEntry
			require.Equal(t, 1, state.CombatScene.TurnQueue.FillForecast(forecast[:]))
			launches, impacts, clears := 0, 0, 0
			reachedFinalImpact := false
			state.SystemCallbacks.Visual.StartProjectile = func(_, _ references.Position, _ references.MissileType) uint64 {
				launches++
				return 100
			}
			state.SystemCallbacks.Visual.ClearProjectile = func() { clears++ }
			state.SystemCallbacks.Combat.PublishEvent = func(event game_state.CombatEvent) {
				if event.Kind == game_state.CombatEventRangedAttack {
					impacts++
				}
			}
			state.SystemCallbacks.Visual.PlayKapowAtPosition = func(_ references.Position) {
				if boundary != "after final impact before cleanup" {
					return
				}
				reachedFinalImpact = true
				require.Equal(t, 1, impacts)
				require.Nil(t, state.CombatScene.AttackTargeting)
				assert.Equal(t, actor, state.CombatScene.GetCurrentTurnIdentity())
				assert.Equal(t, 10, state.ActiveDurationEffect.TurnsRemaining)
				assert.True(t, scene.inputBlocked)
				state.EventQueueManager.Shutdown()
			}
			scene.combatMapInputHandler(ebiten.KeyTab)
			switch boundary {
			case "before first update":
				state.EventQueueManager.Shutdown()
				assert.Zero(t, launches)
				assert.Zero(t, impacts)
				assert.Equal(t, uint16(2), state.PartyState.Inventory.Equipment.Get(references.Arrows))
			case "during return":
				state.EventQueueManager.Update()
				state.ElapsedMs += 100
				for range 4 {
					state.EventQueueManager.Update()
				}
				require.Equal(t, 2, launches)
				require.Equal(t, 1, impacts)
				state.ElapsedMs += 50
				state.EventQueueManager.Update()
				assert.True(t, scene.inputBlocked)
				assert.Equal(t, 10, state.ActiveDurationEffect.TurnsRemaining)
				state.EventQueueManager.Shutdown()
				assert.Equal(t, 1, impacts, "return cancellation cannot resolve the hit again")
			case "after final impact before cleanup":
				state.EventQueueManager.Update()
				state.ElapsedMs += 100
				for range 4 {
					state.EventQueueManager.Update()
				}
				require.True(t, reachedFinalImpact, "shutdown must occur inside the post-resolution/pre-cleanup boundary")
				assert.Equal(t, 1, impacts)
				assert.Equal(t, uint16(1), state.PartyState.Inventory.Equipment.Get(references.Arrows))
			}
			assert.Greater(t, clears, 0)
			assert.Nil(t, state.CombatScene.AttackTargeting)
			assert.Equal(t, game_state.CombatAttackTargetIgnored, state.ResolvePendingCombatAttack().Outcome)
			assert.False(t, state.EventQueueManager.HasWorkflow(combatAutoAttackWorkflowName))
			assert.False(t, scene.inputBlocked)
			assert.Equal(t, PrimaryInput, scene.secondaryKeyState)
			assert.Equal(t, 9, state.ActiveDurationEffect.TurnsRemaining, "one accepted Tab action, even on cancellation")
			assert.Equal(t, forecast[0].CombatTurnIdentity, state.CombatScene.GetCurrentTurnIdentity())
			after := snapshotBlockedCombatState(t, state)
			clearsAfter := clears
			state.EventQueueManager.Shutdown()
			for range 4 {
				state.EventQueueManager.Update()
			}
			assert.Equal(t, after, snapshotBlockedCombatState(t, state))
			assert.Equal(t, clearsAfter, clears)
		})
	}
}

func TestCombatAutoAttackReturningSourcesAwaitReturnAndShutdownCleansOnce(t *testing.T) {
	for _, shutdown := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "shutdown during outbound"}[shutdown], func(t *testing.T) {
			scene, actor := setupCombatAutoAttackInputTest(t)
			state := scene.gameState
			character := state.PartyState.GetCharacter(actor.EntityIndex)
			character.SetEquippedItem(references.SlotWeapon, references.MagicAxe)
			character.SetEquippedItem(references.SlotShield, references.MagicAxe)
			target := state.CombatScene.MonsterManager.GetMonster(0)
			require.Equal(t, 100, target.CurrentHP())
			launches, clears, impacts := 0, 0, 0
			state.SystemCallbacks.Visual.StartProjectile = func(from, to references.Position, missile references.MissileType) uint64 {
				launches++
				assert.Equal(t, references.MissileAxe, missile)
				return 100
			}
			state.SystemCallbacks.Visual.ClearProjectile = func() { clears++ }
			state.SystemCallbacks.Combat.PublishEvent = func(event game_state.CombatEvent) {
				if event.Kind == game_state.CombatEventRangedAttack {
					impacts++
				}
			}
			scene.combatMapInputHandler(ebiten.KeyTab)
			state.EventQueueManager.Update()
			require.Equal(t, 1, launches)
			assert.Zero(t, impacts)
			if shutdown {
				state.EventQueueManager.Shutdown()
				assert.Equal(t, 1, impacts, "committed attack resolves, uncommitted source is cancelled")
				assert.Nil(t, state.CombatScene.AttackTargeting)
				assert.False(t, scene.inputBlocked)
				after := snapshotBlockedCombatState(t, state)
				clearsAfter := clears
				state.EventQueueManager.Shutdown()
				assert.Equal(t, after, snapshotBlockedCombatState(t, state))
				assert.Equal(t, clearsAfter, clears)
				return
			}
			state.ElapsedMs += 100
			for range 4 {
				state.EventQueueManager.Update()
			}
			assert.Equal(t, 2, launches, "first return, not the next source")
			assert.Equal(t, 1, impacts)
			require.False(t, target.IsDead(), "this lifecycle fixture must retain the same victim for its second source")
			require.NotNil(t, state.CombatScene.AttackTargeting)
			assert.Equal(t, 1, state.CombatScene.AttackTargeting.SourceIndex)
			assert.Equal(t, references.MagicAxe, state.CombatScene.AttackTargeting.CurrentSource().Equipment)
			assert.Equal(t, actor, state.CombatScene.GetCurrentTurnIdentity())
			assert.True(t, scene.inputBlocked)
			state.ElapsedMs += 100
			for range 4 {
				state.EventQueueManager.Update()
			}
			assert.Equal(t, 3, launches, "next source starts after the first return")
			assert.Equal(t, 1, impacts)
			state.ElapsedMs += 100
			for range 4 {
				state.EventQueueManager.Update()
			}
			assert.Equal(t, 4, launches)
			assert.Equal(t, 2, impacts)
			assert.True(t, scene.inputBlocked, "last return still owns input")
			state.ElapsedMs += 100
			for range 8 {
				state.EventQueueManager.Update()
			}
			assert.False(t, scene.inputBlocked)
			assert.False(t, state.EventQueueManager.HasWorkflow(combatAutoAttackWorkflowName))
			assert.Nil(t, state.CombatScene.AttackTargeting)
			assert.Equal(t, references.MagicAxe, character.GetEquippedItem(references.SlotWeapon))
			assert.Greater(t, clears, 0)
		})
	}
}
