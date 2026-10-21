package references

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
)

func TestLoadCombatMapParsesOriginalMonsterPositionLayout(t *testing.T) {
	t.Parallel()

	const (
		originalMonsterXRow = 6
		originalMonsterYRow = 7
		metadataOffset      = 11
	)

	data := make([]byte, CombatMapDataSize)

	// Poison the obsolete row-5/interleaved layout so it cannot accidentally pass.
	for i := metadataOffset; i < CombatMapBytesPerRow; i++ {
		data[5*CombatMapBytesPerRow+i] = 0xee
	}

	for monsterIndex := range MaxMonstersPerMap {
		data[originalMonsterXRow*CombatMapBytesPerRow+metadataOffset+monsterIndex] = byte(monsterIndex + 1)
		data[originalMonsterYRow*CombatMapBytesPerRow+metadataOffset+monsterIndex] = byte(monsterIndex + 101)
	}

	// Verify adjacent metadata parsing remains unchanged.
	data[2*CombatMapBytesPerRow+PlayerXCoordOffset] = 4
	data[2*CombatMapBytesPerRow+PlayerYCoordOffset] = 9
	data[3*CombatMapBytesPerRow+2] = 0x7f

	dataPath := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dataPath, files.BritCbt), data, 0o600))

	combatMap, err := LoadCombatMap(0, BritanniaCombatTerritory, combatMapTestConfig(dataPath))
	require.NoError(t, err)

	for monsterIndex := range MaxMonstersPerMap {
		assert.Equal(t, Position{
			X: Coordinate(monsterIndex + 1),
			Y: Coordinate(monsterIndex + 101),
		}, combatMap.MonsterPositions[monsterIndex])
	}

	assert.Equal(t, Position{X: 4, Y: 9}, combatMap.PlayerPositions[1][0])
	assert.Equal(t, indexes.SpriteIndex(0x7f), combatMap.GetTileNumber(Position{X: 2, Y: 3}))
}

func TestLoadCombatMapParsesFixedDungeonRoomEntityDescriptors(t *testing.T) {
	t.Parallel()

	data := make([]byte, CombatMapDataSize)
	for slot := range MaxMonstersPerMap {
		data[DungeonEntityRow*CombatMapBytesPerRow+DungeonEntityOffset+slot] = byte(slot + 1)
	}
	dataPath := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dataPath, files.DungeonCbt), data, 0o600))

	combatMap, err := LoadCombatMap(0, DungeonCombatTerritory, combatMapTestConfig(dataPath))
	require.NoError(t, err)
	for slot := range MaxMonstersPerMap {
		assert.Equal(t, byte(slot+1), combatMap.DungeonEntityTiles[slot])
	}
}

func TestLoadCombatMapParsesAllDungeonTriggerSlotsFromExactOffsets(t *testing.T) {
	t.Parallel()

	data := make([]byte, CombatMapDataSize)
	want := [DungeonCombatTriggerSlotCount]DungeonCombatMapTrigger{}
	for slotIndex := range DungeonCombatTriggerSlotCount {
		trigger := Position{X: Coordinate(slotIndex), Y: Coordinate(slotIndex + 1)}
		if slotIndex == 0 {
			trigger = Position{X: 0, Y: 0}
		}
		want[slotIndex] = DungeonCombatMapTrigger{
			ReplacementTile: indexes.SpriteIndex(0x20 + slotIndex),
			Trigger:         trigger,
			Destinations: [2]Position{
				{X: Coordinate(slotIndex + 1), Y: Coordinate(slotIndex + 2)},
				{X: Coordinate(10 - slotIndex), Y: Coordinate(9 - slotIndex)},
			},
		}
		setDungeonTriggerSlot(data, slotIndex, want[slotIndex])
	}
	dataPath := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dataPath, files.DungeonCbt), data, 0o600))

	combatMap, err := LoadCombatMap(0, DungeonCombatTerritory, combatMapTestConfig(dataPath))
	require.NoError(t, err)

	got, present := combatMap.DungeonTriggerSlots()
	require.True(t, present)
	assert.Equal(t, want, got)
	assert.Equal(t, Position{X: 0, Y: 0}, got[0].Trigger, "zero coordinate is an active source value")
}

func TestCombatMapTriggerSlotsAreAbsentWithoutDungeonCBTParsing(t *testing.T) {
	t.Parallel()

	generated := NewCombatMapReference(0, DungeonCombatTerritory)
	_, present := generated.DungeonTriggerSlots()
	assert.False(t, present, "generated dungeon maps must not inherit zero-valued slots")

	dataPath := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dataPath, files.BritCbt),
		make([]byte, CombatMapDataSize),
		0o600,
	))
	britannia, err := LoadCombatMap(0, BritanniaCombatTerritory, combatMapTestConfig(dataPath))
	require.NoError(t, err)
	_, present = britannia.DungeonTriggerSlots()
	assert.False(t, present, "BRIT.CBT metadata bytes are not dungeon trigger slots")
}

