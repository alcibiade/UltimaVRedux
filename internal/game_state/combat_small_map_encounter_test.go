package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestSelectSmallMapCombatMapUsesOriginalTownTerrainSelector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		terrain *references.Tile
		want    references.BritanniaCombatMap
	}{
		{name: "missing terrain", terrain: nil, want: references.BrickCombatMap},
		{name: "indoor floor", terrain: smallMapTerrain(indexes.BrickFloor), want: references.BrickCombatMap},
		{name: "unlisted town tile", terrain: smallMapTerrain(indexes.WoodFloor1), want: references.BrickCombatMap},
		{name: "grass", terrain: smallMapTerrain(indexes.Grass), want: references.GladeCombatMap},
		{name: "swamp", terrain: smallMapTerrain(indexes.Swamp), want: references.SwampCombatMap},
		{name: "brush", terrain: smallMapTerrain(indexes.Forest1), want: references.TreedCombatMap},
		{name: "desert", terrain: smallMapTerrain(indexes.Desert1), want: references.DesertCombatMap},
		{name: "forest", terrain: smallMapTerrain(indexes.Forest3), want: references.CleanTreeCombatMap},
		{name: "mountains", terrain: smallMapTerrain(indexes.SmallMountains), want: references.MountainsCombatMap},
		{name: "bridge", terrain: smallMapTerrain(indexes.Bridge), want: references.BigBridgeCombatMap},
		{name: "left planks", terrain: smallMapTerrain(indexes.LeftWoodFloor2), want: references.BigBridgeCombatMap},
		{name: "right planks", terrain: smallMapTerrain(indexes.RightWoodFloor2), want: references.BigBridgeCombatMap},
		{name: "water", terrain: smallMapTerrain(indexes.Water1), want: references.LandSeaCombatMap},
		{name: "river", terrain: smallMapTerrain(indexes.WaterStream1), want: references.LandSeaCombatMap},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, SelectSmallMapCombatMap(test.terrain))
		})
	}
}

func TestSmallMapNPCMonsterTypeUsesShapesNotLocationAllowlists(t *testing.T) {
	tests := []struct {
		name     string
		location references.Location
		npcSlot  int
		sprite   indexes.SpriteIndex
		want     references.MonsterType
		wantErr  bool
	}{
		{name: "Windemere devil", location: references.Windemere, npcSlot: 10, sprite: indexes.Daemon_KeyIndex, want: references.MonsterDevil},
		{name: "other Windemere slot", location: references.Windemere, npcSlot: 9, sprite: indexes.Daemon_KeyIndex, want: references.MonsterDevil},
		{name: "daemon elsewhere", location: references.Stonegate, npcSlot: 10, sprite: indexes.Daemon_KeyIndex, want: references.MonsterDevil},
		{name: "Iolo hut rat", location: references.IolosHut, npcSlot: 1, sprite: indexes.Rat_KeyIndex, want: references.MonsterRat},
		{name: "other Iolo rat slot", location: references.IolosHut, npcSlot: 4, sprite: indexes.Rat_KeyIndex, want: references.MonsterRat},
		{name: "Skara Brae bat", location: references.SkaraBrae, npcSlot: 8, sprite: indexes.Bat_KeyIndex, want: references.MonsterBat},
		{name: "other Skara bat slot", location: references.SkaraBrae, npcSlot: 7, sprite: indexes.Bat_KeyIndex, want: references.MonsterBat},
		{name: "rat elsewhere", location: references.Yew, npcSlot: 1, sprite: indexes.Rat_KeyIndex, want: references.MonsterRat},
		{name: "quest object", location: references.Stonegate, npcSlot: 9, sprite: indexes.SpriteIndex(437), want: references.MonsterNone, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reference := references.NPCReference{Location: test.location}
			reference.SetKeyIndex(test.sprite)
			source := map_units.NewNPCFriendly(reference, test.npcSlot)

			got, err := smallMapNPCMonsterType(source)
			if test.wantErr {
				assert.ErrorIs(t, err, ErrInvalidSmallMapEncounter)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.want, got)
		})
	}
}

