package game_state

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBonusTerrainExcludesSurfaceObjectsAndReflection(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(10, 10).
		WithRandomSeed(522).
		WithSystemCallbacks().
		Build()
	position := references.Position{X: 10, Y: 9}
	layer := state.GetLayeredMapByCurrentLocation()
	layer.SetTileByLayer(map_state.MapLayer, &position, indexes.Mirror)
	layer.UnSetTileByLayer(map_state.MapOverrideLayer, &position)
	state.SurfaceCorpses = []SurfaceCorpse{
		{Class: references.SearchObjectDeadPC, Position: position, Location: references.Britain, Floor: 0},
		{Class: references.SearchObjectDeadMonster, Position: position, Location: references.Britain, Floor: 0},
	}
	require.NoError(t, state.ItemStacksMap.AddOrMerge(&position, references.ItemAndQuantity{Item: references.Food, Quantity: 3}))
	itemSprite := state.GameReferences.InventoryItemReferences.GetReferenceByItem(references.Food).ItemSprite

	// Each gameplay overlay retains its priority; terrain sampling never sees it.
	assert.Equal(t, itemSprite, state.GetEffectiveTile(&position))
	assert.Equal(t, indexes.SpriteIndex(indexes.Mirror), state.GetBonusTerrainTile(&position))
	require.NotNil(t, state.ItemStacksMap.Pop(&position))
	assert.Equal(t, indexes.SpriteIndex(indexes.Splat), state.GetEffectiveTile(&position))
	assert.Equal(t, indexes.SpriteIndex(indexes.Mirror), state.GetBonusTerrainTile(&position))
	state.SurfaceCorpses = state.SurfaceCorpses[:1]
	assert.Equal(t, indexes.SpriteIndex(indexes.DeadBody), state.GetEffectiveTile(&position))
	assert.Equal(t, indexes.SpriteIndex(indexes.Mirror), state.GetBonusTerrainTile(&position))
	state.SurfaceCorpses = nil
	assert.Equal(t, indexes.SpriteIndex(indexes.MirrorAvatar), state.GetEffectiveTile(&position))
	assert.Equal(t, indexes.SpriteIndex(indexes.Mirror), state.GetBonusTerrainTile(&position))

	state.MapState.PlayerLocation.Position.X++
	assert.Equal(t, indexes.SpriteIndex(indexes.Mirror), state.GetEffectiveTile(&position))
	assert.Equal(t, indexes.SpriteIndex(indexes.Mirror), state.GetBonusTerrainTile(&position))
	state.MapState.PlayerLocation.Position.X--
	layer.SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.MirrorBroken)
	assert.Equal(t, indexes.SpriteIndex(indexes.MirrorBroken), state.GetEffectiveTile(&position))
	assert.Equal(t, indexes.SpriteIndex(indexes.MirrorBroken), state.GetBonusTerrainTile(&position))
}

func TestBonusTerrainExcludesCombatObjects(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(10, 10).
		WithRandomSeed(522).
		WithSystemCallbacks().
		Build()
	require.NoError(t, state.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth))
	position := references.Position{X: 5, Y: 5}
	layer := state.GetLayeredMapByCurrentLocation()
	layer.SetTileByLayer(map_state.MapLayer, &position, indexes.WaterCoast)
	layer.SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.Swamp)
	require.NoError(t, state.ItemStacksMap.AddOrMerge(&position, references.ItemAndQuantity{Item: references.Food, Quantity: 3}))
	state.CombatScene.AddObject(NewCombatChest(position, 10, false))

	assert.Equal(t, indexes.SpriteIndex(indexes.Chest), state.GetEffectiveTile(&position))
	assert.Equal(t, indexes.SpriteIndex(indexes.Swamp), state.GetBonusTerrainTile(&position))
	state.CombatScene.AddObject(NewCombatLooseItem(position, references.Gold, 5, indexes.ItemMoney))
	assert.Equal(t, indexes.SpriteIndex(indexes.ItemMoney), state.GetEffectiveTile(&position))
	assert.Equal(t, indexes.SpriteIndex(indexes.Swamp), state.GetBonusTerrainTile(&position))
	layer.UnSetTileByLayer(map_state.MapOverrideLayer, &position)
	assert.Equal(t, indexes.SpriteIndex(indexes.WaterCoast), state.GetBonusTerrainTile(&position))
	assert.Equal(t, 2, state.CombatScene.ObjectCount())
}

