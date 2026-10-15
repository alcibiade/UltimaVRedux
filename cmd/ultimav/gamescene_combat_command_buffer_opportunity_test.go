package main

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

// Fixture adapter for a live enemy DEX change. All other eligibility reads stay
// backed by the real monster; no scheduling or action is simulated by the test.
type bufferedInputDexProvider struct {
	game_state.CombatantProvider
	dexterity int
}

func (provider *bufferedInputDexProvider) GetDexterity() int { return provider.dexterity }

func TestBufferedCombatOpportunitySurvivesEnemyChangesThroughUpdate(t *testing.T) {
	for _, change := range []string{"enemy killed", "enemy removed", "enemy DEX changed", "Time Stop"} {
		t.Run(change, func(t *testing.T) {
			scene, held := newBufferedCombatUpdateScene(t)
			state := scene.gameState
			combat := state.CombatScene
			actor := game_state.CombatTurnIdentity{IsPartyMember: true, EntityIndex: combat.GetSoloMoveCharacter()}
			character := state.PartyState.GetCharacter(actor.EntityIndex)
			character.Dexterity = 30
			character.SetMaxHp(1000)
			character.SetHp(1000)
			selected := combat.TurnQueue.GetEntryByIdentity(actor)
			combat.TurnQueue.ReplaceCombatant(actor, actor, selected.Combatant)
			// The new enemy is due before the selected PC, so deleting/slowing it
			// changes the number of intervening actors rather than an irrelevant tail.
			ref := (*state.GameReferences.EnemyReferences)[references.MonsterRat]
			ref.Dexterity = 35
			secondEnemy := game_state.NewCombatMonster(references.MonsterRat, &ref,
				references.Position{X: 10, Y: 9}, state.AsRandomProvider())
			secondIndex := combat.AddMonsterToQueue(secondEnemy)
			secondIdentity := game_state.CombatTurnIdentity{EntityIndex: secondIndex}
			positionBefore := secondEnemy.Position()
			randomBefore, dateBefore := state.GetRNG().Calls(), state.DateTime
			held[ebiten.KeyG] = true
			require.NoError(t, scene.Update(nil))
			require.True(t, scene.transientCommandInput.pending.active)
			intended := scene.transientCommandInput.pending.opportunity
			require.Equal(t, randomBefore, state.GetRNG().Calls(), "buffer capture consumes no gameplay RNG")
			require.Equal(t, dateBefore, state.DateTime)
			switch change {
			case "enemy killed":
				secondEnemy.SetCurrentHP(0)
				combat.TurnQueue.MarkDead(secondIndex, false)
			case "enemy removed":
				combat.TurnQueue.MarkFled(secondIndex, false)
				require.Same(t, secondEnemy, combat.MonsterManager.RemoveMonster(secondIndex))
			case "enemy DEX changed":
				combat.TurnQueue.ReplaceCombatant(secondIdentity, secondIdentity,
					&bufferedInputDexProvider{CombatantProvider: secondEnemy, dexterity: 1})
			case "Time Stop":
				require.True(t, state.ActivateTimeStopEffect(100))
			}
			require.NoError(t, scene.Update(nil))
			require.True(t, scene.transientCommandInput.pending.active)
			require.Equal(t, intended, scene.transientCommandInput.pending.opportunity)
			for range 2000 {
				state.ElapsedMs += 16
				require.NoError(t, scene.Update(nil))
				if scene.secondaryKeyState == GetDirectionInput {
					break
				}
				require.True(t, scene.transientCommandInput.pending.active, "intervening enemy changes must not discard Get")
				require.Equal(t, intended, scene.transientCommandInput.pending.opportunity)
			}
			require.Equal(t, GetDirectionInput, scene.secondaryKeyState)
			require.False(t, scene.transientCommandInput.pending.active)
			ready, eligible := state.ProjectSoloCombatInput()
			require.True(t, eligible)
			require.True(t, ready.Ready)
			require.Equal(t, intended, ready.Opportunity, "acceptance belongs to the actually reached selected-entry generation")
			if change == "Time Stop" {
				require.Equal(t, positionBefore, secondEnemy.Position())
				snapshot, found := combat.TurnQueue.GetEntrySnapshot(secondIdentity)
				require.True(t, found)
				require.False(t, snapshot.CanTakeTurn, "the intervening enemy's actions were suppressed")
			}
			// Cancel freely, then pay one ordinary Pass. The still-held original
			// G must not refill the slot or execute on a later actor opportunity.
			held[ebiten.KeyEscape] = true
			require.NoError(t, scene.Update(nil))
			require.Equal(t, PrimaryInput, scene.secondaryKeyState)
			delete(held, ebiten.KeyEscape)
			held[ebiten.KeySpace] = true
			require.NoError(t, scene.Update(nil))
			delete(held, ebiten.KeySpace)
			var next game_state.CombatInputProjection
			for range 2000 {
				state.ElapsedMs += 16
				require.NoError(t, scene.Update(nil))
				require.Equal(t, PrimaryInput, scene.secondaryKeyState)
				require.False(t, scene.transientCommandInput.pending.active)
				next, eligible = state.ProjectSoloCombatInput()
				if eligible && next.Ready && !scene.inputBlocked {
					break
				}
			}
			require.True(t, eligible)
			require.True(t, next.Ready)
			require.NotEqual(t, intended, next.Opportunity)
		})
	}
}
