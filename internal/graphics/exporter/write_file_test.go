package exporter

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteFilePreservesEncodingAndCloseFailures(t *testing.T) {
	encodeErr := errors.New("encoding failed")
	err := writeFile(filepath.Join(t.TempDir(), "failed.png"), func(writer io.Writer) error {
		// Closing in the injected encoder makes the owner's close fail as well.
		require.NoError(t, writer.(*os.File).Close())
		return encodeErr
	})
	require.ErrorIs(t, err, encodeErr)
	require.ErrorIs(t, err, os.ErrClosed)
}
