package save_snapshot

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func hutMutation(x, y int) MapMutationSnapshot {
	return MapMutationSnapshot{
		Position: PositionSnapshot{Location: int(references.BritanniaUnderworld), X: x, Y: y, Floor: 0},
		Kind:     MapMutationSetTile, Value: int(indexes.Hut),
	}
}

// validCraftingFixture holds two deeds, one across the world's wrapping seam, with their huts,
// and a save made on the second deed's map.
func validCraftingFixture() SaveSnapshot {
	snapshot := validSnapshot()
	active := 3
	layout, err := references.GenerateDeedMap(references.DeedTerrainGrassland, 7, 1)
	if err != nil {
		panic(err)
	}
	nodes := layout.Nodes
	ground := func(x, y int) int {
		return int(layout.Tile(references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}))
	}
	floor := int(references.PieceWoodFloor)
	snapshot.Crafting = CraftingSnapshot{
		Materials: [CraftingMaterialCount]uint16{10, 5, 0, 2, 0, 0, 1},
		Tools:     [CraftingToolCount]uint16{1, 0},
		Pieces:    [CraftingPieceCount]uint16{references.PieceStoneWall: 4, references.PieceForge: 1},
		Deeds: []DeedSnapshot{
			{ID: 0, Name: "Homestead", X: 255, Y: 0, Terrain: "grassland", GeneratorVersion: 1, Seed: 7,
				TileOverrides: []DeedTileOverrideSnapshot{
					{X: 4, Y: 5, Sprite: int(indexes.Grass)},
					{X: 13, Y: 13, Sprite: int(indexes.StoneBrickWall)},
					{X: 14, Y: 13, Sprite: int(indexes.Box)},
					{X: 15, Y: 13, Sprite: int(indexes.ChairBackForward)},
				},
				Placements: []DeedPlacementSnapshot{
					{X: 13, Y: 13, PieceID: int(references.PieceStoneWall), Under: ground(13, 13)},
					{X: 14, Y: 13, PieceID: int(references.PieceStorageChest), Under: ground(14, 13)},
					{X: 15, Y: 13, PieceID: int(references.PieceChairForward), Under: ground(15, 13), FloorPieceID: &floor},
				},
				Chests: []DeedChestSnapshot{{X: 14, Y: 13, Materials: [CraftingMaterialCount]uint16{9999},
					Provisions: [CraftingChestProvisionCount]uint16{4: 9999}, Gold: 9999,
					Equipment: [CraftingChestEquipmentCount]uint16{int(references.LongSword): 99},
					Potions:   [CraftingChestPotionCount]uint16{int(references.Yellow): 99},
					Scrolls:   [CraftingChestScrollCount]uint16{int(references.ScrollInManiCorp): 9999}}},
				NodeHarvests: []DeedNodeHarvestSnapshot{
					{X: int(nodes[0].Position.X), Y: int(nodes[0].Position.Y), Harvests: 1},
					{X: int(nodes[1].Position.X), Y: int(nodes[1].Position.Y), Harvests: references.DeedNodeRuleFor(nodes[1].Kind).Capacity},
				},
				TrammelPhaseChanges: 28900},
			{ID: 3, Name: "Lord's Rest", X: 100, Y: 100, Terrain: "desert", GeneratorVersion: 1, Seed: 0},
		},
		ActiveDeedID: &active,
	}
	snapshot.World.Position = PositionSnapshot{Location: int(references.DeedLocation), X: 15, Y: 30, Floor: 0}
	snapshot.MapUnits = nil
	snapshot.ItemStacks = nil
	snapshot.MapMutations = []MapMutationSnapshot{hutMutation(255, 0), hutMutation(100, 100)}

	return snapshot
}

// nodesOfFixture returns the resource node squares of the fixture's first deed.
func nodesOfFixture() []references.Position {
	layout, err := references.GenerateDeedMap(references.DeedTerrainGrassland, 7, 1)
	if err != nil {
		panic(err)
	}
	positions := make([]references.Position, 0, len(layout.Nodes))
	for _, node := range layout.Nodes {
		if references.DeedSquareDepth(node.Position) >= references.DeedInteriorDepth {
			positions = append(positions, node.Position)
		}
	}

	return positions
}

