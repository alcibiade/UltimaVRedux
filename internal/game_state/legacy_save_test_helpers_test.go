package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func mustNewGameStateFromLegacySaveBytes(
	rawSaveData []byte,
	gameConfig *config.UltimaVConfiguration,
	gameReferences *references.GameReferences,
	xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen int,
) *GameState {
	gameState, err := NewGameStateFromLegacyTemplateBytes(
		rawSaveData,
		gameConfig,
		gameReferences,
		xTilesVisibleOnGameScreen,
		yTilesVisibleOnGameScreen,
	)
	if err != nil {
		panic(err)
	}

	return gameState
}
