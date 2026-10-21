package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestYewBasementActionAttackUsesProductionRatsAndCompletesReturnTurn(t *testing.T) {
	for _, slot := range []int{15, 16, 17} {
		t.Run(fmt.Sprint(slot), func(t *testing.T) {
			state, callbacks, source := authoredSmallMapCombatFixture(t, references.Yew, slot)
			require.Equal(t, references.FloorNumber(-1), source.Floor())
			player, direction, ok := emptyAdjacentAttackPosition(state.CurrentNPCAIController.GetNpcs(), source)
			require.True(t, ok)
			state.MapState.PlayerLocation.Position = player
			location := state.MapState.SavePlayerLocation()
			before := state.DateTime
			state.PartyState.Karma.Value = 60
			state.SetCriminalStatus(false)
			require.True(t, state.ActionAttackSmallMap(direction))
			require.NotNil(t, state.CombatScene)
			origin := state.CombatScene.SmallMapOrigin
			assert.Same(t, source, origin.SourceNPC)
			assert.Equal(t, references.MonsterRat, origin.MonsterType)
			assert.Equal(t, SmallMapEncounterAttack, origin.Cause)
			assert.Equal(t, before, state.DateTime)
			assert.Empty(t, callbacks.TimeAdvanced())
			assert.Equal(t, 60, state.PartyState.Karma.Value)
			assert.False(t, state.IsCriminal())
			assert.False(t, state.PartyState.IsNPCDead(references.Yew, slot))
			if slot == 16 {
				assert.Equal(t, 1, state.CombatScene.MonsterManager.Count())
			} else {
				assert.Greater(t, state.CombatScene.MonsterManager.Count(), 1)
				assert.LessOrEqual(t, state.CombatScene.MonsterManager.Count(), 10)
			}
			state.EndCombat(CombatEvacuated)
			assert.Equal(t, location, state.MapState.PlayerLocation)
			before.Advance(DefaultSmallMapMinutesPerTurn)
			assert.Equal(t, before, state.DateTime)
			assert.Nil(t, state.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(source.Pos(), source.Floor()))
			assert.False(t, state.PartyState.IsNPCDead(references.Yew, slot))
		})
	}
}

func TestStonegateProductionGroupsUseOriginalCompositionDrawOrder(t *testing.T) {
	for slot := 5; slot <= 8; slot++ {
		for _, cause := range []SmallMapEncounterCause{SmallMapEncounterAttack, SmallMapEncounterHostileNPCAdjacency} {
			t.Run(fmt.Sprintf("slot_%d/cause_%d", slot, cause), func(t *testing.T) {
				state, _, source := authoredSmallMapCombatFixture(t, references.Stonegate, slot)
				monsterType, eligible := source.NPCReference.SmallMapCombatMonsterType()
				require.True(t, eligible)
				require.GreaterOrEqual(t, monsterType, references.MonsterSeaHorse)
				require.NotEqual(t, references.MonsterDevil, monsterType, "fixture must not introduce guard-activation RNG")
				ref, err := state.enemyReferenceForCombat(monsterType)
				require.NoError(t, err)
				require.Greater(t, ref.MaxPerMap, 1, "production slot must exercise group construction")
				state.openingCampaignEncounterReduction = false
				state.SetRandomSeed(86)
				oracle := beradrng.New(86)
				count := ref.MaxPerMap
				if count != 8 && count != 16 {
					count = oracle.Intn(count) + 1
				}
				if cause == SmallMapEncounterAttack {
					player, direction, ok := emptyAdjacentAttackPosition(state.CurrentNPCAIController.GetNpcs(), source)
					require.True(t, ok)
					state.MapState.PlayerLocation.Position = player
					require.True(t, state.ActionAttackSmallMap(direction))
				} else {
					source.MapUnitDetails().SetOverriddenAiType(references.ExtortOrAttackOrFollow)
					_, err = state.StartSmallMapCombatEncounter(source, cause)
					require.NoError(t, err)
				}
				require.NotNil(t, state.CombatScene)
				assertOriginalEncounterMonsterDraws(t, state, oracle, monsterType, ref.Friend.MonsterType, count, true)
				assert.False(t, state.PartyState.IsNPCDead(references.Stonegate, slot))
			})
		}
	}
}