func TestOriginalDataDefinesOnlyReportedVisitLocalRatAndBatSources(t *testing.T) {
	tests := []struct {
		name      string
		location  references.Location
		monster   references.MonsterType
		positions map[int]references.Position
	}{
		{
			name:     "Iolo hut rats",
			location: references.IolosHut,
			monster:  references.MonsterRat,
			positions: map[int]references.Position{
				1: {X: 23, Y: 8},
				2: {X: 13, Y: 5},
				3: {X: 3, Y: 6},
			},
		},
		{
			name:     "Skara Brae bats",
			location: references.SkaraBrae,
			monster:  references.MonsterBat,
			positions: map[int]references.Position{
				8:  {X: 15, Y: 15},
				9:  {X: 14, Y: 14},
				10: {X: 16, Y: 14},
				11: {X: 16, Y: 16},
				12: {X: 14, Y: 16},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).
				WithLocation(test.location).
				WithFloor(references.MainFloor).
				WithRandomSeed(484).
				Build()
			state.ShadowlordState.Reset()

			for slot, position := range test.positions {
				source := visitLocalRatOrBatBySlot(t, state, test.location, slot, test.monster)
				monsterType, ok := source.NPCReference.SmallMapCombatMonsterType()
				require.True(t, ok)
				assert.Equal(t, test.monster, monsterType)
				assert.Equal(t, position, source.Pos())
				assert.Equal(t, byte(references.NoResponse), source.NPCReference.DialogNumber)
				assert.False(t, state.PartyState.IsNPCDead(test.location, slot))
				for scheduleSlot := range source.NPCReference.Schedule.Ai {
					assert.Equal(t, byte(references.ExtortOrAttackOrFollow), source.NPCReference.Schedule.Ai[scheduleSlot])
					assert.Equal(t, byte(references.MainFloor), source.NPCReference.Schedule.Floor[scheduleSlot])
				}
			}
		})
	}
}

func TestVisitLocalRatAndBatAttackUsesMonsterConsequencesAndReloadsSource(t *testing.T) {
	tests := []struct {
		name     string
		location references.Location
		npcSlot  int
		monster  references.MonsterType
		position references.Position
	}{
		{name: "Iolo hut rat", location: references.IolosHut, npcSlot: 1, monster: references.MonsterRat, position: references.Position{X: 23, Y: 8}},
		{name: "Skara Brae bat", location: references.SkaraBrae, npcSlot: 8, monster: references.MonsterBat, position: references.Position{X: 15, Y: 15}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(test.location).
				WithFloor(references.MainFloor).
				WithSystemCallbacks().
				WithRandomSeed(484).
				Build()
			state.ShadowlordState.Reset()
			source := visitLocalRatOrBatBySlot(t, state, test.location, test.npcSlot, test.monster)
			player, direction, found := emptyAdjacentAttackPosition(state.CurrentNPCAIController.GetNpcs(), source)
			require.True(t, found)
			state.MapState.PlayerLocation.Position = player
			state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{source})
			state.CurrentNPCAIController.FreshenExistingNPCsOnMap()
			state.PartyState.Karma.Value = 60
			state.SetCriminalStatus(false)
			callbacks.Reset()

			require.True(t, state.ActionAttackSmallMap(direction))
			require.NotNil(t, state.CombatScene)
			origin := state.CombatScene.SmallMapOrigin
			require.NotNil(t, origin)
			assert.Same(t, source, origin.SourceNPC)
			assert.Equal(t, SmallMapEncounterAttack, origin.Cause)
			assert.Equal(t, test.monster, origin.MonsterType)
			assert.Equal(t, 1, state.CombatScene.MonsterManager.Count())
			assert.Equal(t, test.monster, state.CombatScene.MonsterManager.GetMonster(0).MonsterType())
			assert.Equal(t, 60, state.PartyState.Karma.Value)
			assert.False(t, state.IsCriminal())
			assert.False(t, state.PartyState.IsNPCDead(test.location, test.npcSlot))
			assert.Zero(t, origin.Consequences.KarmaPenalty)
			assert.False(t, origin.Consequences.ActivateGuards)
			assert.False(t, origin.Consequences.PersistentSourceDead)
			assert.True(t, origin.Consequences.CompleteSourceTurn)
			assert.Zero(t, origin.Consequences.GuardActivationCount)
			assert.Empty(t, callbacks.TimeAdvanced())
			beforeTurn := state.DateTime

			state.EndCombat(CombatVictory)

			expectedDate := beforeTurn
			expectedDate.Advance(DefaultSmallMapMinutesPerTurn)
			assert.Equal(t, expectedDate, state.DateTime)
			assert.Nil(t, state.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(
				test.position,
				references.MainFloor,
			))
			assert.False(t, state.PartyState.IsNPCDead(test.location, test.npcSlot))

			state.UpdateSmallMap(state.GameReferences.TileReferences, state.GameReferences.LocationReferences)
			reloaded := visitLocalRatOrBatBySlot(t, state, test.location, test.npcSlot, test.monster)
			assert.Equal(t, test.position, reloaded.Pos())
			assert.True(t, reloaded.IsVisible())
		})
	}
}

