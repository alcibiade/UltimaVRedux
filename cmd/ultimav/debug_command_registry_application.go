package main

import "github.com/bradhannah/UltimaVRedux/internal/debugging"

func registerApplicationDebugCommands(registry *debugCommandRegistry) {
	registry.registerGroup("application diagnostics", debugCommandApplication, func(
		context debugCommandContext,
		request debugging.Request,
	) debugging.Result {
		result, handled := executeApplicationDebugCommand(request, context.game.currentScene.GetUltimaConfiguration())
		if !handled {
			panic("registered application command was not handled: " + request.CommandID)
		}

		return result
	},
		debugging.CommandCombatMapCatalog,
		debugging.CommandLoggingStatus,
		debugging.CommandLoggingEnabledSet,
		debugging.CommandLoggingStdoutSet,
		debugging.CommandLoggingLevelSet,
		debugging.CommandLoggingCategorySet,
		debugging.CommandLoggingCategoriesSetAll,
		debugging.CommandShrineReferenceInspect,
	)
}

func registerDisplayDebugCommands(registry *debugCommandRegistry) {
	registry.registerGroup("display manager", debugCommandApplication, func(
		context debugCommandContext,
		request debugging.Request,
	) debugging.Result {
		result, handled := executeDisplayDebugCommand(
			request,
			context.game.currentScene.GetUltimaConfiguration(),
		)
		if !handled {
			panic("registered display command was not handled: " + request.CommandID)
		}

		return result
	},
		debugging.CommandDisplayStatus,
		debugging.CommandDisplayModeSet,
		debugging.CommandDisplayResolutionSet,
		debugging.CommandDisplayFullscreenSet,
	)
}

func registerReduxModificationsDebugCommands(registry *debugCommandRegistry) {
	registry.registerGroup("redux modifications", debugCommandApplication, func(
		context debugCommandContext,
		request debugging.Request,
	) debugging.Result {
		result, handled := executeReduxModificationsDebugCommand(
			request,
			context.game.currentScene.GetUltimaConfiguration(),
		)
		if !handled {
			panic("registered Redux modifications command was not handled: " + request.CommandID)
		}

		return result
	},
		debugging.CommandReduxModificationsStatus,
		debugging.CommandReduxCombatRepeatedMonsterDivisionSet,
		debugging.CommandReduxDungeonsPersistHiddenDiscoveriesSet,
		debugging.CommandReduxDungeonsSubduedUnlitVisibilitySet,
		debugging.CommandReduxPresentationSmoothSurfaceMovementSet,
		debugging.CommandReduxNavigationFlexibleSextantUseSet,
		debugging.CommandCombatPresentationSpeedSet,
		debugging.CommandReduxConfirmQuickSaveSet,
		debugging.CommandReduxCombatHPBarsSet,
		debugging.CommandReduxActionPopupsSet,
	)
}

func registerScenarioDefinitionDebugCommands(registry *debugCommandRegistry) {
	registry.registerGroup("scenario definitions", debugCommandApplication, func(
		context debugCommandContext,
		request debugging.Request,
	) debugging.Result {
		result, handled := executeScenarioDefinitionDebugCommand(
			context.game.scenarioRepository,
			context.game.debugCatalog,
			request,
		)
		if !handled {
			panic("registered scenario definition command was not handled: " + request.CommandID)
		}

		return result
	},
		debugging.CommandScenarioDefinitionList,
		debugging.CommandScenarioDefinitionGet,
		debugging.CommandScenarioDefinitionValidate,
		debugging.CommandScenarioDefinitionCreate,
		debugging.CommandScenarioDefinitionUpdate,
		debugging.CommandScenarioDefinitionDelete,
		debugging.CommandScenarioDefinitionDuplicate,
	)
}

func registerScenarioRunDebugCommands(registry *debugCommandRegistry) {
	registry.registerGroup("scenario runner", debugCommandApplication, applicationHandledDebugCommand(
		func(game *Game, request debugging.Request) (debugging.Result, bool) {
			return game.executeScenarioRunCommand(request)
		}),
		debugging.CommandScenarioRunStatus,
		debugging.CommandScenarioRunCancel,
	)
	if err := registry.register(debugCommandRegistration{
		commandID: debugging.CommandScenarioRunStart,
		owner:     "scenario runner",
		scope:     debugCommandApplicationValidated,
		handler: applicationHandledDebugCommand(func(game *Game, request debugging.Request) (debugging.Result, bool) {
			return game.executeScenarioRunCommand(request)
		}),
		guard:           nil,
		runtimeResolver: nil,
	}); err != nil {
		panic(err)
	}
}

func registerSnapshotDebugCommands(registry *debugCommandRegistry) {
	queryHandler := applicationHandledDebugCommand(func(game *Game, request debugging.Request) (debugging.Result, bool) {
		return game.executeSnapshotQueryCommand(request)
	})
	registry.registerGroup("snapshot repository", debugCommandApplication, queryHandler,
		debugging.CommandSaveSnapshotList,
		debugging.CommandSaveSnapshotGet,
	)
	mutationHandler := applicationHandledDebugCommand(
		func(game *Game, request debugging.Request) (debugging.Result, bool) {
			return game.executeSnapshotMutationCommand(request)
		},
	)
	for _, commandID := range []string{
		debugging.CommandSaveSnapshotCapture,
		debugging.CommandSaveSnapshotLoad,
		debugging.CommandSaveSnapshotDelete,
	} {
		if err := registry.register(debugCommandRegistration{
			commandID:       commandID,
			owner:           "snapshot repository",
			scope:           debugCommandApplicationValidated,
			handler:         mutationHandler,
			guard:           nil,
			runtimeResolver: nil,
		}); err != nil {
			panic(err)
		}
	}
}
