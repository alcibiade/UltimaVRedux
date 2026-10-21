package sprites

import (
	"errors"
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/originaldata/dungeonarchive"
	"github.com/bradhannah/UltimaVRedux/pkg/graphics/legacy"
)

var (
	dungeonPanelSizes       = dungeonarchive.PanelSizes()
	dungeonItemSizes        = dungeonarchive.ItemSizes()
	dungeonMonsterHalfSizes = dungeonarchive.MonsterHalfSizes()
)

func TestLoadDungeonSpritesReportsMissingOriginalGraphics(t *testing.T) {
	_, err := LoadDungeonSprites(t.TempDir())

	require.ErrorContains(t, err, "DNG1.16")
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestNewDungeonSpritesRejectsWrongArchiveEntryCounts(t *testing.T) {
	t.Run("wall panels", func(t *testing.T) {
		walls, items, monsters := syntheticDungeonArchives()
		walls[1].Entries = walls[1].Entries[:dungeonPanelCount-1]

		_, err := newDungeonSprites(walls, items, monsters)

		require.ErrorContains(t, err, "DNG2.16 has 27 entries, want 28")
	})

	t.Run("item pairs", func(t *testing.T) {
		walls, items, monsters := syntheticDungeonArchives()
		items.Entries = items.Entries[:dungeonItemCount*2-1]

		_, err := newDungeonSprites(walls, items, monsters)

		require.ErrorContains(t, err, "ITEMS.16 has 39 entries, want 40")
	})
	t.Run("monster half-frame pairs", func(t *testing.T) {
		walls, items, monsters := syntheticDungeonArchives()
		monsters[7].Entries = monsters[7].Entries[:dungeonMonsterHalfCount*2-1]

		_, err := newDungeonSprites(walls, items, monsters)

		require.ErrorContains(t, err, "MON7.16 has 11 entries, want 12")
	})
}

func TestNewDungeonSpritesValidatesKnownMissingPanels(t *testing.T) {
	t.Run("required missing panel is present", func(t *testing.T) {
		walls, items, monsters := syntheticDungeonArchives()
		walls[0].Entries[8] = syntheticDungeonEntry(8, image.Pt(8, 164), legacy.PixelEGA4)

		_, err := newDungeonSprites(walls, items, monsters)

		require.ErrorContains(t, err, "DNG1.16 entry 8 is present, want missing")
	})

	t.Run("unexpected panel is missing", func(t *testing.T) {
		walls, items, monsters := syntheticDungeonArchives()
		walls[2].Entries[7].Missing = true
		walls[2].Entries[7].Image = nil

		_, err := newDungeonSprites(walls, items, monsters)

		require.ErrorContains(t, err, "DNG3.16 entry 7 is unavailable")
	})
}

func TestNewDungeonSpritesValidatesOriginalDimensions(t *testing.T) {
	t.Run("wall panel", func(t *testing.T) {
		walls, items, monsters := syntheticDungeonArchives()
		walls[0].Entries[9].Width = 55
		walls[0].Entries[9].Image = image.NewPaletted(image.Rect(0, 0, 55, 164), legacy.EGAPalette)

		_, err := newDungeonSprites(walls, items, monsters)

		require.ErrorContains(t, err, "DNG1.16 entry 9 declares dimensions 55x164, want 56x164")
	})

	t.Run("item mask", func(t *testing.T) {
		walls, items, monsters := syntheticDungeonArchives()
		items.Entries[9].Height = 79
		items.Entries[9].Image = image.NewPaletted(image.Rect(0, 0, 40, 79), dungeonMaskPalette())

		_, err := newDungeonSprites(walls, items, monsters)

		require.ErrorContains(t, err, "ITEMS.16 entry 9 declares dimensions 40x79, want 40x80")
	})
	t.Run("monster half-frame", func(t *testing.T) {
		walls, items, monsters := syntheticDungeonArchives()
		monsters[3].Entries[10].Width = 7
		monsters[3].Entries[10].Image = image.NewPaletted(image.Rect(0, 0, 7, 6), legacy.EGAPalette)

		_, err := newDungeonSprites(walls, items, monsters)

		require.ErrorContains(t, err, "MON3.16 entry 10 declares dimensions 7x6, want 8x6")
	})
}

func TestDungeonSpritesAccessors(t *testing.T) {
	walls, items, monsters := syntheticDungeonArchives()
	sprites, err := newDungeonSprites(walls, items, monsters)
	require.NoError(t, err)

	panel, ok := sprites.Panel(2, 12)
	require.True(t, ok)
	assert.Equal(t, image.Pt(80, 164), panel.Bounds().Size())
	for _, missing := range []int{8, 24} {
		panel, ok = sprites.Panel(0, missing)
		assert.False(t, ok)
		assert.Nil(t, panel)
	}
	for _, coordinates := range [][2]int{{-1, 0}, {3, 0}, {0, -1}, {0, 28}} {
		panel, ok = sprites.Panel(coordinates[0], coordinates[1])
		assert.False(t, ok)
		assert.Nil(t, panel)
	}

	item, ok := sprites.Item(19)
	require.True(t, ok)
	assert.Equal(t, image.Pt(16, 16), item.Bounds().Size())
	for _, index := range []int{-1, 20} {
		item, ok = sprites.Item(index)
		assert.False(t, ok)
		assert.Nil(t, item)
	}

	monster, ok := sprites.Monster(7, 5)
	require.True(t, ok)
	assert.Equal(t, image.Pt(8, 6), monster.Bounds().Size())
	for _, coordinates := range [][2]int{{-1, 0}, {8, 0}, {0, -1}, {0, 6}} {
		monster, ok = sprites.Monster(coordinates[0], coordinates[1])
		assert.False(t, ok)
		assert.Nil(t, monster)
	}
}

func TestDungeonSpritesCombinesItemColorAndMask(t *testing.T) {
	_, items, _ := syntheticDungeonArchives()
	items.Entries[0].Image.SetColorIndex(0, 0, 12)
	items.Entries[0].Image.SetColorIndex(1, 0, 10)
	items.Entries[1].Image.SetColorIndex(1, 0, 1)

	item := combineDungeonMaskedImage(items.Entries[0].Image, items.Entries[1].Image)

	wantRed, wantGreen, wantBlue, wantAlpha := legacy.EGAPalette[12].RGBA()
	red, green, blue, alpha := item.At(0, 0).RGBA()
	assert.Equal(t, wantRed, red)
	assert.Equal(t, wantGreen, green)
	assert.Equal(t, wantBlue, blue)
	assert.Equal(t, wantAlpha, alpha)

	red, green, blue, alpha = item.At(1, 0).RGBA()
	assert.Zero(t, red)
	assert.Zero(t, green)
	assert.Zero(t, blue)
	assert.Zero(t, alpha)
}

func TestDungeonPanelDeltaPreservesOnlyChangedSpecialWallPixels(t *testing.T) {
	normal := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	special := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for y := range 2 {
		for x := range 3 {
			normal.SetNRGBA(x, y, color.NRGBA{G: 0x80, A: 0xff})
			special.SetNRGBA(x, y, color.NRGBA{G: 0x80, A: 0xff})
		}
	}
	special.SetNRGBA(1, 0, color.NRGBA{R: 0xff, A: 0xff})
	special.SetNRGBA(2, 1, color.NRGBA{A: 0xff})

	delta := makeDungeonPanelDelta(special, normal)

	assert.Equal(t, color.NRGBA{}, delta.NRGBAAt(0, 0))
	assert.Equal(t, color.NRGBA{R: 0xff, A: 0xff}, delta.NRGBAAt(1, 0))
	assert.Equal(t, color.NRGBA{A: 0xff}, delta.NRGBAAt(2, 1))
	for y := range 2 {
		for x := range 3 {
			got := color.NRGBAModel.Convert(normal.At(x, y)).(color.NRGBA)
			if deltaColor := delta.NRGBAAt(x, y); deltaColor.A != 0 {
				got = deltaColor
			}
			want := color.NRGBAModel.Convert(special.At(x, y)).(color.NRGBA)
			assert.Equal(t, want, got)
		}
	}
}

func TestDungeonSpritesCachesSpecialWallPanelDeltas(t *testing.T) {
	walls, items, monsters := syntheticDungeonArchives()
	for _, pair := range dungeonSpecialWallPanelPairs {
		walls[2].Entries[pair.special].Image.SetColorIndex(0, 0, 10)
		walls[2].Entries[pair.normal].Image.SetColorIndex(0, 0, 10)
	}
	sprites, err := newDungeonSprites(walls, items, monsters)
	require.NoError(t, err)
	deltas := makeDungeonSpecialWallPanelDeltas(walls)

	for _, pair := range dungeonSpecialWallPanelPairs {
		delta, ok := sprites.SpecialWallPanelDelta(2, pair.special)
		require.True(t, ok, "special panel %d", pair.special)
		assert.Equal(t, dungeonPanelSizes[pair.special], delta.Bounds().Size())
		assert.Zero(t, deltas[2][pair.special].NRGBAAt(0, 0).A,
			"special panel %d must subtract its paired normal panel", pair.special)
		second, ok := sprites.SpecialWallPanelDelta(2, pair.special)
		assert.True(t, ok)
		assert.Same(t, delta, second)
	}
	missing, ok := sprites.SpecialWallPanelDelta(2, 24)
	assert.False(t, ok)
	assert.Nil(t, missing)
}

func syntheticDungeonArchives() (
	[dungeonWallSetCount]legacy.Archive,
	legacy.Archive,
	[dungeonMonsterArchiveCount]legacy.Archive,
) {
	var walls [dungeonWallSetCount]legacy.Archive
	for wallSet := range walls {
		entries := make([]legacy.Entry, dungeonPanelCount)
		for panel, size := range dungeonPanelSizes {
			if size == (image.Point{}) {
				entries[panel] = legacy.Entry{Index: panel, PixelType: legacy.PixelEGA4, Missing: true}
				continue
			}
			entries[panel] = syntheticDungeonEntry(panel, size, legacy.PixelEGA4)
		}
		walls[wallSet] = legacy.Archive{
			Spec:    legacy.FileSpec{Name: dungeonArchiveName(wallSet), Format: legacy.FormatShape4},
			Entries: entries,
		}
	}

	itemEntries := make([]legacy.Entry, dungeonItemCount*2)
	for item, size := range dungeonItemSizes {
		colorIndex := item * 2
		itemEntries[colorIndex] = syntheticDungeonEntry(colorIndex, size, legacy.PixelEGA4)
		itemEntries[colorIndex+1] = syntheticDungeonEntry(colorIndex+1, size, legacy.PixelMask)
	}

	items := legacy.Archive{
		Spec:    legacy.FileSpec{Name: "ITEMS.16", Format: legacy.FormatPaired},
		Entries: itemEntries,
	}

	var monsters [dungeonMonsterArchiveCount]legacy.Archive
	for monster := range monsters {
		entries := make([]legacy.Entry, dungeonMonsterHalfCount*2)
		for halfFrame, size := range dungeonMonsterHalfSizes {
			colorIndex := halfFrame * 2
			entries[colorIndex] = syntheticDungeonEntry(colorIndex, size, legacy.PixelEGA4)
			entries[colorIndex+1] = syntheticDungeonEntry(colorIndex+1, size, legacy.PixelMask)
		}
		monsters[monster] = legacy.Archive{
			Spec:    legacy.FileSpec{Name: dungeonMonsterArchiveName(monster), Format: legacy.FormatPaired},
			Entries: entries,
		}
	}

	return walls, items, monsters
}

func syntheticDungeonEntry(index int, size image.Point, pixelType legacy.PixelType) legacy.Entry {
	palette := legacy.EGAPalette
	if pixelType == legacy.PixelMask {
		palette = dungeonMaskPalette()
	}

	return legacy.Entry{
		Index:       index,
		Width:       size.X,
		Height:      size.Y,
		StoredWidth: (size.X + 7) &^ 7,
		PixelType:   pixelType,
		Image:       image.NewPaletted(image.Rectangle{Max: size}, palette),
	}
}

func dungeonMaskPalette() color.Palette {
	return color.Palette{color.Transparent, color.White}
}