func TestBonusTerrainLegacyQuestUnderlay(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(10, 10).
		WithRandomSeed(522).
		WithSystemCallbacks().
		Build()
	position := references.Position{X: 11, Y: 10}
	layer := state.GetLayeredMapByCurrentLocation()
	for _, test := range []struct {
		name   string
		item   references.QuestItem
		sprite indexes.SpriteIndex
	}{
		{name: "crown", item: references.Crown, sprite: indexes.Crown},
		{name: "sceptre", item: references.Sceptre, sprite: indexes.Sceptre},
		{name: "amulet", item: references.Amulet, sprite: indexes.Amulet},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, sourceLayer := range []map_state.LayerType{map_state.MapLayer, map_state.MapOverrideLayer} {
				layer.SetTileByLayer(map_state.MapLayer, &position, indexes.BrickFloor)
				layer.UnSetTileByLayer(map_state.MapOverrideLayer, &position)
				layer.SetTileByLayer(sourceLayer, &position, test.sprite)
				for _, quantity := range []uint16{0, 1} {
					state.PartyState.Inventory.QuestItems.Set(test.item, quantity)
					wantEffective := test.sprite
					if quantity != 0 {
						wantEffective = indexes.BrickFloor
					}
					assert.Equal(t, wantEffective, state.GetEffectiveTile(&position))
					assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetBonusTerrainTile(&position))
					assert.Equal(t, test.sprite, layer.GetTileTopMapOnlyTile(&position).Index)
				}
				layer.SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.Lava)
				assert.Equal(t, indexes.SpriteIndex(indexes.Lava), state.GetEffectiveTile(&position))
				assert.Equal(t, indexes.SpriteIndex(indexes.Lava), state.GetBonusTerrainTile(&position))
			}
		})
	}
}

func TestBonusTerrainModernQuestPlacementsPreserveTerrain(t *testing.T) {
	for _, placement := range GetAllQuestItemPlacements() {
		t.Run(fmt.Sprint(placement.QuestItemType), func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).
				WithLocation(placement.Location).
				WithFloor(placement.Floor).
				WithPlayerAt(int(placement.Position.X), int(placement.Position.Y)).
				WithRandomSeed(522).
				WithSystemCallbacks().
				Build()
			position := placement.Position
			layer := state.GetLayeredMapByCurrentLocation()
			layer.SetTileByLayer(map_state.MapLayer, &position, indexes.DryStone)
			layer.UnSetTileByLayer(map_state.MapOverrideLayer, &position)
			found, item := state.IsQuestItemAtPosition(position)
			require.True(t, found)
			require.Equal(t, placement.QuestItemType, item)
			for _, quantity := range []uint16{0, 1} {
				state.PartyState.Inventory.QuestItems.Set(item, quantity)
				assert.Equal(t, indexes.SpriteIndex(indexes.DryStone), state.GetEffectiveTile(&position))
				assert.Equal(t, indexes.SpriteIndex(indexes.DryStone), state.GetBonusTerrainTile(&position))
			}
			layer.SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.WaterCoast)
			assert.Equal(t, indexes.SpriteIndex(indexes.WaterCoast), state.GetBonusTerrainTile(&position))
		})
	}
}

