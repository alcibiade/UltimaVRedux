package game_state

import (
	"fmt"
	"slices"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// REDUX DEVIATION: CRAFT-0010: the original has no building. With Gameplay.Crafting.Enabled,
// Alt+B on the party's own deed map places crafted pieces and demolishes them; see ref:CRAFT-0010.

// DeedPlacement is one piece the party built on a deed map. Under is the ground the piece
// replaced. A piece built on a placed floor keeps that floor beneath it (OnFloor, Floor), and
// demolishing the piece uncovers the floor rather than the ground.
type DeedPlacement struct {
	Piece   references.BuildPieceID
	Under   indexes.SpriteIndex
	OnFloor bool
	Floor   references.BuildPieceID
}

// Placement returns the piece built at a deed-map square, if any.
func (d *Deed) Placement(position references.Position) (DeedPlacement, bool) {
	placement, ok := d.placements[position]

	return placement, ok
}

// PlacementPositions returns every square with a built piece, in row-major order.
func (d *Deed) PlacementPositions() []references.Position {
	positions := make([]references.Position, 0, len(d.placements))
	for position := range d.placements {
		positions = append(positions, position)
	}
	slices.SortFunc(positions, compareRowMajor)

	return positions
}

// BuildPieceStock is one piece the party carries, as the Build menu shows it.
type BuildPieceStock struct {
	Piece    references.BuildPiece
	Quantity uint16
}

// BuildPieceStocks lists every piece the party carries, in catalogue order.
func (g *GameState) BuildPieceStocks() []BuildPieceStock {
	stocks := make([]BuildPieceStock, 0, references.NumberOfBuildPieces)
	for _, piece := range references.BuildPieceCatalog() {
		if quantity := g.PartyState.Inventory.Pieces.Get(piece.ID); quantity > 0 {
			stocks = append(stocks, BuildPieceStock{Piece: piece, Quantity: quantity})
		}
	}

	return stocks
}

// isDeedBuildSquare reports whether a deed-map square lies inside the open interior, clear of the
// decorative ring and its ragged fringe.
func isDeedBuildSquare(position references.Position) bool {
	return isDeedMapSquare(position) && references.DeedSquareDepth(position) >= references.DeedInteriorDepth
}

// deedBuildTarget resolves the square a Build or Demolish acts on, and prints the refusal when
// there is none (ref:CRAFT-0010).
func (g *GameState) deedBuildTarget(direction references.Direction) (references.Position, bool) {
	messages := g.SystemCallbacks.Message
	if !g.CraftingCommandApplies(CraftingBuild) || !g.IsOnDeedMap() {
		messages.AddRowStr("Not here!")

		return references.Position{}, false
	}
	target := direction.GetNewPositionInDirection(&g.MapState.PlayerLocation.Position)
	if target == nil || !isDeedMapSquare(*target) {
		messages.AddRowStr("Thou canst not build there!")

		return references.Position{}, false
	}
	if g.deedSquareOccupied(*target) {
		messages.AddRowStr("Something is in the way!")

		return references.Position{}, false
	}

	return *target, true
}

// BuildDeedPiece places one carried piece on the deed-map square next to the party
// (ref:CRAFT-0010) and reports whether it did. A placement spends the piece, writes its tile with
// SetDeedTile, records what it covered and spends the build minutes beyond the ordinary turn; the
// caller then completes the ordinary command cycle. A refusal prints why and spends nothing.
func (g *GameState) BuildDeedPiece(id references.BuildPieceID, direction references.Direction) bool {
	messages := g.SystemCallbacks.Message
	piece, known := references.BuildPieceByID(id)
	if !known {
		panic(fmt.Sprintf("BuildDeedPiece: unknown piece %d", id))
	}
	position, ok := g.deedBuildTarget(direction)
	if !ok {
		return false
	}
	deed := g.crafting.active
	if _, isNode := g.crafting.activeLayout.NodeAt(position); isNode || !isDeedBuildSquare(position) {
		messages.AddRowStr("Thou canst not build there!")

		return false
	}
	placement := DeedPlacement{Piece: id}
	if existing, built := deed.placements[position]; built {
		if !existing.Piece.Piece().IsFloor() || piece.IsFloor() {
			messages.AddRowStr("Something is built there!")

			return false
		}
		placement.Under, placement.OnFloor, placement.Floor = existing.Under, true, existing.Piece
	} else {
		current, _ := g.DeedTileAt(deed.ID, position)
		if !references.IsDeedBuildableGround(current) {
			messages.AddRowStr("Thou canst not build there!")

			return false
		}
		placement.Under = current
	}
	if !g.PartyState.Inventory.Pieces.DecrementBy(id, 1) {
		messages.AddRowStr(fmt.Sprintf("Thou hast no %s!", piece.Name))

		return false
	}
	if deed.placements == nil {
		deed.placements = make(map[references.Position]DeedPlacement)
	}
	deed.placements[position] = placement
	if piece.StorageChest {
		if deed.chests == nil {
			deed.chests = make(map[references.Position]*StorageChest)
		}
		deed.chests[position] = &StorageChest{}
	}
	g.SetDeedTile(deed.ID, position, piece.Sprite)
	messages.AddRowStr(fmt.Sprintf("Built: %s.", piece.Name))
	g.SystemCallbacks.Audio.Play(SoundPushObject).Start()
	g.spendCraftingLabor(references.BuildPlaceMinutes)

	return true
}

// DemolishDeedPiece removes the piece the party built on the deed-map square next to the party
// (ref:CRAFT-0010) and reports whether it did. The tile beneath returns (a placed floor, or the
// ground) and the piece goes back to stock. A storage chest must be empty first. Demolishing
// spends the demolish minutes beyond the ordinary turn; a refusal spends nothing.
func (g *GameState) DemolishDeedPiece(direction references.Direction) bool {
	messages := g.SystemCallbacks.Message
	position, ok := g.deedBuildTarget(direction)
	if !ok {
		return false
	}
	deed := g.crafting.active
	placement, built := deed.placements[position]
	if !built {
		messages.AddRowStr("Thou didst not build that!")

		return false
	}
	piece := placement.Piece.Piece()
	if chest, isChest := deed.chests[position]; isChest && !chest.IsEmpty() {
		messages.AddRowStr("Empty the chest first!")

		return false
	}
	if g.PartyState.Inventory.Pieces.Get(piece.ID) >= party_state.ItemQuantitySmallMax {
		messages.AddRowStr(fmt.Sprintf("Thou canst carry no more %s!", piece.Name))

		return false
	}
	// An opened built door lives in the override layer with its closing timer; the timer must
	// not later restore a door on the uncovered square (ref:CMD-0006).
	g.forgetTrackedOpenDoorAt(position)
	delete(deed.chests, position)
	uncovered := placement.Under
	if placement.OnFloor {
		uncovered = placement.Floor.Piece().Sprite
		deed.placements[position] = DeedPlacement{Piece: placement.Floor, Under: placement.Under}
	} else {
		delete(deed.placements, position)
	}
	g.SetDeedTile(deed.ID, position, uncovered)
	g.PartyState.Inventory.Pieces.IncrementBy(piece.ID, 1)
	messages.AddRowStr(fmt.Sprintf("Demolished: %s.", piece.Name))
	g.SystemCallbacks.Audio.Play(SoundPushObject).Start()
	g.spendCraftingLabor(references.BuildDemolishMinutes)

	return true
}

// isBuiltDeedPiece reports whether the party built a piece on the square of the loaded deed map.
// Such a piece cannot be pushed or pulled: its placement, chest and station reach stay on its
// square (ref:CRAFT-0009).
func (g *GameState) isBuiltDeedPiece(position references.Position) bool {
	if !g.IsOnDeedMap() {
		return false
	}
	_, built := g.crafting.active.placements[position]

	return built
}

// forgetTrackedOpenDoorAt drops the player-open door timer when it belongs to the square, without
// closing the door: the square is about to get a new tile.
func (g *GameState) forgetTrackedOpenDoorAt(position references.Position) {
	door := g.MapState.TrackedOpenDoor()
	if door != nil && door.Position.Position == position && door.Position.Floor == g.MapState.PlayerLocation.Floor {
		g.MapState.ResetTransientDoor()
	}
}

// deedStationWithinReach reports whether a placed piece serving as the station stands within one
// square of the party, diagonals and the party's own square included, on the loaded deed map.
func (g *GameState) deedStationWithinReach(station references.CraftStation) bool {
	if !g.IsOnDeedMap() {
		return false
	}
	party := g.MapState.PlayerLocation.Position
	for dy := references.Coordinate(-1); dy <= 1; dy++ {
		for dx := references.Coordinate(-1); dx <= 1; dx++ {
			placement, built := g.crafting.active.placements[references.Position{X: party.X + dx, Y: party.Y + dy}]
			if built && placement.Piece.Piece().Station == station {
				return true
			}
		}
	}

	return false
}

// deedPieceLookDescription names a built piece whose original tile's Look text would mislead,
// on the loaded deed map (ref:CRAFT-0009). Other pieces keep their original Look text.
func (g *GameState) deedPieceLookDescription(position references.Position) (string, bool) {
	if !g.IsOnDeedMap() {
		return "", false
	}
	placement, built := g.crafting.active.placements[position]
	if !built {
		return "", false
	}
	piece := placement.Piece.Piece()
	if piece.LookName == "" {
		return "", false
	}
	if chest, isChest := g.crafting.active.chests[position]; isChest && chest.IsEmpty() {
		return piece.LookName + " (empty)", true
	}

	return piece.LookName, true
}