func TestSmallMapGuardAttackConsumesSameTypeFriendRolls(t *testing.T) {
	state, _, source, direction := newSmallMapCombatSourceTestState(t, true, 86)
	state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{source})
	state.SetRandomSeed(86)
	require.True(t, state.ActionAttackSmallMap(direction))
	require.NotNil(t, state.CombatScene)
	assert.Zero(t, state.CombatScene.SmallMapOrigin.Consequences.GuardActivation.CivilianRollCount)
	assertOriginalEncounterMonsterDraws(t, state, beradrng.New(86), references.MonsterGuard, references.MonsterGuard, 8, true)
}

func TestProductionGargoylesAndBalinorSupportAttackAndActivatedCombat(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).Build()
	type sourceRecord struct {
		location references.Location
		slot     int
		monster  references.MonsterType
	}
	sources := []sourceRecord{{references.Stonegate, 4, references.MonsterDevil}}
	for index, ref := range state.GameReferences.NPCReferences.Npcs {
		if monster, ok := ref.SmallMapCombatMonsterType(); ok && monster == references.MonsterGargoyle {
			sources = append(sources, sourceRecord{ref.Location, index % 32, monster})
		}
	}
	require.Greater(t, len(sources), 1, "production references must include Gargoyles")
	for _, record := range sources {
		for _, cause := range []SmallMapEncounterCause{SmallMapEncounterAttack, SmallMapEncounterHostileNPCAdjacency} {
			t.Run(fmt.Sprintf("%s/slot_%d/cause_%d", record.location, record.slot, cause), func(t *testing.T) {
				state, _, source := authoredSmallMapCombatFixture(t, record.location, record.slot)
				monster, ok := source.NPCReference.SmallMapCombatMonsterType()
				require.True(t, ok)
				require.Equal(t, record.monster, monster)
				if record.monster == references.MonsterDevil {
					require.Equal(t, byte(10), source.NPCReference.DialogNumber, "Balinor uses KEEP.TLK conversation 10")
				}
				state.PartyState.Karma.Value = 60
				state.SetCriminalStatus(false)
				if cause == SmallMapEncounterAttack {
					player, direction, found := emptyAdjacentAttackPosition(state.CurrentNPCAIController.GetNpcs(), source)
					require.True(t, found)
					state.MapState.PlayerLocation.Position = player
					require.True(t, state.ActionAttackSmallMap(direction))
				} else {
					if record.monster == references.MonsterDevil {
						_, err := state.ActivateGuards()
						require.NoError(t, err)
					} else {
						source.MapUnitDetails().SetOverriddenAiType(references.ExtortOrAttackOrFollow)
					}
					_, err := state.StartSmallMapCombatEncounter(source, cause)
					require.NoError(t, err)
				}
				require.NotNil(t, state.CombatScene)
				assert.Equal(t, record.monster, state.CombatScene.MonsterManager.GetMonster(0).MonsterType())
				assert.Equal(t, 60, state.PartyState.Karma.Value)
				assert.Equal(t, cause == SmallMapEncounterAttack && record.monster == references.MonsterDevil, state.IsCriminal())
				assert.False(t, state.PartyState.IsNPCDead(record.location, record.slot))
				state.EndCombat(CombatDefeat)
				assert.False(t, state.PartyState.IsNPCDead(record.location, record.slot))
				assert.Nil(t, state.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(source.Pos(), source.Floor()))
			})
		}
	}
}

