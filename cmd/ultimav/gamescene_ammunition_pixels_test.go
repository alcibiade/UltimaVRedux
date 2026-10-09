package main

import (
	"image"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/ui/mainscreen"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

// Compare every supplies glyph with an unobstructed native-coordinate reference.
// Rectangle containment alone missed the border raster occupying part of the
// nominal content rectangle, including the bottom strokes of LOW and OUT.
func TestAmmunitionRenderedGlyphsRemainCompleteInsideRasterizedFrame(t *testing.T) {
	manager := display.GetManager()
	saved := *manager
	t.Cleanup(func() { *manager = saved })
	// The CPU right-rail tests cover every supported resolution. These two
	// raster samples cover the smallest layout and its fractional-width neighbor.
	for _, size := range []image.Point{{1280, 720}, {1366, 768}} {
		t.Run(size.String(), func(t *testing.T) {
			bounds := logicalGameBounds(size.X, size.Y)
			manager.PublishLogicalSize(bounds.Dx(), bounds.Dy())
			layout := calculateRightRailLayout(bounds)
			_, output := newMainOutput(bounds.Dx())
			require.Equal(t, 11, output.GetLineCapacity(layout.outputContent.Dy()))
			frame, options := sprites.NewBorderSprites().VeryPixelatedRoundedBlueWhite.CreateSizedAndScaledBorderSprite(
				borderWidthScaling, rectanglePlacement(bounds, layout.provisionBorder))
			// Include a full row of guard pixels around the complete panel so the
			// unframed oracle cannot silently crop an overflowing glyph. Nonzero
			// bounds preserve the production text/border coordinates and pixel ties.
			sample := layout.provisionBorder.Inset(-mainscreen.ProvisionAmmoRowHeight(bounds.Dx()))
			actual := ebiten.NewImageWithOptions(sample, &ebiten.NewImageOptions{Unmanaged: true})
			reference := ebiten.NewImageWithOptions(sample, &ebiten.NewImageOptions{Unmanaged: true})
			defer actual.Deallocate()
			defer reference.Deallocate()
			summary := mainscreen.NewProvisionSummaryForWidth(bounds.Dx())
			state := &game_state.GameState{DateTime: datetime.UltimaDate{Year: 65535, Month: 13, Day: 28, Hour: 12, Minute: 59}}
			state.PartyState.Inventory.Provisions.Food.Set(9999)
			state.PartyState.Inventory.Provisions.Torches.Set(99)
			state.PartyState.Inventory.Provisions.Keys.Set(99)
			state.PartyState.Inventory.Provisions.Gems.Set(99)
			state.PartyState.Inventory.Gold.Set(9999)
			state.PartyState.Karma.Value = 99
			// Sample the whole panel, including all surrounding chrome, so neither
			// a top-row overlap nor a bottom-row descender can evade the comparison.
			pixelBytes := sample.Dx() * sample.Dy() * 4
			chromePixels, actualPixels, referencePixels := make([]byte, pixelBytes), make([]byte, pixelBytes), make([]byte, pixelBytes)
			actual.DrawImage(frame, options)
			actual.ReadPixels(chromePixels)
			// Both columns exercise healthy, LOW and OUT; single/two-digit counts,
			// the LOW threshold, and long healthy counts have distinct glyph shapes.
			for _, counts := range []mainscreen.AmmunitionCounts{{21, 99}, {1, 20}, {0, 0}} {
				state.PartyState.Inventory.Equipment.Set(references.Arrows, counts[0])
				state.PartyState.Inventory.Equipment.Set(references.Quarrels, counts[1])
				actual.Clear()
				reference.Clear()
				actual.DrawImage(frame, options)
				summary.SetBounds(layout.provisionContent)
				summary.Draw(state, actual)
				actual.ReadPixels(actualPixels)
				// Identical coordinates and typography, with no frame nearby.
				summary.Draw(state, reference)
				reference.ReadPixels(referencePixels)
				inkPixels := 0
				for index := 0; index < len(referencePixels); index += 4 {
					if referencePixels[index+3] == 0 {
						continue
					}
					inkPixels++
					x, y := sample.Min.X+(index/4)%sample.Dx(), sample.Min.Y+(index/4)/sample.Dx()
					require.True(t, image.Pt(x, y).In(layout.provisionBorder), "complete supplies glyph escaped the panel at %d,%d", x, y)
					require.Zero(t, chromePixels[index+3], "supplies with ammo %v pixel (%d,%d) must not touch the frame", counts, x, y)
					require.Equal(t, referencePixels[index:index+4], actualPixels[index:index+4],
						"ammo %v pixel (%d,%d) must retain its complete reference glyph", counts, x, y)
				}
				require.Positive(t, inkPixels, "the comparison must include actual glyphs")
			}
		})
	}
}
