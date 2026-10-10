package sprites

import (
	"context"
	"image"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/originaldata/dungeonarchive"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationLoadDungeonSprites(t *testing.T) {
	dataDir := originaldata.Directory(t)

	archives, err := dungeonarchive.LoadAndValidate(context.Background(), dataDir)
	require.NoError(t, err)

	for wallSet := range dungeonWallSetCount {
		panel := archives.Walls[wallSet].Entries[12]
		require.NotNil(t, panel.Image)
		assert.Equal(t, image.Pt(80, 164), panel.Image.Bounds().Size())
		assert.True(t, archives.Walls[wallSet].Entries[8].Missing)
	}
	item := archives.Items.Entries[0].Image
	require.NotNil(t, item)
	assert.Equal(t, image.Pt(40, 80), item.Bounds().Size())
	for monster := range dungeonMonsterArchiveCount {
		for halfFrame, expectedSize := range dungeonMonsterHalfSizes {
			half := archives.Monsters[monster].Entries[halfFrame*2].Image
			require.NotNil(t, half, "MON%d.16 half-frame %d", monster, halfFrame)
			assert.Equal(t, expectedSize, half.Bounds().Size(), "MON%d.16 half-frame %d", monster, halfFrame)
		}
	}

	items := archives.Items
	ladder := combineDungeonMaskedImage(items.Entries[0].Image, items.Entries[1].Image)
	transparent := false
	opaqueArtwork := false
	for y := ladder.Bounds().Min.Y; y < ladder.Bounds().Max.Y; y++ {
		for x := ladder.Bounds().Min.X; x < ladder.Bounds().Max.X; x++ {
			red, green, blue, alpha := ladder.At(x, y).RGBA()
			if alpha == 0 {
				transparent = true
			} else if red != 0 || green != 0 || blue != 0 {
				opaqueArtwork = true
			}
		}
	}
	assert.True(t, transparent, "ladder item must preserve transparent background mask pixels")
	assert.True(t, opaqueArtwork, "ladder item must preserve non-black authored artwork")
}
