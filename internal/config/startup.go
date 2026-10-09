package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const startupDataOvlSize = 48_464

var errStartupDataOvlSize = errors.New("configured DATA.OVL has an unexpected size")

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
		return loadRecoveryConfiguration(err)
	}

	configuration, err := newUltimaVConfiguration(source, "", false)
	if err != nil {
		return loadRecoveryConfiguration(err)
	}
	dataErr := validateStartupDataFile(configuration.SavedConfigData.DataFilePath)

	return StartupConfiguration{Configuration: configuration, ConfigurationError: nil, OriginalDataError: dataErr}
}

func validateStartupDataFile(directory string) error {
	path := filepath.Join(directory, "DATA.OVL")
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("checking configured DATA.OVL: %w", err)
	}
	if info.Size() != startupDataOvlSize {
		return fmt.Errorf("%w: got %d, expected %d", errStartupDataOvlSize, info.Size(), startupDataOvlSize)
	}

	return nil
}

func loadRecoveryConfiguration(configurationErr error) StartupConfiguration {
	recoverySource, recoveryErr := defaultRecoveryConfigurationSource()
	if recoveryErr != nil {
		return StartupConfiguration{
			Configuration: nil, ConfigurationError: errors.Join(configurationErr, recoveryErr), OriginalDataError: nil,
		}
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
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("locating configuration home: %w", err)
	}
	source := viper.New()
	source.SetConfigType("yaml")
	source.SetConfigFile(filepath.Join(home, ".ultima_v_redux", "config.yaml"))
	source.Set("DataFilePath", "./OLD")

	return source, nil
}
