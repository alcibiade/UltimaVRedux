package config

import (
	"errors"
	"fmt"
	"math"

	"github.com/spf13/viper"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

var (
	// ErrGameplayPreference reports a malformed numeric preference or specification.
	ErrGameplayPreference = errors.New("invalid gameplay preference")
	// ErrConfigurationEditing rejects an immediate write over a staged player transaction.
	ErrConfigurationEditing = errors.New("close Configuration with Apply or Cancel before changing persistent preferences")
)

const (
	gameplayCombatSpeedKey      = "Gameplay.Combat.SpeedFactor"
	gameplayConfirmQuickSaveKey = "Gameplay.Saves.ConfirmQuickSave"
	gameplayShowCombatHPBarsKey = "Gameplay.Combat.ShowHealthBars"
	gameplayMaxActionPopupsKey  = "Gameplay.Presentation.MaxVisibleActionPopups"
	legacyCombatSpeedKey        = "Debug.CombatMovementSpeedFactor"
	numericGridTolerance        = 1e-9 // In step-index units; half-steps remain exactly representable.
)

// NumericSettingSpec defines a bounded discrete preference. Values are derived
// from integer step indices rather than repeated floating-point addition.
type NumericSettingSpec struct {
	Min, Max, Step, Default float64
}

// CombatSpeedSpec returns the single specification used by persistence and UI.
func CombatSpeedSpec() NumericSettingSpec {
	return NumericSettingSpec{Min: 0.5, Max: 4, Step: 0.5, Default: 1} //nolint:mnd // Approved player preference domain.
}

// ActionPopupsSpec returns the integer popup preference domain; zero means Off.
func ActionPopupsSpec() NumericSettingSpec {
	return NumericSettingSpec{Min: 0, Max: 3, Step: 1, Default: 3} //nolint:mnd // Approved player preference domain.
}

// Validate checks specification invariants and finite, bounded grid membership.
func (s NumericSettingSpec) Validate(value float64) error {
	for _, number := range []float64{s.Min, s.Max, s.Step, s.Default, value} {
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return fmt.Errorf("%w: numbers must be finite", ErrGameplayPreference)
		}
	}
	if s.Step <= 0 || s.Max < s.Min || !s.onGrid(s.Max) || !s.onGrid(s.Default) {
		return fmt.Errorf("%w: invalid numeric specification", ErrGameplayPreference)
	}
	if !s.onGrid(value) {
		return fmt.Errorf("%w: %g must be %g..%g in steps of %g", ErrGameplayPreference, value, s.Min, s.Max, s.Step)
	}

	return nil
}

func (s NumericSettingSpec) onGrid(value float64) bool {
	index := (value - s.Min) / s.Step

	return value >= s.Min && value <= s.Max && math.Abs(index-math.Round(index)) <= numericGridTolerance
}

// Advance moves by whole steps, clamping at either boundary.
func (s NumericSettingSpec) Advance(value float64, steps int) float64 {
	if err := s.Validate(value); err != nil {
		panic(err) // Definitions and staged models are validated before use.
	}
	index := int(math.Round((value - s.Min) / s.Step))
	last := int(math.Round((s.Max - s.Min) / s.Step))
	steps = max(-index, min(last-index, steps))

	return s.Min + float64(index+steps)*s.Step
}

// ValidateGameplayConfiguration rejects invalid explicit numeric settings.
func ValidateGameplayConfiguration(settings GameplayConfiguration) error {
	if err := CombatSpeedSpec().Validate(settings.CombatSpeedFactor); err != nil {
		return fmt.Errorf("%s: %w", gameplayCombatSpeedKey, err)
	}
	if err := ActionPopupsSpec().Validate(float64(settings.MaxVisibleActionPopups)); err != nil {
		return fmt.Errorf("%s: %w", gameplayMaxActionPopupsKey, err)
	}

	return nil
}

// DefaultGameplayConfiguration returns a complete valid model, including numeric
// defaults. These are the only gameplay defaults: loading starts from them and
// applies what the configuration file sets, and detached settings and
// lightweight fixtures use them directly.
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