func TestBonusTerrainMapStateAndTime(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.SerpentsHold).
		WithPlayerAt(10, 10).
		WithRandomSeed(522).
		WithSystemCallbacks().
		Build()
	position := references.Position{X: 14, Y: 22}
	layer := state.GetLayeredMapByCurrentLocation()
	arch := position.Up()
	require.Equal(t, indexes.SpriteIndex(indexes.BrickWallArchway), state.GetCurrentSmallLocationReference().GetTileNumber(arch, 0))
	require.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetCurrentSmallLocationReference().GetTileNumber(position, 0))
	for _, override := range []indexes.SpriteIndex{indexes.NoSprites, indexes.BrickFloor, indexes.Portcullis, indexes.StoneBrickWall} {
		layer.SetTileByLayer(map_state.MapOverrideLayer, &position, override)
		for _, hour := range []byte{4, 5, 19, 20} {
			state.DateTime.Hour = hour
			want := override
			if override == indexes.NoSprites {
				want = indexes.Portcullis
				if hour >= 5 && hour < 20 {
					want = indexes.BrickFloor
				}
			}
			assert.Equal(t, want, state.GetEffectiveTile(&position))
			assert.Equal(t, want, state.GetBonusTerrainTile(&position))
			assert.Equal(t, indexes.SpriteIndex(indexes.BrickWallArchway), state.GetEffectiveTile(&arch))
			assert.Equal(t, indexes.SpriteIndex(indexes.BrickWallArchway), state.GetBonusTerrainTile(&arch))
			assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), layer.SpriteIndexByLayer(map_state.MapLayer, position))
		}
	}

	for _, test := range []struct {
		name      string
		location  references.Location
		position  references.Position
		base      indexes.SpriteIndex
		override  indexes.SpriteIndex
		wantDay   indexes.SpriteIndex
		wantNight indexes.SpriteIndex
	}{
		{"standalone portcullis", references.SerpentsHold, references.Position{X: 0, Y: 0}, indexes.Portcullis, indexes.NoSprites, indexes.Portcullis, indexes.Portcullis},
		{"standalone archway", references.SerpentsHold, references.Position{X: 0, Y: 0}, indexes.BrickWallArchway, indexes.NoSprites, indexes.BrickWallArchway, indexes.BrickWallArchway},
		{"left drawbridge", references.Britain, references.Position{X: 14, Y: 28}, indexes.LeftWoodFloor2, indexes.NoSprites, indexes.LeftWoodFloor2, indexes.WaterCoast},
		{"right drawbridge override", references.Britain, references.Position{X: 16, Y: 29}, indexes.WaterCoast, indexes.RightWoodFloor2, indexes.RightWoodFloor2, indexes.WaterCoast},
		{"drawbridge terrain overridden", references.Britain, references.Position{X: 15, Y: 28}, indexes.LeftWoodFloor2, indexes.MetalFloor, indexes.MetalFloor, indexes.MetalFloor},
		{"outside drawbridge area", references.Britain, references.Position{X: 13, Y: 28}, indexes.LeftWoodFloor2, indexes.NoSprites, indexes.LeftWoodFloor2, indexes.LeftWoodFloor2},
		{"not Britain", references.Minoc, references.Position{X: 14, Y: 28}, indexes.LeftWoodFloor2, indexes.NoSprites, indexes.LeftWoodFloor2, indexes.LeftWoodFloor2},
		{"opened door override", references.Britain, position, indexes.MagicLockDoor, indexes.BrickFloor, indexes.BrickFloor, indexes.BrickFloor},
		{"unchanged water", references.Britain, position, indexes.WaterCoast, indexes.NoSprites, indexes.WaterCoast, indexes.WaterCoast},
		{"unchanged hazard", references.Britain, position, indexes.Lava, indexes.NoSprites, indexes.Lava, indexes.Lava},
	} {
		t.Run(test.name, func(t *testing.T) {
			state.MapState.PlayerLocation.Location = test.location
			layer.SetTileByLayer(map_state.MapLayer, &test.position, test.base)
			layer.SetTileByLayer(map_state.MapOverrideLayer, &test.position, test.override)
			for _, hour := range []byte{4, 5, 19, 20} {
				state.DateTime.Hour = hour
				want := test.wantDay
				if hour == 4 || hour == 20 {
					want = test.wantNight
				}
				assert.Equal(t, want, state.GetEffectiveTile(&test.position))
				assert.Equal(t, want, state.GetBonusTerrainTile(&test.position))
			}
		})
	}
}

