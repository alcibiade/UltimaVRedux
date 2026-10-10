package references

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDocksLoadsCanonicalShipwrightCoordinates(t *testing.T) {
	t.Parallel()

	rawData := make([]byte, startDockYOffset+4)
	expected := []DockReference{
		{Location: Jhelom, Position: Position{X: 0x27, Y: 0xdd}},
		{Location: Minoc, Position: Position{X: 0x97, Y: 0x15}},
		{Location: EastBritanny, Position: Position{X: 0x4f, Y: 0x6d}},
		{Location: BuccaneersDen, Position: Position{X: 0x8a, Y: 0x9f}},
	}
	for index, dock := range expected {
		rawData[startDockXOffset+index] = byte(dock.Position.X)
		rawData[startDockYOffset+index] = byte(dock.Position.Y)
	}

	docks, err := NewDocks(&config.UltimaVConfiguration{RawDataOvl: rawData}) //nolint:exhaustruct
	require.NoError(t, err)
	require.NotNil(t, docks)
	assert.Equal(t, expected, docks.Docks)
}

func TestDockPositionCheckedDistinguishesMissingFromZeroPosition(t *testing.T) {
	t.Parallel()

	docks := &DockReferences{Docks: []DockReference{
		{Location: Jhelom, Position: Position{}},
		{Location: Minoc, Position: Position{X: 0x97, Y: 0x15}},
	}}

	position, ok := docks.GetDockPositionChecked(Jhelom)
	assert.True(t, ok)
	assert.Equal(t, Position{}, position)

	position, ok = docks.GetDockPositionChecked(Minoc)
	assert.True(t, ok)
	assert.Equal(t, Position{X: 0x97, Y: 0x15}, position)

	position, ok = docks.GetDockPositionChecked(Britain)
	assert.False(t, ok)
	assert.Equal(t, Position{}, position)

	var nilDocks *DockReferences
	_, ok = nilDocks.GetDockPositionChecked(Jhelom)
	assert.False(t, ok)
}
