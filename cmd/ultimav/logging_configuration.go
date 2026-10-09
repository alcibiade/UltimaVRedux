package main

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

type loggingRuntimeState struct {
	enabled        bool
	stdoutEnabled  bool
	minimumLevel   logging.LogLevel
	categoryStates map[string]bool
}

func currentLoggingRuntimeState() loggingRuntimeState {
	logger := logging.GetGlobalLogger()

	return loggingRuntimeState{
		enabled: logger.IsEnabled(), stdoutEnabled: logger.IsStdoutEnabled(),
		minimumLevel: logger.GetMinLevel(), categoryStates: logger.GetCategoryStates(),
	}
}

func applyLoggingRuntimeState(state loggingRuntimeState) {
	logger := logging.GetGlobalLogger()
	logger.SetEnabled(state.enabled)
	logger.SetStdoutEnabled(state.stdoutEnabled)
	logger.SetMinLevel(state.minimumLevel)
	logger.SetCategoryStates(state.categoryStates)
}

func applySavedLoggingConfiguration(configuration *config.UltimaVConfiguration) {
	if configuration == nil || configuration.LoggingConfiguration == nil {
		panic("logging configuration is required")
	}
	saved := configuration.LoggingConfiguration
	applyLoggingRuntimeState(loggingRuntimeState{
		enabled: saved.Enabled, stdoutEnabled: saved.StdoutEnabled,
		minimumLevel: logging.ParseLogLevel(saved.MinLevel), categoryStates: cloneCategoryStates(saved.CategoryStates),
	})
}

func mutateAndPersistLoggingConfiguration(configuration *config.UltimaVConfiguration, mutate func()) error {
	runtimeBefore := currentLoggingRuntimeState()
	if configuration == nil {
		mutate()

		return nil
	}
	if configuration.LoggingConfiguration == nil {
		panic("logging configuration is required")
	}
	configurationBefore := *configuration.LoggingConfiguration
	configurationBefore.CategoryStates = cloneCategoryStates(configurationBefore.CategoryStates)

	mutate()
	runtimeAfter := currentLoggingRuntimeState()
	configuration.LoggingConfiguration.Enabled = runtimeAfter.enabled
	configuration.LoggingConfiguration.StdoutEnabled = runtimeAfter.stdoutEnabled
	configuration.LoggingConfiguration.MinLevel = runtimeAfter.minimumLevel.String()
	configuration.LoggingConfiguration.CategoryStates = cloneCategoryStates(runtimeAfter.categoryStates)
	if err := configuration.SaveLoggingConfiguration(); err != nil {
		applyLoggingRuntimeState(runtimeBefore)
		*configuration.LoggingConfiguration = configurationBefore

		return fmt.Errorf("saving logging configuration: %w", err)
	}

	return nil
}

func cloneCategoryStates(states map[string]bool) map[string]bool {
	cloned := make(map[string]bool, len(states))
	for category, enabled := range states {
		cloned[category] = enabled
	}

	return cloned
}