func TestBonusTerrainDungeonEntrances(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithRandomSeed(522).
		WithSystemCallbacks().
		Build()
	for _, test := range []struct {
		name       string
		dungeon    references.Location
		floor      references.FloorNumber
		source     indexes.SpriteIndex
		openSprite indexes.SpriteIndex
		transforms bool
	}{
		{"Deceit overworld", references.Deceit, 0, indexes.DoomEntrance, indexes.DoomEntrance, true},
		{"Deceit underworld", references.Deceit, -1, indexes.BlockEntrance, indexes.DoomEntrance, true},
		{"Doom underworld", references.Doom, -1, indexes.CaveEntrance, indexes.CaveEntrance, true},
		{"Doom not on overworld", references.Doom, 0, indexes.CaveEntrance, indexes.CaveEntrance, false},
		{"noncanonical art at Doom coordinate", references.Doom, -1, indexes.DoomEntrance, indexes.CaveEntrance, false},
		{"non entrance at dungeon coordinate", references.Deceit, 0, indexes.Forest1, indexes.DoomEntrance, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			state.MapState.PlayerLocation.Floor = test.floor
			position := state.GetDungeonEntranceCoordinates()[test.dungeon]
			layer := state.GetLayeredMapByCurrentLocation()
			layer.SetTileByLayer(map_state.MapLayer, &position, test.source)
			layer.UnSetTileByLayer(map_state.MapOverrideLayer, &position)
			for _, open := range []bool{false, true} {
				state.SetDungeonState(test.dungeon, open)
				want := test.source
				if test.transforms {
					want = indexes.BlockEntrance
					if open {
						want = test.openSprite
					}
				}
				assert.Equal(t, want, state.GetEffectiveTile(&position))
				assert.Equal(t, want, state.GetBonusTerrainTile(&position))
			}
			layer.SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.Lava)
			assert.Equal(t, indexes.SpriteIndex(indexes.Lava), state.GetEffectiveTile(&position))
			assert.Equal(t, indexes.SpriteIndex(indexes.Lava), state.GetBonusTerrainTile(&position))
		})
	}
}

func TestBonusTerrainRuinedShrine(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithRandomSeed(522).
		WithSystemCallbacks().
		Build()
	shrine := GetShrineData(0)
	position := references.Position{X: references.Coordinate(shrine.WorldX), Y: references.Coordinate(shrine.WorldY)}
	layer := state.GetLayeredMapByCurrentLocation()
	layer.SetTileByLayer(map_state.MapLayer, &position, indexes.Shrine)
	layer.UnSetTileByLayer(map_state.MapOverrideLayer, &position)
	for _, ruined := range []bool{false, true, false} {
		state.SetShrineRuined(0, ruined)
		want := indexes.SpriteIndex(indexes.Shrine)
		if ruined {
			want = indexes.BrokenShrine
		}
		assert.Equal(t, want, state.GetEffectiveTile(&position))
		assert.Equal(t, want, state.GetBonusTerrainTile(&position))
	}
	state.SetShrineRuined(0, true)
	layer.SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.Forest1)
	assert.Equal(t, indexes.SpriteIndex(indexes.Forest1), state.GetEffectiveTile(&position))
	assert.Equal(t, indexes.SpriteIndex(indexes.Forest1), state.GetBonusTerrainTile(&position))
	layer.SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.Shrine)
	assert.Equal(t, indexes.SpriteIndex(indexes.BrokenShrine), state.GetBonusTerrainTile(&position))
}

func TestBonusTerrainReadOnlyStateAndRNG(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(10, 10).
		WithRandomSeed(522).
		WithSystemCallbacks().
		Build()
	layer := state.GetLayeredMapByCurrentLocation()
	for position, sprite := range map[references.Position]indexes.SpriteIndex{
		{X: 10, Y: 9}:  indexes.Mirror,
		{X: 11, Y: 9}:  indexes.Crown,
		{X: 12, Y: 9}:  indexes.Portcullis,
		{X: 14, Y: 28}: indexes.LeftWoodFloor2,
	} {
		layer.SetTileByLayer(map_state.MapOverrideLayer, &position, sprite)
	}
	position := references.Position{X: 10, Y: 9}
	require.NoError(t, state.ItemStacksMap.AddOrMerge(&position, references.ItemAndQuantity{Item: references.Food, Quantity: 3}))
	state.DateTime.Hour = 20
	state.initializeMoongateStage()
	before, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	for range 3 {
		for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
			for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
				state.GetBonusTerrainTile(&references.Position{X: x, Y: y})
			}
		}
	}
	after, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	assert.Equal(t, before.Random, after.Random, "terrain sampling must not consume or reseed gameplay RNG")
	assert.Equal(t, before, after, "terrain sampling must not mutate gameplay state")
}

