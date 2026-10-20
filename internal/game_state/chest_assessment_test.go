package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func chestSeedForRoll(t *testing.T, roll int) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 10000; seed++ {
		if beradrng.New(seed).Intn(30)+1 == roll {
			return seed
		}
	}
	t.Fatal("no seed found for chest roll")
	return 0
}

func TestChestFirstObjectInspectionExactThresholds(t *testing.T) {
	for _, armed := range []bool{false, true} {
		for _, intelligence := range []int{1, 20, 30} {
			t.Run(fmt.Sprintf("armed=%t/INT=%d", armed, intelligence), func(t *testing.T) {
				state := &GameState{}
				conclusive := 0
				for roll := 1; roll <= 30; roll++ {
					state.SetRandomSeed(chestSeedForRoll(t, roll))
					got := state.inspectObjectChest(references.ChestUninspected, armed, 30, intelligence)
					difficulty := (30 - intelligence) / 2
					if armed {
						difficulty = (60 - intelligence) / 2
					}
					want := references.ChestUncertain
					if roll >= difficulty {
						conclusive++
						want = references.ChestSafe
						if armed {
							want = references.ChestTrapped
						}
					}
					require.Equal(t, want, got, "roll %d", roll)
					require.Equal(t, uint64(1), state.GetRNG().Calls())
					assert.Equal(t, got, state.inspectObjectChest(got, armed, 30, 255))
					assert.Equal(t, uint64(1), state.GetRNG().Calls(), "no repeat inspection roll")
				}
				if armed && intelligence == 20 {
					assert.Equal(t, 11, conclusive)
				}
				if armed && intelligence == 30 {
					assert.Equal(t, 16, conclusive)
				}
			})
		}
	}
}

func TestSurfaceChestUncertainSequences(t *testing.T) {
	for _, location := range []references.Location{references.LordBritishsCastle, references.BritanniaUnderworld} {
		for _, armed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/armed=%t", location, armed), func(t *testing.T) {
				state, callbacks := NewIntegrationTestBuilder(t).WithLocation(location).WithFloor(references.MainFloor).
					WithPlayerAt(15, 15).WithSystemCallbacks().Build()
				target := references.Position{X: 16, Y: 15}
				chest := map_units.NewNPCFriendlySurfaceChest(location, target, references.MainFloor, 30)
				chest.SetPos(target)
				chest.SetFloor(references.MainFloor)
				*state.CurrentNPCAIController.GetNpcs() = map_units.MapUnits{chest}
				number := byte(30)
				if armed {
					number |= 0x80
				}
				chest.RestoreSurfaceChestNumber(number)
				actor := state.PartyState.GetCharacter(0)
				actor.Intelligence, actor.Dexterity = 1, 1
				state.PartyState.Inventory.Provisions.Keys.Set(3)
				state.SetRandomSeed(chestSeedForRoll(t, 1))
				response := state.SearchManager.checkChests(&target, 0)
				require.Equal(t, chestUncertainMessage, response.Message)
				assert.Equal(t, references.ChestUncertain, chest.SurfaceChestAssessment())
				assert.Equal(t, uint64(1), state.GetRNG().Calls())
				for attempt := range 3 {
					state.SetRandomSeed(chestSeedForRoll(t, 1))
					assert.Equal(t, response, state.SearchManager.checkChests(&target, 1))
					assert.Zero(t, state.GetRNG().Calls())
					assert.Equal(t, JimmyBrokenPick, state.JimmyDoor(references.Right, actor))
					assert.Equal(t, uint16(2-attempt), state.PartyState.Inventory.Provisions.Keys.Get())
					assert.Equal(t, armed, chest.SurfaceChestLocked())
					assert.Equal(t, references.ChestUncertain, chest.SurfaceChestAssessment())
					wantRolls := uint64(0)
					if armed {
						wantRolls = 1
					}
					assert.Equal(t, wantRolls, state.GetRNG().Calls())
					assert.NotContains(t, callbacks.Messages, "Trapped!")
				}
				calls := state.GetRNG().Calls()
				assert.Equal(t, JimmyDoorNoLockPicks, state.JimmyDoor(references.Right, actor))
				assert.Equal(t, calls, state.GetRNG().Calls())
				assert.Equal(t, references.ChestUncertain, chest.SurfaceChestAssessment())
				state.SetRandomSeed(findDungeonTrapSeed(t, state, TrapAcid))
				callbacks.Reset()
				if location == references.LordBritishsCastle {
					require.True(t, state.ActionOpenSmallMapWithCharacter(references.Right, actor))
				} else {
					require.True(t, state.ActionOpenLargeMapWithCharacter(references.Right, actor))
				}
				assert.Nil(t, state.surfaceChestAt(&target))
				assert.Equal(t, armed, stringIndex(callbacks.Messages, "Trapped!") >= 0)
			})
		}
	}
}

