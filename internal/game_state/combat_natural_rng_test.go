package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestNaturalEncounterUsesSourceInterleavedRNGInBothWorlds(t *testing.T) {
	for _, floor := range []references.FloorNumber{0, -1} {
		for _, primary := range []references.MonsterType{
			references.MonsterSeaSerpent, references.MonsterGuard, references.MonsterBat,
			references.MonsterRat, references.MonsterOrc, references.MonsterBard,
		} {
			for _, opening := range []bool{false, true} {
				for _, seed := range []uint64{0, 86, 179} {
					t.Run(fmt.Sprintf("floor%d/%s/opening%v/seed%d", floor, primary, opening, seed), func(t *testing.T) {
						state, _ := newOutdoorFireballFixture(t, floor, references.NoPartyVehicle)
						fillOutdoorFireballTerrain(state, indexes.Grass)
						state.openingCampaignEncounterReduction = opening
						state.PartyState.GetCharacter(0).SetEquippedItem(references.SlotRing, references.RingInvisibility)
						source := addOverworldTestEnemy(t, state, primary, references.Position{X: 101, Y: 100})
						if primary == references.MonsterBard {
							state.GetCurrentLargeMapNPCAIController().GetNpcs().RemoveEnemy(source)
							source = addPirateTestEnemy(t, state, references.Position{X: 101, Y: 100}, indexes.PirateShipUp)
						}
						state.SetRandomSeed(seed)
						oracle := beradrng.New(seed)
						// Party ring initialization precedes group size;
						// fixed reference group bound values 1, 8, 16 bypass both count rolls.
						// See ref:ENEMY-0003.
						vanished := oracle.Intn(16) == 11
						ref := (*state.GameReferences.EnemyReferences)[primary]
						count := ref.MaxPerMap
						if count != 1 && count != 8 && count != 16 {
							count = oracle.Intn(min(count, 16)) + 1
							if opening {
								count = oracle.Intn(count) + 1
							}
						}
						friend := primary
						if ref.Friend != nil {
							friend = ref.Friend.MonsterType
						}

						require.NoError(t, state.StartOverworldCombatEncounter(source))

						assert.Equal(t, !vanished, state.CombatScene.PartyInvisible[0])
						assertOriginalEncounterMonsterDraws(t, state, oracle, primary, friend, count, true)
						assertNaturalEncounterQueue(t, state, count)
					})
				}
			}
		}
	}
}

func assertNaturalEncounterQueue(t *testing.T, state *GameState, count int) {
	t.Helper()
	pcs, monsters := state.CombatScene.TurnQueue.CountRemaining()
	assert.Equal(t, 1, pcs)
	assert.Equal(t, count, monsters)
	for index := range count {
		monster := state.CombatScene.MonsterManager.GetMonster(index)
		assert.Equal(t, state.CombatScene.CombatMapRef.MonsterPositions[index], monster.Position())
		entry := state.CombatScene.TurnQueue.getActiveEntryAtSourceSlot(6 + index)
		require.NotNil(t, entry)
		assert.Same(t, monster, entry.Combatant)
		assert.Equal(t, index, entry.EntityIndex)
		assert.Equal(t, 36-monster.CurrentDex(), entry.turnCountdown)
	}
}

func TestNaturalEncounterStaticFailureDoesNotRollRingsOrComposition(t *testing.T) {
	for _, floor := range []references.FloorNumber{0, -1} {
		t.Run(fmt.Sprintf("floor%d", floor), func(t *testing.T) {
			state, _ := newOutdoorFireballFixture(t, floor, references.NoPartyVehicle)
			state.PartyState.GetCharacter(0).SetEquippedItem(references.SlotRing, references.RingInvisibility)
			source := addOverworldTestEnemy(t, state, references.MonsterOrc, references.Position{X: 101, Y: 100})
			// Use a detached reference table: never corrupt another fixture's data.
			refs := append(references.EnemyReferences(nil), (*state.GameReferences.EnemyReferences)...)
			refs[references.MonsterOrc].MaxPerMap = 0
			gameRefs := *state.GameReferences
			gameRefs.EnemyReferences = &refs
			state.GameReferences = &gameRefs
			before := state.MapState.PlayerLocation
			state.SetRandomSeed(86)

			require.ErrorIs(t, state.StartOverworldCombatEncounter(source), ErrInvalidOverworldEncounter)
			assert.Zero(t, state.GetRNG().Calls())
			assert.Equal(t, beradrng.New(86).Intn(1000000), state.GetRNG().Intn(1000000))
			assert.Equal(t, references.RingInvisibility, state.PartyState.GetCharacter(0).GetEquippedItem(references.SlotRing))
			assert.Equal(t, before, state.MapState.PlayerLocation)
			assert.Nil(t, state.CombatScene)
		})
	}
}
