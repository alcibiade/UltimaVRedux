package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestSceptreClearsWrappedLargeMapBarriersInSourceOrder(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(0, 0).
		WithSystemCallbacks().
		Build()
	state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
	positions := []references.Position{
		{X: 255, Y: 255}, {X: 255, Y: 0}, {X: 255, Y: 1},
		{X: 0, Y: 255}, {X: 0, Y: 0}, {X: 0, Y: 1},
		{X: 1, Y: 255}, {X: 1, Y: 0}, {X: 1, Y: 1},
	}
	setSceptreTestBarriers(state, positions)

	cleared := state.ClearNearbySceptreBarriers()

	assert.Equal(t, positions, cleared)
	assertSceptreTestPositionsAreGrass(t, state, positions)
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
}

func TestSceptreClipsSmallMapScanWithoutMutatingOverflow(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(0, 0).
		WithSystemCallbacks().
		Build()
	state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
	positions := []references.Position{{X: 0, Y: 0}, {X: 0, Y: 1}, {X: 1, Y: 0}, {X: 1, Y: 1}}
	setSceptreTestBarriers(state, positions)
	overflow := references.Position{X: -1, Y: -1}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &overflow, indexes.ShadowlordBoundary16)

	cleared := state.ClearNearbySceptreBarriers()

	assert.Equal(t, positions, cleared)
	assertSceptreTestPositionsAreGrass(t, state, positions)
	assert.Equal(t, indexes.SpriteIndex(indexes.ShadowlordBoundary16),
		state.GetCurrentLayeredMap().SpriteIndexByLayer(map_state.MapLayer, overflow))
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
}

func TestSceptreBarrierClearSurvivesNativeSnapshot(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	barrier := references.Position{X: 14, Y: 15}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &barrier, indexes.ShadowlordBoundary1)
	require.Equal(t, []references.Position{barrier}, state.ClearNearbySceptreBarriers())

	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	hydrated, err := NewGameStateFromSaveSnapshot(
		snapshot,
		state.GameConfig,
		state.GameReferences,
		xTilesVisibleOnGameScreen,
		yTilesVisibleOnGameScreen,
	)
	require.NoError(t, err)

	assert.Equal(t, indexes.SpriteIndex(indexes.Grass),
		hydrated.GetCurrentLayeredMap().SpriteIndexByLayer(map_state.MapLayer, barrier))
}

func TestCombatSceptreClearsTerrainInBothCombatRepresentations(t *testing.T) {
	state, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
	activeIdx := state.CombatScene.GetActiveCombatantIndex()
	state.CombatScene.SetPartyMemberPosition(activeIdx, references.Position{X: 2, Y: 2})
	positions := []references.Position{{X: 1, Y: 1}, {X: 1, Y: 2}, {X: 2, Y: 2}, {X: 3, Y: 3}}
	for index, position := range positions {
		state.setCombatTerrain(position, indexes.SpriteIndex(int(indexes.ShadowlordBoundary1)+index))
	}
	fieldPosition := references.Position{X: 2, Y: 3}
	require.True(t, state.CombatScene.AddField(NewCombatField(CombatFieldFire, fieldPosition)))

	cleared := state.ClearNearbySceptreBarriers()

	assert.Equal(t, positions, cleared)
	for _, position := range positions {
		assert.Equal(t, indexes.SpriteIndex(indexes.Grass), state.CombatScene.CombatMapRef.GetTileNumber(position))
		assert.Equal(t, indexes.SpriteIndex(indexes.Grass),
			state.GetCurrentLayeredMap().SpriteIndexByLayer(map_state.MapLayer, position))
	}
	require.Len(t, state.CombatScene.Fields(), 1)
	assert.Equal(t, fieldPosition, state.CombatScene.Fields()[0].Position())
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
}

