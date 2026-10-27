package game_state

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// REDUX DEVIATION: CRAFT-0001: the original has no gathering, crafting, building or land ownership.
// Redux adds them on the overworld surface behind Gameplay.Crafting.Enabled; see ref:CRAFT-0001.

// Crafting rejection reasons.
var (
	ErrCraftingDisabled       = errors.New("crafting is disabled")
	ErrCraftingNotOnSurface   = errors.New("crafting works only on the overworld surface")
	ErrUnknownRecipe          = errors.New("unknown crafting recipe")
	ErrUnknownBuildPiece      = errors.New("unknown build piece")
	ErrNotEnoughMaterials     = errors.New("not enough materials")
	ErrNotEnoughGold          = errors.New("not enough gold")
	ErrTooManyLandClaims      = errors.New("too many land claims")
	ErrLandAlreadyClaimed     = errors.New("land is already claimed")
	ErrLandClaimHasSettlement = errors.New("a settlement lies within the claim")
	ErrLandClaimTooWet        = errors.New("too much water within the claim")
	ErrInvalidLandClaimName   = errors.New("invalid land claim name")
)

// LandClaim is one owned overworld rectangle. Origin is its north-west corner; the
// rectangle wraps with the overworld.
type LandClaim struct {
	ID     int
	Name   string
	Origin references.Position
}

// Center returns the claim's central tile.
func (c LandClaim) Center() references.Position {
	half := references.Coordinate(references.LandClaimSize / 2)

	return *(&references.Position{X: c.Origin.X + half, Y: c.Origin.Y + half}).GetWrapped(
		references.XLargeMapTiles, references.YLargeMapTiles)
}

// Contains reports whether the overworld position lies within the claim.
func (c LandClaim) Contains(position references.Position) bool {
	return wrappedOffset(c.Origin.X, position.X) < references.LandClaimSize &&
		wrappedOffset(c.Origin.Y, position.Y) < references.LandClaimSize
}

func wrappedOffset(origin, value references.Coordinate) int {
	size := int(references.XLargeMapTiles)

	return ((int(value)-int(origin))%size + size) % size
}

// HomeChest is the contents of one storage chest built inside a claim.
type HomeChest struct {
	Materials  references.MaterialCost
	Tools      [references.NumberOfTools]uint16
	Provisions [save_snapshot.CraftingProvisionCount]uint16
}

type buildPlacement struct {
	pieceID int
	under   indexes.SpriteIndex
}

// craftingState is the persistent Redux crafting world state.
type craftingState struct {
	claims     []LandClaim
	placements map[references.Position]buildPlacement
	homeChests map[references.Position]*HomeChest
}

func (c *craftingState) ensure() {
	if c.placements == nil {
		c.placements = make(map[references.Position]buildPlacement)
	}
	if c.homeChests == nil {
		c.homeChests = make(map[references.Position]*HomeChest)
	}
}

// IsCraftingEnabled reports whether the optional crafting rules are on.
func (g *GameState) IsCraftingEnabled() bool {
	return g.GameConfig != nil && g.GameConfig.GameplayConfiguration != nil &&
		g.GameConfig.GameplayConfiguration.CraftingEnabled
}

// isOnOverworldSurface reports whether the party stands on the overworld (floor 0).
func (g *GameState) isOnOverworldSurface() bool {
	return g.MapState.PlayerLocation.Location.GetMapType() == references.LargeMapType &&
		g.MapState.PlayerLocation.Floor == references.FloorNumber(references.OVERWORLD)
}

func (g *GameState) overworldMap() *map_state.LayeredMap {
	return g.MapState.LayeredMaps.GetLayeredMap(references.LargeMapType, references.FloorNumber(references.OVERWORLD))
}

// LandClaims returns a copy of the party's claims in creation order.
func (g *GameState) LandClaims() []LandClaim {
	return slices.Clone(g.crafting.claims)
}

// LandClaimAt returns the claim containing the overworld position, if any.
func (g *GameState) LandClaimAt(position references.Position) (LandClaim, bool) {
	for _, claim := range g.crafting.claims {
		if claim.Contains(position) {
			return claim, true
		}
	}

	return LandClaim{}, false
}

