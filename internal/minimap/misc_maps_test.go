package minimap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeMiscMapUsesPaddedRecordLayout(t *testing.T) {
	data := make([]byte, miscMapBytes*miscMapCount)
	for mapIndex := range miscMapCount {
		for row := range MiscMapSize {
			for column := range MiscMapSize {
				data[mapIndex*miscMapBytes+row*miscMapStoredCols+column] = byte(mapIndex*40 + row*MiscMapSize + column)
			}
		}
	}

	decoded, err := DecodeMiscMap(data, 2)
	require.NoError(t, err)
	require.Equal(t, byte(80), decoded.Tiles[0][0])
	require.Equal(t, byte(80+3*MiscMapSize+7), decoded.Tiles[7][3])
}

func TestDecodeMiscMapRejectsInvalidInput(t *testing.T) {
	_, err := DecodeMiscMap(make([]byte, miscMapBytes), -1)
	require.ErrorIs(t, err, errMiscMapDataInvalid)

	_, err = DecodeMiscMap(make([]byte, miscMapBytes), 1)
	require.ErrorIs(t, err, errMiscMapDataInvalid)
}
