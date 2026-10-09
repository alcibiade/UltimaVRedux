package mainscreen

import (
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// CombatConditions is a detached, exact-combatant view for the turn HUD and
// battlefield markers. It never changes the character's persistent status.
type CombatConditions struct {
	Possessed   bool
	Underground bool
	Sleeping    bool
}

// AppendCombatConditions labels only the exact source or clone shown by the HUD.
func AppendCombatConditions(line *text.LineBuilder, conditions CombatConditions) *text.LineBuilder {
	separator := ""
	for _, condition := range []struct {
		label   string
		present bool
		color   ucolor.EGAColor
	}{
		{"Possessed", conditions.Possessed, ucolor.CombatPossessed},
		{"Underground", conditions.Underground, ucolor.CombatUndergroundText},
		{"Sleep", conditions.Sleeping, ucolor.EGALightMagenta},
	} {
		if !condition.present {
			continue
		}
		line.Text(separator, ucolor.EGALightGray).Text(condition.label, condition.color)
		separator = " | "
	}

	return line
}
