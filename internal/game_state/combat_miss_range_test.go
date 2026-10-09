package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestPlayerToPlayerMissStaysWithinWeaponRange(t *testing.T) {
	for _, weapon := range []references.Equipment{
		references.Dagger, references.Sling, references.MorningStar, references.Halberd, references.Bow,
	} {
		t.Run(weapon.String(), func(t *testing.T) {
			state, attacker, target := newSyntheticExactIdentityCastUseState(t)
			mock := NewMockSystemCallbacks(t)
			state.SystemCallbacks = mock.ToSystemCallbacks()
			character := state.PartyState.GetCharacter(attacker.EntityIndex)
			character.Dexterity = 1
			character.Strength = 1
			character.SetEquippedItem(references.SlotWeapon, weapon)
			state.PartyState.Inventory.Equipment.Set(references.Dagger, 2)
			state.PartyState.Inventory.Equipment.Set(references.Arrows, 2)
			origin := references.Position{X: 2, Y: 5}
			aim := references.Position{X: 2 + references.Coordinate(weapon.GetAttackRange()), Y: 5}
			overshoot := references.Position{X: aim.X + 1, Y: aim.Y}
			state.CombatScene.setCombatantPosition(attacker, origin)
			state.CombatScene.setCombatantPosition(target, aim)
			state.CombatScene.setCombatPCPossessed(target, true)
			state.CombatScene.addPCCloneFromSource(attacker, attacker.EntityIndex, overshoot, NewCombatPCProvider(character))
			require.True(t, state.BeginCombatAttack())
			state.CombatScene.AttackTargeting.Cursor = aim
			// Select a fixed-seed miss whose first scatter would hit the
			// beyond-range tactical copy without the range restriction.
			source := state.CombatScene.AttackTargeting.CurrentSource()
			found := false
			for seed := uint64(0); seed < 1000; seed++ {
				state.SetRandomSeed(seed)
				if state.commitPhysicalRangedHit(attacker, source, target).Outcome != CombatHitMiss {
					continue
				}
				if randomAdjacentCombatMissAimWithRNG(aim, origin, state.AsRandomProvider()) == overshoot {
					state.SetRandomSeed(seed)
					found = true
					break
				}
			}
			require.True(t, found, "expected a deterministic outward miss seed")
			hpBefore := character.GetCurrentHp()

			result := state.ConfirmCombatAttackTarget()

			require.Equal(t, CombatAttackSourcePending, result.Outcome)
			require.NotNil(t, result.Launch)
			assert.Equal(t, CombatHitMiss, state.CombatScene.pendingAttack.diagnostics.Hit.Outcome)
			assert.NotEqual(t, aim, result.Launch.Aim)
			assert.NotEqual(t, origin, result.Launch.Aim)
			assert.LessOrEqual(t, integerDistance(origin, result.Launch.Aim), weapon.GetAttackRange())
			assert.LessOrEqual(t, integerDistance(origin, result.Launch.Impact), weapon.GetAttackRange())
			assert.Greater(t, state.GetRNG().Calls(), uint64(3), "outward scatter must be rejected")
			assert.Equal(t, weapon != references.MorningStar && weapon != references.Halberd, result.Launch.Animate)
			assert.Equal(t, CombatAttackSequenceComplete, state.ResolvePendingCombatAttack().Outcome)
			assert.Equal(t, hpBefore, character.GetCurrentHp(), "no tactical copy may take beyond-range collateral damage")
			require.Len(t, mock.CombatEvents, 1)
			assert.Equal(t, CombatFeedbackMiss, mock.CombatEvents[0].Result)
			assert.Equal(t, result.Launch.Impact, mock.CombatEvents[0].To)
		})
	}
}

func TestPhysicalMissScatterRejectsOnlyBeyondRangeCandidates(t *testing.T) {
	origin := references.Position{X: 5, Y: 5}
	for _, test := range []struct {
		name  string
		aim   references.Position
		rolls []int
		want  references.Position
	}{
		{name: "east boundary", aim: references.Position{X: 9, Y: 5}, rolls: []int{2, 1, 1, 2}, want: references.Position{X: 9, Y: 6}},
		{name: "west boundary", aim: references.Position{X: 1, Y: 5}, rolls: []int{0, 1, 1, 2}, want: references.Position{X: 1, Y: 6}},
		{name: "north boundary", aim: references.Position{X: 5, Y: 1}, rolls: []int{1, 0, 2, 1}, want: references.Position{X: 6, Y: 1}},
		{name: "south boundary", aim: references.Position{X: 5, Y: 9}, rolls: []int{1, 2, 2, 1}, want: references.Position{X: 6, Y: 9}},
		{name: "diagonal boundary", aim: references.Position{X: 8, Y: 8}, rolls: []int{2, 2, 0, 1}, want: references.Position{X: 7, Y: 8}},
		{name: "inside range", aim: references.Position{X: 8, Y: 5}, rolls: []int{2, 1}, want: references.Position{X: 9, Y: 5}},
		{name: "map corner", aim: references.Position{X: 10, Y: 10}, rolls: []int{2, 2, 0, 1}, want: references.Position{X: 9, Y: 10}},
	} {
		t.Run(test.name, func(t *testing.T) {
			rng := newSequenceRNG(test.rolls...)
			maxRange := 4
			if test.name == "map corner" {
				maxRange = 7
			}
			got := randomAdjacentCombatAttackMissAimWithRNG(test.aim, origin, maxRange, rng)
			assert.Equal(t, test.want, got)
			assert.Equal(t, len(test.rolls), rng.index)
		})
	}
}

func TestMonsterMissStaysWithinAttackRange(t *testing.T) {
	for _, forced := range []bool{false, true} {
		name := "ordinary miss"
		if forced {
			name = "amulet forced miss"
		}
		t.Run(name, func(t *testing.T) {
			state, mock, _, target, _, monster, monsterIndex := newAttackWakeFixture(t)
			monster.SetPosition(references.Position{X: 2, Y: 5})
			monster.enemyRef.AttackRange = 3
			character := state.PartyState.GetCharacter(target.EntityIndex)
			character.Dexterity = 30
			rng := newSequenceRNG(0, 0, 2, 1, 1, 2)
			if forced {
				monster.enemyRef.EnemyAbilities = map[references.EnemyAbility]bool{references.RangedMagic: true}
				character.SetEquippedItem(references.SlotAmulet, references.AmuletOfTurning)
			}
			nearest := nearestPCResult{found: true, identity: target, pcIndex: target.EntityIndex, distance: 3}
			hpBefore := character.GetCurrentHp()

			launch, attacked := state.prepareMonsterAttackWithRNG(monster, monsterIndex, nearest, rng)

			require.True(t, attacked)
			require.NotNil(t, launch)
			assert.Equal(t, references.Position{X: 5, Y: 6}, launch.Aim)
			assert.Equal(t, launch.Aim, launch.Impact)
			assert.Equal(t, 6, rng.index, "decision/hit or amulet/decision, rejected scatter, accepted scatter")
			assert.Equal(t, forced, state.CombatScene.pendingMonsterMissile.diagnostics.ForcedMiss)
			require.True(t, state.ResolvePendingMonsterMissile())
			assert.Equal(t, hpBefore, character.GetCurrentHp())
			assert.Equal(t, CombatFeedbackMiss, mock.CombatEvents[0].Result)
		})
	}
}
