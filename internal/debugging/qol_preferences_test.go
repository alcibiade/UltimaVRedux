package debugging

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQoLCatalogSteppedDomainsAndScenarioExclusion(t *testing.T) {
	catalog := NewInitialCatalog()
	allowed := map[int]bool{50: true, 100: true, 150: true, 200: true, 300: true, 400: true,
		600: true, 800: true, 1200: true, 1600: true, 2000: true}
	for percent := 0; percent <= 2100; percent++ {
		err := catalog.ValidateRequest(Request{SessionID: "test", ID: "speed", CommandID: CommandCombatPresentationSpeedSet,
			Arguments: map[string]json.RawMessage{"percent": json.RawMessage(strconv.Itoa(percent))}})
		if allowed[percent] {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
	for _, id := range []string{CommandCombatPresentationSpeedSet, CommandReduxConfirmQuickSaveSet, CommandReduxCombatHPBarsSet, CommandReduxActionPopupsSet} {
		definition, exists := catalog.Get(id)
		require.True(t, exists)
		assert.True(t, definition.ScenarioExcluded)
	}
	for _, raw := range []string{"-1", "4", "1.5", "null", `"0"`} {
		require.Error(t, catalog.ValidateRequest(Request{SessionID: "test", ID: "popups", CommandID: CommandReduxActionPopupsSet,
			Arguments: map[string]json.RawMessage{"count": json.RawMessage(raw)}}))
	}
	for count := 0; count <= 3; count++ {
		require.NoError(t, catalog.ValidateRequest(Request{SessionID: "test", ID: "popups", CommandID: CommandReduxActionPopupsSet,
			Arguments: map[string]json.RawMessage{"count": json.RawMessage(strconv.Itoa(count))}}))
	}
}

func TestNumericStepCatalogValidationAndCloneIsolation(t *testing.T) {
	definition, exists := NewInitialCatalog().Get(CommandCombatPresentationSpeedSet)
	require.True(t, exists)
	*definition.Parameters[0].Step = 0
	_, err := NewCatalog([]CommandDefinition{definition})
	require.ErrorIs(t, err, ErrInvalidCatalog)
	catalog := NewInitialCatalog()
	definition, _ = catalog.Get(CommandCombatPresentationSpeedSet)
	*definition.Parameters[0].Step = 1
	original, _ := catalog.Get(CommandCombatPresentationSpeedSet)
	assert.Equal(t, 50, *original.Parameters[0].Step)
}

func TestIntegerChoicesMustSatisfyBoundsAndStep(t *testing.T) {
	definition, exists := NewInitialCatalog().Get(CommandCombatPresentationSpeedSet)
	require.True(t, exists)
	assert.Equal(t, Choice{Value: "1200", Label: "12x"}, definition.Parameters[0].Choices[8])
	for _, value := range []string{"2050", "75", "twelve"} {
		changed, _ := NewInitialCatalog().Get(CommandCombatPresentationSpeedSet)
		changed.Parameters[0].Choices = append(changed.Parameters[0].Choices, Choice{Value: value, Label: value})
		_, err := NewCatalog([]CommandDefinition{changed})
		require.ErrorIs(t, err, ErrInvalidCatalog, value)
	}
}
