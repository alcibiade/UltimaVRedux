package widgets

import (
	"image"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

func TestCompassionWisdomWrapsNaturallyWithinShrineDialog(t *testing.T) {
	const wisdom = "Only a detested life owes its pleasures to another's pain."
	require.Equal(t, wisdom, game_state.GetShrineData(1).AnswerText())
	for _, size := range []image.Point{{1280, 720}, {1920, 1080}, {3840, 2160}} {
		t.Run(size.String(), func(t *testing.T) {
			setResponsiveDialogTestResolution(t, size.X, size.Y)
			viewport := image.Rect(size.X/4, 0, size.X*3/4, size.Y*4/5)
			setMessageDialogTestPopupViewport(t, func(image.Rectangle) image.Rectangle { return viewport })
			overlay := NewOverlayDialog("Thou dost read:\n" + game_state.GetShrineData(1).AnswerText()).
				WithSizing(SizingAutoFitDefault).WithPosition(PositionMapCenterLower).
				WithEffects(EffectsNone).WithContentAlignment(ContentAlignCenter).
				WithTextAlignment(TextAlignCenter).WithUserDismissed(newWidgetTestKeyboard(-1)).Build()
			overlay.ShowWithoutAnimation()
			dialog := overlay.dialog
			overlay.PrepareLayout(image.Rectangle{Max: size})
			require.Equal(t, strings.Fields("Thou dost read: "+wisdom), strings.Fields(dialog.renderOutput.GetOutputStr(false)))
			require.True(t, dialog.frameBounds.In(image.Rectangle{Max: size}))
			spacing := dialog.getScaledLineSpacing()
			layout := dialog.calculateDualOutputLayout(dialog.frameBounds, spacing, spacing, spacing)
			require.LessOrEqual(t, dialog.renderOutput.GetPanelContentLineCount(), dialog.renderOutput.GetLineCapacity(layout.ContentHeight))
			// The legacy data forced four wisdom fragments plus the heading even
			// at wide resolutions. The existing wrapper now needs fewer rows.
			require.Less(t, dialog.renderOutput.GetPanelContentLineCount(), 5)
			require.Equal(t, DialogCloseHint, dialog.promptOutput.GetOutputStr(false))
		})
	}
}
