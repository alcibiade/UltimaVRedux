package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

const (
	codexGuardianX = 0xe9
	codexGuardianY = 0xeb
)

func (g *GameState) processCodexGuardianCheckpoint() {
	location := g.MapState.PlayerLocation
	if location.Location != references.BritanniaUnderworld ||
		location.Floor != references.FloorNumber(references.OVERWORLD) ||
		int(location.Position.X) != codexGuardianX || int(location.Position.Y) != codexGuardianY {
		return
	}

	if g.GetFirstActiveQuest() != -1 {
		g.SystemCallbacks.Message.AddRowStr("\"Pass, Seeker!\"")

		return
	}

	g.SetPlayerPosition(references.Position{X: codexGuardianX, Y: codexGuardianY + 1})
	g.SystemCallbacks.Message.AddRowStr("\"Thou art not upon a Sacred Quest!\nPassage denied!\"")
	g.SystemCallbacks.Audio.PlaySoundEffect(SoundGuardianBlock)
}
