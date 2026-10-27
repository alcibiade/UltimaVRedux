package save_snapshot

import (
	"fmt"
	"unicode/utf8"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const craftingMapSize = 256

// validateCrafting checks the Redux crafting extension (ref:CRAFT-0001).
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
	if len(crafting.LandClaims) > references.MaxLandClaims {
		add("crafting.landClaims", "out_of_range", "too many land claims")
	}
	claimIDs := make(map[int]struct{}, len(crafting.LandClaims))
	for index, claim := range crafting.LandClaims {
		path := fmt.Sprintf("crafting.landClaims[%d]", index)
		if _, exists := claimIDs[claim.ID]; exists || claim.ID < 0 {
			add(path+".id", "duplicate", "land claim IDs must be unique and non-negative")
		}
		claimIDs[claim.ID] = struct{}{}
		nameLength := utf8.RuneCountInString(claim.Name)
		if !utf8.ValidString(claim.Name) || nameLength == 0 || nameLength > references.MaxLandClaimNameLength {
			add(path+".name", "invalid", "land claim name must be 1 to 24 UTF-8 characters")
		}
		validateCraftingCoordinate(path, claim.X, claim.Y, add)
		if claim.Width != references.LandClaimSize || claim.Height != references.LandClaimSize {
			add(path, "out_of_range", "land claim size must match the claim rule")
		}
	}
	placements := make(map[[2]int]struct{}, len(crafting.Placements))
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
		placements[key] = struct{}{}
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
}

func validateCraftingCoordinate(path string, x, y int, add func(string, string, string)) {
	if x < 0 || x >= craftingMapSize || y < 0 || y >= craftingMapSize {
		add(path, "out_of_range", "coordinates must lie on the 256x256 overworld")
	}
}
