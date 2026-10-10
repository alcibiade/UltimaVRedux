package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestLargeMapProductionMovementUsesOriginalCreatureTerrain(t *testing.T) {
	for _, test := range []struct {
		name    string
		monster references.MonsterType
		terrain indexes.SpriteIndex
		rolls   []int
		bounds  []int
		want    bool
	}{
		{name: "sea horse river", monster: references.MonsterSeaHorse, terrain: indexes.WaterStream1, want: true},
		{name: "shark river end", monster: references.MonsterShark, terrain: indexes.WaterStream14, want: true},
		{name: "sea serpent river bridge", monster: references.MonsterSeaSerpent, terrain: indexes.TrollBridgeHoriz, want: true},
		{name: "ghost wall", monster: references.MonsterGhost, terrain: indexes.StoneBrickWall, want: true},
		{name: "ghost mountain", monster: references.MonsterGhost, terrain: indexes.SmallMountains, rolls: []int{2}, bounds: []int{3}, want: true},
		{name: "ghost rejects sea", monster: references.MonsterGhost, terrain: indexes.Water1},
		{name: "ghost rejects river", monster: references.MonsterGhost, terrain: indexes.WaterStream1},
		{name: "shadowlord wall", monster: references.MonsterShadowlord, terrain: indexes.StoneBrickWall, want: true},
		{name: "shadowlord rejects river", monster: references.MonsterShadowlord, terrain: indexes.WaterStream1},
		{name: "wisp rejects sea", monster: references.MonsterWisp, terrain: indexes.Water1},
		{name: "wisp rejects river", monster: references.MonsterWisp, terrain: indexes.WaterStream1},
		{name: "wisp rejects wall", monster: references.MonsterWisp, terrain: indexes.StoneBrickWall},
		{name: "wisp grass", monster: references.MonsterWisp, terrain: indexes.Grass, want: true},
		{name: "daemon sea", monster: references.MonsterDevil, terrain: indexes.Water1, want: true},
		{name: "daemon river", monster: references.MonsterDevil, terrain: indexes.WaterStream1, want: true},
		{name: "daemon rejects wall", monster: references.MonsterDevil, terrain: indexes.StoneBrickWall},
		{name: "corpser grass", monster: references.MonsterCorpser, terrain: indexes.Grass, want: true},
		{name: "corpser rejects swamp", monster: references.MonsterCorpser, terrain: indexes.Swamp},
		{name: "corpser rejects forest", monster: references.MonsterCorpser, terrain: indexes.Forest1},
		{name: "rotworm swamp", monster: references.MonsterRotWorm, terrain: indexes.Swamp, rolls: []int{0}, bounds: []int{2}, want: true},
		{name: "rotworm rejects grass", monster: references.MonsterRotWorm, terrain: indexes.Grass},
		{name: "whirlpool deep", monster: references.MonsterWhirlpool, terrain: indexes.Water1, want: true},
		{name: "whirlpool rejects sea", monster: references.MonsterWhirlpool, terrain: indexes.Water2},
	} {
		t.Run(test.name, func(t *testing.T) {
			rng := &scriptedGenerationRNG{t: t, values: test.rolls}
			controller, layer := newLargeMapGenerationTestController(t, 0, 12, rng)
			start := references.Position{X: 92, Y: 100}
			destination := references.Position{X: 93, Y: 100}
			enemy := addTerrainTestEnemy(t, controller, test.monster, start)
			// The live override is terrain; the map-unit layer is not.
			layer.SetTileByLayer(map_state.MapLayer, &destination, indexes.Water1)
			layer.SetTileByLayer(map_state.MapOverrideLayer, &destination, test.terrain)
			controller.FreshenExistingNPCsOnMap()

			controller.tryMoveEnemyTo(enemy, destination)

			wantPosition := start
			if test.want {
				wantPosition = destination
			}
			assert.Equal(t, wantPosition, enemy.Pos())
			assert.Equal(t, test.bounds, rng.calls, "illegal terrain must not roll slowdown")
		})
	}
}

