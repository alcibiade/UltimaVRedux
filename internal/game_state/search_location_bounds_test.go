package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestAuthenticSearchRejectsInvalidPositionsAndIndices(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	aslm := gs.SearchManager.GetAuthenticLocationManager()

	for _, tc := range []struct {
		x, y   references.Coordinate
		level  references.FloorNumber
		mapLoc references.Location
		desc   string
	}{
		{-1, -1, 0, references.Moonglow, "negative coordinates"},
		{255, 255, 0, references.Moonglow, "large coordinates"},
		{0, 0, -2, references.Moonglow, "invalid level"},
		{0, 0, 0, references.Location(255), "invalid map"},
	} {
		assert.Nil(t, aslm.SearchAtPosition(tc.x, tc.y, tc.level, tc.mapLoc), tc.desc)
	}

	for _, index := range []int{-1, 113, 200, 65535} {
		aslm.markAsFound(index)
		assert.True(t, aslm.isAlreadyFound(index), "out-of-range index %d reads as found", index)
	}
}
