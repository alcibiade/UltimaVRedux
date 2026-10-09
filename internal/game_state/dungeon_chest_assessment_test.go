package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestDungeonChestFirstInspectionExactChecksAndSafetyBits(t *testing.T) {
	state, _ := newDungeonInteractionTestState(t, 0x40, 655)
	position := state.MapState.PlayerLocation.Position
	actor := state.PartyState.GetCharacter(0)
	actor.Dexterity, actor.Intelligence = 20, 30
	for _, floor := range []references.FloorNumber{0, 4, 7} {
		state.MapState.PlayerLocation.Floor = floor
		for _, raw := range []references.DungeonCell{0x40, 0x48, 0x41, 0x4f} {
			for _, roll := range []int{5 + int(floor), 6 + int(floor)} {
				t.Run(fmt.Sprintf("floor=%d/raw=%x/roll=%d", floor, raw, roll), func(t *testing.T) {
					setDungeonInteractionCell(t, state, position, 0x00)
					setDungeonInteractionCell(t, state, position, raw)
					state.SetRandomSeed(chestSeedForRoll(t, roll))
					message := state.searchDungeonChest(position, raw, actor)
					want := references.ChestUncertain
					if roll > 5+int(floor) {
						want = references.ChestSafe
						if raw.Raw()&7 != 0 {
							want = references.ChestTrapped
						}
					}
					assert.Equal(t, want, state.activeDungeon.ChestAssessment(floor, position))
					assert.Equal(t, uint64(1), state.GetRNG().Calls(), "no fabricated severity roll")
					assert.Equal(t, message, state.searchDungeonChest(position, raw, state.PartyState.GetCharacter(1)))
					assert.Equal(t, uint64(1), state.GetRNG().Calls())
					assert.Equal(t, raw, currentDungeonInteractionCell(t, state))
					if want == references.ChestSafe {
						assert.Equal(t, chestSafeMessage, message)
					}
				})
			}
		}
	}
}

func TestDungeonChestUncertainJimmyAndOpenSequences(t *testing.T) {
	for _, armed := range []bool{false, true} {
		for _, disarm := range []bool{false, true} {
			t.Run(fmt.Sprintf("armed=%t/disarm=%t", armed, disarm), func(t *testing.T) {
				raw := references.DungeonCell(0x48)
				if armed {
					raw = 0x49
				}
				state, callbacks := newDungeonInteractionTestState(t, raw, 655)
				position := state.MapState.PlayerLocation.Position
				actor := state.PartyState.GetCharacter(0)
				actor.Dexterity = 1
				state.SetRandomSeed(chestSeedForRoll(t, 1))
				runDungeonInteraction(t, state, func() {
					calls := state.GetRNG().Calls()
					require.True(t, state.ActionSearchDungeonMapWithCharacter(references.Down, 0))
					assert.Equal(t, calls+1, state.GetRNG().Calls())
				})
				require.Equal(t, references.ChestUncertain, state.activeDungeon.ChestAssessment(0, position))
				state.PartyState.Inventory.Provisions.Keys.Set(2)
				for attempt := range 2 {
					state.SetRandomSeed(chestSeedForRoll(t, 1))
					runDungeonInteraction(t, state, func() {
						assert.False(t, state.ActionJimmyDungeonMap(0))
						assert.Equal(t, uint16(1-attempt), state.PartyState.Inventory.Provisions.Keys.Get())
						assert.Equal(t, references.ChestUncertain, state.activeDungeon.ChestAssessment(0, position))
						assert.Equal(t, raw, currentDungeonInteractionCell(t, state))
					})
				}
				runDungeonInteraction(t, state, func() {
					calls := state.GetRNG().Calls()
					assert.False(t, state.ActionJimmyDungeonMap(0))
					assert.Equal(t, calls, state.GetRNG().Calls())
				})
				runDungeonInteraction(t, state, func() {
					calls := state.GetRNG().Calls()
					assert.True(t, state.ActionSearchDungeonMapWithCharacter(references.Down, 0))
					assert.Equal(t, calls, state.GetRNG().Calls())
					assert.Equal(t, references.ChestUncertain, state.activeDungeon.ChestAssessment(0, position))
				})
				if disarm && armed {
					actor.Dexterity = 30
					state.PartyState.Inventory.Provisions.Keys.Set(1)
					runDungeonInteraction(t, state, func() {
						require.True(t, state.ActionJimmyDungeonMap(0))
						assert.Equal(t, uint16(1), state.PartyState.Inventory.Provisions.Keys.Get())
						assert.Equal(t, references.ChestSafe, state.activeDungeon.ChestAssessment(0, position))
					})
					state.PartyState.Inventory.Provisions.Keys.Set(0)
					runDungeonInteraction(t, state, func() {
						calls := state.GetRNG().Calls()
						assert.False(t, state.ActionJimmyDungeonMap(0))
						assert.Equal(t, calls, state.GetRNG().Calls())
					})
					assert.Contains(t, callbacks.Messages, chestAlreadySafeMessage)
				}
				assert.NotContains(t, callbacks.Messages, "Trapped!")
				state.SetRandomSeed(findDungeonTrapSeed(t, state, TrapAcid))
				callbacks.Reset()
				runDungeonInteraction(t, state, func() { require.True(t, state.ActionOpenDungeonMap(0)) })
				assert.Equal(t, references.DungeonCell(0x78), currentDungeonInteractionCell(t, state))
				assert.Equal(t, references.ChestUninspected, state.activeDungeon.ChestAssessment(0, position))
				assert.Equal(t, armed && !disarm, stringIndex(callbacks.Messages, "ACID!") >= 0)
			})
		}
	}
}

