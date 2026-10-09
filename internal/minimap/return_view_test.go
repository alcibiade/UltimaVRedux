package minimap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestDecodeReturnViewDataBuildsMapsAndTypedScript(t *testing.T) {
	data := make([]byte, returnViewScriptOffset+returnViewScriptBytes)
	for mapIndex := range returnViewMapCount {
		mapOffset := returnViewMapsOffset + mapIndex*returnViewMapBytes
		for row := range returnViewMapHeight {
			for column := range returnViewMapWidth {
				data[mapOffset+row*returnViewMapStoredCols+column] = byte(mapIndex + row + column)
			}
		}
	}
	script := data[returnViewScriptOffset:]
	copy(script, []byte{0x06, 0, 0x0c, 0x03, 1})
	for index := 5; index < len(script)-1; index++ {
		script[index] = 0x0c
	}
	script[len(script)-1] = 0x09

	decoded, err := DecodeReturnViewData(data)

	require.NoError(t, err)
	require.Len(t, decoded.Maps, returnViewMapCount)
	first := decoded.Maps[returnViewMapID(0)]
	assert.Equal(t, returnViewMapWidth, first.Width)
	assert.Equal(t, returnViewMapHeight, first.Height)
	assert.Equal(t, indexes.SpriteIndex(1), first.Tiles[1])
	require.NoError(t, decoded.Script.Validate())
	assert.Equal(t, TileCommandSetMap, decoded.Script.Steps[0].TileCommand.Type)
	assert.Equal(t, TileCommandReset, decoded.Script.Steps[len(decoded.Script.Steps)-1].TileCommand.Type)
}

func TestDecodeReturnViewScriptExpandsOriginalLoop(t *testing.T) {
	data := []byte{
		0x06, 0,
		0x0e, 3,
		0x0d, 0, 1,
		0x0f,
		0x09,
	}

	script, err := decodeReturnViewScript(data)

	require.NoError(t, err)
	moveCount, refreshCount := 0, 0
	for _, step := range script.Steps {
		switch step.TileCommand.Type {
		case TileCommandMoveActor:
			moveCount++
		case TileCommandRefresh:
			refreshCount++
		case TileCommandSetMap, TileCommandClearActors, TileCommandAddActor, TileCommandRemoveActor,
			TileCommandSetTile, TileCommandRaiseGate, TileCommandDropGate, TileCommandFizzActor,
			TileCommandFadeActor, TileCommandNukActor, TileCommandReset:
		}
	}
	assert.Equal(t, 3, moveCount)
	assert.Equal(t, 3, refreshCount)
}

func TestDecodeReturnViewScriptRejectsMalformedData(t *testing.T) {
	_, err := decodeReturnViewScript([]byte{0x06})
	require.ErrorContains(t, err, "truncated operands")

	_, err = decodeReturnViewScript([]byte{0xff})
	require.ErrorContains(t, err, "unknown opcode")

	_, err = decodeReturnViewScript([]byte{0x00, 32, 5, 1, 1, 0x09})
	require.ErrorContains(t, err, "object index 32")

	_, err = decodeReturnViewScript([]byte{0x0a, 5, 19, 0, 0x09})
	require.ErrorContains(t, err, "outside the 19x4 map")
}

func TestDecodeReturnViewActorUsesOriginalObjectSpriteBank(t *testing.T) {
	script, err := decodeReturnViewScript([]byte{0x06, 0, 0x00, 0, 5, 1, 1, 0x09})

	require.NoError(t, err)
	assert.Equal(t, indexes.SpriteIndex(261), script.Steps[1].TileCommand.Sprite)
}

func TestDecodeReturnViewFizzesActorEnteringMoongate(t *testing.T) {
	script, err := decodeReturnViewScript([]byte{
		0x06, 0,
		0x00, 0, 5, 13, 2,
		0x04, 13, 1,
		0x0d, 0, 0,
		0x01, 0,
		0x09,
	})

	require.NoError(t, err)
	assert.Equal(t, TileCommandFadeActor, script.Steps[5].TileCommand.Type)
	assert.Equal(t, 12*ReturnViewRefreshMs, script.Steps[5].TileCommand.DurationMs)
}

func TestDecodeReturnViewFizzesAvatarAppearingAtMoongate(t *testing.T) {
	script, err := decodeReturnViewScript([]byte{
		0x06, 2,
		0x04, 9, 1,
		0x00, 0, returnViewPlayerSprite, 9, 1,
		0x09,
	})

	require.NoError(t, err)
	assert.Equal(t, TileCommandAddActor, script.Steps[2].TileCommand.Type)
	assert.Equal(t, TileCommandFizzActor, script.Steps[3].TileCommand.Type)
	assert.Equal(t, returnViewActorID(0), script.Steps[3].TileCommand.ActorID)
}

func TestDecodeReturnViewImmediatelyRemovesActorAwayFromMoongate(t *testing.T) {
	script, err := decodeReturnViewScript([]byte{
		0x06, 0,
		0x00, 0, 5, 1, 1,
		0x01, 0,
		0x09,
	})

	require.NoError(t, err)
	assert.Equal(t, TileCommandRemoveActor, script.Steps[2].TileCommand.Type)
}

