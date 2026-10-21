//nolint:mnd // Archive counts, missing slots, and dimensions are verified original asset facts.
package sprites

import (
	"context"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/originaldata/dungeonarchive"
	"github.com/bradhannah/UltimaVRedux/pkg/graphics/legacy"
)

const (
	dungeonWallSetCount        = dungeonarchive.WallSetCount
	dungeonPanelCount          = dungeonarchive.PanelCount
	dungeonItemCount           = dungeonarchive.ItemCount
	dungeonMonsterArchiveCount = dungeonarchive.MonsterArchiveCount
	dungeonMonsterHalfCount    = dungeonarchive.MonsterHalfCount
)

type dungeonSpecialWallPanelPair struct {
	special int
	normal  int
}

//nolint:gochecknoglobals // Original DNG panel indices define immutable special-wall pairings.
var dungeonSpecialWallPanelPairs = [...]dungeonSpecialWallPanelPair{
	{special: 20, normal: 0},
	{special: 21, normal: 1},
	{special: 22, normal: 2},
	{special: 23, normal: 3},
	{special: 25, normal: 9},
	{special: 26, normal: 10},
	{special: 27, normal: 11},
}

// DungeonSprites owns the original first-person dungeon panel, item, and monster assets.
// Images returned by its accessors are shared scene assets and must not be modified.
type DungeonSprites struct {
	panels                 [dungeonWallSetCount][dungeonPanelCount]*ebiten.Image
	specialWallPanelDeltas [dungeonWallSetCount][dungeonPanelCount]*ebiten.Image
	items                  [dungeonItemCount]*ebiten.Image
	monsters               [dungeonMonsterArchiveCount][dungeonMonsterHalfCount]*ebiten.Image
}

// LoadDungeonSprites decodes original dungeon graphics from the configured game data directory.
func LoadDungeonSprites(dataDir string) (*DungeonSprites, error) {
	archives, err := dungeonarchive.LoadAndValidate(context.Background(), dataDir)
	if err != nil {
		return nil, fmt.Errorf("loading dungeon archives: %w", err)
	}

	return newDungeonSprites(archives.Walls, archives.Items, archives.Monsters)
}

// Panel returns one shared wall panel. Missing or out-of-range entries return false.
func (s *DungeonSprites) Panel(wallSet, panel int) (*ebiten.Image, bool) {
	if s == nil || wallSet < 0 || wallSet >= dungeonWallSetCount || panel < 0 || panel >= dungeonPanelCount {
		return nil, false
	}

	result := s.panels[wallSet][panel]

	return result, result != nil
}

// SpecialWallPanelDelta returns transparent pixels that differ between paired special and normal wall panels.
func (s *DungeonSprites) SpecialWallPanelDelta(wallSet, specialPanel int) (*ebiten.Image, bool) {
	if s == nil || wallSet < 0 || wallSet >= dungeonWallSetCount ||
		specialPanel < 0 || specialPanel >= dungeonPanelCount {
		return nil, false
	}
	delta := s.specialWallPanelDeltas[wallSet][specialPanel]

	return delta, delta != nil
}

// Item returns one shared transparent item image. Out-of-range entries return false.
func (s *DungeonSprites) Item(item int) (*ebiten.Image, bool) {
	if s == nil || item < 0 || item >= dungeonItemCount {
		return nil, false
	}

	return s.items[item], true
}

// Monster returns one shared transparent monster half-frame. Out-of-range entries return false.
func (s *DungeonSprites) Monster(monster, halfFrame int) (*ebiten.Image, bool) {
	if s == nil || monster < 0 || monster >= dungeonMonsterArchiveCount ||
		halfFrame < 0 || halfFrame >= dungeonMonsterHalfCount {
		return nil, false
	}

	return s.monsters[monster][halfFrame], true
}

