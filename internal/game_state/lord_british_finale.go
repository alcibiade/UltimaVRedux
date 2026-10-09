package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/minimap"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const lordBritishFinaleMapIndex = 3

var errLordBritishFinaleInvalid = errors.New("invalid Lord British finale transition")

// LordBritishFinaleOutcome is the authoritative, non-resumable game outcome.
type LordBritishFinaleOutcome uint8

const (
	// LordBritishFinaleInProgress marks the rescue and dialogue before an answer resolves it.
	LordBritishFinaleInProgress LordBritishFinaleOutcome = iota
	// LordBritishFinaleSucceeded is the terminal branch with the owned Wooden Box.
	LordBritishFinaleSucceeded
	// LordBritishFinaleFailed is every other accepted terminal branch.
	LordBritishFinaleFailed
)

// LordBritishFinalePrompt identifies the original finale rules question currently awaiting Y/N.
// See ref:DNG-0015.
type LordBritishFinalePrompt uint8

const (
	// LordBritishFinaleBoxPrompt asks whether the Avatar brought Lord British's box.
	LordBritishFinaleBoxPrompt LordBritishFinalePrompt = iota
	// LordBritishFinaleSandalwoodPrompt clarifies a first accepted No answer.
	LordBritishFinaleSandalwoodPrompt
	// LordBritishFinaleNoPrompt marks that the source dialogue has resolved.
	LordBritishFinaleNoPrompt
)

// LordBritishFinaleActor is one source-defined actor placement in the rescue room.
type LordBritishFinaleActor struct {
	PartyIndex  int
	Start       references.Position
	Destination references.Position
	Sprite      indexes.SpriteIndex
	Revived     bool
}

// LordBritishFinaleReview is a detached presentation plan for the verified rescue handoff.
type LordBritishFinaleReview struct {
	Map                 minimap.MiscMap
	LordBritishStart    references.Position
	LordBritishEnd      references.Position
	BoxMoongatePosition references.Position
	Party               []LordBritishFinaleActor
	AvatarName          string
}

// LordBritishFinaleAnswerResult reports whether one raw key was accepted and what it resolved.
type LordBritishFinaleAnswerResult struct {
	Accepted   bool
	Prompt     LordBritishFinalePrompt
	Outcome    LordBritishFinaleOutcome
	MissingBox bool
}

// LordBritishFinaleState retains authority while GameScene owns presentation timing.
type LordBritishFinaleState struct {
	rescueMap      minimap.MiscMap
	prompt         LordBritishFinalePrompt
	outcome        LordBritishFinaleOutcome
	completionDate *datetime.BritannianDate
}

var lordBritishFinalePartyDestinations = [...]references.Position{ //nolint:gochecknoglobals // Immutable original finale rules table.
	// See ref:DNG-0015.
	{X: 5, Y: 5}, {X: 4, Y: 6}, {X: 6, Y: 6},
	{X: 3, Y: 7}, {X: 5, Y: 7}, {X: 7, Y: 7},
}

// HasPendingLordBritishFinale reports whether verified combat has handed off exactly once.
func (g *GameState) HasPendingLordBritishFinale() bool {
	return g.finaleHandoffPending
}

// BeginLordBritishFinale consumes only the verified Doom mirror handoff and installs
// original mini-map definition's fourth shared mini-map record as authoritative terrain.
// See ref:DNG-0015.
func (g *GameState) BeginLordBritishFinale() (LordBritishFinaleReview, error) {
	if !g.finaleHandoffPending || g.lordBritishFinale != nil || g.CombatScene != nil {
		return LordBritishFinaleReview{}, errLordBritishFinaleInvalid
	}
	decoded, err := minimap.LoadMiscMap(g.GameConfig.SavedConfigData.DataFilePath, lordBritishFinaleMapIndex)
	if err != nil {
		return LordBritishFinaleReview{}, fmt.Errorf("loading Lord British rescue map: %w", err)
	}

	return g.beginLordBritishFinaleWithMap(decoded)
}

