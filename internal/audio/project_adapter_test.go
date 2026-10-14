package audio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestReduxAdapterUsesItsInjectedLibraryAcrossRuntimeSubsystems(t *testing.T) {
	base, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, base.Close()) })
	document := base.library.Document()
	heal := document.Cues["sfx/SoundHeal"]
	heal.VolumeModifier = 0.17
	document.Cues["sfx/SoundHeal"] = heal
	music := document.Cues["music/track/town"]
	music.VolumeModifier = 0.23
	document.Cues["music/track/town"] = music
	ambient := document.Cues[string(AmbientFire)]
	ambient.VolumeModifier = 0.13
	document.Cues[string(AmbientFire)] = ambient
	title := "music/track/title"
	document.Routing.Music[locationMusicKey(references.Britain)] = &title
	library, err := NewLibrary(encodeAudioValue(document), soundFiles, LibraryOptions{AssetRoot: "assets", Validate: ReduxLibraryValidator})
	require.NoError(t, err)
	custom, err := NewAudioManagerWithLibrary(base.GetConfiguration(), library, getSharedAudioContext(), time.Now)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, custom.Close()) })
	require.NoError(t, custom.PlaySoundEffect(game_state.SoundHeal))
	healing := activeSoundsByEffect(custom.soundEffectPlayer)[game_state.SoundHeal]
	require.InDelta(t, 0.17*library.document.recordingCalibration(heal.RecordingID), healing.BaseGain, 1e-9)
	require.NoError(t, custom.PlayBackgroundMusic("town"))
	require.InDelta(t, custom.config.MusicVolume*0.23*library.document.recordingCalibration(music.RecordingID),
		custom.backgroundMusic.player.Volume(), 1e-9)
	definition, found := custom.GetAmbientSoundDefinition(AmbientFire)
	require.True(t, found)
	require.InDelta(t, 0.13*library.document.recordingCalibration(ambient.RecordingID), definition.BaseVolume, 1e-9)
	require.Equal(t, "title", custom.GetMusicManager().GetMusicForLocation(references.Britain))
	require.Equal(t, "town", base.GetMusicManager().GetMusicForLocation(references.Britain))
	require.Contains(t, custom.GetMusicManager().GetAllLocationMusicMappings(), references.BritanniaUnderworld)
	require.Equal(t, custom.GetMusicManager().GetMusicForLocation(references.BritanniaUnderworld),
		custom.GetMusicManager().GetAllLocationMusicMappings()[references.BritanniaUnderworld])
	require.NotEqual(t, 0.17, productionLibrary().Cues["sfx/SoundHeal"].VolumeModifier)
}
