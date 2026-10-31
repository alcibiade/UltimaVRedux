package game_state

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// DeedProposal describes whether the party may stake a deed on the square it stands on.
type DeedProposal struct {
	Square  references.Position
	Terrain references.DeedTerrain
	Err     error
}

// ProposeDeed checks every deed rule for the party's square (ref:CRAFT-0002). The gold check
// comes last, so a refusal for lack of gold means the square itself is suitable.
func (g *GameState) ProposeDeed() DeedProposal {
	square := g.MapState.PlayerLocation.Position
	proposal := DeedProposal{Square: square}
	switch {
	case !g.IsCraftingEnabled():
		proposal.Err = ErrCraftingDisabled
	case !g.isOnOverworldSurface():
		proposal.Err = ErrDeedNotOnSurface
	case !g.isPartyOnFoot():
		proposal.Err = ErrDeedNotOnFoot
	case g.deedAt(square) != nil:
		proposal.Err = ErrDeedAlreadyOwned
	case g.touchesDeed(square):
		proposal.Err = ErrDeedTooClose
	case g.GameReferences.LocationReferences.WorldLocations.GetLocationByPosition(square) != references.EmptyLocation:
		proposal.Err = ErrDeedSettlement
	case g.isMoongateSquare(square):
		proposal.Err = ErrDeedMoongate
	}
	if proposal.Err != nil {
		return proposal
	}
	terrain, buildable := references.DeedTerrainForSprite(g.overworldMap().SpriteIndexByLayer(map_state.MapLayer, square))
	switch {
	case !buildable:
		proposal.Err = ErrDeedNotBuildable
	case g.PartyState.Inventory.Gold.Get() < references.DeedGoldCost:
		proposal.Err = ErrDeedNotEnoughGold
	}
	proposal.Terrain = terrain

	return proposal
}

// touchesDeed reports whether an orthogonal neighbour of the square, wrapping with the
// overworld, is a deed square, so two huts never stand side by side.
func (g *GameState) touchesDeed(square references.Position) bool {
	for _, direction := range []references.Direction{references.Up, references.Down, references.Left, references.Right} {
		neighbour := direction.GetNewPositionInDirection(&square)
		if neighbour == nil {
			continue
		}
		if g.deedAt(*neighbour.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)) != nil {
			return true
		}
	}

	return false
}

// isMoongateSquare reports whether a moongate can open on the overworld square: one of the
// eight charted moonstone sites, or wherever a moonstone now lies buried.
func (g *GameState) isMoongateSquare(square references.Position) bool {
	for _, origin := range references.ChartedMoonstoneOrigins() {
		if origin.Location.Floor == references.FloorNumber(references.OVERWORLD) && origin.Location.Position == square {
			return true
		}
	}
	if g.moonstoneSystem == nil {
		g.moonstoneSystem = NewMoonstoneSystem()
	}

	return g.moonstoneSystem.CanCreateMoongateAt(references.BritanniaUnderworld,
		references.FloorNumber(references.OVERWORLD), square)
}

// DeedRejection returns the player-facing reason a proposal failed.
func DeedRejection(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrCraftingDisabled):
		return "What?"
	case errors.Is(err, ErrDeedAlreadyOwned):
		return "This land is already yours!"
	case errors.Is(err, ErrDeedTooClose):
		return "Too close to your other land!"
	case errors.Is(err, ErrDeedSettlement):
		return "Someone else lives here!"
	case errors.Is(err, ErrDeedMoongate):
		return "The moongates belong to no one!"
	case errors.Is(err, ErrDeedNotBuildable):
		return "No one can build here!"
	case errors.Is(err, ErrDeedNotOnFoot):
		return "On foot!"
	case errors.Is(err, ErrDeedNotEnoughGold):
		return fmt.Sprintf("A deed costs %d gold!", references.DeedGoldCost)
	default:
		return "Not here!"
	}
}

// NormalizeDeedName trims a proposed deed name and reports whether it is acceptable.
func NormalizeDeedName(name string) (string, bool) {
	name = strings.TrimSpace(name)

	return name, references.IsValidDeedName(name)
}

