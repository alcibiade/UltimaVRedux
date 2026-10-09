package audio

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

// MusicTrackMetadata is derived from the authored library, not a filename table.
type MusicTrackMetadata struct {
	Name           string
	RecordingID    string
	VolumeModifier float64
	Description    string
}

//nolint:gochecknoglobals // Immutable metadata derived from the validated embedded library.
var musicTrackMetadataMap = libraryMusicMetadata()

func libraryMusicMetadata() map[string]MusicTrackMetadata {
	metadata := make(map[string]MusicTrackMetadata)
	for _, track := range musicTrackIDs() {
		cue := productionLibrary().Cues["music/track/"+track]
		metadata[track] = MusicTrackMetadata{
			Name: cue.Name, RecordingID: cue.RecordingID,
			VolumeModifier: cue.VolumeModifier, Description: cue.HowToHear,
		}
	}

	return metadata
}

// getMusicTrackMetadata returns the exact current file paths (empty means an intentional slot).
func getMusicTrackMetadata() map[string]string {
	tracks := make(map[string]string)
	for name, metadata := range musicTrackMetadataMap {
		tracks[name] = productionLibrary().recordingPath(metadata.RecordingID)
	}

	return tracks
}

// GetMusicTrackMetadata returns the code-backed track's library metadata.
func GetMusicTrackMetadata(track string) (MusicTrackMetadata, bool) {
	metadata, exists := musicTrackMetadataMap[track]

	return metadata, exists
}

// BackgroundMusicSystem handles finite background music playback and volume control.
type BackgroundMusicSystem struct {
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
	StreamFactory  func() io.ReadSeeker
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

	return &BackgroundMusicSystem{
		config: config, audioContext: context, currentTrack: "", player: nil,
		mutex: sync.RWMutex{}, assetsLoaded: false,
		musicAssets: make(map[string]*MusicAsset), isPlaying: false, volumeGain: 1,
	}, nil
}

// LoadAssets resolves exact manifest assignments; null-file slots deliberately stay silent.
func (bms *BackgroundMusicSystem) LoadAssets() error {
	bms.mutex.Lock()
	defer bms.mutex.Unlock()
	if bms.assetsLoaded {
		return nil
	}
	for track, path := range getMusicTrackMetadata() {
		if path == "" {
			continue
		}
		if err := bms.loadMusicAsset(track, path); err != nil {
			panic(fmt.Sprintf("Failed to load music track %s: %v", track, err))
		}
	}
	bms.assetsLoaded = true

	return nil
}

func (bms *BackgroundMusicSystem) loadMusicAsset(track, path string) error {
	data, err := soundFiles.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read music %s: %w", path, err)
	}
	metadata := musicTrackMetadataMap[track]
	window := productionLibrary().Recordings[metadata.RecordingID].PlaybackWindow
	stream, err := decodeRecording(data, path, bms.audioContext.SampleRate(), window, false)
	if err != nil {
		return fmt.Errorf("decode music %s: %w", path, err)
	}
	length, err := decodedStreamLength(stream)
	if err != nil {
		return err
	}
	bms.musicAssets[track] = &MusicAsset{
		Name: track,
		StreamFactory: func() io.ReadSeeker {
			fresh, err := decodeRecording(data, path, bms.audioContext.SampleRate(), window, false)
			if err != nil {
				logging.Error(logging.CategoryAudio, "Failed to recreate music %s: %v", path, err)

				return &silentStream{position: 0}
			}

			return fresh
		},
		Duration:       length * millisecondsPerSecond / (int64(bms.audioContext.SampleRate()) * decodedPCMBytesPerFrame),
		VolumeModifier: metadata.VolumeModifier * productionLibrary().recordingCalibration(metadata.RecordingID),
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
	bms.stopLocked()
	if !bms.assetsLoaded {
		return fmt.Errorf("%w: music assets not loaded", ErrAudioLibrary)
	}
	metadata, known := GetMusicTrackMetadata(track)
	if !known {
		return fmt.Errorf("%w: unknown music track %s", ErrAudioLibrary, track)
	}
	if productionLibrary().Recordings[metadata.RecordingID].File == nil {
		return nil
	}
	asset, exists := bms.musicAssets[track]
	if !exists {
		return fmt.Errorf("%w: track %s was not loaded", ErrAudioLibrary, track)
	}
	position = boundedMusicPosition(position, asset.Duration)
	stream := asset.StreamFactory()
	if stream == nil {
		return fmt.Errorf("%w: music stream unavailable for %s", ErrAudioLibrary, track)
	}
	player, err := newGainPlayer(bms.audioContext, stream)
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
