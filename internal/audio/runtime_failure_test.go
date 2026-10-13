package audio

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

type faultPCM struct {
	*bytes.Reader
	failure, seekFailure error
	empty                bool
	closes, reads        int
}

func (source *faultPCM) Read(buffer []byte) (int, error) {
	source.reads++
	if source.empty {
		return 0, nil
	}
	count, err := source.Reader.Read(buffer)
	if source.Reader.Len() == 0 && source.failure != nil {
		err = source.failure
		source.failure = nil // Deliberately report the error just once.
	}
	return count, err
}
func (source *faultPCM) Seek(offset int64, whence int) (int64, error) {
	if source.seekFailure != nil {
		return 0, source.seekFailure
	}
	return source.Reader.Seek(offset, whence)
}
func (source *faultPCM) Close() error  { source.closes++; return nil }
func (source *faultPCM) Length() int64 { return source.Reader.Size() }

func TestRuntimeDecoderBoundaryKeepsFailuresFiniteAndObservable(t *testing.T) {
	broken := errors.New("decoder broke")
	for _, test := range []struct {
		name    string
		pcm     []byte
		failure error
		empty   bool
		want    error
	}{
		{"partial read failure", []byte{1, 2, 3, 4}, broken, false, broken},
		{"truncated PCM", []byte{1}, nil, false, ErrAudioLibrary},
		{"decoder unexpected EOF", []byte{1, 2, 3, 4}, io.ErrUnexpectedEOF, false, io.ErrUnexpectedEOF},
		{"no progress", nil, nil, true, io.ErrNoProgress},
		{"natural EOF", []byte{1, 2, 3, 4}, nil, false, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := &faultPCM{Reader: bytes.NewReader(test.pcm), failure: test.failure, empty: test.empty}
			player, err := newGainPlayer(getSharedAudioContext(), source)
			require.NoError(t, err)
			// Drive the decoder boundary, not a native device; one-byte reads
			// additionally exercise pending calibrated PCM/error delivery.
			var delivered []byte
			for range 10 {
				buffer := make([]byte, 1)
				count, err := player.boundary.Read(buffer)
				delivered = append(delivered, buffer[:count]...)
				if err != nil {
					require.ErrorIs(t, err, io.EOF, "decoder errors must not reach the context error hook")
					break
				}
			}
			if test.want == nil {
				require.NoError(t, player.Failure())
			} else {
				require.ErrorIs(t, player.Failure(), test.want)
			}
			if len(test.pcm)%2 == 0 {
				require.Equal(t, test.pcm, delivered)
			}
			before := source.reads
			_, err = player.boundary.Read(make([]byte, 4))
			require.ErrorIs(t, err, io.EOF)
			if test.want != nil {
				require.Equal(t, before, source.reads, "failed decoder is terminal")
			}
			require.NoError(t, player.Close())
			require.Equal(t, 1, source.closes)
		})
	}
}

func TestRuntimeSeekFailureIsTerminalAndPublishedConcurrently(t *testing.T) {
	broken := errors.New("decoder seek failure")
	source := &faultPCM{Reader: bytes.NewReader([]byte{1, 2, 3, 4})}
	player, err := newGainPlayer(getSharedAudioContext(), source)
	require.NoError(t, err)
	source.seekFailure = broken
	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		<-start
		_, _ = player.boundary.Seek(0, io.SeekStart)
	}()
	close(start)
	for range 1000 {
		if failure := player.Failure(); failure != nil {
			require.ErrorIs(t, failure, broken)
		}
	}
	workers.Wait()
	require.ErrorIs(t, player.Failure(), broken)
	_, err = player.boundary.Read(make([]byte, 4))
	require.ErrorIs(t, err, io.EOF)
	require.Zero(t, source.reads)
	require.NoError(t, player.Close())
	require.Equal(t, 1, source.closes)
}