func TestLargeMapMovementOccupancyIsIndependentOfEtherealTerrain(t *testing.T) {
	for _, test := range []struct {
		name       string
		actor      bool
		layer      bool
		otherFloor bool
		hidden     bool
		player     bool
		want       bool
	}{
		{name: "empty wall", want: true},
		{name: "projected object", layer: true},
		{name: "live unprojected object", actor: true},
		{name: "live projected object", actor: true, layer: true},
		{name: "other floor is free", actor: true, otherFloor: true, want: true},
		{name: "hidden object is free", actor: true, hidden: true, want: true},
		{name: "party occupies destination", player: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			rng := &scriptedGenerationRNG{t: t}
			controller, layer := newLargeMapGenerationTestController(t, 0, 12, rng)
			start := references.Position{X: 92, Y: 100}
			destination := references.Position{X: 93, Y: 100}
			enemy := addTerrainTestEnemy(t, controller, references.MonsterGhost, start)
			layer.SetTileByLayer(map_state.MapLayer, &destination, indexes.StoneBrickWall)
			if test.actor {
				object := map_units.NewNPCFriendly(*references.NewNPCReferenceForSurfaceChest(
					references.BritanniaUnderworld, destination, 0,
				), 1)
				object.SetPos(destination)
				if test.otherFloor {
					object.SetFloor(-1)
				}
				object.SetVisible(!test.hidden)
				controller.mapUnits = append(controller.mapUnits, object)
			}
			if test.layer {
				layer.SetTileByLayer(map_state.MapUnitLayer, &destination, indexes.Carpet)
			}
			if test.player {
				controller.mapState.PlayerLocation.Position = destination
			}

			controller.tryMoveEnemyTo(enemy, destination)

			wantPosition := start
			if test.want {
				wantPosition = destination
			}
			assert.Equal(t, wantPosition, enemy.Pos())
			assert.Empty(t, rng.calls)
		})
	}
}

func TestLargeMapMovementRoutesCannotEnterOccupiedTiles(t *testing.T) {
	for _, route := range []string{"random", "direct fallback", "A star"} {
		t.Run(route, func(t *testing.T) {
			rng := &scriptedGenerationRNG{t: t}
			controller, layer := newLargeMapGenerationTestController(t, 0, 12, rng)
			start := references.Position{X: 92, Y: 100}
			enemy := addTerrainTestEnemy(t, controller, references.MonsterGhost, start)
			for _, destination := range start.GetFourDirectionsWrapped(references.XLargeMapTiles, references.YLargeMapTiles) {
				layer.SetTileByLayer(map_state.MapLayer, &destination, indexes.Grass)
				object := map_units.NewNPCFriendly(*references.NewNPCReferenceForSurfaceChest(
					references.BritanniaUnderworld, destination, 0,
				), len(controller.mapUnits))
				object.SetPos(destination)
				controller.mapUnits = append(controller.mapUnits, object)
			}
			controller.FreshenExistingNPCsOnMap()
			switch route {
			case "random":
				rng.values = []int{0, 1, 2}
				controller.moveRandomDirection(enemy)
				assert.Equal(t, []int{4, 4, 4}, rng.calls)
			case "direct fallback":
				controller.setBestNextPositionToMoveTowardsWalkablePointDumb(enemy)
			case "A star":
				controller.setBestNextPositionToMoveTowardsWalkablePoint(enemy)
			}
			assert.Equal(t, start, enemy.Pos())
		})
	}
}

func TestLargeMapBatMongbatAndCrawlerUseOrdinaryPursuit(t *testing.T) {
	for _, monster := range []references.MonsterType{references.MonsterBat, references.MonsterMongbat, references.MonsterCrawler} {
		t.Run(monster.Name(), func(t *testing.T) {
			rng := &scriptedGenerationRNG{t: t, values: []int{1, 1}}
			controller, layer := newLargeMapGenerationTestController(t, 0, 12, rng)
			start := references.Position{X: 92, Y: 100}
			for x := start.X; x <= 100; x++ {
				position := references.Position{X: x, Y: start.Y}
				layer.SetTileByLayer(map_state.MapLayer, &position, indexes.Grass)
			}
			enemy := addTerrainTestEnemy(t, controller, monster, start)
			controller.FreshenExistingNPCsOnMap()

			controller.calculateNextNPCPosition(enemy)

			assert.Equal(t, references.Position{X: 93, Y: 100}, enemy.Pos())
			assert.Equal(t, []int{4, 4}, rng.calls)
		})
	}
}

