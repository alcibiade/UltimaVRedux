package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Test support: fixtures and seams that tests in other packages use. The
// dead-code check treats this file as test support (workshop deadcode config).

// SaveGameplayConfiguration immediately persists current gameplay settings.
func (uc *UltimaVConfiguration) SaveGameplayConfiguration() error {
	if uc.ConfigurationEditing() {
		return ErrConfigurationEditing
	}
	if uc.GameplayConfiguration == nil {
		return ErrGameplayConfigNil
	}
	if err := uc.saveConfiguration(func(candidate *viper.Viper) {
		setGameplayConfiguration(candidate, *uc.GameplayConfiguration)
	}); err != nil {
		return fmt.Errorf("saving gameplay configuration: %w", err)
	}

	return nil
}
