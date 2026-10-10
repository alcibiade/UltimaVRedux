package widgets

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

func TestSinglePaneDialogsUseRoundedClassicFrames(t *testing.T) {
	setResponsiveDialogTestResolution(t, 640, 360)
	screen := ebiten.NewImage(640, 360)
	t.Cleanup(screen.Dispose)
	placement := sprites.PercentBasedPlacement{
		StartPercentX: 0.15,
		EndPercentX:   0.85,
		StartPercentY: 0.15,
		EndPercentY:   0.85,
	}

	t.Run("legacy path", func(t *testing.T) {
		dialog := newTestLegacyPathEntryModal(new(fakeLegacyPathTextField), func(string, bool) {})
		screen.Clear()
		dialog.Draw(screen)
		requireSingleClassicPane(t, screen, dialog.border.GetBounds())
	})

	t.Run("astronomy", func(t *testing.T) {
		dialog := NewAstronomyDialog(AstronomyDialogConfig{
			Placement:        placement,
			GameYear:         139,
			GameMonth:        4,
			GameDay:          5,
			RandomIntInRange: func(minimum, _ int) int { return minimum },
		})
		screen.Clear()
		dialog.Draw(screen)
		requireSingleClassicPane(t, screen, dialog.bounds)
	})

	t.Run("gem view flash", func(t *testing.T) {
		dialog := NewGemViewDialog(GemViewDialogConfig{
			Placement:    placement,
			PlayerWorldX: 0,
			PlayerWorldY: 0,
		})
		dialog.ToggleFlash()
		screen.Clear()
		dialog.Draw(screen)
		requireSingleClassicPane(t, screen, dialog.bounds)
	})

	t.Run("dungeon view", func(t *testing.T) {
		dialog := NewDungeonViewDialog(DungeonViewDialogConfig{
			Placement: placement,
			Model:     game_state.DungeonViewModel{},
		})
		screen.Clear()
		dialog.Draw(screen)
		requireSingleClassicPane(t, screen, dialog.bounds)
	})

	t.Run("sign", func(t *testing.T) {
		dialog := NewSignDialog(SignDialogConfig{
			Placement: placement,
			Keyboard:  newWidgetTestKeyboard(0),
			RawText:   []byte("WELCOME"),
		})
		screen.Clear()
		dialog.Draw(screen)
		requireSingleClassicPane(t, screen, dialog.bounds)
	})
}

func requireSingleClassicPane(t *testing.T, screen *ebiten.Image, bounds image.Rectangle) {
	t.Helper()
	topCenter := image.Pt(bounds.Min.X+bounds.Dx()/2, bounds.Min.Y)
	requireColorRGBA(t, DialogRuleColor, screen.At(topCenter.X, topCenter.Y))
	require.Zero(t, alphaAt(screen, bounds.Min))
	center := image.Pt(bounds.Min.X+bounds.Dx()/2, bounds.Min.Y+bounds.Dy()/2)
	require.NotZero(t, alphaAt(screen, center))
}