func TestAllProductionAuthoredAttackersHaveSafeSmallMapEncounters(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().WithRandomSeed(484).Build()
	count := 0
	for index, reference := range state.GameReferences.NPCReferences.Npcs {
		for scheduleSlot, action := range reference.Schedule.Ai {
			if action != references.ExtortOrAttackOrFollow && action != references.DrudgeWorthThing {
				continue
			}
			count++
			t.Run(fmt.Sprintf("%s/slot_%d/action_%d/schedule_%d", reference.Location, index%32, action, scheduleSlot), func(t *testing.T) {
				monster, ok := reference.SmallMapCombatMonsterType()
				require.True(t, ok, "authored action %d sprite %d must be safely classified", action, reference.GetSpriteIndex())
				state.MapState.PlayerLocation.Location = reference.Location
				state.MapState.PlayerLocation.Floor = references.FloorNumber(reference.Schedule.Floor[scheduleSlot])
				state.UpdateSmallMap(state.GameReferences.TileReferences, state.GameReferences.LocationReferences)
				state.ShadowlordState.Reset()
				source := map_units.NewNPCFriendly(reference, index%32)
				source.PlaceAtScheduleSlot(references.NPCScheduleSlot(scheduleSlot))
				source.SetVisible(true)
				source.MapUnitDetails().SetOverriddenAiType(references.AiType(action))
				state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{source})
				player, _, found := emptyAdjacentAttackPosition(state.CurrentNPCAIController.GetNpcs(), source)
				require.True(t, found)
				state.MapState.PlayerLocation.Position = player
				cause := SmallMapEncounterHostileNPCAdjacency
				if monster == references.MonsterGuard {
					cause = SmallMapEncounterArrestRefusal
				}
				origin, err := state.buildSmallMapEncounterOrigin(source, cause)
				require.NoError(t, err)
				assert.Equal(t, monster, origin.MonsterType)
				assert.Equal(t, SmallMapEncounterOrdinaryNPC, origin.SourceKind)
			})
		}
	}
	require.Positive(t, count)
	t.Logf("validated %d authored action-6/7 schedule records across production references", count)
}

func TestAuthoredSmallMapGroupBitmaps(t *testing.T) {
	t.Parallel()
	for location := references.Location(1); location <= references.Location(32); location++ {
		for slot := range 32 {
			want := (location == references.Yew && (slot == 15 || slot == 17)) ||
				(location == references.Minoc && slot == 1) ||
				(location == references.Windemere && slot >= 3 && slot <= 9) ||
				(location == references.Stonegate && slot >= 5 && slot <= 8)
			assert.Equal(t, want, smallMapNPCIsGroup(location, slot), "%s slot %d", location, slot)
		}
	}
}

func TestAuthoredSmallMapGroupsUseDeterministicFriendsAndFixedBatCount(t *testing.T) {
	var first []combatMonsterSignature
	for run := range 2 {
		state, _, source := authoredSmallMapCombatFixture(t, references.Yew, 15)
		state.SetRandomSeed(86)
		result, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterHostileNPCAdjacency)
		require.NoError(t, err)
		assert.LessOrEqual(t, result.PrimaryCount+result.FriendCount, 10)
		assert.Equal(t, references.MonsterRat, state.CombatScene.MonsterManager.GetMonster(0).MonsterType())
		require.Positive(t, result.FriendCount, "fixed seed exercises original Skeleton friends")
		assert.Equal(t, result.FriendCount, state.CombatScene.MonsterManager.GetMonsterTypeCount(references.MonsterSkeleton))
		if run == 0 {
			first = combatMonsterSignatures(state)
		} else {
			assert.Equal(t, first, combatMonsterSignatures(state))
		}
	}
	state, _, source := authoredSmallMapCombatFixture(t, references.Yew, 15)
	source.NPCReference.SetKeyIndex(indexes.BatKeyIndex)
	result, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterHostileNPCAdjacency)
	require.NoError(t, err)
	assert.Equal(t, 16, result.PrimaryCount, "group bitmap, not location eligibility, selects original swarm composition")
	assert.Zero(t, result.FriendCount)
}

