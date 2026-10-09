package mainscreen

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestAppendCombatConditionsPreservesExactIdentityLabelsAndColours(t *testing.T) {
	for _, test := range []struct {
		conditions CombatConditions
		want       string
	}{
		{CombatConditions{}, ""},
		{CombatConditions{Possessed: true}, "Possessed"},
		{CombatConditions{Underground: true}, "Underground"},
		{CombatConditions{Sleeping: true}, "Sleep"},
		{CombatConditions{Possessed: true, Underground: true, Sleeping: true}, "Possessed | Underground | Sleep"},
	} {
		output := text.NewOutput(nil, 1, 4, 80)
		line := AppendCombatConditions(output.AppendLine(), test.conditions)
		for _, segment := range line.GetSegments() {
			switch segment.Text {
			case "Possessed":
				assert.Equal(t, ucolor.CombatPossessed, segment.Color)
				assert.Equal(t, ucolor.EGALightBlue, segment.Color)
			case "Underground":
				assert.Equal(t, ucolor.CombatUndergroundText, segment.Color)
			case "Sleep":
				assert.Equal(t, ucolor.EGALightMagenta, segment.Color)
			}
		}
		line.Done()
		assert.Equal(t, test.want, output.GetOutputStr(false))
	}
}
