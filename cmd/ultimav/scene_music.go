package main

import "github.com/bradhannah/UltimaVRedux/internal/audio"

// Keep exploration history across load, new-game and menu round trips, but not
// across application restarts. Failed candidates cannot mutate this snapshot.
func (g *Game) musicHistoryForReplacement() audio.MusicHistory {
	if scene, ok := g.currentScene.(*GameScene); ok && scene.audioManager != nil {
		return scene.audioManager.GetMusicManager().History()
	}

	return g.musicHistory
}