func TestYewBasementRatCombatReturnsWithoutPersistentDeath(t *testing.T) {
	for _, conclusion := range []CombatEndReason{CombatVictory, CombatDefeat, CombatEvacuated} {
		for _, slot := range []int{15, 16, 17} {
			t.Run(fmt.Sprintf("slot_%d/%s", slot, combatConclusionName(conclusion)), func(t *testing.T) {
				state, callbacks, source := authoredSmallMapCombatFixture(t, references.Yew, slot)
				state.SetRandomSeed(0)
				require.Equal(t, references.FloorNumber(-1), source.Floor())
				state.PartyState.Karma.Value = 60
				state.SetCriminalStatus(false)
				location := state.MapState.SavePlayerLocation()
				result, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterHostileNPCAdjacency)
				require.NoError(t, err)
				assert.Equal(t, references.MonsterRat, state.CombatScene.SmallMapOrigin.MonsterType)
				if slot == 16 {
					assert.Equal(t, 1, result.PrimaryCount)
				} else {
					assert.Greater(t, result.PrimaryCount, 1, "fixed seed must exercise grouped rats")
					assert.LessOrEqual(t, result.PrimaryCount, 10)
				}
				assert.Zero(t, result.FriendCount)
				assert.False(t, state.PartyState.IsNPCDead(references.Yew, slot))
				assert.Equal(t, 60, state.PartyState.Karma.Value)
				assert.False(t, state.IsCriminal())
				before := state.DateTime
				state.EndCombat(conclusion)
				assert.Equal(t, location, state.MapState.PlayerLocation)
				assert.Equal(t, before, state.DateTime)
				assert.Empty(t, callbacks.TimeAdvanced())
				assert.Nil(t, state.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(source.Pos(), source.Floor()))
				assert.False(t, state.PartyState.IsNPCDead(references.Yew, slot))
				state.UpdateSmallMap(state.GameReferences.TileReferences, state.GameReferences.LocationReferences)
				assert.NotNil(t, state.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(source.Pos(), source.Floor()))
			})
		}
	}
}

func TestAuthoredHostileOrdinaryNPCPersistsAtEntryAndRejectsAstarothInfluence(t *testing.T) {
	for _, action := range []references.AiType{references.ExtortOrAttackOrFollow, references.DrudgeWorthThing} {
		for _, conclusion := range []CombatEndReason{CombatVictory, CombatDefeat, CombatEvacuated} {
			t.Run(fmt.Sprintf("action_%d/%s", action, combatConclusionName(conclusion)), func(t *testing.T) {
				state, _, source, _ := newSmallMapCombatSourceTestState(t, false, 484)
				source.MapUnitDetails().SetAIOverride(action, references.AIOverrideProvenanceAutomaticAstaroth)
				_, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterHostileNPCAdjacency)
				require.ErrorIs(t, err, ErrInvalidSmallMapEncounter)
				assert.False(t, state.PartyState.IsNPCDead(references.Britain, source.MapUnitDetails().NPCNum))
				source.MapUnitDetails().SetAIOverride(action, references.AIOverrideProvenanceNone)
				state.PartyState.Karma.Value = 60
				_, err = state.StartSmallMapCombatEncounter(source, SmallMapEncounterHostileNPCAdjacency)
				require.NoError(t, err)
				assert.True(t, state.PartyState.IsNPCDead(references.Britain, source.MapUnitDetails().NPCNum))
				assert.Equal(t, 60, state.PartyState.Karma.Value)
				assert.False(t, state.CombatScene.SmallMapOrigin.Consequences.ActivateGuards)
				state.EndCombat(conclusion)
				assert.True(t, state.PartyState.IsNPCDead(references.Britain, source.MapUnitDetails().NPCNum))
			})
		}
	}
}