func TestDungeonChestKnowledgeLifetimeAndRestoreValidation(t *testing.T) {
	state, _ := newDungeonInteractionTestState(t, 0x48, 655)
	dungeon := state.activeDungeon
	position := state.MapState.PlayerLocation.Position
	dungeon.SetChestAssessment(0, position, references.ChestSafe)
	snapshot := dungeon.Snapshot()
	snapshot.ChestAssessments[0][position.Y][position.X] = references.ChestTrapped
	assert.Equal(t, references.ChestSafe, dungeon.ChestAssessment(0, position), "snapshot is detached")
	require.Error(t, dungeon.RestoreChestAssessments(snapshot.ChestAssessments))
	assert.Equal(t, references.ChestSafe, dungeon.ChestAssessment(0, position), "restore is atomic")
	snapshot.ChestAssessments[0][position.Y][position.X] = references.ChestUncertain
	require.NoError(t, dungeon.RestoreChestAssessments(snapshot.ChestAssessments))
	assert.Equal(t, references.ChestUncertain, dungeon.ChestAssessment(0, position))
	state.MapState.PlayerLocation.Floor = 1
	state.MapState.PlayerLocation.Floor = 0
	state.installPreparedDungeonState(dungeon)
	assert.Equal(t, references.ChestUncertain, state.activeDungeon.ChestAssessment(0, position))
	setDungeonInteractionCell(t, state, position, 0x40)
	assert.Equal(t, references.ChestUncertain, dungeon.ChestAssessment(0, position), "ceiling bit is not a new chest")
	setDungeonInteractionCell(t, state, position, 0x41)
	assert.Equal(t, references.ChestUninspected, dungeon.ChestAssessment(0, position), "rearming clears stale knowledge")
	dungeon.SetChestAssessment(0, position, references.ChestTrapped)
	state.SetRandomSeed(655)
	result := state.CastDungeonPackedSpell(references.AnSanct)
	require.True(t, result.Applied)
	assert.Equal(t, references.DungeonCell(0x70), currentDungeonInteractionCell(t, state))
	assert.Equal(t, references.ChestUninspected, dungeon.ChestAssessment(0, position))
	assert.Zero(t, state.GetRNG().Calls())
	assert.Panics(t, func() { dungeon.SetChestAssessment(0, position, references.ChestSafe) })
	setDungeonInteractionCell(t, state, position, 0x40)
	assert.Equal(t, references.ChestUninspected, dungeon.ChestAssessment(0, position), "new chest at old position")
	dungeon.SetChestAssessment(0, position, references.ChestSafe)
	fresh, err := state.PrepareDungeonState(references.Deceit, references.DungeonFacingEast, references.DungeonArrivedFromAbove)
	require.NoError(t, err)
	assert.Equal(t, references.ChestUninspected, fresh.ChestAssessment(0, position))
	assert.Panics(t, func() { dungeon.SetChestAssessment(-1, position, references.ChestSafe) })
	assert.Panics(t, func() { dungeon.SetChestAssessment(0, position, references.ChestAssessment(4)) })
}