func TestLargeMapAStarRoutesAroundLiveUnprojectedObject(t *testing.T) {
	rng := &scriptedGenerationRNG{t: t}
	controller, layer := newLargeMapGenerationTestController(t, 0, 12, rng)
	start := references.Position{X: 92, Y: 100}
	blocked := references.Position{X: 93, Y: 100}
	for x := start.X; x <= 100; x++ {
		for y := references.Coordinate(99); y <= 101; y++ {
			position := references.Position{X: x, Y: y}
			layer.SetTileByLayer(map_state.MapLayer, &position, indexes.Grass)
		}
	}
	enemy := addTerrainTestEnemy(t, controller, references.MonsterGhost, start)
	object := map_units.NewNPCFriendly(*references.NewNPCReferenceForSurfaceChest(
		references.BritanniaUnderworld, blocked, 0,
	), 1)
	object.SetPos(blocked)
	controller.mapUnits = append(controller.mapUnits, object)

	controller.setBestNextPositionToMoveTowardsWalkablePoint(enemy)

	assert.Contains(t, []references.Position{{X: 92, Y: 99}, {X: 92, Y: 101}}, enemy.Pos(),
		"A* must route around live occupancy, not merely reject the final occupied step")
	assert.Empty(t, rng.calls)
}

func TestLargeMapPirateMovementChecksTerrainAndOccupancy(t *testing.T) {
	for _, test := range []struct {
		name     string
		terrain  indexes.SpriteIndex
		occupied bool
		want     bool
	}{
		{name: "deep water", terrain: indexes.Water1, want: true},
		{name: "sea", terrain: indexes.Water2, want: true},
		{name: "shoals", terrain: indexes.WaterCoast},
		{name: "river", terrain: indexes.WaterStream1},
		{name: "occupied deep water", terrain: indexes.Water1, occupied: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			rng := &scriptedGenerationRNG{t: t}
			controller, layer := newLargeMapGenerationTestController(t, 0, 12, rng)
			start := references.Position{X: 92, Y: 100}
			destination := references.Position{X: 93, Y: 100}
			pirate := map_units.NewEnemyNPC(references.NewPirateShipEnemyReference(
				controller.tileRefs.GetTile(indexes.PirateShipRight)), 0)
			pirate.SetPos(start)
			layer.SetTileByLayer(map_state.MapLayer, &destination, test.terrain)
			if test.occupied {
				layer.SetTileByLayer(map_state.MapUnitLayer, &destination, indexes.Carpet)
			}

			assert.Equal(t, test.want, controller.movePirateShipInDirection(&pirate, references.Right))
			wantPosition := start
			if test.want {
				wantPosition = destination
			}
			assert.Equal(t, wantPosition, pirate.Pos())
			assert.Empty(t, rng.calls)
		})
	}
}

func addTerrainTestEnemy(
	t *testing.T,
	controller *NPCAIControllerLargeMap,
	monster references.MonsterType,
	position references.Position,
) *map_units.NPCEnemy {
	t.Helper()
	ref := findEnemyReferenceForMovementTest(t, controller.enemyReferences, monster)
	enemy := map_units.NewEnemyNPC(*ref, 0)
	enemy.SetPos(position)
	enemy.SetFloor(controller.mapState.PlayerLocation.Floor)
	requireAddedEnemy(t, &controller.mapUnits, enemy)
	added, ok := controller.mapUnits[len(controller.mapUnits)-1].(*map_units.NPCEnemy)
	require.True(t, ok)
	return added
}
