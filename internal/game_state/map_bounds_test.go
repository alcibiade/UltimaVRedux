package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestIsOutOfBoundsUsesTheSameInclusiveLimitOnBothAxes(t *testing.T) {
	large, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithSystemCallbacks().
		Build()
	small, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		Build()
	smallMaxX := small.GetCurrentSmallLocationReference().GetMaxX()
	smallMaxY := small.GetCurrentSmallLocationReference().GetMaxY()

	for _, tc := range []struct {
		name  string
		state *GameState
		pos   references.Position
		out   bool
	}{
		{"large max X", large, references.Position{X: references.XLargeMapTiles - 1, Y: 0}, false},
		{"large max Y", large, references.Position{X: 0, Y: references.YLargeMapTiles - 1}, false},
		{"large past X", large, references.Position{X: references.XLargeMapTiles, Y: 0}, true},
		{"large past Y", large, references.Position{X: 0, Y: references.YLargeMapTiles}, true},
		{"small origin", small, references.Position{X: 0, Y: 0}, false},
		{"small negative X", small, references.Position{X: -1, Y: 0}, true},
		{"small negative Y", small, references.Position{X: 0, Y: -1}, true},
		{"small max", small, references.Position{X: smallMaxX, Y: smallMaxY}, false},
		{"small past X", small, references.Position{X: smallMaxX + 1, Y: 0}, true},
		{"small past Y", small, references.Position{X: 0, Y: smallMaxY + 1}, true},
	} {
		assert.Equal(t, tc.out, tc.state.IsOutOfBounds(tc.pos), tc.name)
	}
}