func TestOrdinarySmallMapNPCAttackReturnsEveryConclusion(t *testing.T) {
	for _, conclusion := range []CombatEndReason{CombatVictory, CombatDefeat, CombatEvacuated} {
		t.Run(combatConclusionName(conclusion), func(t *testing.T) {
			state, callbacks, source, direction := newSmallMapCombatSourceTestState(t, false, 294)
			sourceLocation := state.MapState.SavePlayerLocation()
			sourcePosition := source.Pos()
			sourceFloor := source.Floor()
			sourceNumber := source.MapUnitDetails().NPCNum
			monsterType, err := smallMapNPCMonsterType(source)
			require.NoError(t, err)
			expectedMap := SelectSmallMapCombatMap(
				state.GetLayeredMapByCurrentLocation().GetTileTopMapOnlyTile(source.PosPtr()),
			)
			state.PartyState.Karma.Value = 50
			require.True(t, state.SetActivePlayer(1))
			callbacks.Reset()

			require.True(t, state.ActionAttackSmallMap(direction))
			require.NotNil(t, state.CombatScene)
			origin := state.CombatScene.SmallMapOrigin
			require.NotNil(t, origin)
			assert.Same(t, source, origin.SourceNPC)
			assert.Equal(t, sourceLocation.Location, origin.Location)
			assert.Equal(t, sourceFloor, origin.Floor)
			assert.Equal(t, sourcePosition, origin.Position)
			assert.Equal(t, sourceNumber, origin.NPCNumber)
			assert.Equal(t, monsterType, origin.MonsterType)
			assert.Equal(t, expectedMap, origin.CombatMap)
			assert.Equal(t, CombatEntryNorth, state.CombatScene.EntryDirection)
			assert.Equal(t, 1, state.CombatScene.MonsterManager.Count())
			assert.Equal(t, monsterType, state.CombatScene.MonsterManager.GetMonster(0).MonsterType())
			assert.Equal(t, 45, state.PartyState.Karma.Value)
			assert.True(t, state.IsCriminal())
			assert.True(t, state.PartyState.IsNPCDead(origin.Location, sourceNumber))
			assert.Equal(t, 1, origin.Consequences.GuardActivationCount)
			assert.Equal(t, 45, origin.Consequences.GuardActivation.KarmaAtActivation)
			assert.Empty(t, callbacks.TimeAdvanced(), "Attack completes its source turn only after combat")
			assertRemainingSmallMapGuardsHostile(t, state, source)
			beforeTurn := state.DateTime
			state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 2}

			state.EndCombat(conclusion)

			assert.Nil(t, state.CombatScene)
			assert.Nil(t, state.pendingSmallMapEncounter)
			assert.Equal(t, sourceLocation, state.MapState.PlayerLocation)
			assert.True(t, state.HasActivePlayer())
			assert.Equal(t, 1, state.GetActivePlayerIndex())
			assert.Nil(t, state.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(
				sourcePosition,
				sourceFloor,
			))
			expectedDate := beforeTurn
			expectedDate.Advance(DefaultSmallMapMinutesPerTurn)
			assert.Equal(t, expectedDate, state.DateTime)
			assert.Equal(t, 1, state.ActiveDurationEffect.TurnsRemaining)
			assertRemainingSmallMapGuardsHostile(t, state, source)
			require.Len(t, callbacks.CombatConclusions, 1)
			assert.Equal(t, conclusion, callbacks.CombatConclusions[0].Reason)

			state.EndCombat(conclusion)
		})
	}
}

