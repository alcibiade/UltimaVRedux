package widgets

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/grammar"
)

func TestTextInputCommandColorMatchesManualAndHistoryText(t *testing.T) {
	commands := grammar.TextCommands{
		*grammar.NewTextCommand([]grammar.Match{
			grammar.MatchString{Str: "mv", Description: "Movement", CaseSensitive: false},
			grammar.MatchString{Str: "teleport", Description: "Destination", CaseSensitive: false},
		}, nil),
	}
	input := &TextInput{ //nolint:exhaustruct // Focused syntax-color fixture.
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
