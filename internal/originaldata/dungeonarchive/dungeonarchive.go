// Package dungeonarchive validates original first-person dungeon graphics
// without allocating Ebitengine resources.
package dungeonarchive

import (
	"context"
	"errors"
	"fmt"
	"image"

	"github.com/bradhannah/UltimaVRedux/internal/graphics/legacy"
	"github.com/bradhannah/UltimaVRedux/internal/originaldata/graphicsarchive"
)

const (
	// WallSetCount is the number of original dungeon wall archives.
	WallSetCount = 3
	// PanelCount is the number of entries in each wall archive.
	PanelCount = 28
	// ItemCount is the number of image/mask pairs in ITEMS.16.
	ItemCount = 20
	// MonsterArchiveCount is the number of original monster archives.
	MonsterArchiveCount = 8
	// MonsterHalfCount is the number of image/mask pairs in each monster archive.
	MonsterHalfCount = 6
	pairedEntryCount = 2
)

var errInvalid = errors.New("dungeon graphics archive is invalid")

//nolint:gochecknoglobals,mnd // Verified original dimensions are immutable validation tables.
var (
	panelSizes = [PanelCount]image.Point{
		image.Pt(24, 164), image.Pt(32, 164), image.Pt(16, 164), image.Pt(8, 164),
		image.Pt(24, 164), image.Pt(32, 164), image.Pt(16, 164), image.Pt(8, 164),
		image.Pt(0, 0), image.Pt(56, 164), image.Pt(24, 164), image.Pt(8, 164),
		image.Pt(80, 164), image.Pt(56, 164), image.Pt(24, 164), image.Pt(8, 164),
		image.Pt(24, 164), image.Pt(32, 164), image.Pt(16, 164), image.Pt(8, 164),
		image.Pt(24, 164), image.Pt(32, 164), image.Pt(16, 164), image.Pt(8, 164),
		image.Pt(0, 0), image.Pt(56, 164), image.Pt(24, 164), image.Pt(8, 164),
	}
	itemSizes = [ItemCount]image.Point{
		image.Pt(40, 80), image.Pt(24, 56), image.Pt(16, 24), image.Pt(8, 8),
		image.Pt(40, 80), image.Pt(24, 56), image.Pt(16, 24), image.Pt(8, 8),
		image.Pt(40, 24), image.Pt(24, 32), image.Pt(16, 16), image.Pt(8, 8),
		image.Pt(40, 24), image.Pt(24, 32), image.Pt(16, 16), image.Pt(8, 8),
		image.Pt(40, 24), image.Pt(24, 32), image.Pt(16, 16), image.Pt(16, 16),
	}
	monsterHalfSizes = [MonsterHalfCount]image.Point{
		image.Pt(24, 66), image.Pt(16, 25), image.Pt(8, 6),
		image.Pt(24, 66), image.Pt(16, 25), image.Pt(8, 6),
	}
)

// Set contains all decoded archives required by first-person dungeon rendering.
type Set struct {
	Walls    [WallSetCount]legacy.Archive
	Items    legacy.Archive
	Monsters [MonsterArchiveCount]legacy.Archive
}

// LoadAndValidate performs bounded CPU-only loading and exact structural validation.
func LoadAndValidate(ctx context.Context, dataDirectory string) (Set, error) {
	var result Set
	for wallSet := range result.Walls {
		name := WallArchiveName(wallSet)
		archive, err := graphicsarchive.Load(ctx, dataDirectory, name)
		if err != nil {
			return Set{}, fmt.Errorf("loading dungeon wall set %d: %w", wallSet, err)
		}
		result.Walls[wallSet] = archive
	}
	items, err := graphicsarchive.Load(ctx, dataDirectory, "ITEMS.16")
	if err != nil {
		return Set{}, fmt.Errorf("loading dungeon items: %w", err)
	}
	result.Items = items
	for monster := range result.Monsters {
		name := MonsterArchiveName(monster)
		archive, loadErr := graphicsarchive.Load(ctx, dataDirectory, name)
		if loadErr != nil {
			return Set{}, fmt.Errorf("loading dungeon monster set %d: %w", monster, loadErr)
		}
		result.Monsters[monster] = archive
	}
	if err := Validate(result); err != nil {
		return Set{}, err
	}

	return result, nil
}

