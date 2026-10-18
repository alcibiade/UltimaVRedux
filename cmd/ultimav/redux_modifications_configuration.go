package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/debugging"
)

const (
	repeatedMonsterDivisionChangePath         = "gameplay.redux.combat.allowRepeatedMonsterDivision"
	persistHiddenDungeonDiscoveriesChangePath = "gameplay.redux.dungeons.persistHiddenDungeonDiscoveries"
	showSubduedUnlitDungeonsChangePath        = "gameplay.redux.dungeons.showSubduedUnlitDungeons"
	smoothSurfaceMovementChangePath           = "gameplay.redux.presentation.smoothSurfaceMovement"
	flexibleSextantUseChangePath              = "gameplay.redux.navigation.flexibleSextantUse"
)

func executeReduxModificationsDebugCommand(
	request debugging.Request,
	configuration *config.UltimaVConfiguration,
) (debugging.Result, bool) {
	if configuration.ConfigurationEditing() && request.CommandID != debugging.CommandReduxModificationsStatus &&
		(strings.HasPrefix(request.CommandID, "redux.") || request.CommandID == debugging.CommandCombatPresentationSpeedSet) {
		return debugging.FailureResult(request, "configuration_editing", config.ErrConfigurationEditing.Error()), true
	}
	if result, handled := executeQoLPreference(request, configuration); handled {
		return result, true
	}
	if request.CommandID == debugging.CommandReduxModificationsStatus {
		return debugSuccessResult(
			request,
			"Redux gameplay modifications loaded.",
			newReduxModificationsStatusData(configuration),
		), true
	}

	return executeReduxBooleanPreference(request, configuration)
}

func executeReduxBooleanPreference(
	request debugging.Request,
	configuration *config.UltimaVConfiguration,
) (debugging.Result, bool) {
	switch request.CommandID {
	case debugging.CommandReduxCombatRepeatedMonsterDivisionSet:
		return persistReduxBooleanPreference(request, configuration,
			requireGameplayConfiguration(configuration).AllowRepeatedMonsterDivision,
			configuration.SetAllowRepeatedMonsterDivision, repeatedMonsterDivisionChangePath,
			"Repeated monster division preference updated."), true
	case debugging.CommandReduxDungeonsPersistHiddenDiscoveriesSet:
		return persistReduxBooleanPreference(request, configuration,
			requireGameplayConfiguration(configuration).PersistHiddenDungeonDiscoveries,
			configuration.SetPersistHiddenDungeonDiscoveries, persistHiddenDungeonDiscoveriesChangePath,
			"Persistent dungeon discovery preference updated."), true
	case debugging.CommandReduxDungeonsSubduedUnlitVisibilitySet:
		return persistReduxBooleanPreference(request, configuration,
			requireGameplayConfiguration(configuration).ShowSubduedUnlitDungeons,
			configuration.SetShowSubduedUnlitDungeons, showSubduedUnlitDungeonsChangePath,
			"Subdued unlit dungeon visibility preference updated."), true
	case debugging.CommandReduxPresentationSmoothSurfaceMovementSet:
		return persistReduxBooleanPreference(request, configuration,
			requireGameplayConfiguration(configuration).SmoothSurfaceMovement,
			configuration.SetSmoothSurfaceMovement, smoothSurfaceMovementChangePath,
			"Smooth surface movement preference updated."), true
	case debugging.CommandReduxNavigationFlexibleSextantUseSet:
		return persistReduxBooleanPreference(request, configuration,
			requireGameplayConfiguration(configuration).FlexibleSextantUse,
			configuration.SetFlexibleSextantUse, flexibleSextantUseChangePath,
			"Flexible sextant use preference updated."), true
	default:
		return debugging.Result{ //nolint:exhaustruct // Unhandled dispatch returns the zero result by contract.
		}, false
	}
}

func persistReduxBooleanPreference(
	request debugging.Request,
	configuration *config.UltimaVConfiguration,
	before bool,
	save func(bool) error,
	path, summary string,
) debugging.Result {
	after := mustDebugBooleanArgument(request, "enabled")
	if err := save(after); err != nil {
		return debugging.FailureResult(request, "gameplay_configuration_save_failed",
			fmt.Sprintf("saving %s: %v", path, err))
	}
	result := debugSuccessResultWithMode(request, debugging.ExecutionLiveSafe,
		summary, newReduxModificationsStatusData(configuration))
	appendSemanticChange(&result.Changes, path, before, after)

	return result
}