func TestBonusTerrainMatchesEffectiveTileNilBehavior(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(10, 10).
		WithRandomSeed(522).
		WithSystemCallbacks().
		Build()
	assert.Equal(t, indexes.SpriteIndex(indexes.NoSprites), state.GetEffectiveTile(nil))
	assert.Equal(t, indexes.SpriteIndex(indexes.NoSprites), state.GetBonusTerrainTile(nil))
	position := references.Position{X: 11, Y: 10}
	layer := state.GetLayeredMapByCurrentLocation()
	layer.UnSetTileByLayer(map_state.MapLayer, &position)
	layer.UnSetTileByLayer(map_state.MapOverrideLayer, &position)
	assert.Equal(t, indexes.SpriteIndex(indexes.NoSprites), state.GetEffectiveTile(&position))
	assert.Equal(t, indexes.SpriteIndex(indexes.NoSprites), state.GetBonusTerrainTile(&position))
	state.MapState.PlayerLocation.Floor = 100
	require.Nil(t, state.GetLayeredMapByCurrentLocation())
	assert.Equal(t, indexes.SpriteIndex(indexes.NoSprites), state.GetEffectiveTile(&position))
	assert.Equal(t, indexes.SpriteIndex(indexes.NoSprites), state.GetBonusTerrainTile(&position))
	var missingState *GameState
	assert.Panics(t, func() { missingState.GetEffectiveTile(&position) })
	assert.Panics(t, func() { missingState.GetBonusTerrainTile(&position) })
}

