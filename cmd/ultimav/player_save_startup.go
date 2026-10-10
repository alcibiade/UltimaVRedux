package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/doom_recovery"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

type playerSaveStartupOptions struct {
	root string
}

func addPlayerSaveFlags(options *playerSaveStartupOptions) {
	flag.StringVar(&options.root, "player-save-dir", "", "player save directory (default ~/.ultima_v_redux/saves)")
}

func configurePlayerSaveRepository(game *Game, options playerSaveStartupOptions) error {
	if game == nil || game.currentScene == nil {
		return fmt.Errorf(
			"configuring player save repository: %w: game and current scene are required",
			ErrPlayerSaveUnavailable,
		)
	}
	if options.root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("locating player save directory: %w", err)
		}
		options.root = filepath.Join(home, ".ultima_v_redux", "saves")
	}

	configuration := game.currentScene.GetUltimaConfiguration()
	game.notebookRoot = filepath.Join(options.root, "excerpts")
	var (
		once           sync.Once
		gameReferences *references.GameReferences
		referencesErr  error
	)
	validateCompatibility := func(snapshot save_snapshot.SaveSnapshot) error {
		once.Do(func() {
			if configuration == nil {
				referencesErr = fmt.Errorf(
					"%w: game configuration is required for compatibility validation",
					ErrPlayerSaveUnavailable,
				)

				return
			}
			gameReferences, referencesErr = references.NewGameReferences(configuration)
		})
		if referencesErr != nil {
			return fmt.Errorf("loading runtime references for player saves: %w", referencesErr)
		}

		return game_state.ValidateSaveSnapshotCompatibility(snapshot, gameReferences)
	}
	repository, err := player_save.NewRepository(options.root, validateCompatibility)
	if err != nil {
		return fmt.Errorf("configuring player save repository: %w", err)
	}
	game.ConfigurePlayerSaveRepository(
		repository,
		func(configuration *config.UltimaVConfiguration, snapshot save_snapshot.SaveSnapshot) (Scene, error) {
			return NewGameSceneFromSaveSnapshot(configuration, snapshot)
		},
	)
	recoveryRepository, err := doom_recovery.NewRepository(options.root, validateCompatibility)
	if err != nil {
		return fmt.Errorf("configuring pre-Doom recovery repository: %w", err)
	}
	game.ConfigureDoomRecoveryRepository(
		recoveryRepository,
		func(configuration *config.UltimaVConfiguration, snapshot save_snapshot.SaveSnapshot) (Scene, error) {
			return NewGameSceneFromSaveSnapshot(configuration, snapshot)
		},
	)

	return nil
}

func runtimeGameVersion() string {
	if buildInfo, ok := debug.ReadBuildInfo(); ok && buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
		return buildInfo.Main.Version
	}

	return developmentGameVersion
}
