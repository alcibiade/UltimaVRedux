package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Test support: fixtures and seams that tests in other packages use. The
// dead-code check treats this file as test support (workshop deadcode config).

// DefaultGameplayConfiguration returns a complete valid model, including numeric
// defaults. Use this constructor for detached settings and lightweight fixtures.
func DefaultGameplayConfiguration() *GameplayConfiguration {
	return &GameplayConfiguration{
		BatchMerchantTheft:              true,
		AllowRepeatedMonsterDivision:    false,
		ShowConcealedTurnOrderSlots:     true,
		PersistHiddenDungeonDiscoveries: true,
		ShowSubduedUnlitDungeons:        true,
		SmoothSurfaceMovement:           false,
		FlexibleSextantUse:              false,
		CombatSpeedFactor:               CombatSpeedSpec().Default,
		ConfirmQuickSave:                true,
		ShowCombatHPBars:                true,
		MaxVisibleActionPopups:          int(ActionPopupsSpec().Default),
	}
}

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
