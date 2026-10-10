package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	dungeonTorchMinimumMinutes = 112
	dungeonTorchMaximumMinutes = 127
	dungeonTorchFuelMaximum    = 255
)

// ActionIgnite handles torch ignition - non-directional action
func (g *GameState) ActionIgnite() bool {
	g.SystemCallbacks.Message.ShowCommandPrompt("Ignite torch!")
	isDungeon := g.MapState.PlayerLocation.Location.GetMapType() == references.DungeonMapType
	if isDungeon {
		g.requireActiveDungeon()
		g.claimDungeonCommandAction(DungeonFullEndPlayer)
	}
	if !g.PartyState.Inventory.Provisions.Torches.DecrementByOne() {
		g.SystemCallbacks.Message.AddRowStr("None owned!")

		return false
	}

	if isDungeon {
		g.MapState.Lighting.ExtendTorchMinutes(
			g.RandomIntInRange(dungeonTorchMinimumMinutes, dungeonTorchMaximumMinutes),
			dungeonTorchFuelMaximum,
		)
	} else {
		g.MapState.Lighting.LightTorch()
	}
	// Invalidate light source cache since torch state changed
	g.GetCurrentLayeredMap().InvalidateLightSourceCache()
	// REDUX DEVIATION: ignite consumes a torch without a dedicated sound.
	// Redux reserves an ignition cue at the successful consumption boundary.
	// See ref:PROV-0003.
	g.SystemCallbacks.Audio.PlaySoundEffect(SoundTorchIgnite)
	if !isDungeon {
		g.AdvanceOrdinarySurfaceCommandMinutes(1)
	}

	return true
}
