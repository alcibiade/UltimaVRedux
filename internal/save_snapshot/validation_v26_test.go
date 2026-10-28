package save_snapshot

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// validCraftingSnapshot holds a claim wrapping the world's corner, a floor and a stocked storage chest.
func validCraftingSnapshot() CraftingSnapshot {
	return CraftingSnapshot{
		Materials: [CraftingMaterialCount]uint16{10, 5},
		Tools:     [CraftingToolCount]uint16{1, 0},
		LandClaims: []LandClaimSnapshot{
			{ID: 0, Name: "Homestead", X: 250, Y: 250, Width: 16, Height: 16},
			{ID: 3, Name: "Lord's Rest", X: 100, Y: 100, Width: 16, Height: 16},
		},
		Placements: []PlacementSnapshot{
			{X: 2, Y: 3, PieceID: 0, Under: 5},
			{X: 252, Y: 251, PieceID: 22, Under: 5},
		},
		HomeChests: []HomeChestSnapshot{{X: 252, Y: 251, Provisions: [CraftingProvisionCount]uint16{0, 0, 0, 0, 0, 300}}},
	}
}

func TestValidateCraftingRejectsEachInvalidShape(t *testing.T) {
	var issues []Issue
	add := func(path, code, message string) {
		issues = append(issues, Issue{Path: path, Code: code, Message: message})
	}
	validateCrafting(validCraftingSnapshot(), add)
	assert.Empty(t, issues, "the baseline snapshot is valid")

	for _, testCase := range []struct {
		name   string
		mutate func(*CraftingSnapshot)
		path   string
		code   string
	}{
		{"material over cap", func(c *CraftingSnapshot) { c.Materials[0] = 10000 }, "crafting.materials[0]", "out_of_range"},
		{"tool over cap", func(c *CraftingSnapshot) { c.Tools[1] = 100 }, "crafting.tools[1]", "out_of_range"},
		{"too many claims", func(c *CraftingSnapshot) {
			for index := range references.MaxLandClaims - 1 {
				c.LandClaims = append(c.LandClaims, LandClaimSnapshot{ID: 10 + index, Name: "Spare", X: 20 * index, Y: 150, Width: 16, Height: 16})
			}
		}, "crafting.landClaims", "out_of_range"},
		{"duplicate claim ID", func(c *CraftingSnapshot) { c.LandClaims[1].ID = 0 }, "crafting.landClaims[1].id", "duplicate"},
		{"negative claim ID", func(c *CraftingSnapshot) { c.LandClaims[0].ID = -1 }, "crafting.landClaims[0].id", "duplicate"},
		{"empty name", func(c *CraftingSnapshot) { c.LandClaims[0].Name = "" }, "crafting.landClaims[0].name", "invalid"},
		{"padded name", func(c *CraftingSnapshot) { c.LandClaims[0].Name = " Home" }, "crafting.landClaims[0].name", "invalid"},
		{"non-ASCII name", func(c *CraftingSnapshot) { c.LandClaims[0].Name = "Café" }, "crafting.landClaims[0].name", "invalid"},
		{"long name", func(c *CraftingSnapshot) { c.LandClaims[0].Name = "abcdefghijklmnopqrstuvwxy" }, "crafting.landClaims[0].name", "invalid"},
		{"claim off the map", func(c *CraftingSnapshot) { c.LandClaims[1].X = 256 }, "crafting.landClaims[1]", "out_of_range"},
		{"claim size", func(c *CraftingSnapshot) { c.LandClaims[1].Width = 15 }, "crafting.landClaims[1]", "out_of_range"},
		{"overlapping claims across the seam", func(c *CraftingSnapshot) { c.LandClaims[1].X, c.LandClaims[1].Y = 5, 240 }, "crafting.landClaims[1]", "invalid"},
		{"unknown piece", func(c *CraftingSnapshot) { c.Placements[0].PieceID = 99 }, "crafting.placements[0].pieceId", "unsupported"},
		{"under out of range", func(c *CraftingSnapshot) { c.Placements[0].Under = 256 }, "crafting.placements[0].under", "out_of_range"},
		{"duplicate placement", func(c *CraftingSnapshot) {
			c.Placements = append(c.Placements, PlacementSnapshot{X: 2, Y: 3, PieceID: 1, Under: 5})
		}, "crafting.placements[2]", "duplicate"},
		{"placement outside every claim", func(c *CraftingSnapshot) { c.Placements[0].X = 50 }, "crafting.placements[0]", "invalid"},
		{"duplicate chest", func(c *CraftingSnapshot) { c.HomeChests = append(c.HomeChests, c.HomeChests[0]) }, "crafting.homeChests[1]", "duplicate"},
		{"chest without a chest placement", func(c *CraftingSnapshot) { c.Placements[1].PieceID = 17 }, "crafting.homeChests[0]", "invalid"},
		{"chest placement without contents", func(c *CraftingSnapshot) { c.HomeChests = nil }, "crafting.placements[1]", "invalid"},
		{"chest provision over cap", func(c *CraftingSnapshot) { c.HomeChests[0].Provisions[5] = 10000 }, "crafting.homeChests[0].provisions[5]", "out_of_range"},
		{"chest tool over cap", func(c *CraftingSnapshot) { c.HomeChests[0].Tools[0] = 100 }, "crafting.homeChests[0].tools[0]", "out_of_range"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			crafting := validCraftingSnapshot()
			testCase.mutate(&crafting)
			issues = nil

			validateCrafting(crafting, add)

			assert.Contains(t, issues, Issue{Path: testCase.path, Code: testCase.code, Message: issueMessage(issues, testCase.path)})
		})
	}
}

func issueMessage(issues []Issue, path string) string {
	for _, issue := range issues {
		if issue.Path == path {
			return issue.Message
		}
	}

	return ""
}
