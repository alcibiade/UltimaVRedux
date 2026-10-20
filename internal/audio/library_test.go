package audio

import (
	"encoding/json"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

func TestLibraryPrepopulatesEveryKnownCueWithResolvableRecordingOrExplicitSlot(t *testing.T) {
	library := productionLibrary()
	require.Len(t, library.Cues, len(knownLibraryCueIDs(library)))
	for cueID := range knownLibraryCueIDs(library) {
		cue, exists := library.Cues[cueID]
		require.True(t, exists, cueID)
		recording, exists := library.Recordings[cue.PrimaryRecording()]
		require.True(t, exists, cueID)
		require.NotEmpty(t, cue.HowToHear, cueID)
		if recording.File != nil {
			require.NotEmpty(t, *recording.File, "null is the only empty-file representation")
		}
	}
	manager, _, _ := gameAudioLibraryFixture(t, defaultRuntimeLibrary())
	for effect := range catalogEffectNames() {
		before := len(manager.sfx.Snapshot())
		require.NoError(t, manager.PlaySoundEffect(effect), "every known SFX is playable or intentionally silent")
		if isPlaceholderSound(effect) {
			require.Len(t, manager.sfx.Snapshot(), before)
		}
		manager.StopSound(game_state.SoundStopRequest{Effect: effect})
	}
	require.Error(t, manager.PlaySoundEffect(game_state.SoundEffect(-1)), "unknown IDs are not silently treated as slots")
}

func TestLibraryV2RequiresExhaustiveAuthoredClassifications(t *testing.T) {
	for name, edit := range map[string]func(*audioLibrary){
		"old schema": func(l *audioLibrary) { l.SchemaVersion = 1 },
		"missing recording": func(l *audioLibrary) {
			l.Taxonomy.RecordingFamilies["fire"] = nil
		},
		"missing cue": func(l *audioLibrary) {
			l.Taxonomy.CueProfiles["interaction_action"] = l.Taxonomy.CueProfiles["interaction_action"][1:]
		},
		"duplicate cue": func(l *audioLibrary) {
			l.Taxonomy.CueProfiles["magic_activation"] = append(l.Taxonomy.CueProfiles["magic_activation"], "sfx/SoundHeal")
		},
		"unknown family": func(l *audioLibrary) {
			l.Taxonomy.RecordingFamilies["magik"] = l.Taxonomy.RecordingFamilies["magic"]
		},
		"unsorted tags": func(l *audioLibrary) {
			profile := l.Taxonomy.Profiles["footstep"]
			profile.Contexts = []string{"travel", "movement"}
			l.Taxonomy.Profiles["footstep"] = profile
		},
	} {
		t.Run(name, func(t *testing.T) {
			library := freshTestLibrary(t)
			edit(library)
			data, err := json.Marshal(library)
			require.NoError(t, err)
			_, err = parseAudioLibraryDocument(data)
			require.ErrorIs(t, err, ErrAudioLibrary)
		})
	}
}

func TestLibraryRejectsInvalidManualEdits(t *testing.T) {
	cases := map[string]func(*audioLibrary){
		"unknown cue":       func(l *audioLibrary) { l.Cues["sfx/SoundTypo"] = l.Cues["sfx/SoundHeal"] },
		"missing known cue": func(l *audioLibrary) { delete(l.Cues, "sfx/SoundHeal") },
		"unknown recording": func(l *audioLibrary) {
			cue := l.Cues["sfx/SoundHeal"]
			cue.RecordingID = "not_registered"
			l.Cues["sfx/SoundHeal"] = cue
		},
		"empty filename": func(l *audioLibrary) { l.Recordings["new_candidate"] = testRecording("") },
		"missing file": func(l *audioLibrary) {
			l.Recordings["new_candidate"] = testRecording("sounds/no_such_file.wav")
		},
		"no extension guessing": func(l *audioLibrary) {
			l.Recordings["new_candidate"] = testRecording("sounds/magic/healing_01.ogg")
		},
		"path escape": func(l *audioLibrary) {
			l.Recordings["new_candidate"] = testRecording("sounds/../magic/healing_01.wav")
		},
		"absolute path": func(l *audioLibrary) {
			l.Recordings["new_candidate"] = testRecording("/tmp/sound.wav")
		},
		"bad ID": func(l *audioLibrary) { l.Recordings["My sound"] = testRecording("sounds/magic/healing_01.wav") },
		"bad gain": func(l *audioLibrary) {
			cue := l.Cues["sfx/SoundHeal"]
			cue.VolumeModifier = 2
			l.Cues["sfx/SoundHeal"] = cue
		},
		"excessive calibration": func(l *audioLibrary) {
			recording := l.Recordings["healing_01"]
			recording.CalibrationGainDB = floatPointer(13)
			l.Recordings["healing_01"] = recording
		},
		"calibration on missing asset": func(l *audioLibrary) {
			l.Recordings["new_candidate"] = libraryRecording{
				Provenance: unknownRecordingProvenance(), CalibrationGainDB: floatPointer(0),
			}
		},
		"bad window": func(l *audioLibrary) {
			recording := l.Recordings["stone_steps_01"]
			recording.PlaybackWindow.EndMs = -1
			l.Recordings["stone_steps_01"] = recording
		},
		"window on empty slot": func(l *audioLibrary) {
			l.Recordings["new_candidate"] = libraryRecording{
				Provenance: unknownRecordingProvenance(), PlaybackWindow: &PlaybackWindow{StartMs: 0, EndMs: 10},
			}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			library := freshTestLibrary(t)
			mutate(library)
			data, err := json.Marshal(library)
			require.NoError(t, err)
			_, err = parseAudioLibrary(data, soundFiles)
			require.ErrorIs(t, err, ErrAudioLibrary)
		})
	}
}

