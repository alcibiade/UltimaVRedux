package audio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestLighthouseLocationsUseExistingCueWithoutChangingOverworldRadius(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	definition, exists := manager.GetAmbientSoundDefinition(AmbientLighthouse)
	require.True(t, exists)
	require.Equal(t, 10.0, definition.MaxDistance)
	require.Equal(t, "seagulls_01", productionLibrary().Cues[string(AmbientLighthouse)].RecordingID)
	for _, location := range []references.Location{references.Fogsbane, references.Stormcrow, references.Greyhaven, references.Waveguide} {
		// Floor is deliberately not an eligibility argument; the location owns the bed.
		for _, listener := range []references.Position{{X: 1, Y: 1}, {X: 16, Y: 16}, {X: 31, Y: 31}} {
			source, active := manager.LighthouseAmbientSource(location, listener)
			require.True(t, active, location.String())
			require.Equal(t, listener, source.Position)
			require.Zero(t, source.MaxDistance)
			require.Equal(t, definition.SoundType, source.SoundType)
			require.Equal(t, definition.FileName, source.SoundFile)
			require.Equal(t, definition.BaseVolume, source.GetBaseVolume())
			require.Equal(t, definition.FadeInTime, source.FadeInTime)
			require.Equal(t, definition.FadeOutTime, source.FadeOutTime)
		}
	}
	for _, location := range []references.Location{references.Britain, references.BritanniaUnderworld, references.CombatLocation, references.Deceit} {
		_, active := manager.LighthouseAmbientSource(location, references.Position{})
		require.False(t, active)
	}
}

func TestLighthouseSharesTileVoiceAcrossMovementFloorAndExit(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	ambient := manager.ambientSounds
	now := time.Unix(100, 0)
	ambient.now = func() time.Time { return now }
	require.NoError(t, manager.SetAmbientVolume(1))
	listener := references.Position{X: 4, Y: 4}
	// Tile 27 is the production lighthouse mapping; start outside with its spatial voice.
	tiles := []TileWithPosition{{Tile: &references.Tile{Index: indexes.SpriteIndex(27)}, Position: listener}}
	manager.UpdateAmbientSounds(listener, tiles)
	voice := ambient.activeSounds[AmbientLighthouse]
	require.NotNil(t, voice)
	now = now.Add(voice.Definition.FadeInTime)
	for _, location := range []references.Location{references.Fogsbane, references.Stormcrow, references.Greyhaven, references.Waveguide} {
		for floor := range 3 {
			listener = references.Position{X: references.Coordinate(floor * 7), Y: references.Coordinate(floor * 5)}
			tiles[0].Position = listener // Even a co-located static source cannot duplicate the interior bed.
			source, exists := manager.LighthouseAmbientSource(location, listener)
			require.True(t, exists)
			manager.UpdateAmbientSoundsWithDynamicSources(listener, tiles, []AmbientSoundSource{source})
			require.Len(t, ambient.activeSounds, 1, "tile plus location must share one voice")
			require.Same(t, voice, ambient.activeSounds[AmbientLighthouse])
			require.Zero(t, voice.Definition.MaxDistance)
			require.InDelta(t, source.GetBaseVolume(), voice.Player.Volume(), 1e-9)
		}
	}
	require.NoError(t, manager.SetAmbientVolume(0))
	require.Zero(t, voice.Player.Volume())
	require.NoError(t, manager.SetAmbientVolume(0.5))
	require.InDelta(t, voice.Definition.BaseVolume*0.5, voice.Player.Volume(), 1e-9)
	manager.UpdateAmbientSounds(tiles[0].Position, tiles)
	require.Len(t, ambient.activeSounds, 1)
	require.Same(t, voice, ambient.activeSounds[AmbientLighthouse])
	require.Equal(t, 10.0, voice.Definition.MaxDistance, "outside resumes existing proximity range")
	outside := references.Position{X: tiles[0].Position.X + 5, Y: tiles[0].Position.Y}
	manager.UpdateAmbientSounds(outside, tiles)
	require.Same(t, voice, ambient.activeSounds[AmbientLighthouse])
	require.InDelta(t, voice.Definition.BaseVolume*AmbientDistanceGain(5, 10)*0.5, voice.Player.Volume(), 1e-9,
		"the interior's full-volume source must not dominate after exit")
	require.Less(t, voice.Player.Volume(), voice.Definition.BaseVolume*0.5)
	outside.X = tiles[0].Position.X + 11
	manager.UpdateAmbientSounds(outside, tiles)
	require.Equal(t, AmbientFadeOut, voice.FadeState)
	now = now.Add(voice.Definition.FadeOutTime)
	manager.UpdateAmbientSounds(outside, tiles)
	require.Empty(t, ambient.activeSounds, "an out-of-range static tile must not retain the interior source")
	source, _ := manager.LighthouseAmbientSource(references.Fogsbane, listener)
	manager.UpdateAmbientSoundsWithDynamicSources(listener, nil, []AmbientSoundSource{source})
	require.Len(t, ambient.activeSounds, 1)
	manager.SetAmbientSuppressed(true)
	manager.UpdateAmbientSoundsWithDynamicSources(listener, nil, []AmbientSoundSource{source})
	require.Empty(t, ambient.activeSounds)
	manager.SetAmbientSuppressed(false)
	manager.UpdateAmbientSoundsWithDynamicSources(listener, nil, []AmbientSoundSource{source})
	require.Len(t, ambient.activeSounds, 1)
}
