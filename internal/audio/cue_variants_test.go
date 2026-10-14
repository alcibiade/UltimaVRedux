package audio

import (
	"crypto/sha256"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestHexGrateCyclesFourDistinctSharedFileClipsAndLeavesBrickAssignments(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.ensureAssetsLoaded())
	player := manager.soundEffectPlayer
	assets := player.soundAssets[game_state.SoundWalkMetal1]
	require.Len(t, assets, 4)
	wantIDs := []string{"stone_steps_01", "stone_steps_02", "stone_steps_03", "stone_steps_04"}
	hashes := map[[sha256.Size]byte]bool{}
	for index, asset := range assets {
		require.Equal(t, wantIDs[index], asset.RecordingID)
		require.Equal(t, "sounds/footsteps/stone_steps_01.ogg", *productionLibrary().Recordings[asset.RecordingID].File)
		stream, err := asset.StreamFactory(game_state.SoundPlaybackOneShot)
		require.NoError(t, err)
		pcm, err := io.ReadAll(stream)
		require.NoError(t, err)
		require.Equal(t, 1, audibleBursts(pcm, asset.SampleRate), asset.RecordingID)
		hashes[sha256.Sum256(pcm)] = true
	}
	require.Len(t, hashes, 4, "the four metadata clips must select different audio")
	for step := range 9 {
		require.NoError(t, manager.PlayWalkingSound(&references.Tile{Index: indexes.MetalFloor}))
		var grateVoices int
		for _, sound := range player.activeSounds {
			if sound.Effect != game_state.SoundWalkMetal1 {
				continue
			}
			grateVoices++
			require.Equal(t, wantIDs[step%4], sound.RecordingID)
			require.InDelta(t, 0.35*productionLibrary().recordingCalibration(sound.RecordingID), sound.BaseGain, 1e-9)
		}
		require.Equal(t, 1, grateVoices)
		if step == 1 {
			require.NoError(t, manager.PlayWalkingSound(&references.Tile{Index: indexes.BrickFloor}))
		}
	}
	for index, id := range []string{"sfx/SoundWalkStone1", "sfx/SoundWalkStone2", "sfx/SoundWalkStone3", "sfx/SoundWalkStone4"} {
		cue := productionLibrary().Cues[id]
		require.Equal(t, []string{"footstep_concrete_01", "footstep_concrete_02", "footstep_concrete_03", "footstep_concrete_04"}[index], cue.RecordingID)
		require.Empty(t, cue.RecordingIDs)
		require.Empty(t, cue.Selection)
	}
}

func TestCueCycleDoesNotAdvanceOnRejectedOrDroppedRequests(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.ensureAssetsLoaded())
	player := manager.soundEffectPlayer
	effect := game_state.SoundWalkMetal1
	request := game_state.SoundPlaybackRequest{Effect: effect, Mode: game_state.SoundPlaybackOneShot, Volume: 1}
	invalid := request
	invalid.Volume = -1
	require.Error(t, player.Play(invalid))
	require.Zero(t, player.nextVariant[effect])
	asset := player.soundAssets[effect][0]
	factory := asset.StreamFactory
	asset.StreamFactory = func(game_state.SoundPlaybackMode) (io.ReadSeeker, error) { return nil, errSoundStreamUnavailable }
	require.Error(t, player.Play(request))
	require.Zero(t, player.nextVariant[effect])
	asset.StreamFactory = factory
	for range 8 {
		require.NoError(t, player.Play(game_state.SoundPlaybackRequest{
			Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 1}))
	}
	require.NoError(t, player.Play(request)) // Finite request dropped while all voices are owned loops.
	require.Zero(t, player.nextVariant[effect])
	player.StopAll()
	require.NoError(t, player.Play(request))
	require.Equal(t, 1, player.nextVariant[effect])
}

func TestCueVariantSchemaAndCatalogLinks(t *testing.T) {
	for name, change := range map[string]func(*audioLibrary){
		"both forms": func(l *audioLibrary) {
			c := l.Cues["sfx/SoundWalkMetal1"]
			c.RecordingID = "stone_steps_01"
			l.Cues["sfx/SoundWalkMetal1"] = c
		},
		"duplicate": func(l *audioLibrary) {
			c := l.Cues["sfx/SoundWalkMetal1"]
			c.RecordingIDs[1] = c.RecordingIDs[0]
			l.Cues["sfx/SoundWalkMetal1"] = c
		},
		"unknown": func(l *audioLibrary) {
			c := l.Cues["sfx/SoundWalkMetal1"]
			c.RecordingIDs[1] = "absent"
			l.Cues["sfx/SoundWalkMetal1"] = c
		},
		"missing policy": func(l *audioLibrary) {
			c := l.Cues["sfx/SoundWalkMetal1"]
			c.Selection = ""
			l.Cues["sfx/SoundWalkMetal1"] = c
		},
		"unsupported music": func(l *audioLibrary) {
			c := l.Cues["music/track/town"]
			l.Cues["music/track/town"] = c.WithRecordings([]string{"stone_steps_01", "stone_steps_02"})
		},
	} {
		t.Run(name, func(t *testing.T) {
			library := freshTestLibrary(t)
			change(library)
			data, err := json.Marshal(library)
			require.NoError(t, err)
			_, err = parseAudioLibrary(data, soundFiles)
			require.ErrorIs(t, err, ErrAudioLibrary)
		})
	}
	catalog, err := BuildCatalog(false)
	require.NoError(t, err)
	moment := findCatalogMoment(t, catalog, "sfx/SoundWalkMetal1")
	require.Len(t, moment.AssetIDs, 4)
	require.Equal(t, SelectionCycle, moment.Selection)
	for _, asset := range catalog.Assets {
		if asset.ID == "stone_steps_02" || asset.ID == "stone_steps_03" || asset.ID == "stone_steps_04" {
			require.Contains(t, asset.MomentIDs, moment.ID)
			require.NotNil(t, asset.Window)
		}
	}
	require.Contains(t, catalog.GenerationGuide(), "stone_steps_04")
}