// IsInsideOwnClaim reports whether an overworld-surface position lies in one of the party's claims.
func (g *GameState) IsInsideOwnClaim(location references.Location, floor references.FloorNumber, position references.Position) bool {
	if location.GetMapType() != references.LargeMapType || floor != references.FloorNumber(references.OVERWORLD) {
		return false
	}
	_, ok := g.LandClaimAt(position)

	return ok
}

// isPartyInsideOwnClaim reports whether the party itself stands in one of its claims.
func (g *GameState) isPartyInsideOwnClaim() bool {
	return g.IsInsideOwnClaim(g.MapState.PlayerLocation.Location, g.MapState.PlayerLocation.Floor,
		g.MapState.PlayerLocation.Position)
}

// HomeChestAt returns the storage chest at the overworld position, if any.
func (g *GameState) HomeChestAt(position references.Position) (*HomeChest, bool) {
	chest, ok := g.crafting.homeChests[position]

	return chest, ok
}

// NormalizeLandClaimName trims a proposed claim name and reports whether it is acceptable.
func NormalizeLandClaimName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	length := utf8.RuneCountInString(name)

	return name, utf8.ValidString(name) && length > 0 && length <= references.MaxLandClaimNameLength
}

func (g *GameState) captureCrafting(snapshot *save_snapshot.SaveSnapshot) {
	inventory := &g.PartyState.Inventory
	crafting := save_snapshot.CraftingSnapshot{
		LandClaims: make([]save_snapshot.LandClaimSnapshot, 0, len(g.crafting.claims)),
		Placements: make([]save_snapshot.PlacementSnapshot, 0, len(g.crafting.placements)),
		HomeChests: make([]save_snapshot.HomeChestSnapshot, 0, len(g.crafting.homeChests)),
	}
	for _, material := range references.MaterialCatalog() {
		crafting.Materials[material] = inventory.Materials.Get(material)
	}
	for _, tool := range references.ToolCatalog() {
		crafting.Tools[tool] = inventory.Tools.Get(tool)
	}
	for _, claim := range g.crafting.claims {
		crafting.LandClaims = append(crafting.LandClaims, save_snapshot.LandClaimSnapshot{
			ID: claim.ID, Name: claim.Name, X: int(claim.Origin.X), Y: int(claim.Origin.Y),
			Width: references.LandClaimSize, Height: references.LandClaimSize,
		})
	}
	for _, position := range sortedPositions(g.crafting.placements) {
		placement := g.crafting.placements[position]
		crafting.Placements = append(crafting.Placements, save_snapshot.PlacementSnapshot{
			X: int(position.X), Y: int(position.Y), PieceID: placement.pieceID, Under: int(placement.under),
		})
	}
	for _, position := range sortedPositions(g.crafting.homeChests) {
		chest := g.crafting.homeChests[position]
		crafting.HomeChests = append(crafting.HomeChests, save_snapshot.HomeChestSnapshot{
			X: int(position.X), Y: int(position.Y),
			Materials: chest.Materials, Tools: chest.Tools, Provisions: chest.Provisions,
		})
	}
	snapshot.Crafting = crafting
}

func (g *GameState) restoreCrafting(saved save_snapshot.CraftingSnapshot) {
	inventory := &g.PartyState.Inventory
	for _, material := range references.MaterialCatalog() {
		inventory.Materials.Set(material, saved.Materials[material])
	}
	for _, tool := range references.ToolCatalog() {
		inventory.Tools.Set(tool, saved.Tools[tool])
	}
	g.crafting = craftingState{}
	g.crafting.ensure()
	for _, claim := range saved.LandClaims {
		g.crafting.claims = append(g.crafting.claims, LandClaim{
			ID: claim.ID, Name: claim.Name,
			Origin: references.Position{X: references.Coordinate(claim.X), Y: references.Coordinate(claim.Y)},
		})
	}
	for _, placement := range saved.Placements {
		position := references.Position{X: references.Coordinate(placement.X), Y: references.Coordinate(placement.Y)}
		g.crafting.placements[position] = buildPlacement{pieceID: placement.PieceID, under: indexes.SpriteIndex(placement.Under)}
	}
	for _, chest := range saved.HomeChests {
		position := references.Position{X: references.Coordinate(chest.X), Y: references.Coordinate(chest.Y)}
		g.crafting.homeChests[position] = &HomeChest{Materials: chest.Materials, Tools: chest.Tools, Provisions: chest.Provisions}
	}
}