func TestGuardSmallMapAttackCreatesOriginalGroupWithoutPersistentDeath(t *testing.T) {
	state, callbacks, source, direction := newSmallMapCombatSourceTestState(t, true, 295)
	sourceLocation := state.MapState.SavePlayerLocation()
	sourceNumber := source.MapUnitDetails().NPCNum
	state.PartyState.Karma.Value = 60
	callbacks.Reset()

	require.True(t, state.ActionAttackSmallMap(direction))
	require.NotNil(t, state.CombatScene)
	origin := state.CombatScene.SmallMapOrigin
	require.NotNil(t, origin)
	assert.Equal(t, references.MonsterGuard, origin.MonsterType)
	assert.Equal(t, references.GuardGroupCount, state.CombatScene.MonsterManager.Count())
	for index := range references.GuardGroupCount {
		assert.Equal(t, references.MonsterGuard, state.CombatScene.MonsterManager.GetMonster(index).MonsterType())
	}
	assert.Equal(t, 55, state.PartyState.Karma.Value)
	assert.False(t, state.PartyState.IsNPCDead(sourceLocation.Location, sourceNumber))
	assert.True(t, guardActivationNPCIsHostile(source))
	assert.Equal(t, 1, origin.Consequences.GuardActivationCount)
	assert.Equal(t, 55, origin.Consequences.GuardActivation.KarmaAtActivation)
	assertRemainingSmallMapGuardsHostile(t, state, source)
	beforeTurn := state.DateTime

	state.EndCombat(CombatVictory)

	assert.Equal(t, sourceLocation, state.MapState.PlayerLocation)
	assert.False(t, state.PartyState.IsNPCDead(sourceLocation.Location, sourceNumber))
	expectedDate := beforeTurn
	expectedDate.Advance(DefaultSmallMapMinutesPerTurn)
	assert.Equal(t, expectedDate, state.DateTime)
	assertRemainingSmallMapGuardsHostile(t, state, source)
}

func TestWindemereDevilAttackHasMonsterConsequencesAndReturnsOnReload(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Windemere).
		WithFloor(references.MainFloor).
		WithSystemCallbacks().
		WithRandomSeed(472).
		Build()
	state.ShadowlordState.Reset()
	source := windemereDevilBySlot(t, state, 10)
	player, direction, found := emptyAdjacentAttackPosition(state.CurrentNPCAIController.GetNpcs(), source)
	require.True(t, found)
	state.MapState.PlayerLocation.Position = player
	baseTile := state.GetLayeredMapByCurrentLocation().GetTileTopMapOnlyTile(source.PosPtr())
	require.NotNil(t, baseTile)
	require.False(t, state.isDefenselessBaseTile(baseTile.Index))
	state.PartyState.Karma.Value = 60
	callbacks.Reset()

	require.True(t, state.ActionAttackSmallMap(direction))
	require.NotNil(t, state.CombatScene)
	origin := state.CombatScene.SmallMapOrigin
	require.NotNil(t, origin)
	assert.Same(t, source, origin.SourceNPC)
	assert.Equal(t, references.MonsterDevil, origin.MonsterType)
	assert.Equal(t, SmallMapEncounterAttack, origin.Cause)
	assert.Equal(t, 1, state.CombatScene.MonsterManager.Count())
	assert.Equal(t, references.MonsterDevil, state.CombatScene.MonsterManager.GetMonster(0).MonsterType())
	assert.Equal(t, 60, state.PartyState.Karma.Value)
	assert.True(t, state.IsCriminal())
	assert.False(t, state.PartyState.IsNPCDead(references.Windemere, 10))
	assert.Equal(t, 1, origin.Consequences.GuardActivationCount)
	assert.True(t, guardActivationNPCIsHostile(windemereDevilBySlot(t, state, 11)))
	beforeTurn := state.DateTime

	state.EndCombat(CombatDefeat)

	assert.Nil(t, state.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(
		origin.Position,
		origin.Floor,
	))
	assert.False(t, state.PartyState.IsNPCDead(references.Windemere, 10))
	expectedDate := beforeTurn
	expectedDate.Advance(DefaultSmallMapMinutesPerTurn)
	assert.Equal(t, expectedDate, state.DateTime)

	state.UpdateSmallMap(state.GameReferences.TileReferences, state.GameReferences.LocationReferences)
	reloaded := windemereDevilBySlot(t, state, 10)
	assert.Equal(t, references.Position{X: 13, Y: 20}, reloaded.Pos())
	assert.True(t, reloaded.IsVisible())
}

