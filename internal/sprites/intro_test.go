package sprites

import (
	"image"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/graphics/legacy"
	"github.com/stretchr/testify/require"
)

var (
	testTitleEntrySizes = []image.Point{
		image.Pt(24, 3), image.Pt(40, 7), image.Pt(72, 11), image.Pt(112, 20), image.Pt(152, 32),
		image.Pt(216, 45), image.Pt(280, 61), image.Pt(104, 33), image.Pt(16, 15), image.Pt(112, 33),
	}
	testBritishEntrySizes = []image.Point{image.Pt(272, 62)}
	testUltimaEntrySizes  = []image.Point{
		image.Pt(319, 61), image.Pt(288, 49), image.Pt(288, 49), image.Pt(288, 49), image.Pt(288, 50),
	}
	testFlameMaskEntrySizes      = []image.Point{image.Pt(288, 49)}
	testBritishPathCommandCounts = []int{856, 548, 411, 964}
)

func TestLoadIntroSpritesReportsMissingOriginalGraphics(t *testing.T) {
	_, err := LoadIntroSprites(t.TempDir())

	require.ErrorContains(t, err, "TITLE.BIT")
}

func TestNewIntroSpritesRejectsMissingTitleEntry(t *testing.T) {
	title := testIntroArchive("TITLE.BIT", 10, legacy.PixelMask)
	title.Entries[4].Missing = true
	title.Entries[4].Image = nil

	_, err := newIntroSprites(
		title,
		testIntroArchive("BRITISH.BIT", 1, legacy.PixelMask),
		testPathArchive(),
		testIntroArchive("ULTIMA.16", 5, legacy.PixelEGA4),
		testIntroArchive("WD.BIT", 1, legacy.PixelMask),
	)

	require.ErrorContains(t, err, "entry 4 is unavailable")
}

func TestNewIntroSpritesRejectsWrongPixelType(t *testing.T) {
	title := testIntroArchive("TITLE.BIT", 10, legacy.PixelMask)
	title.Entries[4].PixelType = legacy.PixelEGA4

	_, err := newIntroSprites(
		title,
		testIntroArchive("BRITISH.BIT", 1, legacy.PixelMask),
		testPathArchive(),
		testIntroArchive("ULTIMA.16", 5, legacy.PixelEGA4),
		testIntroArchive("WD.BIT", 1, legacy.PixelMask),
	)

	require.ErrorContains(t, err, "pixel type ega-4bpp, want mask-1bpp")
}

func TestNewIntroSpritesRejectsWrongEntryDimensions(t *testing.T) {
	title := testIntroArchive("TITLE.BIT", 10, legacy.PixelMask)
	title.Entries[4].Width = 151
	title.Entries[4].Image = image.NewPaletted(image.Rect(0, 0, 151, 32), legacy.EGAPalette)

	_, err := newIntroSprites(
		title,
		testIntroArchive("BRITISH.BIT", 1, legacy.PixelMask),
		testPathArchive(),
		testIntroArchive("ULTIMA.16", 5, legacy.PixelEGA4),
		testIntroArchive("WD.BIT", 1, legacy.PixelMask),
	)

	require.ErrorContains(t, err, "entry 4 declares dimensions 151x32, want 152x32")
}

func TestNewIntroSpritesRejectsWrongSignatureStrokeLength(t *testing.T) {
	paths := testPathArchive()
	paths.Entries[2].Path = paths.Entries[2].Path[:len(paths.Entries[2].Path)-1]

	_, err := newIntroSprites(
		testIntroArchive("TITLE.BIT", 10, legacy.PixelMask),
		testIntroArchive("BRITISH.BIT", 1, legacy.PixelMask),
		paths,
		testIntroArchive("ULTIMA.16", 5, legacy.PixelEGA4),
		testIntroArchive("WD.BIT", 1, legacy.PixelMask),
	)

	require.ErrorContains(t, err, "entry 2 has 410 commands, want 411")
}

func TestNewIntroSpritesRejectsWrongFlameMaskDimensions(t *testing.T) {
	flameMask := testIntroArchive("WD.BIT", 1, legacy.PixelMask)
	flameMask.Entries[0].Height = 48
	flameMask.Entries[0].Image = image.NewPaletted(image.Rect(0, 0, 288, 48), legacy.EGAPalette)

	_, err := newIntroSprites(
		testIntroArchive("TITLE.BIT", 10, legacy.PixelMask),
		testIntroArchive("BRITISH.BIT", 1, legacy.PixelMask),
		testPathArchive(),
		testIntroArchive("ULTIMA.16", 5, legacy.PixelEGA4),
		flameMask,
	)

	require.ErrorContains(t, err, "WD.BIT entry 0 declares dimensions 288x48, want 288x49")
}

func testIntroArchive(name string, count int, pixelType legacy.PixelType) legacy.Archive {
	sizes := map[string][]image.Point{
		"TITLE.BIT":   testTitleEntrySizes,
		"BRITISH.BIT": testBritishEntrySizes,
		"ULTIMA.16":   testUltimaEntrySizes,
		"WD.BIT":      testFlameMaskEntrySizes,
	}[name]
	entries := make([]legacy.Entry, count)
	for index := range entries {
		size := sizes[index]
		entries[index] = legacy.Entry{
			Index: index, Width: size.X, Height: size.Y, StoredWidth: (size.X + 7) &^ 7, PixelType: pixelType,
			Image: image.NewPaletted(image.Rect(0, 0, size.X, size.Y), legacy.EGAPalette),
		}
	}

	return legacy.Archive{Spec: legacy.FileSpec{Name: name}, Entries: entries}
}

func testPathArchive() legacy.Archive {
	entries := make([]legacy.Entry, 4)
	for index := range entries {
		entries[index] = legacy.Entry{
			Index: index, PixelType: legacy.PixelPath, Path: make([]legacy.PathCommand, testBritishPathCommandCounts[index]),
		}
	}

	return legacy.Archive{Spec: legacy.FileSpec{Name: "BRITISH.PTH"}, Entries: entries}
}