func sortedPositions[V any](values map[references.Position]V) []references.Position {
	positions := make([]references.Position, 0, len(values))
	for position := range values {
		positions = append(positions, position)
	}
	slices.SortFunc(positions, func(a, b references.Position) int {
		if a.Y != b.Y {
			return int(a.Y) - int(b.Y)
		}

		return int(a.X) - int(b.X)
	})

	return positions
}

// isClaimSpawnBlocked suppresses overworld monster generation in and into land claims (ref:CRAFT-0006).
func (g *GameState) isClaimSpawnBlocked(position references.Position) bool {
	if !g.IsCraftingEnabled() || !g.isOnOverworldSurface() {
		return false
	}
	_, claimed := g.LandClaimAt(position)

	return claimed
}

// isClaimedCampSite reports whether a camp pitched at the location is inside an own claim (ref:CRAFT-0006).
func (g *GameState) isClaimedCampSite(site references.PlayerLocation) bool {
	return g.IsCraftingEnabled() && g.IsInsideOwnClaim(site.Location, site.Floor, site.Position)
}

// HomeChestCategory is a group of goods moved together into or out of a storage chest.
type HomeChestCategory int

// Home chest categories, in menu order.
const (
	HomeChestMaterials HomeChestCategory = iota
	HomeChestTools
	HomeChestGold
	HomeChestFood
	HomeChestSupplies
)

// HomeChestCategories returns every category in menu order.
func HomeChestCategories() []HomeChestCategory {
	return []HomeChestCategory{HomeChestMaterials, HomeChestTools, HomeChestGold, HomeChestFood, HomeChestSupplies}
}

// Name returns the menu label for the category.
func (c HomeChestCategory) Name() string {
	switch c {
	case HomeChestMaterials:
		return "materials"
	case HomeChestTools:
		return "tools"
	case HomeChestGold:
		return "gold"
	case HomeChestFood:
		return "food"
	case HomeChestSupplies:
		return "torches, gems and keys"
	default:
		return "goods"
	}
}

// homeChestSlot pairs one party stack with the chest counter that stores it.
type homeChestSlot struct {
	party party_state.ItemQuantity
	chest *uint16
	limit uint16
}

func (g *GameState) homeChestSlots(chest *HomeChest, category HomeChestCategory) []homeChestSlot {
	inventory := &g.PartyState.Inventory
	provision := func(party party_state.ItemQuantity, which references.Provision, limit uint16) homeChestSlot {
		return homeChestSlot{party: party, chest: &chest.Provisions[which], limit: limit}
	}
	switch category {
	case HomeChestMaterials:
		slots := make([]homeChestSlot, 0, references.NumberOfMaterials)
		for _, material := range references.MaterialCatalog() {
			slots = append(slots, homeChestSlot{party: *inventory.Materials.GetQuantity(material),
				chest: &chest.Materials[material], limit: party_state.ItemQuantityLargeMax})
		}

		return slots
	case HomeChestTools:
		slots := make([]homeChestSlot, 0, references.NumberOfTools)
		for _, tool := range references.ToolCatalog() {
			slots = append(slots, homeChestSlot{party: *inventory.Tools.GetQuantity(tool),
				chest: &chest.Tools[tool], limit: party_state.ItemQuantitySmallMax})
		}

		return slots
	case HomeChestGold:
		return []homeChestSlot{provision(&inventory.Gold, references.Gold, party_state.ItemQuantityLargeMax)}
	case HomeChestFood:
		return []homeChestSlot{provision(&inventory.Provisions.Food, references.Food, party_state.ItemQuantityLargeMax)}
	case HomeChestSupplies:
		return []homeChestSlot{
			provision(&inventory.Provisions.Torches, references.Torches, party_state.ItemQuantitySmallMax),
			provision(&inventory.Provisions.Gems, references.Gem, party_state.ItemQuantitySmallMax),
			provision(&inventory.Provisions.Keys, references.Key, party_state.ItemQuantitySmallMax),
			provision(&inventory.Provisions.SkullKeys, references.SkullKeys, party_state.ItemQuantitySmallMax),
		}
	default:
		return nil
	}
}

