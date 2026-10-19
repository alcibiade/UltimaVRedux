package widgets

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/visual"
)

func TestMessageDialogDissolveOwnershipTransitions(t *testing.T) {
	for _, transition := range []string{"show", "show without animation", "hide", "hide without animation", "reveal complete", "hide complete"} {
		t.Run(transition, func(t *testing.T) {
			old := visual.NewPixelDissolveEffect(8, 8, 4, 4, visual.DissolveModeReveal, deterministic_random.New(42))
			t.Cleanup(old.Dispose)
			old.UpdateProgress(.5)
			require.NotNil(t, old.GetMask())
			survivor := visual.NewPixelDissolveEffect(8, 8, 4, 4, visual.DissolveModeReveal, deterministic_random.New(43))
			t.Cleanup(survivor.Dispose)
			survivor.UpdateProgress(1)
			content := ebiten.NewImage(8, 8)
			t.Cleanup(content.Deallocate)
			content.Fill(color.RGBA{R: 80, G: 40, B: 20, A: 128})
			dialog := &MessageDialog{
				enableDissolveEffects: true, dissolveEffect: old, contentBuffer: content,
				animationClock: func() int64 { return 100 }, fadeDurationInMs: 100, fadeDurationOutMs: 100,
				visible: true,
			}
			switch transition {
			case "show":
				dialog.Show()
			case "show without animation":
				dialog.ShowWithoutAnimation()
			case "hide":
				dialog.Hide()
				require.NotNil(t, dialog.dissolveEffect)
				require.NotSame(t, old, dialog.dissolveEffect)
				t.Cleanup(dialog.dissolveEffect.Dispose)
				require.Equal(t, fadingOut, dialog.fadeState)
			case "hide without animation":
				dialog.HideWithoutAnimation()
			case "reveal complete":
				dialog.fadeState = fadingIn
				dialog.Update()
				require.True(t, dialog.visible)
				require.Zero(t, dialog.dissolveLevel)
			case "hide complete":
				dialog.fadeState = fadingOut
				dialog.Update()
				require.False(t, dialog.visible)
				require.Equal(t, float64(1), dialog.dissolveLevel)
			}
			require.Nil(t, old.GetMask(), "the replaced/completed effect must release its owned native views")
			require.Equal(t, color.RGBA{R: 80, G: 40, B: 20, A: 128}, color.RGBAModel.Convert(content.At(0, 0)),
				"retiring a mask must preserve the dialog content")
			require.Equal(t, color.RGBA{R: 255, G: 255, B: 255, A: 255}, color.RGBAModel.Convert(survivor.GetMask().At(0, 0)),
				"another effect's owner and borrower must survive")
		})
	}
}
