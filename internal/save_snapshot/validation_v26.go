package save_snapshot

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// MaxDeeds bounds the deed list of one save. The game sets no deed limit (ref:CRAFT-0002);
// this only keeps a hostile save from exhausting memory.
const MaxDeeds = 4096

// validateCrafting checks the Redux crafting extension (ref:CRAFT-0001): stock ranges, deed
// identity and placement rules, generator identity, tile overrides, node harvests, the respawn
// marker, built pieces and storage chests, the hut map mutation of every deed and the active deed of a save made on a deed map.
func validateCrafting(snapshot SaveSnapshot, add func(string, string, string)) {
	crafting := snapshot.Crafting
	for index, quantity := range crafting.Materials {
		if quantity > party_state.ItemQuantityLargeMax {
			add(fmt.Sprintf("crafting.materials[%d]", index), "out_of_range", "material quantity exceeds the large stack cap")
		}
	}
	for index, quantity := range crafting.Tools {
		if quantity > party_state.ItemQuantitySmallMax {
			add(fmt.Sprintf("crafting.tools[%d]", index), "out_of_range", "tool quantity exceeds the small stack cap")
		}
	}
	for index, quantity := range crafting.Pieces {
		if quantity > party_state.ItemQuantitySmallMax {
			add(fmt.Sprintf("crafting.pieces[%d]", index), "out_of_range", "piece quantity exceeds the small stack cap")
		}
	}
	if len(crafting.Deeds) > MaxDeeds {
		add("crafting.deeds", "out_of_range", "too many deeds")
	}
	huts := make(map[[2]int]bool)
	for _, mutation := range snapshot.MapMutations {
		if mutation.Position.Location == int(references.BritanniaUnderworld) && mutation.Position.Floor == 0 &&
			mutation.Kind == MapMutationSetTile && mutation.Value == int(indexes.Hut) {
			huts[[2]int{mutation.Position.X, mutation.Position.Y}] = true
		}
	}
	ids := make(map[int]bool, len(crafting.Deeds))
	squares := make(map[[2]int]bool, len(crafting.Deeds))
	for index, deed := range crafting.Deeds {
		path := fmt.Sprintf("crafting.deeds[%d]", index)
		if deed.ID < 0 || ids[deed.ID] {
			add(path+".id", "duplicate", "deed IDs must be unique and non-negative")
		}
		ids[deed.ID] = true
		if !references.IsValidDeedName(deed.Name) {
			add(path+".name", "invalid", "deed name must be 1 to 24 printable characters with no surrounding spaces")
		}
		validateDeedSquare(path, deed, squares, huts, add)
		if _, ok := references.DeedTerrainFromKey(deed.Terrain); !ok {
			add(path+".terrain", "unsupported", "deed terrain is not a known kind")
		}
		if deed.GeneratorVersion < 1 || deed.GeneratorVersion > references.DeedGeneratorVersion {
			add(path+".generatorVersion", "unsupported", "deed generator version is not supported by this build")
		}
		validateDeedOverrides(path, deed.TileOverrides, add)
		validateDeedNodeHarvests(path, deed, add)
		if deed.TrammelPhaseChanges < 0 {
			add(path+".trammelPhaseChanges", "out_of_range", "the respawn marker cannot be negative")
		}
		validateDeedPlacements(path, deed, add)
		validateDeedChests(path, deed, add)
	}
	validateActiveDeed(snapshot, ids, add)
}

func validateDeedSquare(path string, deed DeedSnapshot, squares, huts map[[2]int]bool, add func(string, string, string)) {
	size := int(references.XLargeMapTiles)
	if deed.X < 0 || deed.X >= size || deed.Y < 0 || deed.Y >= size {
		add(path, "out_of_range", "deed square must lie on the 256x256 overworld")

		return
	}
	square := [2]int{deed.X, deed.Y}
	if squares[square] {
		add(path, "duplicate", "two deeds share a square")
	}
	for _, offset := range [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
		neighbour := [2]int{(deed.X + offset[0] + size) % size, (deed.Y + offset[1] + size) % size}
		if squares[neighbour] {
			add(path, "invalid", "deed squares must not be orthogonally adjacent")

			break
		}
	}
	squares[square] = true
	if !huts[square] {
		add(path, "invalid", "every deed square must carry its hut map mutation")
	}
}

