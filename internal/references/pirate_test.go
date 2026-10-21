package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestNewPirateShipEnemyReference prevents natural water generation from
// producing the immobile person-in-stocks representation for pirate ships.
func TestNewPirateShipEnemyReference(t *testing.T) {
	tile := &Tile{Index: indexes.PirateShipLeft}

	pirate := NewPirateShipEnemyReference(tile)

	require.Same(t, tile, pirate.KeyFrameTile)
	assert.Equal(t, MonsterStocks, pirate.MonsterType)
	assert.Equal(t, PirateShipMaxHP, pirate.HitPoints)
	assert.True(t, pirate.IsWaterEnemy)
	assert.True(t, pirate.ActivelyAttacks)
	assert.False(t, pirate.DoNotMove)
	assert.True(t, pirate.HasAbility(NoCorpse))
}

func TestNewPirateShipEnemyReferenceRejectsNonPirateTile(t *testing.T) {
	tile := &Tile{Index: indexes.Water1}

	assert.PanicsWithValue(t, "pirate ship requires a pirate ship tile", func() {
		NewPirateShipEnemyReference(tile)
	})
}
