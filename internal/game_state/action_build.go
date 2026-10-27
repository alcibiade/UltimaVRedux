package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// LandClaimProposal describes whether the party may stake a claim where it stands.
type LandClaimProposal struct {
	Origin references.Position
	Err    error
}

// ProposeLandClaim checks every claim rule for a claim centred on the party.
func (g *GameState) ProposeLandClaim() LandClaimProposal {
	if !g.IsCraftingEnabled() {
		return LandClaimProposal{Err: ErrCraftingDisabled}
	}
	if !g.isOnOverworldSurface() {
		return LandClaimProposal{Err: ErrCraftingNotOnSurface}
	}
	if len(g.crafting.claims) >= references.MaxLandClaims {
		return LandClaimProposal{Err: ErrTooManyLandClaims}
	}
	half := references.Coordinate(references.LandClaimSize / 2)
	party := g.MapState.PlayerLocation.Position
	origin := *(&references.Position{X: party.X - half, Y: party.Y - half}).GetWrapped(
		references.XLargeMapTiles, references.YLargeMapTiles)
	proposed := LandClaim{Origin: origin}
	worldLocations := g.GameReferences.LocationReferences.WorldLocations
	overworld := g.overworldMap()
	water := 0
	for dy := range references.LandClaimSize {
		for dx := range references.LandClaimSize {
			position := *(&references.Position{
				X: origin.X + references.Coordinate(dx), Y: origin.Y + references.Coordinate(dy),
			}).GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
			if _, claimed := g.LandClaimAt(position); claimed {
				return LandClaimProposal{Err: ErrLandAlreadyClaimed}
			}
			if worldLocations.GetLocationByPosition(position) != references.EmptyLocation ||
				isSettlementSprite(overworld.SpriteIndexByLayer(map_state.MapLayer, position)) {
				return LandClaimProposal{Err: ErrLandClaimHasSettlement}
			}
			if tile := g.GameReferences.TileReferences.GetTile(overworld.SpriteIndexByLayer(map_state.MapLayer, position)); tile != nil && tile.IsWater() {
				water++
			}
		}
	}
	if water > references.MaxLandClaimWaterTiles {
		return LandClaimProposal{Err: ErrLandClaimTooWet}
	}
	if !proposed.Contains(party) {
		panic("land claim proposal does not contain the party")
	}
	if g.PartyState.Inventory.Gold.Get() < references.LandClaimGoldCost {
		return LandClaimProposal{Origin: origin, Err: ErrNotEnoughGold}
	}
	if !g.PartyState.Inventory.CanAfford(references.LandClaimStakeCost) {
		return LandClaimProposal{Origin: origin, Err: ErrNotEnoughMaterials}
	}

	return LandClaimProposal{Origin: origin}
}

func isSettlementSprite(sprite indexes.SpriteIndex) bool {
	return (sprite >= indexes.Hut && sprite <= indexes.Lighthouse) ||
		sprite == indexes.EvilCastleEntrance || sprite == indexes.CastleBritianEntrace
}

// LandClaimRejection returns the player-facing reason a proposal failed.
func LandClaimRejection(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrTooManyLandClaims):
		return fmt.Sprintf("You may hold only %d claims.", references.MaxLandClaims)
	case errors.Is(err, ErrLandAlreadyClaimed):
		return "That land is already yours!"
	case errors.Is(err, ErrLandClaimHasSettlement):
		return "Too near a settlement!"
	case errors.Is(err, ErrLandClaimTooWet):
		return "Too much water here!"
	case errors.Is(err, ErrNotEnoughGold):
		return fmt.Sprintf("A claim costs %d gold.", references.LandClaimGoldCost)
	case errors.Is(err, ErrNotEnoughMaterials):
		return fmt.Sprintf("Stakes need %s.", references.LandClaimStakeCost.Describe())
	default:
		return "Not here!"
	}
}

// StakeLandClaim pays for and records a named claim centred on the party.
// REDUX DEVIATION: CRAFT-0004: Redux-only land ownership; see ref:CRAFT-0004.
func (g *GameState) StakeLandClaim(name string) (LandClaim, error) {
	normalized, ok := NormalizeLandClaimName(name)
	if !ok {
		return LandClaim{}, ErrInvalidLandClaimName
	}
	proposal := g.ProposeLandClaim()
	if proposal.Err != nil {
		g.SystemCallbacks.Message.AddRowStr(LandClaimRejection(proposal.Err))

		return LandClaim{}, proposal.Err
	}
	inventory := &g.PartyState.Inventory
	if !inventory.SpendMaterials(references.LandClaimStakeCost) || !inventory.Gold.DecrementBy(references.LandClaimGoldCost) {
		panic("validated land claim payment failed")
	}
	nextID := 0
	for _, claim := range g.crafting.claims {
		nextID = max(nextID, claim.ID+1)
	}
	claim := LandClaim{ID: nextID, Name: normalized, Origin: proposal.Origin}
	g.crafting.claims = append(g.crafting.claims, claim)
	g.SystemCallbacks.Message.AddRowStr(fmt.Sprintf("You claim %s!", normalized))
	if !g.pinLandClaimOnAtlas(claim) {
		g.SystemCallbacks.Message.AddRowStr("Your Atlas has no room to mark it.")
	}
	g.AdvanceGameplayMinutesAndProcessProvisions(references.GatherMinutes)

	return claim, nil
}