func TestAuthoredStonegateShadowlordsKeepQuestIdentityWithoutSummoningProvenance(t *testing.T) {
	for slot := 1; slot <= 3; slot++ {
		for _, conclusion := range []CombatEndReason{CombatVictory, CombatDefeat, CombatEvacuated} {
			t.Run(fmt.Sprintf("slot_%d/%s", slot, combatConclusionName(conclusion)), func(t *testing.T) {
				state, _, source := authoredSmallMapCombatFixture(t, references.Stonegate, slot)
				lord := references.Shadowlord(slot - 1)
				state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
				before := *state.ShadowlordState
				assert.Nil(t, state.queueAdjacentSmallMapShadowlordEncounter(), "fixed authored actors do not acquire automatic contact")
				_, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterAttack)
				require.NoError(t, err)
				require.NotNil(t, state.CombatScene)
				origin := state.CombatScene.SmallMapOrigin
				assert.Equal(t, references.ShadowlordCombatMap, origin.CombatMap)
				assert.Equal(t, 1, state.CombatScene.MonsterManager.Count())
				assert.Equal(t, SmallMapEncounterOrdinaryNPC, origin.SourceKind)
				assert.Equal(t, lord, origin.Shadowlord)
				assert.True(t, origin.Consequences.SceptreReclaimed)
				assert.Equal(t, before, *state.ShadowlordState)
				state.EndCombat(conclusion)
				assert.True(t, state.ShadowlordState.IsAlive(lord))
				assert.False(t, state.ShadowlordState.Summoned)
				assert.Equal(t, references.NoShadowlord, state.ShadowlordState.AutomaticPresence)
				assert.False(t, state.PartyState.IsNPCDead(references.Stonegate, slot))
				assert.Zero(t, state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
			})
		}
	}
}

func TestAuthoredStonegateShadowlordHostilityAndQuestDeathValidation(t *testing.T) {
	for slot := 1; slot <= 3; slot++ {
		t.Run(fmt.Sprint(slot), func(t *testing.T) {
			state, _, source := authoredSmallMapCombatFixture(t, references.Stonegate, slot)
			_, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterHostileNPCAdjacency)
			require.ErrorIs(t, err, ErrInvalidSmallMapEncounter, "authored fixed schedule is not hostile")
			source.MapUnitDetails().SetOverriddenAiType(references.DrudgeWorthThing)
			require.NoError(t, state.QueueSmallMapCombatEncounter(source, SmallMapEncounterHostileNPCAdjacency))
			state.PartyState.SetDeadNpc(references.Stonegate, slot)
			require.ErrorIs(t, state.ResolvePendingOverworldContact(), ErrInvalidSmallMapEncounter)
			assert.Nil(t, state.CombatScene)
		})
	}
}

func TestSmallMapHostileHandoffRevalidatesActionAndKeepsGuardArrest(t *testing.T) {
	state, _, source := authoredSmallMapCombatFixture(t, references.Yew, 15)
	require.NoError(t, state.QueueSmallMapCombatEncounter(source, SmallMapEncounterHostileNPCAdjacency))
	source.MapUnitDetails().SetOverriddenAiType(references.Fixed)
	require.ErrorIs(t, state.ResolvePendingOverworldContact(), ErrInvalidSmallMapEncounter)
	assert.Nil(t, state.CombatScene)
	source.NPCReference.SetKeyIndex(indexes.GuardKeyIndex)
	source.MapUnitDetails().SetOverriddenAiType(references.ExtortOrAttackOrFollow)
	_, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterHostileNPCAdjacency)
	require.ErrorIs(t, err, ErrInvalidSmallMapEncounter)
}

func authoredSmallMapCombatFixture(t *testing.T, location references.Location, slot int) (*GameState, *MockSystemCallbacks, *map_units.NPCFriendly) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).WithLocation(location).WithSystemCallbacks().WithRandomSeed(484).Build()
	state.ShadowlordState.Reset()
	reference := (*state.GameReferences.NPCReferences.GetNPCReferencesByLocation(location))[slot]
	source := map_units.NewNPCFriendly(reference, slot)
	source.PlaceAtScheduleSlot(0)
	source.SetVisible(true)
	state.MapState.PlayerLocation.Floor = source.Floor()
	state.UpdateSmallMap(state.GameReferences.TileReferences, state.GameReferences.LocationReferences)
	state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{source})
	player, _, found := emptyAdjacentAttackPosition(state.CurrentNPCAIController.GetNpcs(), source)
	require.True(t, found)
	state.MapState.PlayerLocation.Position = player
	state.CurrentNPCAIController.FreshenExistingNPCsOnMap()
	state.SetRandomSeed(484)
	callbacks.Reset()

	return state, callbacks, source
}