func TestFailedRuntimeVoicesRetireWithoutStoppingOtherSounds(t *testing.T) {
	for _, mode := range []game_state.SoundPlaybackMode{game_state.SoundPlaybackOneShot, game_state.SoundPlaybackLoop} {
		t.Run(strconv.Itoa(int(mode)), func(t *testing.T) {
			manager, err := NewAudioManager()
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, manager.Close()) })
			require.NoError(t, manager.PlaySoundEffect(game_state.SoundHeal))
			broken := errors.New("late SFX read")
			source := &faultPCM{Reader: bytes.NewReader([]byte{1, 2, 3, 4}), failure: broken}
			manager.soundEffectPlayer.soundAssets[game_state.SoundEarthquake][0].StreamFactory =
				func(game_state.SoundPlaybackMode) (io.ReadSeeker, error) { return source, nil }
			require.NoError(t, manager.PlaySound(game_state.SoundPlaybackRequest{Effect: game_state.SoundEarthquake, Mode: mode, Volume: 1}))
			failed := activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundEarthquake]
			_, err = failed.Player.boundary.Read(make([]byte, 8))
			require.ErrorIs(t, err, io.EOF)
			require.ErrorIs(t, failed.Player.Failure(), broken)
			manager.Update()
			manager.Update()
			require.Equal(t, 1, source.closes)
			require.NotContains(t, activeSoundsByEffect(manager.soundEffectPlayer), game_state.SoundEarthquake)
			require.Contains(t, activeSoundsByEffect(manager.soundEffectPlayer), game_state.SoundHeal)
		})
	}
}

func TestMusicPreparationFailurePreservesCurrentVoice(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.PlayBackgroundMusic("town"))
	previous := manager.backgroundMusic.player
	broken := errors.New("music decoder unavailable")
	manager.backgroundMusic.musicAssets["combat"].StreamFactory = func() (io.ReadSeeker, error) { return nil, broken }
	require.ErrorIs(t, manager.PlayBackgroundMusic("combat"), broken)
	require.Same(t, previous, manager.backgroundMusic.player)
	require.True(t, previous.IsPlaying())
	require.False(t, previous.retired)
}

func TestPlayerCreationFailurePreservesCurrentSFXAndClosesRejectedSource(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	effect := game_state.SoundWalkMetal1
	require.NoError(t, manager.PlaySoundEffect(effect))
	before := activeSoundsByEffect(manager.soundEffectPlayer)[effect]
	variant := manager.soundEffectPlayer.nextVariant[effect]
	broken := errors.New("initial decoder seek failed")
	source := &faultPCM{Reader: bytes.NewReader([]byte{1, 2, 3, 4}), seekFailure: broken}
	for _, asset := range manager.soundEffectPlayer.soundAssets[effect] {
		asset.StreamFactory = func(game_state.SoundPlaybackMode) (io.ReadSeeker, error) { return source, nil }
	}
	require.ErrorIs(t, manager.PlaySoundEffect(effect), broken)
	require.Equal(t, 1, source.closes)
	require.Equal(t, variant, manager.soundEffectPlayer.nextVariant[effect])
	require.Same(t, before.Player, activeSoundsByEffect(manager.soundEffectPlayer)[effect].Player)
	require.False(t, before.Player.retired)
}

func TestAmbientReplacementWaitsForPreparationAndClosesOldDecoderOnce(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	ambient := manager.ambientSounds
	now := time.Unix(100, 0)
	ambient.now = func() time.Time { return now }
	definition := AmbientSoundDefinition{SoundType: "ambient/test", FileName: "first.pcm", BaseVolume: 1, MaxDistance: 8, LoopSound: true}
	ambient.AddCustomAmbientSound(indexes.FireLogs, definition)
	tiles := []TileWithPosition{{Tile: &references.Tile{Index: indexes.FireLogs}}}
	first := &faultPCM{Reader: bytes.NewReader([]byte{1, 2, 3, 4})}
	ambient.openStream = func(string) (io.ReadSeeker, error) { return first, nil }
	manager.UpdateAmbientSounds(references.Position{}, tiles)
	voice := ambient.activeSounds[definition.SoundType]
	require.NotNil(t, voice)
	definition.FileName = "second.pcm"
	ambient.AddCustomAmbientSound(indexes.FireLogs, definition)
	ambient.openStream = func(string) (io.ReadSeeker, error) { return nil, errors.New("replacement unavailable") }
	manager.UpdateAmbientSounds(references.Position{}, tiles)
	require.Same(t, voice, ambient.activeSounds[definition.SoundType])
	require.Zero(t, first.closes)
	require.True(t, voice.Player.IsPlaying())
	second := &faultPCM{Reader: bytes.NewReader([]byte{4, 3, 2, 1})}
	ambient.openStream = func(string) (io.ReadSeeker, error) { return second, nil }
	now = now.Add(time.Second)
	manager.Update()
	require.NotSame(t, voice, ambient.activeSounds[definition.SoundType])
	require.Equal(t, 1, first.closes)
	require.Empty(t, ambient.Failures())
	manager.SetAmbientSuppressed(true)
	require.Equal(t, 1, second.closes)
	manager.SetAmbientSuppressed(true)
	require.Equal(t, 1, second.closes)
}

