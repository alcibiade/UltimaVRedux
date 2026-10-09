package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

const defaultSnapshotProjectDirectory = "debug/snapshots"

type snapshotStartupOptions struct {
	projectRoot  string
	personalRoot string
	rollbackRoot string
}

func addSnapshotFlags(options *snapshotStartupOptions) {
	flag.StringVar(&options.projectRoot, "snapshot-project-dir", defaultSnapshotProjectDirectory, "project named-snapshot directory")
	flag.StringVar(&options.personalRoot, "snapshot-personal-dir", "", "personal named-snapshot directory (default ~/.ultima_v_redux/snapshots)")
	flag.StringVar(&options.rollbackRoot, "snapshot-rollback-dir", "", "automatic rollback directory (default ~/.ultima_v_redux/snapshots/rollback)")
}

func configureSnapshotRepository(game *Game, options snapshotStartupOptions) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("locating personal snapshot directory: %w", err)
	}
	if options.personalRoot == "" {
		options.personalRoot = filepath.Join(home, ".ultima_v_redux", "snapshots")
	}
	if options.projectRoot == "" {
		options.projectRoot = defaultSnapshotProjectDirectory
	}
	if options.rollbackRoot == "" {
		options.rollbackRoot = filepath.Join(home, ".ultima_v_redux", "snapshots", "rollback")
	}
	repository, err := save_snapshot.NewRepository(options.projectRoot, options.personalRoot, options.rollbackRoot)
	if err != nil {
		return fmt.Errorf("configuring save snapshot repository: %w", err)
	}
	game.ConfigureSnapshotRepository(repository, func(configuration *config.UltimaVConfiguration, snapshot save_snapshot.SaveSnapshot) (Scene, error) {
		return NewGameSceneFromSaveSnapshot(configuration, snapshot)
	})

	return nil
}