func validateDeedOverrides(path string, overrides []DeedTileOverrideSnapshot, add func(string, string, string)) {
	if len(overrides) > MaxDeedTileOverrides {
		add(path+".tileOverrides", "out_of_range", "a deed has at most one override per square")
	}
	seen := make(map[[2]int]bool, len(overrides))
	for index, override := range overrides {
		overridePath := fmt.Sprintf("%s.tileOverrides[%d]", path, index)
		if override.X < 0 || override.X >= DeedMapTiles || override.Y < 0 || override.Y >= DeedMapTiles {
			add(overridePath, "out_of_range", "override must lie on the 32x32 deed map")
		}
		if override.Sprite < 0 || override.Sprite > 0xff {
			add(overridePath+".sprite", "out_of_range", "override sprite must be a map tile index")
		}
		key := [2]int{override.X, override.Y}
		if seen[key] {
			add(overridePath, "duplicate", "override squares must be unique")
		}
		seen[key] = true
	}
}

func validateActiveDeed(snapshot SaveSnapshot, ids map[int]bool, add func(string, string, string)) {
	onDeedMap := snapshot.World.Position.Location == int(references.DeedLocation)
	active := snapshot.Crafting.ActiveDeedID
	switch {
	case onDeedMap && active == nil:
		add("crafting.activeDeedId", "invalid", "a save on a deed map must name its deed")
	case onDeedMap && !ids[*active]:
		add("crafting.activeDeedId", "invalid", "the active deed does not exist")
	case !onDeedMap && active != nil:
		add("crafting.activeDeedId", "invalid", "only a save on a deed map has an active deed")
	}
	if onDeedMap && snapshot.World.Position.Floor != 0 {
		add("world.position.floor", "out_of_range", "a deed map has one floor")
	}
}

// validateDeedNodeHarvests checks that every harvest count names a distinct resource node of the
// deed's generated map and lies between 1 and that node kind's capacity (ref:CRAFT-0005).
func validateDeedNodeHarvests(path string, deed DeedSnapshot, add func(string, string, string)) {
	if len(deed.NodeHarvests) == 0 {
		return
	}
	if len(deed.NodeHarvests) > MaxDeedTileOverrides {
		add(path+".nodeHarvests", "out_of_range", "a deed has at most one harvest count per square")

		return
	}
	terrain, ok := references.DeedTerrainFromKey(deed.Terrain)
	if !ok {
		return
	}
	layout, err := references.GenerateDeedMap(terrain, deed.Seed, deed.GeneratorVersion)
	if err != nil {
		return // The generator version is reported on its own.
	}
	seen := make(map[[2]int]bool, len(deed.NodeHarvests))
	for index, harvest := range deed.NodeHarvests {
		harvestPath := fmt.Sprintf("%s.nodeHarvests[%d]", path, index)
		if harvest.X < 0 || harvest.X >= DeedMapTiles || harvest.Y < 0 || harvest.Y >= DeedMapTiles {
			add(harvestPath, "out_of_range", "a harvest count must lie on the 32x32 deed map")

			continue
		}
		key := [2]int{harvest.X, harvest.Y}
		if seen[key] {
			add(harvestPath, "duplicate", "harvest count squares must be unique")
		}
		seen[key] = true
		node, isNode := layout.NodeAt(references.Position{X: references.Coordinate(harvest.X), Y: references.Coordinate(harvest.Y)})
		if !isNode {
			add(harvestPath, "invalid", "a harvest count must name a resource node of the deed")

			continue
		}
		if harvest.Harvests < 1 || harvest.Harvests > references.DeedNodeRuleFor(node.Kind).Capacity {
			add(harvestPath+".harvests", "out_of_range", "harvests must lie between 1 and the node's capacity")
		}
	}
}

// deedValidationLayout regenerates a deed's map for checks that need its generated squares. It
// reports false when the terrain or generator version is invalid; those are reported on their own.
func deedValidationLayout(deed DeedSnapshot) (references.DeedLayout, bool) {
	terrain, ok := references.DeedTerrainFromKey(deed.Terrain)
	if !ok {
		return references.DeedLayout{}, false
	}
	layout, err := references.GenerateDeedMap(terrain, deed.Seed, deed.GeneratorVersion)

	return layout, err == nil
}

