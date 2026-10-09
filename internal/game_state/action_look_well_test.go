package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

// TestWellDetection_NilPosition tests well detection with nil position
func TestWellDetection_NilPosition(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Paws).
		WithPlayerAt(10, 10).
		WithSystemCallbacks().
		Build()

	// Detect well at nil position should return false
	assert.False(t, gs.DetectWishingWellAtPosition(nil),
		"Well detection with nil position should return false")
}

// TestWellSprite validates well sprite index constant
func TestWellSprite(t *testing.T) {
	// Verify that the well sprite index is correct
	// Well tile should be indexes.Well (161)
	assert.Equal(t, 161, int(indexes.Well),
		"Well sprite should be index 161")
}