func TestWindemereOriginalDataDefinesTwoFixedNoResponseDevils(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Windemere).
		WithFloor(references.MainFloor).
		Build()

	for slot, position := range map[int]references.Position{
		10: {X: 13, Y: 20},
		11: {X: 17, Y: 20},
	} {
		devil := windemereDevilBySlot(t, state, slot)
		assert.Equal(t, position, devil.Pos())
		assert.Equal(t, byte(references.NoResponse), devil.NPCReference.DialogNumber)
		assert.NotEqual(t, references.Guard, devil.NPCReference.GetNPCType())
		for scheduleSlot := range devil.NPCReference.Schedule.Ai {
			assert.Equal(t, byte(references.Fixed), devil.NPCReference.Schedule.Ai[scheduleSlot])
		}
	}
}

func TestWindemereHostileContactRetainsLastEligibleNPCSlot(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Windemere).
		WithFloor(references.MainFloor).
		WithRandomSeed(472).
		Build()
	controller, ok := state.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
	require.True(t, ok)
	first := windemereDevilBySlot(t, state, 10)
	last := windemereDevilBySlot(t, state, 11)
	first.SetPos(references.Position{X: 14, Y: 20})
	last.SetPos(references.Position{X: 16, Y: 20})
	state.MapState.PlayerLocation.Position = references.Position{X: 15, Y: 20}
	controller.GetNpcs().ReplaceWith([]map_units.MapUnit{last, first})
	_, err := state.ActivateGuards()
	require.NoError(t, err)

	controller.AdvanceNextTurnCalcAndMoveNPCs()

	assert.Equal(t, map_units.PendingNPCInteractionNone, first.MapUnitDetails().PendingInteraction)
	assert.Equal(t, map_units.PendingNPCInteractionHostileCombat, last.MapUnitDetails().PendingInteraction)
}

func TestRefuseArrestQueuesAndStartsSourceGuardCombat(t *testing.T) {
	state, callbacks, source, _ := newSmallMapCombatSourceTestState(t, true, 296)
	sourceLocation := state.MapState.SavePlayerLocation()
	sourcePosition := source.Pos()
	sourceFloor := source.Floor()
	sourceNumber := source.MapUnitDetails().NPCNum
	state.PartyState.Karma.Value = 70
	callbacks.Reset()

	require.True(t, state.RefuseArrest(source, true))
	assert.Nil(t, state.CombatScene)
	require.NotNil(t, state.pendingSmallMapEncounter)
	assert.Equal(t, SmallMapEncounterArrestRefusal, state.pendingSmallMapEncounter.Cause)
	assert.NotContains(t, callbacks.Messages, "Combat not yet implemented!")

	require.NoError(t, state.ResolvePendingOverworldContact())
	require.NotNil(t, state.CombatScene)
	assert.Nil(t, state.pendingSmallMapEncounter)
	origin := state.CombatScene.SmallMapOrigin
	require.NotNil(t, origin)
	assert.Equal(t, references.MonsterGuard, origin.MonsterType)
	assert.Equal(t, references.GuardGroupCount, state.CombatScene.MonsterManager.Count())
	assert.Equal(t, 70, state.PartyState.Karma.Value)
	assert.True(t, state.IsCriminal())
	assert.True(t, guardActivationNPCIsHostile(source))
	assert.Equal(t, 1, origin.Consequences.GuardActivationCount)
	assert.Empty(t, callbacks.TimeAdvanced())
	assertRemainingSmallMapGuardsHostile(t, state, source)
	beforeTurn := state.DateTime
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 2}

	state.EndCombat(CombatDefeat)

	assert.Equal(t, sourceLocation, state.MapState.PlayerLocation)
	assert.False(t, state.PartyState.IsNPCDead(sourceLocation.Location, sourceNumber))
	assert.Nil(t, state.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(
		sourcePosition,
		sourceFloor,
	))
	expectedDate := beforeTurn
	expectedDate.Advance(DefaultSmallMapMinutesPerTurn)
	assert.Equal(t, expectedDate, state.DateTime)
	assert.Equal(t, 1, state.ActiveDurationEffect.TurnsRemaining)
	assertRemainingSmallMapGuardsHostile(t, state, source)
}

