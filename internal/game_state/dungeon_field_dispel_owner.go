package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

// DungeonFieldDispelOwner is an opaque, visit-local presentation handle. It is
// neither persisted state nor a replay capability. Reconciliation must retire a
// mismatching handle permanently, even if a later cell happens to match again.
type DungeonFieldDispelOwner struct {
	dungeon  *DungeonState
	floor    references.FloorNumber
	position references.Position
	expected references.DungeonCell
}

// CaptureDungeonFieldDispelOwner accepts a just-committed packed field removal.
// Revision is checked only at admission; unrelated later mutations are harmless.
func (g *GameState) CaptureDungeonFieldDispelOwner(change DungeonCellChange) (DungeonFieldDispelOwner, bool) {
	if g.activeDungeon == nil || change.Before.Kind() != references.DungeonEnergyField ||
		change.After != references.DungeonCell(change.Before.Raw()&0x08) ||
		change.Revision == 0 || change.Revision != g.activeDungeon.revision {
		return DungeonFieldDispelOwner{}, false
	}
	owner := DungeonFieldDispelOwner{
		dungeon: g.activeDungeon, floor: change.Floor, position: change.Position, expected: change.After,
	}
	if !g.MatchesDungeonFieldDispelOwner(owner) {
		return DungeonFieldDispelOwner{}, false
	}

	return owner, true
}

// MatchesDungeonFieldDispelOwner checks current authority without mutating it.
// Turning, moving, changing wanderers, and editing other cells preserve a remnant.
func (g *GameState) MatchesDungeonFieldDispelOwner(owner DungeonFieldDispelOwner) bool {
	player := g.MapState.PlayerLocation
	if owner.dungeon == nil || g.activeDungeon != owner.dungeon || g.CombatScene != nil ||
		player.Location != owner.dungeon.location || player.Floor != owner.floor {
		return false
	}
	cell, valid := owner.dungeon.Cell(owner.floor, owner.position)

	return valid && cell == owner.expected
}
