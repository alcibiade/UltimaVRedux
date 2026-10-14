package mainscreen

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

const combatConditionPossessedLabel = "Possessed"

// CombatConditions is a detached, exact-combatant view for the turn HUD and
// battlefield markers. It never changes the character's persistent status.
type CombatConditions struct {
	Possessed       bool
	Underground     bool
	Sleeping        bool
	TimedPossession bool
	PossessionTurns int
}

// AppendCombatConditions labels only the exact source or clone shown by the HUD.
func AppendCombatConditions(line *text.LineBuilder, conditions CombatConditions) *text.LineBuilder {
	separator := ""
	possession := combatConditionPossessedLabel
	if conditions.TimedPossession {
		possession = fmt.Sprintf("%s - %d/12 turns", combatConditionPossessedLabel, conditions.PossessionTurns)
	}
	for _, condition := range []struct {
		label   string
		present bool
		color   ucolor.EGAColor
	}{
		{possession, conditions.Possessed, ucolor.CombatPossessed},
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