func TestSurfaceChestDisarmingAndKnownSafeRefusal(t *testing.T) {
	for _, magic := range []bool{false, true} {
		t.Run(fmt.Sprintf("magic=%t", magic), func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.LordBritishsCastle).WithSystemCallbacks().Build()
			chest, direction := positionBesideSurfaceChest(t, state)
			target := direction.GetNewPositionInDirection(&state.MapState.PlayerLocation.Position)
			chest.RestoreSurfaceChestNumber(lockedSurfaceChestNumber)
			actor := state.PartyState.GetCharacter(0)
			actor.Intelligence, actor.Dexterity = 1, 30
			state.PartyState.Inventory.Provisions.Keys.Set(1)
			state.SetRandomSeed(chestSeedForRoll(t, 1))
			state.SearchManager.checkChests(target, 0)
			require.Equal(t, references.ChestUncertain, chest.SurfaceChestAssessment())
			state.SetRandomSeed(chestSeedForRoll(t, 30))
			if magic {
				require.Equal(t, AnSanctChestDisarmed, state.CastAnSanct(target))
				assert.Zero(t, state.GetRNG().Calls())
			} else {
				require.Equal(t, JimmyChestUnlocked, state.JimmyDoor(direction, actor))
				assert.Equal(t, uint64(1), state.GetRNG().Calls())
			}
			assert.False(t, chest.SurfaceChestLocked())
			assert.Equal(t, references.ChestSafe, chest.SurfaceChestAssessment())
			assert.Equal(t, uint16(1), state.PartyState.Inventory.Provisions.Keys.Get())
			calls := state.GetRNG().Calls()
			assert.Equal(t, chestSafeMessage, state.SearchManager.checkChests(target, 1).Message)
			assert.Equal(t, JimmyChestAlreadySafe, state.JimmyDoor(direction, actor))
			assert.Equal(t, uint16(1), state.PartyState.Inventory.Provisions.Keys.Get())
			state.PartyState.Inventory.Provisions.Keys.Set(0)
			assert.Equal(t, JimmyChestAlreadySafe, state.JimmyDoor(direction, nil))
			assert.Equal(t, calls, state.GetRNG().Calls())
			callbacks.Reset()
			assert.False(t, state.ActionJimmySmallMapWithCharacter(direction, nil))
			assert.Equal(t, []string{chestAlreadySafeMessage}, callbacks.Messages)
			assert.Equal(t, []int{DefaultSmallMapMinutesPerTurn}, callbacks.TimeAdvanced())
			callbacks.Reset()
			require.True(t, state.ActionOpenSmallMapWithCharacter(direction, actor))
			assert.NotContains(t, callbacks.Messages, "Trapped!")
		})
	}
}

func TestSurfaceChestBlindFailurePreservesFirstInspection(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.LordBritishsCastle).WithSystemCallbacks().Build()
	chest, direction := positionBesideSurfaceChest(t, state)
	target := direction.GetNewPositionInDirection(&state.MapState.PlayerLocation.Position)
	actor := state.PartyState.GetCharacter(0)
	actor.Intelligence = 30
	state.PartyState.Inventory.Provisions.Keys.Set(0)
	assert.Equal(t, JimmyDoorNoLockPicks, state.JimmyDoor(direction, actor))
	assert.Equal(t, references.ChestUninspected, chest.SurfaceChestAssessment())
	state.PartyState.Inventory.Provisions.Keys.Set(1)
	assert.Equal(t, JimmyBrokenPick, state.JimmyDoor(direction, actor))
	assert.Equal(t, references.ChestUninspected, chest.SurfaceChestAssessment())
	state.SetRandomSeed(655)
	assert.Equal(t, chestSafeMessage, state.SearchManager.checkChests(target, 0).Message)
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
}

func TestSurfaceChestKnowledgeFollowsObjectLifetime(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.LordBritishsCastle).WithSystemCallbacks().Build()
	chest, direction := positionBesideSurfaceChest(t, state)
	target := direction.GetNewPositionInDirection(&state.MapState.PlayerLocation.Position)
	npcNumber := chest.MapUnitDetails().NPCNum
	floor := chest.Floor()
	chest.RestoreSurfaceChestAssessment(references.ChestUncertain)
	state.reloadCurrentSmallMapTerrain()
	assert.Same(t, chest, state.surfaceChestAt(target))
	state.MapState.PlayerLocation.Floor = references.Basement
	state.reloadCurrentSmallMapTerrain()
	state.MapState.PlayerLocation.Floor = floor
	state.reloadCurrentSmallMapTerrain()
	assert.Equal(t, references.ChestUncertain, state.surfaceChestAt(target).SurfaceChestAssessment())
	require.NoError(t, state.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth))
	state.EndCombat(CombatVictory)
	assert.Equal(t, references.ChestUncertain, surfaceChestByNumber(t, state, npcNumber).SurfaceChestAssessment())
	state.UpdateSmallMap(state.GameReferences.TileReferences, state.GameReferences.LocationReferences)
	assert.Equal(t, references.ChestUncertain, surfaceChestByNumber(t, state, npcNumber).SurfaceChestAssessment())
}

func TestLargeMapChestAnSanctUsesFloorAwareAuthority(t *testing.T) {
	for _, floor := range []references.FloorNumber{references.FloorNumber(references.OVERWORLD), references.FloorNumber(references.UNDERWORLD)} {
		t.Run(fmt.Sprintf("floor=%d", floor), func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
				WithFloor(floor).WithPlayerAt(15, 15).WithSystemCallbacks().Build()
			target := references.Position{X: 16, Y: 15}
			chest := map_units.NewNPCFriendlySurfaceChest(references.BritanniaUnderworld, target, floor, 30)
			chest.SetPos(target)
			chest.SetFloor(floor)
			chest.RestoreSurfaceChestNumber(lockedSurfaceChestNumber)
			*state.CurrentNPCAIController.GetNpcs() = map_units.MapUnits{chest}
			state.SetRandomSeed(655)
			require.Equal(t, AnSanctChestDisarmed, state.CastAnSanct(&target))
			assert.False(t, chest.SurfaceChestLocked())
			assert.Equal(t, references.ChestSafe, chest.SurfaceChestAssessment())
			assert.Equal(t, chestSafeMessage, state.SearchManager.checkChests(&target, 0).Message)
			assert.Zero(t, state.GetRNG().Calls())
			callbacks.Reset()
			require.True(t, state.ActionOpenLargeMapWithCharacter(references.Right, state.PartyState.GetCharacter(0)))
			assert.NotContains(t, callbacks.Messages, "Trapped!")
		})
	}
}
