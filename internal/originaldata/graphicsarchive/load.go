// Package graphicsarchive performs bounded, cancellable CPU-only loading of
// original graphics archives.
package graphicsarchive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/pkg/graphics/legacy"
)

const maximumArchiveBytes = 4 << 20

var (
	errArchiveTooLarge     = errors.New("original graphics archive exceeds the supported size limit")
	errCatalogEntryMissing = errors.New("graphics catalog entry is unavailable")
)

// Load reads and decodes one catalogued archive without allocating GPU resources.
func Load(ctx context.Context, dataDirectory, name string) (_ legacy.Archive, err error) {
	if err := ctx.Err(); err != nil {
		return legacy.Archive{}, fmt.Errorf("loading graphics archive %s: %w", name, err)
	}
	spec, ok := legacy.FindSpec(name)
	if !ok {
		return legacy.Archive{}, fmt.Errorf("%w: %s", errCatalogEntryMissing, name)
	}
	path := files.Resolve(dataDirectory, name)
	info, err := os.Stat(path)
	if err != nil {
		return legacy.Archive{}, fmt.Errorf("checking graphics archive %s: %w", path, err)
	}
	if info.Size() > maximumArchiveBytes {
		return legacy.Archive{}, fmt.Errorf("%w: %s is %d bytes", errArchiveTooLarge, name, info.Size())
	}
	//nolint:gosec // The player deliberately selects the original-data directory; name comes from the fixed catalog.
	file, err := os.Open(path)
	if err != nil {
		return legacy.Archive{}, fmt.Errorf("opening graphics archive %s: %w", path, err)
	}
	defer func() { err = errors.Join(err, file.Close()) }()

	reader := readerFunc(func(buffer []byte) (int, error) {
		if err := ctx.Err(); err != nil {
			return 0, fmt.Errorf("reading graphics archive %s: %w", name, err)
		}
		read, readErr := file.Read(buffer)
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return read, io.EOF
			}

			return read, fmt.Errorf("reading graphics archive %s: %w", name, readErr)
		}

		return read, nil
	})
	data, err := io.ReadAll(io.LimitReader(reader, maximumArchiveBytes+1))
	if err != nil {
		return legacy.Archive{}, fmt.Errorf("reading graphics archive %s: %w", path, err)
	}
	if len(data) > maximumArchiveBytes {
		return legacy.Archive{}, fmt.Errorf("%w: %s changed while being read", errArchiveTooLarge, name)
	}
	if err := ctx.Err(); err != nil {
		return legacy.Archive{}, fmt.Errorf("loading graphics archive %s: %w", name, err)
	}
	archive, err := legacy.Decode(spec, data)
	if err != nil {
		return legacy.Archive{}, fmt.Errorf("decoding graphics archive %s: %w", name, err)
	}
	if err := ctx.Err(); err != nil {
		return legacy.Archive{}, fmt.Errorf("loading graphics archive %s: %w", name, err)
	}

	return archive, nil
}

type readerFunc func([]byte) (int, error)

func (read readerFunc) Read(buffer []byte) (int, error) {
	return read(buffer)
}
