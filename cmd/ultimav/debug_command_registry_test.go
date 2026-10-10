package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
)

func TestDebugCommandRegistryExactlyCoversCatalog(t *testing.T) {
	catalog := debugging.NewInitialCatalog()
	registry, err := newDebugCommandRegistry(catalog)
	require.NoError(t, err)

	definitions := catalog.Definitions()
	require.Len(t, registry.registrations, len(definitions))
	for _, definition := range definitions {
		registration, ok := registry.lookup(definition.ID)
		assert.True(t, ok, "catalog command %q has no application registration", definition.ID)
		if !ok {
			continue
		}
		assert.NotEmpty(t, registration.owner, definition.ID)
		assert.NotNil(t, registration.handler, definition.ID)
	}
}

func TestDebugCommandRegistryRejectsDuplicateOwner(t *testing.T) {
	registry := newEmptyDebugCommandRegistry()
	handler := func(debugCommandContext, debugging.Request) debugging.Result { return debugging.Result{} }
	require.NoError(t, registry.register(debugCommandRegistration{
		commandID: debugging.CommandEnvironmentWindStatus,
		owner:     "environment",
		handler:   handler,
	}))

	err := registry.register(debugCommandRegistration{
		commandID: debugging.CommandEnvironmentWindStatus,
		owner:     "shadowing-owner",
		handler:   handler,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), debugging.CommandEnvironmentWindStatus)
}

func TestDebugCommandRegistryCoLocatesGuardsAndRuntimeResolvers(t *testing.T) {
	catalog := debugging.NewInitialCatalog()
	registry, err := newDebugCommandRegistry(catalog)
	require.NoError(t, err)

	guarded := []string{
		debugging.CommandQuestItemEnsure,
		debugging.CommandQuestItemRemove,
		debugging.CommandMovementTeleport,
		debugging.CommandInventoryEquipmentClear,
		debugging.CommandPartyEquipmentLoadoutUpdate,
		debugging.CommandCombatMonsterSleepSet,
		debugging.CommandCombatExitDebug,
		debugging.CommandCombatMonsterSpawn,
		debugging.CommandCombatDungeonRoomClear,
	}
	for _, commandID := range guarded {
		registration, ok := registry.lookup(commandID)
		require.True(t, ok, commandID)
		assert.NotNil(t, registration.guard, commandID)
	}

	for _, definition := range catalog.Definitions() {
		if !commandHasScenarioRuntimeParameter(definition) {
			continue
		}
		registration, ok := registry.lookup(definition.ID)
		require.True(t, ok, definition.ID)
		assert.NotNil(t, registration.runtimeResolver, definition.ID)
	}
}

func TestDebugCommandRegistryRejectsMissingScenarioRuntimeResolver(t *testing.T) {
	const commandID = "test.scenario-runtime"
	catalog, err := debugging.NewCatalog([]debugging.CommandDefinition{{
		ID: commandID, Category: "Test", Name: "Runtime", Description: "Tests runtime resolver validation.",
		Parameters: []debugging.ParameterDefinition{{
			ID: "expectedFingerprint", Name: "Expected Fingerprint", Kind: debugging.ParameterString,
			ScenarioRuntime: true,
		}},
		ExecutionModes: []debugging.ExecutionMode{debugging.ExecutionSynchronized},
	}})
	require.NoError(t, err)
	registry := newEmptyDebugCommandRegistry()
	require.NoError(t, registry.register(debugCommandRegistration{
		commandID: commandID,
		owner:     "test",
		handler:   func(debugCommandContext, debugging.Request) debugging.Result { return debugging.Result{} },
	}))

	err = validateDebugCommandRegistry(catalog, registry)

	require.ErrorIs(t, err, errDebugRuntimeResolverMissing)
	assert.Contains(t, err.Error(), commandID)
}

func TestDebugCommandRegistryDoesNotRequireRuntimeResolverForScenarioValueResolver(t *testing.T) {
	const commandID = "test.scenario-value-resolver"
	catalog, err := debugging.NewCatalog([]debugging.CommandDefinition{{
		ID: commandID, Category: "Test", Name: "Value Resolver", Description: "Tests generic scenario value resolution.",
		Parameters: []debugging.ParameterDefinition{{
			ID: "target", Name: "Target", Kind: debugging.ParameterCharacter,
			ScenarioChoices:       []debugging.Choice{{Value: "1", Label: "Party Position 1"}},
			ScenarioValueResolver: debugging.ScenarioValueResolverActivePartyPosition,
		}},
		ExecutionModes: []debugging.ExecutionMode{debugging.ExecutionSynchronized},
	}})
	require.NoError(t, err)
	registry := newEmptyDebugCommandRegistry()
	require.NoError(t, registry.register(debugCommandRegistration{
		commandID: commandID,
		owner:     "test",
		handler:   func(debugCommandContext, debugging.Request) debugging.Result { return debugging.Result{} },
	}))

	assert.NoError(t, validateDebugCommandRegistry(catalog, registry))
}

func TestDebugCommandRegistryLookupSelectsOneOwnerWithoutRoutingOrder(t *testing.T) {
	registry, err := newDebugCommandRegistry(debugging.NewInitialCatalog())
	require.NoError(t, err)

	registration, ok := registry.lookup(debugging.CommandCombatStatus)
	require.True(t, ok)
	assert.Equal(t, "active combat", registration.owner)
	assert.Equal(t, debugCommandScene, registration.scope)
}

func TestDebugCommandRegistryOwnsDisplayCommandsAtApplicationScope(t *testing.T) {
	registry, err := newDebugCommandRegistry(debugging.NewInitialCatalog())
	require.NoError(t, err)

	for _, commandID := range []string{
		debugging.CommandDisplayStatus,
		debugging.CommandDisplayModeSet,
		debugging.CommandDisplayResolutionSet,
		debugging.CommandDisplayFullscreenSet,
	} {
		registration, ok := registry.lookup(commandID)
		require.True(t, ok, commandID)
		assert.Equal(t, "display manager", registration.owner, commandID)
		assert.Equal(t, debugCommandApplication, registration.scope, commandID)
	}
}

func TestDebugCommandRegistryOwnsSextantPreferenceAtApplicationScope(t *testing.T) {
	registry, err := newDebugCommandRegistry(debugging.NewInitialCatalog())
	require.NoError(t, err)
	registration, ok := registry.lookup(debugging.CommandReduxNavigationFlexibleSextantUseSet)
	require.True(t, ok)
	assert.Equal(t, "redux modifications", registration.owner)
	assert.Equal(t, debugCommandApplication, registration.scope)
}
