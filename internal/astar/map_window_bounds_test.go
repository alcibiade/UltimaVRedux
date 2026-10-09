package astar

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

func TestMapWindowRejectsRadiusOutsideCoordinateSpace(t *testing.T) {
	for _, spec := range []struct {
		radius int
		width  references.Coordinate
		height references.Coordinate
	}{
		{-1, 256, 256},
		{257, 256, 256},
		{15, 0, 256},
	} {
		m := NewAStarMap()
		assert.Panics(t, func() {
			m.InitializeByLayeredMapWithLimit(nil, nil, nil, true, references.Position{},
				spec.radius, spec.width, spec.height)
		})
	}
	assert.Equal(t, references.Coordinate(-15), mapWindowRemainder(-15, 256))
	assert.Equal(t, references.Coordinate(14), mapWindowRemainder(270, 256))
}