func craftingIssues(snapshot SaveSnapshot) []Issue {
	var issues []Issue
	validateCrafting(snapshot, func(path, code, message string) {
		issues = append(issues, Issue{Path: path, Code: code, Message: message})
	})

	return issues
}

func TestValidateCraftingAcceptsDeedsAndAnActiveDeedMap(t *testing.T) {
	require.Empty(t, craftingIssues(validCraftingFixture()))

	overworld := validCraftingFixture()
	overworld.World.Position = validPosition()
	overworld.Crafting.ActiveDeedID = nil
	assert.Empty(t, craftingIssues(overworld), "a save off the deed map has no active deed")
	assert.Empty(t, craftingIssues(validSnapshot()), "a save without crafting state is valid")
}

func TestValidateCraftingRejectsEachInvalidShape(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		mutate func(*SaveSnapshot)
		path   string
		code   string
	}{
		{"material over cap", func(s *SaveSnapshot) { s.Crafting.Materials[6] = 10000 }, "crafting.materials[6]", "out_of_range"},
		{"tool over cap", func(s *SaveSnapshot) { s.Crafting.Tools[1] = 100 }, "crafting.tools[1]", "out_of_range"},
		{"duplicate deed ID", func(s *SaveSnapshot) { s.Crafting.Deeds[1].ID = 0 }, "crafting.deeds[1].id", "duplicate"},
		{"negative deed ID", func(s *SaveSnapshot) { s.Crafting.Deeds[0].ID = -1 }, "crafting.deeds[0].id", "duplicate"},
		{"empty name", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Name = "" }, "crafting.deeds[0].name", "invalid"},
		{"padded name", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Name = " Home" }, "crafting.deeds[0].name", "invalid"},
		{"non-ASCII name", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Name = "Café" }, "crafting.deeds[0].name", "invalid"},
		{"long name", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Name = "abcdefghijklmnopqrstuvwxy" }, "crafting.deeds[0].name", "invalid"},
		{"square off the map", func(s *SaveSnapshot) { s.Crafting.Deeds[1].X = 256 }, "crafting.deeds[1]", "out_of_range"},
		{"shared square", func(s *SaveSnapshot) { s.Crafting.Deeds[1].X, s.Crafting.Deeds[1].Y = 255, 0 }, "crafting.deeds[1]", "duplicate"},
		{"adjacent across the seam", func(s *SaveSnapshot) {
			s.Crafting.Deeds[1].X, s.Crafting.Deeds[1].Y = 0, 0
			s.MapMutations = append(s.MapMutations, hutMutation(0, 0))
		}, "crafting.deeds[1]", "invalid"},
		{"missing hut", func(s *SaveSnapshot) { s.MapMutations = s.MapMutations[:1] }, "crafting.deeds[1]", "invalid"},
		{"unknown terrain", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Terrain = "swamp" }, "crafting.deeds[0].terrain", "unsupported"},
		{"future generator", func(s *SaveSnapshot) {
			s.Crafting.Deeds[0].GeneratorVersion = references.DeedGeneratorVersion + 1
		}, "crafting.deeds[0].generatorVersion", "unsupported"},
		{"zero generator", func(s *SaveSnapshot) { s.Crafting.Deeds[0].GeneratorVersion = 0 }, "crafting.deeds[0].generatorVersion", "unsupported"},
		{"override off the deed map", func(s *SaveSnapshot) { s.Crafting.Deeds[0].TileOverrides[0].X = 32 }, "crafting.deeds[0].tileOverrides[0]", "out_of_range"},
		{"override sprite", func(s *SaveSnapshot) { s.Crafting.Deeds[0].TileOverrides[0].Sprite = 256 }, "crafting.deeds[0].tileOverrides[0].sprite", "out_of_range"},
		{"duplicate override", func(s *SaveSnapshot) {
			s.Crafting.Deeds[0].TileOverrides = append(s.Crafting.Deeds[0].TileOverrides, s.Crafting.Deeds[0].TileOverrides[0])
		}, "crafting.deeds[0].tileOverrides[4]", "duplicate"},
		{"harvest off the deed map", func(s *SaveSnapshot) { s.Crafting.Deeds[0].NodeHarvests[0].Y = 32 }, "crafting.deeds[0].nodeHarvests[0]", "out_of_range"},
		{"harvest on no node", func(s *SaveSnapshot) {
			s.Crafting.Deeds[0].NodeHarvests[0].X, s.Crafting.Deeds[0].NodeHarvests[0].Y = 15, 15
		}, "crafting.deeds[0].nodeHarvests[0]", "invalid"},
		{"duplicate harvest", func(s *SaveSnapshot) {
			s.Crafting.Deeds[0].NodeHarvests[1].X = s.Crafting.Deeds[0].NodeHarvests[0].X
			s.Crafting.Deeds[0].NodeHarvests[1].Y = s.Crafting.Deeds[0].NodeHarvests[0].Y
		}, "crafting.deeds[0].nodeHarvests[1]", "duplicate"},
		{"zero harvests", func(s *SaveSnapshot) { s.Crafting.Deeds[0].NodeHarvests[0].Harvests = 0 }, "crafting.deeds[0].nodeHarvests[0].harvests", "out_of_range"},
		{"harvests over capacity", func(s *SaveSnapshot) { s.Crafting.Deeds[0].NodeHarvests[1].Harvests++ }, "crafting.deeds[0].nodeHarvests[1].harvests", "out_of_range"},
		{"negative respawn marker", func(s *SaveSnapshot) { s.Crafting.Deeds[1].TrammelPhaseChanges = -1 }, "crafting.deeds[1].trammelPhaseChanges", "out_of_range"},
		{"deed map without active deed", func(s *SaveSnapshot) { s.Crafting.ActiveDeedID = nil }, "crafting.activeDeedId", "invalid"},
		{"unknown active deed", func(s *SaveSnapshot) { unknown := 9; s.Crafting.ActiveDeedID = &unknown }, "crafting.activeDeedId", "invalid"},
		{"active deed off the deed map", func(s *SaveSnapshot) { s.World.Position = validPosition() }, "crafting.activeDeedId", "invalid"},
		{"deed map floor", func(s *SaveSnapshot) { s.World.Position.Floor = 1 }, "world.position.floor", "out_of_range"},
		{"piece over cap", func(s *SaveSnapshot) { s.Crafting.Pieces[0] = 100 }, "crafting.pieces[0]", "out_of_range"},
		{"placement in the ring", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Placements[0].X = 3 }, "crafting.deeds[0].placements[0]", "out_of_range"},
		{"duplicate placement", func(s *SaveSnapshot) {
			s.Crafting.Deeds[0].Placements[1].X = 13
		}, "crafting.deeds[0].placements[1]", "duplicate"},
		{"placement on a node", func(s *SaveSnapshot) {
			s.Crafting.Deeds[0].Placements[0].X, s.Crafting.Deeds[0].Placements[0].Y = int(nodesOfFixture()[0].X), int(nodesOfFixture()[0].Y)
		}, "crafting.deeds[0].placements[0]", "invalid"},
		{"unknown piece", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Placements[0].PieceID = 26 }, "crafting.deeds[0].placements[0].pieceId", "unsupported"},
		{"wrong ground", func(s *SaveSnapshot) {
			s.Crafting.Deeds[0].Placements[0].Under = int(indexes.Water1)
		}, "crafting.deeds[0].placements[0].under", "invalid"},
		{"non-floor beneath", func(s *SaveSnapshot) {
			wall := int(references.PieceStoneWall)
			s.Crafting.Deeds[0].Placements[2].FloorPieceID = &wall
		}, "crafting.deeds[0].placements[2].floorPieceId", "invalid"},
		{"floor beneath a floor", func(s *SaveSnapshot) {
			s.Crafting.Deeds[0].Placements[2].PieceID = int(references.PieceStoneFloor)
			s.Crafting.Deeds[0].TileOverrides[3].Sprite = int(indexes.BrickFloor)
		}, "crafting.deeds[0].placements[2].floorPieceId", "invalid"},
		{"placement without its tile", func(s *SaveSnapshot) {
			s.Crafting.Deeds[0].TileOverrides[1].Sprite = int(indexes.Grass)
		}, "crafting.deeds[0].placements[0]", "invalid"},
		{"chest without a placed chest", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Chests[0].X = 13 }, "crafting.deeds[0].chests[0]", "invalid"},
		{"placed chest without contents", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Chests = nil }, "crafting.deeds[0].chests", "invalid"},
		{"duplicate chest", func(s *SaveSnapshot) {
			s.Crafting.Deeds[0].Chests = append(s.Crafting.Deeds[0].Chests, s.Crafting.Deeds[0].Chests[0])
		}, "crafting.deeds[0].chests[1]", "duplicate"},
		{"chest material over cap", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Chests[0].Materials[0] = 10000 }, "crafting.deeds[0].chests[0].materials[0]", "out_of_range"},
		{"chest tool over cap", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Chests[0].Tools[0] = 100 }, "crafting.deeds[0].chests[0].tools[0]", "out_of_range"},
		{"chest piece over cap", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Chests[0].Pieces[3] = 100 }, "crafting.deeds[0].chests[0].pieces[3]", "out_of_range"},
		{"chest torches over cap", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Chests[0].Provisions[0] = 100 }, "crafting.deeds[0].chests[0].provisions[0]", "out_of_range"},
		{"chest food over cap", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Chests[0].Provisions[4] = 10000 }, "crafting.deeds[0].chests[0].provisions[4]", "out_of_range"},
		{"chest gold over cap", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Chests[0].Gold = 10000 }, "crafting.deeds[0].chests[0].gold", "out_of_range"},
		{"chest equipment over cap", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Chests[0].Equipment[30] = 100 }, "crafting.deeds[0].chests[0].equipment[30]", "out_of_range"},
		{"chest potion over cap", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Chests[0].Potions[1] = 100 }, "crafting.deeds[0].chests[0].potions[1]", "out_of_range"},
		{"chest scroll over cap", func(s *SaveSnapshot) { s.Crafting.Deeds[0].Chests[0].Scrolls[6] = 10000 }, "crafting.deeds[0].chests[0].scrolls[6]", "out_of_range"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			snapshot := validCraftingFixture()
			testCase.mutate(&snapshot)

			issues := craftingIssues(snapshot)

			paths := make([]string, 0, len(issues))
			for _, issue := range issues {
				if issue.Path == testCase.path && issue.Code == testCase.code {
					return
				}
				paths = append(paths, issue.Path+":"+issue.Code)
			}
			t.Fatalf("want %s:%s, got %v", testCase.path, testCase.code, paths)
		})
	}
}

