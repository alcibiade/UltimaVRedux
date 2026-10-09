package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestReachWeaponsNeverLaunchTravellingProjectiles(t *testing.T) {
	for _, weapon := range []references.Equipment{references.MorningStar, references.Halberd} {
		for _, outcome := range []string{"hit", "miss", "empty"} {
			for _, distance := range []int{1, 2} {
				t.Run(weapon.String()+"/"+outcome+"/"+string(rune('0'+distance)), func(t *testing.T) {
					state, mock, target, _, position := setupPlayerRangedAttackTest(t, weapon, references.MonsterOrc, distance)
					if outcome == "miss" {
						target.currentDex = 255 // Forces a miss for every source d30 result.
					} else if outcome == "empty" {
						target.SetPosition(references.Position{X: 10, Y: 0})
					}
					result := state.ConfirmCombatAttackTarget()
					require.Equal(t, CombatAttackSourcePending, result.Outcome)
					require.NotNil(t, result.Launch)
					assert.False(t, result.Launch.Animate)
					assert.Nil(t, result.ReturnLaunch)
					assert.Equal(t, []SoundEffect{SoundCombatSwing}, mock.SoundEffectsPlayed,
						"hit, miss and empty reach attacks request one audible swing at commitment")
					if outcome == "hit" {
						assert.Equal(t, position, result.Launch.Impact)
					}
					if outcome == "miss" {
						assert.Equal(t, CombatHitMiss, state.CombatScene.pendingAttack.diagnostics.Hit.Outcome)
					}
					assert.Equal(t, CombatAttackSequenceComplete, state.ResolvePendingCombatAttack().Outcome)
					assert.Equal(t, CombatAttackTargetIgnored, state.ResolvePendingCombatAttack().Outcome)
				})
			}
		}
	}
}

func TestPhysicalAttackCuePreservesMeleeAndProjectileSelection(t *testing.T) {
	for _, test := range []struct {
		weapon   references.Equipment
		distance int
		cue      SoundEffect
	}{
		{references.LongSword, 1, SoundCombatSwing},
		{references.Bow, 3, SoundCombatRangedLaunch},
	} {
		t.Run(test.weapon.String(), func(t *testing.T) {
			state, mock, _, _, _ := setupPlayerRangedAttackTest(t, test.weapon, references.MonsterOrc, test.distance)
			state.PartyState.Inventory.Equipment.Set(references.Arrows, 2)
			result := state.ConfirmCombatAttackTarget()
			require.NotEqual(t, CombatAttackTargetIgnored, result.Outcome)
			assert.Contains(t, mock.SoundEffectsPlayed, test.cue)
			other := SoundCombatSwing
			if test.cue == SoundCombatSwing {
				other = SoundCombatRangedLaunch
			}
			assert.NotContains(t, mock.SoundEffectsPlayed, other)
		})
	}
}
