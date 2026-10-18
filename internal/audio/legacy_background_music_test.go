package audio

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

// BackgroundMusicSystem handles finite background music playback and volume control.
type BackgroundMusicSystem struct {
	library      *Library
	backend      PlaybackBackend
	config       *AudioConfiguration
	audioContext *audio.Context
	currentTrack string
	player       *gainPlayer
	mutex        sync.RWMutex
	assetsLoaded bool
	musicAssets  map[string]*MusicAsset
	isPlaying    bool
	volumeGain   float64 // Context/audition gain independent of user preference.
}

// MusicAsset represents a loaded music track.
type MusicAsset struct {
	Name           string
	StreamFactory  func() (io.ReadSeeker, error)
	Duration       int64
	VolumeModifier float64
}

// NewBackgroundMusicSystemWithContext creates music playback using the shared context.
func NewBackgroundMusicSystemWithContext(
	config *AudioConfiguration, context *audio.Context,
) (*BackgroundMusicSystem, error) {
	if config == nil || context == nil {
		return nil, fmt.Errorf("%w: music requires configuration and audio context", ErrAudioLibrary)
	}

	return newProjectMusicSystem(config, defaultRuntimeLibrary(), EbitengineBackend{Context: context}, context), nil
}

func newProjectMusicSystem(config *AudioConfiguration, library *Library, backend PlaybackBackend,
	context *audio.Context) *BackgroundMusicSystem {
	return &BackgroundMusicSystem{
		library: library, backend: backend,
		config: config, audioContext: context, currentTrack: "", player: nil,
		mutex: sync.RWMutex{}, assetsLoaded: false,
		musicAssets: make(map[string]*MusicAsset), isPlaying: false, volumeGain: 1,
	}
}

// LoadAssets resolves exact manifest assignments; null-file slots deliberately stay silent.
func (bms *BackgroundMusicSystem) LoadAssets() error {
	bms.mutex.Lock()
	defer bms.mutex.Unlock()
	if bms.assetsLoaded {
		return nil
	}
	for _, track := range musicTrackIDs() {
		cue, exists := bms.library.document.Cues["music/track/"+track]
		if !exists {
			continue
		}
		path := bms.library.recordingPath(cue.RecordingID)
		if path == "" {
			continue
		}
		if err := bms.loadMusicAsset(track, path); err != nil {
			return fmt.Errorf("load music track %s: %w", track, err)
		}
	}
	bms.assetsLoaded = true

	return nil
}

func (bms *BackgroundMusicSystem) loadMusicAsset(track, path string) error {
	data, err := bms.library.readFile(path)
	if err != nil {
		return fmt.Errorf("read music %s: %w", path, err)
	}
	metadata := bms.library.document.Cues["music/track/"+track]
	window := bms.library.document.Recordings[metadata.RecordingID].PlaybackWindow
	stream, err := decodeRecording(data, path, bms.backend.SampleRate(), window, false)
	if err != nil {
		return fmt.Errorf("decode music %s: %w", path, err)
	}
	length, err := decodedStreamLength(stream)
	if err != nil {
		return err
	}
	bms.musicAssets[track] = &MusicAsset{
		Name: track,
		StreamFactory: func() (io.ReadSeeker, error) {
			fresh, err := decodeRecording(data, path, bms.backend.SampleRate(), window, false)
			if err != nil {
				return nil, fmt.Errorf("recreate music %s: %w", path, err)
			}

			return fresh, nil
		},
		Duration:       length * millisecondsPerSecond / (int64(bms.backend.SampleRate()) * decodedPCMBytesPerFrame),
		VolumeModifier: metadata.VolumeModifier * bms.library.document.recordingCalibration(metadata.RecordingID),
	}

	return nil
}

// Play starts playing background music, replacing the preceding player.
func (bms *BackgroundMusicSystem) Play(track string) error {
	bms.mutex.Lock()
	defer bms.mutex.Unlock()

	return bms.playLocked(track, 0)
}

func (bms *BackgroundMusicSystem) playLocked(track string, position time.Duration) error {
	if !bms.assetsLoaded {
		return fmt.Errorf("%w: music assets not loaded", ErrAudioLibrary)
	}
	metadata, known := bms.library.document.Cues["music/track/"+track]
	if !known {
		return fmt.Errorf("%w: unknown music track %s", ErrAudioLibrary, track)
	}
	if bms.library.document.Recordings[metadata.RecordingID].File == nil {
		bms.stopLocked()

		return nil
	}
	asset, exists := bms.musicAssets[track]
	if !exists {
		return fmt.Errorf("%w: track %s was not loaded", ErrAudioLibrary, track)
	}
	position = boundedMusicPosition(position, asset.Duration)
	stream, err := asset.StreamFactory()
	if err != nil {
		return fmt.Errorf("prepare music %s: %w", track, err)
	}
	if stream == nil {
		return fmt.Errorf("%w: music stream unavailable for %s", ErrAudioLibrary, track)
	}
	player, err := newGainPlayerWithBackend(bms.backend, stream)
	if err != nil {
		return fmt.Errorf("create music player: %w", err)
	}
	player.SetVolume(outputGain(bms.config.GetEffectiveMusicVolume() * asset.VolumeModifier * bms.volumeGain))
	if position > 0 {
		if err := player.SetPosition(position); err != nil {
			if closeErr := player.Close(); closeErr != nil {
				logging.Warn(logging.CategoryAudio, "Close unseekable music: %v", closeErr)
			}

			return fmt.Errorf("seek resumed music: %w", err)
		}
	}
	bms.stopLocked()
	player.Play()
	bms.player, bms.currentTrack, bms.isPlaying = player, track, true

	return nil
}

