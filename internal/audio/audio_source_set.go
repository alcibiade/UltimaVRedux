package audio

import (
	"fmt"
	"io"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// AudioSourceSet manages multiple audio clips for the same sound effect
// to provide variation and avoid repetitive audio playback
type AudioSourceSet struct {
	clips       []*audio.Player
	lastClipIdx int
	rng         *rand.Rand
}

// NewAudioSourceSet creates a new audio source set with the given clips
func NewAudioSourceSet(audioContext *audio.Context, streams []io.ReadSeeker) (*AudioSourceSet, error) {
	if len(streams) == 0 {
		return nil, nil // Empty set is valid
	}

	clips := make([]*audio.Player, len(streams))

	for i, stream := range streams {
		player, err := audioContext.NewPlayer(stream)
		if err != nil {
			err = fmt.Errorf("creating audio source player %d: %w", i, err)
			// Clean up any successfully created players
			for j := range i {
				clips[j].PauseAndStopReading()
			}

			return nil, err
		}
		clips[i] = player
	}

	return &AudioSourceSet{
		clips:       clips,
		lastClipIdx: -1,
		rng:         rand.New(rand.NewSource(42)), // Fixed seed for consistent behavior
	}, nil
}

// GetNext returns the next audio player, attempting to avoid the last played sound
// to give a less uniform feel, similar to the C# implementation
func (ass *AudioSourceSet) GetNext() *audio.Player {
	if len(ass.clips) == 0 {
		return nil
	}

	if len(ass.clips) == 1 {
		return ass.clips[0]
	}

	// Pick a random sound that isn't the last sound played
	nextIdx := ass.rng.Intn(len(ass.clips))
	if nextIdx == ass.lastClipIdx {
		ass.lastClipIdx = (ass.lastClipIdx + 1) % len(ass.clips)
	} else {
		ass.lastClipIdx = nextIdx
	}

	return ass.clips[ass.lastClipIdx]
}

// Close stops all owned players and releases their references.
func (ass *AudioSourceSet) Close() {
	for _, clip := range ass.clips {
		if clip != nil {
			clip.PauseAndStopReading()
		}
	}
	ass.clips = nil
}

// SetVolume sets the volume for all clips in the set
func (ass *AudioSourceSet) SetVolume(volume float64) {
	for _, clip := range ass.clips {
		if clip != nil {
			clip.SetVolume(volume)
		}
	}
}

// IsEmpty returns true if this audio source set has no clips
func (ass *AudioSourceSet) IsEmpty() bool {
	return len(ass.clips) == 0
}
