package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestReviewCodexMemorialPaths(t *testing.T) {
	tests := []struct {
		name       string
		martyrs    int
		wantUrns   int
		wantText   string
		absentText string
	}{
		{name: "none", martyrs: 0, wantUrns: 0, wantText: ""},
		{name: "singular", martyrs: 1, wantUrns: 1, wantText: "an urn marked:", absentText: "urns marked:"},
		{name: "plural", martyrs: 4, wantUrns: 4, wantText: "urns marked:"},
		{name: "more than source layouts", martyrs: 10, wantUrns: 8, wantText: "MARTYR10"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).
				WithLocation(references.PalaceOfBlackthorn).
				WithSystemCallbacks().
				Build()
			for index := range party_state.NRosterSlots {
				character := state.PartyState.GetCharacter(index)
				character.PartyMap = party_state.PartyMapUnrecruited
				if index > 0 && index <= test.martyrs {
					clear(character.Name[:])
					copy(character.Name[:], fmt.Sprintf("martyr%d", index))
					character.PartyMap = party_state.PartyMapMartyr
				}
			}

			memorial := state.ReviewCodexMemorial()
			require.Len(t, memorial.Names, test.martyrs)
			require.Len(t, memorial.UrnPositions, test.wantUrns)
			require.Contains(t, memorial.Text, test.wantText)
			if test.absentText != "" {
				require.NotContains(t, memorial.Text, test.absentText)
			}
		})
	}
}
