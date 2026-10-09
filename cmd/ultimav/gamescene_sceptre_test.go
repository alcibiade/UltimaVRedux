package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestSurfaceSceptreClearsBarriersWithoutDirectionAndCompletesTurn(t *testing.T) {
	state, mock := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	clearUsableItemsForDungeonView(state)
	state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
	barrier := references.Position{X: 14, Y: 16}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &barrier, indexes.ShadowlordBoundary1)
	expectedTime := state.DateTime
	expectedTime.Advance(1)
	mock.Reset()
	scene := &GameScene{
		gameState:   state,
		output:      text.NewOutput(nil, 1, 10, 80),
		keyboard:    input.NewKeyboard(0),
		spriteSheet: &sprites.SpriteSheet{},
	}
	state.SystemCallbacks.Flow.Items.StartSceptreWorkflow = scene.startSceptreWorkflow

	scene.StartUseWorkflow()
	dialog := waitForCombatItemDialog(t, scene)
	dialog.HandleEnter()
	drainCombatUseWorkflow(t, scene)
	drainSceptreWorkflow(t, scene)

	assert.Equal(t, indexes.SpriteIndex(indexes.Grass),
		state.GetCurrentLayeredMap().SpriteIndexByLayer(map_state.MapLayer, barrier))
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
	assert.Contains(t, mock.Messages, "Wielding the Sceptre of Lord British...")
	assert.NotContains(t, mock.Messages, "Direction-")
	assert.NotContains(t, mock.Messages, "Field dissolved!")
	assert.NotContains(t, mock.Messages, "No effect!")
	assert.Equal(t, expectedTime, state.DateTime)
}

func TestSurfaceSceptreSafeFallbackReportsNoEffectAndCompletesTurn(t *testing.T) {
	state, mock := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
	for x := references.Coordinate(14); x <= 16; x++ {
		for y := references.Coordinate(14); y <= 16; y++ {
			position := references.Position{X: x, Y: y}
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Grass)
		}
	}
	expectedTime := state.DateTime
	expectedTime.Advance(1)
	mock.Reset()
	scene := &GameScene{gameState: state, keyboard: input.NewKeyboard(0)}

	scene.StartSceptreWorkflow()
	drainSceptreWorkflow(t, scene)

	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
	assert.Contains(t, mock.Messages, "No effect!")
	assert.NotContains(t, mock.Messages, "Direction-")
	assert.Equal(t, expectedTime, state.DateTime)
}

func TestLargeMapSceptreUsesWrappedScanAndSurfaceTurnCost(t *testing.T) {
	state, mock := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(0, 0).
		WithRandomSeed(1).
		WithSystemCallbacks().
		Build()
	state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
	current := state.MapState.PlayerLocation.Position
	barrier := references.Position{X: 255, Y: 255}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &current, indexes.Grass)
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &barrier, indexes.ShadowlordBoundary16)
	expectedTime := state.DateTime
	expectedTime.Advance(2)
	mock.Reset()
	scene := &GameScene{gameState: state, keyboard: input.NewKeyboard(0)}

	scene.StartSceptreWorkflow()
	drainSceptreWorkflow(t, scene)

	assert.Equal(t, indexes.SpriteIndex(indexes.Grass),
		state.GetCurrentLayeredMap().SpriteIndexByLayer(map_state.MapLayer, barrier))
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
	assert.Equal(t, expectedTime, state.DateTime)
}

func drainSceptreWorkflow(t *testing.T, scene *GameScene) {
	t.Helper()
	for range 10 {
		scene.gameState.ElapsedMs += DialogPollingRateMs
		scene.gameState.EventQueueManager.Update()
		if !scene.gameState.EventQueueManager.HasWorkflow("sceptre_workflow") {
			return
		}
	}
	t.Fatal("Sceptre workflow did not complete")
}
