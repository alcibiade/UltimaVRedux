package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// SmashedCommandInterception reports whether intoxication replaced an accepted town command.
type SmashedCommandInterception struct {
	Intercepted     bool
	Direction       references.Direction
	SmashedBefore   int
	SmashedAfter    int
	GuardActivation GuardActivationResult
}

// InterceptSmashedCommand applies the original town-command intoxication check.
func (g *GameState) InterceptSmashedCommand() (SmashedCommandInterception, error) {
	result := SmashedCommandInterception{
		SmashedBefore: g.smashed,
		SmashedAfter:  g.smashed,
	}
	if g.smashed == 0 || g.RandomIntInRange(0, 1) == 0 {
		return result, nil
	}

	guardActivation, err := g.ActivateGuards()
	if err != nil {
		return result, fmt.Errorf("activating guards for smashed command: %w", err)
	}
	result.GuardActivation = guardActivation

	g.smashed--
	result.SmashedAfter = g.smashed
	g.SystemCallbacks.Message.AddRowStr("Hic!")

	switch g.RandomIntInRange(0, 3) {
	case 0:
		result.Direction = references.Up
	case 1:
		result.Direction = references.Down
	case 2:
		result.Direction = references.Right
	case 3:
		result.Direction = references.Left
	}
	result.Intercepted = true

	return result, nil
}

func (g *GameState) resetSmashedForFreshTown() {
	g.smashed = 0
}
