package save_snapshot

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// validateCrafting checks the Redux crafting extension (ref:CRAFT-0001): ranges, uniqueness and
// the catalog, then the structure the game maintains: claims never overlap, every placement and
// storage chest lies inside a claim, and every storage chest stands on a storage-chest placement.
//
// Placed tiles themselves are not cross-checked against mapMutations: a floor built over the same
// original floor tile changes nothing on the map, so it legitimately has no mutation.
func validateCrafting(crafting CraftingSnapshot, add func(string, string, string)) {
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
	validateLandClaims(crafting.LandClaims, add)
	insideClaim := func(x, y int) bool {
		for _, claim := range crafting.LandClaims {
			if craftingWrappedOffset(claim.X, x) < references.LandClaimSize &&
				craftingWrappedOffset(claim.Y, y) < references.LandClaimSize {
				return true
			}
		}

		return false
	}
	placements := make(map[[2]int]PlacementSnapshot, len(crafting.Placements))
	for index, placement := range crafting.Placements {
		path := fmt.Sprintf("crafting.placements[%d]", index)
		validateCraftingCoordinate(path, placement.X, placement.Y, add)
		if _, ok := references.BuildPieceByID(placement.PieceID); !ok {
			add(path+".pieceId", "unsupported", "placement piece is not in the build catalog")
		}
		if placement.Under < 0 || placement.Under >= craftingMapSize {
			add(path+".under", "out_of_range", "covered tile must be an overworld sprite index")
		}
		key := [2]int{placement.X, placement.Y}
		if _, exists := placements[key]; exists {
			add(path, "duplicate", "placement positions must be unique")
		}
		placements[key] = placement
		if !insideClaim(placement.X, placement.Y) {
			add(path, "invalid", "placements must lie inside a land claim")
		}
	}
	chests := make(map[[2]int]struct{}, len(crafting.HomeChests))
	for index, chest := range crafting.HomeChests {
		path := fmt.Sprintf("crafting.homeChests[%d]", index)
		validateCraftingCoordinate(path, chest.X, chest.Y, add)
		key := [2]int{chest.X, chest.Y}
		if _, exists := chests[key]; exists {
			add(path, "duplicate", "home chest positions must be unique")
		}
		chests[key] = struct{}{}
		placement, placed := placements[key]
		piece, known := references.BuildPieceByID(placement.PieceID)
		if !placed || !known || !piece.HomeChest {
			add(path, "invalid", "home chests must stand on a storage chest placement")
		}
		for slot, quantity := range chest.Materials {
			if quantity > party_state.ItemQuantityLargeMax {
				add(fmt.Sprintf("%s.materials[%d]", path, slot), "out_of_range", "material quantity exceeds the large stack cap")
			}
		}
		for slot, quantity := range chest.Tools {
			if quantity > party_state.ItemQuantitySmallMax {
				add(fmt.Sprintf("%s.tools[%d]", path, slot), "out_of_range", "tool quantity exceeds the small stack cap")
			}
		}
		for slot, quantity := range chest.Provisions {
			if quantity > party_state.ItemQuantityLargeMax {
				add(fmt.Sprintf("%s.provisions[%d]", path, slot), "out_of_range", "provision quantity exceeds the large stack cap")
			}
		}
	}
	for index, placement := range crafting.Placements {
		piece, known := references.BuildPieceByID(placement.PieceID)
		if _, hasChest := chests[[2]int{placement.X, placement.Y}]; known && piece.HomeChest && !hasChest {
			add(fmt.Sprintf("crafting.placements[%d]", index), "invalid", "a storage chest placement needs its home chest contents")
		}
	}
}

func validateLandClaims(claims []LandClaimSnapshot, add func(string, string, string)) {
	if len(claims) > references.MaxLandClaims {
		add("crafting.landClaims", "out_of_range", "too many land claims")
	}
	claimIDs := make(map[int]struct{}, len(claims))
	for index, claim := range claims {
		path := fmt.Sprintf("crafting.landClaims[%d]", index)
		if _, exists := claimIDs[claim.ID]; exists || claim.ID < 0 {
			add(path+".id", "duplicate", "land claim IDs must be unique and non-negative")
		}
		claimIDs[claim.ID] = struct{}{}
		if !references.IsValidLandClaimName(claim.Name) {
			add(path+".name", "invalid", "land claim name must be 1 to 24 printable characters with no surrounding spaces")
		}
		validateCraftingCoordinate(path, claim.X, claim.Y, add)
		if claim.Width != references.LandClaimSize || claim.Height != references.LandClaimSize {
			add(path, "out_of_range", "land claim size must match the claim rule")
		}
		for _, earlier := range claims[:index] {
			if craftingClaimsOverlap(earlier, claim) {
				add(path, "invalid", "land claims must not overlap")

				break
			}
		}
	}
}

// craftingMapSize is the overworld width and height; claims and coordinates wrap at it.
const craftingMapSize = int(references.XLargeMapTiles)

// craftingWrappedOffset is how far value lies east or south of origin on the wrapping overworld.
func craftingWrappedOffset(origin, value int) int {
	return ((value-origin)%craftingMapSize + craftingMapSize) % craftingMapSize
}

func craftingClaimsOverlap(a, b LandClaimSnapshot) bool {
	axisOverlaps := func(first, second int) bool {
		return craftingWrappedOffset(first, second) < references.LandClaimSize ||
			craftingWrappedOffset(second, first) < references.LandClaimSize
	}

	return axisOverlaps(a.X, b.X) && axisOverlaps(a.Y, b.Y)
}

func validateCraftingCoordinate(path string, x, y int, add func(string, string, string)) {
	if x < 0 || x >= craftingMapSize || y < 0 || y >= craftingMapSize {
		add(path, "out_of_range", "coordinates must lie on the 256x256 overworld")
	}
}
