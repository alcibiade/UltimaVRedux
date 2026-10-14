package audio

import (
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

func TestVoiceHandlesAreManagerScopedAndCannotStopAnotherInstance(t *testing.T) {
	first, err := NewAudioManager()
	require.NoError(t, err)
	second, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, first.Close()); require.NoError(t, second.Close()) })
	request := game_state.SoundPlaybackRequest{Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 1}
	one, err := first.PlayVoice(request)
	require.NoError(t, err)
	two, err := first.PlayVoice(request)
	require.NoError(t, err)
	foreign, err := second.PlayVoice(request)
	require.NoError(t, err)
	require.Equal(t, one.sequence, foreign.sequence, "same numeric sequence is deliberately possible across managers")
	require.False(t, first.StopVoice(foreign, 0))
	require.True(t, first.StopVoice(one, 0))
	require.False(t, first.StopVoice(one, 0))
	require.Len(t, first.soundEffectPlayer.activeSounds, 1)
	require.Len(t, second.soundEffectPlayer.activeSounds, 1)
	two.Stop(0)
	require.Empty(t, first.soundEffectPlayer.activeSounds)
	VoiceHandle{}.Stop(0)
}

func TestLifetimeOwnerCleanupProtectsOtherOwnersAndLegacyStopRemainsBroad(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	first, second := manager.NewVoiceOwner(), manager.NewVoiceOwner()
	request := game_state.SoundPlaybackRequest{Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 1}
	_, err = first.Play(request)
	require.NoError(t, err)
	request.Effect = game_state.SoundWind
	_, err = first.Play(request)
	require.NoError(t, err)
	other, err := second.Play(request)
	require.NoError(t, err)
	require.NoError(t, manager.PlaySound(request))
	require.Len(t, manager.soundEffectPlayer.activeSounds, 4)
	require.NoError(t, first.Close())
	require.NoError(t, first.Close())
	require.Len(t, manager.soundEffectPlayer.activeSounds, 2)
	_, err = first.Play(request)
	require.ErrorIs(t, err, ErrVoiceOwnerClosed)
	manager.StopSound(game_state.SoundStopRequest{Effect: game_state.SoundWind})
	require.Empty(t, manager.soundEffectPlayer.activeSounds)
	require.False(t, manager.StopVoice(other, 0))
	require.NoError(t, second.Close())
}

func TestSameEffectReplacementOnlyReplacesTheRequestingLifetime(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	owner := manager.NewVoiceOwner()
	request := game_state.SoundPlaybackRequest{Effect: game_state.SoundWalkMetal1, Mode: game_state.SoundPlaybackOneShot, Volume: 1}
	first, err := owner.Play(request)
	require.NoError(t, err)
	independent, err := manager.PlayVoice(request)
	require.NoError(t, err)
	replacement, err := owner.Play(request)
	require.NoError(t, err)
	require.False(t, manager.StopVoice(first, 0))
	require.Len(t, manager.soundEffectPlayer.activeSounds, 2)
	require.NoError(t, manager.PlaySound(request))
	require.NoError(t, manager.PlaySound(request))
	require.Len(t, manager.soundEffectPlayer.activeSounds, 3)
	require.True(t, manager.StopVoice(independent, 0))
	require.True(t, manager.StopVoice(replacement, 0))
}

func TestOwnedReleaseIsIdempotentAndCloseFinishesPendingFades(t *testing.T) {
	now := time.Time{}
	config, err := NewAudioConfiguration()
	require.NoError(t, err)
	manager, err := NewAudioManagerWithRuntime(config, getSharedAudioContext(), func() time.Time { return now })
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	owner := manager.NewVoiceOwner()
	voice, err := owner.Play(game_state.SoundPlaybackRequest{Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 1})
	require.NoError(t, err)
	voice.Stop(1000)
	now = now.Add(500 * time.Millisecond)
	voice.Stop(1000)
	now = now.Add(500 * time.Millisecond)
	manager.Update()
	require.Empty(t, manager.soundEffectPlayer.activeSounds)
	_, err = owner.Play(game_state.SoundPlaybackRequest{Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 1})
	require.NoError(t, err)
	owner.Stop(1000)
	require.Len(t, manager.soundEffectPlayer.activeSounds, 1)
	require.NoError(t, owner.Close())
	require.Empty(t, manager.soundEffectPlayer.activeSounds)
	_, err = owner.Play(game_state.SoundPlaybackRequest{Effect: game_state.SoundHeal, Volume: 1})
	require.ErrorIs(t, err, ErrVoiceOwnerClosed)
}

func TestInaudibleFiniteRequestsDoNotPrepareEvictOrAdvanceSelection(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	for range 8 {
		require.NoError(t, manager.PlaySoundEffect(game_state.SoundHeal))
	}
	player := manager.soundEffectPlayer
	before := player.nextSoundSequence
	for _, asset := range player.soundAssets[game_state.SoundWalkMetal1] {
		asset.StreamFactory = func(game_state.SoundPlaybackMode) (io.ReadSeeker, error) {
			t.Fatal("inaudible request decoded a new stream")
			return nil, nil
		}
	}
	voice, err := manager.PlayVoice(game_state.SoundPlaybackRequest{Effect: game_state.SoundWalkMetal1, Volume: 0})
	require.NoError(t, err)
	require.Equal(t, VoiceHandle{}, voice)
	require.NoError(t, manager.SetSoundEffectVolume(0))
	require.NoError(t, manager.PlaySoundEffect(game_state.SoundWalkMetal1))
	require.Equal(t, before, player.nextSoundSequence)
	require.Zero(t, player.nextVariant[game_state.SoundWalkMetal1])
	require.Len(t, player.activeSounds, 8)
}

func TestOwnershipCloseAndPlaybackAreSerialized(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	owner := manager.NewVoiceOwner()
	var work sync.WaitGroup
	work.Add(3)
	go func() {
		defer work.Done()
		for range 30 {
			voice, err := owner.Play(game_state.SoundPlaybackRequest{Effect: game_state.SoundHeal, Volume: 1})
			if err != nil {
				require.True(t, err == ErrVoiceOwnerClosed || err == ErrAudioClosed)
				continue
			}
			voice.Stop(20)
		}
	}()
	go func() {
		defer work.Done()
		for range 30 {
			manager.Update()
			owner.Stop(20)
		}
	}()
	go func() { defer work.Done(); require.NoError(t, manager.Close()) }()
	work.Wait()
	require.NoError(t, owner.Close())
	require.Empty(t, manager.soundEffectPlayer.activeSounds)
}

func TestVoiceHandlesDoNotAliasAcrossDisableAndTerminalClose(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	owner := manager.NewVoiceOwner()
	request := game_state.SoundPlaybackRequest{Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 1}
	old, err := owner.Play(request)
	require.NoError(t, err)
	manager.SetEnabled(false)
	manager.SetEnabled(true)
	current, err := owner.Play(request)
	require.NoError(t, err)
	require.NotEqual(t, old, current)
	require.False(t, manager.StopVoice(old, 0))
	require.Len(t, manager.soundEffectPlayer.activeSounds, 1)
	require.NoError(t, manager.Close())
	require.False(t, manager.StopVoice(current, 0))
	_, err = manager.PlayVoice(request)
	require.ErrorIs(t, err, ErrAudioClosed)
	_, err = owner.Play(request)
	require.ErrorIs(t, err, ErrAudioClosed)
	require.NoError(t, owner.Close())
}
