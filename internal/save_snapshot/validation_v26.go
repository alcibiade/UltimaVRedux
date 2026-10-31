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
// identity and placement rules, generator identity, tile overrides, the hut map mutation of every
// deed and the active deed of a save made on a deed map.
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