func TestCombatSceptreUsesExactClonePosition(t *testing.T) {
	state, mock, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	identity := state.CombatScene.GetCurrentTurnIdentity()
	originalPosition := references.Position{X: 7, Y: 7}
	clonePosition := references.Position{X: 2, Y: 2}
	state.CombatScene.SetPartyMemberPosition(identity.EntityIndex, originalPosition)
	clone := state.CombatScene.addPCCloneFromSource(
		identity,
		identity.EntityIndex,
		clonePosition,
		NewCombatPCProvider(state.PartyState.GetCharacter(identity.EntityIndex)),
	)
	state.CombatScene.CurrentTurnIdentity = clone
	state.CombatScene.CurrentTurnIndex = clone.EntityIndex
	cloneBarrier := references.Position{X: 2, Y: 1}
	originalBarrier := references.Position{X: 7, Y: 6}
	state.setCombatTerrain(cloneBarrier, indexes.ShadowlordBoundary1)
	state.setCombatTerrain(originalBarrier, indexes.ShadowlordBoundary2)
	originalField := references.Position{X: 7, Y: 8}
	cloneField := references.Position{X: 2, Y: 3}
	require.True(t, state.CombatScene.AddField(NewCombatField(CombatFieldFire, originalField)))
	require.True(t, state.CombatScene.AddField(NewCombatField(CombatFieldSleep, cloneField)))

	cleared := state.ClearNearbySceptreBarriers()
	mock.Reset()
	require.True(t, state.UseCombatSceptre(references.Down))

	assert.Equal(t, []references.Position{cloneBarrier}, cleared)
	assert.Equal(t, indexes.SpriteIndex(indexes.Grass), state.CombatScene.CombatMapRef.GetTileNumber(cloneBarrier))
	assert.Equal(t, indexes.SpriteIndex(indexes.ShadowlordBoundary2),
		state.CombatScene.CombatMapRef.GetTileNumber(originalBarrier))
	require.Len(t, state.CombatScene.Fields(), 1)
	assert.Equal(t, originalField, state.CombatScene.Fields()[0].Position())
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, clone.Slot, mock.CombatEvents[0].Attacker.Slot)
	assert.Equal(t, clonePosition, mock.CombatEvents[0].Attacker.Position)
}

func TestDungeonSceptreClearsEveryPackedFieldVariant(t *testing.T) {
	for raw := 0x80; raw <= 0x8f; raw++ {
		t.Run(fmt.Sprintf("%02x", raw), func(t *testing.T) {
			state, callbacks := newDungeonInteractionTestState(t, references.DungeonCell(raw), uint64(raw))
			state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
			beforeRevision := state.activeDungeon.revision

			result := state.UseDungeonSceptre()

			require.True(t, result.Cleared)
			assert.Equal(t, state.MapState.PlayerLocation.Position, result.Change.Position)
			assert.Equal(t, references.DungeonCell(raw&0x08), currentDungeonInteractionCell(t, state))
			assert.Equal(t, beforeRevision+1, result.Change.Revision)
			assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetCurrentLayeredMap().SpriteIndexByLayer(
				map_state.MapLayer,
				state.MapState.PlayerLocation.Position,
			))
			assert.Equal(t, []string{"Field destroyed!"}, callbacks.Messages)
			assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
		})
	}
}

func TestDungeonSceptreUsesCurrentBeforeWrappedFacingCell(t *testing.T) {
	state, _ := newDungeonInteractionTestState(t, 0x80, 42)
	state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
	state.SetPlayerPosition(references.Position{X: 7, Y: 1})
	state.activeDungeon.setFacingAndLastDirection(references.DungeonFacingEast)
	current := state.MapState.PlayerLocation.Position
	facing := references.Position{X: 0, Y: 1}
	setDungeonInteractionCell(t, state, current, 0x8a)
	setDungeonInteractionCell(t, state, facing, 0x8b)

	first := state.UseDungeonSceptre()

	require.True(t, first.Cleared)
	assert.Equal(t, current, first.Change.Position)
	assert.Equal(t, references.DungeonCell(0x08), dungeonInteractionCellAt(t, state, current))
	assert.Equal(t, references.DungeonCell(0x8b), dungeonInteractionCellAt(t, state, facing))

	second := state.UseDungeonSceptre()

	require.True(t, second.Cleared)
	assert.Equal(t, facing, second.Change.Position)
	assert.Equal(t, references.DungeonCell(0x08), dungeonInteractionCellAt(t, state, facing))
}

func TestDungeonSceptreReportsNoEffectWithoutMutation(t *testing.T) {
	state, callbacks := newDungeonInteractionTestState(t, 0x00, 42)
	state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
	state.activeDungeon.setFacingAndLastDirection(references.DungeonFacingEast)
	setDungeonInteractionCell(t, state, references.Position{X: 2, Y: 1}, 0x70)
	beforeRevision := state.activeDungeon.revision
	callbacks.Reset()

	result := state.UseDungeonSceptre()

	assert.False(t, result.Cleared)
	assert.Equal(t, beforeRevision, state.activeDungeon.revision)
	assert.Equal(t, []string{"No effect!"}, callbacks.Messages)
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
}

func setSceptreTestBarriers(state *GameState, positions []references.Position) {
	for index, position := range positions {
		tile := indexes.SpriteIndex(int(indexes.ShadowlordBoundary1) + index%16)
		state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, tile)
	}
}

func assertSceptreTestPositionsAreGrass(t *testing.T, state *GameState, positions []references.Position) {
	t.Helper()
	for _, position := range positions {
		assert.Equal(t, indexes.SpriteIndex(indexes.Grass),
			state.GetCurrentLayeredMap().SpriteIndexByLayer(map_state.MapLayer, position))
	}
}