// StakeDeed pays for a deed on the party's square, marks it with a hut and pins it on the
// Atlas. It spends no time itself: the scene completes the Build command with the ordinary
// surface turn (ref:CRAFT-0002).
// REDUX DEVIATION: CRAFT-0002: Redux-only land ownership; the original has no deeds.
func (g *GameState) StakeDeed(name string) (Deed, error) {
	normalized, ok := NormalizeDeedName(name)
	if !ok {
		return Deed{}, ErrInvalidDeedName
	}
	proposal := g.ProposeDeed()
	if proposal.Err != nil {
		g.SystemCallbacks.Message.AddRowStr(DeedRejection(proposal.Err))

		return Deed{}, proposal.Err
	}
	if !g.PartyState.Inventory.Gold.DecrementBy(references.DeedGoldCost) {
		panic("validated deed payment failed")
	}
	nextID := 0
	for _, deed := range g.crafting.deeds {
		nextID = max(nextID, deed.ID+1)
	}
	deed := &Deed{
		ID: nextID, Name: normalized, Square: proposal.Square, Terrain: proposal.Terrain,
		GeneratorVersion: references.DeedGeneratorVersion, Seed: references.DeedSeed(nextID, proposal.Square),
	}
	g.crafting.deeds = append(g.crafting.deeds, deed)
	// The hut is an ordinary overworld map mutation; the native save diffs it against BRIT.DAT.
	overworld := g.overworldMap()
	overworld.SetTileByLayer(map_state.MapLayer, &deed.Square, indexes.Hut)
	overworld.UnSetTileByLayer(map_state.MapOverrideLayer, &deed.Square)
	g.SystemCallbacks.Message.AddRowStr(fmt.Sprintf("%s is yours!", normalized))
	switch g.pinDeedOnAtlas(deed) {
	case atlasPinned, atlasAlreadyMarked:
	case atlasNoRoom:
		g.SystemCallbacks.Message.AddRowStr("Your Atlas has no room to mark it.")
	case atlasRefused:
		g.SystemCallbacks.Message.AddRowStr("Your Atlas cannot mark it.")
	}

	return *deed, nil
}

// atlasPinOutcome reports what pinning a deed on the Atlas did.
type atlasPinOutcome int

const (
	atlasPinned atlasPinOutcome = iota
	// atlasAlreadyMarked means a note already stands on the square; it is kept as is.
	atlasAlreadyMarked
	// atlasNoRoom means the Atlas holds its maximum number of notes.
	atlasNoRoom
	// atlasRefused means the note failed Atlas validation.
	atlasRefused
)

// pinDeedOnAtlas adds an Atlas note on the hut square, without a sextant fix: the party knows
// where its own land lies.
func (g *GameState) pinDeedOnAtlas(deed *Deed) atlasPinOutcome {
	record := journal.AtlasNoteRecord{
		Position: journal.SiteLocation{Map: references.BritanniaUnderworld, Floor: references.FloorNumber(references.OVERWORLD),
			X: int(deed.Square.X), Y: int(deed.Square.Y)},
		Label:       deed.Name,
		Description: fmt.Sprintf("Your deed: %s, 32 by 32 squares of %s.", deed.Name, deed.Terrain.Name()),
		RecordedAt:  g.sampleDungeonObservation().at.Time,
	}
	candidate := g.CaptureAtlasNotes()
	for _, existing := range candidate.Records {
		if existing.Position == record.Position {
			return atlasAlreadyMarked
		}
	}
	if len(candidate.Records) >= journal.MaxAtlasNotes || candidate.LastID >= journal.MaxAtlasIdentity {
		return atlasNoRoom
	}
	record.ID = candidate.LastID + 1
	candidate.LastID = record.ID
	candidate.Records = append(candidate.Records, record)
	if journal.ValidateAtlasNotes(candidate) != nil {
		return atlasRefused
	}
	g.atlasNotes = candidate

	return atlasPinned
}

// ResolveEnterDeed handles Enter on a deed's hut square and reports whether it did. With the
// option off a hut is plain scenery and Enter keeps its original answer (ref:CRAFT-0004).
func (g *GameState) ResolveEnterDeed() bool {
	if !g.IsCraftingEnabled() || !g.isOnOverworldSurface() {
		return false
	}
	deed := g.deedAt(g.MapState.PlayerLocation.Position)
	if deed == nil {
		return false
	}
	g.SystemCallbacks.Message.AppendToCurrentRowStr(deed.Name)
	g.StartEnterLocationWorkflow(g.activateDeed(deed))

	return true
}
