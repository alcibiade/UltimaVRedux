package game_state

import (
	"errors"
	"sync"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/test/helpers"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/assert"

	_ "embed"
)

const (
	xTilesVisibleOnGameScreen = 19
	yTilesVisibleOnGameScreen = 13
)

//go:embed testdata/britain2_SAVED.GAM
var saveFile []byte

//nolint:gochecknoglobals
var (
	baseGameReferences   *references.GameReferences
	cachedIntegrationCfg *config.UltimaVConfiguration
	integrationDataMu    sync.Mutex
)

func getBaseGameReferences() *references.GameReferences {
	integrationDataMu.Lock()
	defer integrationDataMu.Unlock()
	cfg, refs, err := helpers.LoadGameData()
	if err != nil {
		if errors.Is(err, originaldata.ErrUnavailable) && !originaldata.Required() {
			if baseGameReferences != nil {
				cachedIntegrationCfg, baseGameReferences = nil, nil
			}

			return nil
		}
		// Legacy snapshot callers have no testing.T argument. Keep their existing
		// nil-on-optional-absence contract, but never hide selected-data failures.
		panic(err)
	}
	// Existing legacy callers read these globals after initialization. Do not
	// rewrite them on same-source cache hits, preserving concurrent read safety.
	if baseGameReferences != refs {
		cachedIntegrationCfg, baseGameReferences = cfg, refs
	}

	return baseGameReferences
}

func getBritain2GameState() *GameState {
	getBaseGameReferences()

	if cachedIntegrationCfg == nil {
		return nil // No configuration available
	}

	baseState := mustNewGameStateFromLegacySaveBytes(
		saveFile, cachedIntegrationCfg, baseGameReferences,
		xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen,
	)

	return baseState
}

func Test_LoadMetAndDeadNpcs(t *testing.T) {
	baseState := getBritain2GameState()
	if baseState == nil {
		originaldata.Check(t, originaldata.ErrUnavailable)
	}
	ps := &baseState.PartyState

	moonglowLoc := references.Moonglow
	assert.False(t, ps.MetNpcs()[moonglowLoc][0], "NPC 0 in Moonglow should not be met")
	assert.False(t, ps.DeadNpcs()[moonglowLoc][0], "NPC 0 in Moonglow should not be dead")
	assert.False(t, ps.MetNpcs()[moonglowLoc][1], "NPC 1 in Moonglow should not be met")
	assert.False(t, ps.DeadNpcs()[moonglowLoc][1], "NPC 1 in Moonglow should not be dead")
	assert.False(t, ps.MetNpcs()[moonglowLoc][2], "NPC 2 in Moonglow should not be met")
	assert.False(t, ps.DeadNpcs()[moonglowLoc][2], "NPC 2 in Moonglow should not be dead")
	assert.True(t, ps.MetNpcs()[moonglowLoc][3], "NPC 3 in Moonglow should be met")
	assert.False(t, ps.DeadNpcs()[moonglowLoc][3], "NPC 3 in Moonglow should not be dead")
	assert.False(t, ps.MetNpcs()[moonglowLoc][4], "NPC 4 in Moonglow should not be met")
	assert.False(t, ps.DeadNpcs()[moonglowLoc][4], "NPC 4 in Moonglow should not be dead")

	britainLoc := references.Britain
	assert.False(t, ps.MetNpcs()[britainLoc][0], "NPC 0 in Britain should not be met")
	assert.False(t, ps.DeadNpcs()[britainLoc][0], "NPC 0 in Britain should not be dead")

	// Debug print actual values for the first few NPCs in Moonglow and Britain
	for i := 0; i < 5; i++ {
		metMoonglow := ps.MetNpcs()[moonglowLoc][i]
		deadMoonglow := ps.DeadNpcs()[moonglowLoc][i]
		t.Logf("Moonglow NPC %d: met=%v, dead=%v", i, metMoonglow, deadMoonglow)
	}
	for i := 0; i < 5; i++ {
		metBritain := ps.MetNpcs()[britainLoc][i]
		deadBritain := ps.DeadNpcs()[britainLoc][i]
		t.Logf("Britain NPC %d: met=%v, dead=%v", i, metBritain, deadBritain)
	}
}
