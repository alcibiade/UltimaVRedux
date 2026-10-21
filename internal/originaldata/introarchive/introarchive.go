// Package introarchive validates original title archives without allocating
// Ebitengine resources.
package introarchive

import (
	"context"
	"errors"
	"fmt"
	"image"

	"github.com/bradhannah/UltimaVRedux/internal/originaldata/graphicsarchive"
	"github.com/bradhannah/UltimaVRedux/pkg/graphics/legacy"
)

var errInvalid = errors.New("intro archive is invalid")

//nolint:gochecknoglobals,mnd // Verified original archive dimensions are immutable.
var (
	titleEntrySizes = []image.Point{
		image.Pt(24, 3), image.Pt(40, 7), image.Pt(72, 11), image.Pt(112, 20), image.Pt(152, 32),
		image.Pt(216, 45), image.Pt(280, 61), image.Pt(104, 33), image.Pt(16, 15), image.Pt(112, 33),
	}
	britishEntrySizes = []image.Point{image.Pt(272, 62)}
	ultimaEntrySizes  = []image.Point{
		image.Pt(319, 61), image.Pt(288, 49), image.Pt(288, 49), image.Pt(288, 49), image.Pt(288, 50),
	}
	flameMaskEntrySizes      = []image.Point{image.Pt(288, 49)}
	britishPathCommandCounts = []int{856, 548, 411, 964}
)

// Set contains the decoded original archives used by the title presentation.
type Set struct {
	Title     legacy.Archive
	British   legacy.Archive
	Paths     legacy.Archive
	Ultima    legacy.Archive
	FlameMask legacy.Archive
}

// LoadAndValidate reads and validates all original title archives.
func LoadAndValidate(dataDirectory string) (Set, error) {
	return LoadAndValidateContext(context.Background(), dataDirectory)
}

// LoadAndValidateContext reads and validates all title archives while observing cancellation.
func LoadAndValidateContext(ctx context.Context, dataDirectory string) (Set, error) {
	title, err := graphicsarchive.Load(ctx, dataDirectory, "TITLE.BIT")
	if err != nil {
		return Set{}, fmt.Errorf("loading TITLE.BIT: %w", err)
	}
	british, err := graphicsarchive.Load(ctx, dataDirectory, "BRITISH.BIT")
	if err != nil {
		return Set{}, fmt.Errorf("loading BRITISH.BIT: %w", err)
	}
	paths, err := graphicsarchive.Load(ctx, dataDirectory, "BRITISH.PTH")
	if err != nil {
		return Set{}, fmt.Errorf("loading BRITISH.PTH: %w", err)
	}
	ultima, err := graphicsarchive.Load(ctx, dataDirectory, "ULTIMA.16")
	if err != nil {
		return Set{}, fmt.Errorf("loading ULTIMA.16: %w", err)
	}
	flameMask, err := graphicsarchive.Load(ctx, dataDirectory, "WD.BIT")
	if err != nil {
		return Set{}, fmt.Errorf("loading WD.BIT: %w", err)
	}
	archives := Set{Title: title, British: british, Paths: paths, Ultima: ultima, FlameMask: flameMask}
	if err := Validate(archives); err != nil {
		return Set{}, err
	}

	return archives, nil
}

// Validate verifies decoded archive structure and dimensions.
func Validate(archives Set) error {
	if err := validateArchive(archives.Title, "TITLE.BIT", legacy.PixelMask, titleEntrySizes); err != nil {
		return err
	}
	if err := validateArchive(archives.British, "BRITISH.BIT", legacy.PixelMask, britishEntrySizes); err != nil {
		return err
	}
	if err := validatePaths(archives.Paths); err != nil {
		return err
	}
	if err := validateArchive(archives.Ultima, "ULTIMA.16", legacy.PixelEGA4, ultimaEntrySizes); err != nil {
		return err
	}

	return validateArchive(archives.FlameMask, "WD.BIT", legacy.PixelMask, flameMaskEntrySizes)
}

func validateArchive(
	archive legacy.Archive,
	name string,
	pixelType legacy.PixelType,
	expectedSizes []image.Point,
) error {
	if archive.Spec.Name != name {
		return fmt.Errorf("%w: name is %q, want %q", errInvalid, archive.Spec.Name, name)
	}
	if len(archive.Entries) != len(expectedSizes) {
		return fmt.Errorf("%w: %s has %d entries, want %d", errInvalid, name, len(archive.Entries), len(expectedSizes))
	}
	for index, entry := range archive.Entries {
		if entry.PixelType != pixelType {
			return fmt.Errorf("%w: %s entry %d has pixel type %s, want %s", errInvalid, name, index, entry.PixelType, pixelType)
		}
		if entry.Missing || entry.Image == nil {
			return fmt.Errorf("%w: %s entry %d is unavailable", errInvalid, name, index)
		}
		if entry.Width != expectedSizes[index].X || entry.Height != expectedSizes[index].Y {
			return fmt.Errorf(
				"%w: %s entry %d declares dimensions %dx%d, want %dx%d",
				errInvalid, name, index, entry.Width, entry.Height, expectedSizes[index].X, expectedSizes[index].Y,
			)
		}
		actualSize := archive.Entries[index].Image.Bounds().Size()
		if actualSize != expectedSizes[index] {
			return fmt.Errorf(
				"%w: %s entry %d has dimensions %dx%d, want %dx%d",
				errInvalid, name, index, actualSize.X, actualSize.Y, expectedSizes[index].X, expectedSizes[index].Y,
			)
		}
	}

	return nil
}

func validatePaths(archive legacy.Archive) error {
	const name = "BRITISH.PTH"
	if archive.Spec.Name != name {
		return fmt.Errorf("%w: name is %q, want %q", errInvalid, archive.Spec.Name, name)
	}
	if len(archive.Entries) != len(britishPathCommandCounts) {
		return fmt.Errorf(
			"%w: %s has %d entries, want %d",
			errInvalid, name, len(archive.Entries), len(britishPathCommandCounts),
		)
	}
	for index, entry := range archive.Entries {
		if entry.Missing || entry.PixelType != legacy.PixelPath {
			return fmt.Errorf("%w: %s entry %d is unavailable", errInvalid, name, index)
		}
		if len(entry.Path) != britishPathCommandCounts[index] {
			return fmt.Errorf(
				"%w: %s entry %d has %d commands, want %d",
				errInvalid, name, index, len(entry.Path), britishPathCommandCounts[index],
			)
		}
	}

	return nil
}