func TestRecordingCalibrationUsesDBAndSharedIdentity(t *testing.T) {
	recording := productionLibrary().Recordings["healing_01"]
	recording.CalibrationGainDB = floatPointer(math.Inf(1))
	require.ErrorIs(t, validateLibraryRecording("healing_01", recording, soundFiles), ErrAudioLibrary)
	library := freshTestLibrary(t)
	id := library.Cues["sfx/SoundHeal"].RecordingID
	recording = library.Recordings[id]
	require.NotNil(t, recording.File)
	// Exercise an uncalibrated fixture independently of authored production gains.
	recording.CalibrationGainDB, recording.CalibrationPolicy = nil, ""
	library.Recordings[id] = recording
	require.Nil(t, recording.CalibrationGainDB, "unreviewed is not an accepted 0 dB")
	require.Equal(t, 1.0, library.recordingCalibration(id))
	recording.CalibrationGainDB = floatPointer(-6)
	library.Recordings[id] = recording
	data, err := json.Marshal(library)
	require.NoError(t, err)
	validated, err := parseAudioLibrary(data, soundFiles)
	require.NoError(t, err)
	want := math.Pow(10, -6.0/20)
	require.InDelta(t, want, validated.recordingCalibration(id), 1e-9)
	heal := validated.Cues["sfx/SoundHeal"]
	tick := validated.Cues["sfx/SoundHealingTick"]
	require.Equal(t, id, tick.RecordingID)
	require.InDelta(t, want, heal.VolumeModifier*validated.recordingCalibration(heal.RecordingID), 1e-9)
	require.InDelta(t, 0.6*want, tick.VolumeModifier*validated.recordingCalibration(tick.RecordingID), 1e-9)
}

func floatPointer(value float64) *float64 { return &value }

func TestLibraryStrictJSONRejectsDuplicatesTyposOmittedFileAndTrailingDocuments(t *testing.T) {
	source, err := soundFiles.ReadFile(libraryManifestPath)
	require.NoError(t, err)
	for name, data := range map[string][]byte{
		"duplicate":        []byte(`{"schemaVersion":1,"schemaVersion":2}`),
		"nested duplicate": []byte(`{"recordings":{"a":{"file":null,"file":"sounds/a.wav"}}}`),
		"omitted file":     []byte(`{"recordings":{"a":{}}}`),
		"misspelled field": []byte(strings.Replace(string(source), `"recordingId"`, `"recordngId"`, 1)),
		"missing gain":     []byte(strings.Replace(string(source), `"volumeModifier": 0.8,`, "", 1)),
		"null gain":        []byte(strings.Replace(string(source), `"volumeModifier": 0.8`, `"volumeModifier": null`, 1)),
		"trailing":         append(slices.Clone(source), []byte("{}")...),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseAudioLibrary(data, soundFiles)
			require.ErrorIs(t, err, ErrAudioLibrary)
		})
	}
}

func TestFillingPreparedSlotLoadsExactRecordingWithoutNewGameplayIdentifier(t *testing.T) {
	library := freshTestLibrary(t)
	cue := library.Cues["sfx/SoundRingVanish"]
	recording := library.Recordings[cue.RecordingID]
	require.Nil(t, recording.File)
	// The designer changes only the existing slot's file field.
	recording.File = stringPointer("sounds/magic/healing_01.wav")
	library.Recordings[cue.RecordingID] = recording
	data, err := json.Marshal(library)
	require.NoError(t, err)
	validated, err := parseAudioLibrary(data, soundFiles)
	require.NoError(t, err)
	require.NotEmpty(t, validated.recordingPath(cue.RecordingID))
	project, err := NewLibrary(data, soundFiles, LibraryOptions{AssetRoot: "assets", Validate: ReduxLibraryValidator})
	require.NoError(t, err)
	manager, _, _ := gameAudioLibraryFixture(t, project)
	require.NoError(t, manager.PlaySoundEffect(game_state.SoundRingVanish))
	require.Len(t, manager.sfx.Snapshot(), 1)
	require.Equal(t, cue.RecordingID, string(manager.sfx.Snapshot()[0].Recording))
	// Parsing/adopting a candidate must not mutate this build's production registry.
	require.Nil(t, productionLibrary().Recordings[cue.RecordingID].File)
}

func TestRecordingIdentityDoesNotDependOnFilename(t *testing.T) {
	library := freshTestLibrary(t)
	library.Recordings["new_metal_01"] = testRecording("sounds/footsteps/footstep_grass_left_01.ogg")
	library.Taxonomy.RecordingFamilies["footstep"] = append(library.Taxonomy.RecordingFamilies["footstep"], "new_metal_01")
	cue := library.Cues["sfx/SoundWalkMetal1"]
	cue.RecordingID, cue.RecordingIDs, cue.Selection = "new_metal_01", nil, ""
	library.Cues["sfx/SoundWalkMetal1"] = cue
	data, err := json.Marshal(library)
	require.NoError(t, err)
	validated, err := parseAudioLibrary(data, soundFiles)
	require.NoError(t, err)
	require.Nil(t, validated.Recordings[cue.RecordingID].PlaybackWindow, "replacement cannot inherit another recording's excerpt")
}

func freshTestLibrary(t *testing.T) *audioLibrary {
	t.Helper()
	source, err := soundFiles.ReadFile(libraryManifestPath)
	require.NoError(t, err)
	library, err := parseAudioLibrary(source, soundFiles)
	require.NoError(t, err)
	return library
}

func stringPointer(value string) *string { return &value }

func testRecording(file string) libraryRecording {
	return libraryRecording{File: stringPointer(file), Provenance: unknownRecordingProvenance()}
}