func TestNPCInitiatedArrestRefusalDoesNotCompleteASecondTurn(t *testing.T) {
	state, callbacks, source, _ := newSmallMapCombatSourceTestState(t, true, 296)
	require.True(t, state.RefuseArrest(source, false))
	require.NoError(t, state.ResolvePendingOverworldContact())
	beforeTurn := state.DateTime
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 2}

	state.EndCombat(CombatDefeat)

	assert.Equal(t, beforeTurn, state.DateTime)
	assert.Equal(t, 2, state.ActiveDurationEffect.TurnsRemaining)
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestSmallMapCombatReturnReloadsTerrainAndPreservesNPCState(t *testing.T) {
	state, _, source, direction := newSmallMapCombatSourceTestState(t, false, 318)
	controller := state.CurrentNPCAIController
	attackPosition := state.MapState.PlayerLocation.Position
	doorPosition, doorDirection, doorPlayerPosition := findSmallMapDoorForCombatReturnTest(t, state)
	state.MapState.PlayerLocation.Position = doorPlayerPosition
	require.Equal(t, map_state.OpenDoorOpened, state.MapState.OpenDoor(doorDirection))
	state.MapState.PlayerLocation.Position = attackPosition
	require.True(t, state.MapState.HasTransientDoor())

	require.True(t, state.ActionAttackSmallMap(direction))
	var survivor *map_units.NPCFriendly
	for _, unit := range *controller.GetNpcs() {
		friendly, ok := unit.(*map_units.NPCFriendly)
		if ok && friendly != source && friendly.IsVisible() &&
			friendly.MapUnitDetails().AIOverrideProvenance() != references.AIOverrideProvenanceNone {
			survivor = friendly

			break
		}
	}
	require.NotNil(t, survivor)
	aiBefore := survivor.MapUnitDetails().GetOverriddenAiType()
	provenanceBefore := survivor.MapUnitDetails().AIOverrideProvenance()

	state.EndCombat(CombatVictory)

	assert.Same(t, controller, state.CurrentNPCAIController)
	assert.False(t, state.MapState.HasTransientDoor())
	reloadedMap := state.GetLayeredMapByCurrentLocation()
	assert.True(t, reloadedMap.GetTileTopMapOnlyTile(&doorPosition).Index.IsUnlockedDoor())
	assert.Equal(t, aiBefore, survivor.MapUnitDetails().GetOverriddenAiType())
	assert.Equal(t, provenanceBefore, survivor.MapUnitDetails().AIOverrideProvenance())
}

