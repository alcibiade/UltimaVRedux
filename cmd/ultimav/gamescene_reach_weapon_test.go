package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestReachWeaponManualAndTabCompleteWithoutProjectile(t *testing.T) {
	for _, weapon := range []references.Equipment{references.MorningStar, references.Halberd} {
		for _, mode := range []string{"tab", "manual", "empty"} {
			t.Run(weapon.String()+"/"+mode, func(t *testing.T) {
				scene, actor := setupCombatAutoAttackInputTest(t)
				state := scene.gameState
				character := state.PartyState.GetCharacter(actor.EntityIndex)
				character.SetEquippedItem(references.SlotWeapon, weapon)
				character.SetEquippedItem(references.SlotShield, references.NoEquipment)
				state.ActiveDurationEffect = game_state.DurationEffect{EffectType: game_state.DurationEffectProtection, TurnsRemaining: 10}
				launches := 0
				var sounds []game_state.SoundEffect
				state.SystemCallbacks.Audio.PlaySoundEffect = func(sound game_state.SoundEffect) {
					sounds = append(sounds, sound)
				}
				state.SystemCallbacks.Visual.StartProjectile = func(_, _ references.Position, _ references.MissileType) uint64 { launches++; return 100 }
				if mode == "tab" {
					scene.combatMapInputHandler(ebiten.KeyTab)
				} else {
					scene.combatMapInputHandler(ebiten.KeyA)
					require.NotNil(t, state.CombatScene.AttackTargeting)
					position := references.Position{X: 4, Y: 4}
					if mode == "empty" {
						position = references.Position{X: 7, Y: 5}
					}
					state.CombatScene.AttackTargeting.Cursor = position
					scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
					scene.handleCombatAttackTargetInput()
				}
				for range 20 {
					state.ElapsedMs += 100
					state.EventQueueManager.Update()
				}
				assert.Zero(t, launches)
				assert.Contains(t, sounds, game_state.SoundCombatSwing)
				assert.NotContains(t, sounds, game_state.SoundCombatRangedLaunch)
				assert.Nil(t, state.CombatScene.AttackTargeting)
				assert.False(t, scene.inputBlocked)
				assert.Equal(t, PrimaryInput, scene.secondaryKeyState)
				assert.Equal(t, 9, state.ActiveDurationEffect.TurnsRemaining)
				assert.Equal(t, game_state.CombatAttackTargetIgnored, state.ResolvePendingCombatAttack().Outcome)
			})
		}
	}
}
