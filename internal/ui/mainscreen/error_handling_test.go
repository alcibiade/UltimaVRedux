package mainscreen

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
)

func TestCharacterSummaryErrorHandling(t *testing.T) {
	summary := refreshTestCharacterSummary(t)
	screen := ebiten.NewImage(1, 1)
	t.Cleanup(screen.Dispose)
	party := party_state.LoadFromRaw([4192]byte{})
	summary.Draw(nil, screen, -1, false, -1)
	summary.Draw(party, nil, -1, false, -1)
	summary.Draw(party, screen, -1, false, -1)
	summary.Draw(nil, nil, -1, false, -1)
}

func TestProvisionSummaryErrorHandling(t *testing.T) {
	summary := &ProvisionSummary{}
	screen := ebiten.NewImage(1, 1)
	t.Cleanup(screen.Dispose)
	state := &game_state.GameState{}
	summary.Draw(nil, screen)
	summary.Draw(state, nil)
	summary.Draw(state, screen)
	summary.Draw(nil, nil)
	// Preserve the provision half of the retired refresh integration test.
	require.False(t, summary.IsInventoryRefreshNeeded())
	summary.MarkInventoryChanged()
	require.True(t, summary.IsInventoryRefreshNeeded())
}