func newDungeonSprites(
	walls [dungeonWallSetCount]legacy.Archive,
	items legacy.Archive,
	monsters [dungeonMonsterArchiveCount]legacy.Archive,
) (*DungeonSprites, error) {
	if err := dungeonarchive.Validate(dungeonarchive.Set{Walls: walls, Items: items, Monsters: monsters}); err != nil {
		return nil, fmt.Errorf("validating dungeon sprite archives: %w", err)
	}

	sprites := &DungeonSprites{
		panels:                 [dungeonWallSetCount][dungeonPanelCount]*ebiten.Image{},
		specialWallPanelDeltas: [dungeonWallSetCount][dungeonPanelCount]*ebiten.Image{},
		items:                  [dungeonItemCount]*ebiten.Image{},
		monsters:               [dungeonMonsterArchiveCount][dungeonMonsterHalfCount]*ebiten.Image{},
	}
	for wallSet, archive := range walls {
		for panel, entry := range archive.Entries {
			if !entry.Missing {
				sprites.panels[wallSet][panel] = ebiten.NewImageFromImage(entry.Image)
			}
		}
	}
	for wallSet, deltas := range makeDungeonSpecialWallPanelDeltas(walls) {
		for panel, delta := range deltas {
			if delta != nil {
				sprites.specialWallPanelDeltas[wallSet][panel] = ebiten.NewImageFromImage(delta)
			}
		}
	}
	for item := range dungeonItemCount {
		colorEntry := items.Entries[item*2]
		maskEntry := items.Entries[item*2+1]
		sprites.items[item] = ebiten.NewImageFromImage(combineDungeonMaskedImage(colorEntry.Image, maskEntry.Image))
	}
	for monster, archive := range monsters {
		for halfFrame := range dungeonMonsterHalfCount {
			colorEntry := archive.Entries[halfFrame*2]
			maskEntry := archive.Entries[halfFrame*2+1]
			sprites.monsters[monster][halfFrame] = ebiten.NewImageFromImage(
				combineDungeonMaskedImage(colorEntry.Image, maskEntry.Image),
			)
		}
	}

	return sprites, nil
}

func makeDungeonSpecialWallPanelDeltas(
	walls [dungeonWallSetCount]legacy.Archive,
) [dungeonWallSetCount][dungeonPanelCount]*image.NRGBA {
	var deltas [dungeonWallSetCount][dungeonPanelCount]*image.NRGBA
	for wallSet, archive := range walls {
		for _, pair := range dungeonSpecialWallPanelPairs {
			deltas[wallSet][pair.special] = makeDungeonPanelDelta(
				archive.Entries[pair.special].Image,
				archive.Entries[pair.normal].Image,
			)
		}
	}

	return deltas
}

func makeDungeonPanelDelta(special, normal image.Image) *image.NRGBA {
	if special.Bounds().Size() != normal.Bounds().Size() {
		panic("special and normal dungeon panels have mismatched dimensions")
	}
	bounds := special.Bounds()
	delta := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := range bounds.Dy() {
		for x := range bounds.Dx() {
			specialColor := dungeonPanelNRGBA(special, bounds.Min.X+x, bounds.Min.Y+y)
			normalColor := dungeonPanelNRGBA(normal, bounds.Min.X+x, bounds.Min.Y+y)
			if specialColor != normalColor {
				delta.SetNRGBA(x, y, specialColor)
			}
		}
	}

	return delta
}

func dungeonPanelNRGBA(source image.Image, x, y int) color.NRGBA {
	converted := color.NRGBAModel.Convert(source.At(x, y))
	value, ok := converted.(color.NRGBA)
	if !ok {
		panic("NRGBA color model did not produce NRGBA")
	}

	return value
}

func combineDungeonMaskedImage(colorImage, maskImage *image.Paletted) *image.NRGBA {
	colorBounds := colorImage.Bounds()
	maskBounds := maskImage.Bounds()
	combined := image.NewNRGBA(image.Rect(0, 0, colorBounds.Dx(), colorBounds.Dy()))
	for y := range colorBounds.Dy() {
		for x := range colorBounds.Dx() {
			if maskImage.ColorIndexAt(maskBounds.Min.X+x, maskBounds.Min.Y+y) == 0 {
				combined.Set(x, y, colorImage.At(colorBounds.Min.X+x, colorBounds.Min.Y+y))
			}
		}
	}

	return combined
}
