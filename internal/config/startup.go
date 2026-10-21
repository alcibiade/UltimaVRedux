package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/bradhannah/UltimaVRedux/internal/files"
)

const startupDataOvlSize = 48_464

var (
	errStartupDataOvlSize = errors.New("configured DATA.OVL has an unexpected size")
	errHomeNotAbsolute    = errors.New(
		"set an absolute home folder before starting Redux; settings cannot use the working directory",
	)
	errHomeNotDirectory = errors.New("the configured home path must be a folder before starting Redux")
)

// StartupConfiguration contains enough configuration to initialize the window
// even when the selected original game data or configuration file is unusable.
type StartupConfiguration struct {
	Configuration      *UltimaVConfiguration
	ConfigurationError error
	OriginalDataError  error
}

// LoadStartupConfiguration loads player preferences independently from original
// game data. Strict constructors remain available to callers that require a
// fully playable configuration.
func LoadStartupConfiguration() StartupConfiguration {
	source, err := loadUserConfigurationSource()
	if err != nil {
		return loadRecoveryConfiguration(err, source)
	}

	configuration, err := newUltimaVConfiguration(source, "", false)
	if err != nil {
		return loadRecoveryConfiguration(err, source)
	}
	dataErr := validateStartupDataFile(configuration.SavedConfigData.DataFilePath)

	return StartupConfiguration{Configuration: configuration, ConfigurationError: nil, OriginalDataError: dataErr}
}

func validateStartupDataFile(directory string) error {
	if strings.TrimSpace(directory) == "" {
		return ErrDataFilePathUnset
	}
	path := files.Resolve(directory, files.DataOvl)
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("checking configured DATA.OVL: %w", err)
	}
	if info.Size() != startupDataOvlSize {
		return fmt.Errorf("%w: got %d, expected %d", errStartupDataOvlSize, info.Size(), startupDataOvlSize)
	}

	return nil
}

func loadRecoveryConfiguration(configurationErr error, selected *viper.Viper) StartupConfiguration {
	recoverySource, recoveryErr := defaultRecoveryConfigurationSource()
	if recoveryErr != nil {
		return StartupConfiguration{
			Configuration: nil, ConfigurationError: errors.Join(configurationErr, recoveryErr), OriginalDataError: nil,
		}
	}
	if selected != nil && selected.ConfigFileUsed() != "" {
		recoverySource.SetConfigFile(selected.ConfigFileUsed())
	}
	configuration, decodeErr := newUltimaVConfiguration(recoverySource, "", false)
	if decodeErr != nil {
		return StartupConfiguration{
			Configuration: nil, ConfigurationError: errors.Join(configurationErr, decodeErr), OriginalDataError: nil,
		}
	}

	return StartupConfiguration{
		Configuration: configuration, ConfigurationError: configurationErr, OriginalDataError: nil,
	}
}

func defaultRecoveryConfigurationSource() (*viper.Viper, error) {
	home, err := configurationHome()
	if err != nil {
		return nil, fmt.Errorf("locating configuration home: %w", err)
	}
	source := viper.New()
	source.SetConfigType("yaml")
	source.SetConfigFile(filepath.Join(home, ".ultima_v_redux", "config.yaml"))

	return source, nil
}

func configurationHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve your home folder before starting Redux: %w", err)
	}
	if strings.TrimSpace(home) == "" || !filepath.IsAbs(home) {
		return "", errHomeNotAbsolute
	}
	info, err := os.Stat(home)
	if err != nil {
		return "", fmt.Errorf("make your home folder available before starting Redux: %w", err)
	}
	if !info.IsDir() {
		return "", errHomeNotDirectory
	}

	return home, nil
}