func TestScriptExecutorRunsAndInterruptsLoopingTileScene(t *testing.T) {
	scene := NewMiniMapScene()
	require.NoError(t, scene.RegisterTileMap("map", TileMap{
		Width: 2, Height: 1, Tiles: []indexes.SpriteIndex{indexes.Grass, indexes.Grass}, Caption: "Map",
	}))
	script := NewMiniMapScript("tile").
		TileCommand(TileCommand{Type: TileCommandSetMap, MapID: "map"}).
		TileCommand(TileCommand{
			Type: TileCommandAddActor, ActorID: "actor", Sprite: indexes.AvatarKeyIndex,
			Position: references.Position{X: 0, Y: 0},
		}).
		TileCommand(TileCommand{Type: TileCommandMoveActor, ActorID: "actor", Direction: references.Right}).
		TileCommand(TileCommand{Type: TileCommandRefresh, Cycles: 1, CycleMs: 10}).
		TileCommand(TileCommand{Type: TileCommandReset}).
		Build()
	executor := NewScriptExecutor(scene, script)

	executor.Update(0)
	assert.Equal(t, references.Position{X: 1, Y: 0}, scene.GetActorPosition("actor"))
	executor.Update(10)
	executor.Update(0)
	assert.False(t, executor.IsComplete())
	executor.Interrupt()
	assert.True(t, executor.IsComplete())
}

func TestScriptExecutorCarriesOverflowAcrossTileCommands(t *testing.T) {
	scene := NewMiniMapScene()
	require.NoError(t, scene.RegisterTileMap("map", TileMap{
		Width: 1, Height: 1, Tiles: []indexes.SpriteIndex{indexes.Grass}, Caption: "Map",
	}))
	script := NewMiniMapScript("timing").
		TileCommand(TileCommand{Type: TileCommandSetMap, MapID: "map"}).
		TileCommand(TileCommand{Type: TileCommandRefresh, Cycles: 1, CycleMs: 56}).
		TileCommand(TileCommand{Type: TileCommandRefresh, Cycles: 1, CycleMs: 56}).
		Build()
	executor := NewScriptExecutor(scene, script)

	executor.Update(112)

	assert.True(t, executor.IsComplete())
}

func TestTileSceneRevealExpandsBothSidesEverySecondRefresh(t *testing.T) {
	scene := NewMiniMapScene()
	tiles := make([]indexes.SpriteIndex, 19)
	require.NoError(t, scene.RegisterTileMap("map", TileMap{Width: 19, Height: 1, Tiles: tiles, Caption: "Map"}))
	script := NewMiniMapScript("reveal").
		TileCommand(TileCommand{Type: TileCommandSetMap, MapID: "map"}).
		TileCommand(TileCommand{Type: TileCommandRefresh, Cycles: 1, CycleMs: 10}).
		TileCommand(TileCommand{Type: TileCommandRefresh, Cycles: 1, CycleMs: 10}).
		Build()
	executor := NewScriptExecutor(scene, script)

	executor.Update(10)
	assert.False(t, scene.IsColumnRevealed(8))
	assert.True(t, scene.IsColumnRevealed(9))
	assert.False(t, scene.IsColumnRevealed(10))
	executor.Update(10)
	assert.True(t, scene.IsColumnRevealed(8))
	assert.True(t, scene.IsColumnRevealed(10))
}

func TestTileSceneMapChangeResetsRevealCadence(t *testing.T) {
	scene := NewMiniMapScene()
	tiles := make([]indexes.SpriteIndex, 19)
	require.NoError(t, scene.RegisterTileMap("first", TileMap{Width: 19, Height: 1, Tiles: tiles, Caption: "First"}))
	require.NoError(t, scene.RegisterTileMap("second", TileMap{Width: 19, Height: 1, Tiles: tiles, Caption: "Second"}))
	script := NewMiniMapScript("maps").
		TileCommand(TileCommand{Type: TileCommandSetMap, MapID: "first"}).
		TileCommand(TileCommand{Type: TileCommandRefresh, Cycles: 2, CycleMs: 10}).
		TileCommand(TileCommand{Type: TileCommandSetMap, MapID: "second"}).
		Build()
	executor := NewScriptExecutor(scene, script)

	executor.Update(20)

	assert.True(t, executor.IsComplete())
	assert.False(t, scene.IsColumnRevealed(8))
	assert.True(t, scene.IsColumnRevealed(9))
	assert.False(t, scene.IsColumnRevealed(10))
}

func TestTileSceneResetCarriesRemainingTimeIntoNextLoop(t *testing.T) {
	scene := NewMiniMapScene()
	require.NoError(t, scene.RegisterTileMap("map", TileMap{
		Width: 1, Height: 1, Tiles: []indexes.SpriteIndex{indexes.Grass}, Caption: "Map",
	}))
	script := NewMiniMapScript("loop").
		TileCommand(TileCommand{Type: TileCommandSetMap, MapID: "map"}).
		TileCommand(TileCommand{Type: TileCommandRefresh, Cycles: 1, CycleMs: 10}).
		TileCommand(TileCommand{Type: TileCommandReset}).
		Build()
	executor := NewScriptExecutor(scene, script)

	executor.Update(25)

	assert.False(t, executor.IsComplete())
	assert.InDelta(t, 0.5, executor.GetStepProgress(), 0.001)
}

func TestTileSceneRemoveAndReaddKeepsOneOrderedActor(t *testing.T) {
	scene := NewMiniMapScene()
	scene.AddActor("actor", references.Position{}, indexes.AvatarKeyIndex)
	scene.removeActor("actor")
	scene.AddActor("actor", references.Position{}, indexes.AvatarKeyIndex)

	assert.Len(t, scene.GetAllActors(), 1)
}