// loadGameplayPreferences replaces each default with the configured value, if
// the configuration file sets one.
func (uc *UltimaVConfiguration) loadGameplayPreferences() {
	settings := DefaultGameplayConfiguration()
	uc.GameplayConfiguration = settings
	uc.configuredBoolean(&settings.BatchMerchantTheft, gameplayBatchMerchantTheftKey)
	uc.configuredBoolean(&settings.SmoothSurfaceMovement, gameplaySmoothSurfaceMovementKey)
	uc.configuredBoolean(&settings.FlexibleSextantUse, gameplayFlexibleSextantUseKey)
	uc.configuredBoolean(&settings.AllowRepeatedMonsterDivision, "Gameplay.Combat.AllowRepeatedMonsterDivision")
	uc.configuredBoolean(&settings.ShowConcealedTurnOrderSlots, gameplayShowConcealedTurnOrderSlotsKey)
	uc.configuredBoolean(&settings.PersistHiddenDungeonDiscoveries, gameplayPersistHiddenDungeonDiscoveriesKey)
	uc.configuredBoolean(&settings.ShowSubduedUnlitDungeons, gameplayShowSubduedUnlitDungeonsKey)
	uc.configuredBoolean(&settings.ConfirmQuickSave, gameplayConfirmQuickSaveKey)
	uc.configuredBoolean(&settings.ShowCombatHPBars, gameplayShowCombatHPBarsKey)
	if uc.source.IsSet(gameplayCombatSpeedKey) {
		settings.CombatSpeedFactor = uc.source.GetFloat64(gameplayCombatSpeedKey)
	} else if uc.source.IsSet(legacyCombatSpeedKey) {
		legacy := uc.source.GetFloat64(legacyCombatSpeedKey)
		spec := CombatSpeedSpec()
		// All values are positive: math.Round rounds exact midpoints upward.
		settings.CombatSpeedFactor = max(spec.Min, min(spec.Max, math.Round(legacy/spec.Step)*spec.Step))
		if legacy != settings.CombatSpeedFactor {
			message := fmt.Sprintf("At load, normalized legacy combat speed %g to %g.", legacy, settings.CombatSpeedFactor)
			uc.configurationDiagnostics = append(uc.configurationDiagnostics, message)
			logging.Warnf(logging.CategoryConfig, "%s", message)
		}
	}
	if uc.source.IsSet(gameplayMaxActionPopupsKey) {
		settings.MaxVisibleActionPopups = uc.source.GetInt(gameplayMaxActionPopupsKey)
	}
}

// ConfigurationDiagnostics returns detached startup diagnostics even when the
// categorized logger was disabled during loading.
func (uc *UltimaVConfiguration) ConfigurationDiagnostics() []string {
	if uc == nil {
		return nil
	}

	return append([]string(nil), uc.configurationDiagnostics...)
}

// GetCombatSpeedFactor returns the canonical master combat presentation speed.
func (uc *UltimaVConfiguration) GetCombatSpeedFactor() float64 {
	if uc == nil || uc.GameplayConfiguration == nil {
		return CombatSpeedSpec().Default
	}

	return uc.GameplayConfiguration.CombatSpeedFactor
}

// GetConfirmQuickSave defaults to confirmation when a fixture has no settings.
func (uc *UltimaVConfiguration) GetConfirmQuickSave() bool {
	return uc == nil || uc.GameplayConfiguration == nil || uc.GameplayConfiguration.ConfirmQuickSave
}

// GetShowCombatHPBars defaults to visible when a fixture has no settings.
func (uc *UltimaVConfiguration) GetShowCombatHPBars() bool {
	return uc == nil || uc.GameplayConfiguration == nil || uc.GameplayConfiguration.ShowCombatHPBars
}

// GetMaxVisibleActionPopups preserves explicit zero (Off).
func (uc *UltimaVConfiguration) GetMaxVisibleActionPopups() int {
	if uc == nil || uc.GameplayConfiguration == nil {
		return int(ActionPopupsSpec().Default)
	}

	return uc.GameplayConfiguration.MaxVisibleActionPopups
}

// ConfigurationEditing reports a staged player transaction on this model.
func (uc *UltimaVConfiguration) ConfigurationEditing() bool {
	return uc != nil && uc.configurationEditors > 0
}

// BeginConfigurationEditing reserves persistent preference editing until release.
// Dialog owners must call the returned release function exactly once on dismissal.
func (uc *UltimaVConfiguration) BeginConfigurationEditing() func() {
	uc.configurationEditors++
	released := false

	return func() {
		if !released {
			uc.configurationEditors--
			released = true
		}
	}
}

func (uc *UltimaVConfiguration) saveGameplayPreference(update func(*GameplayConfiguration)) error {
	if uc == nil || uc.GameplayConfiguration == nil {
		return ErrGameplayConfigNil
	}
	if uc.ConfigurationEditing() {
		return ErrConfigurationEditing
	}
	staged := *uc.GameplayConfiguration
	update(&staged)
	if err := ValidateGameplayConfiguration(staged); err != nil {
		return err
	}
	if err := uc.saveConfiguration(func(candidate *viper.Viper) {
		setGameplayConfiguration(candidate, staged)
	}); err != nil {
		return fmt.Errorf("saving gameplay preference: %w", err)
	}
	*uc.GameplayConfiguration = staged

	return nil
}

// SetCombatSpeedFactor validates and persists before changing the live model.
func (uc *UltimaVConfiguration) SetCombatSpeedFactor(value float64) error {
	return uc.saveGameplayPreference(func(settings *GameplayConfiguration) { settings.CombatSpeedFactor = value })
}

// SetConfirmQuickSave persists the quick-save confirmation preference atomically.
func (uc *UltimaVConfiguration) SetConfirmQuickSave(value bool) error {
	return uc.saveGameplayPreference(func(settings *GameplayConfiguration) { settings.ConfirmQuickSave = value })
}

// SetShowCombatHPBars persists battlefield health-bar visibility atomically.
func (uc *UltimaVConfiguration) SetShowCombatHPBars(value bool) error {
	return uc.saveGameplayPreference(func(settings *GameplayConfiguration) { settings.ShowCombatHPBars = value })
}

// SetMaxVisibleActionPopups persists the bounded popup count atomically.
func (uc *UltimaVConfiguration) SetMaxVisibleActionPopups(value int) error {
	return uc.saveGameplayPreference(func(settings *GameplayConfiguration) { settings.MaxVisibleActionPopups = value })
}
