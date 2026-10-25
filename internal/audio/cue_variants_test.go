package audio

import (
	"crypto/sha256"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestHexGrateCyclesFourDistinctSharedFileClipsAndLeavesBrickAssignments(t *testing.T) {
	manager, backend, _ := gameAudioLibraryFixture(t, defaultRuntimeLibrary())
	wantIDs := []string{"stone_steps_01", "stone_steps_02", "stone_steps_03", "stone_steps_04"}
	hashes := map[[sha256.Size]byte]bool{}
	for step := range 9 {
		require.NoError(t, manager.PlayWalkingSound(&references.Tile{Index: indexes.MetalFloor}))
		var grateVoices int
		for _, sound := range manager.sfx.Snapshot() {
			if sound.Cue != "sfx/SoundWalkMetal1" {
				continue
			}
			grateVoices++
			require.Equal(t, wantIDs[step%4], string(sound.Recording))
			recording := productionLibrary().Recordings[string(sound.Recording)]
			require.Equal(t, "sounds/footsteps/stone_steps_01.ogg", *recording.File)
			pcm, err := io.ReadAll(backend.players[len(backend.players)-1].source)
			require.NoError(t, err)
			require.Equal(t, 1, audibleBursts(pcm, EditorSampleRate))
			hashes[sha256.Sum256(pcm)] = true
		}
		require.Equal(t, 1, grateVoices)
		if step == 1 {
			require.NoError(t, manager.PlayWalkingSound(&references.Tile{Index: indexes.BrickFloor}))
		}
	}
	require.Len(t, hashes, 4, "the four authored clips must select distinct audio")
	for index, id := range []string{"sfx/SoundWalkStone1", "sfx/SoundWalkStone2", "sfx/SoundWalkStone3", "sfx/SoundWalkStone4"} {
		cue := productionLibrary().Cues[id]
		require.Equal(t, []string{"footstep_concrete_01", "footstep_concrete_02", "footstep_concrete_03", "footstep_concrete_04"}[index], cue.RecordingID)
		require.Empty(t, cue.RecordingIDs)
		require.Empty(t, cue.Selection)
	}
}

func TestCueVariantSchemaAndCatalogLinks(t *testing.T) {
	for name, change := range map[string]func(*LibraryDocument){
		"both forms": func(l *LibraryDocument) {
			c := l.Cues["sfx/SoundWalkMetal1"]
			c.RecordingID = "stone_steps_01"
			l.Cues["sfx/SoundWalkMetal1"] = c
		},
		"duplicate": func(l *LibraryDocument) {
			c := l.Cues["sfx/SoundWalkMetal1"]
			c.RecordingIDs[1] = c.RecordingIDs[0]
			l.Cues["sfx/SoundWalkMetal1"] = c
		},
		"unknown": func(l *LibraryDocument) {
			c := l.Cues["sfx/SoundWalkMetal1"]
			c.RecordingIDs[1] = "absent"
			l.Cues["sfx/SoundWalkMetal1"] = c
		},
		"missing policy": func(l *LibraryDocument) {
			c := l.Cues["sfx/SoundWalkMetal1"]
			c.Selection = ""
			l.Cues["sfx/SoundWalkMetal1"] = c
		},
		"unsupported music": func(l *LibraryDocument) {
			c := l.Cues["music/track/town"]
			c.RecordingID, c.RecordingIDs, c.Selection = "", []string{"stone_steps_01", "stone_steps_02"}, SelectionCycle
			l.Cues["music/track/town"] = c
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