// Pause retains the finite player and reports a position usable after a preview
// replaces it. Paused playback is not a natural completion.
func (bms *BackgroundMusicSystem) Pause() time.Duration {
	bms.mutex.Lock()
	defer bms.mutex.Unlock()
	if bms.player == nil {
		return 0
	}
	bms.player.Pause()
	bms.isPlaying = false

	return boundedMusicPosition(bms.player.Position(), bms.musicAssets[bms.currentTrack].Duration)
}

// Resume reuses a paused player, or seeks a fresh stream after preview ownership.
func (bms *BackgroundMusicSystem) Resume(track string, position time.Duration) error {
	bms.mutex.Lock()
	defer bms.mutex.Unlock()
	if bms.player != nil && bms.currentTrack == track && !bms.isPlaying {
		// The retained player may be a same-theme settings preview. Always
		// restore the policy's saved position rather than trusting its identity.
		position = boundedMusicPosition(position, bms.musicAssets[track].Duration)
		if err := bms.player.SetPosition(position); err != nil {
			return fmt.Errorf("restore music position: %w", err)
		}
		bms.player.Play()
		bms.isPlaying = true

		return nil
	}

	return bms.playLocked(track, position)
}

// Ebitengine smooths device positions using its buffered samples and stopwatch.
// At rapid pause/seek boundaries that estimate can escape the finite stream;
// never feed an out-of-range estimate back into the bounded recording seeker.
func boundedMusicPosition(position time.Duration, durationMs int64) time.Duration {
	return max(0, min(position, time.Duration(durationMs)*time.Millisecond))
}

// Stop stops current background music.
func (bms *BackgroundMusicSystem) Stop() error {
	bms.mutex.Lock()
	defer bms.mutex.Unlock()
	bms.stopLocked()

	return nil
}

func (bms *BackgroundMusicSystem) stopLocked() {
	if bms.player != nil {
		if err := bms.player.Close(); err != nil {
			logging.Warn(logging.CategoryAudio, "Close music player: %v", err)
		}
		bms.player = nil
	}
	bms.currentTrack, bms.isPlaying = "", false
}

// IsPlaying reports whether the active music player is playing.
func (bms *BackgroundMusicSystem) IsPlaying() bool {
	bms.mutex.RLock()
	defer bms.mutex.RUnlock()

	return bms.isPlaying && bms.player != nil && bms.player.IsPlaying()
}

// Update releases completed finite players and failed decoders, including a
// decoder that failed while policy playback was suspended. Failure is reported
// exactly once; paused healthy playback is retained.
func (bms *BackgroundMusicSystem) Update() error {
	bms.mutex.Lock()
	defer bms.mutex.Unlock()
	if bms.player == nil {
		return nil
	}
	if err := bms.player.Failure(); err != nil {
		track := bms.currentTrack
		bms.stopLocked()

		return fmt.Errorf("retire failed music %s: %w", track, err)
	}
	if bms.isPlaying && !bms.player.IsPlaying() {
		bms.stopLocked()
	}

	return nil
}

// GetCurrentTrack returns the playing track identity.
func (bms *BackgroundMusicSystem) GetCurrentTrack() string {
	bms.mutex.RLock()
	defer bms.mutex.RUnlock()

	return bms.currentTrack
}

// SetVolume applies a multiplier to the configured music volume and track gain.
func (bms *BackgroundMusicSystem) SetVolume(volume float64) error {
	if err := validateVolumeRange(volume); err != nil {
		return fmt.Errorf("music context gain: %w", err)
	}
	bms.mutex.Lock()
	defer bms.mutex.Unlock()
	bms.volumeGain = volume
	bms.refreshVolumeLocked()

	return nil
}

// RefreshVolume recomputes the active player after a user preference changes.
func (bms *BackgroundMusicSystem) RefreshVolume() error {
	bms.mutex.Lock()
	defer bms.mutex.Unlock()
	bms.refreshVolumeLocked()

	return nil
}

func (bms *BackgroundMusicSystem) refreshVolumeLocked() {
	if bms.player != nil && bms.currentTrack != "" {
		gain := bms.config.GetEffectiveMusicVolume() * bms.musicAssets[bms.currentTrack].VolumeModifier * bms.volumeGain
		bms.player.SetVolume(outputGain(gain))
	}
}

// Close releases music players and loaded stream factories.
func (bms *BackgroundMusicSystem) Close() error {
	bms.mutex.Lock()
	defer bms.mutex.Unlock()
	bms.stopLocked()
	bms.musicAssets = make(map[string]*MusicAsset)
	bms.assetsLoaded = false

	return nil
}
