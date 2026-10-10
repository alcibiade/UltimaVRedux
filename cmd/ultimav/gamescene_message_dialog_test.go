package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestStyledMessageAdapterPreservesPrimaryFormattingAndSecondaryTranscript(t *testing.T) {
	for _, modal := range []bool{false, true} {
		for _, leadingBlank := range []bool{false, true} {
			output := text.NewOutput(nil, 1, 10, 80)
			expectedOutput := text.NewOutput(nil, 1, 10, 80)
			scene := &GameScene{output: output}
			message := game_state.StyledMessage{
				LeadingBlank: leadingBlank,
				Segments: []game_state.MessageSegment{
					{Text: "\"", Style: game_state.MessageQuotation},
					{Text: "Thou art now level", Style: game_state.MessageSpoken},
					{Text: " 2", Style: game_state.MessageHighlight},
					{Text: "\"", Style: game_state.MessageQuotation},
				},
			}
			var actualDialog, expectedDialog *widgets.MessageDialog
			builder := expectedOutput.AppendLine()
			if modal {
				actualDialog = widgets.NewCampingMessageDialog()
				expectedDialog = widgets.NewCampingMessageDialog()
				scene.dialogStack.PushModalDialog(actualDialog)
				builder = expectedDialog.AppendLineBuilder()
				builder.SetPlainTextCallback(expectedOutput.AddText)
			}
			if leadingBlank {
				builder.NewLine()
			}
			builder.QuoteStart().Spoken("Thou art now level").LightGreen(" 2").QuoteEnd().Done()
			scene.appendStyledMessage(message)
			assert.Equal(t, expectedOutput, output, "modal=%t leadingBlank=%t", modal, leadingBlank)
			if modal {
				assert.Equal(t, expectedDialog.AppendLineBuilder(), actualDialog.AppendLineBuilder())
				assert.Equal(t, 1, output.GetPanelContentLineCount(), "secondary receives no spacing row")
			} else if leadingBlank {
				assert.Equal(t, 2, output.GetPanelContentLineCount())
			}
		}
	}
}
