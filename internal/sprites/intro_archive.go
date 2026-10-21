package sprites

import (
	"errors"
	"fmt"
	"os"

	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/internal/graphics/legacy"
)

var errIntroArchiveInvalid = errors.New("intro archive is invalid")

func loadIntroArchive(dataDirectory, name string) (legacy.Archive, error) {
	spec, ok := legacy.FindSpec(name)
	if !ok {
		return legacy.Archive{}, fmt.Errorf("%w: graphics catalog has no entry for %s", errIntroArchiveInvalid, name)
	}
	path := files.Resolve(dataDirectory, name)
	data, err := os.ReadFile(path) //nolint:gosec // Configured original-data directory is user-selected.
	if err != nil {
		return legacy.Archive{}, fmt.Errorf("reading intro graphics %s: %w", path, err)
	}
	archive, err := legacy.Decode(spec, data)
	if err != nil {
		return legacy.Archive{}, fmt.Errorf("decoding intro graphics %s: %w", name, err)
	}

	return archive, nil
}
