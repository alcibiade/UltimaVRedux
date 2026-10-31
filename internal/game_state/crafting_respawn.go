package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// REDUX DEVIATION: CRAFT-0006: the original has no resource nodes to restore. Redux restores
// used-up deed-map nodes at each Trammel phase change; see ref:CRAFT-0006.

// processDeedRespawn brings every deed up to the current Trammel phase-change count
// (ref:CRAFT-0006). Each deed keeps its own marker: the count it was last brought up to. It
// runs at every change of day, the only moment Trammel's day-indexed phase can change, and
// before every gathering attempt. A deed staked today starts at today's count, so it never
// receives phase changes from before it existed. A date moved backwards (a debug change) only
// resets the marker. A deed whose map is suspended under combat or camping is skipped and
// catches up at the next change of day or gathering attempt, so a node never reappears under a
// party that has not yet returned. It consumes no game RNG.
func (g *GameState) processDeedRespawn() {
	current := datetime.TrammelPhaseChanges(g.DateTime.GetTotalDays())
	for _, deed := range g.crafting.deeds {
		if deed == g.crafting.active && !g.MapState.PlayerLocation.Location.IsDeedLocation() {
			continue
		}
		elapsed := current - deed.trammelPhaseChanges
		deed.trammelPhaseChanges = current
		if elapsed > 0 && len(deed.harvests) > 0 {
			g.respawnDeedNodes(deed, elapsed)
		}
	}
}

// respawnDeedNodes applies a number of Trammel phase changes to one deed. At each change:
//
//   - every node that has given harvests but is not used up becomes fresh again;
//   - of the used-up nodes that still show their depleted sprite, DeedRespawnPercent (rounded
//     up) are restored: their harvests clear and the generated tile returns. They are taken in
//     row-major order (top row first, then left to right), the order the generator lists nodes;
//   - on the loaded deed map, a node under the party, a creature or an item stack is passed
//     over for that change and the next node in order is restored instead.
//
// Used-up nodes covered by something else (a later building step) are not counted and wait
// until they show their depleted sprite again. The loop ends early once nothing is left to do.
func (g *GameState) respawnDeedNodes(deed *Deed, changes int) {
	layout := g.deedLayout(deed)
	live := deed == g.crafting.active && g.MapState.PlayerLocation.Location.IsDeedLocation()
	for range changes {
		eligible := make([]DeedResourceNodeState, 0, len(deed.harvests))
		partial := 0
		for _, node := range layout.Nodes {
			if deed.harvests[node.Position] == 0 {
				continue
			}
			state := deed.nodeState(layout, node)
			switch {
			case !state.Depleted:
				delete(deed.harvests, node.Position)
				partial++
			case !state.Covered:
				eligible = append(eligible, state)
			}
		}
		quota := (len(eligible)*references.DeedRespawnPercent + 99) / 100 //nolint:mnd // Percent, rounded up.
		restored := 0
		for _, node := range eligible {
			if restored == quota {
				break
			}
			if live && g.deedSquareOccupied(node.Position) {
				continue
			}
			delete(deed.harvests, node.Position)
			g.SetDeedTile(deed.ID, node.Position, node.Sprite)
			restored++
		}
		if restored == 0 && partial == 0 {
			return
		}
	}
}

// deedSquareOccupied reports whether the party, a map unit or an item stack stands on a square
// of the loaded deed map.
func (g *GameState) deedSquareOccupied(position references.Position) bool {
	return g.MapState.PlayerLocation.Position == position || g.largeMapPushOccupied(position)
}
