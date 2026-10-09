package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const lastLegacyImportDirectoryKey = "LegacyImport.LastSuccessfulDirectory"

var (
	errDataDirectoryNotAbsolute   = errors.New("data directory must be an absolute path")
	errLegacyDirectoryNotAbsolute = errors.New("legacy import directory must be an absolute path")
)

// ConfigurationFilePath returns the exact absolute settings file selected at startup.
func (uc *UltimaVConfiguration) ConfigurationFilePath() (string, error) {
	if uc == nil || uc.source == nil || uc.source.ConfigFileUsed() == "" {
		return "", ErrConfigFileUnknown
	}
	path, err := filepath.Abs(uc.source.ConfigFileUsed())
	if err != nil {
		return "", fmt.Errorf("resolving configuration path: %w", err)
	}

	return path, nil
}

// PersistedDataFilePath returns the next-launch data path without changing active assets.
func (uc *UltimaVConfiguration) PersistedDataFilePath() string {
	if uc == nil || uc.source == nil {
		return ""
	}

	return uc.source.GetString("DataFilePath")
}

// AdoptPersistedConfiguration shares another configuration's already-committed
// persistence source without changing this configuration's active data path.
func (uc *UltimaVConfiguration) AdoptPersistedConfiguration(committed *UltimaVConfiguration) {
	if uc == nil || committed == nil || committed.source == nil {
		panic("AdoptPersistedConfiguration: committed configuration source is required")
	}
	uc.source = committed.source
}

// RefreshPersistedConfiguration rereads only the persistence source. It leaves
// active runtime preferences, assets, and original data unchanged.
func (uc *UltimaVConfiguration) RefreshPersistedConfiguration() error {
	path, err := uc.ConfigurationFilePath()
	if err != nil {
		return err
	}
	refreshed := viper.New()
	refreshed.SetConfigType("yaml")
	refreshed.SetConfigFile(path)
	if err := refreshed.ReadInConfig(); err != nil {
		return fmt.Errorf("rereading configuration %s: %w", path, err)
	}
	uc.source = refreshed

	return nil
}

// PersistDataFilePath deliberately stores a validated absolute directory for
// the next launch. It does not mutate the current runtime data path or caches.
func (uc *UltimaVConfiguration) PersistDataFilePath(directory string) error {
	directory = filepath.Clean(strings.TrimSpace(directory))
	if directory == "." || directory == "" || !filepath.IsAbs(directory) {
		return errDataDirectoryNotAbsolute
	}
	if err := uc.saveConfiguration(func(candidate *viper.Viper) {
		candidate.Set("DataFilePath", directory)
	}); err != nil {
		return fmt.Errorf("saving data directory: %w", err)
	}

	return nil
}

// LastLegacyImportDirectory returns the last source directory committed by a
// successful legacy import.
func (uc *UltimaVConfiguration) LastLegacyImportDirectory() string {
	if uc == nil || uc.source == nil {
		return ""
	}

	return uc.source.GetString(lastLegacyImportDirectoryKey)
}

// PersistLastLegacyImportDirectory records successful import history separately
// from the selected original-data installation.
func (uc *UltimaVConfiguration) PersistLastLegacyImportDirectory(directory string) error {
	directory = filepath.Clean(strings.TrimSpace(directory))
	if directory == "." || directory == "" || !filepath.IsAbs(directory) {
		return errLegacyDirectoryNotAbsolute
	}
	if err := uc.saveConfiguration(func(candidate *viper.Viper) {
		candidate.Set(lastLegacyImportDirectoryKey, directory)
	}); err != nil {
		return fmt.Errorf("saving legacy import directory: %w", err)
	}

	return nil
}
