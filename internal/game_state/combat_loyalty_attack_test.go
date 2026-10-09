package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func setupSadujCombatCopies(t *testing.T) (*GameState, *MockSystemCallbacks, CombatTurnIdentity, CombatTurnIdentity) {
	t.Helper()
	gs, mock := setupCombatStatusTest(t)
	require.NotNil(t, gs)
	source := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 1}
	character := gs.PartyState.GetCharacter(source.EntityIndex)
	clear(character.Name[:])
	copy(character.Name[:], "Saduj")
	character.SetStatus(party_state.Good)
	character.SetMaxHp(200)
	character.SetHp(200)
	character.Dexterity = 1
	for index := range party_state.MaxActiveParty {
		gs.CombatScene.SetPartyMemberPosition(index, references.Position{X: 10, Y: 10})
	}
	clone := gs.CombatScene.addPCCloneFromSource(source, source.EntityIndex,
		references.Position{X: 9, Y: 9}, NewCombatPCProvider(character))
	require.NotZero(t, clone.Slot)

	return gs, mock, source, clone
}

func TestControlledMonsterTargetsUnpossessedSadujCopies(t *testing.T) {
	for _, useClone := range []bool{false, true} {
		for _, ranged := range []bool{false, true} {
			for _, condition := range []string{"awake", "sleeping", "underground"} {
				t.Run(fmt.Sprintf("clone=%t/ranged=%t/%s", useClone, ranged, condition), func(t *testing.T) {
					gs, mock, source, clone := setupSadujCombatCopies(t)
					target, other := source, clone
					if useClone {
						target, other = clone, source
					}
					attacker, attackerIndex := addControlledTestMonster(gs, references.MonsterMage,
						references.Position{X: 5, Y: 5}, CombatMonsterLoyaltyParty)
					attacker.currentDex = 255
					attacker.enemyRef.AttackRange = 4
					position := references.Position{X: 6, Y: 5}
					if ranged {
						position.X = 8
					}
					for x := references.Coordinate(5); x <= 8; x++ {
						floor := references.Position{X: x, Y: 5}
						gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &floor, indexes.BrickFloor)
						gs.CombatScene.CombatMapRef.SetTileNumber(floor, indexes.BrickFloor)
					}
					gs.CombatScene.setCombatantPosition(target, position)
					gs.setCombatPCSleeping(other, true)
					gs.setCombatPCSleeping(target, condition == "sleeping")
					gs.CombatScene.setCombatPCUnderground(target, condition == "underground")
					require.False(t, gs.CombatScene.isCombatPCPossessed(target))
					authorizeControlledMonster(gs, attackerIndex)
					mock.Reset()
					gs.SetRandomSeed(42)
					hpBefore := gs.PartyState.GetCharacter(target.EntityIndex).GetCurrentHp()

					require.True(t, gs.BeginCombatAttack())
					gs.CombatScene.AttackTargeting.Cursor = position
					wantAim := CombatAimEnemy
					if condition == "underground" {
						wantAim = CombatAimEmpty
					}
					assert.Equal(t, wantAim, gs.GetCombatAimDisposition(position), "preview agrees with actual loyalty and concealment")
					result := gs.ConfirmCombatAttackTarget()
					if condition == "underground" {
						assert.Empty(t, mock.CombatEvents)
						if ranged {
							require.Equal(t, CombatAttackSourcePending, result.Outcome)
							result = gs.ResolvePendingCombatAttack()
							assert.False(t, result.ShowImpact, "underground Saduj is not a missile impact occupant")
						}
						assert.Equal(t, CombatAttackSequenceComplete, result.Outcome)
						assert.Zero(t, gs.GetRNG().Calls())
						assert.Equal(t, hpBefore, gs.PartyState.GetCharacter(target.EntityIndex).GetCurrentHp())
						return
					}
					kind := CombatEventAttack
					if ranged {
						require.Equal(t, CombatAttackSourcePending, result.Outcome)
						result = gs.ResolvePendingCombatAttack()
						kind = CombatEventRangedAttack
					}
					assert.Equal(t, CombatAttackSequenceComplete, result.Outcome)
					require.NotEmpty(t, mock.CombatEvents)
					event := mock.CombatEvents[0]
					assert.Equal(t, kind, event.Kind)
					assert.False(t, event.Diagnostics.ProjectileBlocked)
					assert.Equal(t, CombatHitNormal, event.Diagnostics.Hit.Outcome)
					assert.Equal(t, combatFeedbackIdentity(target, position), event.Target)
					assert.False(t, event.Diagnostics.MagicDamage)
					assert.False(t, gs.CombatScene.isCombatPCSleeping(target))
					assert.True(t, gs.CombatScene.isCombatPCSleeping(other), "only directed tactical copy wakes")
					assert.False(t, gs.CombatScene.isCombatPCPossessed(target))
				})
			}
		}
	}
}
