package widgets

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/grammar"
	"github.com/stretchr/testify/assert"
)

func TestTextInputCommandColorMatchesManualAndHistoryText(t *testing.T) {
	commands := grammar.TextCommands{
		*grammar.NewTextCommand([]grammar.Match{
			grammar.MatchStringList{Strings: []string{"mv"}, Description: "Movement"},
			grammar.MatchStringList{Strings: []string{"teleport"}, Description: "Destination"},
		}, nil),
	}
	input := &TextInput{
		output:       text.NewOutput(nil, 0, 1, 90),
		textCommands: &commands,
		inputColors: TextInputColors{
			DefaultColor:          ucolor.EGAWhite,
			NoMatchesColor:        ucolor.EGARed,
			OneMatchColor:         ucolor.EGALightGreen,
			MoreThanOneMatchColor: ucolor.EGAYellow,
		},
	}

	input.output.AppendText("m")
	input.output.AppendText("v")
	assert.Equal(t, ucolor.EGALightGreen, input.getTextColor())

	input.output.AppendText("a")
	assert.Equal(t, ucolor.EGARed, input.getTextColor())

	input.SetText("mv")
	assert.Equal(t, ucolor.EGALightGreen, input.getTextColor(),
		"history-loaded and manually typed commands must use the same grammar state")
}