func TestValidateRejectsCraftingThroughTheCodec(t *testing.T) {
	snapshot := validCraftingFixture()
	snapshot.Crafting.Deeds[0].Terrain = "swamp"

	issues := Validate(snapshot)

	assert.Contains(t, issues, Issue{Path: "crafting.deeds[0].terrain", Code: "unsupported", Message: "deed terrain is not a known kind"})
}

func TestV26CraftingCodecAndSchemaAgree(t *testing.T) {
	schema, _ := compileCurrentNativeSaveSchema(t)
	requireV23RoundTrip(t, schema, validCraftingFixture())

	data, err := EncodeCanonical(validEnvelope(validCraftingFixture()))
	require.NoError(t, err)
	var document map[string]any
	require.NoError(t, json.Unmarshal(data, &document))
	deed := document["snapshot"].(map[string]any)["crafting"].(map[string]any)["deeds"].([]any)[0].(map[string]any)
	deed["terrain"] = "swamp"
	assert.Error(t, schema.Validate(document), "the schema rejects an unknown terrain")
	deed["terrain"] = "grassland"
	require.NoError(t, schema.Validate(document))
	deed["seed"] = 4294967296
	assert.Error(t, schema.Validate(document), "the schema bounds the seed to 32 bits")
	deed["seed"] = 7
	require.NoError(t, schema.Validate(document))
	deed["nodeHarvests"].([]any)[0].(map[string]any)["harvests"] = 0
	assert.Error(t, schema.Validate(document), "the schema rejects an empty harvest count")
	deed["nodeHarvests"].([]any)[0].(map[string]any)["harvests"] = 1
	deed["trammelPhaseChanges"] = -1
	assert.Error(t, schema.Validate(document), "the schema rejects a negative respawn marker")
	deed["trammelPhaseChanges"] = 0
	require.NoError(t, schema.Validate(document))
	placement := deed["placements"].([]any)[0].(map[string]any)
	placement["pieceId"] = 26
	assert.Error(t, schema.Validate(document), "the schema bounds piece IDs")
	placement["pieceId"] = 0
	placement["x"] = 3
	assert.Error(t, schema.Validate(document), "the schema keeps placements off the ring")
	placement["x"] = 13
	chest := deed["chests"].([]any)[0].(map[string]any)
	chest["gold"] = 10000
	assert.Error(t, schema.Validate(document), "the schema caps chest gold")
	chest["gold"] = 0
	delete(chest, "provisions")
	assert.Error(t, schema.Validate(document), "chest provisions are required")
	chest["provisions"] = []any{0, 0, 0, 0, 0}
	require.NoError(t, schema.Validate(document))
	chest["equipment"].([]any)[30] = 100
	assert.Error(t, schema.Validate(document), "the schema caps chest equipment")
	chest["equipment"].([]any)[30] = 0
	chest["potions"] = []any{0, 0, 0}
	assert.Error(t, schema.Validate(document), "the schema needs all eight potion slots")
	chest["potions"] = []any{0, 0, 0, 0, 0, 0, 0, 0}
	delete(chest, "scrolls")
	assert.Error(t, schema.Validate(document), "chest scrolls are required")

	withDoor := validCraftingFixture()
	withDoor.OpenDoor = &OpenDoorSnapshot{
		Position:       PositionSnapshot{Location: int(references.DeedLocation), X: 13, Y: 12, Floor: 0},
		RemainingTurns: 1, ClosedTile: int(indexes.RegularDoor),
	}
	requireV23RoundTrip(t, schema, withDoor)
	data, err = EncodeCanonical(validEnvelope(withDoor))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &document))
	door := document["snapshot"].(map[string]any)["openDoor"].(map[string]any)
	door["position"].(map[string]any)["floor"] = 1
	assert.Error(t, schema.Validate(document), "a deed map's open door is on floor 0")

	invalid := validCraftingFixture()
	invalid.Crafting.Deeds[0].Terrain = "swamp"
	_, err = EncodeCanonical(validEnvelope(invalid))
	assert.ErrorIs(t, err, ErrInvalidSnapshot)
}

func TestValidateAcceptsATimedDoorOnTheDeedMapOnly(t *testing.T) {
	snapshot := validCraftingFixture()
	snapshot.OpenDoor = &OpenDoorSnapshot{
		Position:       PositionSnapshot{Location: int(references.DeedLocation), X: 13, Y: 12, Floor: 0},
		RemainingTurns: 2, ClosedTile: int(indexes.RegularDoor),
	}
	var issues []Issue
	validateOpenDoor(snapshot, func(path, code, message string) { issues = append(issues, Issue{path, code, message}) })
	assert.Empty(t, issues, "a built door on the deed map may be open when saving")

	snapshot.OpenDoor.Position.Floor = 1
	issues = nil
	validateOpenDoor(snapshot, func(path, code, message string) { issues = append(issues, Issue{path, code, message}) })
	assert.NotEmpty(t, issues, "a deed map has one floor")
}
