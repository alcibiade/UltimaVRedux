package config

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
