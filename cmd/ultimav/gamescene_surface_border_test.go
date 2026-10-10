package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/environment"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestSurfaceBorderApertureAndWind(t *testing.T) {
	manager := display.GetManager()
	saved := *manager
	t.Cleanup(func() { *manager = saved })
	scene := newSurfaceCameraTestScene(t)
	wind := NewWindDisplay()
	for _, size := range []image.Point{{X: 640, Y: 360}, {X: 1001, Y: 701}, {X: 1280, Y: 720}, {X: 1367, Y: 769}, {X: 1921, Y: 1081}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			manager.PublishLogicalSize(size.X, size.Y)
			outer, aperture := surfaceMapBorderGeometry()
			scene.borders.surfaceAperture = aperture
			require.Equal(t, scene.mapViewportRectangle(), outer)
			require.True(t, aperture.In(outer))
			require.False(t, aperture.Empty())
			border := sprites.NewBorderSprites()
			frame, op := border.VeryPixelatedRoundedBlueWhite.CreateSizedAndScaledBorderSprite(borderWidthScaling, gameScreenPercents)
			screen := newPresentationRaster(size.X, size.Y)
			defer screen.Deallocate()
			baseline := newPresentationRaster(size.X, size.Y)
			defer baseline.Deallocate()
			baseline.Fill(color.Black)
			baseline.DrawImage(frame, op)
			want := make([]byte, size.X*size.Y*4)
			baseline.ReadPixels(want)
			got := make([]byte, len(want))
			for _, state := range []struct{ enabled, active bool }{{false, false}, {true, false}, {true, true}} {
				scene.gameConfig.GameplayConfiguration.SmoothSurfaceMovement = state.enabled
				scene.surfaceCamera = surfaceCameraProjection{active: state.active, x: 7.25, y: -3.5}
				scene.screenShakeOffsetX, scene.screenShakeOffsetY = -9.5, 8.25
				screen.Fill(color.Black)
				target := scene.worldRenderTarget(screen)
				require.Equal(t, aperture, target.Bounds())
				// Saturate the entire world target, including overscan and VFX. This
				// is stronger than a single tile or projectile crossing the frame.
				target.Fill(color.RGBA{R: 255, G: 255, A: 255})
				screen.DrawImage(frame, op)
				screen.ReadPixels(got)
				for y := 0; y < size.Y; y++ {
					for x := 0; x < size.X; x++ {
						if image.Pt(x, y).In(aperture) {
							continue
						}
						i := (y*size.X + x) * 4
						if !assert.Equal(t, want[i:i+4], got[i:i+4], "world leak at %d,%d state=%+v", x, y, state) {
							return
						}
					}
				}
			}
			wind.Update()
			stroke, run := 0, 0
			for row := aperture.Max.Y; row < outer.Max.Y; row++ {
				if frame.At(outer.Dx()/2, row) == ucolor.EGAWhite.ToRGBA() {
					run++
					stroke = max(stroke, run)
				} else {
					run = 0
				}
			}
			require.Positive(t, stroke)
			assert.Equal(t, stroke, wind.borderStroke, "use the frame's actual white stroke at this resolution")
			for _, direction := range []environment.WindDirection{environment.Calm, environment.North, environment.South, environment.East, environment.West} {
				wind.Draw(screen, direction)
				capBounds := wind.background.Bounds()
				capWidth := max(1, capBounds.Dy()/2)
				slope := float64(2*(capWidth-1)) / float64(max(1, capBounds.Dy()-1))
				expectedSpan := int(math.Round(float64(stroke) * math.Hypot(1, slope)))
				whiteSpan := 0
				for x := range capWidth {
					if wind.background.At(x, capBounds.Dy()/2) == ucolor.EGAWhite.ToRGBA() {
						whiteSpan++
					}
				}
				assert.Equal(t, expectedSpan, whiteSpan, "diagonal stroke must match the frame's perpendicular thickness")
				assert.True(t, wind.backgroundBounds.In(outer), "Wind backing must fit screen: %v", wind.backgroundBounds)
				assert.GreaterOrEqual(t, wind.backgroundBounds.Min.Y, aperture.Max.Y, "Wind must stay in bottom band")
				assert.InDelta(t, float64(outer.Max.X)/2, float64(wind.backgroundBounds.Min.X+wind.backgroundBounds.Max.X)/2, 0.5)
				assert.InDelta(t, float64(aperture.Max.Y+outer.Max.Y)/2, float64(wind.textBounds.Min.Y+wind.textBounds.Max.Y)/2, 0.5)
			}
		})
	}
}