func TestQueuedSmallMapEncounterBlocksSaveAndCleansUpAfterInvalidation(t *testing.T) {
	state, callbacks, source, _ := newSmallMapCombatSourceTestState(t, true, 297)
	state.PartyState.Karma.Value = 70
	callbacks.Reset()

	require.True(t, state.RefuseArrest(source, false))
	origin := state.pendingSmallMapEncounter
	require.NotNil(t, origin)
	assert.ErrorIs(t, state.ValidateSaveSnapshotEligibility(), ErrSaveSnapshotNotRepresentable)
	assert.ErrorIs(t,
		state.QueueSmallMapCombatEncounter(source, SmallMapEncounterArrestRefusal),
		ErrSmallMapEncounterPending,
	)
	require.True(t, state.CurrentNPCAIController.GetNpcs().RemoveNPCAtPositionAndFloor(
		source.Pos(),
		source.Floor(),
	))

	err := state.ResolvePendingOverworldContact()

	assert.ErrorIs(t, err, ErrInvalidSmallMapEncounter)
	assert.Nil(t, state.pendingSmallMapEncounter)
	assert.Nil(t, state.CombatScene)
	assert.Equal(t, 70, state.PartyState.Karma.Value)
	assert.False(t, state.IsCriminal())
	assert.Zero(t, origin.Consequences.GuardActivationCount)
	assert.NoError(t, state.ValidateSaveSnapshotEligibility())
}

func TestSmallMapGuardCombatConstructionIsDeterministic(t *testing.T) {
	firstActivation, firstMonsters := smallMapGuardCombatSnapshot(t, 317)
	secondActivation, secondMonsters := smallMapGuardCombatSnapshot(t, 317)

	assert.Equal(t, firstActivation, secondActivation)
	assert.Equal(t, firstMonsters, secondMonsters)
}

type smallMapCombatMonsterSnapshot struct {
	MonsterType references.MonsterType
	Position    references.Position
	HitPoints   int
}

func smallMapGuardCombatSnapshot(
	t *testing.T,
	seed uint64,
) (GuardActivationResult, []smallMapCombatMonsterSnapshot) {
	t.Helper()
	state, _, _, direction := newSmallMapCombatSourceTestState(t, true, seed)
	require.True(t, state.ActionAttackSmallMap(direction))
	origin := state.CombatScene.SmallMapOrigin
	snapshots := make([]smallMapCombatMonsterSnapshot, state.CombatScene.MonsterManager.Count())
	for index := range snapshots {
		monster := state.CombatScene.MonsterManager.GetMonster(index)
		snapshots[index] = smallMapCombatMonsterSnapshot{
			MonsterType: monster.MonsterType(),
			Position:    monster.Position(),
			HitPoints:   monster.CurrentHP(),
		}
	}

	return origin.Consequences.GuardActivation, snapshots
}

func newSmallMapCombatSourceTestState(
	t *testing.T,
	wantGuard bool,
	seed uint64,
) (*GameState, *MockSystemCallbacks, *map_units.NPCFriendly, references.Direction) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(seed).
		Build()
	state.ShadowlordState.Reset()
	units := state.CurrentNPCAIController.GetNpcs()
	for _, unit := range *units {
		source, isFriendly := unit.(*map_units.NPCFriendly)
		if !isFriendly || !source.IsVisible() || source.Floor() != state.MapState.PlayerLocation.Floor {
			continue
		}
		monsterType, err := smallMapNPCMonsterType(source)
		if err != nil || (monsterType == references.MonsterGuard) != wantGuard {
			continue
		}
		baseTile := state.GetLayeredMapByCurrentLocation().GetTileTopMapOnlyTile(source.PosPtr())
		if baseTile == nil || state.isDefenselessBaseTile(baseTile.Index) {
			continue
		}
		if player, direction, ok := emptyAdjacentAttackPosition(units, source); ok {
			state.MapState.PlayerLocation.Position = player

			return state, callbacks, source, direction
		}
	}

	t.Fatalf("Britain has no eligible guard=%v small-map combat source", wantGuard)

	return nil, nil, nil, references.NoneDirection
}

func emptyAdjacentAttackPosition(
	units *map_units.MapUnits,
	source *map_units.NPCFriendly,
) (references.Position, references.Direction, bool) {
	tests := []struct {
		fromSource references.Direction
		attack     references.Direction
	}{
		{fromSource: references.Down, attack: references.Up},
		{fromSource: references.Up, attack: references.Down},
		{fromSource: references.Left, attack: references.Right},
		{fromSource: references.Right, attack: references.Left},
	}
	for _, test := range tests {
		player := test.fromSource.GetNewPositionInDirection(source.PosPtr())
		if player == nil || player.X < 0 || player.X >= references.XSmallMapTiles ||
			player.Y < 0 || player.Y >= references.YSmallMapTiles {
			continue
		}
		if units.GetMapUnitAtPositionAndFloorOrNil(*player, source.Floor()) == nil {
			return *player, test.attack, true
		}
	}

	return references.Position{}, references.NoneDirection, false
}