func TestAmbientPreviewFailureRetiresOnService(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.PlayAmbientSoundPreview())
	voice := manager.ambientPreviewPlayer
	require.NotNil(t, voice)
	voice.boundary.failure.Store(&streamFailure{err: errors.New("ambient preview decoder")})
	manager.Update()
	require.Nil(t, manager.ambientPreviewPlayer)
	require.True(t, voice.retired)
}

func TestAmbientInvalidLoopLengthClosesSourceAndSchedulesRetry(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	ambient := manager.ambientSounds
	definition := AmbientSoundDefinition{SoundType: "ambient/test", FileName: "short.pcm", BaseVolume: 1,
		MaxDistance: 8, LoopSound: true}
	ambient.AddCustomAmbientSound(indexes.FireLogs, definition)
	source := &faultPCM{Reader: bytes.NewReader([]byte{1})}
	ambient.openStream = func(string) (io.ReadSeeker, error) { return source, nil }
	require.NotPanics(t, func() {
		manager.UpdateAmbientSounds(references.Position{}, []TileWithPosition{{Tile: &references.Tile{Index: indexes.FireLogs}}})
	})
	require.Equal(t, 1, source.closes)
	require.Empty(t, ambient.activeSounds)
	require.Contains(t, ambient.Failures()[definition.SoundType].Message, "whole-frame")
}

func TestMusicServiceRetainsPauseButRetiresFailuresDuringHolds(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.PlayMusicForLocation(references.Britain))
	policy := manager.musicManager.policy
	manager.GetMusicManager().SetSuppressed(MusicMenu, true)
	paused := manager.backgroundMusic.player
	manager.Update()
	require.Same(t, paused, manager.backgroundMusic.player)
	require.True(t, policy.suspended)
	broken := errors.New("paused decoder failure")
	paused.boundary.failure.Store(&streamFailure{err: broken})
	manager.SetEnabled(false)
	manager.Update()
	require.Nil(t, manager.backgroundMusic.player)
	require.True(t, paused.retired)
	require.Equal(t, "town", policy.track, "held score can resume from a fresh stream after repair")
	manager.SetEnabled(true)
	manager.GetMusicManager().SetSuppressed(MusicMenu, false)
	manager.Update()
	require.NotNil(t, manager.backgroundMusic.player)
	require.NotSame(t, paused, manager.backgroundMusic.player)
}

func TestMusicLateFailureConsumesFinitePresentation(t *testing.T) {
	for _, kind := range []string{"exploration", "combat", "sailing", "cue"} {
		t.Run(kind, func(t *testing.T) {
			now := time.Unix(100, 0)
			config, err := NewAudioConfiguration()
			require.NoError(t, err)
			manager, err := NewAudioManagerWithRuntime(config, getSharedAudioContext(), func() time.Time { return now })
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, manager.Close()) })
			require.NoError(t, manager.ensureAssetsLoaded())
			policy := manager.musicManager.policy
			policy.exploration = "town"
			switch kind {
			case "combat":
				policy.setCombatEncounter(1)
			case "sailing":
				policy.setSailing(true, "hornpipe")
			case "cue":
				policy.cue = "victory"
			}
			manager.Update()
			voice := manager.backgroundMusic.player
			require.NotNil(t, voice)
			voice.boundary.failure.Store(&streamFailure{err: errors.New("late score failure")})
			manager.Update()
			require.True(t, voice.retired)
			require.Nil(t, manager.backgroundMusic.player)
			for range 60 {
				now = now.Add(time.Second)
				manager.Update()
				require.Nil(t, manager.backgroundMusic.player)
			}
			if kind == "combat" {
				require.True(t, policy.combatFinished)
			}
			if kind == "sailing" {
				require.True(t, policy.sailingPlayed)
			}
		})
	}
}