func TestDungeonChestAssessmentSurvivesCombatReturn(t *testing.T) {
	state, _ := newDungeonWandererEncounterTestState(t, 0)
	position := references.Position{X: 3, Y: 3}
	setDungeonInteractionCell(t, state, position, 0x41)
	state.activeDungeon.SetChestAssessment(0, position, references.ChestUncertain)
	state.activeDungeon.wanderer = &DungeonWanderer{
		Monster: references.MonsterRat, Floor: 0,
		Position: references.Position{X: 2, Y: 1}, PreviousPosition: references.Position{X: 2, Y: 2},
	}
	dungeon := state.activeDungeon
	token, _ := state.BeginDungeonCommandCycle()
	require.True(t, state.ActionAttackDungeonMap(references.NoneDirection))
	state.CompleteDungeonCommandCycle(token, DungeonCommandOnly)
	require.True(t, state.IsInCombat())
	state.EndCombat(CombatVictory)
	assert.Same(t, dungeon, state.activeDungeon)
	assert.Equal(t, references.ChestUncertain, state.activeDungeon.ChestAssessment(0, position))
}

func TestDungeonChestBlindJimmyDoesNotInspectAndDarknessDoesNotRoll(t *testing.T) {
	state, callbacks := newDungeonInteractionTestState(t, 0x48, 655)
	position := state.MapState.PlayerLocation.Position
	state.MapState.Lighting.RestoreTorchMinutes(0)
	state.MapState.Lighting.SetMagicLight(false)
	runDungeonInteraction(t, state, func() {
		calls := state.GetRNG().Calls()
		assert.False(t, state.ActionSearchDungeonMapWithCharacter(references.Down, 0))
		assert.Equal(t, calls, state.GetRNG().Calls())
	})
	assert.Equal(t, references.ChestUninspected, state.activeDungeon.ChestAssessment(0, position))
	state.PartyState.Inventory.Provisions.Keys.Set(0)
	runDungeonInteraction(t, state, func() { assert.False(t, state.ActionJimmyDungeonMap(0)) })
	assert.Equal(t, references.ChestUninspected, state.activeDungeon.ChestAssessment(0, position))
	state.PartyState.Inventory.Provisions.Keys.Set(1)
	runDungeonInteraction(t, state, func() {
		calls := state.GetRNG().Calls()
		assert.False(t, state.ActionJimmyDungeonMap(0))
		assert.Equal(t, calls, state.GetRNG().Calls())
	})
	assert.Zero(t, state.PartyState.Inventory.Provisions.Keys.Get())
	assert.Equal(t, references.ChestUninspected, state.activeDungeon.ChestAssessment(0, position))
	state.MapState.Lighting.RestoreTorchMinutes(10)
	runDungeonInteraction(t, state, func() {
		calls := state.GetRNG().Calls()
		require.True(t, state.ActionSearchDungeonMapWithCharacter(references.Down, 0))
		assert.Equal(t, calls+1, state.GetRNG().Calls())
	})
	assert.Equal(t, references.ChestSafe, state.activeDungeon.ChestAssessment(0, position))
	callbacks.Reset()
	runDungeonInteraction(t, state, func() {
		calls := state.GetRNG().Calls()
		assert.False(t, state.ActionJimmyDungeonMap(0))
		assert.Equal(t, calls, state.GetRNG().Calls())
	})
	assert.Contains(t, callbacks.Messages, chestAlreadySafeMessage)
}
