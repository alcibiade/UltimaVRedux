package legacy_save

import "github.com/bradhannah/UltimaVRedux/internal/save_snapshot"

func convertDungeon(game SavedGame, snapshot *save_snapshot.SaveSnapshot) {
	if !game.Dungeon.Active {
		return
	}

	dungeon := &save_snapshot.DungeonSnapshot{
		Location:          int(game.Position.Location),
		Cells:             make([]int, len(game.Dungeon.Cells)),
		Facing:            int(game.Dungeon.Facing),
		LastDirection:     int(game.Dungeon.LastDirection),
		Wanderer:          nil,
		QuicknessSkipNext: false,
	}
	for index, cell := range game.Dungeon.Cells {
		dungeon.Cells[index] = int(cell)
	}

	current := game.ActiveObjects[1]
	if current.Number != originalNothing {
		previous := game.ActiveObjects[2]
		dungeon.Wanderer = &save_snapshot.DungeonWandererSnapshot{
			Monster:   int(current.Number),
			Floor:     int(current.Level),
			X:         int(current.X),
			Y:         int(current.Y),
			PreviousX: int(previous.X),
			PreviousY: int(previous.Y),
			OnCeiling: current.Misc == originalNothing,
		}
	}
	snapshot.Dungeon = dungeon
}
