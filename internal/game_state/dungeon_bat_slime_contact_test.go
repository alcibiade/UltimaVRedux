package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// Original dungeon rules gives bats and slimes the same random cardinal movement,
// terrain gates, previous-square rejection, and post-movement contact test.
// See ref:DNG-0010.
func TestDungeonBatSlimeControlledMovementComparison(t *testing.T) {
	for _, test := range []struct {
		name        string
		draws       []int
		blockNorth  bool
		previous    references.Position
		wantMoved   bool
		wantContact bool
		wantCalls   int
	}{
		{name: "onto party", draws: []int{0}, previous: references.Position{X: 3, Y: 3}, wantMoved: true, wantContact: true, wantCalls: 1},
		{name: "away despite adjacency", draws: []int{1}, previous: references.Position{X: 3, Y: 3}, wantMoved: true, wantCalls: 1},
		{name: "all eight terrain attempts fail", draws: []int{0, 0, 0, 0, 0, 0, 0, 0}, blockNorth: true, previous: references.Position{X: 3, Y: 3}, wantCalls: 8},
		{name: "previous square rejected then contact", draws: []int{1, 0, 0}, previous: references.Position{X: 4, Y: 3}, wantMoved: true, wantContact: true, wantCalls: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			var batResult DungeonWandererMovementResult
			for _, monster := range []references.MonsterType{references.MonsterBat, references.MonsterSlime} {
				var cells references.DungeonCells
				if test.blockNorth {
					require.True(t, cells.SetCell(0, references.Position{X: 3, Y: 2}, 0xb0))
				}
				state := newDungeonWandererMovementTestState(t, cells, monster, references.Position{X: 3, Y: 3}, test.previous)
				state.MapState.PlayerLocation.Position = references.Position{X: 2, Y: 3}
				if !test.blockNorth {
					state.MapState.PlayerLocation.Position = references.Position{X: 3, Y: 2}
				}
				random := newScriptedDungeonRandom(t, test.draws...)
				result := state.moveDungeonWandererWithRandom(random.next)
				random.requireExhausted()
				assert.Equal(t, test.wantMoved, result.Moved, "%s", monster)
				assert.Equal(t, test.wantContact, result.Contact, "%s", monster)
				assert.Equal(t, test.wantCalls, random.index)
				assert.NotEqual(t, state.MapState.PlayerLocation.Position, state.activeDungeon.wanderer.Position,
					"contact rewinds; ordinary movement cannot create overlap")
				result.Monster = references.MonsterNone
				if monster == references.MonsterBat {
					batResult = result
				} else {
					assert.Equal(t, batResult, result, "species must not change movement or contact")
				}
			}
		})
	}
}

// Moving into a dungeon creature blocks the party step but still runs
// the actor-completion phase. Subsequent creature movement, rather than the
// initial bump, can initiate contact combat. ref:DNG-0010
func TestDungeonBatSlimeProductionContactAfterPassAndBlockedStep(t *testing.T) {
	for _, monster := range []references.MonsterType{references.MonsterBat, references.MonsterSlime} {
		for _, blockedStep := range []bool{false, true} {
			for _, suppression := range []string{"none", "time stop", "quickness"} {
				t.Run(fmt.Sprintf("%s/blocked_%t/%s", monster, blockedStep, suppression), func(t *testing.T) {
					state, callbacks, party, origin := prepareDungeonWandererContactTest(t, 0)
					require.NotNil(t, state)
					state.activeDungeon.wanderer.Monster = monster
					facing, ok := dungeonWandererAttackFacing(party, origin)
					require.True(t, ok)
					state.activeDungeon.setFacingAndLastDirection(facing)
					if suppression != "none" {
						if suppression == "time stop" {
							state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectTimeStop, TurnsRemaining: 1}
						} else {
							state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectQuickness, TurnsRemaining: 3}
							state.activeDungeon.quicknessSkipNext = true
						}
						before := state.GetRNG().Calls()
						result := completeBatSlimeTestCommand(t, state, blockedStep)
						assert.False(t, result.WandererMovement.Contact)
						assert.Equal(t, before, state.GetRNG().Calls(), "suppressed actors consume no movement RNG")
						assert.Nil(t, state.CombatScene)
						assert.False(t, state.EventQueueManager.HasWorkflow(dungeonWandererContactWorkflowName))
					}

					result := completeBatSlimeTestCommand(t, state, blockedStep)

					require.True(t, result.WandererMovement.Contact)
					assert.Equal(t, 1, result.WandererMovement.Attempts)
					assert.Equal(t, origin, result.WandererMovement.AttackOrigin)
					assert.Equal(t, party, state.MapState.PlayerLocation.Position)
					assert.Equal(t, origin, state.activeDungeon.wanderer.Position)
					require.True(t, state.EventQueueManager.HasWorkflow(dungeonWandererContactWorkflowName))
					drainDungeonWandererContactWorkflow(t, state)
					require.NotNil(t, state.CombatScene)
					assert.Equal(t, DungeonWandererEncounterContact, state.CombatScene.DungeonWandererOrigin.Kind)
					assert.Equal(t, monster, state.CombatScene.MonsterManager.GetMonster(0).MonsterType())
					assert.Contains(t, callbacks.Messages, "Attacked!")
					state.EndCombat(CombatVictory)
					assert.True(t, state.IsAtDungeonCommandBoundary())
				})
			}
		}
	}
}

func completeBatSlimeTestCommand(t *testing.T, state *GameState, blockedStep bool) DungeonCommandCycleResult {
	t.Helper()
	token, _ := state.BeginDungeonCommandCycle()
	if blockedStep {
		before := state.MapState.PlayerLocation.Position
		movement := state.TryDungeonAdvance()
		require.Equal(t, DungeonMoveWandererBlocked, movement.Outcome)
		require.Equal(t, before, state.MapState.PlayerLocation.Position)
		require.Equal(t, DungeonFullEndPlayer, movement.Completion)
	} else {
		// Same path as the dungeon Space handler: prompt, then full end-player.
		state.SystemCallbacks.Message.ShowCommandPrompt("Pass")
	}

	return state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
}