// validateDeedPlacements checks every built piece (ref:CRAFT-0010): a known piece on a distinct
// interior square that is no resource node, covering the generated ground, with the piece's
// sprite as the square's tile override, and only a floor beneath a piece built on a floor.
func validateDeedPlacements(path string, deed DeedSnapshot, add func(string, string, string)) {
	if len(deed.Placements) == 0 {
		return
	}
	if len(deed.Placements) > MaxDeedTileOverrides {
		add(path+".placements", "out_of_range", "a deed has at most one placement per square")

		return
	}
	layout, ok := deedValidationLayout(deed)
	if !ok {
		return
	}
	overrides := make(map[[2]int]int, len(deed.TileOverrides))
	for _, override := range deed.TileOverrides {
		overrides[[2]int{override.X, override.Y}] = override.Sprite
	}
	seen := make(map[[2]int]bool, len(deed.Placements))
	for index, placement := range deed.Placements {
		placementPath := fmt.Sprintf("%s.placements[%d]", path, index)
		position := references.Position{X: references.Coordinate(placement.X), Y: references.Coordinate(placement.Y)}
		if placement.X < 0 || placement.X >= DeedMapTiles || placement.Y < 0 || placement.Y >= DeedMapTiles ||
			references.DeedSquareDepth(position) < references.DeedInteriorDepth {
			add(placementPath, "out_of_range", "a placement must lie inside the deed map's open interior")

			continue
		}
		key := [2]int{placement.X, placement.Y}
		if seen[key] {
			add(placementPath, "duplicate", "placement squares must be unique")
		}
		seen[key] = true
		if _, isNode := layout.NodeAt(position); isNode {
			add(placementPath, "invalid", "a piece cannot stand on a resource node")
		}
		piece, known := references.BuildPieceByID(references.BuildPieceID(placement.PieceID))
		if !known {
			add(placementPath+".pieceId", "unsupported", "unknown build piece")

			continue
		}
		if placement.Under != int(layout.Tile(position)) {
			add(placementPath+".under", "invalid", "a placement must record the generated ground it covers")
		}
		if placement.FloorPieceID != nil {
			floor, floorKnown := references.BuildPieceByID(references.BuildPieceID(*placement.FloorPieceID))
			if !floorKnown || !floor.IsFloor() || piece.IsFloor() {
				add(placementPath+".floorPieceId", "invalid", "only a floor can lie beneath a piece, and never beneath a floor")
			}
		}
		if sprite, overridden := overrides[key]; !overridden || sprite != int(piece.Sprite) {
			add(placementPath, "invalid", "a placed piece's square must show the piece's tile")
		}
	}
}

// validateDeedChests checks storage chest contents (ref:CRAFT-0011): exactly one record per built
// storage chest, and every slot within the cap of the party stack it comes from.
func validateDeedChests(path string, deed DeedSnapshot, add func(string, string, string)) {
	chestSquares := make(map[[2]int]bool)
	for _, placement := range deed.Placements {
		if piece, known := references.BuildPieceByID(references.BuildPieceID(placement.PieceID)); known && piece.StorageChest {
			chestSquares[[2]int{placement.X, placement.Y}] = true
		}
	}
	seen := make(map[[2]int]bool, len(deed.Chests))
	for index, chest := range deed.Chests {
		chestPath := fmt.Sprintf("%s.chests[%d]", path, index)
		key := [2]int{chest.X, chest.Y}
		if !chestSquares[key] {
			add(chestPath, "invalid", "a chest record must belong to a built storage chest")
		}
		if seen[key] {
			add(chestPath, "duplicate", "chest squares must be unique")
		}
		seen[key] = true
		validateDeedChestContents(chestPath, chest, add)
	}
	for square := range chestSquares {
		if !seen[square] {
			add(path+".chests", "invalid", fmt.Sprintf("the storage chest at %d,%d has no contents record", square[0], square[1]))
		}
	}
}

func validateDeedChestContents(path string, chest DeedChestSnapshot, add func(string, string, string)) {
	check := func(field string, index int, quantity, limit uint16) {
		if quantity > limit {
			add(fmt.Sprintf("%s.%s[%d]", path, field, index), "out_of_range", "chest quantity exceeds the stack cap")
		}
	}
	for index, quantity := range chest.Materials {
		check("materials", index, quantity, party_state.ItemQuantityLargeMax)
	}
	for index, quantity := range chest.Tools {
		check("tools", index, quantity, party_state.ItemQuantitySmallMax)
	}
	for index, quantity := range chest.Pieces {
		check("pieces", index, quantity, party_state.ItemQuantitySmallMax)
	}
	for index, quantity := range chest.Provisions {
		limit := uint16(party_state.ItemQuantitySmallMax)
		if references.Provision(index) == references.Food {
			limit = party_state.ItemQuantityLargeMax
		}
		check("provisions", index, quantity, limit)
	}
	if chest.Gold > party_state.ItemQuantityLargeMax {
		add(path+".gold", "out_of_range", "chest gold exceeds the stack cap")
	}
}
