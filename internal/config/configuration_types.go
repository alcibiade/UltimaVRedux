package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

var errConfigurationType = errors.New("invalid preference type")

// Validate known preferences before Viper's permissive getters can silently
// replace malformed values with defaults. Unknown keys remain inert and intact.
func validateConfigurationTypes(source *viper.Viper) error {
	if err := validateConfigurationSections(source); err != nil {
		return err
	}
	groups := []struct {
		name  string
		keys  []string
		valid func(any) bool
	}{
		{"string", []string{"DataFilePath", "Display.Mode", "Logging.MinLevel", "Debug.SurfaceMovementCurve"},
			func(value any) bool {
				_, ok := value.(string)

				return ok
			}},
		{"integer", []string{"Resolution", "Audio.MusicVolume", "Audio.SoundEffectVolume", "Audio.AmbientVolume",
			"Display.WindowWidth", "Display.WindowHeight", "Display.TargetFPS", "Lighting.AmbientLightRadius",
			"Debug.SurfaceMovementDurationMs"},
			func(value any) bool {
				_, ok := value.(int)

				return ok
			}},
		{"boolean", []string{"FullScreen", "Display.EnableFpsLimiting", "Logging.Enabled", "Logging.StdoutEnabled",
			"Debug.CombatDetailShown", "Gameplay.Combat.AllowRepeatedMonsterDivision", gameplayBatchMerchantTheftKey,
			gameplayShowConcealedTurnOrderSlotsKey, gameplayPersistHiddenDungeonDiscoveriesKey,
			gameplayShowSubduedUnlitDungeonsKey, gameplaySmoothSurfaceMovementKey, gameplayFlexibleSextantUseKey},
			func(value any) bool {
				_, ok := value.(bool)

				return ok
			}},
		{"number", []string{"Debug.CombatMovementSpeedFactor"}, func(value any) bool {
			switch value.(type) {
			case int, float64:
				return true
			default:
				return false
			}
		}},
	}
	for _, group := range groups {
		for _, key := range group.keys {
			if source.IsSet(key) && !group.valid(source.Get(key)) {
				return fmt.Errorf("decoding configuration: %w: %s must be a %s", errConfigurationType, key, group.name)
			}
		}
	}

	return nil
}

func validateConfigurationSections(source *viper.Viper) error {
	for _, key := range []string{"Audio", "Display", "Gameplay", "Gameplay.Merchants", "Gameplay.Combat",
		"Gameplay.Dungeons", "Gameplay.Presentation", "Gameplay.Navigation", "Logging", "Logging.CategoryStates",
		"Lighting", "Debug"} {
		if source.IsSet(key) {
			if _, ok := source.Get(key).(map[string]any); !ok {
				return fmt.Errorf("decoding configuration: %w: %s must be a mapping", errConfigurationType, key)
			}
		}
	}
	for category, value := range source.GetStringMap("Logging.CategoryStates") {
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("decoding configuration: %w: Logging.CategoryStates.%s must be a boolean",
				errConfigurationType, category)
		}
	}

	return nil
}