func TestResolveDungeonTriggerConsumesDuplicatesAndAppliesDestinationsInSlotOrder(t *testing.T) {
	t.Parallel()

	data := make([]byte, CombatMapDataSize)
	for row := range YCombatMapTiles {
		for column := range CombatMapTileDataBytes {
			data[int(row)*CombatMapBytesPerRow+column] = 1
		}
	}
	for slotIndex := range DungeonCombatTriggerSlotCount {
		setDungeonTriggerSlot(data, slotIndex, DungeonCombatMapTrigger{
			Trigger: Position{X: DungeonTriggerConsumedPosition, Y: DungeonTriggerConsumedPosition},
		})
	}
	trigger := Position{X: 3, Y: 4}
	setDungeonTriggerSlot(data, 0, DungeonCombatMapTrigger{
		ReplacementTile: 10,
		Trigger:         trigger,
		Destinations: [2]Position{
			{X: 0, Y: 0},
			{X: XCombatMapTiles, Y: 1},
		},
	})
	setDungeonTriggerSlot(data, 1, DungeonCombatMapTrigger{
		ReplacementTile: 20,
		Trigger:         trigger,
		Destinations: [2]Position{
			{X: 0, Y: 0},
			{X: 1, Y: 1},
		},
	})
	setDungeonTriggerSlot(data, 2, DungeonCombatMapTrigger{
		ReplacementTile: 30,
		Trigger:         trigger,
		Destinations: [2]Position{
			{X: 2, Y: YCombatMapTiles},
			{X: DungeonTriggerConsumedPosition, Y: DungeonTriggerConsumedPosition},
		},
	})
	dataPath := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dataPath, files.DungeonCbt), data, 0o600))
	combatMap, err := LoadCombatMap(0, DungeonCombatTerritory, combatMapTestConfig(dataPath))
	require.NoError(t, err)

	assert.True(t, combatMap.ResolveDungeonTrigger(trigger))
	assert.Equal(t, indexes.SpriteIndex(20), combatMap.GetTileNumber(Position{X: 0, Y: 0}),
		"later duplicate-coordinate writes win")
	assert.Equal(t, indexes.SpriteIndex(20), combatMap.GetTileNumber(Position{X: 1, Y: 1}))
	assert.Equal(t, indexes.SpriteIndex(1), combatMap.GetTileNumber(Position{X: 2, Y: 2}),
		"a destination pair is invalid when either coordinate is outside the map")
	slots, present := combatMap.DungeonTriggerSlots()
	require.True(t, present)
	for slotIndex := range 3 {
		assert.Equal(t, Position{
			X: DungeonTriggerConsumedPosition,
			Y: DungeonTriggerConsumedPosition,
		}, slots[slotIndex].Trigger)
	}
	assert.False(t, combatMap.ResolveDungeonTrigger(trigger), "consumed slots cannot fire twice")

	freshMap, loadErr := LoadCombatMap(0, DungeonCombatTerritory, combatMapTestConfig(dataPath))
	require.NoError(t, loadErr)
	freshSlots, present := freshMap.DungeonTriggerSlots()
	require.True(t, present)
	assert.Equal(t, trigger, freshSlots[0].Trigger, "fresh room loads restore source slots")
	assert.Equal(t, indexes.SpriteIndex(1), freshMap.GetTileNumber(Position{X: 0, Y: 0}),
		"trigger tile substitutions are encounter-local")
}

func setDungeonTriggerSlot(data []byte, slotIndex int, trigger DungeonCombatMapTrigger) {
	data[DungeonTriggerTileRow*CombatMapBytesPerRow+DungeonTriggerXOffset+slotIndex] = byte(trigger.ReplacementTile)
	data[DungeonTriggerCoordinateRow*CombatMapBytesPerRow+DungeonTriggerXOffset+slotIndex] = byte(trigger.Trigger.X)
	data[DungeonTriggerCoordinateRow*CombatMapBytesPerRow+DungeonTriggerYOffset+slotIndex] = byte(trigger.Trigger.Y)
	data[DungeonTriggerDestination1Row*CombatMapBytesPerRow+DungeonTriggerXOffset+slotIndex] =
		byte(trigger.Destinations[0].X)
	data[DungeonTriggerDestination1Row*CombatMapBytesPerRow+DungeonTriggerYOffset+slotIndex] =
		byte(trigger.Destinations[0].Y)
	data[DungeonTriggerDestination2Row*CombatMapBytesPerRow+DungeonTriggerXOffset+slotIndex] =
		byte(trigger.Destinations[1].X)
	data[DungeonTriggerDestination2Row*CombatMapBytesPerRow+DungeonTriggerYOffset+slotIndex] =
		byte(trigger.Destinations[1].Y)
}

