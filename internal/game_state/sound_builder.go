package game_state

// SoundPlaybackMode controls whether a requested sound is finite or loops until stopped.
type SoundPlaybackMode int

const (
	// SoundPlaybackOneShot plays a finite stream once.
	SoundPlaybackOneShot SoundPlaybackMode = iota
	// SoundPlaybackLoop loops the stream until an explicit stop request.
	SoundPlaybackLoop
)

// SoundPlaybackRequest contains all configuration for one playback dispatch.
type SoundPlaybackRequest struct {
	Effect   SoundEffect
	Mode     SoundPlaybackMode
	Volume   float64
	FadeInMs int64
}

// SoundStopRequest contains all configuration for one stop dispatch.
type SoundStopRequest struct {
	Effect    SoundEffect
	FadeOutMs int64
}

// SoundVoice is an opaque presentation-only cleanup capability. Gameplay cannot
// query device state or use audio completion to govern time, RNG or outcomes.
type SoundVoice interface {
	Stop(fadeOutMs int64)
}

// SilentSoundVoice is the safe cleanup value before a workflow starts playback
// or when its presentation adapter intentionally has no audio device.
type SilentSoundVoice struct{}

// Stop has no effect for a silent presentation.
func (SilentSoundVoice) Stop(_ int64) {}

// SoundBuilder provides a fluent API for configuring sound effects before playing them.
type SoundBuilder struct {
	effect    SoundEffect
	callbacks *AudioCallbacks
	mode      SoundPlaybackMode
	fadeInMs  int64
	volume    float64
}

// StopBuilder provides a fluent API for stopping sound effects
type StopBuilder struct {
	effect    SoundEffect
	callbacks *AudioCallbacks
	fadeOutMs int64
}

// newSoundBuilder creates a new SoundBuilder for the given effect
func newSoundBuilder(effect SoundEffect, callbacks *AudioCallbacks) *SoundBuilder {
	return &SoundBuilder{
		effect:    effect,
		callbacks: callbacks,
		mode:      SoundPlaybackOneShot,
		volume:    1.0,
	}
}

// newStopBuilder creates a new StopBuilder for the given effect
func newStopBuilder(effect SoundEffect, callbacks *AudioCallbacks) *StopBuilder {
	return &StopBuilder{
		effect:    effect,
		callbacks: callbacks,
	}
}

// Loop configures the sound to play indefinitely until explicitly stopped
func (b *SoundBuilder) Loop() *SoundBuilder {
	b.mode = SoundPlaybackLoop

	return b
}

// OneShot configures the sound to play once and stop (default behavior)
func (b *SoundBuilder) OneShot() *SoundBuilder {
	b.mode = SoundPlaybackOneShot

	return b
}

// FadeIn configures the sound to fade in over the specified duration
func (b *SoundBuilder) FadeIn(ms int64) *SoundBuilder {
	b.fadeInMs = ms

	return b
}

// Volume configures the playback volume (0.0 to 1.0)
func (b *SoundBuilder) Volume(v float64) *SoundBuilder {
	if v < 0.0 {
		v = 0.0
	} else if v > 1.0 {
		v = 1.0
	}
	b.volume = v

	return b
}

// Start executes the configured sound playback
func (b *SoundBuilder) Start() {
	b.callbacks.RequestPlayback(SoundPlaybackRequest{
		Effect: b.effect, Mode: b.mode, Volume: b.volume, FadeInMs: b.fadeInMs,
	})
}

// StartOwned returns a cleanup capability for this instance, independent of
// other presentations of the same effect. Its workflow releases it in Finally.
//
//nolint:ireturn // Opaque presentation cleanup keeps gameplay independent of audio backends.
func (b *SoundBuilder) StartOwned() SoundVoice {
	voice := b.callbacks.RequestVoice(SoundPlaybackRequest{
		Effect: b.effect, Mode: b.mode, Volume: b.volume, FadeInMs: b.fadeInMs,
	})
	if voice == nil {
		panic("AudioCallbacks.RequestVoice must return a cleanup capability")
	}

	return voice
}

// FadeOut configures the stop to fade out over the specified duration
func (sb *StopBuilder) FadeOut(ms int64) *StopBuilder {
	sb.fadeOutMs = ms

	return sb
}

// Now executes an immediate stop (or fade-out if configured)
func (sb *StopBuilder) Now() {
	sb.callbacks.RequestStop(SoundStopRequest{Effect: sb.effect, FadeOutMs: sb.fadeOutMs})
}
