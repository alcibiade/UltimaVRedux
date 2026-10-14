package audio

import (
	"errors"
	"sync"
	"testing"
	"time"

	eaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestHostAudioContextIsBorrowed(t *testing.T) {
	// Run alone as well as in the suite: a fresh test process proves host-first
	// initialization at a nondefault rate without resetting a backend singleton.
	host := eaudio.CurrentContext()
	if host == nil {
		host = eaudio.NewContext(48000)
	}
	require.Same(t, host, getSharedAudioContext())
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.Same(t, host, manager.sharedContext)
	require.Same(t, host, manager.ambientSounds.audioContext)
	require.NoError(t, manager.PlaySoundEffect(game_state.SoundHeal))
	require.Equal(t, host.SampleRate(), manager.soundEffectPlayer.soundAssets[game_state.SoundHeal][0].SampleRate)
}

func TestAudioConfigurationSnapshotDoesNotMutateLiveMix(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	snapshot := manager.GetConfiguration()
	snapshot.SoundEffectVolume = 0
	require.NotZero(t, manager.GetConfiguration().SoundEffectVolume)
}

func TestAmbientPreviewUsesAuthoredGainAndLiveMute(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.PlayAmbientSoundPreview())
	require.NotNil(t, manager.ambientPreviewPlayer)
	cue := productionLibrary().Cues[string(AmbientForest)]
	base := cue.VolumeModifier * productionLibrary().recordingCalibration(cue.RecordingID)
	require.InDelta(t, manager.GetConfiguration().AmbientVolume*base, manager.ambientPreviewPlayer.Volume(), 1e-9)
	require.NoError(t, manager.SetAmbientVolume(0))
	require.Zero(t, manager.ambientPreviewPlayer.Volume())
	require.NoError(t, manager.SetAmbientVolume(0.5))
	require.InDelta(t, 0.5*base, manager.ambientPreviewPlayer.Volume(), 1e-9)
	manager.SetEnabled(false)
	require.Nil(t, manager.ambientPreviewPlayer, "global disable owns preview retirement too")
}

func TestAudioCloseIsTerminalForColdAndWarmManagers(t *testing.T) {
	for _, warm := range []bool{false, true} {
		t.Run(map[bool]string{false: "cold", true: "warm"}[warm], func(t *testing.T) {
			manager, err := NewAudioManager()
			require.NoError(t, err)
			if warm {
				require.NoError(t, manager.PlaySoundEffect(game_state.SoundHeal))
			}
			require.NoError(t, manager.Close())
			require.NoError(t, manager.Close())
			manager.SetEnabled(true)
			require.False(t, manager.IsEnabled())
			require.Error(t, manager.PlaySoundEffect(game_state.SoundHeal))
			require.Error(t, manager.PlayAmbientSoundPreview())
			require.Error(t, manager.SetAmbientVolume(0.5))
			require.Empty(t, manager.soundEffectPlayer.activeSounds)
		})
	}
}

func TestFailedSailingPresentationDoesNotRetryUntilNextHoist(t *testing.T) {
	policy, player, advance := newPolicyTest(t)
	player.fail = errors.New("unavailable stream")
	policy.setSailing(true, "hornpipe")
	require.ErrorIs(t, policy.update(), player.fail)
	for range 1000 {
		advance(time.Second)
	}
	require.Equal(t, []string{"hornpipe"}, player.starts)
	policy.setSailing(false, "hornpipe")
	policy.setSailing(true, "hornpipe")
	require.ErrorIs(t, policy.update(), player.fail)
	require.Equal(t, []string{"hornpipe", "hornpipe"}, player.starts)
}

func TestSailingAlonePreparesColdAudioAssets(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	manager.musicManager.policy.exploration = ""
	manager.musicManager.policy.setSailing(true, "hornpipe")
	require.False(t, manager.assetsLoaded)
	manager.Update()
	require.True(t, manager.assetsLoaded)
	require.Equal(t, "hornpipe", manager.musicManager.policy.track)
}

func TestManagerBindsAndDetachesPositionalConfiguration(t *testing.T) {
	configuration, err := NewAudioConfiguration()
	require.NoError(t, err)
	configuration.MaxAudioDistance = 2
	manager, err := newAudioManagerWithConfig(configuration)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	configuration.MaxAudioDistance = 100
	volume, _ := manager.positionalCalculator.CalculateAudio(0, 0, 3, 0)
	require.Zero(t, volume)
	require.Equal(t, float64(2), manager.GetConfiguration().MaxAudioDistance)
}

func TestWalkingCycleContinuesAcrossFormerCounterReset(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	walking := manager.walkingSounds
	walking.stepCounter = 999
	var tile *references.Tile
	// Resolve a wood tile from the actual routing rather than rely on a visual alias.
	for sprite := range *references.NewTileReferences() {
		candidate := &references.Tile{Index: sprite}
		if walking.getSurfaceTypeForTile(candidate) == SurfaceWood {
			tile = candidate
			break
		}
	}
	require.NotNil(t, tile)
	require.Equal(t, game_state.SoundWalkWood1, walking.getSoundEffectForTile(tile))
	require.NoError(t, walking.PlayWalkingSound(tile))
	require.Equal(t, game_state.SoundWalkWood2, walking.getSoundEffectForTile(tile))
}

func TestConcurrentWalkingAndConfigurationSnapshots(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	var workers sync.WaitGroup
	for range 4 {
		workers.Go(func() {
			for range 20 {
				if err := manager.PlayWalkingSound(nil); err != nil {
					t.Error(err)
				}
				snapshot := manager.GetConfiguration()
				snapshot.SoundEffectVolume = 0
			}
		})
	}
	workers.Wait()
	require.EqualValues(t, 80, manager.walkingSounds.stepCounter)
}
