package audio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func admissionFixture(t *testing.T, count int) (*AudioManager, []AmbientSoundSource, *time.Time) {
	t.Helper()
	now := time.Unix(100, 0)
	config, err := NewAudioConfiguration()
	require.NoError(t, err)
	manager, err := NewAudioManagerWithRuntime(config, getSharedAudioContext(), func() time.Time { return now })
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	manager.ambientSounds.openStream = func(string) (io.ReadSeeker, error) {
		return benchmarkPCM{bytes.NewReader(make([]byte, 256))}, nil
	}
	sources := make([]AmbientSoundSource, count)
	for index := range sources {
		sources[index] = AmbientSoundSource{ID: fmt.Sprintf("test/%03d", index), SoundType: AmbientFire,
			IsActive: true, IsDynamic: true, SoundFile: "synthetic.pcm", MaxDistance: 8, baseVolume: 1, VolumeModifier: 1, FadeOutTime: time.Second}
	}
	return manager, sources, &now
}

func TestAmbientBudgetRankingAndSourceOrderAreDeterministic(t *testing.T) {
	manager, sources, _ := admissionFixture(t, 32)
	manager.UpdateAmbientSoundsWithDynamicSources(references.Position{}, nil, sources)
	require.Len(t, manager.ambientSounds.activeSounds, AmbientVoiceLimit)
	before := make(map[AmbientSoundType]*AmbientSoundInstance)
	for key, voice := range manager.ambientSounds.activeSounds {
		before[key] = voice
	}
	for index := range AmbientVoiceLimit {
		require.Contains(t, before, AmbientSoundType(fmt.Sprintf("source/test/%03d", index)))
	}
	slices.Reverse(sources)
	for range 10 {
		manager.UpdateAmbientSoundsWithDynamicSources(references.Position{}, nil, sources)
		manager.Update()
	}
	require.Equal(t, before, manager.ambientSounds.activeSounds)
	require.NoError(t, manager.SetAmbientVolume(0))
	manager.Update()
	require.Equal(t, before, manager.ambientSounds.activeSounds, "category mute preserves loop continuity")
	for _, voice := range before {
		require.Zero(t, voice.Player.Volume())
	}
}

func TestAmbientReleaseTailsCountAgainstBudget(t *testing.T) {
	manager, sources, now := admissionFixture(t, AmbientVoiceLimit)
	manager.UpdateAmbientSoundsWithDynamicSources(references.Position{}, nil, sources)
	stronger := sources[0]
	stronger.ID, stronger.baseVolume = "stronger", 2
	sources = append(sources, stronger)
	manager.UpdateAmbientSoundsWithDynamicSources(references.Position{}, nil, sources)
	require.Len(t, manager.ambientSounds.activeSounds, AmbientVoiceLimit)
	require.NotContains(t, manager.ambientSounds.activeSounds, AmbientSoundType("source/stronger"))
	*now = now.Add(time.Second)
	manager.Update()
	require.Len(t, manager.ambientSounds.activeSounds, AmbientVoiceLimit)
	require.Contains(t, manager.ambientSounds.activeSounds, AmbientSoundType("source/stronger"))
}

func TestAmbientBackoffDoesNotReserveAPlaybackSlot(t *testing.T) {
	manager, sources, _ := admissionFixture(t, AmbientVoiceLimit+1)
	sources[0].SoundFile, sources[0].baseVolume = "broken", 2
	open := manager.ambientSounds.openStream
	attempts := 0
	manager.ambientSounds.openStream = func(file string) (io.ReadSeeker, error) {
		if file == "broken" {
			attempts++
			return nil, errors.New("decoder unavailable")
		}
		return open(file)
	}
	manager.UpdateAmbientSoundsWithDynamicSources(references.Position{}, nil, sources)
	manager.Update()
	require.Len(t, manager.ambientSounds.activeSounds, AmbientVoiceLimit)
	require.Equal(t, 1, attempts)
	require.NotContains(t, manager.ambientSounds.activeSounds, AmbientSoundType("source/test/000"))
}

func TestStableEmitterIdentitySurvivesProjectedCoordinateChange(t *testing.T) {
	manager, sources, _ := admissionFixture(t, 1)
	sources[0].Position.X = -1
	manager.UpdateAmbientSoundsWithDynamicSources(references.Position{}, nil, sources)
	voice := manager.ambientSounds.activeSounds["source/test/000"]
	require.NotNil(t, voice)
	sources[0].Position.X = 255
	manager.UpdateAmbientSoundsWithDynamicSources(references.Position{X: 255}, nil, sources)
	require.Same(t, voice, manager.ambientSounds.activeSounds["source/test/000"])
	require.Len(t, manager.ambientSounds.activeSounds, 1)
}

func TestNonSpatialAmbientSourceIgnoresStaleSourceCoordinates(t *testing.T) {
	manager, sources, _ := admissionFixture(t, 1)
	sources[0].MaxDistance = 0
	sources[0].Position = references.Position{X: 200, Y: 100}
	manager.UpdateAmbientSoundsWithDynamicSources(references.Position{}, nil, sources)
	voice := manager.ambientSounds.activeSounds["background/ambient/AmbientFire"]
	require.NotNil(t, voice)
	require.Zero(t, voice.LastDistance)
	manager.UpdateAmbientSoundsWithDynamicSources(references.Position{X: 17, Y: 99}, nil, sources)
	require.Same(t, voice, manager.ambientSounds.activeSounds["background/ambient/AmbientFire"])
	require.Equal(t, 1.0, voice.Volume)
}

func TestBardOwnershipUsesEmitterRoleRatherThanIdentitySpelling(t *testing.T) {
	manager, sources, _ := admissionFixture(t, 1)
	sources[0].ID, sources[0].Emitter = "stable-independent-id", "bard"
	manager.UpdateAmbientSoundsWithDynamicSources(references.Position{}, nil, sources)
	require.True(t, manager.ambientSounds.hasBardMusic())
	require.NoError(t, manager.SetAmbientVolume(0))
	require.False(t, manager.ambientSounds.hasBardMusic())
}