// HomeChestInDirection returns the storage chest adjacent to the party in the direction, if any.
func (g *GameState) HomeChestInDirection(direction references.Direction) (references.Position, bool) {
	if !g.IsCraftingEnabled() || !g.isOnOverworldSurface() {
		return references.Position{}, false
	}
	target := direction.GetNewPositionInDirection(&g.MapState.PlayerLocation.Position)
	if target == nil {
		return references.Position{}, false
	}
	position := *target.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
	if _, ok := g.crafting.homeChests[position]; !ok {
		return references.Position{}, false
	}

	return position, true
}

// TransferHomeChest moves every unit of a category into (deposit) or out of a storage chest,
// up to the receiving stack's cap, and reports how many units moved.
// REDUX DEVIATION: CRAFT-0007: Redux-only storage; opening it costs no karma (ref:CRAFT-0007).
func (g *GameState) TransferHomeChest(position references.Position, category HomeChestCategory, deposit bool) int {
	chest, ok := g.crafting.homeChests[position]
	if !ok {
		return 0
	}
	moved := 0
	for _, slot := range g.homeChestSlots(chest, category) {
		if deposit {
			amount := min(slot.party.Get(), slot.limit-*slot.chest)
			if amount > 0 && slot.party.DecrementBy(amount) {
				*slot.chest += amount
				moved += int(amount)
			}

			continue
		}
		amount := min(*slot.chest, slot.limit-min(slot.limit, slot.party.Get()))
		if amount > 0 {
			slot.party.IncrementBy(amount)
			*slot.chest -= amount
			moved += int(amount)
		}
	}

	return moved
}

// HomeChestSummary describes a chest's contents for the storage menu.
func (g *GameState) HomeChestSummary(position references.Position) string {
	chest, ok := g.crafting.homeChests[position]
	if !ok {
		return ""
	}
	parts := make([]string, 0, 8)
	for _, material := range references.MaterialCatalog() {
		if chest.Materials[material] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", chest.Materials[material], material.Name()))
		}
	}
	for _, tool := range references.ToolCatalog() {
		if chest.Tools[tool] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", chest.Tools[tool], tool.Name()))
		}
	}
	names := [...]string{"Torches", "Gems", "Keys", "Skull Keys", "Food", "Gold"}
	for index, quantity := range chest.Provisions {
		if quantity > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", quantity, names[index]))
		}
	}
	if len(parts) == 0 {
		return "The chest is empty."
	}

	return "Chest: " + strings.Join(parts, ", ")
}

// advanceCraftingLabor spends an ordinary surface turn plus any extra minutes the labor takes.
// Surface command routing owns the ordinary turn; the extra minutes always pass.
func (g *GameState) advanceCraftingLabor(minutes int) {
	g.AdvanceOrdinarySurfaceCommandMinutes(DefaultLargeMapMinutesPerTurn)
	if extra := minutes - DefaultLargeMapMinutesPerTurn; extra > 0 {
		g.AdvanceGameplayMinutesAndProcessProvisions(extra)
	}
}

// pinLandClaimOnAtlas adds an Atlas note at the claim's centre, without needing a sextant fix:
// the party knows where its own land lies. A full or occupied Atlas leaves the claim unpinned.
func (g *GameState) pinLandClaimOnAtlas(claim LandClaim) bool {
	center := claim.Center()
	record := journal.AtlasNoteRecord{
		Position: journal.SiteLocation{Map: references.BritanniaUnderworld, Floor: references.FloorNumber(references.OVERWORLD),
			X: int(center.X), Y: int(center.Y)},
		Label:       claim.Name,
		Description: "Your land claim.",
		RecordedAt:  g.sampleDungeonObservation().at.Time,
	}
	candidate := g.CaptureAtlasNotes()
	if len(candidate.Records) >= journal.MaxAtlasNotes || candidate.LastID >= journal.MaxAtlasIdentity {
		return false
	}
	for _, existing := range candidate.Records {
		if existing.Position == record.Position {
			return false
		}
	}
	record.ID = candidate.LastID + 1
	candidate.LastID = record.ID
	candidate.Records = append(candidate.Records, record)
	if journal.ValidateAtlasNotes(candidate) != nil {
		return false
	}
	g.atlasNotes = candidate

	return true
}