func TestLoadCombatMapMonsterPositionsMatchOriginalData(t *testing.T) {
	t.Parallel()

	dataPath := combatTestDataPath(t)
	data, err := os.ReadFile(filepath.Join(dataPath, files.BritCbt))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(data), TotalCombatMaps*CombatMapDataSize)

	for mapIndex := range TotalCombatMaps {
		combatMap, loadErr := LoadCombatMap(mapIndex, BritanniaCombatTerritory, combatMapTestConfig(dataPath))
		require.NoError(t, loadErr)

		mapOffset := mapIndex * CombatMapDataSize
		for monsterIndex := range MaxMonstersPerMap {
			expected := Position{
				X: Coordinate(data[mapOffset+6*CombatMapBytesPerRow+11+monsterIndex]),
				Y: Coordinate(data[mapOffset+7*CombatMapBytesPerRow+11+monsterIndex]),
			}
			assert.Equal(t, expected, combatMap.MonsterPositions[monsterIndex],
				"map %d monster position %d", mapIndex, monsterIndex)
		}
	}
}

func TestLoadDungeonCombatMapEntitiesMatchOriginalData(t *testing.T) {
	t.Parallel()

	dataPath := dungeonCombatTestDataPath(t)
	data, err := os.ReadFile(filepath.Join(dataPath, files.DungeonCbt))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(data), (MaxDungeonLinearIndex+1)*CombatMapDataSize)

	for mapIndex := 0; mapIndex <= MaxDungeonLinearIndex; mapIndex++ {
		combatMap, loadErr := LoadCombatMap(mapIndex, DungeonCombatTerritory, combatMapTestConfig(dataPath))
		require.NoError(t, loadErr)
		mapOffset := mapIndex * CombatMapDataSize
		for slot := range MaxMonstersPerMap {
			expected := data[mapOffset+DungeonEntityRow*CombatMapBytesPerRow+DungeonEntityOffset+slot]
			assert.Equal(t, expected, combatMap.DungeonEntityTiles[slot], "map %d slot %d", mapIndex, slot)
		}
	}
}

func TestLoadDungeonCombatMapTriggersMatchOriginalData(t *testing.T) {
	t.Parallel()

	dataPath := dungeonCombatTestDataPath(t)
	data, err := os.ReadFile(filepath.Join(dataPath, files.DungeonCbt))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(data), (MaxDungeonLinearIndex+1)*CombatMapDataSize)

	for mapIndex := 0; mapIndex <= MaxDungeonLinearIndex; mapIndex++ {
		combatMap, loadErr := LoadCombatMap(mapIndex, DungeonCombatTerritory, combatMapTestConfig(dataPath))
		require.NoError(t, loadErr)
		slots, present := combatMap.DungeonTriggerSlots()
		require.True(t, present)
		mapOffset := mapIndex * CombatMapDataSize
		for slotIndex := range DungeonCombatTriggerSlotCount {
			expected := DungeonCombatMapTrigger{
				ReplacementTile: indexes.SpriteIndex(data[mapOffset+DungeonTriggerTileRow*CombatMapBytesPerRow+DungeonTriggerXOffset+slotIndex]),
				Trigger: Position{
					X: Coordinate(data[mapOffset+DungeonTriggerCoordinateRow*CombatMapBytesPerRow+DungeonTriggerXOffset+slotIndex]),
					Y: Coordinate(data[mapOffset+DungeonTriggerCoordinateRow*CombatMapBytesPerRow+DungeonTriggerYOffset+slotIndex]),
				},
				Destinations: [2]Position{
					{
						X: Coordinate(data[mapOffset+DungeonTriggerDestination1Row*CombatMapBytesPerRow+DungeonTriggerXOffset+slotIndex]),
						Y: Coordinate(data[mapOffset+DungeonTriggerDestination1Row*CombatMapBytesPerRow+DungeonTriggerYOffset+slotIndex]),
					},
					{
						X: Coordinate(data[mapOffset+DungeonTriggerDestination2Row*CombatMapBytesPerRow+DungeonTriggerXOffset+slotIndex]),
						Y: Coordinate(data[mapOffset+DungeonTriggerDestination2Row*CombatMapBytesPerRow+DungeonTriggerYOffset+slotIndex]),
					},
				},
			}
			assert.Equal(t, expected, slots[slotIndex], "map %d trigger slot %d", mapIndex, slotIndex)
		}
	}
}

func combatMapTestConfig(dataPath string) *config.UltimaVConfiguration {
	gameConfig := &config.UltimaVConfiguration{}
	gameConfig.SavedConfigData = &config.UltimaVConfigurationFlags{
		Resolution:   0,
		FullScreen:   false,
		DataFilePath: dataPath,
	}

	return gameConfig
}

func combatTestDataPath(t *testing.T) string {
	return combatDataFilePath(t, files.BritCbt)
}

func dungeonCombatTestDataPath(t *testing.T) string {
	return combatDataFilePath(t, files.DungeonCbt)
}

func combatDataFilePath(t *testing.T, filename string) string {
	t.Helper()

	return originaldata.Directory(t, filename)
}
