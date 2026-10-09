package game_state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/minimap"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestDebugBeginBlackthornInterrogationPreparesProductionWorkflow(t *testing.T) {
	state, _ := newBlackthornTestState(t, 2)
	writeDebugScriptedSceneMaps(t, state)
	state.SetShrineRuined(debugBlackthornShrineIndex, true)

	result, review, err := state.DebugBeginBlackthornInterrogation()

	require.NoError(t, err)
	assert.Equal(t, references.BlackthornPrison, state.MapState.PlayerLocation.Location)
	assert.Equal(t, references.FloorNumber(0), state.MapState.PlayerLocation.Floor)
	assert.True(t, result.ShrineRuinedBefore)
	assert.False(t, result.ShrineRuinedAfter)
	assert.True(t, state.HasActiveBlackthornInterrogation())
	assert.True(t, review.NeedsAnswer)
	assert.Equal(t, debugBlackthornShrineIndex, review.ShrineIndex)
}

func TestDebugBeginLordBritishFinaleControlsBoxAndUsesProductionAuthority(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	for index := 2; index < party_state.NRosterSlots; index++ {
		state.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapUnrecruited
	}
	writeDebugScriptedSceneMaps(t, state)
	state.PartyState.Inventory.QuestItems.Set(references.WoodenBox, 4)
	companion := state.PartyState.GetCharacter(1)
	companion.SetStatus(party_state.Dead)
	companion.SetHp(0)

	result, review, err := state.DebugBeginLordBritishFinale(false)

	require.NoError(t, err)
	assert.Equal(t, uint16(4), result.WoodenBoxBefore)
	assert.Zero(t, result.WoodenBoxAfter)
	assert.Equal(t, references.FinaleLocation, state.MapState.PlayerLocation.Location)
	assert.False(t, state.HasPendingLordBritishFinale())
	assert.False(t, state.CanAcceptGameplayCommands())
	assert.False(t, state.PartyState.Inventory.QuestItems.HasSome(references.WoodenBox))
	require.Len(t, review.Party, 2)
	assert.True(t, review.Party[1].Revived)
	assert.Equal(t, party_state.Good, companion.GetStatus())
	assert.Equal(t, companion.GetMaxHp(), companion.GetCurrentHp())
}

func writeDebugScriptedSceneMaps(t *testing.T, state *GameState) {
	t.Helper()
	directory := t.TempDir()
	const storedColumns = 16
	data := make([]byte, 4*minimap.MiscMapSize*storedColumns)
	for mapIndex := range 4 {
		for row := range minimap.MiscMapSize {
			for column := range minimap.MiscMapSize {
				offset := mapIndex*minimap.MiscMapSize*storedColumns + row*storedColumns + column
				data[offset] = byte(indexes.BrickFloor)
			}
		}
	}
	require.NoError(t, os.WriteFile(filepath.Join(directory, "MISCMAPS.DAT"), data, 0o600))
	require.NotNil(t, state.GameConfig)
	require.NotNil(t, state.GameConfig.SavedConfigData)
	configuration := *state.GameConfig
	saved := *configuration.SavedConfigData
	saved.DataFilePath = directory
	configuration.SavedConfigData = &saved
	state.GameConfig = &configuration
}
