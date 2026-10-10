package audio

import (
	"errors"
	"fmt"
)

// Audition is a single seekable editor voice. All control methods are called on
// the caller's event loop. It needs no GameScene or Ebitengine graphics loop.
type Audition struct {
	player  *auditionPlayer
	startMs int64
	endMs   int64
	loop    bool
}

// Play replaces the preview only after a valid replacement player is ready.
func (a *Audition) Play(clip *RecordingAudio, window *PlaybackWindow, loop bool, gain float64) error {
	stream, err := clip.Stream(window, loop)
	if err != nil {
		return err
	}
	player, err := newAuditionPlayer(stream)
	if err != nil {
		return fmt.Errorf("create audition player: %w", err)
	}
	if err := a.Close(); err != nil {
		return errors.Join(err, player.Close())
	}
	a.startMs, a.endMs, a.loop = 0, clip.DurationMs, loop
	if window != nil {
		a.startMs, a.endMs = window.StartMs, window.EndMs
	}
	a.player = player
	a.SetGain(gain)
	player.Play()

	return nil
}

// SetGain changes the current voice without restarting it.
func (a *Audition) SetGain(gain float64) {
	if a.player != nil {
		a.player.SetVolume(outputGain(gain))
	}
}

// Playing reports whether the current voice is advancing.
func (a *Audition) Playing() bool { return a.player != nil && a.player.IsPlaying() }

// Active includes paused or completed voices retained for seeking.
func (a *Audition) Active() bool { return a.player != nil }

// Pause changes playback state without replacing its stream.
func (a *Audition) Pause(paused bool) {
	if a.player == nil {
		return
	}
	if paused {
		a.player.Pause()
	} else {
		a.player.Play()
	}
}

// PositionMs is always a source-file position, including selection and looping.
func (a *Audition) PositionMs() int64 {
	if a.player == nil {
		return 0
	}
	position := a.player.PositionMs()
	if a.loop && a.endMs > a.startMs {
		position %= a.endMs - a.startMs
	}

	return min(a.endMs, a.startMs+position)
}

// SeekMs clamps a source-relative timestamp to the playable selection.
func (a *Audition) SeekMs(sourceMs int64) error {
	if a.player == nil {
		return nil
	}
	position := max(a.startMs, min(a.endMs, sourceMs)) - a.startMs
	if err := a.player.SeekMs(position); err != nil {
		return fmt.Errorf("seek audition: %w", err)
	}

	return nil
}

// Close releases the voice and is safe to call more than once.
func (a *Audition) Close() error {
	if a.player == nil {
		return nil
	}
	err := a.player.Close()
	a.player = nil
	if err != nil {
		return fmt.Errorf("close audition: %w", err)
	}

	return nil
}
