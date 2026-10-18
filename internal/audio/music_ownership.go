package audio

import "github.com/bradhannah/UltimaVRedux/internal/game_state"

func isInstrumentEffect(effect game_state.SoundEffect) bool {
	return effect == game_state.SoundBardLute || effect == game_state.SoundTavernBardMusic ||
		(effect >= game_state.SoundHarpsichord0 && effect <= game_state.SoundHarpsichord9)
}
