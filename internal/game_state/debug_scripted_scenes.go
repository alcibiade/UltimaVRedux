package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/minimap"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const debugBlackthornShrineIndex = 0

// BlackthornInterrogationSetupResult reports the authoritative state prepared
// before the production interrogation workflow begins.
type BlackthornInterrogationSetupResult struct {
	WorldSetupResult
	ShrineIndex        int
	ShrineRuinedBefore bool
	ShrineRuinedAfter  bool
}

// LordBritishFinaleSetupResult reports the authoritative state installed while
// beginning the production finale workflow.
type LordBritishFinaleSetupResult struct {
	WorldSetupResult
	WoodenBoxBefore uint16
	WoodenBoxAfter  uint16
}

// DebugBeginBlackthornInterrogation prepares a deterministic Palace arrest
// context and begins the same authoritative session used by normal gameplay.
func (g *GameState) DebugBeginBlackthornInterrogation() (BlackthornInterrogationSetupResult, BlackthornInterrogationReview, error) {
	result := BlackthornInterrogationSetupResult{
		WorldSetupResult:   g.currentWorldSetupResult(),
		ShrineIndex:        debugBlackthornShrineIndex,
		ShrineRuinedBefore: g.IsShrineRuined(debugBlackthornShrineIndex),
	}
	if err := g.validateWorldSetupContext(); err != nil {
		return result, BlackthornInterrogationReview{}, err
	}
	if g.blackthornSession != nil || g.PartyState.GetLivingPartyCount() == 0 {
		return result, BlackthornInterrogationReview{}, fmt.Errorf("%w: Blackthorn interrogation requires a living party and no active session", ErrInvalidWorldSetup)
	}
	dataPath, err := g.debugScriptedSceneDataPath()
	if err != nil {
		return result, BlackthornInterrogationReview{}, err
	}
	decoded, err := minimap.LoadMiscMap(dataPath, blackthornPrisonMapIndex)
	if err != nil {
		return result, BlackthornInterrogationReview{}, fmt.Errorf("%w: validating Blackthorn prison map: %w", ErrInvalidWorldSetup, err)
	}

	g.TransitionToLocation(
		references.PalaceOfBlackthorn,
		references.Basement,
		references.Position{X: 10, Y: 7},
		TransitionFullLocation,
		false,
	)
	g.SetShrineRuined(debugBlackthornShrineIndex, false)
	result.ShrineRuinedAfter = g.IsShrineRuined(debugBlackthornShrineIndex)
	review, err := g.beginBlackthornInterrogation(&decoded)
	if err != nil {
		panic("DebugBeginBlackthornInterrogation: validated setup could not begin: " + err.Error())
	}
	result.After = g.MapState.PlayerLocation
	result.SmashedAfter = g.smashed

	return result, review, nil
}

// DebugBeginLordBritishFinale validates source data, chooses exact Wooden Box
// ownership, and begins the same authoritative finale used by Doom combat.
func (g *GameState) DebugBeginLordBritishFinale(
	woodenBox bool,
) (LordBritishFinaleSetupResult, LordBritishFinaleReview, error) {
	boxBefore := g.PartyState.Inventory.QuestItems.Get(references.WoodenBox)
	result := LordBritishFinaleSetupResult{
		WorldSetupResult: g.currentWorldSetupResult(),
		WoodenBoxBefore:  boxBefore,
		WoodenBoxAfter:   boxBefore,
	}
	if err := g.validateWorldSetupContext(); err != nil {
		return result, LordBritishFinaleReview{}, err
	}
	if g.finaleHandoffPending || g.lordBritishFinale != nil {
		return result, LordBritishFinaleReview{}, fmt.Errorf("%w: Lord British finale is already active", ErrInvalidWorldSetup)
	}
	activeCount := g.PartyState.GetActivePartyCount()
	if activeCount < 1 || activeCount > party_state.MaxActiveParty {
		return result, LordBritishFinaleReview{}, fmt.Errorf("%w: Lord British finale requires an active party", ErrInvalidWorldSetup)
	}
	dataPath, err := g.debugScriptedSceneDataPath()
	if err != nil {
		return result, LordBritishFinaleReview{}, err
	}
	decoded, err := minimap.LoadMiscMap(dataPath, lordBritishFinaleMapIndex)
	if err != nil {
		return result, LordBritishFinaleReview{}, fmt.Errorf("%w: validating Lord British rescue map: %w", ErrInvalidWorldSetup, err)
	}

	boxAfter := uint16(0)
	if woodenBox {
		boxAfter = 1
	}
	g.PartyState.Inventory.QuestItems.Set(references.WoodenBox, boxAfter)
	g.finaleHandoffPending = true
	review, err := g.beginLordBritishFinaleWithMap(decoded)
	if err != nil {
		g.finaleHandoffPending = false
		g.PartyState.Inventory.QuestItems.Set(references.WoodenBox, boxBefore)

		return result, LordBritishFinaleReview{}, err
	}
	result.After = g.MapState.PlayerLocation
	result.SmashedAfter = g.smashed
	result.WoodenBoxAfter = boxAfter

	return result, review, nil
}

func (g *GameState) debugScriptedSceneDataPath() (string, error) {
	if g.GameConfig == nil || g.GameConfig.SavedConfigData == nil || g.GameConfig.SavedConfigData.DataFilePath == "" {
		return "", fmt.Errorf("%w: original game data path is unavailable", ErrInvalidWorldSetup)
	}

	return g.GameConfig.SavedConfigData.DataFilePath, nil
}