func (g *GameState) beginLordBritishFinaleWithMap(decoded minimap.MiscMap) (LordBritishFinaleReview, error) {
	if !g.finaleHandoffPending || g.lordBritishFinale != nil || g.CombatScene != nil {
		return LordBritishFinaleReview{}, errLordBritishFinaleInvalid
	}
	activeCount := g.PartyState.GetActivePartyCount()
	if activeCount < 1 || activeCount > len(lordBritishFinalePartyDestinations) {
		return LordBritishFinaleReview{}, fmt.Errorf("%w: active party count %d", errLordBritishFinaleInvalid, activeCount)
	}

	var tiles [references.XCombatMapTiles][references.YCombatMapTiles]byte
	for x := range minimap.MiscMapSize {
		for y := range minimap.MiscMapSize {
			tiles[x][y] = decoded.Tiles[x][y]
		}
	}
	g.MapState.LayeredMaps.ResetAndCreateMiniMapFromData(
		&tiles,
		g.GameReferences.TileReferences,
		g.MapState.XTilesVisibleOnGameScreen,
		g.MapState.YTilesVisibleOnGameScreen,
	)
	g.MapState.PlayerLocation = references.PlayerLocation{
		Location: references.FinaleLocation,
		Floor:    0,
		Position: references.Position{X: 5, Y: 5},
	}
	g.CurrentNPCAIController = nil
	g.finaleHandoffPending = false
	g.lordBritishFinale = &LordBritishFinaleState{
		rescueMap:      decoded,
		prompt:         LordBritishFinaleBoxPrompt,
		outcome:        LordBritishFinaleInProgress,
		completionDate: nil,
	}

	review := LordBritishFinaleReview{
		Map:                 decoded,
		LordBritishStart:    references.Position{X: 5, Y: 8},
		LordBritishEnd:      references.Position{X: 5, Y: 3},
		BoxMoongatePosition: references.Position{X: 5, Y: 4},
		Party:               make([]LordBritishFinaleActor, 0, activeCount),
		AvatarName:          g.PartyState.GetAvatar().GetNameAsString(),
	}
	for index := range activeCount {
		character := g.PartyState.GetCharacter(index)
		revived := character.GetStatus() == party_state.Dead
		if revived {
			character.SetStatus(party_state.Good)
			character.SetHp(character.GetMaxHp())
		}
		sprite := character.GetKeySpriteIndex()
		if sprite == 0 {
			sprite = indexes.AvatarKeyIndex
		}
		review.Party = append(review.Party, LordBritishFinaleActor{
			PartyIndex:  index,
			Start:       references.Position{X: 5, Y: 9},
			Destination: lordBritishFinalePartyDestinations[index],
			Sprite:      sprite,
			Revived:     revived,
		})
	}

	return review, nil
}

// SubmitLordBritishFinaleAnswer applies original finale rules's exact uppercase Y/N loop and
// checks actual Wooden Box ownership only when the final accepted answer is Y.
// See ref:DNG-0015.
func (g *GameState) SubmitLordBritishFinaleAnswer(answer byte) LordBritishFinaleAnswerResult {
	state := g.lordBritishFinale
	if state == nil || state.outcome != LordBritishFinaleInProgress || state.prompt == LordBritishFinaleNoPrompt {
		return LordBritishFinaleAnswerResult{}
	}
	if answer != 'Y' && answer != 'N' {
		return LordBritishFinaleAnswerResult{Prompt: state.prompt, Outcome: state.outcome}
	}
	if state.prompt == LordBritishFinaleBoxPrompt && answer == 'N' {
		state.prompt = LordBritishFinaleSandalwoodPrompt

		return LordBritishFinaleAnswerResult{Accepted: true, Prompt: state.prompt, Outcome: state.outcome}
	}

	state.prompt = LordBritishFinaleNoPrompt
	hasBox := g.PartyState.Inventory.QuestItems.HasSome(references.WoodenBox)
	if answer == 'Y' && hasBox {
		state.outcome = LordBritishFinaleSucceeded
		date := g.DateTime.DateOnly()
		state.completionDate = &date
	} else {
		state.outcome = LordBritishFinaleFailed
	}

	return LordBritishFinaleAnswerResult{
		Accepted: true, Prompt: state.prompt, Outcome: state.outcome,
		MissingBox: answer == 'Y' && !hasBox,
	}
}

// LordBritishFinaleOutcome returns the current authoritative outcome.
func (g *GameState) LordBritishFinaleOutcome() LordBritishFinaleOutcome {
	if g.lordBritishFinale == nil {
		return LordBritishFinaleInProgress
	}

	return g.lordBritishFinale.outcome
}

// LordBritishFinaleCompletionDate returns the recorded success date, if any.
func (g *GameState) LordBritishFinaleCompletionDate() (datetime.BritannianDate, bool) {
	if g.lordBritishFinale == nil || g.lordBritishFinale.completionDate == nil {
		return datetime.BritannianDate{}, false
	}

	return *g.lordBritishFinale.completionDate, true
}

// IsGameplayTerminal reports the two original non-returning finale branches.
func (g *GameState) IsGameplayTerminal() bool {
	return g.lordBritishFinale != nil && g.lordBritishFinale.outcome != LordBritishFinaleInProgress
}

// CanAcceptGameplayCommands excludes both the rescue workflow and its terminal outcomes.
func (g *GameState) CanAcceptGameplayCommands() bool {
	return !g.finaleHandoffPending && g.lordBritishFinale == nil
}

// LordBritishFinaleTile returns source terrain for presentation-owned walking audio.
func (g *GameState) LordBritishFinaleTile(position references.Position) (indexes.SpriteIndex, bool) {
	if g.lordBritishFinale == nil || int(position.X) < 0 || int(position.X) >= minimap.MiscMapSize ||
		int(position.Y) < 0 || int(position.Y) >= minimap.MiscMapSize {
		return 0, false
	}

	return indexes.SpriteIndex(g.lordBritishFinale.rescueMap.Tiles[position.X][position.Y]), true
}

// SetLordBritishFinaleTile applies a source-scripted rescue-room terrain change.
func (g *GameState) SetLordBritishFinaleTile(position references.Position, tile indexes.SpriteIndex) bool {
	if g.lordBritishFinale == nil || int(position.X) < 0 || int(position.X) >= minimap.MiscMapSize ||
		int(position.Y) < 0 || int(position.Y) >= minimap.MiscMapSize {
		return false
	}
	g.lordBritishFinale.rescueMap.Tiles[position.X][position.Y] = byte(tile)
	g.GetLayeredMapByCurrentLocation().SetTileByLayer(map_state.MapLayer, &position, tile)

	return true
}
