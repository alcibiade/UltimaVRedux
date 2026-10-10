package map_units

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestGetVisibleEnemyAtPositionAndFloorOrNil(t *testing.T) {
	t.Parallel()

	position := references.Position{X: 12, Y: 34}
	enemy := NewEnemyNPC(references.EnemyReference{}, 1) //nolint:exhaustruct // Identity-only lookup fixture.
	enemy.SetPos(position)
	enemy.SetFloor(2)
	enemy.SetVisible(true)

	friendly := NewNPCFriendlyVehiceNewRef(references.FrigateVehicle, position, 2)
	mapUnits := MapUnits{friendly, &enemy}

	found := mapUnits.GetVisibleEnemyAtPositionAndFloorOrNil(position, 2)
	require.NotNil(t, found)
	assert.Same(t, &enemy, found, "lookup must preserve stable enemy identity for combat outcomes")
}

func TestGetVisibleEnemyAtPositionAndFloorOrNilFiltersInvalidContacts(t *testing.T) {
	t.Parallel()

	position := references.Position{X: 12, Y: 34}
	enemy := NewEnemyNPC(references.EnemyReference{}, 1) //nolint:exhaustruct // Identity-only lookup fixture.
	enemy.SetPos(position)
	enemy.SetFloor(2)
	enemy.SetVisible(true)
	mapUnits := MapUnits{&enemy}

	assert.Nil(t, mapUnits.GetVisibleEnemyAtPositionAndFloorOrNil(references.Position{X: 13, Y: 34}, 2))
	assert.Nil(t, mapUnits.GetVisibleEnemyAtPositionAndFloorOrNil(position, 1))

	enemy.SetVisible(false)
	assert.Nil(t, mapUnits.GetVisibleEnemyAtPositionAndFloorOrNil(position, 2))

	var nilMapUnits *MapUnits
	assert.Nil(t, nilMapUnits.GetVisibleEnemyAtPositionAndFloorOrNil(position, 2))
}

func TestRemoveEnemyRemovesExactStoredInstance(t *testing.T) {
	t.Parallel()

	position := references.Position{X: 12, Y: 34}
	first := NewEnemyNPC(references.EnemyReference{}, 1)  //nolint:exhaustruct // Identity-only removal fixture.
	second := NewEnemyNPC(references.EnemyReference{}, 2) //nolint:exhaustruct // Identity-only removal fixture.
	first.SetPos(position)
	second.SetPos(position)
	first.SetVisible(true)
	second.SetVisible(true)
	mapUnits := MapUnits{&first, &second}

	require.True(t, mapUnits.RemoveEnemy(&second))
	require.Len(t, mapUnits, 1)
	assert.Same(t, &first, mapUnits[0])
	assert.True(t, first.IsVisible())
	assert.False(t, second.IsVisible())
	assert.False(t, mapUnits.RemoveEnemy(&second))
}

func TestEnemyInstanceOwnsDurabilityFacingVariantAndShadowlordIdentity(t *testing.T) {
	t.Parallel()
	up := &references.Tile{Index: indexes.PirateShipUp}     //nolint:exhaustruct // Sprite identity is sufficient.
	left := &references.Tile{Index: indexes.PirateShipLeft} //nolint:exhaustruct // Sprite identity is sufficient.
	pirateReference := references.NewPirateShipEnemyReference(up)
	pirate := NewEnemyNPC(pirateReference, 1)
	pirate.SetCurrentHP(43)
	pirate.SetFacing(references.Left, left)

	assert.Equal(t, EnemyVariantPirate, pirate.Variant())
	assert.Equal(t, 43, pirate.CurrentHP())
	assert.Equal(t, references.Left, pirate.Facing())
	assert.Equal(t, indexes.SpriteIndex(indexes.PirateShipLeft), pirate.GetKeySpriteIndex())
	assert.Equal(t, indexes.SpriteIndex(indexes.PirateShipUp), pirate.EnemyReference.KeyFrameTile.Index,
		"instance facing must not mutate reference data")

	shadowlord := NewEnemyNPC(references.EnemyReference{
		MonsterType:  references.MonsterShadowlord,
		KeyFrameTile: &references.Tile{Index: indexes.ShadowLord_KeyIndex}, //nolint:exhaustruct
	}, 2) //nolint:exhaustruct
	shadowlord.SetShadowlordIdentity(references.Nosfentor)
	lord, ok := shadowlord.ShadowlordIdentity()
	assert.True(t, ok)
	assert.Equal(t, references.Nosfentor, lord)
}