func TestBonusTerrainSynthetic(t *testing.T) {
	state := &GameState{}
	state.SetRandomSeed(522)
	state.PartyState.Inventory = *party_state.NewInventory()
	state.GameReferences = &references.GameReferences{
		InventoryItemReferences: references.NewInventoryItemsReferences(),
	}
	state.MapState.PlayerLocation = references.PlayerLocation{
		Location: references.BritanniaUnderworld,
		Floor:    0,
		Position: references.Position{X: 100, Y: 100},
	}
	state.MapState.LayeredMaps = *map_state.NewLayeredMaps(
		references.NewTileReferences(), &references.LargeMapReference{}, &references.LargeMapReference{}, 19, 13,
	)
	position := references.Position{X: 100, Y: 99}
	layer := state.GetLayeredMapByCurrentLocation()
	for _, test := range []struct {
		name          string
		base          indexes.SpriteIndex
		override      indexes.SpriteIndex
		corpse        references.SearchObjectClass
		item          bool
		owned         bool
		hour          byte
		wantEffective indexes.SpriteIndex
		wantBonus     indexes.SpriteIndex
	}{
		{name: "base water", base: indexes.WaterCoast, wantEffective: indexes.WaterCoast, wantBonus: indexes.WaterCoast},
		{name: "empty terrain stays empty", wantEffective: indexes.NoSprites, wantBonus: indexes.NoSprites},
		{name: "mirror override without bonus reflection", base: indexes.WaterCoast, override: indexes.Mirror, wantEffective: indexes.MirrorAvatar, wantBonus: indexes.Mirror},
		{name: "broken mirror override", base: indexes.Mirror, override: indexes.MirrorBroken, wantEffective: indexes.MirrorBroken, wantBonus: indexes.MirrorBroken},
		{name: "PC corpse above mirror", base: indexes.WaterCoast, override: indexes.Mirror, corpse: references.SearchObjectDeadPC, wantEffective: indexes.DeadBody, wantBonus: indexes.Mirror},
		{name: "monster corpse above mirror", base: indexes.WaterCoast, override: indexes.Mirror, corpse: references.SearchObjectDeadMonster, wantEffective: indexes.Splat, wantBonus: indexes.Mirror},
		{name: "items above corpse and mirror", base: indexes.WaterCoast, override: indexes.Mirror, corpse: references.SearchObjectDeadPC, item: true, wantEffective: indexes.ItemFood, wantBonus: indexes.Mirror},
		{name: "overworld portcullis at night", base: indexes.Portcullis, hour: 4, wantEffective: indexes.Portcullis, wantBonus: indexes.Portcullis},
		{name: "overworld portcullis stays closed during day", base: indexes.Portcullis, hour: 5, wantEffective: indexes.Portcullis, wantBonus: indexes.Portcullis},
		{name: "overworld archway override during day", base: indexes.StoneBrickWall, override: indexes.BrickWallArchway, hour: 19, wantEffective: indexes.BrickWallArchway, wantBonus: indexes.BrickWallArchway},
		{name: "overworld archway override stays archway at night", base: indexes.StoneBrickWall, override: indexes.BrickWallArchway, hour: 20, wantEffective: indexes.BrickWallArchway, wantBonus: indexes.BrickWallArchway},
		{name: "items above overworld archway", base: indexes.WaterCoast, override: indexes.BrickWallArchway, item: true, hour: 20, wantEffective: indexes.ItemFood, wantBonus: indexes.BrickWallArchway},
		{name: "unowned crown", base: indexes.Crown, wantEffective: indexes.Crown, wantBonus: indexes.BrickFloor},
		{name: "owned crown", base: indexes.Crown, owned: true, wantEffective: indexes.BrickFloor, wantBonus: indexes.BrickFloor},
		{name: "unowned sceptre override", base: indexes.BrickFloor, override: indexes.Sceptre, wantEffective: indexes.Sceptre, wantBonus: indexes.BrickFloor},
		{name: "owned sceptre override", base: indexes.BrickFloor, override: indexes.Sceptre, owned: true, wantEffective: indexes.BrickFloor, wantBonus: indexes.BrickFloor},
		{name: "unowned amulet", base: indexes.Amulet, wantEffective: indexes.Amulet, wantBonus: indexes.BrickFloor},
		{name: "owned amulet", base: indexes.Amulet, owned: true, wantEffective: indexes.BrickFloor, wantBonus: indexes.BrickFloor},
		{name: "real override above artifact", base: indexes.Crown, override: indexes.Lava, owned: true, wantEffective: indexes.Lava, wantBonus: indexes.Lava},
	} {
		t.Run(test.name, func(t *testing.T) {
			layer.SetTileByLayer(map_state.MapLayer, &position, test.base)
			layer.SetTileByLayer(map_state.MapOverrideLayer, &position, test.override)
			layer.SetTileByLayer(map_state.MapUnitLayer, &position, indexes.Chest)
			state.DateTime.Hour = test.hour
			state.SurfaceCorpses = nil
			if test.corpse == references.SearchObjectDeadPC || test.corpse == references.SearchObjectDeadMonster {
				state.SurfaceCorpses = []SurfaceCorpse{{
					Class: test.corpse, Position: position, Location: references.BritanniaUnderworld, Floor: 0,
				}}
			}
			state.ItemStacksMap = *references.NewItemStacksMap()
			if test.item {
				require.NoError(t, state.ItemStacksMap.AddOrMerge(&position, references.ItemAndQuantity{Item: references.Food, Quantity: 3}))
			}
			quantity := uint16(0)
			if test.owned {
				quantity = 1
			}
			for _, item := range []references.QuestItem{references.Crown, references.Sceptre, references.Amulet} {
				state.PartyState.Inventory.QuestItems.Set(item, quantity)
			}

			beforeLocation, beforeTime := state.MapState.PlayerLocation, state.DateTime
			beforeCorpses := append([]SurfaceCorpse(nil), state.SurfaceCorpses...)
			beforeItems, beforeHasItems := state.ItemStacksMap.Snapshot(&position)
			beforeSeed, beforeCalls := state.GetRNG().Seed(), state.GetRNG().Calls()
			for range 3 {
				assert.Equal(t, test.wantBonus, state.GetBonusTerrainTile(&position))
				assert.Equal(t, test.wantEffective, state.GetEffectiveTile(&position))
			}
			assert.Equal(t, beforeSeed, state.GetRNG().Seed())
			assert.Equal(t, beforeCalls, state.GetRNG().Calls(), "terrain queries must not consume gameplay RNG")
			assert.Equal(t, beforeLocation, state.MapState.PlayerLocation)
			assert.Equal(t, beforeTime, state.DateTime)
			assert.Equal(t, beforeCorpses, state.SurfaceCorpses)
			afterItems, afterHasItems := state.ItemStacksMap.Snapshot(&position)
			assert.Equal(t, beforeHasItems, afterHasItems)
			assert.Equal(t, beforeItems, afterItems)
			assert.Equal(t, test.base, layer.SpriteIndexByLayer(map_state.MapLayer, position))
			assert.Equal(t, test.override, layer.SpriteIndexByLayer(map_state.MapOverrideLayer, position))
			for _, item := range []references.QuestItem{references.Crown, references.Sceptre, references.Amulet} {
				assert.Equal(t, quantity, state.PartyState.Inventory.QuestItems.Get(item))
			}
		})
	}
}
