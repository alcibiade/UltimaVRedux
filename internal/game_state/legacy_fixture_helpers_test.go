package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// One-file template loading is fixture setup. Production continuation uses the
// paired SAVED.GAM/SAVED.OOL conversion and native snapshot hydration path.
func newGameStateFromLegacyTemplateFile(savedGamFilePath string,
	gameConfig *config.UltimaVConfiguration,
	gameReferences *references.GameReferences,
	xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen int,
) (*GameState, error) {
	rawSaveData, err := ReadLegacySaveFile(savedGamFilePath)
	if err != nil {
		return nil, fmt.Errorf("loading legacy save file %s: %w", savedGamFilePath, err)
	}

	return NewGameStateFromLegacyTemplateBytes(rawSaveData,
		gameConfig, gameReferences, xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
}

// Focused scroll fixtures leave command completion to their test harness.
func (g *GameState) useScroll(scroll references.Scroll) bool {
	return g.useScrollWithCompletion(scroll, nil)
}
