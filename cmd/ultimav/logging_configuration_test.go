package main

import (
	"encoding/json"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

func TestLoggingBulkMutationPersistsAndSavedConfigurationRestores(t *testing.T) {
	logger := logging.GetGlobalLogger()
	runtimeBefore := currentLoggingRuntimeState()
	t.Cleanup(func() { applyLoggingRuntimeState(runtimeBefore) })

	configuration, configFile := initializeDialogConfigFile(t, "logging_preferences")
	configuration.LoggingConfiguration = &config.LoggingConfiguration{
		Enabled: true, StdoutEnabled: false, MinLevel: "DEBUG",
		CategoryStates: map[string]bool{logging.CategoryMovement: false},
	}
	applySavedLoggingConfiguration(configuration)
	assert.False(t, logger.GetCategoryStates()[logging.CategoryMovement])

	result, handled := executeApplicationDebugCommand(debugging.Request{
		SessionID: "test", ID: "enable-all", CommandID: debugging.CommandLoggingCategoriesSetAll,
		Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`true`)},
	}, configuration)
	require.True(t, handled)
	require.True(t, result.Success)
	assert.Len(t, result.Changes, len(logging.GetAllCategories()))
	for _, category := range logging.GetAllCategories() {
		assert.True(t, logger.GetCategoryStates()[category], category)
		assert.True(t, configuration.LoggingConfiguration.CategoryStates[category], category)
	}

	source := viper.New()
	source.SetConfigFile(configFile)
	require.NoError(t, source.ReadInConfig())
	persisted := source.GetStringMap("Logging.CategoryStates")
	for _, category := range logging.GetAllCategories() {
		assert.Equal(t, true, persisted[category], category)
	}
}
