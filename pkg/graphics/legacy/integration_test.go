package legacy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bradhannah/UltimaVRedux/test/originaldata"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationDecodeCatalog(t *testing.T) {
	dataDir := originaldata.Directory(t)

	for _, spec := range Catalog() {
		spec := spec
		t.Run(spec.Name, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(filepath.Join(dataDir, spec.Name))
			require.NoError(t, err)

			archive, err := Decode(spec, data)
			require.NoError(t, err)
			assert.NotEmpty(t, archive.Entries)
		})
	}
}