func smallMapTerrain(index indexes.SpriteIndex) *references.Tile {
	return &references.Tile{Index: index} //nolint:exhaustruct // Map selection needs only the source tile index.
}

func findSmallMapDoorForCombatReturnTest(
	t *testing.T,
	state *GameState,
) (references.Position, references.Direction, references.Position) {
	t.Helper()
	layeredMap := state.GetLayeredMapByCurrentLocation()
	for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
		for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
			door := references.Position{X: x, Y: y}
			tile := layeredMap.GetTileTopMapOnlyTile(&door)
			if tile == nil || !tile.Index.IsUnlockedDoor() {
				continue
			}
			for _, candidate := range []struct {
				direction references.Direction
				player    references.Position
			}{
				{direction: references.Up, player: references.Position{X: x, Y: y + 1}},
				{direction: references.Down, player: references.Position{X: x, Y: y - 1}},
				{direction: references.Left, player: references.Position{X: x + 1, Y: y}},
				{direction: references.Right, player: references.Position{X: x - 1, Y: y}},
			} {
				if candidate.player.X >= 0 && candidate.player.X < references.XSmallMapTiles &&
					candidate.player.Y >= 0 && candidate.player.Y < references.YSmallMapTiles {
					return door, candidate.direction, candidate.player
				}
			}
		}
	}
	t.Fatal("small-map fixture has no unlocked door")

	return references.Position{}, references.NoneDirection, references.Position{}
}

func assertRemainingSmallMapGuardsHostile(
	t *testing.T,
	state *GameState,
	excluded *map_units.NPCFriendly,
) {
	t.Helper()
	guardCount := 0
	for _, unit := range *state.CurrentNPCAIController.GetNpcs() {
		guard, isFriendly := unit.(*map_units.NPCFriendly)
		if !isFriendly || guard == excluded || !guard.IsVisible() ||
			guard.NPCReference.GetNPCType() != references.Guard {
			continue
		}
		guardCount++
		assert.True(t, guardActivationNPCIsHostile(guard))
	}
	require.Positive(t, guardCount, "fixture must retain another Guard to verify hostility consequences")
}

func windemereDevilBySlot(t *testing.T, state *GameState, slot int) *map_units.NPCFriendly {
	t.Helper()
	for _, unit := range *state.CurrentNPCAIController.GetNpcs() {
		friendly, ok := unit.(*map_units.NPCFriendly)
		if !ok || friendly.MapUnitDetails().NPCNum != slot || friendly.NPCReference.Location != references.Windemere ||
			(slot != 10 && slot != 11) {
			continue
		}
		if monster, classified := friendly.NPCReference.SmallMapCombatMonsterType(); classified && monster == references.MonsterDevil {
			return friendly
		}
	}

	t.Fatalf("Windemere devil slot %d is not active", slot)

	return nil
}

func visitLocalRatOrBatBySlot(
	t *testing.T,
	state *GameState,
	location references.Location,
	slot int,
	monster references.MonsterType,
) *map_units.NPCFriendly {
	t.Helper()
	for _, unit := range *state.CurrentNPCAIController.GetNpcs() {
		friendly, ok := unit.(*map_units.NPCFriendly)
		if !ok || friendly.NPCReference.Location != location || friendly.MapUnitDetails().NPCNum != slot {
			continue
		}
		classifiedMonster, classified := friendly.NPCReference.SmallMapCombatMonsterType()
		if classified && classifiedMonster == monster {
			return friendly
		}
	}

	t.Fatalf("%s slot %d is not an active %s", location, slot, monster)

	return nil
}

func combatConclusionName(reason CombatEndReason) string {
	switch reason {
	case CombatVictory:
		return "victory"
	case CombatDefeat:
		return "defeat or flight"
	case CombatEvacuated:
		return "evacuated"
	case CombatNotEnded:
		return "not ended"
	}

	return "unknown"
}
