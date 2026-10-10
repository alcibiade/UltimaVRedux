package widgets

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCampingResultsUseCompactCenteredWorkflowOwnedDialog(t *testing.T) {
	for _, size := range []image.Point{{1280, 720}, {1920, 1080}} {
		setResponsiveDialogTestResolution(t, size.X, size.Y)
		for _, message := range []string{"No effect...", "Party rested!"} {
			result := NewCampingResultDialog(message)
			result.SetAcknowledgementPrompt(DialogAnyKeyHint)
			result.ShowWithoutAnimation()
			result.resolvePopupLayout(image.Rect(0, 0, size.X, size.Y))
			wanderer := NewCampingMessageDialog()
			wanderer.ShowWithoutAnimation()
			wanderer.resolvePopupLayout(image.Rect(0, 0, size.X, size.Y))
			assert.Equal(t, ContentAlignCenter, result.config.ContentAlignment)
			assert.True(t, result.config.TextHorizontalCenter)
			assert.False(t, result.enableInputDismissal, "camping workflow retains key/cleanup ownership")
			assert.Less(t, result.frameBounds.Dy(), wanderer.frameBounds.Dy())
			require.Equal(t, message, result.output.GetOutputStr(false))
			assert.Equal(t, DialogAnyKeyHint, result.promptOutput.GetOutputStr(false))
			assert.Equal(t, DialogCampingLines, wanderer.config.TextLines)
		}
	}
}
