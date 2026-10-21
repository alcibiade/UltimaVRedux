package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ActionTalkLargeMap handles Talk command on the overworld map.
// Per original Ultima V: Talk on overworld always returns "Funny, no response!"
func (g *GameState) ActionTalkLargeMap(direction references.Direction) bool {
	g.SystemCallbacks.Message.AddRowStr("Talk-Funny, no response!")
	g.AdvanceOrdinarySurfaceCommandMinutes(1)

	return true
}