func TestAmbientFailuresBackOffAndForgetDepartedSources(t *testing.T) {
	now := time.Unix(100, 0)
	config, err := NewAudioConfiguration()
	require.NoError(t, err)
	manager, err := NewAudioManagerWithRuntime(config, getSharedAudioContext(), func() time.Time { return now })
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	ambient := manager.ambientSounds
	definition := AmbientSoundDefinition{SoundType: "ambient/test", FileName: "test.pcm", BaseVolume: 1, MaxDistance: 8, LoopSound: true}
	ambient.AddCustomAmbientSound(indexes.FireLogs, definition)
	tiles := []TileWithPosition{{Tile: &references.Tile{Index: indexes.FireLogs}}}
	attempts := 0
	broken := errors.New("ambient unavailable")
	ambient.openStream = func(string) (io.ReadSeeker, error) { attempts++; return nil, broken }
	manager.UpdateAmbientSounds(references.Position{}, tiles)
	for range 100 {
		manager.UpdateAmbientSounds(references.Position{}, tiles)
		manager.Update()
	}
	require.Equal(t, 1, attempts)
	require.Contains(t, ambient.Failures()[definition.SoundType].Message, broken.Error())
	for _, wait := range []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 30 * time.Second, 30 * time.Second} {
		now = now.Add(wait - time.Nanosecond)
		before := attempts
		manager.Update()
		require.Equal(t, before, attempts)
		now = now.Add(time.Nanosecond)
		manager.Update()
		require.Equal(t, before+1, attempts)
	}
	manager.UpdateAmbientSounds(references.Position{}, nil)
	require.Empty(t, ambient.Failures())
	require.Empty(t, ambient.attempts)
	manager.UpdateAmbientSounds(references.Position{}, tiles)
	require.Equal(t, 9, attempts, "a new source-presence epoch gets one immediate attempt")
	manager.SetAmbientSuppressed(true)
	now = now.Add(time.Hour)
	manager.SetAmbientSuppressed(false)
	manager.Update()
	require.Empty(t, ambient.desired, "lifting a hold needs a fresh world snapshot")
	require.Equal(t, 9, attempts)
}

func TestAmbientLateFailureRetriesAndCompletedOneShotStaysComplete(t *testing.T) {
	for _, looping := range []bool{false, true} {
		t.Run(map[bool]string{false: "finite", true: "loop"}[looping], func(t *testing.T) {
			manager, err := NewAudioManager()
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, manager.Close()) })
			ambient := manager.ambientSounds
			now := time.Unix(100, 0)
			ambient.now = func() time.Time { return now }
			definition := AmbientSoundDefinition{SoundType: "ambient/test", FileName: "test.pcm", BaseVolume: 1, MaxDistance: 8, LoopSound: looping}
			ambient.AddCustomAmbientSound(indexes.FireLogs, definition)
			tiles := []TileWithPosition{{Tile: &references.Tile{Index: indexes.FireLogs}}}
			var sources []*faultPCM
			ambient.openStream = func(string) (io.ReadSeeker, error) {
				source := &faultPCM{Reader: bytes.NewReader([]byte{1, 2, 3, 4}), failure: errors.New("late ambient read")}
				sources = append(sources, source)
				return source, nil
			}
			manager.UpdateAmbientSounds(references.Position{}, tiles)
			voice := ambient.activeSounds[definition.SoundType]
			_, err = voice.Player.boundary.Read(make([]byte, 8))
			require.ErrorIs(t, err, io.EOF)
			manager.Update()
			require.Empty(t, ambient.activeSounds)
			require.Equal(t, 1, sources[0].closes)
			now = now.Add(time.Second)
			manager.Update()
			require.Len(t, sources, 2)
			require.Empty(t, ambient.Failures())
			if !looping {
				// Device-independent completion signal; native draining is separate acceptance.
				ambient.activeSounds[definition.SoundType].Player.Pause()
				manager.Update()
				now = now.Add(time.Hour)
				manager.UpdateAmbientSounds(references.Position{}, tiles)
				require.Len(t, sources, 2, "finite source does not restart until departure/re-entry")
				require.Empty(t, ambient.activeSounds)
				manager.UpdateAmbientSounds(references.Position{}, nil)
				manager.UpdateAmbientSounds(references.Position{}, tiles)
				require.Len(t, sources, 3)
			}
		})
	}
}
