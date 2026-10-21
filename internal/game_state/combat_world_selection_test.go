package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatWorldSelectionRoundTrips(t *testing.T) {
	for _, entry := range []string{"overworld", "composed", "generic", "direct wanderer", "contact wanderer", "live room", "cleared room"} {
		for _, reason := range []CombatEndReason{CombatVictory, CombatDefeat, CombatEvacuated} {
			for _, selected := range []int{-1, 1} {
				t.Run(fmt.Sprintf("%s/reason_%d/selected_%d", entry, reason, selected), func(t *testing.T) {
					state := enterWorldSelectionTestCombat(t, entry, selected)
					assert.Equal(t, -1, state.GetActivePlayerIndex(), "combat starts in round-robin mode")
					require.True(t, state.SetActivePlayer(0), "combat-local selection may change")
					state.CombatScene.SetSoloMoveCharacter(0)
					state.CombatScene.ExitRoute = CombatExitWest
					state.CombatScene.Victory = reason == CombatVictory
					state.EndCombat(reason)

					require.Nil(t, state.CombatScene)
					assert.Equal(t, selected, state.GetActivePlayerIndex())
					assert.False(t, state.IsSoloMoveMode())
				})
			}
		}
	}
}

func TestCombatWorldSelectionRejectsDeadOrSleepingOriginalMember(t *testing.T) {
	for _, status := range []party_state.CharacterStatus{party_state.Dead, party_state.Sleep} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			state := enterWorldSelectionTestCombat(t, "direct wanderer", 1)
			require.True(t, state.SetActivePlayer(0))
			state.PartyState.GetCharacter(1).SetStatus(status)

			state.EndCombat(CombatVictory)

			assert.Equal(t, -1, state.GetActivePlayerIndex())
			assert.Equal(t, status, state.PartyState.GetCharacter(1).GetStatus(),
				"restoring selection must not change eligibility")
		})
	}
}

func TestCombatWorldSelectionSurvivesContactToRoomContinuation(t *testing.T) {
	state, _, origin, _ := prepareDungeonWandererContactTest(t, 0)
	require.NotNil(t, state)
	require.True(t, state.SetActivePlayer(1))
	roomPosition, moved := applyDungeonCardinalExit(origin, CombatExitEast)
	require.True(t, moved)
	_, err := state.TransformDungeonCell(0, roomPosition, func(references.DungeonCell) references.DungeonCell { return 0xa0 })
	require.NoError(t, err)
	token, _ := state.BeginDungeonCommandCycle()
	result := state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	require.True(t, result.WandererMovement.Contact)
	drainDungeonWandererContactWorkflow(t, state)
	require.True(t, state.SetActivePlayer(0))
	state.CombatScene.ExitRoute = CombatExitEast
	state.EndCombat(CombatVictory)
	require.Equal(t, 1, state.GetActivePlayerIndex())
	require.True(t, state.HasPendingDungeonRoomEncounter())

	require.NoError(t, state.ResolvePendingDungeonRoomEncounter())
	require.True(t, state.SetActivePlayer(2))
	state.CombatScene.ExitRoute = CombatExitEast
	state.EndCombat(CombatVictory)

	assert.Equal(t, 1, state.GetActivePlayerIndex())
	assert.True(t, state.IsAtDungeonCommandBoundary())
}

func TestCombatWorldSelectionRestoredByDirectTransitionAndFlee(t *testing.T) {
	for _, flee := range []bool{false, true} {
		t.Run(fmt.Sprintf("flee_%t", flee), func(t *testing.T) {
			state := enterWorldSelectionTestCombat(t, "generic", 1)
			require.True(t, state.SetActivePlayer(0))
			if flee {
				for index, active := range state.CombatScene.PartyActive {
					if active {
						require.True(t, state.fleePCFromCombat(CombatTurnIdentity{IsPartyMember: true, EntityIndex: index}, CombatExitNorth))
					}
				}
				require.Equal(t, CombatEvacuated, state.checkCombatEndConditions())
			} else {
				state.TransitionFromCombatMap()
			}
			assert.Nil(t, state.CombatScene)
			assert.Equal(t, 1, state.GetActivePlayerIndex())
		})
	}
}

func TestCombatWorldSelectionRestoredWhenPartyFleesHostileWanderer(t *testing.T) {
	for _, entry := range []string{"direct wanderer", "contact wanderer"} {
		t.Run(entry, func(t *testing.T) {
			state := enterWorldSelectionTestCombat(t, entry, 1)
			require.True(t, state.SetActivePlayer(0))
			require.Positive(t, state.CombatScene.MonsterManager.CountActiveHostile())
			for index, active := range state.CombatScene.PartyActive {
				if active {
					require.True(t, state.fleePCFromCombat(CombatTurnIdentity{IsPartyMember: true, EntityIndex: index}, CombatExitWest))
				}
			}

			require.Equal(t, CombatDefeat, state.checkCombatEndConditions())

			assert.Nil(t, state.CombatScene)
			assert.Equal(t, 1, state.GetActivePlayerIndex())
			assert.True(t, state.IsAtDungeonCommandBoundary())
		})
	}
}

func enterWorldSelectionTestCombat(t *testing.T, entry string, selected int) *GameState {
	t.Helper()
	var state *GameState
	switch entry {
	case "overworld", "composed", "generic":
		state, _ = NewIntegrationTestBuilder(t).WithPlayerAt(50, 50).
			WithRandomSeed(638).WithSystemCallbacks().Build()
	case "contact wanderer":
		state, _, _, _ = prepareDungeonWandererContactTest(t, 0)
	default:
		state, _ = newDungeonWandererEncounterTestState(t, 0)
	}
	require.NotNil(t, state)
	require.True(t, state.SetActivePlayer(selected))
	switch entry {
	case "overworld":
		enemy := addOverworldTestEnemy(t, state, references.MonsterOrc, references.Position{X: 51, Y: 50})
		require.NoError(t, state.StartOverworldCombatEncounter(enemy))
	case "composed":
		_, err := state.StartCombatEncounter(newComposerRequest())
		require.NoError(t, err)
	case "generic":
		require.NoError(t, state.TransitionToCombatMap(int(references.GladeCombatMap), references.BritanniaCombatTerritory, CombatEntryNorth))
	case "direct wanderer":
		state.activeDungeon.wanderer = &DungeonWanderer{
			Monster: references.MonsterBat, Floor: 0,
			Position: references.Position{X: 2, Y: 1}, PreviousPosition: references.Position{X: 3, Y: 1},
		}
		token, _ := state.BeginDungeonCommandCycle()
		require.True(t, state.ActionAttackDungeonMap(references.NoneDirection))
		state.CompleteDungeonCommandCycle(token, DungeonCommandOnly)
	case "contact wanderer":
		token, _ := state.BeginDungeonCommandCycle()
		result := state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
		require.True(t, result.WandererMovement.Contact)
		drainDungeonWandererContactWorkflow(t, state)
	case "live room", "cleared room":
		cell := references.DungeonCell(0xf0)
		if entry == "cleared room" {
			cell = 0xa0
		}
		_, err := state.TransformDungeonCell(0, state.MapState.PlayerLocation.Position,
			func(references.DungeonCell) references.DungeonCell { return cell })
		require.NoError(t, err)
		state.synchronizeDungeonRoomEncounterForStablePosition()
		require.NoError(t, state.ResolvePendingDungeonRoomEncounter())
	default:
		t.Fatalf("unknown combat entry %q", entry)
	}
	require.NotNil(t, state.CombatScene)

	return state
}
