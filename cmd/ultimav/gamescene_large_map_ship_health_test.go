package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestLargeMapShipHealthReturnsPirateHull(t *testing.T) {
	tile := &references.Tile{Index: indexes.PirateShipUp} //nolint:exhaustruct // Sprite identity fixture.
	pirate := map_units.NewEnemyNPC(references.NewPirateShipEnemyReference(tile), 1)
	pirate.SetCurrentHP(63)

	currentHP, maxHP, ok := largeMapShipHealth(&pirate)

	require.True(t, ok)
	assert.Equal(t, 63, currentHP)
	assert.Equal(t, references.PirateShipMaxHP, maxHP)
}

func TestLargeMapShipHealthReturnsFriendlyFrigateHull(t *testing.T) {
	frigate := map_units.NewNPCFriendlyVehicle(
		references.FrigateVehicle,
		*references.NewNPCReferenceForVehicle(
			references.FrigateVehicle,
			references.Position{},
			0,
		),
	)
	frigate.GetVehicleDetails().SetHullHP(42)

	currentHP, maxHP, ok := largeMapShipHealth(frigate)

	require.True(t, ok)
	assert.Equal(t, 42, currentHP)
	assert.Equal(t, map_units.MaxHullHP, maxHP)
}