// BuildPieceOption is one build piece as presented to the Build menu.
type BuildPieceOption struct {
	Piece      references.BuildPiece
	Affordable bool
}

// BuildPieceOptions lists every piece with whether the party can afford it now.
func (g *GameState) BuildPieceOptions() []BuildPieceOption {
	pieces := references.BuildPieces()
	options := make([]BuildPieceOption, 0, len(pieces))
	for _, piece := range pieces {
		options = append(options, BuildPieceOption{Piece: piece, Affordable: g.PartyState.Inventory.CanAfford(piece.Cost)})
	}

	return options
}

// buildTarget returns the wrapped overworld square in the direction, if the party may build there.
func (g *GameState) buildTarget(direction references.Direction) (references.Position, bool) {
	if !g.IsCraftingEnabled() {
		g.SystemCallbacks.Message.AddRowStr("What?")

		return references.Position{}, false
	}
	if !g.isOnOverworldSurface() || g.PartyVehicle.GetVehicleDetails().VehicleType != references.NoPartyVehicle {
		g.SystemCallbacks.Message.AddRowStr("Not here!")

		return references.Position{}, false
	}
	target := direction.GetNewPositionInDirection(&g.MapState.PlayerLocation.Position)
	if target == nil {
		g.SystemCallbacks.Message.AddRowStr("Not here!")

		return references.Position{}, false
	}
	position := *target.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
	if _, owned := g.LandClaimAt(position); !owned {
		g.SystemCallbacks.Message.AddRowStr("Not your land!")

		return references.Position{}, false
	}
	if g.largeMapPushOccupied(position) {
		g.SystemCallbacks.Message.AddRowStr("Something is in the way!")

		return references.Position{}, false
	}

	return position, true
}

// ActionBuildLargeMap places a build piece on the adjacent square of an owned claim.
// REDUX DEVIATION: CRAFT-0005: Redux-only building; see ref:CRAFT-0005.
func (g *GameState) ActionBuildLargeMap(pieceID int, direction references.Direction) bool {
	piece, ok := references.BuildPieceByID(pieceID)
	if !ok {
		g.SystemCallbacks.Message.AddRowStr("What?")

		return false
	}
	position, ok := g.buildTarget(direction)
	if !ok {
		return false
	}
	overworld := g.overworldMap()
	current := overworld.SpriteIndexByLayer(map_state.MapLayer, position)
	if !references.IsBuildableGround(current) {
		g.SystemCallbacks.Message.AddRowStr("Can't build there!")

		return false
	}
	if !g.PartyState.Inventory.SpendMaterials(piece.Cost) {
		g.SystemCallbacks.Message.AddRowStr(fmt.Sprintf("You need %s.", piece.Cost.Describe()))

		return false
	}
	g.crafting.ensure()
	under := current
	if previous, placed := g.crafting.placements[position]; placed {
		// Building over a placed floor keeps the original terrain underneath for demolition.
		under = previous.under
		if previousPiece, known := references.BuildPieceByID(previous.pieceID); known {
			g.PartyState.Inventory.AddMaterials(previousPiece.Cost.Halved())
		}
	}
	overworld.SetTileByLayer(map_state.MapLayer, &position, piece.Sprite)
	overworld.UnSetTileByLayer(map_state.MapOverrideLayer, &position)
	g.crafting.placements[position] = buildPlacement{pieceID: piece.ID, under: under}
	if piece.HomeChest {
		g.crafting.homeChests[position] = &HomeChest{}
	}
	g.SystemCallbacks.Message.AddRowStr(fmt.Sprintf("You build a %s.", piece.Name))
	g.SystemCallbacks.Audio.Play(SoundPushObject).Start()
	g.advanceCraftingLabor(references.BuildMinutes)

	return true
}

// ActionDemolishLargeMap removes a placed piece, restoring the terrain beneath and refunding half its cost.
// A storage chest must be empty first.
func (g *GameState) ActionDemolishLargeMap(direction references.Direction) bool {
	position, ok := g.buildTarget(direction)
	if !ok {
		return false
	}
	placement, placed := g.crafting.placements[position]
	if !placed {
		g.SystemCallbacks.Message.AddRowStr("You didn't build that!")

		return false
	}
	if chest, isChest := g.crafting.homeChests[position]; isChest && !chest.isEmpty() {
		g.SystemCallbacks.Message.AddRowStr("Empty the chest first!")

		return false
	}
	overworld := g.overworldMap()
	overworld.SetTileByLayer(map_state.MapLayer, &position, placement.under)
	overworld.UnSetTileByLayer(map_state.MapOverrideLayer, &position)
	delete(g.crafting.placements, position)
	delete(g.crafting.homeChests, position)
	if piece, known := references.BuildPieceByID(placement.pieceID); known {
		refund := piece.Cost.Halved()
		g.PartyState.Inventory.AddMaterials(refund)
		g.SystemCallbacks.Message.AddRowStr(fmt.Sprintf("Demolished! You salvage %s.", refund.Describe()))
	}
	g.advanceCraftingLabor(references.BuildMinutes)

	return true
}

func (c *HomeChest) isEmpty() bool {
	return c.Materials == references.MaterialCost{} && c.Tools == [references.NumberOfTools]uint16{} &&
		c.Provisions == [len(c.Provisions)]uint16{}
}
