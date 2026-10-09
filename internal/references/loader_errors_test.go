package references

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
)

func TestConfiguredReferenceLoadersReturnErrors(t *testing.T) {
	t.Parallel()

	gameConfig := &config.UltimaVConfiguration{ //nolint:exhaustruct // Invalid-resource fixture.
		RawDataOvl: make([]byte, 1),
		SavedConfigData: &config.UltimaVConfigurationFlags{
			DataFilePath: t.TempDir(),
		},
	}

	_, err := NewDataOvl(gameConfig)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "location names")

	_, err = NewNPCReferences(gameConfig)
	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))

	_, err = NewLookReferences(gameConfig)
	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))

}

func TestDataOvlOffsetLoadersRejectTruncatedData(t *testing.T) {
	t.Parallel()
	gameConfig := &config.UltimaVConfiguration{RawDataOvl: make([]byte, 1)} //nolint:exhaustruct

	_, err := NewDocks(gameConfig)
	require.Error(t, err)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)

	_, err = NewWorldLocations(gameConfig)
	require.Error(t, err)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
}