// Validate verifies exact entry types, availability, and dimensions.
func Validate(archives Set) error {
	for wallSet, archive := range archives.Walls {
		if err := validatePanels(archive, WallArchiveName(wallSet)); err != nil {
			return err
		}
	}
	if err := validatePairedArchive(archives.Items, "ITEMS.16", itemSizes[:]); err != nil {
		return err
	}
	for monster, archive := range archives.Monsters {
		if err := validatePairedArchive(archive, MonsterArchiveName(monster), monsterHalfSizes[:]); err != nil {
			return err
		}
	}

	return nil
}

// WallArchiveName returns the catalog name for a wall set index.
func WallArchiveName(wallSet int) string {
	return fmt.Sprintf("DNG%d.16", wallSet+1)
}

// MonsterArchiveName returns the catalog name for a monster set index.
func MonsterArchiveName(monster int) string {
	return fmt.Sprintf("MON%d.16", monster)
}

func validatePanels(archive legacy.Archive, name string) error {
	if archive.Spec.Name != name {
		return fmt.Errorf("%w: name is %q, want %q", errInvalid, archive.Spec.Name, name)
	}
	if len(archive.Entries) != PanelCount {
		return fmt.Errorf("%w: %s has %d entries, want %d", errInvalid, name, len(archive.Entries), PanelCount)
	}
	for panel, entry := range archive.Entries {
		if entry.PixelType != legacy.PixelEGA4 {
			return fmt.Errorf(
				"%w: %s entry %d has pixel type %s, want %s",
				errInvalid, name, panel, entry.PixelType, legacy.PixelEGA4,
			)
		}
		if panelSizes[panel] == image.Pt(0, 0) {
			if !entry.Missing || entry.Image != nil {
				return fmt.Errorf("%w: %s entry %d is present, want missing", errInvalid, name, panel)
			}

			continue
		}
		if err := validateEntry(name, panel, entry, panelSizes[panel]); err != nil {
			return err
		}
	}

	return nil
}

func validatePairedArchive(archive legacy.Archive, name string, sizes []image.Point) error {
	if archive.Spec.Name != name {
		return fmt.Errorf("%w: name is %q, want %q", errInvalid, archive.Spec.Name, name)
	}
	expectedEntries := len(sizes) * pairedEntryCount
	if len(archive.Entries) != expectedEntries {
		return fmt.Errorf("%w: %s has %d entries, want %d", errInvalid, name, len(archive.Entries), expectedEntries)
	}
	for imageIndex, size := range sizes {
		colorIndex := imageIndex * pairedEntryCount
		colorEntry := archive.Entries[colorIndex]
		if colorEntry.PixelType != legacy.PixelEGA4 {
			return fmt.Errorf(
				"%w: %s entry %d has pixel type %s, want %s",
				errInvalid, name, colorIndex, colorEntry.PixelType, legacy.PixelEGA4,
			)
		}
		if err := validateEntry(name, colorIndex, colorEntry, size); err != nil {
			return err
		}
		maskIndex := colorIndex + 1
		maskEntry := archive.Entries[maskIndex]
		if maskEntry.PixelType != legacy.PixelMask {
			return fmt.Errorf(
				"%w: %s entry %d has pixel type %s, want %s",
				errInvalid, name, maskIndex, maskEntry.PixelType, legacy.PixelMask,
			)
		}
		if err := validateEntry(name, maskIndex, maskEntry, size); err != nil {
			return err
		}
	}

	return nil
}

func validateEntry(name string, index int, entry legacy.Entry, expected image.Point) error {
	if entry.Missing || entry.Image == nil {
		return fmt.Errorf("%w: %s entry %d is unavailable", errInvalid, name, index)
	}
	if entry.Width != expected.X || entry.Height != expected.Y {
		return fmt.Errorf(
			"%w: %s entry %d declares dimensions %dx%d, want %dx%d",
			errInvalid, name, index, entry.Width, entry.Height, expected.X, expected.Y,
		)
	}
	actual := entry.Image.Bounds().Size()
	if actual != expected {
		return fmt.Errorf(
			"%w: %s entry %d has dimensions %dx%d, want %dx%d",
			errInvalid, name, index, actual.X, actual.Y, expected.X, expected.Y,
		)
	}

	return nil
}
