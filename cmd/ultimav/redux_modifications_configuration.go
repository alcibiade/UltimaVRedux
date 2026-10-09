package main

import (
	"fmt"

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
	switch request.CommandID {
	case debugging.CommandReduxModificationsStatus:
		return debugSuccessResult(
			request,
			"Redux gameplay modifications loaded.",
			newReduxModificationsStatusData(configuration),
		), true
	case debugging.CommandReduxCombatRepeatedMonsterDivisionSet:
		before := requireGameplayConfiguration(configuration).AllowRepeatedMonsterDivision
		after := mustDebugBooleanArgument(request, "enabled")
		if err := configuration.SetAllowRepeatedMonsterDivision(after); err != nil {
			return debugging.FailureResult(
				request,
				"gameplay_configuration_save_failed",
				fmt.Sprintf("saving repeated monster division preference: %v", err),
			), true
		}

		result := debugSuccessResultWithMode(
			request,
			debugging.ExecutionLiveSafe,
			"Repeated monster division preference updated.",
			newReduxModificationsStatusData(configuration),
		)
		appendSemanticChange(&result.Changes, repeatedMonsterDivisionChangePath, before, after)

		return result, true
	case debugging.CommandReduxDungeonsPersistHiddenDiscoveriesSet:
		before := requireGameplayConfiguration(configuration).PersistHiddenDungeonDiscoveries
		after := mustDebugBooleanArgument(request, "enabled")
		if err := configuration.SetPersistHiddenDungeonDiscoveries(after); err != nil {
			return debugging.FailureResult(
				request,
				"gameplay_configuration_save_failed",
				fmt.Sprintf("saving persistent dungeon discovery preference: %v", err),
			), true
		}

		result := debugSuccessResultWithMode(
			request,
			debugging.ExecutionLiveSafe,
			"Persistent dungeon discovery preference updated.",
			newReduxModificationsStatusData(configuration),
		)
		appendSemanticChange(&result.Changes, persistHiddenDungeonDiscoveriesChangePath, before, after)

		return result, true
	case debugging.CommandReduxDungeonsSubduedUnlitVisibilitySet:
		before := requireGameplayConfiguration(configuration).ShowSubduedUnlitDungeons
		after := mustDebugBooleanArgument(request, "enabled")
		if err := configuration.SetShowSubduedUnlitDungeons(after); err != nil {
			return debugging.FailureResult(
				request,
				"gameplay_configuration_save_failed",
				fmt.Sprintf("saving subdued unlit dungeon preference: %v", err),
			), true
		}

		result := debugSuccessResultWithMode(
			request,
			debugging.ExecutionLiveSafe,
			"Subdued unlit dungeon visibility preference updated.",
			newReduxModificationsStatusData(configuration),
		)
		appendSemanticChange(&result.Changes, showSubduedUnlitDungeonsChangePath, before, after)

		return result, true
	case debugging.CommandReduxPresentationSmoothSurfaceMovementSet:
		before := requireGameplayConfiguration(configuration).SmoothSurfaceMovement
		after := mustDebugBooleanArgument(request, "enabled")
		if err := configuration.SetSmoothSurfaceMovement(after); err != nil {
			return debugging.FailureResult(request, "gameplay_configuration_save_failed",
				fmt.Sprintf("saving smooth surface movement preference: %v", err)), true
		}
		result := debugSuccessResultWithMode(request, debugging.ExecutionLiveSafe,
			"Smooth surface movement preference updated.", newReduxModificationsStatusData(configuration))
		appendSemanticChange(&result.Changes, smoothSurfaceMovementChangePath, before, after)

		return result, true
	case debugging.CommandReduxNavigationFlexibleSextantUseSet:
		before := requireGameplayConfiguration(configuration).FlexibleSextantUse
		after := mustDebugBooleanArgument(request, "enabled")
		if err := configuration.SetFlexibleSextantUse(after); err != nil {
			return debugging.FailureResult(request, "gameplay_configuration_save_failed",
				fmt.Sprintf("saving flexible sextant use preference: %v", err)), true
		}
		result := debugSuccessResultWithMode(request, debugging.ExecutionLiveSafe,
			"Flexible sextant use preference updated.", newReduxModificationsStatusData(configuration))
		appendSemanticChange(&result.Changes, flexibleSextantUseChangePath, before, after)

		return result, true
	default:
		return debugging.Result{ //nolint:exhaustruct // Unhandled dispatch returns the zero result by contract.
		}, false
	}
}

func newReduxModificationsStatusData(
	configuration *config.UltimaVConfiguration,
) debugging.ReduxModificationsStatusData {
	gameplay := requireGameplayConfiguration(configuration)

	return debugging.ReduxModificationsStatusData{
		Navigation: debugging.ReduxNavigationModificationsData{
			FlexibleSextantUse: gameplay.FlexibleSextantUse,
		},
		Presentation: debugging.ReduxPresentationModificationsData{
			SmoothSurfaceMovement: gameplay.SmoothSurfaceMovement,
		},
		Combat: debugging.ReduxCombatModificationsData{
			AllowRepeatedMonsterDivision: gameplay.AllowRepeatedMonsterDivision,
		},
		Dungeons: debugging.ReduxDungeonModificationsData{
			PersistHiddenDungeonDiscoveries: gameplay.PersistHiddenDungeonDiscoveries,
			ShowSubduedUnlitDungeons:        gameplay.ShowSubduedUnlitDungeons,
		},
	}
}

func requireGameplayConfiguration(configuration *config.UltimaVConfiguration) *config.GameplayConfiguration {
	if configuration == nil || configuration.GameplayConfiguration == nil {
		panic("Redux modifications require initialized gameplay configuration")
	}

	return configuration.GameplayConfiguration
}