func newReduxModificationsStatusData(
	configuration *config.UltimaVConfiguration,
) debugging.ReduxModificationsStatusData {
	gameplay := requireGameplayConfiguration(configuration)

	return debugging.ReduxModificationsStatusData{
		ConfigurationDiagnostics: configuration.ConfigurationDiagnostics(),
		ConfigurationEditing:     configuration.ConfigurationEditing(),
		ConfirmQuickSave:         configuration.GetConfirmQuickSave(),
		Navigation: debugging.ReduxNavigationModificationsData{
			FlexibleSextantUse: gameplay.FlexibleSextantUse,
		},
		Presentation: debugging.ReduxPresentationModificationsData{
			SmoothSurfaceMovement:  gameplay.SmoothSurfaceMovement,
			MaxVisibleActionPopups: configuration.GetMaxVisibleActionPopups(),
		},
		Combat: debugging.ReduxCombatModificationsData{
			AllowRepeatedMonsterDivision: gameplay.AllowRepeatedMonsterDivision,
			SpeedPercent:                 int(configuration.GetCombatSpeedFactor() * debugPercentScale),
			ShowHealthBars:               configuration.GetShowCombatHPBars(),
		},
		Dungeons: debugging.ReduxDungeonModificationsData{
			PersistHiddenDungeonDiscoveries: gameplay.PersistHiddenDungeonDiscoveries,
			ShowSubduedUnlitDungeons:        gameplay.ShowSubduedUnlitDungeons,
		},
	}
}

func executeQoLPreference(
	request debugging.Request, configuration *config.UltimaVConfiguration,
) (debugging.Result, bool) {
	var before, after any
	var err error
	var path string
	mode := debugging.ExecutionLiveSafe
	switch request.CommandID {
	case debugging.CommandCombatPresentationSpeedSet:
		var percent int
		if err = json.Unmarshal(request.Arguments["percent"], &percent); err != nil {
			return debugging.FailureResult(request, "invalid_argument", "Combat speed requires an integer percent."), true
		}
		before, after = int(configuration.GetCombatSpeedFactor()*debugPercentScale), percent
		err = configuration.SetCombatSpeedFactor(float64(percent) / debugPercentScale)
		path, mode = "gameplay.combat.speedPercent", debugging.ExecutionPresentation
	case debugging.CommandReduxActionPopupsSet:
		var count int
		if err = json.Unmarshal(request.Arguments["count"], &count); err != nil {
			return debugging.FailureResult(request, "invalid_argument", "Action popups requires an integer count."), true
		}
		before, after = configuration.GetMaxVisibleActionPopups(), count
		err = configuration.SetMaxVisibleActionPopups(count)
		path = "gameplay.presentation.maxVisibleActionPopups"
	case debugging.CommandReduxConfirmQuickSaveSet, debugging.CommandReduxCombatHPBarsSet:
		var enabled bool
		if err = json.Unmarshal(request.Arguments["enabled"], &enabled); err != nil {
			return debugging.FailureResult(request, "invalid_argument", "Preference requires a boolean."), true
		}
		after = enabled
		if request.CommandID == debugging.CommandReduxConfirmQuickSaveSet {
			before, path = configuration.GetConfirmQuickSave(), "gameplay.saves.confirmQuickSave"
			err = configuration.SetConfirmQuickSave(enabled)
		} else {
			before, path = configuration.GetShowCombatHPBars(), "gameplay.combat.showHealthBars"
			err = configuration.SetShowCombatHPBars(enabled)
		}
	default:
		return debugging.Result{}, false //nolint:exhaustruct // Unhandled dispatch returns the zero result.
	}
	if err != nil {
		code := "gameplay_configuration_save_failed"
		if errors.Is(err, config.ErrGameplayPreference) {
			code = "invalid_argument"
		}

		return debugging.FailureResult(request, code, err.Error()), true
	}
	var data any = newReduxModificationsStatusData(configuration)
	if request.CommandID == debugging.CommandCombatPresentationSpeedSet {
		// Preserve the compatibility command's percent result as well as its ID.
		data = struct {
			Percent int `json:"percent"`
		}{
			Percent: int(configuration.GetCombatSpeedFactor() * debugPercentScale),
		}
	}
	result := debugSuccessResultWithMode(request, mode, "Gameplay preference updated.", data)
	appendSemanticChange(&result.Changes, path, before, after)

	return result, true
}

func requireGameplayConfiguration(configuration *config.UltimaVConfiguration) *config.GameplayConfiguration {
	if configuration == nil || configuration.GameplayConfiguration == nil {
		panic("Redux modifications require initialized gameplay configuration")
	}

	return configuration.GameplayConfiguration
}
