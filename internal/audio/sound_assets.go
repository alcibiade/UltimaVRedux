package audio

import (
	"embed"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

// Burning is brief game feedback. Release the final 200ms of its authored window.
const fireBurnReleaseMs = 200

// The game owns its manifest, recordings and notices; the reusable module embeds none.
//
//go:embed assets/*
var soundFiles embed.FS

// GetSoundEffectName resolves the game-owned display label independently of playback.
func GetSoundEffectName(effect game_state.SoundEffect) string {
	if metadata, exists := GetSoundEffectMetadata(effect); exists {
		return metadata.Name
	}

	return "Unknown Sound Effect"
}
