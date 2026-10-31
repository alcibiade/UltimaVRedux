package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// REDUX DEVIATION: CRAFT-0012: the original rests the party fully only at a paid inn; a town bed
// gives no healing. Hole Up on a bed the party built on its own deed rests it like an inn, for
// free; see ref:CRAFT-0012.

// IsOnOwnDeedBed reports whether the party stands on the head of a bed it built on its own deed
// map, with the option on. Hole Up there rests like an inn (ref:CRAFT-0012).
func (g *GameState) IsOnOwnDeedBed() bool {
	if !g.IsCraftingEnabled() || !g.IsOnDeedMap() {
		return false
	}
	position := g.MapState.PlayerLocation.Position
	placement, built := g.crafting.active.placements[position]
	if !built || !placement.Piece.Piece().BedHead {
		return false
	}
	tile := g.GetLayeredMapByCurrentLocation().GetTileByLayer(map_state.MapLayer, &position)

	return tile != nil && tile.Index == indexes.LeftBed
}

// DeedBedRestResult reports what one night in the party's own bed did.
type DeedBedRestResult struct {
	DateTimeBefore datetime.UltimaDate
	DateTimeAfter  datetime.UltimaDate
	// PoisonDeaths names members who were still poisoned at dawn and died, as at an inn.
	PoisonDeaths []string
	Regeneration []InnRegenerationResult
}

// RestInOwnDeedBed sleeps the night in the party's own bed (ref:CRAFT-0012): the inn overnight of
// ref:CAMP-0007 with no charge, no vacancy check and no theft. The party stays on the bed; the
// scene moves it out afterwards as after any bed rest. The caller checked IsOnOwnDeedBed.
func (g *GameState) RestInOwnDeedBed() DeedBedRestResult {
	if !g.IsOnOwnDeedBed() {
		panic("RestInOwnDeedBed: the party is not on its own bed")
	}
	partySize, err := g.validateContiguousActiveParty()
	if err != nil {
		panic("RestInOwnDeedBed: invalid packed party: " + err.Error())
	}
	result := DeedBedRestResult{DateTimeBefore: g.DateTime}
	poisoned := make([]bool, partySize)
	for index := range partySize {
		poisoned[index] = g.PartyState.GetCharacter(index).GetStatus() == party_state.Poisoned
	}
	result.Regeneration = g.sleepInnNight(partySize)
	for index := range partySize {
		character := g.PartyState.GetCharacter(index)
		if poisoned[index] && character.GetStatus() == party_state.Dead {
			result.PoisonDeaths = append(result.PoisonDeaths, character.GetNameAsString())
		}
	}
	result.DateTimeAfter = g.DateTime

	return result
}
